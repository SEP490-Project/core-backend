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
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/httpclient"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	mockSession     string
	mockSessionOnce sync.Once
)

type ghnProxy struct {
	*BaseProxy
	cfg      *config.AppConfig
	adminCfg *config.AdminConfig
	client   *http.Client
	db       *gorm.DB
	//mocking
	tokenMutex   sync.Mutex
	cachedToken  *string
	tokenExpires time.Time
	provinceRepo irepository.GenericRepository[model.Province]
	districtRepo irepository.GenericRepository[model.District]
	wardRepo     irepository.GenericRepository[model.Ward]
}

func (g *ghnProxy) GetAvailableNextActions(info *dtos.OrderInfo) (map[string]bool, error) {
	currentState := info.Status
	return getAvailableState(currentState)
}

func (g *ghnProxy) CancelOrder(ctx context.Context, orderCode string) (*dtos.CancelOrder, error) {
	//TODO implement me
	panic("implement me")
}

func (g *ghnProxy) CreateOrder(ctx context.Context, orderID uuid.UUID) (*dtos.CreatedGHNOrderResponse, error) {
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

	dto := g.convertOrderToGHNOrderCreationDTO(&order)

	// call GHN create endpoint
	url := g.cfg.GHN.BaseURL + "/v2/shipping-order/create"
	headers := map[string]string{
		"Content-Type": "application/json",
		"ShopId":       fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		"Token":        g.cfg.GHN.Token,
	}

	return doGHNRequest[dtos.CreatedGHNOrderResponse](ctx, http.MethodPost, url, headers, dto)
}

// CalculateDeliveryPriceByID calculates delivery fee for an order by contacting GHN and returns the first fee result.
func (g *ghnProxy) CalculateDeliveryPriceByID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeePath := g.cfg.GHN.BaseURL + "/v2/shipping-order/fee"
	var deliveryFee *dtos.DeliveryFeeSuccess

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
			"Content-Type": "application/json",
			"Token":        g.cfg.GHN.Token,
			"ShopId":       fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		}

		deliveryFee, _ = doGHNRequest[dtos.DeliveryFeeSuccess](ctx, http.MethodPost, deliveryFeePath, headers, body)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return deliveryFee, nil
}

