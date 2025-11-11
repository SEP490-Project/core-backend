package proxies

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/httpclient"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
)

type ghnProxy struct {
	*BaseProxy
	cfg    *config.AppConfig
	client *http.Client
	db     *gorm.DB
}

func (g ghnProxy) CreateOrder(ctx context.Context, orderID uuid.UUID) (*dtos.CreatedGHNOrderResponse, error) {
	// find order with items and variant/product data
	var order model.Order
	if err := g.db.WithContext(ctx).
		Preload("OrderItems").
		Preload("OrderItems.Variant").
		Preload("OrderItems.Variant.Product").
		First(&order, "id = ?", orderID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("order not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query order: %w", err)
	}

	//// try to find related payment transaction (most recent)
	//var pt model.PaymentTransaction
	//if err := g.db.WithContext(ctx).
	//	Where("reference_id = ? AND reference_type = ?", orderID, enum.PaymentTransactionReferenceTypeOrder).
	//	Order("created_at desc").
	//	First(&pt).Error; err == nil {
	//	order.PaymentID = &pt.ID
	//	if pt.PayOSMetadata != nil {
	//		order.PaymentBin = &pt.PayOSMetadata.Bin
	//	}
	//} else if err != nil && err != gorm.ErrRecordNotFound {
	//	// non-fatal: log and continue
	//	zap.L().Warn("failed to lookup payment transaction for order", zap.Error(err), zap.String("order_id", orderID.String()))
	//}

	// convert order to GHN request DTO
	dto := convertOrderToGHNOrderCreationDTO(&order)

	// call GHN create endpoint
	path := "/v2/shipping-order/create"
	headers := map[string]string{
		"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		"Token":  g.cfg.GHN.Token,
	}

	resp, err := g.BaseProxy.Post(ctx, path, headers, dto)
	if err != nil {
		return nil, fmt.Errorf("failed to call GHN create order: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// decode response generically
	var ghnResp dtos.GHNWrapperResponse[dtos.CreatedGHNOrderResponse]
	if err := json.NewDecoder(resp.Body).Decode(&ghnResp); err != nil {
		return nil, fmt.Errorf("failed to decode GHN response: %w", err)
	}

	if ghnResp.Code != 200 {
		return nil, fmt.Errorf("GHN create order failed: %s", ghnResp.Message)
	}

	zap.L().Info("GHN create order succeeded", zap.Any("response_data", ghnResp.Data), zap.String("order_id", order.ID.String()))

	return &ghnResp.Data, nil
}

// CalculateDeliveryPriceByID calculates delivery fee for an order by contacting GHN and returns the first fee result.
func (g ghnProxy) CalculateDeliveryPriceByID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeeURL := g.cfg.GHN.BaseURL + "/v2/shipping-order/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// check order
		orderIncludes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, orderIncludes)
		if err != nil {
			return fmt.Errorf("order Not Found: %w", err)
		}

		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidation(order)
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
func (g ghnProxy) GetAvailableDeliveryServicesByOrderID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) {
	deliverySvcURL := g.cfg.GHN.BaseURL + "/v2/shipping-order/available-services"
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

func (g ghnProxy) GetAvailableDeliveryServicesByDistrictID(ctx context.Context, districtID int, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) {
	deliverySvcURL := g.cfg.GHN.BaseURL + "/v2/shipping-order/available-services"
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

func (g ghnProxy) CalculateDeliveryPriceByDimensionItems(ctx context.Context, toDistrictID int, toWardCode string, items []dtos.ApplicationDeliveryFeeItem, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeeURL := g.cfg.GHN.BaseURL + "/v2/shipping-order/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {

		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV2(toDistrictID, toWardCode, items)
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

func (g ghnProxy) CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx context.Context, shippingAddressID uuid.UUID, items []requests.OrderItemRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeeURL := g.cfg.GHN.BaseURL + "/v2/shipping-order/fee"
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
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV3(*address, appDeliveryFeeItems)
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

func (g ghnProxy) GetOrderInfo(ctx context.Context, orderCode string) (*dtos.OrderInfo, error) {
	path := "/v2/shipping-order/detail"

	// build body
	body := map[string]string{"order_code": orderCode}

	headers := map[string]string{
		"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		"Token":  g.cfg.GHN.Token,
	}

	resp, err := g.BaseProxy.Post(ctx, path, headers, body)
	if err != nil {
		zap.L().Error("Failed to execute GHN order detail request", zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Parse response
	var orderInfoResp dtos.GHNWrapperResponse[dtos.OrderInfo]
	if err := json.NewDecoder(resp.Body).Decode(&orderInfoResp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	//Check response code
	if orderInfoResp.Code == 200 {
		zap.L().Info("Successfully fetched GHN order info", zap.String("orderCode", orderCode))
		return &orderInfoResp.Data, nil
	} else {
		zap.L().Error("Failed to fetch GHN order info", zap.String("orderCode", orderCode), zap.String("responseCode", fmt.Sprintf("%d", orderInfoResp.Code)), zap.String("responseDesc", orderInfoResp.Message))
		return nil, fmt.Errorf("failed to fetch GHN order info: %s", orderInfoResp.Message)
	}
}

func (g ghnProxy) CancelOrder(ctx context.Context, orderCode string) (*dtos.CancelOrder, error) {
	// Call GHN switch-status cancel endpoint
	path := "/v2/switch-status/cancel"
	// request body expects an array of order codes (we only send one)
	body := map[string][]string{"order_codes": {orderCode}}
	headers := map[string]string{
		"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		"Token":  g.cfg.GHN.Token,
	}

	resp, err := g.BaseProxy.Post(ctx, path, headers, body)
	if err != nil {
		zap.L().Error("Failed to call GHN cancel endpoint", zap.Error(err), zap.String("orderCode", orderCode))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var cancelResp dtos.GHNWrapperResponse[[]dtos.CancelOrder]
	if err := json.NewDecoder(resp.Body).Decode(&cancelResp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if cancelResp.Code != 200 {
		zap.L().Error("GHN cancel returned non-200 code", zap.Int("code", cancelResp.Code), zap.String("message", cancelResp.Message), zap.String("orderCode", orderCode))
		return nil, fmt.Errorf("failed to cancel order: %s", cancelResp.Message)
	}

	// Only cancel one at a time: return the first result if present
	if len(cancelResp.Data) > 0 {
		item := cancelResp.Data[0]
		zap.L().Info("GHN cancel result", zap.String("orderCode", orderCode), zap.Bool("result", item.Result), zap.String("message", item.Message))
		return &item, nil
	}

	return nil, fmt.Errorf("no cancel result returned for order: %s", orderCode)
}

func (g ghnProxy) GetExpectedDeliveryTime(ctx context.Context, toDistrictID int, toWardCode string) (*dtos.ExpectedDeliveryTime, error) {
	path := "/v2/shipping-order/leadtime"

	body := struct {
		ToDistrictID int    `json:"to_district_id"`
		ToWardCode   string `json:"to_ward_code"`
	}{
		ToDistrictID: toDistrictID,
		ToWardCode:   toWardCode,
	}

	headers := map[string]string{
		"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		"Token":  g.cfg.GHN.Token,
	}

	resp, err := g.BaseProxy.Post(ctx, path, headers, body)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var estTimeResp dtos.GHNWrapperResponse[dtos.ExpectedDeliveryTime]
	if err := json.NewDecoder(resp.Body).Decode(&estTimeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	if estTimeResp.Code == 200 {
		zap.L().Info("Successfully fetched GHN expected delivery time", zap.Int("toDistrictID", toDistrictID), zap.String("toWardCode", toWardCode))
		return &estTimeResp.Data, nil
	} else {
		zap.L().Error("Failed to fetch GHN expected delivery time", zap.Int("toDistrictID", toDistrictID), zap.String("toWardCode", toWardCode), zap.String("responseCode", fmt.Sprintf("%d", estTimeResp.Code)), zap.String("responseDesc", estTimeResp.Message))
		return nil, fmt.Errorf("failed to fetch GHN expected delivery time: %s", estTimeResp.Message)
	}

}

func NewGHNProxy(httpClient *http.Client, cfg *config.AppConfig, db *gorm.DB) iproxies.GHNProxy {
	return &ghnProxy{
		BaseProxy: NewBaseProxy(httpClient, cfg.GHN.BaseURL),
		cfg:       cfg,
		db:        db,
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

func convertOrderToGHNOrderCreationDTO(order *model.Order) *dtos.CreateGHNOrderDTO {
	var totalLength, totalWeight, totalHeight, totalWidth int
	var dtoItems []dtos.ApplicationDeliveryFeeItem

	for _, item := range order.OrderItems {
		totalWeight += item.Variant.Weight * item.Quantity
		totalLength += item.Variant.Length * item.Quantity
		totalHeight += item.Variant.Height * item.Quantity
		totalWidth += item.Variant.Width * item.Quantity
		dtoItems = append(dtoItems, dtos.ApplicationDeliveryFeeItem{}.ToApplicationDeliveryFeeItem(item))
	}

	requiredNote := ""
	if order.UserNotes != nil {
		requiredNote = *order.UserNotes
	}

	clientOrderCode := order.ID.String()
	if clientOrderCode == "" {
		clientOrderCode = order.UserID.String()
	}

	return &dtos.CreateGHNOrderDTO{
		PaymentTypeID:    2,
		Note:             "",
		RequiredNote:     requiredNote,
		FromName:         "BShowSell.Co",
		FromPhone:        "0944488274",
		FromAddress:      "7 Đ. D1",
		FromWardName:     "Long Thạnh Mỹ",
		FromDistrictName: "Thủ Đức",
		FromProvinceName: "Hồ Chí Minh",
		ClientOrderCode:  clientOrderCode,
		ToName:           order.FullName,
		ToPhone:          order.PhoneNumber,
		ToAddress:        order.AddressLine2,
		ToWardCode:       order.GhnWardCode,
		ToDistrictID:     order.GhnDistrictID,
		Content:          "",
		Weight:           totalWeight,
		Length:           totalLength,
		Width:            totalWidth,
		Height:           totalHeight,
		InsuranceValue:   totalWidth,
		ServiceID:        0,
		ServiceTypeID:    2,
		Items:            dtoItems,
	}
}
