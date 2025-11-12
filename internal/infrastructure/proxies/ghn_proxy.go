package proxies

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/httpclient"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	mockSession     string
	mockSessionOnce sync.Once
)

type ghnProxy struct {
	*BaseProxy
	cfg    *config.AppConfig
	client *http.Client
	db     *gorm.DB
	//mocking
	tokenMutex   sync.Mutex
	cachedToken  *string
	tokenExpires time.Time
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
	deliveryFeePath := "/v2/shipping-order/fee"
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

		// Use BaseProxy.Post and decode GHN wrapper
		headers := map[string]string{
			"Token":  g.cfg.GHN.Token,
			"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		}
		resp, err := g.BaseProxy.Post(ctx, deliveryFeePath, headers, body)
		if err != nil {
			return fmt.Errorf("error when fetching delivery fee: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var ghnResp dtos.GHNWrapperResponse[dtos.DeliveryFeeSuccess]
		if err := json.NewDecoder(resp.Body).Decode(&ghnResp); err != nil {
			return fmt.Errorf("failed to decode GHN delivery fee response: %w", err)
		}
		if ghnResp.Code != 200 {
			return fmt.Errorf("error when fetching delivery fee: %s", ghnResp.Message)
		}
		deliveryFee = ghnResp.Data
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

		// Use httpclient helper for list endpoints (kept as-is)
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
	deliveryFeePath := "/v2/shipping-order/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {

		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV2(toDistrictID, toWardCode, items)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		headers := map[string]string{
			"Token":  g.cfg.GHN.Token,
			"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		}
		resp, err := g.BaseProxy.Post(ctx, deliveryFeePath, headers, body)
		if err != nil {
			return fmt.Errorf("error when fetching delivery fee: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var ghnResp dtos.GHNWrapperResponse[dtos.DeliveryFeeSuccess]
		if err := json.NewDecoder(resp.Body).Decode(&ghnResp); err != nil {
			return fmt.Errorf("failed to decode GHN delivery fee response: %w", err)
		}
		if ghnResp.Code != 200 {
			return fmt.Errorf("error when fetching delivery fee: %s", ghnResp.Message)
		}
		deliveryFee = ghnResp.Data
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return &deliveryFee, nil

}

func (g ghnProxy) CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx context.Context, shippingAddressID uuid.UUID, items []requests.OrderItemRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeePath := "/v2/shipping-order/fee"
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

		headers := map[string]string{
			"Token":  g.cfg.GHN.Token,
			"ShopId": fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		}
		resp, err := g.BaseProxy.Post(ctx, deliveryFeePath, headers, body)
		if err != nil {
			return fmt.Errorf("error when fetching delivery fee: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var ghnResp dtos.GHNWrapperResponse[dtos.DeliveryFeeSuccess]
		if err := json.NewDecoder(resp.Body).Decode(&ghnResp); err != nil {
			return fmt.Errorf("failed to decode GHN delivery fee response: %w", err)
		}
		if ghnResp.Code != 200 {
			return fmt.Errorf("error when fetching delivery fee: %s", ghnResp.Message)
		}
		deliveryFee = ghnResp.Data
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return &deliveryFee, nil
}

func (g ghnProxy) GetOrderInfo(ctx context.Context, orderID string) (*dtos.OrderInfo, error) {
	// Fetch ghnOrderCode from OrderID
	var order *model.Order
	err := g.db.WithContext(ctx).First(&order, "id = ?", orderID).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find order: %w", err)
	}
	orderCode := order.GHNOrderCode
	if orderCode == nil {
		return nil, fmt.Errorf("this order does not have associated GHN order code")
	}

	path := "/v2/shipping-order/detail"

	// build body
	body := map[string]string{"order_code": *orderCode}

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
		zap.L().Info("Successfully fetched GHN order info", zap.String("orderCode", *orderCode))
		return &orderInfoResp.Data, nil
	} else {
		zap.L().Error("Failed to fetch GHN order info", zap.String("orderCode", *orderCode), zap.String("responseCode", fmt.Sprintf("%d", orderInfoResp.Code)), zap.String("responseDesc", orderInfoResp.Message))
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

// ========================================================================= Webhook Mocking =========================================================================
func (g *ghnProxy) UpdateGHNDeliveryStatus(ctx context.Context, orderCode string, deliveryStatus enum.GHNDeliveryStatus) (*dtos.UpdateGHNDeliveryStatusResponse, error) {

	//1. Validate If the orderBelong to orderCode existed?
	var order model.Order
	//Find order by GHNOrderCode
	if err := g.db.WithContext(ctx).
		Where("ghn_order_code = ?", orderCode).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Order not found for GHN order code", zap.String("order_code", orderCode))
			return nil, fmt.Errorf("order not found for GHN order code")
		}
		return nil, fmt.Errorf("failed to find order by GHNOrderCode: %w", err)
	}

	url := "https://dev-online-gateway.ghn.vn/integration/tool-support/public-api/v2/order/switchStatus"

	body := map[string]interface{}{
		"order_code": orderCode,
		"status":     deliveryStatus,
	}

	token, err := g.getValidAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"token":        token,
		"Content-Type": "application/json",
	}

	resultPtr, err := doGHNRequest[[]dtos.UpdateGHNDeliveryStatusResponse](ctx, http.MethodPost, url, headers, body)
	if err != nil && strings.Contains(err.Error(), "Token không hợp lệ") {
		// Token hết hạn → refresh rồi retry
		g.tokenMutex.Lock()
		g.cachedToken = nil
		g.tokenMutex.Unlock()

		token, err = g.getValidAccessToken(ctx)
		if err != nil {
			return nil, err
		}
		headers["token"] = token

		resultPtr, err = doGHNRequest[[]dtos.UpdateGHNDeliveryStatusResponse](ctx, http.MethodPost, url, headers, body)
		if err != nil {
			return nil, err
		}
	}

	if resultPtr == nil || len(*resultPtr) == 0 {
		return nil, fmt.Errorf("empty response from GHN")
	}

	firstRes := (*resultPtr)[0]

	// ✅ Nếu GHN trả Result=true thì thực hiện side-effect
	if firstRes.Result {
		zap.L().Info("Handling Side Effect")
		if firstRes.Result {
			zap.L().Info("Handling Side Effect for GHN status", zap.String("status", string(deliveryStatus)))
			if err := g.handleSideEffect(ctx, deliveryStatus, &order); err != nil {
				zap.L().Warn("side effect failed", zap.Error(err))
			}
		}
	}

	return &firstRes, nil
}

func (g *ghnProxy) handleSideEffect(ctx context.Context, deliveryStatus enum.GHNDeliveryStatus, order *model.Order) error {
	// Map GHNDeliveryStatus → OrderStatus
	var newStatus enum.OrderStatus
	switch deliveryStatus {
	case enum.GHNDeliveryStatusStoring:
		newStatus = enum.OrderStatusShipped
	case enum.GHNDeliveryStatusDelivering:
		newStatus = enum.OrderStatusInTransit
	case enum.GHNDeliveryStatusDelivered:
		newStatus = enum.OrderStatusDelivered
	default:
		zap.L().Info("GHN status does not trigger side effect", zap.String("status", string(deliveryStatus)))
	}

	if err := g.db.WithContext(ctx).
		Model(&order).
		Update("status", newStatus).Error; err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	zap.L().Info("Order status updated successfully",
		zap.String("order_id", order.ID.String()),
		zap.String("order_code", *order.GHNOrderCode),
		zap.String("old_status", string(order.Status)),
		zap.String("new_status", string(newStatus)))

	return nil
}

// 1️⃣ Mock Session
func (g *ghnProxy) GetSession(ctx context.Context) (*dtos.GHNSessionResponse, error) {
	body := map[string]interface{}{
		"user_id":    g.cfg.GHN.MockSessionInfo.UserID,
		"password":   g.cfg.GHN.MockSessionInfo.Password,
		"device_id":  g.cfg.GHN.MockSessionInfo.DeviceID,
		"user_agent": g.cfg.GHN.MockSessionInfo.UserAgent,
	}

	return doGHNRequest[dtos.GHNSessionResponse](ctx, http.MethodGet, g.cfg.GHN.MockSessionInfo.MockURL, map[string]string{
		"Content-Type": "application/json",
		"User-Agent":   g.cfg.GHN.MockSessionInfo.UserAgent,
	}, body)
}

// 2️⃣ Service Token
func (g *ghnProxy) GetGHNServiceToken(ctx context.Context, ssoToken string) (*dtos.GHNServiceToken, error) {
	body := map[string]interface{}{
		"app_key":       "431f8318-bfed-40a6-9611-71a2f7d67025",
		"response_type": "authorization_code",
	}
	headers := map[string]string{
		"token":        ssoToken,
		"Content-Type": "application/json",
	}

	return doGHNRequest[dtos.GHNServiceToken](ctx, http.MethodPost, "https://dev-online-gateway.ghn.vn/sso-v2/public-api/staff/gen-service-token", headers, body)
}

// 3️⃣ GSO Token
func (g *ghnProxy) GetGHNGSOToken(ctx context.Context, authorizationCode string) (*dtos.GHNTokenGSO, error) {
	body := map[string]interface{}{
		"authorization_code": authorizationCode,
	}
	headers := map[string]string{
		"token":        authorizationCode,
		"Content-Type": "application/json",
	}

	return doGHNRequest[dtos.GHNTokenGSO](ctx, http.MethodPost, "https://dev-online-gateway.ghn.vn/integration/tool-support/public-api/auth/generateTokenSSO", headers, body)
}

// Comparison of 3 above
func (g *ghnProxy) getValidAccessToken(ctx context.Context) (string, error) {
	g.tokenMutex.Lock()
	defer g.tokenMutex.Unlock()

	// Nếu token còn hạn thì trả về luôn
	if g.cachedToken != nil && time.Now().Before(g.tokenExpires) {
		return *g.cachedToken, nil
	}

	zap.L().Info("Fetching new GHN AccessToken via Postman flow")

	// 1️⃣ Lấy sso_token từ Mock Session
	session, err := g.GetSession(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get GHN session: %w", err)
	}

	// 2️⃣ Lấy Service Token từ sso_token
	serviceToken, err := g.GetGHNServiceToken(ctx, session.SsoToken)
	if err != nil {
		return "", fmt.Errorf("failed to get GHN service token: %w", err)
	}

	// 3️⃣ Lấy GSO Access Token từ Service Token (authorization_code)
	gsoToken, err := g.GetGHNGSOToken(ctx, serviceToken.Code)
	if err != nil {
		return "", fmt.Errorf("failed to get GHN GSO token: %w", err)
	}

	// 4️⃣ Cache token (ví dụ 25 phút)
	g.cachedToken = &gsoToken.AccessToken
	g.tokenExpires = time.Now().Add(25 * time.Minute)
	zap.L().Info("GHN AccessToken cached", zap.String("token", *g.cachedToken))

	return *g.cachedToken, nil
}

// ========================================================================= Webhook Mocking ===help me======================================================================
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

	userNote := ""
	if order.UserNote != nil {
		userNote = *order.UserNote
	}

	clientOrderCode := order.ID.String()
	if clientOrderCode == "" {
		clientOrderCode = order.UserID.String()
	}

	return &dtos.CreateGHNOrderDTO{
		PaymentTypeID:    2,
		Note:             userNote,
		RequiredNote:     "CHOXEMHANGKHONGTHU",
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

func doGHNRequest[T any](ctx context.Context, method, url string, headers map[string]string, body any) (*T, error) {
	start := time.Now()

	// Marshal body
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		zap.L().Error("GHN request failed",
			zap.String("url", url),
			zap.Any("headers", headers),
			zap.Any("body", body),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			zap.L().Warn("failed to close GHN response body", zap.Error(err))
		}
	}()

	var wrapper dtos.GHNWrapperResponse[T]
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		zap.L().Error("GHN decode response failed",
			zap.String("url", url),
			zap.Int("status_code", resp.StatusCode),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to decode GHN response: %w", err)
	}

	zap.L().Info("GHN request completed",
		zap.String("url", url),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", duration),
		zap.Any("body", body),
		zap.Any("response", wrapper),
	)

	if resp.StatusCode != http.StatusOK || wrapper.Code != 200 {
		return nil, fmt.Errorf("GHN error: http=%d code=%d msg=%s",
			resp.StatusCode, wrapper.Code, wrapper.Message)
	}

	return &wrapper.Data, nil
}
