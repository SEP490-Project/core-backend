package dtos

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"time"
)

type GHNWrapperResponse[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type GHNCreateOrderWrapperResponse[T any] struct {
	Code             int    `json:"code"`
	CodeMessageValue string `json:"code_message_value"`
	Data             T      `json:"data"`
	Message          string `json:"message"`
	MessageDisplay   string `json:"message_display"`
}

// ============================ DELIVERY FEE =============================

type DeliveryFeeBody struct {
	ToDistrictID   int               `json:"to_district_id"`
	ToWardCode     string            `json:"to_ward_code"`
	ServiceID      int               `json:"service_id"`
	ServiceTypeID  int               `json:"service_type_id"`
	InsuranceValue int               `json:"insurance_value"`
	Height         int               `json:"height"`
	Length         int               `json:"length"`
	Weight         int               `json:"weight"`
	Width          int               `json:"width"`
	Items          []DeliveryFeeItem `json:"items"`
}

type DeliveryFeeItem struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Height   int    `json:"height"`
	Weight   int    `json:"weight"`
	Length   int    `json:"length"`
	Width    int    `json:"width"`
}

type ApplicationDeliveryFeeItem struct {
	Name     string
	Quantity int
	Height   int
	Weight   int
	Length   int
	Width    int
	Price    int
}

type DeliveryFeeSuccess struct {
	Total                 int `json:"total"`
	ServiceFee            int `json:"service_fee"`
	InsuranceFee          int `json:"insurance_fee"`
	PickStationFee        int `json:"pick_station_fee"`
	CouponValue           int `json:"coupon_value"`
	R2SFee                int `json:"r2s_fee"`
	ReturnAgain           int `json:"return_again"`
	DocumentReturn        int `json:"document_return"`
	DoubleCheck           int `json:"double_check"`
	CodFee                int `json:"cod_fee"`
	PickRemoteAreasFee    int `json:"pick_remote_areas_fee"`
	DeliverRemoteAreasFee int `json:"deliver_remote_areas_fee"`
	CodFailedFee          int `json:"cod_failed_fee"`
}

// Mapper

func (a ApplicationDeliveryFeeItem) ToApplicationDeliveryFeeItem(item model.OrderItem) ApplicationDeliveryFeeItem {
	return ApplicationDeliveryFeeItem{
		Name:     item.Variant.Product.Name,
		Quantity: item.Quantity,
		Height:   item.Height,
		Weight:   item.Weight,
		Length:   item.Length,
		Width:    item.Width,
		Price:    int(item.UnitPrice),
	}
}

func (d DeliveryFeeBody) ToDeliveryFeeBodyDTOWithValidation(order *model.Order) (*DeliveryFeeBody, error) {
	var (
		totalWeight int
		totalHeight int
		maxLength   int
		maxWidth    int
	)

	for _, item := range order.OrderItems {
		// actualWeight
		totalWeight += item.Weight * item.Quantity
		// max of each dimension (height excluded)
		if item.Length > maxLength {
			maxLength = item.Length
		}
		if item.Width > maxWidth {
			maxWidth = item.Width
		}

		// stack up height
		totalHeight += item.Height * item.Quantity
	}

	maxDimension := max(maxLength, maxWidth, totalHeight)
	if maxDimension > 200 {
		return nil, fmt.Errorf("exceeded maximum dimension limit: %d cm (max 200 cm)", maxDimension)
	}

	chargeableWeight := (maxLength * maxWidth * totalHeight) / 3000

	finalWeight := max(totalWeight, chargeableWeight)
	if finalWeight > 50000 {
		return nil, fmt.Errorf("exceeded maximum weight limit: %d grams (max 500000 grams)", finalWeight)
	}

	return &DeliveryFeeBody{
		ToDistrictID:   order.GhnDistrictID,
		ToWardCode:     order.GhnWardCode,
		ServiceTypeID:  2,
		InsuranceValue: int(order.TotalAmount),
		Items:          DeliveryFeeItem{}.ToDeliveryFeeItemDTOList(order.OrderItems),
		Height:         totalHeight,
		Length:         maxLength,
		Width:          maxWidth,
		Weight:         finalWeight,
	}, nil
}

