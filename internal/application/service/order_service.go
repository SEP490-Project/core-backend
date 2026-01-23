package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	notiBuilder "core-backend/internal/application/service/notification_builder"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/ordersm"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type orderService struct {
	config                       *config.AppConfig
	db                           *gorm.DB
	orderRepository              irepository.OrderRepository
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	userRepository               irepository.GenericRepository[model.User]
	preOrderRepository           irepository.GenericRepository[model.PreOrder]
	ghnProxy                     iproxies.GHNProxy
	paymentTransactionService    iservice.PaymentTransactionService
	notificationService          iservice.NotificationService
	unitOfWork                   irepository.UnitOfWork
}

func (o *orderService) GetOrderPricePercentage(ctx context.Context, orderID uuid.UUID, orderType string) ([]responses.PriceBreakdown, error) {
	var breakdowns []responses.PriceBreakdown

	if orderType == "LIMITED" {
		orderIncludes := []string{
			"OrderItems",
			"OrderItems.Variant",
			"OrderItems.Variant.Product",
			"OrderItems.Variant.Product.Task",
			"OrderItems.Variant.Product.Task.Milestone",
			"OrderItems.Variant.Product.Task.Milestone.Campaign",
			"OrderItems.Variant.Product.Task.Milestone.Campaign.Contract",
		}

		order, err := o.orderRepository.GetByID(ctx, orderID, orderIncludes)
		if err != nil {
			return nil, err
		}

		if order == nil {
			return nil, errors.New("order not found")
		}

		for _, item := range order.OrderItems {
			financialTerm := getFinancialTerms(&item)
			if financialTerm == nil {
				continue
			}

			kolPercentVal, isKOLPctExists, err := getJSONKey(*financialTerm, "profit_split_kol_percent")
			if err != nil {
				return nil, fmt.Errorf("failed to parse KOL percentage: %w", err)
			}
			if !isKOLPctExists {
				continue
			}

			companyPercentVal, isCompPctExists, err := getJSONKey(*financialTerm, "profit_split_company_percent")
			if err != nil {
				return nil, fmt.Errorf("failed to parse company percentage: %w", err)
			}
			if !isCompPctExists {
				continue
			}

			// Convert interface{} to int (JSON numbers come as float64)
			kolPercent := toInt(kolPercentVal)
			companyPercent := toInt(companyPercentVal)

			// Calculate amounts based on item subtotal
			itemTotal := item.Subtotal
			kolAmount := itemTotal * float64(kolPercent) / 100.0
			companyAmount := itemTotal * float64(companyPercent) / 100.0

			breakdown := responses.PriceBreakdown{
				ItemID:            item.ID,
				CompanyPercentage: companyPercent,
				KOLPercentage:     kolPercent,
				CompanyAmount:     companyAmount,
				KOLAmount:         kolAmount,
			}

			breakdowns = append(breakdowns, breakdown)
		}

	} else if orderType == "STANDARD" {
		orderIncludes := []string{
			"OrderItems",
		}
		// Implementation for STANDARD orders can be added here
		order, err := o.orderRepository.GetByID(ctx, orderID, orderIncludes)
		if err != nil {
			return nil, err
		}

		if order == nil {
			return nil, errors.New("order not found")
		}

		for _, item := range order.OrderItems {
			breakdown := responses.PriceBreakdown{
				ItemID:            item.ID,
				CompanyPercentage: 0,
				KOLPercentage:     100,
				CompanyAmount:     0,
				KOLAmount:         item.Subtotal,
			}
			breakdowns = append(breakdowns, breakdown)
		}

	}

	return breakdowns, nil
}

// toInt converts an interface{} (typically float64 from JSON) to int
func toInt(val interface{}) int {
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 0
	}
}

func getFinancialTerms(item *model.OrderItem) *datatypes.JSON {
	if item == nil ||
		item.Variant.ID == uuid.Nil ||
		item.Variant.Product == nil ||
		item.Variant.Product.Task == nil ||
		item.Variant.Product.Task.Milestone == nil ||
		item.Variant.Product.Task.Milestone.Campaign == nil ||
		item.Variant.Product.Task.Milestone.Campaign.Contract == nil {
		return nil
	}

	return &item.Variant.Product.Task.Milestone.Campaign.Contract.FinancialTerms
}