// GetAvailableDeliveryServicesByOrderID fetches available delivery services for an order from GHN.
//
//	@Deprecated
func (g *ghnProxy) GetAvailableDeliveryServicesByOrderID(ctx context.Context, orderID uuid.UUID, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) {
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

// @Deprecated
func (g *ghnProxy) GetAvailableDeliveryServicesByDistrictID(ctx context.Context, districtID int, unitOfWork irepository.UnitOfWork) ([]dtos.DeliveryAvailableServiceDTO, error) {
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

func (g *ghnProxy) CalculateDeliveryPriceByDimensionItems(ctx context.Context, toDistrictID int, toWardCode string, items []dtos.ApplicationDeliveryFeeItem, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeePath := g.cfg.GHN.BaseURL + "/v2/shipping-order/fee"
	var deliveryFee *dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {

		var err error = nil
		// build http client body using order
		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV2(toDistrictID, toWardCode, items)
		if err != nil {
			return fmt.Errorf(err.Error())
		}

		headers := map[string]string{
			"Content-Type": "application/json",
			"Token":        g.cfg.GHN.Token,
			"ShopId":       fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		}

		deliveryFee, err = doGHNRequest[dtos.DeliveryFeeSuccess](ctx, http.MethodPost, deliveryFeePath, headers, body)
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return deliveryFee, nil

}

func (g *ghnProxy) CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx context.Context, shippingAddressID uuid.UUID, items []requests.OrderItemRequest, unitOfWork irepository.UnitOfWork) (*dtos.DeliveryFeeSuccess, error) {
	deliveryFeePath := g.cfg.GHN.BaseURL + "/v2/shipping-order/fee"
	var deliveryFee dtos.DeliveryFeeSuccess

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Get shipping address
		address, err := uow.ShippingAddresses().GetByID(ctx, shippingAddressID, []string{})
		if err != nil {
			return fmt.Errorf("shipping address not found: %w", err)
		}

		appDeliveryFeeItems, err := convertOrderItemRequestToApplicationDeliveryFeeItem(ctx, items, uow)
		if err != nil {
			return fmt.Errorf("failed to convert order items: %w", err)
		}

		body, err := dtos.DeliveryFeeBody{}.ToDeliveryFeeBodyDTOWithValidationV3(*address, appDeliveryFeeItems)
		if err != nil {
			return fmt.Errorf("invalid delivery fee body: %w", err)
		}

		headers := map[string]string{
			"Content-Type": "application/json",
			"Token":        g.cfg.GHN.Token,
			"ShopId":       fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		}

		resp, err := doGHNRequest[dtos.DeliveryFeeSuccess](ctx, http.MethodPost, deliveryFeePath, headers, body)
		if err != nil {
			zap.L().Info("failed to calculate delivery price, use default: 23.450k", zap.Error(err))
			deliveryFee = dtos.DeliveryFeeSuccess{}
			deliveryFee.Total = 23450
			deliveryFee.ServiceFee = 23450
		} else {
			deliveryFee = *resp
		}
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to calculate delivery price: %w", err)
	}

	return &deliveryFee, nil
}

func (g *ghnProxy) GetOrderInfo(ctx context.Context, orderID string) (*dtos.OrderInfo, error) {
	var order model.Order
	if err := g.db.WithContext(ctx).First(&order, "id = ?", orderID).Error; err != nil {
		return nil, fmt.Errorf("failed to find order: %w", err)
	}

	if order.GHNOrderCode == nil {
		return nil, fmt.Errorf("this order does not have an associated GHN order code")
	}

	orderCode := *order.GHNOrderCode
	url := g.cfg.GHN.BaseURL + "/v2/shipping-order/detail"

	body := map[string]string{
		"order_code": orderCode,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Token":        g.cfg.GHN.Token,
	}
	return doGHNRequest[dtos.OrderInfo](ctx, http.MethodPost, url, headers, body)
}

func (g *ghnProxy) GetOrderInfoRaw(ctx context.Context, ghnCode string) (*dtos.OrderInfo, error) {
	url := g.cfg.GHN.BaseURL + "/v2/shipping-order/detail"
	body := map[string]string{
		"order_code": ghnCode,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"Token":        g.cfg.GHN.Token,
	}
	info, err := doGHNRequest[dtos.OrderInfo](ctx, http.MethodPost, url, headers, body)
	if err != nil {
		return nil, err
	}

	extractedInfo := g.extractLocationNameFromInfo(ctx, info)
	return extractedInfo, nil
}

func (g *ghnProxy) extractLocationNameFromInfo(ctx context.Context, info *dtos.OrderInfo) *dtos.OrderInfo {
	var mu sync.Mutex

	err := utils.RunParallel(ctx, 3,
		// Get "Return" location
		func(ctx context.Context) error {
			// Ward Name
			includes := []string{"District", "District.Province"}
			ward, err := g.wardRepo.GetByID(ctx, info.ReturnWardCode, includes)
			if err != nil {
				zap.L().Warn("Failed to get Ward of \"Return\" location ", zap.Error(err))
				return nil
			}
			mu.Lock()
			info.ReturnWardName = ward.Name
			info.ReturnDistrictName = ward.District.Name
			info.ReturnProvinceName = ward.District.Province.Name
			mu.Unlock()
			return nil
		},
		// Get "From" location
		func(ctx context.Context) error {
			// Ward Name
			includes := []string{"District", "District.Province"}
			ward, err := g.wardRepo.GetByID(ctx, info.FromWardCode, includes)
			if err != nil {
				zap.L().Warn("Failed to get Ward of \"From\" location ", zap.Error(err))
				return nil
			}
			mu.Lock()
			info.FromWardName = ward.Name
			info.FromDistrictName = ward.District.Name
			info.FromProvinceName = ward.District.Province.Name
			mu.Unlock()
			return nil
		},
		// Get "To" location
		func(ctx context.Context) error {
			// Ward Name
			includes := []string{"District", "District.Province"}
			ward, err := g.wardRepo.GetByID(ctx, info.ToWardCode, includes)
			if err != nil {
				zap.L().Warn("Failed to get Ward of \"To\" location ", zap.Error(err))
				return nil
			}
			mu.Lock()
			info.ToWardName = ward.Name
			info.ToDistrictName = ward.District.Name
			info.ToProvinceName = ward.District.Province.Name
			mu.Unlock()
			return nil
		},
	)
	if err != nil {
		zap.L().Error("failed to extract location name from info, return raw as api", zap.Error(err))
	}
	return info
}

func (g *ghnProxy) GetExpectedDeliveryTime(ctx context.Context, toDistrictID int, toWardCode string) (*dtos.ExpectedDeliveryTime, error) {
	url := g.cfg.GHN.BaseURL + "/v2/shipping-order/leadtime"

	body := map[string]interface{}{
		"to_district_id": toDistrictID,
		"to_ward_code":   toWardCode,
	}

	headers := map[string]string{
		"Content-Type": "application/json",
		"ShopId":       fmt.Sprintf("%d", g.cfg.GHN.ShopID),
		"Token":        g.cfg.GHN.Token,
	}

	return doGHNRequest[dtos.ExpectedDeliveryTime](ctx, http.MethodGet, url, headers, body)
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

	if firstRes.Result {
		zap.L().Info("Handling Side Effect")
		if firstRes.Result {
			zap.L().Info("Handling Side Effect for GHN status", zap.String("status", string(deliveryStatus)))
			go func() {
				if err := g.handleSideEffect(context.Background(), deliveryStatus, &order); err != nil {
					zap.L().Error("Failed to fire webhook", zap.Error(err))
				}
			}()
		}
	}

	return &firstRes, nil
}

func (g *ghnProxy) handleSideEffect(ctx context.Context, deliveryStatus enum.GHNDeliveryStatus, order *model.Order) error {
	webhookURL := fmt.Sprintf(
		"http://localhost:8080/api/v1/ghn/webhook?status=%s&code=%s",
		string(deliveryStatus),
		*order.GHNOrderCode,
	)

	// Make HTTP GET request to the webhook
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, webhookURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call GHN webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GHN webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	zap.L().Info("GHN webhook called successfully",
		zap.String("order_id", order.ID.String()),
		zap.String("order_code", *order.GHNOrderCode),
		zap.String("sent status", deliveryStatus.String()),
	)

	return nil
}

// Mock Session
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

// Service Token
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

// GSO Token
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

	if g.cachedToken != nil && time.Now().Before(g.tokenExpires) {
		return *g.cachedToken, nil
	}

	zap.L().Info("Fetching new GHN AccessToken via Postman flow")

	// Lấy sso_token từ Mock Session
	session, err := g.GetSession(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get GHN session: %w", err)
	}

	// Lấy Service Token từ sso_token
	serviceToken, err := g.GetGHNServiceToken(ctx, session.SsoToken)
	if err != nil {
		return "", fmt.Errorf("failed to get GHN service token: %w", err)
	}

	// Lấy GSO Access Token từ Service Token (authorization_code)
	gsoToken, err := g.GetGHNGSOToken(ctx, serviceToken.Code)
	if err != nil {
		return "", fmt.Errorf("failed to get GHN GSO token: %w", err)
	}

	//  Cache token (ví dụ 25 phút)
	g.cachedToken = &gsoToken.AccessToken
	g.tokenExpires = time.Now().Add(25 * time.Minute)
	zap.L().Info("GHN AccessToken cached", zap.String("token", *g.cachedToken))

	return *g.cachedToken, nil
}