func (d DeliveryFeeBody) ToDeliveryFeeBodyDTOWithValidationV2(districtID int, wardCode string, items []ApplicationDeliveryFeeItem) (*DeliveryFeeBody, error) {
	var (
		totalWeight    int
		totalHeight    int
		maxLength      int
		maxWidth       int
		insuranceValue int
	)

	for _, item := range items {
		//calculate insurance value
		insuranceValue += item.Price * item.Quantity

		// actualWeight
		totalWeight += item.Weight * item.Quantity
		// max of each dimension (height excluded)
		if item.Length > maxLength {
			maxLength = item.Length
		}
		if item.Width > maxWidth {
			maxWidth = item.Width
		}

		// stack up height
		totalHeight += item.Height * item.Quantity
	}

	maxDimension := max(maxLength, maxWidth, totalHeight)
	if maxDimension > 200 {
		return nil, fmt.Errorf("exceeded maximum dimension limit: %d cm (max 200 cm)", maxDimension)
	}

	chargeableWeight := (maxLength * maxWidth * totalHeight) / 3000

	finalWeight := max(totalWeight, chargeableWeight)
	if finalWeight > 50000 {
		return nil, fmt.Errorf("exceeded maximum weight limit: %d grams (max 500000 grams)", finalWeight)
	}

	return &DeliveryFeeBody{
		ToDistrictID:   districtID,
		ToWardCode:     wardCode,
		InsuranceValue: insuranceValue,
		ServiceTypeID:  2,
		Items: func() []DeliveryFeeItem {
			var deliveryFeeItems []DeliveryFeeItem
			for _, item := range items {
				deliveryFeeItems = append(deliveryFeeItems, DeliveryFeeItem{
					Name:     item.Name,
					Quantity: item.Quantity,
					Height:   item.Height,
					Weight:   item.Weight,
					Length:   item.Length,
					Width:    item.Width,
				})
			}
			return deliveryFeeItems
		}(),

		Height: totalHeight,
		Length: maxLength,
		Width:  maxWidth,
		Weight: finalWeight,
	}, nil

}

func (d DeliveryFeeBody) ToDeliveryFeeBodyDTOWithValidationV3(address model.ShippingAddress, items []ApplicationDeliveryFeeItem) (*DeliveryFeeBody, error) {
	var (
		totalWeight    int
		totalHeight    int
		maxLength      int
		maxWidth       int
		insuranceValue int
	)

	for _, item := range items {
		//calculate insurance value
		insuranceValue += item.Price * item.Quantity
		// actualWeight
		totalWeight += item.Weight * item.Quantity
		// max of each dimension (height excluded)
		if item.Length > maxLength {
			maxLength = item.Length
		}
		if item.Width > maxWidth {
			maxWidth = item.Width
		}
		// stack up height
		totalHeight += item.Height * item.Quantity
	}

	maxDimension := max(maxLength, maxWidth, totalHeight)
	if maxDimension > 200 {
		return nil, fmt.Errorf("exceeded maximum dimension limit: %d cm (max 200 cm)", maxDimension)
	}

	chargeableWeight := (maxLength * maxWidth * totalHeight) / 3000

	finalWeight := max(totalWeight, chargeableWeight)
	if finalWeight > 50000 {
		return nil, fmt.Errorf("exceeded maximum weight limit: %d grams (max 500000 grams)", finalWeight)
	}

	return &DeliveryFeeBody{
		ToDistrictID:   address.GhnDistrictID,
		ToWardCode:     address.GhnWardCode,
		ServiceTypeID:  2,
		InsuranceValue: insuranceValue,
		Items: func() []DeliveryFeeItem {
			var deliveryFeeItems []DeliveryFeeItem
			for _, item := range items {
				deliveryFeeItems = append(deliveryFeeItems, DeliveryFeeItem{
					Name:     item.Name,
					Quantity: item.Quantity,
					Height:   item.Height,
					Weight:   item.Weight,
					Length:   item.Length,
					Width:    item.Width,
				})
			}
			return deliveryFeeItems
		}(),
		Height: totalHeight,
		Length: maxLength,
		Width:  maxWidth,
		Weight: finalWeight,
	}, nil
}

func (d DeliveryFeeItem) ToDeliveryFeeItemDTOList(orderItems []model.OrderItem) []DeliveryFeeItem {
	var deliveryFeeItems []DeliveryFeeItem
	for _, item := range orderItems {
		variantPropConcat := fmt.Sprintf("%d", item.Capacity) + utils.ToTitleCase(item.CapacityUnit.String()) + " - " + utils.ToTitleCase(item.ContainerType.String()) + " - " + utils.ToTitleCase(item.DispenserType.String())
		variantName := utils.ToTitleCase(item.Variant.Product.Name) + fmt.Sprintf(" (%s) ", variantPropConcat)

		deliveryFeeItems = append(deliveryFeeItems, DeliveryFeeItem{
			Name:     variantName,
			Quantity: item.Quantity,
			Height:   item.Height,
			Weight:   item.Weight,
			Length:   item.Length,
			Width:    item.Width,
		})
	}
	return deliveryFeeItems
}