func getJSONKey(j datatypes.JSON, key string) (interface{}, bool, error) {
	if len(j) == 0 {
		return nil, false, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(j, &data); err != nil {
		return nil, false, err
	}

	val, ok := data[key]
	return val, ok, nil
}

// calculateOrderRevenues computes company and KOL revenue for an order
// For STANDARD orders: KOL gets 100% of total, company gets 0
// For LIMITED orders: revenue split based on contract financial terms
func (o *orderService) calculateOrderRevenues(ctx context.Context, order *model.Order) (companyRevenue, kolRevenue float64) {
	if order == nil {
		return 0, 0
	}

	// For STANDARD orders, KOL gets 100%
	if order.OrderType == string(enum.ProductTypeStandard) {
		return 0, order.TotalAmount
	}

	// For LIMITED orders, need to load contract financial terms
	if order.OrderType != string(enum.ProductTypeLimited) {
		// Unknown order type, default to KOL gets all
		return 0, order.TotalAmount
	}

	// Load order with full chain to get financial terms
	orderIncludes := []string{
		"OrderItems",
		"OrderItems.Variant",
		"OrderItems.Variant.Product",
		"OrderItems.Variant.Product.Task",
		"OrderItems.Variant.Product.Task.Milestone",
		"OrderItems.Variant.Product.Task.Milestone.Campaign",
		"OrderItems.Variant.Product.Task.Milestone.Campaign.Contract",
	}

	orderWithDetails, err := o.orderRepository.GetByID(ctx, order.ID, orderIncludes)
	if err != nil || orderWithDetails == nil {
		zap.L().Warn("Failed to load order details for revenue calculation", zap.Error(err), zap.String("orderID", order.ID.String()))
		return 0, order.TotalAmount
	}

	// Calculate revenue from all order items
	for _, item := range orderWithDetails.OrderItems {
		financialTerm := getFinancialTerms(&item)
		if financialTerm == nil {
			// No contract found, assume KOL gets all for this item
			kolRevenue += item.Subtotal
			continue
		}

		kolPercentVal, isKOLPctExists, err := getJSONKey(*financialTerm, "profit_split_kol_percent")
		if err != nil || !isKOLPctExists {
			kolRevenue += item.Subtotal
			continue
		}

		companyPercentVal, isCompPctExists, err := getJSONKey(*financialTerm, "profit_split_company_percent")
		if err != nil || !isCompPctExists {
			kolRevenue += item.Subtotal
			continue
		}

		kolPercent := toInt(kolPercentVal)
		companyPercent := toInt(companyPercentVal)

		kolRevenue += item.Subtotal * float64(kolPercent) / 100.0
		companyRevenue += item.Subtotal * float64(companyPercent) / 100.0
	}

	return companyRevenue, kolRevenue
}

func (o *orderService) ObligateEarlyRefund(ctx context.Context, orderID, actionBy uuid.UUID, reason, fileURL *string) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, actionBy)
	if err != nil {
		return err
	}
	//If this after user action a stanbytime?
	//standByMinutes := o.config.AdminConfig.CensorshipIntervalMinutes
	//isAllow := order.UpdatedAt.Add(time.Duration(standByMinutes) * time.Minute).After(time.Now())
	//if isAllow {
	//	msg := fmt.Sprintf("You can only allow to do this action after %d mins after user action, remaining time: %s", standByMinutes, time.Until(order.UpdatedAt.Add(time.Duration(standByMinutes)*time.Minute)).String())
	//	return errors.New(msg)
	//}

	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusRefunded, reason)
	if err != nil {
		return err
	}
	order.StaffResource = fileURL

	go func() {
		err = o.sendNotification(context.Background(), notiBuilder.OrderNotifyObligateRefund, order, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	if err = helper.WithTransaction(ctx, o.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = o.orderRepository.Update(ctx, order); err != nil {
			zap.L().Error("Failed to update order during obligate refund", zap.Error(err), zap.String("orderID", order.ID.String()))
			return err
		}

		// Record refunded payment transaction
		negativePayment := &model.PaymentTransaction{
			ReferenceID:     order.ID,
			ReferenceType:   enum.PaymentTransactionReferenceTypeOrder,
			Amount:          utils.PtrOrNil(-order.TotalAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         utils.PtrOrNil(actionBy),
			ReceivedByID:    utils.PtrOrNil(order.UserID),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			zap.L().Error("Failed to add negative payment transaction during refund proof review", zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		zap.L().Error("Transaction failed during obligate refund",
			zap.String("orderID", order.ID.String()),
			zap.Error(err))
		return err
	}

	return nil
}

func (o *orderService) RequestCompensation(ctx context.Context, orderID, actionBy uuid.UUID, reason, fileURL *string) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, actionBy)
	if err != nil {
		return err
	}
	//Check if compensation has already been requested
	if order.ActionNotes != nil {
		for _, note := range *order.ActionNotes {
			if note.ActionType == enum.OrderStatusCompensateRequested {
				return errors.New("you've already requested compensation for this order")
			}
		}
	}
	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusCompensateRequested, reason)
	if err != nil {
		return err
	}
	order.UserResource = fileURL
	go func() {
		err = o.sendNotification(context.Background(), notiBuilder.OrderNotifyCompensateRequested, order, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()
	return o.orderRepository.Update(ctx, order)
}

func (o *orderService) ProcessCompensation(ctx context.Context, orderID, actionBy uuid.UUID, isApproved bool, reason, fileURL *string) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, actionBy)
	if err != nil {
		return err
	}

	if !isApproved {
		err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusDelivered, reason)
		if err != nil {
			return err
		}
		if utils.NotEmptyOrNil(fileURL) {
			order.StaffResource = fileURL
		}
		go func() {
			err = o.sendNotification(context.Background(), notiBuilder.OrderNotifyCompensationDenied, order, user)
			if err != nil {
				zap.L().Error(err.Error())
			}
		}()

		return o.orderRepository.Update(ctx, order)

	} else {
		err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusCompensated, reason)
		if err != nil {
			return err
		}
		order.StaffResource = fileURL
		go func() {
			err = o.sendNotification(context.Background(), notiBuilder.OrderNotifyCompensated, order, user)
			if err != nil {
				zap.L().Error(err.Error())
			}
		}()
		return helper.WithTransaction(ctx, o.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
			if err = uow.Order().Update(ctx, order); err != nil {
				zap.L().Error("Failed to update order during compensate", zap.Error(err), zap.String("orderID", order.ID.String()))
				return err
			}

			if !isApproved {
				return nil
			}

			negativePayment := &model.PaymentTransaction{
				ReferenceID:     order.ID,
				ReferenceType:   enum.PaymentTransactionReferenceTypeOrder,
				Amount:          utils.PtrOrNil(-order.TotalAmount),
				Status:          enum.PaymentTransactionStatusRefunded,
				Method:          enum.ContractPaymentMethodBankTransfer.String(),
				TransactionDate: time.Now(),
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
				PayerID:         utils.PtrOrNil(actionBy),
				ReceivedByID:    utils.PtrOrNil(order.UserID),
			}
			if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
				zap.L().Error("Failed to add negative payment transaction during refund proof review", zap.Error(err))
				return err
			}

			return nil
		})
	}
}

