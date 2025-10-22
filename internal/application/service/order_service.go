package service

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
)

type orderService struct {
	orderRepository              irepository.GenericRepository[model.Order]
	orderItemRepository          irepository.GenericRepository[model.OrderItem]
	paymentTransactionRepository irepository.GenericRepository[model.PaymentTransaction]
}

func (o orderService) PlaceOrder(request requests.OrderRequest) {
	//Create Item
}

func NewOrderService(dbRegistry *gormrepository.DatabaseRegistry) *orderService {
	return &orderService{
		orderRepository:              dbRegistry.OrderRepository,
		orderItemRepository:          dbRegistry.OrderItemRepository,
		paymentTransactionRepository: dbRegistry.PaymentTransactionRepository,
	}
}