// ============================ AVAILABLE DELIVERY SERVICES =============================

type DeliveryAvailableServiceDTO struct {
	ServiceID     int    `json:"service_id"`
	ShortName     string `json:"short_name"`
	ServiceTypeID int    `json:"service_type_id"`
}

type DeliveryAvailableServiceBody struct {
	ShopID       int `json:"shop_id"`
	FromDistrict int `json:"from_district"`
	ToDistrict   int `json:"to_district"`
}

// ============================ ORDER MANAGEMENT ============================================

type OrderInfo struct {
	ShopID           int    `json:"shop_id"`
	ClientID         int    `json:"client_id"`
	ReturnName       string `json:"return_name"`
	ReturnPhone      string `json:"return_phone"`
	ReturnAddress    string `json:"return_address"`
	ReturnWardCode   string `json:"return_ward_code"`
	ReturnDistrictID int    `json:"return_district_id"`
	ReturnLocation   struct {
		Lat        float64 `json:"lat"`
		Long       float64 `json:"long"`
		CellCode   string  `json:"cell_code"`
		PlaceID    string  `json:"place_id"`
		TrustLevel int     `json:"trust_level"`
		Wardcode   string  `json:"wardcode"`
		MapSource  string  `json:"map_source"`
	} `json:"return_location"`
	FromName       string `json:"from_name"`
	FromPhone      string `json:"from_phone"`
	FromHotline    string `json:"from_hotline"`
	FromAddress    string `json:"from_address"`
	FromWardCode   string `json:"from_ward_code"`
	FromDistrictID int    `json:"from_district_id"`
	FromLocation   struct {
		Lat        float64 `json:"lat"`
		Long       float64 `json:"long"`
		CellCode   string  `json:"cell_code"`
		PlaceID    string  `json:"place_id"`
		TrustLevel int     `json:"trust_level"`
		Wardcode   string  `json:"wardcode"`
		MapSource  string  `json:"map_source"`
	} `json:"from_location"`
	DeliverStationID int    `json:"deliver_station_id"`
	ToName           string `json:"to_name"`
	ToPhone          string `json:"to_phone"`
	ToAddress        string `json:"to_address"`
	ToWardCode       string `json:"to_ward_code"`
	ToDistrictID     int    `json:"to_district_id"`
	ToLocation       struct {
		Lat        float64 `json:"lat"`
		Long       float64 `json:"long"`
		CellCode   string  `json:"cell_code"`
		PlaceID    string  `json:"place_id"`
		TrustLevel int     `json:"trust_level"`
		Wardcode   string  `json:"wardcode"`
		MapSource  string  `json:"map_source"`
	} `json:"to_location"`
	Weight               int       `json:"weight"`
	Length               int       `json:"length"`
	Width                int       `json:"width"`
	Height               int       `json:"height"`
	ConvertedWeight      int       `json:"converted_weight"`
	CalculateWeight      int       `json:"calculate_weight"`
	ImageIds             any       `json:"image_ids"`
	ServiceTypeID        int       `json:"service_type_id"`
	ServiceID            int       `json:"service_id"`
	PaymentTypeID        int       `json:"payment_type_id"`
	PaymentTypeIds       []int     `json:"payment_type_ids"`
	CustomServiceFee     int       `json:"custom_service_fee"`
	SortCode             string    `json:"sort_code"`
	CodAmount            int       `json:"cod_amount"`
	CodCollectDate       any       `json:"cod_collect_date"`
	CodTransferDate      any       `json:"cod_transfer_date"`
	IsCodTransferred     bool      `json:"is_cod_transferred"`
	IsCodCollected       bool      `json:"is_cod_collected"`
	InsuranceValue       int       `json:"insurance_value"`
	OrderValue           int       `json:"order_value"`
	PickStationID        int       `json:"pick_station_id"`
	ClientOrderCode      string    `json:"client_order_code"`
	CodFailedAmount      int       `json:"cod_failed_amount"`
	CodFailedCollectDate any       `json:"cod_failed_collect_date"`
	RequiredNote         string    `json:"required_note"`
	Content              string    `json:"content"`
	Note                 string    `json:"note"`
	EmployeeNote         string    `json:"employee_note"`
	SealCode             string    `json:"seal_code"`
	PickupTime           time.Time `json:"pickup_time"`
	RequestDeliveryTime  any       `json:"request_delivery_time"`
	DeadlinePickupTime   any       `json:"deadline_pickup_time"`
	Items                []struct {
		Name     string `json:"name"`
		Quantity int    `json:"quantity"`
		Length   int    `json:"length"`
		Width    int    `json:"width"`
		Height   int    `json:"height"`
		Category struct {
		} `json:"category"`
		Weight             int    `json:"weight"`
		Status             string `json:"status"`
		ItemOrderCode      string `json:"item_order_code"`
		CurrentWarehouseID int    `json:"current_warehouse_id"`
	} `json:"items"`
	Coupon           string    `json:"coupon"`
	CouponCampaignID int       `json:"coupon_campaign_id"`
	ID               string    `json:"_id"`
	OrderCode        string    `json:"order_code"`
	VersionNo        string    `json:"version_no"`
	UpdatedIP        string    `json:"updated_ip"`
	UpdatedEmployee  int       `json:"updated_employee"`
	UpdatedClient    int       `json:"updated_client"`
	UpdatedSource    string    `json:"updated_source"`
	UpdatedDate      time.Time `json:"updated_date"`
	UpdatedWarehouse int       `json:"updated_warehouse"`
	CreatedIP        string    `json:"created_ip"`
	CreatedEmployee  int       `json:"created_employee"`
	CreatedClient    int       `json:"created_client"`
	CreatedSource    string    `json:"created_source"`
	CreatedDate      time.Time `json:"created_date"`
	Status           string    `json:"status"`
	InternalProcess  struct {
		Status string `json:"status"`
		Type   string `json:"type"`
	} `json:"internal_process"`
	PickWarehouseID             int       `json:"pick_warehouse_id"`
	DeliverWarehouseID          int       `json:"deliver_warehouse_id"`
	CurrentWarehouseID          int       `json:"current_warehouse_id"`
	ReturnWarehouseID           int       `json:"return_warehouse_id"`
	NextWarehouseID             int       `json:"next_warehouse_id"`
	CurrentTransportWarehouseID int       `json:"current_transport_warehouse_id"`
	Leadtime                    time.Time `json:"leadtime"`
	LeadtimeOrder               struct {
		FromEstimateDate time.Time `json:"from_estimate_date"`
		ToEstimateDate   time.Time `json:"to_estimate_date"`
	} `json:"leadtime_order"`
	OrderDate time.Time `json:"order_date"`
	Data      struct {
	} `json:"data"`
	SocID            string   `json:"soc_id"`
	FinishDate       any      `json:"finish_date"`
	Tag              []string `json:"tag"`
	IsPartialReturn  bool     `json:"is_partial_return"`
	IsDocumentReturn bool     `json:"is_document_return"`
	PickupShift      struct {
	} `json:"pickup_shift"`
	TransactionIds       []string `json:"transaction_ids"`
	TransportationStatus string   `json:"transportation_status"`
	TransportationPhase  string   `json:"transportation_phase"`
	ExtraService         struct {
		DocumentReturn struct {
			Flag bool `json:"flag"`
		} `json:"document_return"`
		DoubleCheck                bool   `json:"double_check"`
		LastmileAhamoveBulky       bool   `json:"lastmile_ahamove_bulky"`
		LastmileTripCode           string `json:"lastmile_trip_code"`
		OriginalDeliverWarehouseID int    `json:"original_deliver_warehouse_id"`
	} `json:"extra_service"`
	ConfigFeeID             string `json:"config_fee_id"`
	ExtraCostID             string `json:"extra_cost_id"`
	StandardConfigFeeID     string `json:"standard_config_fee_id"`
	StandardExtraCostID     string `json:"standard_extra_cost_id"`
	EcomConfigFeeID         int    `json:"ecom_config_fee_id"`
	EcomExtraCostID         int    `json:"ecom_extra_cost_id"`
	EcomStandardConfigFeeID int    `json:"ecom_standard_config_fee_id"`
	EcomStandardExtraCostID int    `json:"ecom_standard_extra_cost_id"`
	IsB2B                   bool   `json:"is_b2b"`
	OperationPartner        string `json:"operation_partner"`
	ProcessPartnerName      string `json:"process_partner_name"`
	DeliveryDaysOfWeek      int    `json:"delivery_days_of_week"`
	IsNewMultiple           bool   `json:"is_new_multiple"`
	FromAddressV2           string `json:"from_address_v2"`
	FromWardIDV2            int    `json:"from_ward_id_v2"`
	FromProvinceIDV2        int    `json:"from_province_id_v2"`
	IsNewFromAddress        bool   `json:"is_new_from_address"`
	ToAddressV2             string `json:"to_address_v2"`
	ToWardIDV2              int    `json:"to_ward_id_v2"`
	ToProvinceIDV2          int    `json:"to_province_id_v2"`
	IsNewToAddress          bool   `json:"is_new_to_address"`
	ReturnAddressV2         string `json:"return_address_v2"`
	ReturnWardIDV2          int    `json:"return_ward_id_v2"`
	ReturnProvinceIDV2      int    `json:"return_province_id_v2"`
	IsNewReturnAddress      bool   `json:"is_new_return_address"`
}

