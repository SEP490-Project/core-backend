package iservice_third_party

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type GHNService interface {
	// Delivery Management
	CalculateDeliveryPriceByID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx context.Context, shippingAddressID uuid.UUID, items []requests.OrderItemRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	GetAvailableDeliveryServicesByOrderID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) //Deprecated

	//Order Management
	GetOrderInfo(ctx context.Context, orderCode string) (*dtos.OrderInfo, error)
	CancelOrder(ctx context.Context, orderCode string) (*dtos.CancelOrder, error)

	//PublicAPI
	CalculateDeliveryPriceByDimensionItems(ctx context.Context, toDistrictID int, toWardCode string, items []dtos.ApplicationDeliveryFeeItem, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	GetAvailableDeliveryServicesByDistrictID(ctx context.Context, districtID int, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error)
	GetExpectedDeliveryTime(ctx context.Context, toDistrictID int, toWardCode string) (float64, error)
}