func (o *orderService) RequestEarlyRefund(ctx context.Context, orderID, actionBy uuid.UUID, requestTime time.Time) error {
	_ = requestTime
	order, user, err := o.lookupOrderAndUser(ctx, orderID, actionBy)
	if err != nil {
		return err
	}

	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusRefundRequested, nil)
	if err != nil {
		return err
	}

	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		return err
	}

	err = o.sendNotification(ctx, notiBuilder.OrderNotifyRefundRequested, order, user)
	if err != nil {
		return err
	}

	return nil
}

func (o *orderService) ApproveEarlyRefund(ctx context.Context, orderID, actionBy uuid.UUID, fileURL string) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, actionBy)
	if err != nil {
		return err
	}
	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusRefunded, nil)
	if err != nil {
		return err
	}
	order.StaffResource = &fileURL
	if err = helper.WithTransaction(ctx, o.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = o.orderRepository.Update(ctx, order); err != nil {
			zap.L().Error("Failed to update order during approve early refund", zap.Error(err), zap.String("orderID", order.ID.String()))
			return err
		}

		negativePayment := &model.PaymentTransaction{
			ReferenceID:     order.ID,
			ReferenceType:   enum.PaymentTransactionReferenceTypeOrder,
			Amount:          utils.PtrOrNil(-order.TotalAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         utils.PtrOrNil(actionBy),
			ReceivedByID:    utils.PtrOrNil(order.UserID),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			zap.L().Error("Failed to add negative payment transaction during approve early refund", zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		zap.L().Error("Transaction failed during approve early refund", zap.Error(err))
		return err
	}

	go func() {
		err = o.sendNotification(context.Background(), notiBuilder.OrderNotifyRefunded, order, user)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()

	return nil
}

func (o *orderService) GetSelfDeliveringOrdersWithPagination(limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error) {
	ctx := context.Background()
	return o.orderRepository.GetSelfDeliveryOrdersWithPagination(ctx, limit, page, search, status, fullName, phone, provinceID, districtID, wardCode)
}

func (o *orderService) MarkSelfDeliveringOrderAsInTransit(ctx context.Context, orderID, userID uuid.UUID) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, userID)
	if err != nil {
		return err
	}
	//Some validate:
	if order.Status != enum.OrderStatusConfirmed {
		return errors.New("only confirmed orders can be marked as in transit")
	}
	if order.OrderType != enum.ProductTypeLimited.String() || order.IsSelfPickedUp {
		return errors.New("only limited product orders with self-delivering can be marked as in transit")
	}

	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusInTransit, nil)
	if err != nil {
		return err
	}

	return o.orderRepository.Update(ctx, order)
}

func (o *orderService) MarkSelfDeliveringOrderAsDelivered(ctx context.Context, orderID, userID uuid.UUID, imageURL string) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, userID)
	if err != nil {
		return err
	}
	//Some validate:
	if order.Status != enum.OrderStatusInTransit {
		return errors.New("only orders in transit can be marked as delivered")
	}
	if order.OrderType != enum.ProductTypeLimited.String() || order.IsSelfPickedUp {
		return errors.New("only limited product orders with self-delivering can be marked as delivered")
	}
	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusDelivered, nil)
	if err != nil {
		return err
	}
	order.ConfirmationImage = &imageURL
	return o.orderRepository.Update(ctx, order)
}

func (o *orderService) MarkAsReceivedAfterPickedUp(ctx context.Context, orderID, userID uuid.UUID, imageURL string) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, userID)
	if err != nil {
		return err
	}
	//Some validate:
	if order.Status != enum.OrderStatusAwaitingPickUp {
		return errors.New("only orders awaiting pick-up can be marked as received")
	}
	//Convert using FSM
	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusReceived, nil)
	if err != nil {
		return err
	}
	order.ConfirmationImage = &imageURL
	return o.orderRepository.Update(ctx, order)
}

func (o *orderService) MarkAsReadyToPickedUp(ctx context.Context, orderID, userID uuid.UUID) error {
	order, user, err := o.lookupOrderAndUser(ctx, orderID, userID)
	if err != nil {
		return err
	}

	//Some validate:
	if order.Status != enum.OrderStatusConfirmed {
		return errors.New("only confirmed orders can be marked as picked up")
	} else if !order.IsSelfPickedUp {
		return errors.New("this product is not for self pick-up")
	}

	err = MoveOrderStateUsingFSM(order, user, enum.OrderStatusAwaitingPickUp, nil)
	if err != nil {
		return err
	}
	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		return err
	}

	err = o.sendNotification(ctx, notiBuilder.OrderNotifyAwaitingPickUp, order, user)
	if err != nil {
		return err
	}

	return nil
}

func (o *orderService) MarkAsReceived(ctx context.Context, orderID, userID uuid.UUID) error {
	order, actionUser, err := o.lookupOrderAndUser(ctx, orderID, userID)
	if err != nil {
		return err
	}

	//Some validate:
	if order.Status != enum.OrderStatusDelivered {
		return errors.New("only delivered orders can be marked as received")
	}

	err = MoveOrderStateUsingFSM(order, actionUser, enum.OrderStatusReceived, nil)
	if err != nil {
		return err
	}

	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		return err
	}

	go func() {
		if order.User.ID == uuid.Nil {
			zap.L().Error("order owner not found when trying notify. Order ID: " + order.ID.String())
		}
		err = o.sendNotification(context.Background(), notiBuilder.OrderNotifyReceived, order, &order.User)
		if err != nil {
			zap.L().Error(err.Error())
		}
	}()
	return nil
}

