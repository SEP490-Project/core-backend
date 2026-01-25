package iproxies

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

type GHNProxy interface {
	// Delivery Management
	CalculateDeliveryPriceByID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx context.Context, shippingAddressID uuid.UUID, items []requests.OrderItemRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	CalculateDeliveryPriceByShippingAddressAndPreOrder(ctx context.Context, item requests.PreOrderRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	// GetAvailableDeliveryServicesByOrderID
	//@Deprecated
	GetAvailableDeliveryServicesByOrderID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error)

	//Order Management
	GetOrderInfo(ctx context.Context, orderID string) (*dtos.OrderInfo, error)
	GetOrderInfoRaw(ctx context.Context, ghnCode string) (*dtos.OrderInfo, error)
	GetAvailableNextActions(*dtos.OrderInfo) (map[string]bool, error)
	CancelOrder(ctx context.Context, orderCode string) (*dtos.CancelOrder, error)
	CreateOrder(ctx context.Context, orderID uuid.UUID) (*dtos.CreatedGHNOrderResponse, error)
	CreatePreOrder(ctx context.Context, preOrderID uuid.UUID) (*dtos.CreatedGHNOrderResponse, error)

	//PublicAPI
	CalculateDeliveryPriceByDimensionItems(ctx context.Context, toDistrictID int, toWardCode string, items []dtos.ApplicationDeliveryFeeItem, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error)
	GetAvailableDeliveryServicesByDistrictID(ctx context.Context, districtID int, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error)
	GetExpectedDeliveryTime(ctx context.Context, toDistrictID int, toWardCode string) (*dtos.ExpectedDeliveryTime, error)

	//GHN Webhook Mocking Service

	//Tokens
	GetSession(ctx context.Context) (*dtos.GHNSessionResponse, error)
	GetGHNServiceToken(ctx context.Context, ghnSession string) (*dtos.GHNServiceToken, error)
	GetGHNGSOToken(ctx context.Context, serviceToken string) (*dtos.GHNTokenGSO, error)

	UpdateGHNDeliveryStatus(ctx context.Context, ghnOrderCode string, deliveryStatus enum.GHNDeliveryStatus) (*dtos.UpdateGHNDeliveryStatusResponse, error)
}
