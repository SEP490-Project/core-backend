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
		variant, err := uow.ProductVariant().GetByID(ctx, request.VariantID, []string{"Product"})
		if err != nil {
			return fmt.Errorf("variant %w not found", err)
		} else if err := validateVariantForPreOrder(*variant); err != nil {
			return err
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

func (p preOrderService) GetPreOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string, status string) ([]model.PreOrder, int, error) {
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

	// Normalize/validate status if provided
	var validStatus *enum.PreOrderStatus
	if status != "" {
		s := enum.PreOrderStatus(status)
		if s.IsValid() {
			validStatus = &s
		}
	}

	// Base filter with user, optional joins for searching by product name/full name, and optional status
	filter := func(db *gorm.DB) *gorm.DB {
		db = db.Where("pre_orders.user_id = ?", userID)
		if validStatus != nil {
			db = db.Where("pre_orders.status = ?", *validStatus)
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
		Select("pre_orders.id").
		Limit(pageSize).
		Offset(offset).
		Pluck("pre_orders.id", &ids).Error; err != nil {
		zap.L().Error("Failed to fetch preorder IDs", zap.Error(err))
		return nil, 0, err
	}

	if len(ids) == 0 {
		return []model.PreOrder{}, 0, nil
	}

	// 2) count total with same criteria but without pagination
	countScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("pre_orders.user_id = ?", userID)
		if validStatus != nil {
			db = db.Where("pre_orders.status = ?", *validStatus)
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

	// Attach transient payment fields (payment_id, payment_bin) similar to orders implementation
	for i := range preorders {
		var pt struct {
			PaymentID  *uuid.UUID
			PaymentBin *string
		}
		if err := p.preOrderRepository.DB().WithContext(ctx).Raw(
			"SELECT id AS payment_id, payos_metadata->>'bin' AS payment_bin FROM payment_transactions WHERE reference_id = ? AND reference_type = 'PREORDER' LIMIT 1",
			preorders[i].ID,
		).Scan(&pt).Error; err == nil {
			preorders[i].PaymentID = pt.PaymentID
			preorders[i].PaymentBin = pt.PaymentBin
		}
	}

	return preorders, int(total), nil
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

func (o preOrderService) generateSignature(amount int64, cancelURL, description string, orderCode int64, returnURL string) (string, error) {
	data := fmt.Sprintf(
		"amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		amount, cancelURL, description, orderCode, returnURL,
	)
	mac := hmac.New(sha256.New, []byte(o.config.PayOS.ChecksumKey))
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
func (p preOrderService) GetStaffAvailablePreOrdersWithPagination(limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.PreOrder, int, error) {
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

	var validStatus *enum.PreOrderStatus
	if status != "" {
		s := enum.PreOrderStatus(status)
		if s.IsValid() {
			validStatus = &s
		}
	}

	whereClauses := make([]string, 0)
	args := make([]any, 0)

	// exclude PENDING by default to match staff view behavior
	whereClauses = append(whereClauses, "pre_orders.status <> ?")
	args = append(args, enum.PreOrderStatusPending)

	if validStatus != nil && *validStatus != enum.PreOrderStatusPending {
		whereClauses = append(whereClauses, "pre_orders.status = ?")
		args = append(args, *validStatus)
	}

	if search != "" {
		whereClauses = append(whereClauses, "(pre_orders.id::text ILIKE ? OR pt.id::text ILIKE ? OR pt.payos_metadata->>'bin' ILIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like)
	}
	if fullName != "" {
		whereClauses = append(whereClauses, "pre_orders.full_name ILIKE ?")
		args = append(args, "%"+fullName+"%")
	}
	if phone != "" {
		whereClauses = append(whereClauses, "pre_orders.phone_number ILIKE ?")
		args = append(args, "%"+phone+"%")
	}
	if provinceID != "" {
		if pid, err := strconv.Atoi(provinceID); err == nil {
			whereClauses = append(whereClauses, "pre_orders.ghn_province_id = ?")
			args = append(args, pid)
		}
	}
	if districtID != "" {
		if did, err := strconv.Atoi(districtID); err == nil {
			whereClauses = append(whereClauses, "pre_orders.ghn_district_id = ?")
			args = append(args, did)
		}
	}
	if wardCode != "" {
		whereClauses = append(whereClauses, "pre_orders.ghn_ward_code = ?")
		args = append(args, wardCode)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	type preOrderWithTotal struct {
		model.PreOrder
		PaymentID  *uuid.UUID `gorm:"column:payment_id"`
		Bin        *string    `gorm:"column:bin"`
		TotalCount int64      `gorm:"column:total_count"`
	}

	sql := fmt.Sprintf(`SELECT pre_orders.*, pt.id AS payment_id, pt.payos_metadata->>'bin' AS bin, COUNT(*) OVER() AS total_count FROM pre_orders LEFT JOIN payment_transactions pt ON pt.reference_id = pre_orders.id AND pt.reference_type = 'PREORDER' WHERE %s ORDER BY pre_orders.created_at DESC LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, offset)

	var rows []preOrderWithTotal
	if err := p.preOrderRepository.DB().WithContext(ctx).Raw(sql, args...).Scan(&rows).Error; err != nil {
		zap.L().Error("Failed to execute staff preorders raw query", zap.Error(err))
		return nil, 0, err
	}

	if len(rows) == 0 {
		return []model.PreOrder{}, 0, nil
	}

	total := int(rows[0].TotalCount)
	preorders := make([]model.PreOrder, 0, len(rows))
	preorderIDs := make([]uuid.UUID, 0, len(rows))
	paymentMap := make(map[uuid.UUID]*uuid.UUID)
	binMap := make(map[uuid.UUID]*string)
	for _, r2 := range rows {
		preorders = append(preorders, r2.PreOrder)
		preorderIDs = append(preorderIDs, r2.ID)
		if r2.PaymentID != nil {
			paymentMap[r2.ID] = r2.PaymentID
			binMap[r2.ID] = r2.Bin
		} else {
			paymentMap[r2.ID] = nil
			binMap[r2.ID] = nil
		}
	}

	// Attach transient payment fields
	for i := range preorders {
		if pid := paymentMap[preorders[i].ID]; pid != nil {
			preorders[i].PaymentID = pid
		}
		if b := binMap[preorders[i].ID]; b != nil {
			preorders[i].PaymentBin = b
		}
	}

	return preorders, total, nil
}
