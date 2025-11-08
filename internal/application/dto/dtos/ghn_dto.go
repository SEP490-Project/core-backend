package dtos

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
)

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
		return nil, errors.New(fmt.Sprintf("exceeded maximum dimension limit: %d cm (max 200 cm)", maxDimension))
	}

	chargeableWeight := (maxLength * maxWidth * totalHeight) / 3000

	finalWeight := max(totalWeight, chargeableWeight)
	if finalWeight > 50000 {
		return nil, errors.New(fmt.Sprintf("exceeded maximum weight limit: %d grams (max 500000 grams)", finalWeight))
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
		return nil, errors.New(fmt.Sprintf("exceeded maximum dimension limit: %d cm (max 200 cm)", maxDimension))
	}

	chargeableWeight := (maxLength * maxWidth * totalHeight) / 3000

	finalWeight := max(totalWeight, chargeableWeight)
	if finalWeight > 50000 {
		return nil, errors.New(fmt.Sprintf("exceeded maximum weight limit: %d grams (max 500000 grams)", finalWeight))
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
		return nil, errors.New(fmt.Sprintf("exceeded maximum dimension limit: %d cm (max 200 cm)", maxDimension))
	}

	chargeableWeight := (maxLength * maxWidth * totalHeight) / 3000

	finalWeight := max(totalWeight, chargeableWeight)
	if finalWeight > 50000 {
		return nil, errors.New(fmt.Sprintf("exceeded maximum weight limit: %d grams (max 500000 grams)", finalWeight))
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
		variantPropConcat := fmt.Sprintf("%d", item.Capacity) + utils.ToTitleCase(*item.CapacityUnit) + " - " + utils.ToTitleCase(item.ContainerType.String()) + " - " + utils.ToTitleCase(item.DispenserType.String())
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
