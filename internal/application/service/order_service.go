package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	responses "core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
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
	orderRepository              irepository.GenericRepository[model.Order]
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOSProxy                   iproxies.PayOSProxy
	shippingAddressRepository    irepository.GenericRepository[model.ShippingAddress]
	ghnService                   iservice_third_party.GHNService
	paymentTransactionService    iservice.PaymentTransactionService
}

func (o *orderService) GetOrdersByUserIDWithPagination(userID uuid.UUID, limit, page int, search string) ([]model.Order, int, error) {
	ctx := context.Background()

	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	filter := func(db *gorm.DB) *gorm.DB {
		db = db.Where("orders.user_id = ?", userID)
		if search != "" {
			db = db.Where("orders.order_no ILIKE ?", "%"+search+"%")
		}
		return db.Order("orders.created_at DESC").Order("orders.id")
	}

	includes := []string{"OrderItems"}

	var orderIDs []uuid.UUID
	err := o.orderRepository.DB().
		WithContext(ctx).
		Model(&model.Order{}).
		Scopes(filter).
		Select("orders.id").
		Limit(pageSize).
		Offset(offset).
		Pluck("orders.id", &orderIDs).Error
	if err != nil {
		zap.L().Error("Failed to fetch order IDs", zap.Error(err))
		return nil, 0, err
	}

	if len(orderIDs) == 0 {
		return []model.Order{}, 0, nil
	}

	countScope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("orders.user_id = ?", userID)
		if search != "" {
			db = db.Where("orders.order_no ILIKE ?", "%"+search+"%")
		}
		return db
	}
	var total int64
	if err := o.orderRepository.DB().
		WithContext(ctx).
		Model(&model.Order{}).
		Scopes(countScope).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to count orders", zap.Error(err))
		return nil, 0, err
	}

	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("orders.id IN ?", orderIDs).
			Order("orders.created_at DESC")
	}

	orders, _, err := o.orderRepository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch orders with includes", zap.Error(err))
		return nil, 0, err
	}

	return orders, int(total), nil
}

func (o *orderService) PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, shippingPrice int, unitOfWork irepository.UnitOfWork) (*model.Order, error) {
	now := time.Now()
	var persistedOrder *model.Order
	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//*1. Create Order
		//1.1.Build order items from request
		var persistedOrderItem []model.OrderItem
		for _, item := range request.Items {
			//check variantID:
			includes := []string{"AttributeValues", "AttributeValues.Attribute", "Images"}
			variant, err := uow.ProductVariant().GetByID(ctx, item.VariantID, includes)
			if err != nil {
				zap.L().Error("ProductVariant().GetByID", zap.Error(err))
				return errors.New("Product Variant not found")
			} else if variant == nil {
				return errors.New("Product Variant not found")
			}
			persistedItem := item.ToModel(*variant, now)
			persistedOrderItem = append(persistedOrderItem, *persistedItem)
		}

		//1.2.Build shipping address
		shippingAddress, err := o.shippingAddressRepository.GetByID(ctx, request.AddressID, nil)
		if err != nil {
			zap.L().Error("ShippingAddress().GetByID", zap.Error(err))
			return err
		}

		persistedOrder = request.ToModel(userID, persistedOrderItem, *shippingAddress, int(shippingPrice), now)

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
	var total int = shippingFee
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

func (o *orderService) GetStaffAvailableOrdersWithPagination(limit, page int, search string, status string) ([]model.Order, int, error) {
	ctx := context.Background()

	pageNum := page
	pageSize := limit
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	// Validate status if provided; enforce non-PENDING results
	var validStatus *enum.OrderStatus
	if status != "" {
		s := enum.OrderStatus(status)
		if s.IsValid() {
			validStatus = &s
		}
	}

	filter := func(db *gorm.DB) *gorm.DB {
		// Always exclude PENDING
		db = db.Where("orders.status <> ?", enum.OrderStatusPending)
		// If a valid non-PENDING status provided, narrow down further
		if validStatus != nil && *validStatus != enum.OrderStatusPending {
			db = db.Where("orders.status = ?", *validStatus)
		}
		if search != "" {
			db = db.Where("orders.order_no ILIKE ?", "%"+search+"%")
		}
		return db.Order("orders.created_at DESC").Order("orders.id")
	}

	includes := []string{"OrderItems"}

	var orderIDs []uuid.UUID
	err := o.orderRepository.DB().
		WithContext(ctx).
		Model(&model.Order{}).
		Scopes(filter).
		Select("orders.id").
		Limit(pageSize).
		Offset(offset).
		Pluck("orders.id", &orderIDs).Error
	if err != nil {
		zap.L().Error("Failed to fetch order IDs", zap.Error(err))
		return nil, 0, err
	}

	if len(orderIDs) == 0 {
		return []model.Order{}, 0, nil
	}

	countScope := func(db *gorm.DB) *gorm.DB {
		// Always exclude PENDING
		db = db.Where("orders.status <> ?", enum.OrderStatusPending)
		// If a valid non-PENDING status provided, narrow down further
		if validStatus != nil && *validStatus != enum.OrderStatusPending {
			db = db.Where("orders.status = ?", *validStatus)
		}
		if search != "" {
			db = db.Where("orders.order_no ILIKE ?", "%"+search+"%")
		}
		return db
	}
	var total int64
	if err := o.orderRepository.DB().
		WithContext(ctx).
		Model(&model.Order{}).
		Scopes(countScope).
		Count(&total).Error; err != nil {
		zap.L().Error("Failed to count orders", zap.Error(err))
		return nil, 0, err
	}

	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("orders.id IN ?", orderIDs).
			Order("orders.created_at DESC")
	}

	orders, _, err := o.orderRepository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch orders with includes", zap.Error(err))
		return nil, 0, err
	}

	return orders, int(total), nil
}

func NewOrderService(cfg *config.AppConfig, dbRegistry *gormrepository.DatabaseRegistry, registry *infrastructure.InfrastructureRegistry, paymentTransactionSvc iservice.PaymentTransactionService) iservice.OrderService {
	return &orderService{
		config:                    cfg,
		orderRepository:           dbRegistry.OrderRepository,
		orderItemRepository:       dbRegistry.OrderItemRepository,
		shippingAddressRepository: dbRegistry.ShippingAddressRepository,
		payOSProxy:                registry.ProxiesRegistry.PayOSProxy,
		ghnService:                registry.GHNService,
		paymentTransactionService: paymentTransactionSvc,
	}
}

//=========== Helper Methods ===========
