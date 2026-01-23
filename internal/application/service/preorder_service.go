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
	"core-backend/internal/infrastructure/asynq"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm/clause"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type preOrderService struct {
	config *config.AppConfig
	db     *gorm.DB
	// preOrderRepository           irepository.GenericRepository[model.PreOrder]
	preOrderRepository           irepository.PreOrderRepository
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	userRepository               irepository.GenericRepository[model.User]
	variantRepository            irepository.GenericRepository[model.ProductVariant]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iproxies.GHNProxy
	paymentTransactionService    iservice.PaymentTransactionService
	stateTransferService         iservice.StateTransferService
	notificationService          iservice.NotificationService
	scheduleRepository           irepository.ScheduleRepository
	taskScheduler                *asynq.AsynqClient
	asynqConfig                  *config.AsynqConfig
	unitOfWork                   irepository.UnitOfWork
}

func (p preOrderService) PreOrderOpeningManualTrigger(ctx context.Context, preOrderID, actionBy uuid.UUID) error {
	//TODO implement me
	panic("implement me")
}

func (p preOrderService) GetPreOrderPricePercentage(ctx context.Context, preOrderID uuid.UUID) ([]responses.PriceBreakdown, error) {
	var breakdowns []responses.PriceBreakdown

	include := []string{
		"ProductVariant",
		"ProductVariant.Product",
		"ProductVariant.Product.Task",
		"ProductVariant.Product.Task.Milestone",
		"ProductVariant.Product.Task.Milestone.Campaign",
		"ProductVariant.Product.Task.Milestone.Campaign.Contract",
	}

	preOrder, err := p.preOrderRepository.GetByID(ctx, preOrderID, include)
	if err != nil {
		return nil, err
	}

	financialTerm := p.getFinancialTerms(preOrder)

	kolPercentVal, isKOLPctExists, err := getJSONKey(*financialTerm, "profit_split_kol_percent")
	if err != nil {
		return nil, fmt.Errorf("failed to parse KOL percentage: %w", err)
	}
	if !isKOLPctExists {
		return breakdowns, nil
	}

	companyPercentVal, isCompPctExists, err := getJSONKey(*financialTerm, "profit_split_company_percent")
	if err != nil {
		return nil, fmt.Errorf("failed to parse company percentage: %w", err)
	}
	if !isCompPctExists {
		return breakdowns, nil
	}

	// Convert interface{} to int (JSON numbers come as float64)
	kolPercent := toInt(kolPercentVal)
	companyPercent := toInt(companyPercentVal)

	// Calculate amounts based on item subtotal
	itemTotal := preOrder.TotalAmount
	kolAmount := itemTotal * float64(kolPercent) / 100.0
	companyAmount := itemTotal * float64(companyPercent) / 100.0

	breakdown := responses.PriceBreakdown{
		ItemID:            preOrder.ID,
		CompanyPercentage: companyPercent,
		KOLPercentage:     kolPercent,
		CompanyAmount:     companyAmount,
		KOLAmount:         kolAmount,
	}

	breakdowns = append(breakdowns, breakdown)
	return breakdowns, nil
}

func (p preOrderService) getFinancialTerms(item *model.PreOrder) *datatypes.JSON {
	if item != nil ||
		item.ProductVariant == nil ||
		item.ProductVariant.Product == nil ||
		item.ProductVariant.Product.Task == nil ||
		item.ProductVariant.Product.Task.Milestone == nil ||
		item.ProductVariant.Product.Task.Milestone.Campaign == nil ||
		item.ProductVariant.Product.Task.Milestone.Campaign.Contract == nil {

	}
	return &item.ProductVariant.Product.Task.Milestone.Campaign.Contract.FinancialTerms
}