func (o *orderService) GetOrdersByUserIDWithPagination(
	userID uuid.UUID, limit, page int, search, status, createdFrom, createdTo string,
) ([]responses.OrderResponse, int, error) {
	ctx := context.Background()

	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	// Validate status early to avoid silently ignoring a bad value (which previously caused no filter)
	if status != "" {
		s := enum.OrderStatus(status)
		if !s.IsValid() {
			return nil, 0, fmt.Errorf("invalid order status: %s", status)
		}
	}

	filterScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("orders.user_id = ?", userID)

		if status != "" {
			s := enum.OrderStatus(status)
			if s.IsValid() {
				db = db.Where("orders.status = ?", s)
			}
		}

		if createdFrom != "" {
			if t, err := time.Parse("2006-01-02", createdFrom); err == nil {
				db = db.Where("orders.created_at >= ?", t)
			}
		}
		if createdTo != "" {
			if t, err := time.Parse("2006-01-02", createdTo); err == nil {
				db = db.Where("orders.created_at < ?", t.Add(24*time.Hour))
			}
		}

		if search != "" {
			if id, err := uuid.Parse(search); err == nil {
				// exact match on UUID or GHN code, or cast UUID to text for partial match
				db = db.Where("(orders.id = ? OR orders.ghn_order_code = ? OR orders.id::text ILIKE ?)", id, search, "%"+search+"%")
			} else {
				like := "%" + search + "%"
				// search GHN code or cast UUID to text for ILIKE
				db = db.Where("(orders.ghn_order_code ILIKE ? OR orders.id::text ILIKE ?)", like, like)
			}
		}

		return db
	}

	var total int64
	if err := o.orderRepository.DB().WithContext(ctx).Model(&model.Order{}).
		Scopes(filterScope).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to count orders", zap.Error(err))
		return nil, 0, err
	}

	var orders []model.Order
	if err := o.orderRepository.DB().WithContext(ctx).
		Scopes(filterScope).
		Preload("OrderItems").
		Preload("OrderItems.ProductReview").
		Preload("OrderItems.Brand").
		Preload("OrderItems.Category").Preload("OrderItems.Category.ParentCategory").Preload("OrderItems.Category.ChildCategories").
		Preload("OrderItems.Variant").
		Preload("OrderItems.Variant.Images").
		Preload("OrderItems.Variant.Product").
		Preload("OrderItems.Variant.Product.Limited").
		Order("orders.created_at DESC, orders.id ASC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error; err != nil {
		zap.L().Error("Failed to fetch orders", zap.Error(err))
		return nil, 0, err
	}

	// Collect order IDs to fetch related payment transactions
	orderIDs := make([]uuid.UUID, 0, len(orders))
	for _, oitem := range orders {
		orderIDs = append(orderIDs, oitem.ID)
	}

	paymentsMap := map[uuid.UUID]model.PaymentTransaction{}
	if len(orderIDs) > 0 {
		var transactions []model.PaymentTransaction
		if err := o.paymentTransactionRepository.DB().WithContext(ctx).
			Model(&model.PaymentTransaction{}).
			Where("reference_type = ? AND reference_id IN (?)", enum.PaymentTransactionReferenceTypeOrder, orderIDs).
			Find(&transactions).Error; err != nil {
			zap.L().Error("Failed to fetch payment transactions for orders", zap.Error(err))
			// non-fatal: continue without payment info
		} else {
			// choose latest transaction per order by UpdatedAt
			for _, tx := range transactions {
				existing, ok := paymentsMap[tx.ReferenceID]
				if !ok || tx.UpdatedAt.After(existing.UpdatedAt) {
					paymentsMap[tx.ReferenceID] = tx
				}
			}
		}
	}

	// Map to DTOs including payment transaction
	orderResponses := responses.OrderResponse{}.ToResponseList(orders, paymentsMap)

	return orderResponses, int(total), nil
}

// categorizeVariant will return int base on its product's type:
// -1: limited
// 0: unknown
// 1: standard
func categorizeVariant(variant model.ProductVariant) int {
	if variant.ProductID == uuid.Nil {
		return 0
	}
	switch variant.Product.Type {
	case enum.ProductTypeLimited:
		return -1
	case enum.ProductTypeStandard:
		return 1
	default:
		return 0
	}
}

// validateAndGetRemainingOrder returns remaining allowed pre-order slots for a user+variant.
// It sums quantities in the DB (excluding CANCELLED) to avoid loading all records.
// This will forbid to create new one if there are transaction existed
func (o orderService) validateAndGetRemainingOrder(_ context.Context, userID string, variantID string, maximumOrder int) (int, error) {
	if maximumOrder <= 0 {
		return 0, fmt.Errorf("invalid maximumOrder: %d", maximumOrder)
	}

	db := o.preOrderRepository.DB()

	// --- 1. SUM PreOrders ---
	validPreOrderStatuses := []enum.PreOrderStatus{
		enum.PreOrderStatusPaid,
		enum.PreOrderStatusPreOrdered,
		enum.PreOrderStatusAwaitingPickup,
		enum.PreOrderStatusInTransit,
		enum.PreOrderStatusDelivered,
		enum.PreOrderStatusReceived,
	}

	var preorderSum int
	err := db.
		Table("pre_orders").
		Select("COALESCE(SUM(quantity), 0)").
		Where("user_id = ? AND variant_id = ? AND status IN ?", userID, variantID, validPreOrderStatuses).
		Scan(&preorderSum).Error
	if err != nil {
		return 0, err
	}

	if preorderSum >= maximumOrder {
		return 0, fmt.Errorf("you have reached the maximum number of orders allowed for this product variant")
	}

	// --- 2. SUM OrderItems ---
	validOrderStatuses := []enum.OrderStatus{
		enum.OrderStatusPaid,
		enum.OrderStatusConfirmed,
		enum.OrderStatusShipped,
		enum.OrderStatusInTransit,
		enum.OrderStatusDelivered,
		enum.OrderStatusReceived,
		enum.OrderStatusAwaitingPickUp,
	}

	var orderSum int
	err = db.
		Table("order_items oi").
		Select("COALESCE(SUM(oi.quantity), 0)").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("o.user_id = ? AND oi.variant_id = ? AND o.status IN ?", userID, variantID, validOrderStatuses).
		Scan(&orderSum).Error
	if err != nil {
		return 0, err
	}

	totalBought := preorderSum + orderSum

	remaining := maximumOrder - totalBought
	if remaining <= 0 {
		return 0, fmt.Errorf("you have reached the maximum number of orders allowed for this product variant")
	}

	return remaining, nil
}

