package iservice_third_party

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type GHNService interface {
	CalculateDeliveryPriceByID(ctx context.Context, orderID uuid.UUID, deliveryService dtos.DeliveryAvailableServiceDTO, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	GetAvailableDeliveryServicesByOrderID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error)
	//PublicAPI
	CalculateDeliveryPriceByDimensionItems(ctx context.Context, toDistrictID int, toWardCode string, deliveryService dtos.DeliveryAvailableServiceDTO, items []dtos.ApplicationDeliveryFeeItem, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	GetAvailableDeliveryServicesByDistrictID(ctx context.Context, districtID int, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error)
}
