package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	responses "core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type preOrderService struct {
	config                       *config.AppConfig
	preOrderRepository           irepository.GenericRepository[model.PreOrder]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iproxies.GHNProxy
	paymentTransactionService    iservice.PaymentTransactionService
}

func (p preOrderService) PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork) (*model.PreOrder, error) {
	var preOrder *model.PreOrder
	now := time.Now()

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//1. validate variant and stocks/products
		includes := []string{"Product", "Product.Limited"}
		variant, err := uow.ProductVariant().GetByID(ctx, request.VariantID, includes)
		if err != nil {
			return fmt.Errorf("variant %w not found", err)
		} else if err = validateVariantForPreOrder(*variant); err != nil {
			return err
		}

		// Check if this preorderable?
		if variant.Product.Limited != nil {
			premiereDate := variant.Product.Limited.PremiereDate
			startDate := variant.Product.Limited.AvailabilityStartDate
			endDate := variant.Product.Limited.AvailabilityEndDate
			isPreOrderable := now.After(premiereDate) && now.Before(startDate) && now.Before(endDate)
			if !isPreOrderable {
				return fmt.Errorf("product is not available for pre-order at this time")
			}

			if variant.PreOrderLimit == variant.PreOrderCount {
				return fmt.Errorf("pre-order limit reached for this variant")
			}

			//Check current orderable stats
			if variant.PreOrderLimit == nil || variant.PreOrderCount == nil {
				zap.L().Debug("Invalid data format: PreOrderLimit or PreOrderCount is nil")
				return fmt.Errorf("pre-order limit or PreOrderCount is nil")
			}

			remainingOrderSlot := *variant.PreOrderLimit - *variant.PreOrderCount
			if remainingOrderSlot <= 0 {
				return fmt.Errorf("no remaining pre-order slots for this variant")
			}
		}

		//1.1 validate shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, request.AddressID, []string{})
		if err != nil {
			return fmt.Errorf("failed to get shipping address: %w", err)
		}

		// 2. Build persistent model -> minus stock, create pre-order record
		// 2.1 build pre-order model
		preOrder = requests.PreOrderRequest{}.ToModel(*address, *variant, now)
		// 2.2 minus 1 stock
		if variant.CurrentStock == nil {
			return fmt.Errorf("variant stock is nil")
		}
		variant.CurrentStock = ptr.Int(*variant.CurrentStock - 1)
		variant.PreOrderCount = ptr.Int(*variant.PreOrderCount + 1)
		err = uow.ProductVariant().Update(ctx, variant)
		if err != nil {
			return fmt.Errorf("failed to update variant stock: %w", err)
		}
		// 2.3 create pre-order
		err = uow.PreOrder().Add(ctx, preOrder)
		if err != nil {
			return fmt.Errorf("failed to add preorder: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return preOrder, nil
}

func (p preOrderService) GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string, statuses []string) ([]responses.PreOrderResponse, int, error) {
	ctx := context.Background()

	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (pageNum - 1) * pageSize

	// Normalize/validate statuses if provided
	var validStatuses []enum.PreOrderStatus
	if len(statuses) > 0 {
		for _, s := range statuses {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			st := enum.PreOrderStatus(s)
			if st.IsValid() {
				validStatuses = append(validStatuses, st)
			}
		}
	}

	// Base filter with user, optional joins for searching by product name/full name, and optional status
	filter := func(db *gorm.DB) *gorm.DB {
		db = db.Where("pre_orders.user_id = ?", userID)
		if len(validStatuses) > 0 {
			// build slice of strings for SQL IN
			vals := make([]string, 0, len(validStatuses))
			for _, v := range validStatuses {
				vals = append(vals, string(v))
			}
			db = db.Where("pre_orders.status IN ?", vals)
		}
		if search != "" {
			// join product tables for searching by product name
			db = db.Joins("LEFT JOIN product_variants pv ON pv.id = pre_orders.variant_id").
				Joins("LEFT JOIN products p ON p.id = pv.product_id").
				Where("p.name ILIKE ? OR pre_orders.full_name ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		return db.Order("pre_orders.created_at DESC").Order("pre_orders.id")
	}

	includes := []string{"ProductVariant", "ProductVariant.Product"}

	// 1) fetch paged IDs first
	var ids []uuid.UUID
	if err := p.preOrderRepository.DB().
		WithContext(ctx).
		Model(&model.PreOrder{}).
		Scopes(filter).
		Select("pre_orders.id, pre_orders.created_at").
		Limit(pageSize).
		Offset(offset).
		Pluck("pre_orders.id", &ids).Error; err != nil {
		zap.L().Error("Failed to fetch preorder IDs", zap.Error(err))
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []responses.PreOrderResponse{}, 0, nil
	}

	// 2) count total with same criteria but without pagination
	countScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("pre_orders.user_id = ?", userID)
		if len(validStatuses) > 0 {
			vals := make([]string, 0, len(validStatuses))
			for _, v := range validStatuses {
				vals = append(vals, string(v))
			}
			db = db.Where("pre_orders.status IN ?", vals)
		}
		if search != "" {
			db = db.Joins("LEFT JOIN product_variants pv ON pv.id = pre_orders.variant_id").
				Joins("LEFT JOIN products p ON p.id = pv.product_id").
				Where("p.name ILIKE ? OR pre_orders.full_name ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		return db
	}

	var total int64
	if err := p.preOrderRepository.DB().
		WithContext(ctx).
		Model(&model.PreOrder{}).
		Scopes(countScope).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to count preorders", zap.Error(err))
		return nil, 0, err
	}

	// 3) final fetch with includes by IDs to avoid duplication
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("pre_orders.id IN ?", ids).
			Order("pre_orders.created_at DESC")
	}

	preorders, _, err := p.preOrderRepository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch preorders with includes", zap.Error(err))
		return nil, 0, err
	}

	// Load payment transactions referencing these preorders (non-fatal)
	var transactions []model.PaymentTransaction
	if p.paymentTransactionRepository != nil && p.paymentTransactionRepository.DB() != nil {
		if err := p.paymentTransactionRepository.DB().WithContext(ctx).
			Model(&model.PaymentTransaction{}).
			Where("reference_type = ? AND reference_id IN (?)", enum.PaymentTransactionReferenceTypePreOrder, ids).
			Find(&transactions).Error; err != nil {
			zap.L().Warn("Failed to fetch payment transactions for preorders, continuing without payments", zap.Error(err))
			transactions = nil
		}
	}

	// Choose latest transaction per preorder by UpdatedAt
	paymentsMap := make(map[uuid.UUID]model.PaymentTransaction)
	for _, tx := range transactions {
		existing, ok := paymentsMap[tx.ReferenceID]
		if !ok || tx.UpdatedAt.After(existing.UpdatedAt) {
			paymentsMap[tx.ReferenceID] = tx
		}
	}

	// Map to response DTOs
	resList := make([]responses.PreOrderResponse, 0, len(preorders))
	for i := range preorders {
		pr := preorders[i]
		resp := responses.PreOrderResponse{}
		resp.PreOrder = pr
		if pt, ok := paymentsMap[pr.ID]; ok {
			// build response manually to avoid method call confusion
			resp.PaymentTx = responses.PaymentTransactionResponse{
				ID:              pt.ID,
				ReferenceID:     pt.ReferenceID.String(),
				ReferenceType:   pt.ReferenceType.String(),
				Amount:          utils.ToString(pt.Amount),
				Method:          pt.Method,
				Status:          string(pt.Status),
				TransactionDate: utils.FormatLocalTime(&pt.TransactionDate, utils.TimeFormat),
				GatewayRef:      pt.GatewayRef,
				GatewayID:       pt.GatewayID,
				UpdatedAt:       utils.FormatLocalTime(&pt.UpdatedAt, utils.TimeFormat),
			}
		}
		resList = append(resList, resp)
	}

	return resList, int(total), nil
}

func (p preOrderService) PayForPreservationSlot(ctx context.Context, preOrderID uuid.UUID, returnURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error) {
	var paymentTransaction *responses.PayOSLinkResponse

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//*1. Preload pre-order
		preOrder, err := uow.PreOrder().GetByID(ctx, preOrderID, []string{"ProductVariant", "ProductVariant.Product"})
		if err != nil {
			return fmt.Errorf("failed to get pre-order: %w", err)
		}

		//*2 Build Payment Request:
		paymentItemRequest, total := toPaymentItemRequestWithTotalPrice(*preOrder)
		paymentRq := requests.PaymentRequest{
			ReferenceID:   preOrderID,
			ReferenceType: enum.PaymentTransactionReferenceTypePreOrder,
			PayerID:       &preOrder.UserID,
			Amount:        total,
			Description:   fmt.Sprintf("Payment for preservation %s", preOrder.ID),
			Items:         paymentItemRequest,
			BuyerName:     preOrder.FullName,
			BuyerEmail:    preOrder.Email,
			BuyerPhone:    preOrder.PhoneNumber,
			ReturnURL:     &returnURL,
			CancelURL:     &cancelURL,
		}

		//*3. Create Payment Transaction
		paymentTransaction, err = p.paymentTransactionService.GeneratePaymentLink(ctx, uow, &paymentRq)
		if err != nil {
			zap.L().Error("Failed to initiate payment transaction for order", zap.Error(err))
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return paymentTransaction, nil
}

func NewPreOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry, paymentTransactionSvc iservice.PaymentTransactionService) iservice.PreOrderService {
	return &preOrderService{
		config:                       cfg,
		preOrderRepository:           dbRegistry.PreOrderRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		payOSProxy:                   registry.ProxiesRegistry.PayOSProxy,
		shippingAddressRepository:    dbRegistry.ShippingAddressRepository,
		ghnService:                   registry.ProxiesRegistry.GHNProxy,
		paymentTransactionService:    paymentTransactionSvc,
	}
}

// ----------------------------Validator-----------------------------//
func validateVariantForPreOrder(variant model.ProductVariant) error {
	//validate if product type is limited
	if variant.Product != nil && variant.Product.Type != enum.ProductTypeLimited {
		return fmt.Errorf("invalid product type for pre-order")
	}

	//validate if product is active
	if variant.Product.IsActive == false {
		return fmt.Errorf("product is not available, please contact admin to activate it")
	}

	//Check stock
	if *variant.CurrentStock == 0 {
		return fmt.Errorf("variant is out of stock")
	}
	return nil
}

//=========== Helper Methods ===========

func (p preOrderService) generateSignature(amount int64, cancelURL, description string, orderCode int64, returnURL string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelURL, description, orderCode, returnURL,
	)
	mac := hmac.New(sha256.New, []byte(p.config.PayOS.ChecksumKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

//----------------------------Mapper-----------------------------//

// preOrderToPayOSItemsWithTotalPrice maps a PreOrder model to a slice of PayOSItem DTOs.
func preOrderToPayOSItemsWithTotalPrice(preOrder model.PreOrder) ([]dtos.PayOSItem, int64) {
	items := make([]dtos.PayOSItem, 0)
	total := 0

	// Build variant descriptive string (e.g. "250ML - Bottle - Spray")
	variantPropConcat := fmt.Sprintf("%v", preOrder.Capacity) +
		utils.ToTitleCase(*preOrder.CapacityUnit) + " - " +
		utils.ToTitleCase(preOrder.ContainerType.String()) + " - " +
		utils.ToTitleCase(preOrder.DispenserType.String())

	// Build readable item name (e.g. "Shampoo (250ML - Bottle - Spray)")
	variantName := utils.ToTitleCase(preOrder.ProductVariant.Product.Name) + fmt.Sprintf(" (%s)", variantPropConcat)

	mappedModel := dtos.PayOSItem{
		Name:     variantName,
		Quantity: 1,
		Price:    preOrder.UnitPrice,
	}

	items = append(items, mappedModel)
	total += int(preOrder.UnitPrice)

	return items, int64(total)
}

func toPaymentItemRequestWithTotalPrice(preOrder model.PreOrder) ([]requests.PaymentItemRequest, int64) {
	items := make([]requests.PaymentItemRequest, 0)
	total := int64(0)

	// Build variant descriptive string (e.g. "250ML - Bottle - Spray")
	variantPropConcat := fmt.Sprintf("%v", preOrder.Capacity) +
		utils.ToTitleCase(*preOrder.CapacityUnit) + " - " +
		utils.ToTitleCase(preOrder.ContainerType.String()) + " - " +
		utils.ToTitleCase(preOrder.DispenserType.String())

	// Build readable item name (e.g. "Shampoo (250ML - Bottle - Spray)")
	variantName := utils.ToTitleCase(preOrder.ProductVariant.Product.Name) + fmt.Sprintf(" (%s)", variantPropConcat)
	mappedModel := requests.PaymentItemRequest{
		Name:     variantName,
		Quantity: 1,
		Price:    int64(preOrder.UnitPrice),
	}

	items = append(items, mappedModel)
	total += int64(preOrder.UnitPrice)
	return items, total
}

// GetStaffAvailablePreOrdersWithPagination returns preorders for staff with same filtering/search as staff orders
func (p preOrderService) GetStaffAvailablePreOrdersWithPagination(
	limit, page int,
	search, fullName, phone, provinceID, districtID, wardCode string,
	statuses []string,
) ([]responses.PreOrderResponse, int, error) {
	ctx := context.Background()

	// --- Pagination defaults ---
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// --- Normalize statuses ---
	var validStatuses []enum.PreOrderStatus
	for _, s := range statuses {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		st := enum.PreOrderStatus(s)
		if st.IsValid() {
			validStatuses = append(validStatuses, st)
		}
	}

	// --- Prefetch preorder IDs matching transactions ---
	var txMatchedPreorderIDs []uuid.UUID
	if strings.TrimSpace(search) != "" && p.paymentTransactionRepository != nil && p.paymentTransactionRepository.DB() != nil {
		like := "%" + search + "%"
		if err := p.paymentTransactionRepository.DB().WithContext(ctx).
			Model(&model.PaymentTransaction{}).
			Where("reference_type = ? AND (id::text ILIKE ? OR payos_metadata->>'bin' ILIKE ?)",
				enum.PaymentTransactionReferenceTypePreOrder, like, like).
			Distinct().Pluck("reference_id", &txMatchedPreorderIDs).Error; err != nil {
			zap.L().Warn("failed to lookup transactions for staff preorders search", zap.Error(err))
			txMatchedPreorderIDs = nil
		}
	}

	// --- Build filter scope ---
	filter := func(db *gorm.DB) *gorm.DB {
		// Status
		if len(validStatuses) > 0 {
			vals := make([]string, 0, len(validStatuses))
			for _, v := range validStatuses {
				vals = append(vals, string(v))
			}
			db = db.Where("pre_orders.status IN ? AND pre_orders.status <> ?", vals, enum.PreOrderStatusPending)
		} else {
			db = db.Where("pre_orders.status <> ?", enum.PreOrderStatusPending)
		}

		// Search by preorder id, full name, or tx matches
		if strings.TrimSpace(search) != "" {
			like := "%" + search + "%"
			if len(txMatchedPreorderIDs) > 0 {
				db = db.Where("(pre_orders.id::text ILIKE ? OR pre_orders.full_name ILIKE ? OR pre_orders.id IN (?))",
					like, like, txMatchedPreorderIDs)
			} else {
				db = db.Where("(pre_orders.id::text ILIKE ? OR pre_orders.full_name ILIKE ?)", like, like)
			}
		}

		// Other filters
		if fullName != "" {
			db = db.Where("pre_orders.full_name ILIKE ?", "%"+fullName+"%")
		}
		if phone != "" {
			db = db.Where("pre_orders.phone_number ILIKE ?", "%"+phone+"%")
		}
		if provinceID != "" {
			if pid, err := strconv.Atoi(provinceID); err == nil {
				db = db.Where("pre_orders.ghn_province_id = ?", pid)
			}
		}
		if districtID != "" {
			if did, err := strconv.Atoi(districtID); err == nil {
				db = db.Where("pre_orders.ghn_district_id = ?", did)
			}
		}
		if wardCode != "" {
			db = db.Where("pre_orders.ghn_ward_code = ?", wardCode)
		}

		return db
	}

	// --- Step 1: Get paged IDs safely ---
	var ids []uuid.UUID
	if err := p.preOrderRepository.DB().WithContext(ctx).
		Model(&model.PreOrder{}).
		Scopes(filter).
		Order("pre_orders.created_at DESC").
		Order("pre_orders.id").
		Limit(limit).
		Offset(offset).
		Pluck("pre_orders.id", &ids).Error; err != nil {
		zap.L().Error("failed to fetch staff preorder ids", zap.Error(err))
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []responses.PreOrderResponse{}, 0, nil
	}

	// --- Step 2: Count total ---
	var total int64
	if err := p.preOrderRepository.DB().WithContext(ctx).
		Model(&model.PreOrder{}).
		Scopes(filter).
		Count(&total).Error; err != nil {
		zap.L().Error("failed to count staff preorders", zap.Error(err))
		return nil, 0, err
	}

	// --- Step 3: Load full models with includes ---
	includes := []string{"ProductVariant", "ProductVariant.Product"}
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("pre_orders.id IN ?", ids).Order("pre_orders.created_at DESC")
	}

	preorders, _, err := p.preOrderRepository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("failed to fetch staff preorders with includes", zap.Error(err))
		return nil, 0, err
	}

	// --- Step 4: Map latest payment transaction ---
	var transactions []model.PaymentTransaction
	if p.paymentTransactionRepository != nil && p.paymentTransactionRepository.DB() != nil {
		if err := p.paymentTransactionRepository.DB().WithContext(ctx).
			Model(&model.PaymentTransaction{}).
			Where("reference_type = ? AND reference_id IN (?)", enum.PaymentTransactionReferenceTypePreOrder, ids).
			Find(&transactions).Error; err != nil {
			zap.L().Warn("failed to fetch payment transactions, continuing without payments", zap.Error(err))
			transactions = nil
		}
	}

	paymentsMap := make(map[uuid.UUID]model.PaymentTransaction)
	for _, tx := range transactions {
		existing, ok := paymentsMap[tx.ReferenceID]
		if !ok || tx.UpdatedAt.After(existing.UpdatedAt) {
			paymentsMap[tx.ReferenceID] = tx
		}
	}

	// --- Step 5: Map to response DTOs ---
	resList := make([]responses.PreOrderResponse, 0, len(preorders))
	for i := range preorders {
		pr := preorders[i]
		resp := responses.PreOrderResponse{PreOrder: pr}
		if pt, ok := paymentsMap[pr.ID]; ok {
			resp.PaymentTx = responses.PaymentTransactionResponse{
				ID:              pt.ID,
				ReferenceID:     pt.ReferenceID.String(),
				ReferenceType:   pt.ReferenceType.String(),
				Amount:          utils.ToString(pt.Amount),
				Method:          pt.Method,
				Status:          string(pt.Status),
				TransactionDate: utils.FormatLocalTime(&pt.TransactionDate, utils.TimeFormat),
				GatewayRef:      pt.GatewayRef,
				GatewayID:       pt.GatewayID,
				UpdatedAt:       utils.FormatLocalTime(&pt.UpdatedAt, utils.TimeFormat),
			}
		}
		resList = append(resList, resp)
	}

	return resList, int(total), nil
}