func (o *orderService) PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, isOrderLimited bool, unitOfWork irepository.UnitOfWork) (*model.Order, error) {
	now := time.Now()
	var persistedOrder *model.Order
	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//*1. Create Order
		//1.1.Build order items from request
		var persistedOrderItem []model.OrderItem

		// Make sure user already input bankAccount
		createdUser, err := uow.Users().GetByID(ctx, userID, nil)
		if err != nil {
			return err
		}
		if createdUser.BankAccount == nil || createdUser.BankName == nil || createdUser.BankAccountHolder == nil {
			return errors.New("please update your profile with bank account information before placing an order")
		}

		var prevItemCategory *int = nil
		for _, item := range request.Items {

			if isOrderLimited {
				//check if there are pending orders => request user to execute it
				filter := func(db *gorm.DB) *gorm.DB {
					return db.
						Table("order_items").
						Joins("JOIN orders ON orders.id = order_items.order_id").
						Joins("JOIN product_variants pv ON pv.id = order_items.variant_id").
						Where("orders.status = ? AND order_items.variant_id = ? AND orders.user_id = ?", enum.OrderStatusPending, item.VariantID, userID)
				}

				// don't need includes when using joins
				var orderItems []model.OrderItem
				var total int64
				orderItems, total, err = o.orderItemRepository.GetAll(ctx, filter, nil, 0, 0)
				if err != nil {
					return err
				}
				if total > 0 {
					var existedOrderID string
					for _, i := range orderItems {
						existedOrderID += "," + i.ID.String()
					}
					return fmt.Errorf("you have pending orders (order IDs: %s) for this product variant. Please complete or cancel them before placing a new pre-order", existedOrderID[1:])
				}
			}

			//check variantID:
			includes := []string{"AttributeValues", "AttributeValues.Attribute", "Images", "Product", "Product.Limited"}
			var variant *model.ProductVariant
			variant, err = uow.ProductVariant().GetByID(ctx, item.VariantID, includes)
			if err != nil {
				zap.L().Error("ProductVariant().GetByID", zap.Error(err))
				return errors.New("product Variant not found")
			} else if variant == nil {
				return errors.New("product Variant not found")
			}

			//backend check to make sure u using the correctAPI:
			if isOrderLimited && variant.Product.Type != enum.ProductTypeLimited {
				return fmt.Errorf("product %s (id = %s) is not LIMITED type, please use STANDARD order API", variant.Product.Name, variant.ID.String())
			} else if !isOrderLimited && variant.Product.Type != enum.ProductTypeStandard {
				return fmt.Errorf("product %s (id = %s) is not LIMITED type, please use LIMITED order API", variant.Product.Name, variant.ID.String())
			}

			//validate if product is active
			if !variant.Product.IsActive {
				return fmt.Errorf("product is not available, please contact admin to activate it")
			}

			if variant.Product != nil && variant.Product.Type == enum.ProductTypeLimited {
				limitedInfo := variant.Product.Limited
				if limitedInfo == nil {
					return fmt.Errorf("limited product info not found for product: %s (id = %s)", variant.Product.Name, variant.Product.ID.String())
				}
				if now.Before(limitedInfo.AvailabilityStartDate) {
					return fmt.Errorf("product %s (id = %s) is not yet available for order", variant.Product.Name, variant.Product.ID.String())
				} else if now.After(limitedInfo.AvailabilityEndDate) {
					return fmt.Errorf("product %s (id = %s) is no longer available for order", variant.Product.Name, variant.Product.ID.String())
				} else if *variant.CurrentStock <= 0 {
					return fmt.Errorf("product %s (id = %s) is out of stock", variant.Product.Name, variant.Product.ID.String())
				}
			}

			//Categorize variant to  check if order item not being mixed between STANDARD and LIMITED
			currentItemCategory := categorizeVariant(*variant)
			if isOrderLimited && currentItemCategory != -1 {
				return fmt.Errorf("STANDARD product found: %s (id = %s)", variant.Product.Name, variant.ID.String())
			}
			if currentItemCategory == 0 {
				return errors.New("unknown product type")
			}
			if prevItemCategory != nil && *prevItemCategory != currentItemCategory {
				return errors.New("cannot place order with mixed product types in a single order")
			}
			//update prevItemCategory
			prevItemCategory = &currentItemCategory

			// Validate maximum order quantity for LIMITED products
			if isOrderLimited {
				if variant.MaxStock == nil || variant.CurrentStock == nil {
					return fmt.Errorf("stock data not found for limited variant: %s", variant.ID.String())
				}
				// check order remain quantity
				generalRemain := *variant.CurrentStock
				if generalRemain <= 0 {
					return fmt.Errorf("this product is out of stock for order")
				}
				if item.Quantity > generalRemain {
					return fmt.Errorf("only %d order slots remaining for this product variant", generalRemain)
				}
				// 2. user achievable limit
				achievable := variant.Product.Limited.AchievableQuantity
				var userRemain int
				userRemain, err = o.validateAndGetRemainingOrder(ctx, userID.String(), item.VariantID.String(), achievable)
				if err != nil {
					return err
				}

				actualRemain := min(userRemain, generalRemain)
				if item.Quantity > actualRemain {
					return fmt.Errorf("you can only order up to %d more units of product: %s (id = %s)", actualRemain, variant.Product.Name, variant.ID.String())
				}
			}

			//Build persisted order item
			persistedItem := item.ToModel(*variant, now)

			if isOrderLimited {
				oldStock := 0
				if variant.CurrentStock != nil {
					oldStock = *variant.CurrentStock
				} else {
					return fmt.Errorf("current stock is nil for product: %s (id = %s)", variant.Product.Name, variant.ID.String())
				}
				if oldStock-item.Quantity < 0 {
					return fmt.Errorf("insufficient stock for product: %s (id = %s). Have %d, need %d", variant.Product.Name, variant.ID.String(), oldStock, item.Quantity)
				}
				err = o.AtomicDecreaseLimitedStock(ctx, variant.ID, item.Quantity)
				if err != nil {
					return err
				}
			}

			persistedOrderItem = append(persistedOrderItem, *persistedItem)
		}

		// Validate at least one item present and decide order type + shipping fee
		if len(persistedOrderItem) == 0 {
			return errors.New("order has no items")
		}

		// Determine order type and shipping fee to store
		// For LIMITED orders: store the actual shipping fee for analytics (KOL revenue deduction)
		// but customer doesn't pay it (handled in PayOrder by passing 0 for shipping)
		orderType := enum.ProductTypeStandard
		applyShippingFee := shippingPrice
		if isOrderLimited || (prevItemCategory != nil && *prevItemCategory == -1) {
			orderType = enum.ProductTypeLimited
			// Keep the actual shipping fee for LIMITED orders (for KOL revenue analytics)
			// Customer payment is handled separately in PayOrder with shippingFee = 0
			applyShippingFee = shippingPrice
		}

		//1.2.Build shipping address add to order
		shippingAddress, err := o.shippingAddressRepository.GetByID(ctx, request.AddressID, nil)
		if err != nil {
			zap.L().Error("ShippingAddress().GetByID", zap.Error(err))
			return err
		}

		// Build order model now that we have items and fee
		persistedOrder = request.ToModel(*createdUser, persistedOrderItem, *shippingAddress, int(applyShippingFee), now)
		// Set order type safely after initialization
		persistedOrder.OrderType = orderType.String()

		//1.3.Persist
		return uow.Order().Add(ctx, persistedOrder)
	})

	if err != nil {
		return nil, err
	}

	return persistedOrder, nil
}

