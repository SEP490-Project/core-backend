package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	responses "core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/ordersm"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type orderService struct {
	config                       *config.AppConfig
	orderRepository              irepository.OrderRepository
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnProxy                     iproxies.GHNProxy
	paymentTransactionService    iservice.PaymentTransactionService
}

func (o *orderService) GetSelfDeliveringOrdersWithPagination(limit, page int, search, status, fullName, phone, provinceID, districtID, wardCode string) ([]model.Order, int, error) {
	ctx := context.Background()
	return o.orderRepository.GetSelfDeliveryOrdersWithPagination(ctx, limit, page, search, status, fullName, phone, provinceID, districtID, wardCode)
}

func (o *orderService) MarkSelfDeliveringOrderAsInTransit(ctx context.Context, orderID uuid.UUID) error {
	order, err := o.orderRepository.GetByID(ctx, orderID, nil)
	if err != nil {
		zap.L().Error("Failed to fetch order for marking as in transit", zap.Error(err))
		return err
	}
	//Some validate:
	if order.Status != enum.OrderStatusConfirmed {
		return errors.New("only confirmed orders can be marked as in transit")
	}
	if order.OrderType != enum.ProductTypeLimited.String() || order.IsSelfPickedUp {
		return errors.New("only limited product orders with self-delivering can be marked as in transit")
	}
	order.Status = enum.OrderStatusInTransit
	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		zap.L().Error("Failed to update order status to in transit", zap.Error(err))
		return err
	}
	return nil
}

func (o *orderService) MarkSelfDeliveringOrderAsDelivered(ctx context.Context, orderID uuid.UUID, imageURL string) error {
	order, err := o.orderRepository.GetByID(ctx, orderID, nil)
	if err != nil {
		zap.L().Error("Failed to fetch order for marking as delivered", zap.Error(err))
		return err
	}
	//Some validate:
	if order.Status != enum.OrderStatusInTransit {
		return errors.New("only orders in transit can be marked as delivered")
	}
	if order.OrderType != enum.ProductTypeLimited.String() || order.IsSelfPickedUp {
		return errors.New("only limited product orders with self-delivering can be marked as delivered")
	}
	order.Status = enum.OrderStatusDelivered
	order.ConfirmationImage = &imageURL
	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		zap.L().Error("Failed to update order status to delivered", zap.Error(err))
		return err
	}
	return nil
}

func (o *orderService) MarkAsReceivedAfterPickedUp(ctx context.Context, orderID uuid.UUID, imageURL string) error {
	order, err := o.orderRepository.GetByID(ctx, orderID, nil)
	if err != nil {
		zap.L().Error("Failed to fetch order for marking as received", zap.Error(err))
		return err
	}
	//Some validate:
	if order.Status != enum.OrderStatusAwaitingPickUp {
		return errors.New("only orders awaiting pick-up can be marked as received")
	}
	order.Status = enum.OrderStatusReceived
	order.ConfirmationImage = &imageURL
	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		zap.L().Error("Failed to update order status to completed", zap.Error(err))
		return err
	}
	return nil
}

func (o *orderService) MarkAsReadyToPickedUp(ctx context.Context, orderID uuid.UUID) error {
	order, err := o.orderRepository.GetByID(ctx, orderID, nil)
	if err != nil {
		zap.L().Error("Failed to fetch order for marking as pickedup", zap.Error(err))
		return err
	}

	//Some validate:
	if order.Status != enum.OrderStatusConfirmed {
		return errors.New("only confirmed orders can be marked as picked up")
	} else if !order.IsSelfPickedUp {
		return errors.New("this product is not for self pick-up")
	}

	order.Status = enum.OrderStatusAwaitingPickUp
	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		zap.L().Error("Failed to update order status to completed", zap.Error(err))
		return err
	}
	return nil
}