// calculatePreOrderRevenues computes company and KOL revenue for a preorder
// PreOrders are always for LIMITED products, so we use contract financial terms
func (p preOrderService) calculatePreOrderRevenues(ctx context.Context, preOrder *model.PreOrder) (companyRevenue, kolRevenue float64) {
	if preOrder == nil {
		return 0, 0
	}

	// Load preorder with full chain to get financial terms
	include := []string{
		"ProductVariant",
		"ProductVariant.Product",
		"ProductVariant.Product.Task",
		"ProductVariant.Product.Task.Milestone",
		"ProductVariant.Product.Task.Milestone.Campaign",
		"ProductVariant.Product.Task.Milestone.Campaign.Contract",
	}

	preOrderWithDetails, err := p.preOrderRepository.GetByID(ctx, preOrder.ID, include)
	if err != nil || preOrderWithDetails == nil {
		zap.L().Warn("Failed to load preorder details for revenue calculation", zap.Error(err), zap.String("preOrderID", preOrder.ID.String()))
		return 0, preOrder.TotalAmount
	}

	financialTerm := p.getFinancialTerms(preOrderWithDetails)
	if financialTerm == nil {
		// No contract found, assume KOL gets all
		return 0, preOrder.TotalAmount
	}

	kolPercentVal, isKOLPctExists, err := getJSONKey(*financialTerm, "profit_split_kol_percent")
	if err != nil || !isKOLPctExists {
		return 0, preOrder.TotalAmount
	}

	companyPercentVal, isCompPctExists, err := getJSONKey(*financialTerm, "profit_split_company_percent")
	if err != nil || !isCompPctExists {
		return 0, preOrder.TotalAmount
	}

	kolPercent := toInt(kolPercentVal)
	companyPercent := toInt(companyPercentVal)

	kolRevenue = preOrder.TotalAmount * float64(kolPercent) / 100.0
	companyRevenue = preOrder.TotalAmount * float64(companyPercent) / 100.0

	return companyRevenue, kolRevenue
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
	if err = helper.WithTransaction(ctx, p.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.PreOrder().Update(ctx, preOrder); err != nil {
			zap.L().Error("Failed to update PreOrder state", zap.Error(err))
			return err
		}

		// Record refunded payment transaction
		negativePayment := &model.PaymentTransaction{
			ReferenceID:     preOrder.ID,
			ReferenceType:   enum.PaymentTransactionReferenceTypePreOrder,
			Amount:          utils.PtrOrNil(-preOrder.TotalAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         utils.PtrOrNil(actionBy),
			ReceivedByID:    utils.PtrOrNil(preOrder.UserID),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			zap.L().Error("Failed to add negative payment transaction during refund proof review", zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		zap.L().Error("Failed to update PreOrder state", zap.Error(err))
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
	if err = helper.WithTransaction(ctx, p.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.PreOrder().Update(ctx, preOrder); err != nil {
			zap.L().Error("Failed to update PreOrder state",
				zap.String("preorder_id", preOrderID.String()),
				zap.Error(err))
			return err
		}

		// Record refunded payment transaction
		negativePayment := &model.PaymentTransaction{
			ReferenceID:     preOrderID,
			ReferenceType:   enum.PaymentTransactionReferenceTypePreOrder,
			Amount:          utils.PtrOrNil(-preOrder.TotalAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         utils.PtrOrNil(actionBy),
			ReceivedByID:    utils.PtrOrNil(preOrder.UserID),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			zap.L().Error("Failed to add negative payment transaction during refund proof review",
				zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		zap.L().Error("Failed to update PreOrder state",
			zap.String("preorder_id", preOrderID.String()),
			zap.Error(err))
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

	// publish delay message to asynq

	if err != nil {
		return err
	}
	return nil
}

// UpdateGHNOrderCode updates the GHN order code for a pre-order
func (p preOrderService) UpdateGHNOrderCode(ctx context.Context, preOrderID uuid.UUID, ghnOrderCode string) error {
	preOrder, err := p.preOrderRepository.GetByID(ctx, preOrderID, nil)
	if err != nil {
		return fmt.Errorf("failed to find pre-order: %w", err)
	}

	preOrder.GHNOrderCode = &ghnOrderCode

	if err := p.preOrderRepository.Update(ctx, preOrder); err != nil {
		zap.L().Error("Failed to update PreOrder GHN order code",
			zap.String("preorder_id", preOrderID.String()),
			zap.String("ghn_order_code", ghnOrderCode),
			zap.Error(err))
		return fmt.Errorf("failed to update pre-order GHN order code: %w", err)
	}

	zap.L().Info("Updated PreOrder GHN order code",
		zap.String("preorder_id", preOrderID.String()),
		zap.String("ghn_order_code", ghnOrderCode))

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

	if err = helper.WithTransaction(ctx, p.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.PreOrder().Update(ctx, preOrder); err != nil {
			zap.L().Error("Failed to update PreOrder state",
				zap.String("preorder_id", preOrderID.String()),
				zap.Error(err))
			return errors.New("failed to update PreOrder state: " + err.Error())
		}

		if !isApproved {
			return nil
		}

		// Record compensated payment transaction if approved
		negativePayment := &model.PaymentTransaction{
			ReferenceID:     preOrder.ID,
			ReferenceType:   enum.PaymentTransactionReferenceTypePreOrder,
			Amount:          utils.PtrOrNil(-preOrder.TotalAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         utils.PtrOrNil(actionBy),
			ReceivedByID:    utils.PtrOrNil(preOrder.UserID),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			zap.L().Error("Failed to add negative payment transaction during compensation processing",
				zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		zap.L().Error("Failed to update PreOrder state",
			zap.String("preorder_id", preOrderID.String()),
			zap.Error(err))
		return err
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
			// Self-pickup: just move to AwaitingPickup status
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusAwaitingPickup, uuid.UUID{}, nil, nil)
		} else {
			// err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusInTransit, uuid.UUID{}, nil, nil)
			// Delivery: Create GHN order first, then move to Shipped status
			ghnResponse, ghnErr := p.ghnService.CreatePreOrder(ctx, preOrder.ID)
			if ghnErr != nil {
				zap.L().Error("Failed to create GHN order for pre-order",
					zap.String("preorder_id", preOrder.ID.String()),
					zap.Error(ghnErr))
				failed++
				continue
			}

			// Update the pre-order with the GHN order code
			if ghnResponse != nil && ghnResponse.OrderCode != "" {
				if updateErr := p.UpdateGHNOrderCode(ctx, preOrder.ID, ghnResponse.OrderCode); updateErr != nil {
					zap.L().Error("Failed to update GHN order code for pre-order",
						zap.String("preorder_id", preOrder.ID.String()),
						zap.String("ghn_order_code", ghnResponse.OrderCode),
						zap.Error(updateErr))
					// Continue even if update fails - the GHN order was created
				}
			}

			// Move to Shipped status (will be updated to InTransit by GHN webhook)
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusShipped, uuid.UUID{}, nil, nil)
		}
		if err != nil {
			msg := fmt.Sprintf("Failed to update pre-order status with ID: %s , Detail: %s", preOrder.ID.String(), err.Error())
			zap.L().Error(msg)
			failed++
		}
	}

	return totalProcessed, failed, upCommingItems
}

func (p *preOrderService) OpeningPreOrderEarly(ctx context.Context, uow irepository.UnitOfWork, productID uuid.UUID, updatedBy uuid.UUID) error {
	zap.L().Info("Opening pre-orders early for product",
		zap.String("product_id", productID.String()))
	currentTime := time.Now().Add(-1 * time.Minute) // backdate 1 minute to avoid timezone issues

	// Update Limited Productd availability_start_date to current time
	if err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err := uow.LimitedProducts().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", productID)
		}, map[string]any{"availability_start_date": currentTime, "premiere_date": currentTime}); err != nil {
			zap.L().Error("Failed to update limited product availability start date", zap.Error(err))
			return fmt.Errorf("failed to update limited product availability start date: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	upcomingFilter := func(db *gorm.DB) *gorm.DB {
		return db.Joins("JOIN product_variants pv ON pv.id = pre_orders.variant_id").
			Joins("JOIN products p ON p.id = pv.product_id").
			Joins("JOIN limited_products lp ON lp.id = p.id").
			Where("pre_orders.status = ?", enum.PreOrderStatusPreOrdered).
			Where("lp.availability_start_date >= ?", currentTime).
			Where("p.id = ?", productID)
	}
	upcomingPreOrders, total, err := p.preOrderRepository.GetAll(ctx, upcomingFilter, []string{}, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch upcoming pre-orders for opening early", zap.Error(err))
		return fmt.Errorf("failed to fetch pre-orders for opening checker: %w", err)
	}

	failedPreOrdersIDs := make([]uuid.UUID, 0)
	for _, preOrder := range upcomingPreOrders {
		if preOrder.IsSelfPickedUp {
			// Self-pickup: just move to AwaitingPickup status
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusAwaitingPickup, uuid.UUID{}, nil, nil)
		} else {
			err = p.stateTransferService.MovePreOrderToState(ctx, preOrder.ID, enum.PreOrderStatusInTransit, uuid.UUID{}, nil, nil)
		}
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to update pre-order status with ID: %s , Detail: %s", preOrder.ID.String(), err.Error()))
			failedPreOrdersIDs = append(failedPreOrdersIDs, preOrder.ID)
		}
	}

	zap.L().Info("Opening pre-orders early completed",
		zap.Int64("total_pre_orders_processed", total),
		zap.Int("total_pre_orders_failed", len(failedPreOrdersIDs)),
		zap.Any("failed_pre_orders_ids", failedPreOrdersIDs),
		zap.Duration("duration", time.Since(currentTime)))

	return nil
}

// validateAndGetRemainingOrder returns remaining allowed pre-order slots for user to a variant.
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
			//0. Check current orderable stats
			if variant.PreOrderLimit == nil || variant.PreOrderCount == nil {
				zap.L().Debug("Invalid data format: PreOrderLimit or PreOrderCount is nil")
				return fmt.Errorf("pre-order limit or PreOrderCount is nil")
			}
			// 1. general system limit
			generalRemain := *variant.PreOrderLimit - *variant.PreOrderCount
			if generalRemain <= 0 {
				return fmt.Errorf("this product is out of stock for this early order, please kindly wait for the nearest start date")
			}
			if request.Quantity > generalRemain {
				return fmt.Errorf("only %d pre-order slots remaining for this product variant", generalRemain)
			}

			// 2. user achievable limit
			achievable := variant.Product.Limited.AchievableQuantity
			userRemain, err := p.validateAndGetRemainingOrder(ctx, userID.String(), variant.ID.String(), achievable)
			if err != nil {
				return err
			}

			// 3. calculate actual remaining preservable items
			actualRemain := min(userRemain, generalRemain)
			if request.Quantity > actualRemain {
				return fmt.Errorf("only %d pre-order slots remaining for this product variant", actualRemain)
			}

			// check preorder time
			premiereDate := variant.Product.Limited.PremiereDate
			startDate := variant.Product.Limited.AvailabilityStartDate
			endDate := variant.Product.Limited.AvailabilityEndDate
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
		}

		//1.1 validate shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, request.AddressID, []string{})
		if err != nil {
			return fmt.Errorf("failed to get shipping address: %w", err)
		}

		// 2. Build persistent model -> minus stock, create pre-order record
		// 2.1 build pre-order model
		preOrder = requests.PreOrderRequest{}.ToModel(*creator, *address, *variant, now, request.Quantity)
		// 2.2 minus stock
		if variant.CurrentStock == nil {
			return fmt.Errorf("variant stock is nil")
		}
		_, err = p.AtomicDecreasePreOrder(
			ctx,
			variant.ID,
			request.Quantity,
		)
		if err != nil {
			return fmt.Errorf("failed to reserve stock: %w", err)
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

func (p preOrderService) AtomicDecreasePreOrder(
	ctx context.Context,
	variantID uuid.UUID,
	qty int,
) (*model.ProductVariant, error) {

	var updated model.ProductVariant

	tx := p.db.WithContext(ctx).Model(&model.ProductVariant{}).
		Where("id = ? AND current_stock >= ?", variantID, qty).
		Updates(map[string]interface{}{
			"current_stock":   gorm.Expr("current_stock - ?", qty),
			"pre_order_count": gorm.Expr("pre_order_count + ?", qty),
		}).
		Clauses(clause.Returning{})

	if tx.Error != nil {
		return nil, tx.Error
	}

	if tx.RowsAffected == 0 {
		return nil, fmt.Errorf("not enough stock")
	}

	// fill updated variant
	if err := tx.Scan(&updated).Error; err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *preOrderService) GetPreOrdersByUserIDWithPagination(
	ctx context.Context,
	userID uuid.UUID,
	limit, page int,
	search string,
	statuses []string,
	createdFrom, createdTo string,
) ([]responses.PreOrderResponse, int, error) {

	// -------------------------------
	// Pagination
	// -------------------------------
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	includes := []string{
		"ProductVariant.Images",
		"Brand",
		"Category",
		"ProductVariant.Product",
		"ProductVariant.Product.Limited",
	}

	// ---- FILTER ----
	filter := func(db *gorm.DB) *gorm.DB {
		q := db.Where("user_id = ?", userID)

		if len(statuses) > 0 {
			q = q.Where("status IN ?", statuses)
		}

		if search != "" {
			like := "%" + search + "%"
			q = q.Where("(product_name ILIKE ? OR email ILIKE ? OR full_name ILIKE ?)", like, like, like)
		}

		if createdFrom != "" {
			q = q.Where("created_at >= ?", createdFrom)
		}

		if createdTo != "" {
			q = q.Where("created_at <= ?", createdTo)
		}

		return q.Order("created_at DESC")
	}

	// ---- LẤY DATA ----
	preOrders, total, err := s.preOrderRepository.GetAll(ctx, filter, includes, limit, page)
	if err != nil {
		return nil, 0, err
	}

	// ---- TỐI ƯU: lấy tất cả PaymentTransaction newest bằng 1 query ----
	ids := make([]uuid.UUID, 0, len(preOrders))
	for _, po := range preOrders {
		ids = append(ids, po.ID)
	}

	var payments []model.PaymentTransaction

	if len(ids) > 0 {
		s.db.
			Raw(`
                SELECT DISTINCT ON (reference_id) *
                FROM payment_transactions
                WHERE reference_id IN ?
                ORDER BY reference_id, created_at DESC
            `, ids).
			Scan(&payments)
	}

	// ---- MAP payments vào preorder ----
	pmMap := map[uuid.UUID]*model.PaymentTransaction{}
	for _, pm := range payments {
		pmCopy := pm
		pmMap[pm.ReferenceID] = &pmCopy
	}

	resp := make([]responses.PreOrderResponse, 0, len(preOrders))
	for _, po := range preOrders {
		resp = append(resp, responses.PreOrderResponse{}.ToPreOrderResponse(po, pmMap[po.ID]))
	}

	return resp, int(total), nil
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
		scheduleRepository:           dbRegistry.ScheduleRepository,
		taskScheduler:                registry.AsynqClient,
		asynqConfig:                  &cfg.Asynq,
		unitOfWork:                   registry.UnitOfWork,
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
	containerTypeStr := ""
	if preOrder.ContainerType != nil {
		containerTypeStr = utils.ToTitleCase(*preOrder.ContainerType)
	}
	dispenserTypeStr := ""
	if preOrder.DispenserType != nil {
		dispenserTypeStr = utils.ToTitleCase(*preOrder.DispenserType)
	}
	variantPropConcat := fmt.Sprintf("%v", preOrder.Capacity) +
		utils.ToTitleCase(*preOrder.CapacityUnit) + " - " +
		containerTypeStr + " - " +
		dispenserTypeStr

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
	containerTypeStr := ""
	if preOrder.ContainerType != nil {
		containerTypeStr = utils.ToTitleCase(*preOrder.ContainerType)
	}
	dispenserTypeStr := ""
	if preOrder.DispenserType != nil {
		dispenserTypeStr = utils.ToTitleCase(*preOrder.DispenserType)
	}
	variantPropConcat := fmt.Sprintf("%v", preOrder.Capacity) +
		utils.ToTitleCase(*preOrder.CapacityUnit) + " - " +
		containerTypeStr + " - " +
		dispenserTypeStr

	// Build readable item name (e.g. "Shampoo (250ML - Bottle - Spray)")
	variantName := utils.ToTitleCase(preOrder.ProductVariant.Product.Name) + fmt.Sprintf(" (%s)", variantPropConcat)
	mappedModel := requests.PaymentItemRequest{
		Name:     variantName,
		Quantity: preOrder.Quantity,
		Price:    int64(preOrder.UnitPrice),
	}

	items = append(items, mappedModel)
	total += int64(preOrder.UnitPrice) * int64(preOrder.Quantity)
	return items, total
}

// GetStaffAvailablePreOrdersWithPagination returns preorders for staff with same filtering/search as staff orders
func (s preOrderService) GetStaffAvailablePreOrdersWithPagination(
	limit, page int,
	search, fullName, phone, provinceID, districtID, wardCode, createdFrom, createdTo, brandID string,
	statuses []string,
) ([]responses.PreOrderResponse, int, error) {

	ctx := context.Background()
	preOrders, total, err := s.preOrderRepository.GetStaffAvailablePreOrdersWithPagination(ctx, limit, page, search, fullName, phone, provinceID, districtID, wardCode, createdFrom, createdTo, brandID, statuses)
	if err != nil {
		return nil, 0, err
	}

	// If no preorders - return early
	if len(preOrders) == 0 {
		return []responses.PreOrderResponse{}, total, nil
	}

	if s.paymentTransactionRepository == nil || s.paymentTransactionRepository.DB() == nil {
		zap.L().Warn("paymentTransactionRepository not available, returning preorders without transaction enrichment")
		// Convert to response without payment enrichment
		return toPreOrderResponseList(preOrders, map[uuid.UUID]model.PaymentTransaction{}), total, nil
	}

	// Collect preorder IDs
	preOrderIDs := make([]uuid.UUID, 0, len(preOrders))
	for _, po := range preOrders {
		preOrderIDs = append(preOrderIDs, po.ID)
	}

	// Load payment transactions referencing these preorders
	var transactions []model.PaymentTransaction
	if err := s.paymentTransactionRepository.DB().WithContext(ctx).
		Model(&model.PaymentTransaction{}).
		Where("reference_type = ? AND reference_id IN (?)", enum.PaymentTransactionReferenceTypePreOrder, preOrderIDs).
		Find(&transactions).Error; err != nil {
		zap.L().Error("Failed to fetch payment transactions for staff preorders", zap.Error(err))
		return toPreOrderResponseList(preOrders, map[uuid.UUID]model.PaymentTransaction{}), total, nil
	}

	// Choose latest transaction per preorder by UpdatedAt
	paymentsMap := make(map[uuid.UUID]model.PaymentTransaction)
	for _, tx := range transactions {
		existing, ok := paymentsMap[tx.ReferenceID]
		if !ok || tx.UpdatedAt.After(existing.UpdatedAt) {
			paymentsMap[tx.ReferenceID] = tx
		}
	}

	// Attach latest payment info back to preorders (transient fields) - keep for compatibility but not required for DTO
	for i := range preOrders {
		if pt, ok := paymentsMap[preOrders[i].ID]; ok {
			preOrders[i].PaymentID = &pt.ID
			if pt.PayOSMetadata != nil {
				preOrders[i].PaymentBin = &pt.PayOSMetadata.Bin
			}
		}
	}

	// Calculate company and KOL revenue for each preorder
	companyRevenues := make(map[uuid.UUID]float64)
	kolRevenues := make(map[uuid.UUID]float64)

	for _, preOrder := range preOrders {
		companyRev, kolRev := s.calculatePreOrderRevenues(ctx, &preOrder)
		companyRevenues[preOrder.ID] = companyRev
		kolRevenues[preOrder.ID] = kolRev
	}

	// Convert model preorders to response DTOs using payment map and revenue maps
	preOrderResponses := toPreOrderResponseListWithRevenue(preOrders, paymentsMap, companyRevenues, kolRevenues)

	return preOrderResponses, total, nil
}

// toPreOrderResponseList converts a list of PreOrder models to PreOrderResponse DTOs
func toPreOrderResponseList(preOrders []model.PreOrder, paymentsMap map[uuid.UUID]model.PaymentTransaction) []responses.PreOrderResponse {
	result := make([]responses.PreOrderResponse, 0, len(preOrders))
	for _, po := range preOrders {
		var pm *model.PaymentTransaction
		if pt, ok := paymentsMap[po.ID]; ok {
			pm = &pt
		}
		result = append(result, responses.PreOrderResponse{}.ToPreOrderResponse(po, pm))
	}
	return result
}

// toPreOrderResponseListWithRevenue converts a list of PreOrder models to PreOrderResponse DTOs with revenue data
func toPreOrderResponseListWithRevenue(preOrders []model.PreOrder, paymentsMap map[uuid.UUID]model.PaymentTransaction, companyRevenues, kolRevenues map[uuid.UUID]float64) []responses.PreOrderResponse {
	result := make([]responses.PreOrderResponse, 0, len(preOrders))
	for _, po := range preOrders {
		var pm *model.PaymentTransaction
		if pt, ok := paymentsMap[po.ID]; ok {
			pm = &pt
		}
		resp := responses.PreOrderResponse{}.ToPreOrderResponse(po, pm)

		// Set revenue fields if available
		if companyRev, ok := companyRevenues[po.ID]; ok {
			resp.CompanyRevenue = &companyRev
		}
		if kolRev, ok := kolRevenues[po.ID]; ok {
			resp.KOLRevenue = &kolRev
		}

		result = append(result, resp)
	}
	return result
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