func (o *orderService) AtomicDecreaseLimitedStock(
	ctx context.Context,
	variantID uuid.UUID,
	qty int,
) error {

	tx := o.db.WithContext(ctx).Model(&model.ProductVariant{}).
		Where("id = ? AND current_stock >= ?", variantID, qty).
		UpdateColumn("current_stock", gorm.Expr("current_stock - ?", qty))

	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("insufficient stock")
	}

	return nil
}

// PayOrder handles the payment process in a atomic transaction
func (o *orderService) PayOrder(ctx context.Context, orderID uuid.UUID, shippingFee int, returnURL, cancelURL string, unitOfWork irepository.UnitOfWork) (*responses.PayOSLinkResponse, error) {
	var paymentTransaction *responses.PayOSLinkResponse

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//*1 Fetch Order with Preload
		includes := []string{"User", "OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, includes)
		if err != nil {
			return err
		}

		//Map order items to PayOS items
		paymentItemRequest, total := toPaymentItemRequestsWithTotalPrice(order.OrderItems, shippingFee)

		//*2 Build Payment Request:
		paymentRq := requests.PaymentRequest{
			ReferenceID:   order.ID,
			ReferenceType: enum.PaymentTransactionReferenceTypeOrder,
			PayerID:       &order.UserID,
			Amount:        int64(total),
			Description:   fmt.Sprintf("Payment for Order %s", order.ID),
			Items:         paymentItemRequest,
			BuyerName:     order.FullName,
			BuyerEmail:    order.Email,
			BuyerPhone:    order.PhoneNumber,
			ReturnURL:     &returnURL,
			CancelURL:     &cancelURL,
		}

		//*3. Create Payment Transaction
		paymentTransaction, err = o.paymentTransactionService.GeneratePaymentLink(ctx, uow, &paymentRq)
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

// PayOrder Helper
func toPaymentItemRequestsWithTotalPrice(items []model.OrderItem, shippingFee int) ([]requests.PaymentItemRequest, int) {
	paymentItems := make([]requests.PaymentItemRequest, 0, len(items))
	total := shippingFee
	for _, item := range items {
		paymentItems = append(paymentItems, requests.PaymentItemRequest{
			Name:     item.Variant.Product.Name,
			Quantity: item.Quantity,
			Price:    int64(item.UnitPrice),
		})
		total += int(item.UnitPrice) * item.Quantity
	}

	//Add shipping fee item as additional info
	shippingFeeItem := requests.PaymentItemRequest{
		Name:     "Shipping Fee from \"Giao Hàng Nhanh\"",
		Quantity: 1,
		Price:    int64(shippingFee),
	}

	paymentItems = append(paymentItems, shippingFeeItem)
	return paymentItems, total
}

func (o *orderService) GetStaffAvailableOrdersWithPagination(limit, page int, search, fullName, phone, provinceID, districtID, wardCode, orderType, createdFrom, createdTo, brandID string, statuses []string) ([]responses.OrderResponse, int, error) {
	ctx := context.Background()
	orders, total, err := o.orderRepository.GetStaffAvailableOrdersWithPagination(ctx, limit, page, search, fullName, phone, provinceID, districtID, wardCode, orderType, createdFrom, createdTo, brandID, statuses)
	if err != nil {
		return nil, 0, err
	}

	// If no orders or no payment repository wired - return early
	if len(orders) == 0 {
		return []responses.OrderResponse{}, total, nil
	}
	if o.paymentTransactionRepository == nil || o.paymentTransactionRepository.DB() == nil {
		zap.L().Warn("paymentTransactionRepository not available, returning orders without transaction enrichment")
		// Convert to response without payment enrichment
		return responses.OrderResponse{}.ToResponseList(orders, map[uuid.UUID]model.PaymentTransaction{}), total, nil
	}

	// Collect order IDs
	orderIDs := make([]uuid.UUID, 0, len(orders))
	for _, oitem := range orders {
		orderIDs = append(orderIDs, oitem.ID)
	}

	// Load payment transactions referencing these orders
	var transactions []model.PaymentTransaction
	if err := o.paymentTransactionRepository.DB().WithContext(ctx).
		Model(&model.PaymentTransaction{}).
		Where("reference_type = ? AND reference_id IN (?)", enum.PaymentTransactionReferenceTypeOrder, orderIDs).
		Find(&transactions).Error; err != nil {
		zap.L().Error("Failed to fetch payment transactions for staff orders", zap.Error(err))
		return responses.OrderResponse{}.ToResponseList(orders, map[uuid.UUID]model.PaymentTransaction{}), total, nil
	}

	// Choose latest transaction per order by UpdatedAt
	paymentsMap := make(map[uuid.UUID]model.PaymentTransaction)
	for _, tx := range transactions {
		existing, ok := paymentsMap[tx.ReferenceID]
		if !ok || tx.UpdatedAt.After(existing.UpdatedAt) {
			paymentsMap[tx.ReferenceID] = tx
		}
	}

	// Attach latest payment info back to orders (transient fields) - keep for compatibility but not required for DTO
	for i := range orders {
		if pt, ok := paymentsMap[orders[i].ID]; ok {
			orders[i].PaymentID = &pt.ID
			if pt.PayOSMetadata != nil {
				orders[i].PaymentBin = &pt.PayOSMetadata.Bin
			}
		}
	}

	// Calculate company and KOL revenue for each order
	companyRevenues := make(map[uuid.UUID]float64)
	kolRevenues := make(map[uuid.UUID]float64)

	for _, order := range orders {
		companyRev, kolRev := o.calculateOrderRevenues(ctx, &order)
		companyRevenues[order.ID] = companyRev
		kolRevenues[order.ID] = kolRev
	}

	// Convert model orders to response DTOs using payment map and revenue maps
	orderResponses := responses.OrderResponse{}.ToResponseListWithRevenue(orders, paymentsMap, companyRevenues, kolRevenues)

	return orderResponses, total, nil
}

// ConfirmOrder transitions an order to CONFIRMED. For LIMITED products, decrements variant stock accordingly.
func (o *orderService) ConfirmOrder(ctx context.Context, orderID uuid.UUID, updatedBy uuid.UUID, orderStatus enum.OrderStatus, unitOfWork irepository.UnitOfWork) error {
	_ = updatedBy // currently unused but kept for API compatibility
	return helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// 1) Load order with items and variants + product
		includes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, includes)
		if err != nil {
			zap.L().Error("Failed to load order for confirm", zap.Error(err))
			return fmt.Errorf("failed to load order: %w", err)
		}

		if order == nil {
			return errors.New("order not found")
		}

		// 2) Validate state transition using state machine
		ctxState := &ordersm.OrderContext{State: ordersm.NewOrderState(order.Status)}
		nextState := ordersm.NewOrderState(enum.OrderStatusConfirmed)
		if nextState == nil {
			return errors.New("invalid target state")
		}
		if err := ctxState.State.Next(ctxState, nextState); err != nil {
			zap.L().Error("Order state transition validation failed", zap.Error(err))
			return fmt.Errorf("state transition not allowed: %w", err)
		}

		// 3) For each order item, if product type is LIMITED, check and decrement variant stock
		for _, it := range order.OrderItems {
			if it.VariantID == uuid.Nil || it.Variant.ProductID == uuid.Nil {
				return fmt.Errorf("invalid order item: variant not found")
			}

			if it.Variant.Product.Type == enum.ProductTypeLimited {
				// ensure CurrentStock present
				if it.Variant.CurrentStock == nil {
					return fmt.Errorf("variant %s stock is nil", it.VariantID.String())
				}

				if *it.Variant.CurrentStock < it.Quantity {
					return fmt.Errorf("insufficient stock for variant %s: have %d, need %d", it.VariantID.String(), *it.Variant.CurrentStock, it.Quantity)
				}

				// decrement stock
				*it.Variant.CurrentStock = *it.Variant.CurrentStock - it.Quantity
				if err := uow.ProductVariant().Update(ctx, &it.Variant); err != nil {
					zap.L().Error("Failed to update variant stock", zap.Error(err))
					return fmt.Errorf("failed to update variant stock: %w", err)
				}
			}
		}

		// 4) Persist order status change to requested state
		order.Status = orderStatus
		if err := uow.Order().Update(ctx, order); err != nil {
			zap.L().Error("Failed to update order status to confirmed", zap.Error(err))
			return fmt.Errorf("failed to update order: %w", err)
		}

		// optionally: publish events or perform side-effects here
		return nil
	})
}

