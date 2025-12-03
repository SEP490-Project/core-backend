package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	notiBuilder "core-backend/internal/application/service/notification_builder"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/preordersm"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
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
	db                           *gorm.DB
	preOrderRepository           irepository.GenericRepository[model.PreOrder]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	userRepository               irepository.GenericRepository[model.User]
	variantRepository            irepository.GenericRepository[model.ProductVariant]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iproxies.GHNProxy
	paymentTransactionService    iservice.PaymentTransactionService
	stateTransferService         iservice.StateTransferService
	notificationService          iservice.NotificationService
}

func (p preOrderService) ApproveRefundRequest(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusRefunded, reason)
	if err != nil {
		return err
	}

	preOrder.StaffResource = fileURL
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		return err
	}

	//Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyRefund, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()
	return nil
}

func (p preOrderService) RefundRequest(ctx context.Context, preOrderID, actionBy uuid.UUID, reason *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusRefundRequest, reason)
	if err != nil {
		return err
	}
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		return err
	}
	//Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyRefundRequest, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()
	return nil
}

func (p preOrderService) ObligateRefund(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusRefunded, reason)
	if err != nil {
		return err
	}
	preOrder.StaffResource = fileURL
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		return err
	}

	//Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyObligateRefund, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (p preOrderService) MarkPreOrderAsReceived(ctx context.Context, preOrderID, updatedBy uuid.UUID) error {
	err := p.stateTransferService.MovePreOrderToState(ctx, preOrderID, enum.PreOrderStatusReceived, updatedBy, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (p preOrderService) RequestCompensation(ctx context.Context, preOrderID, actionBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}

	//Check if compensation has already been requested
	if preOrder.ActionNotes != nil {
		for _, note := range *preOrder.ActionNotes {
			if note.ActionType == enum.PreOrderStatusCompensateRequest {
				return errors.New("you've already requested compensation for this preorder")
			}
		}
	}

	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusCompensateRequest, reason)
	if err != nil {
		return err
	}
	preOrder.UserResource = fileURL
	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		zap.L().Error("Failed to update PreOrder state", zap.String("preorder_id", preOrderID.String()), zap.Error(err))
		return errors.New("failed to update PreOrder state: " + err.Error())
	}

	// Notification
	go func() {
		err = p.sendNotification(context.Background(), notiBuilder.PreOrderNotifyCompensateRequested, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (p preOrderService) ProcessCompensation(ctx context.Context, preOrderID, actionBy uuid.UUID, isApproved bool, reason, fileURL *string) error {
	preOrder, limitedProduct, user, err := p.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, actionBy)
	if err != nil {
		return err
	}
	var notiStatus notiBuilder.PreOrderNotificationType

	if isApproved {
		err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusCompensated, reason)
		if err != nil {
			return err
		}
		if fileURL == nil {
			return errors.New("file can not be empty if the compensate request being approved")
		}
		preOrder.StaffResource = fileURL
		notiStatus = notiBuilder.PreOrderNotifyCompensated
	} else {
		err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, user, enum.PreOrderStatusDelivered, reason)
		if err != nil {
			return err
		}
		if fileURL != nil {
			preOrder.StaffResource = fileURL
		}
		notiStatus = notiBuilder.PreOrderNotifyCompensateDenied
	}

	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		zap.L().Error("Failed to update PreOrder state", zap.String("preorder_id", preOrderID.String()), zap.Error(err))
		return errors.New("failed to update PreOrder state: " + err.Error())
	}

	// Notification
	go func() {
		err = p.sendNotification(context.Background(), notiStatus, preOrder, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (p preOrderService) PreOrderOpeningChecker(ctx context.Context) (int, int, int) {
	var totalProcessed, failed, upCommingItems = 0, 0, 0
	// 0. Count all PAID items
	countFilter := func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN product_variants pv ON pv.id = pre_orders.variant_id").
			Joins("JOIN products p ON p.id = pv.product_id").
			Joins("JOIN limited_products lp ON lp.id = p.id").
			Where("pre_orders.status = ?", enum.PreOrderStatusPreOrdered).
			Where("lp.availability_start_date > ?", time.Now())
	}
	countItem, err := p.preOrderRepository.Count(ctx, countFilter)
	if err != nil {
		zap.L().Error("Failed to count upcomming pre-orders", zap.Error(err))
		return 0, 0, 0
	}
	upCommingItems = int(countItem)

	// 1. Get all pre-orders PRE_ORDERED where current time is after AvailabilityStartDate
	now := time.Now()
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN product_variants pv ON pv.id = pre_orders.variant_id").
			Joins("JOIN products p ON p.id = pv.product_id").
			Joins("JOIN limited_products lp ON lp.id = p.id").
			Where("pre_orders.status = ?", enum.PreOrderStatusPreOrdered).
			Where("lp.availability_start_date <= ?", now)
	}

	preOrders, total, err := p.preOrderRepository.GetAll(ctx, filter, []string{}, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch pre-orders for opening checker", zap.Error(err))
		return 0, 0, 0
	}
	totalProcessed = int(total)
	_ = preOrders
	for _, preOrder := range preOrders {
		var err error
		if preOrder.IsSelfPickedUp {
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusAwaitingPickup, uuid.UUID{}, nil, nil)
		} else {
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusInTransit, uuid.UUID{}, nil, nil)
		}
		if err != nil {
			msg := fmt.Sprintf("Failed to update pre-order status with ID: %s , Detail: %s", preOrder.ID.String(), err.Error())
			zap.L().Error(msg)
			failed++
		}
	}

	return totalProcessed, failed, upCommingItems
}

