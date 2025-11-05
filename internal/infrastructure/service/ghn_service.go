package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/infrastructure/httpclient"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type ghnService struct {
	cfg    *config.AppConfig
	client *http.Client
}

// CalculateDeliveryPriceByID calculates delivery fee for an order by contacting GHN and returns the first fee result.
func (g ghnService) CalculateDeliveryPriceByID(ctx context.Context, orderID uuid.UUID, deliveryService dtos.DeliveryAvailableServiceDTO, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeeURL := g.cfg.GHN.FeeBaseURL + "/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// check order
		orderIncludes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, orderIncludes)
		if err != nil {
			return fmt.Errorf("order Not Found: %w", err)
		}

		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidation(order, deliveryService)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		deliveryFee, err = httpclient.DoRequestSingle[dtos.DeliveryFeeSuccess](ctx, g.client, g.cfg.GHN.Token, http.MethodPost, deliveryFeeURL, body)
		if err != nil {
			return fmt.Errorf("error when fetching delivery fee: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return &deliveryFee, nil
}

// GetAvailableDeliveryServicesByOrderID fetches available delivery services for an order from GHN.
func (g ghnService) GetAvailableDeliveryServicesByOrderID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) {
	deliverySvcURL := g.cfg.GHN.FeeBaseURL + "/available-services"
	var availableSvc []dtos.DeliveryAvailableServiceDTO

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// check order
		orderIncludes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, orderIncludes)
		if err != nil {
			return fmt.Errorf("order Not Found: %w", err)
		}

		// build http client body using order (GHN expects order/shipping info)
		body := dtos.DeliveryAvailableServiceBody{
			ShopID:       g.cfg.GHN.ShopID,
			FromDistrict: g.cfg.GHN.DistrictID,
			ToDistrict:   order.GhnDistrictID,
		}

		availableSvc, err = httpclient.DoRequestList[dtos.DeliveryAvailableServiceDTO](ctx, g.client, g.cfg.GHN.Token, http.MethodPost, deliverySvcURL, body)
		if err != nil {
			return fmt.Errorf("Error when fetching delivery fee: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get available delivery services: %w", err)
	}

	return availableSvc, nil
}

func (g ghnService) GetAvailableDeliveryServicesByDistrictID(ctx context.Context, districtID int, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) {
	deliverySvcURL := g.cfg.GHN.FeeBaseURL + "/available-services"
	var availableSvc []dtos.DeliveryAvailableServiceDTO

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		var err error
		// build http client body using order (GHN expects order/shipping info)
		body := dtos.DeliveryAvailableServiceBody{
			ShopID:       g.cfg.GHN.ShopID,
			FromDistrict: g.cfg.GHN.DistrictID,
			ToDistrict:   districtID,
		}

		availableSvc, err = httpclient.DoRequestList[dtos.DeliveryAvailableServiceDTO](ctx, g.client, g.cfg.GHN.Token, http.MethodPost, deliverySvcURL, body)
		if err != nil {
			return fmt.Errorf("Error when fetching delivery fee: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get available delivery services: %w", err)
	}

	return availableSvc, nil
}

func (g ghnService) CalculateDeliveryPriceByDimensionItems(ctx context.Context, toDistrictID int, toWardCode string, deliveryService dtos.DeliveryAvailableServiceDTO, items []dtos.ApplicationDeliveryFeeItem, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeeURL := g.cfg.GHN.FeeBaseURL + "/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {

		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV2(toDistrictID, toWardCode, items, deliveryService)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		deliveryFee, err = httpclient.DoRequestSingle[dtos.DeliveryFeeSuccess](ctx, g.client, g.cfg.GHN.Token, http.MethodPost, deliveryFeeURL, body)
		if err != nil {
			return fmt.Errorf("error when fetching delivery fee: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return &deliveryFee, nil

}

func (g ghnService) CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx context.Context, shippingAddressID uuid.UUID, deliveryService dtos.DeliveryAvailableServiceDTO, items []requests.OrderItemRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeeURL := g.cfg.GHN.FeeBaseURL + "/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// get shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, shippingAddressID, []string{})
		if err != nil {
			return fmt.Errorf("shipping address Not Found: %w", err)
		}

		// convert order items to application delivery fee items
		appDeliveryFeeItems, err := convertOrderItemRequestToApplicationDeliveryFeeItem(ctx, items, uow)
		if err != nil {
			return err
		}

		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV3(*address, appDeliveryFeeItems, deliveryService)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		deliveryFee, err = httpclient.DoRequestSingle[dtos.DeliveryFeeSuccess](ctx, g.client, g.cfg.GHN.Token, http.MethodPost, deliveryFeeURL, body)
		if err != nil {
			return fmt.Errorf("error when fetching delivery fee: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return &deliveryFee, nil
}

func NewGHNService(cfg *config.AppConfig) iservice_third_party.GHNService {
	client := &http.Client{Timeout: 60 * time.Second}
	return &ghnService{
		cfg:    cfg,
		client: client,
	}
}

// Helper
func convertOrderItemRequestToApplicationDeliveryFeeItem(ctx context.Context, items []requests.OrderItemRequest, uow irepository.UnitOfWork) ([]dtos.ApplicationDeliveryFeeItem, error) {
	var appDeliveryFeeItems []dtos.ApplicationDeliveryFeeItem
	for _, item := range items {
		//validate variant exists
		variant, err := uow.ProductVariant().GetByID(ctx, item.VariantID, []string{"Product"})
		if err != nil {
			return nil, fmt.Errorf("product variant not found, id: %s", item.VariantID.String())
		}
		variantPropConcat := fmt.Sprintf("%d", variant.Capacity) + utils.ToTitleCase(variant.CapacityUnit.String()) + " - " + utils.ToTitleCase(variant.ContainerType.String()) + " - " + utils.ToTitleCase(variant.DispenserType.String())
		variantName := utils.ToTitleCase(variant.Product.Name) + fmt.Sprintf(" (%s) ", variantPropConcat)

		appDeliveryFeeItem := dtos.ApplicationDeliveryFeeItem{
			Name:     variantName,
			Quantity: item.Quantity,
			Height:   variant.Height,
			Weight:   variant.Weight,
			Length:   variant.Length,
			Width:    variant.Width,
		}
		appDeliveryFeeItems = append(appDeliveryFeeItems, appDeliveryFeeItem)
	}
	return appDeliveryFeeItems, nil

}