func (o *orderService) CancelOrder(ctx context.Context, orderID uuid.UUID, updatedBy uuid.UUID, reason string, unitOfWork irepository.UnitOfWork) error {
	_ = updatedBy
	_ = reason
	return helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Load order with items and variants + product
		includes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, includes)
		if err != nil {
			zap.L().Error("Failed to load order for cancel", zap.Error(err))
			return fmt.Errorf("failed to load order: %w", err)
		}

		if order == nil {
			return errors.New("order not found")
		}

		// Validate state transition using state machine
		ctxState := &ordersm.OrderContext{State: ordersm.NewOrderState(order.Status)}
		nextState := ordersm.NewOrderState(enum.OrderStatusCancelled)
		if nextState == nil {
			return errors.New("invalid target state")
		}
		if err := ctxState.State.Next(ctxState, nextState); err != nil {
			zap.L().Error("Order state transition validation failed for cancel", zap.Error(err))
			return fmt.Errorf("state transition not allowed: %w", err)
		}

		// If cancelling an already CONFIRMED order, and products are LIMITED, restock variants
		wasConfirmed := order.Status == enum.OrderStatusConfirmed
		if wasConfirmed {
			for _, it := range order.OrderItems {
				if it.VariantID == uuid.Nil || it.Variant.ProductID == uuid.Nil {
					return fmt.Errorf("invalid order item: variant not found")
				}

				if it.Variant.Product.Type == enum.ProductTypeLimited {
					// ensure CurrentStock present
					if it.Variant.CurrentStock == nil {
						// initialize to zero and then add
						zero := 0
						it.Variant.CurrentStock = &zero
					}

					*it.Variant.CurrentStock = *it.Variant.CurrentStock + it.Quantity
					if err := uow.ProductVariant().Update(ctx, &it.Variant); err != nil {
						zap.L().Error("Failed to update variant stock during cancel", zap.Error(err))
						return fmt.Errorf("failed to update variant stock: %w", err)
					}
				}
			}
		}

		// Persist order status change to CANCELLED
		order.Status = enum.OrderStatusCancelled
		if err := uow.Order().Update(ctx, order); err != nil {
			zap.L().Error("Failed to update order status to cancelled", zap.Error(err))
			return fmt.Errorf("failed to update order: %w", err)
		}

		// optionally log cancellation reason/actor here
		return nil
	})
}