func NewGHNProxy(httpClient *http.Client, cfg *config.AppConfig, db *gorm.DB, dbReg *gormrepository.DatabaseRegistry) iproxies.GHNProxy {
	return &ghnProxy{
		BaseProxy:    NewBaseProxy(httpClient, cfg.GHN.BaseURL, cfg),
		cfg:          cfg,
		adminCfg:     &cfg.AdminConfig,
		db:           db,
		provinceRepo: dbReg.ProvinceRepository,
		districtRepo: dbReg.DistrictRepository,
		wardRepo:     dbReg.WardRepository,
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
		variantPropConcat := fmt.Sprintf("%d", variant.Capacity) + utils.ToTitleCase(variant.CapacityUnit) + " - " + utils.ToTitleCase(variant.ContainerType) + " - " + utils.ToTitleCase(variant.DispenserType)
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

func (g *ghnProxy) convertOrderToGHNOrderCreationDTO(order *model.Order) *dtos.CreateGHNOrderDTO {
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

	fromName := g.adminCfg.RepresentativeGHNCompanyName
	fromPhone := g.adminCfg.RepresentativeGHNPhone
	fromAddress := g.adminCfg.RepresentativeCompanyAddress
	//fromWardName := g.adminCfg.RepresentativeGHNWardName
	//fromDistrictName := g.adminCfg.RepresentativeGHNDistrictName
	//fromProvinceName := g.adminCfg.RepresentativeGHNProvinceName

	return &dtos.CreateGHNOrderDTO{
		PaymentTypeID: 2,
		Note:          userNote,
		RequiredNote:  "CHOXEMHANGKHONGTHU",
		FromName:      fromName,
		FromPhone:     fromPhone,
		FromAddress:   fromAddress,
		//FromWardName:     fromWardName,
		//FromDistrictName: fromDistrictName,
		//FromProvinceName: fromProvinceName,
		ClientOrderCode: clientOrderCode,
		ToName:          order.FullName,
		ToPhone:         order.PhoneNumber,
		ToAddress:       order.AddressLine2,
		ToWardCode:      order.GhnWardCode,
		ToDistrictID:    order.GhnDistrictID,
		Content:         "",
		Weight:          totalWeight,
		Length:          totalLength,
		Width:           totalWidth,
		Height:          totalHeight,
		InsuranceValue:  totalWidth,
		ServiceID:       0,
		ServiceTypeID:   2,
		Items:           dtoItems,
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
	defer resp.Body.Close()

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
		zap.Any("request_body", body),
		zap.Any("response", wrapper),
	)

	if resp.StatusCode != http.StatusOK || wrapper.Code != 200 {
		return nil, fmt.Errorf("GHN API error: http=%d, code=%d, message=%s",
			resp.StatusCode, wrapper.Code, wrapper.Message)
	}

	return &wrapper.Data, nil
}

func getAvailableState(currentState string) (map[string]bool, error) {
	currentState = strings.ToLower(currentState)
	switch currentState {
	case "ready_to_pick":
		return map[string]bool{
			"storing": true,
		}, nil
	case "storing":
		return map[string]bool{
			"delivering": true,
			"lost":       false,
			"return":     false, // seller request return
		}, nil
	case "delivering":
		return map[string]bool{
			"delivered":     true, //end
			"delivery_fail": false,
		}, nil
	case "delivery_fail":
		return map[string]bool{
			"storing": true, // loops
			"lost":    false,
		}, nil
	case "return":
		return map[string]bool{
			"returned": true, //end
		}, nil
	case "lost", "delivered", "returned":
		return map[string]bool{}, nil
	}

	return nil, fmt.Errorf("unknown current state: %s", currentState)
}