// validateAndGetRemainingOrder returns remaining allowed pre-order slots for a user+variant.
// It sums quantities in the DB (excluding CANCELLED) to avoid loading all records.
func (p preOrderService) validateAndGetRemainingOrder(ctx context.Context, userID, variantID string, maximumOrder int) (int, error) {
	if maximumOrder <= 0 {
		return 0, fmt.Errorf("invalid maximumOrder: %d", maximumOrder)
	}

	exclude := []enum.PreOrderStatus{
		enum.PreOrderStatusCancelled,
		enum.PreOrderStatusRefunded,
		enum.PreOrderStatusCompensated,
	}

	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ? AND variant_id = ? AND status NOT IN (?)", userID, variantID, exclude)
	}
	preorders, _, err := p.preOrderRepository.GetAll(ctx, filter, []string{}, 0, 0)
	if err != nil {
		return 0, err
	}

	totalHadBought := 0
	for _, preorder := range preorders {
		totalHadBought += preorder.Quantity
	}

	remaining := maximumOrder - int(totalHadBought)
	if remaining <= 0 {
		return 0, fmt.Errorf("you have reached the maximum number of pre-orders allowed for this product variant")
	}
	return remaining, nil
}

func (p preOrderService) PreserverOrder(ctx context.Context, request requests.PreOrderRequest, unitOfWork irepository.UnitOfWork, userID uuid.UUID) (*model.PreOrder, error) {
	var preOrder *model.PreOrder
	var remainedPreservableItem int
	now := time.Now()

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Validate user bank account
		creator, err := uow.Users().GetByID(ctx, userID, nil)
		if err != nil {
			return err
		}
		if creator.BankAccount == nil || creator.BankName == nil || creator.BankAccountHolder == nil {
			return fmt.Errorf("user bank information is incomplete, please update your bank details before placing a pre-order")
		}

		//1. validate variant and stocks/products
		includes := []string{"Product", "Product.Limited", "Product.Brand", "Product.Category"}
		variant, err := uow.ProductVariant().GetByID(ctx, request.VariantID, includes)
		if err != nil {
			return fmt.Errorf("variant %w not found", err)
		} else if err = ValidateVariantForPreOrder(*variant); err != nil {
			return err
		}

		// Check if this preorderable?
		if variant.Product.Limited != nil {
			// check if user already bought it
			limitItemQuantity := variant.Product.Limited.AchievableQuantity
			remainedPreservableItem, err = p.validateAndGetRemainingOrder(ctx, userID.String(), variant.ID.String(), limitItemQuantity)
			if err != nil {
				return err
			}
			//Check if can user buy this remained Item?
			if request.Quantity > remainedPreservableItem {
				return fmt.Errorf("you can only preorder %d more of this product variant", remainedPreservableItem)
			}

			// check date's validity
			premiereDate := variant.Product.Limited.PremiereDate
			startDate := variant.Product.Limited.AvailabilityStartDate
			endDate := variant.Product.Limited.AvailabilityEndDate
			//isPreOrderable := now.After(premiereDate) && now.Before(startDate) && now.Before(endDate)
			// PREORDER RANGE: [premiereDate, startDate)
			isPreOrderable := !now.Before(premiereDate) && now.Before(startDate)
			if !isPreOrderable {
				// not ready
				if now.Before(premiereDate) {
					remaining := premiereDate.Sub(now)
					return fmt.Errorf("the product is not ready to preorder, time remaining: %s", remaining)
				}

				// already overdue
				if now.After(endDate) || now.Equal(endDate) {
					return fmt.Errorf(
						"the product is overdue to preorder, last available date was %s",
						endDate.Format("2006-01-02 15:04:05"))
				}

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
		preOrder = requests.PreOrderRequest{}.ToModel(*creator, *address, *variant, now, remainedPreservableItem)
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

	includes := []string{"ProductVariant", "ProductVariant.Product", "ProductVariant.Product.Limited", "ProductVariant.Images"}

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

func NewPreOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry, paymentTransactionSvc iservice.PaymentTransactionService, stateTransferService iservice.StateTransferService, notificationService iservice.NotificationService) iservice.PreOrderService {
	return &preOrderService{
		config:                       cfg,
		db:                           dbRegistry.GormDatabase,
		preOrderRepository:           dbRegistry.PreOrderRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		userRepository:               dbRegistry.UserRepository,
		variantRepository:            dbRegistry.ProductVariantRepository,
		payOSProxy:                   registry.ProxiesRegistry.PayOSProxy,
		shippingAddressRepository:    dbRegistry.ShippingAddressRepository,
		ghnService:                   registry.ProxiesRegistry.GHNProxy,
		paymentTransactionService:    paymentTransactionSvc,
		stateTransferService:         stateTransferService,
		notificationService:          notificationService,
	}
}

// ----------------------------Validator-----------------------------//
func ValidateVariantForPreOrder(variant model.ProductVariant) error {
	//validate if product type is limited
	if variant.Product != nil && variant.Product.Type != enum.ProductTypeLimited {
		return fmt.Errorf("invalid product type for pre-order")
	}

	//validate if product is active
	if !variant.Product.IsActive {
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

func MovePreOrderStateUsingFSM(preorder *model.PreOrder, lp *model.LimitedProduct, user *model.User, newStatus enum.PreOrderStatus, reason *string) error {
	ctxState := &preordersm.PreOrderContext{
		State:          preordersm.NewPreOrderState(preorder.Status),
		PreOrder:       preorder,
		LimitedProduct: lp,
		ActionBy:       user,
	}
	nextState := preordersm.NewPreOrderState(newStatus)
	if err := ctxState.State.Next(ctxState, nextState); err != nil {
		zap.L().Error("PreOrder state transition validation failed", zap.Error(err))
		return fmt.Errorf("state transition not allowed: %w", err)
	}
	preorder.AddActionNote(*ctxState.GenerateActionNote(user, reason))
	return nil
}

func (p *preOrderService) lookupPreOrderWithLimitedProductAndUser(ctx context.Context, preorderID, actionBy uuid.UUID) (*model.PreOrder, *model.LimitedProduct, *model.User, error) {
	// 1) Load PreOrder
	preOrder, err := p.preOrderRepository.GetByID(ctx, preorderID, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	variantIncludes := []string{"Product", "Product.Limited"}
	variant, err := p.variantRepository.GetByID(ctx, preOrder.VariantID, variantIncludes)
	if err != nil {
		return nil, nil, nil, err
	}

	// If actionBy is zero value, treat as System user
	var user *model.User
	if actionBy == uuid.Nil {
		user = &model.User{
			ID:       uuid.UUID{},
			FullName: p.config.AdminConfig.SystemName,
			Email:    p.config.AdminConfig.SystemEmail,
		}
	} else {
		user, err = p.userRepository.GetByID(ctx, actionBy, nil)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return preOrder, variant.Product.Limited, user, nil
}

func (p *preOrderService) sendNotification(ctx context.Context, notiStatus notiBuilder.PreOrderNotificationType, preOrder *model.PreOrder, actionBy *model.User) error {
	//This contains many payloads to different receivers
	payloads, err := notiBuilder.BuildPreOrderNotifications(ctx, *p.config, p.db, notiStatus, preOrder, actionBy)
	if err != nil {
		zap.L().Debug("no notification builder for order status", zap.Error(err))
		return nil
	}

	for _, pl := range payloads {
		_, err = p.notificationService.CreateAndPublishNotification(ctx, &pl)
		if err != nil {
			zap.L().Error("Failed to send notification", zap.Error(err))
			return err
		}
	}
	return nil
}