func NewOrderService(
	cfg *config.AppConfig,
	dbRegistry *gormrepository.DatabaseRegistry,
	registry *infrastructure.InfrastructureRegistry,
	paymentTransactionSvc iservice.PaymentTransactionService,
	notificationService iservice.NotificationService,
) iservice.OrderService {
	return &orderService{
		config:                       cfg,
		db:                           dbRegistry.GormDatabase,
		orderRepository:              dbRegistry.OrderRepository,
		orderItemRepository:          dbRegistry.OrderItemRepository,
		shippingAddressRepository:    dbRegistry.ShippingAddressRepository,
		userRepository:               dbRegistry.UserRepository,
		preOrderRepository:           dbRegistry.PreOrderRepository,
		payOSProxy:                   registry.ProxiesRegistry.PayOSProxy,
		ghnProxy:                     registry.ProxiesRegistry.GHNProxy,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		paymentTransactionService:    paymentTransactionSvc,
		notificationService:          notificationService,
		unitOfWork:                   registry.UnitOfWork,
	}
}

// lookupOrderAndUser is a helper to fetch order and user by their IDs (actors that engage in order actions)
func (o *orderService) lookupOrderAndUser(ctx context.Context, orderID, actionBy uuid.UUID) (*model.Order, *model.User, error) {
	order, err := o.orderRepository.GetByID(ctx, orderID, []string{"User"})
	if err != nil {
		return nil, nil, err
	}

	// If actionBy is zero value, treat as System user
	var user *model.User
	if actionBy == uuid.Nil {
		user = &model.User{
			ID:       uuid.UUID{},
			FullName: o.config.AdminConfig.SystemName,
			Email:    o.config.AdminConfig.SystemEmail,
		}
	} else {
		user, err = o.userRepository.GetByID(ctx, actionBy, nil)
		if err != nil {
			return nil, nil, err
		}
	}
	return order, user, nil
}

func MoveOrderStateUsingFSM(order *model.Order, user *model.User, newStatus enum.OrderStatus, reason *string) error {
	ctxState := &ordersm.OrderContext{
		State:    ordersm.NewOrderState(order.Status),
		Order:    order,
		ActionBy: user,
	}
	nextState := ordersm.NewOrderState(newStatus)
	if err := ctxState.State.Next(ctxState, nextState); err != nil {
		zap.L().Error("Order state transition validation failed", zap.Error(err))
		return fmt.Errorf("state transition not allowed: %w", err)
	}
	order.AddActionNote(*ctxState.GenerateActionNote(user, reason))
	return nil
}

func (o orderService) sendNotification(ctx context.Context, notiStatus notiBuilder.OrderNotificationType, order *model.Order, actionBy *model.User) error {
	//This contains many payloads to different receivers
	payloads, err := notiBuilder.BuildOrderNotifications(ctx, *o.config, o.db, notiStatus, order, actionBy)
	if err != nil {
		zap.L().Debug("no notification builder for order status", zap.Error(err))
		return nil
	}

	for _, p := range payloads {
		_, err = o.notificationService.CreateAndPublishNotification(ctx, &p)
		if err != nil {
			zap.L().Error("Failed to send notification", zap.Error(err))
			return err
		}
	}
	return nil
}
