package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type orderService struct {
	orderRepository              irepository.GenericRepository[model.Order]
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
	payOsService                 iservice_third_party.PayOSService
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

func (o *orderService) PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, unitOfWork irepository.UnitOfWork) (*model.Order, error) {
	now := time.Now()
	var persistedOrder *model.Order

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Create Item
		var persistedOrderItem []model.OrderItem

		for _, item := range request.Items {
			//check variantID:
			includes := []string{"AttributeValues", "AttributeValues.Attribute", "Images"}
			variant, err := uow.ProductVariant().GetByID(ctx, item.VariantID, includes)
			if err != nil {
				zap.L().Error("ProductVariant().GetByID", zap.Error(err))
				_ = uow.Rollback()
				return errors.New("Product Variant not found")
			} else if variant == nil {
				_ = uow.Rollback()
				return errors.New("Product Variant not found")
			}
			persistedItem := item.ToModel(*variant, now)
			persistedOrderItem = append(persistedOrderItem, *persistedItem)
		}

		//Create Order
		persistedOrder = request.ToModel(userID, persistedOrderItem, now)
		err := uow.Order().Add(ctx, persistedOrder)
		if err != nil {
			zap.L().Error("Order().Add", zap.Error(err))
			_ = uow.Rollback()
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return persistedOrder, nil
}

func (o *orderService) PayOrder(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) (*model.PaymentTransaction, error) {
	var paymentTransaction *model.PaymentTransaction
	now := time.Now()

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Check Order
		includes := []string{"User", "Address", "OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, includes)
		if err != nil {
			zap.L().Error("Order().GetByID", zap.Error(err))
			_ = uow.Rollback()
			return err
		} else if order == nil {
			_ = uow.Rollback()
			return errors.New("Order not found")
		}

		paymenReq := requests.PaymentRequest{
			Amount:           int64(order.TotalAmount),
			Description:      "Pay:" + order.ID.String(),
			BuyerName:        order.User.FullName,
			BuyerCompanyName: "Temporarily empty",
			BuyerTaxCode:     "",
			BuyerAddress:     "Temporarily empty",
			BuyerEmail:       order.User.Email,
			BuyerPhone:       order.User.Phone,
			Items:            requests.MapPaymentItemsFromOrderItems(order.OrderItems),
			Invoice:          nil,
		}
		payOSResponse, err := o.payOsService.GeneratePayOSLink(paymenReq)
		if err != nil {
			_ = uow.Rollback()
			zap.L().Error("payOsService.GeneratePayOSLink", zap.Error(err))
			return err
		}

		//Create Payment Transaction
		paymentTransaction = &model.PaymentTransaction{
			ReferenceID:     order.ID,
			ReferenceType:   "ORDER",
			Amount:          &order.TotalAmount,
			Method:          "ONLINE",
			Status:          "PENDING",
			TransactionDate: now,
			GatewayRef:      payOSResponse.Data.CheckoutUrl,
			GatewayID:       payOSResponse.Data.Bin,
		}

		err = uow.PaymentTransaction().Add(ctx, paymentTransaction)
		if err != nil {
			zap.L().Error("PaymentTransaction().Add", zap.Error(err))
			_ = uow.Rollback()
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paymentTransaction, nil
}

func NewOrderService(dbRegistry *gormrepository.DatabaseRegistry, service iservice_third_party.PayOSService) iservice.OrderService {
	return &orderService{
		orderRepository:              dbRegistry.OrderRepository,
		orderItemRepository:          dbRegistry.OrderItemRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
		payOsService:                 service,
	}
}