type CancelOrder struct {
	OrderCode string `json:"order_code"`
	Result    bool   `json:"result"`
	Message   string `json:"message"`
}

type CreateGHNOrderDTO struct {
	PaymentTypeID    int                          `json:"payment_type_id"`
	Note             string                       `json:"note"`
	RequiredNote     string                       `json:"required_note"`
	FromName         string                       `json:"from_name"`
	FromPhone        string                       `json:"from_phone"`
	FromAddress      string                       `json:"from_address"`
	FromWardName     string                       `json:"from_ward_name"`
	FromDistrictName string                       `json:"from_district_name"`
	FromProvinceName string                       `json:"from_province_name"`
	ClientOrderCode  string                       `json:"client_order_code"`
	ToName           string                       `json:"to_name"`
	ToPhone          string                       `json:"to_phone"`
	ToAddress        string                       `json:"to_address"`
	ToWardCode       string                       `json:"to_ward_code"`
	ToDistrictID     int                          `json:"to_district_id"`
	Content          string                       `json:"content"`
	Weight           int                          `json:"weight"`
	Length           int                          `json:"length"`
	Width            int                          `json:"width"`
	Height           int                          `json:"height"`
	InsuranceValue   int                          `json:"insurance_value"`
	ServiceID        int                          `json:"service_id"`
	ServiceTypeID    int                          `json:"service_type_id"`
	Items            []ApplicationDeliveryFeeItem `json:"items"`
}

