package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
)

type OrderService interface {
	PlaceOrder(ctx context.Context, userID uuid.UUID, request requests.OrderRequest, unitOfWork irepository.UnitOfWork) (*model.Order, error)
}