func (o *orderService) MarkAsReceived(ctx context.Context, orderID uuid.UUID) error {
	order, err := o.orderRepository.GetByID(ctx, orderID, nil)
	if err != nil {
		zap.L().Error("Failed to fetch order for marking as received", zap.Error(err))
		return err
	}

	//Some validate:
	if order.Status != enum.OrderStatusDelivered {
		return errors.New("only delivered orders can be marked as received")
	}

	order.Status = enum.OrderStatusReceived
	err = o.orderRepository.Update(ctx, order)
	if err != nil {
		zap.L().Error("Failed to update order status to completed", zap.Error(err))
		return err
	}
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

func (o *orderService) PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, isOrderLimited bool, unitOfWork irepository.UnitOfWork) (*model.Order, error) {
	now := time.Now()
	var persistedOrder *model.Order
	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//*1. Create Order
		//1.1.Build order items from request
		var persistedOrderItem []model.OrderItem

		var prevItemCategory *int = nil
		for _, item := range request.Items {
			//check variantID:
			includes := []string{"AttributeValues", "AttributeValues.Attribute", "Images", "Product", "Product.Limited"}
			variant, err := uow.ProductVariant().GetByID(ctx, item.VariantID, includes)
			if err != nil {
				zap.L().Error("ProductVariant().GetByID", zap.Error(err))
				return errors.New("product Variant not found")
			} else if variant == nil {
				return errors.New("product Variant not found")
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
				}
			}

			//Categorize variant
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

			//Build persisted order item
			//Also subtract items stock for LIMITED products
			persistedItem := item.ToModel(*variant, now)

			if isOrderLimited {
				oldStock := 0
				if variant.CurrentStock != nil {
					oldStock = *variant.CurrentStock
				}
				if oldStock-item.Quantity < 0 {
					return fmt.Errorf("insufficient stock for product: %s (id = %s). Have %d, need %d", variant.Product.Name, variant.ID.String(), oldStock, item.Quantity)
				}
				newStock := oldStock - item.Quantity
				variant.CurrentStock = &newStock

				zap.L().Info("Updating stock for LIMITED product",
					zap.String("product_variant_id", variant.ID.String()),
					zap.Int("old_stock", oldStock),
					zap.Int("new_stock", newStock))

				err = uow.ProductVariant().Update(ctx, variant)
				if err != nil {
					zap.L().Error("ProductVariant().Update", zap.Error(err))
					return err
				}
			}

			persistedOrderItem = append(persistedOrderItem, *persistedItem)
		}

		// Validate at least one item present and decide order type + shipping fee
		if len(persistedOrderItem) == 0 {
			return errors.New("order has no items")
		}

		// Determine order type and shipping fee to apply
		orderType := enum.ProductTypeStandard
		applyShippingFee := shippingPrice
		if isOrderLimited || (prevItemCategory != nil && *prevItemCategory == -1) {
			orderType = enum.ProductTypeLimited
			applyShippingFee = 0
		}

		//1.2.Build shipping address add to order
		shippingAddress, err := o.shippingAddressRepository.GetByID(ctx, request.AddressID, nil)
		if err != nil {
			zap.L().Error("ShippingAddress().GetByID", zap.Error(err))
			return err
		}

		// Build order model now that we have items and fee
		persistedOrder = request.ToModel(userID, persistedOrderItem, *shippingAddress, int(applyShippingFee), now)
		// Set order type safely after initialization
		persistedOrder.OrderType = orderType.String()

		//1.3.Persist
		err = uow.Order().Add(ctx, persistedOrder)
		if err != nil {
			zap.L().Error("Order().Add", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return persistedOrder, nil
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

func (o *orderService) GetStaffAvailableOrdersWithPagination(limit, page int, search string, status string, fullName, phone, provinceID, districtID, wardCode, orderType string) ([]responses.OrderResponse, int, error) {
	ctx := context.Background()
	orders, total, err := o.orderRepository.GetStaffAvailableOrdersWithPagination(ctx, limit, page, search, status, fullName, phone, provinceID, districtID, wardCode, orderType)
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
		// non-fatal: return orders without enrichment (convert to DTO)
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

	// Convert model orders to response DTOs using payment map
	orderResponses := responses.OrderResponse{}.ToResponseList(orders, paymentsMap)

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

func NewOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry, paymentTransactionSvc iservice.PaymentTransactionService) iservice.OrderService {
	return &orderService{
		config:                       cfg,
		orderRepository:              dbRegistry.OrderRepository,
		orderItemRepository:          dbRegistry.OrderItemRepository,
		shippingAddressRepository:    dbRegistry.ShippingAddressRepository,
		payOSProxy:                   registry.ProxiesRegistry.PayOSProxy,
		ghnProxy:                     registry.ProxiesRegistry.GHNProxy,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		paymentTransactionService:    paymentTransactionSvc,
	}
}