type CreatedGHNOrderResponse struct {
	OrderCode      string `json:"order_code"`
	SortCode       string `json:"sort_code"`
	TransType      string `json:"trans_type"`
	WardEncode     string `json:"ward_encode"`
	DistrictEncode string `json:"district_encode"`
	Fee            struct {
		MainService                 int `json:"main_service"`
		Insurance                   int `json:"insurance"`
		CodFee                      int `json:"cod_fee"`
		StationDo                   int `json:"station_do"`
		StationPu                   int `json:"station_pu"`
		Return                      int `json:"return"`
		R2S                         int `json:"r2s"`
		ReturnAgain                 int `json:"return_again"`
		Coupon                      int `json:"coupon"`
		DocumentReturn              int `json:"document_return"`
		DoubleCheck                 int `json:"double_check"`
		DoubleCheckDeliver          int `json:"double_check_deliver"`
		PickRemoteAreasFee          int `json:"pick_remote_areas_fee"`
		DeliverRemoteAreasFee       int `json:"deliver_remote_areas_fee"`
		PickRemoteAreasFeeReturn    int `json:"pick_remote_areas_fee_return"`
		DeliverRemoteAreasFeeReturn int `json:"deliver_remote_areas_fee_return"`
		CodFailedFee                int `json:"cod_failed_fee"`
	} `json:"fee"`
	TotalFee             int       `json:"total_fee"`
	ExpectedDeliveryTime time.Time `json:"expected_delivery_time"`
	OperationPartner     string    `json:"operation_partner"`
}

type ExpectedDeliveryTime struct {
	Leadtime      int `json:"leadtime"`
	LeadtimeOrder struct {
		FromEstimateDate time.Time `json:"from_estimate_date"`
		ToEstimateDate   time.Time `json:"to_estimate_date"`
	} `json:"leadtime_order"`
}

// Mock DELIVERY STATUS

type GHNSessionResponse struct {
	TokenTemp         string `json:"token_temp"`
	Stage             string `json:"stage"`
	QrCode            string `json:"qr_code"`
	OtpTTL            int    `json:"otp_ttl"`
	OtpPhoneNumber    string `json:"otp_phone_number"`
	SsoToken          string `json:"sso_token"`
	SsoRefreshToken   string `json:"sso_refresh_token"`
	SsoTokenExpiresIn int    `json:"sso_token_expires_in"`
}

type GHNServiceToken struct {
	Code        string `json:"code"`
	CallbackURL string `json:"callback_url"`
}

type GHNTokenGSO struct {
	AccessToken string `json:"access_token"`
}

type UpdateGHNDeliveryStatusResponse struct {
	CurrentStatus string `json:"current_status"`
	Message       string `json:"message"`
	OrderCode     string `json:"order_code"`
	Result        bool   `json:"result"`
}
