package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"time"
)

type orderService struct {
	orderRepository              irepository.GenericRepository[model.Order]
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
}

func (o orderService) PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, unitOfWork irepository.UnitOfWork) (*model.Order, error) {
	now := time.Now()
	var persistedOrder *model.Order

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Create Item
		var persistedOrderItem []model.OrderItem

		for _, item := range request.Items {
			//check variantID:
			var includes []string
			variant, err := uow.ProductVariant().GetByID(ctx, item.ProductVariantID, includes)
			if err != nil {
				zap.L().Error("ProductVariant().ExistsByID", zap.Error(err))
				_ = uow.Rollback()
			} else if variant == nil {
				_ = uow.Rollback()
				return errors.New("Product Variant not found")
			}
			persistedItem := item.ToModel(*variant, now)
			persistedOrderItem = append(persistedOrderItem, *persistedItem)
		}

		//Create Order
		persistedOrder := request.ToModel(userID, persistedOrderItem, now)
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

func NewOrderService(dbRegistry *gormrepository.DatabaseRegistry) *orderService {
	return &orderService{
		orderRepository:              dbRegistry.OrderRepository,
		orderItemRepository:          dbRegistry.OrderItemRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
	}
}
