package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ===========================ORDER==============================//
type OrderRequest struct {
	AddressID uuid.UUID          `json:"address_id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	Items     []OrderItemRequest `json:"items" validate:"required,dive,required"`
	UserNote  string             `json:"user_note" validate:"omitempty,max=500" example:"Please deliver between 9 AM and 5 PM."`
}

func (or *OrderRequest) ToModel(userID uuid.UUID, orderItems []model.OrderItem, address model.ShippingAddress, shippingPrice int, now time.Time) *model.Order {

	//Calc total amount:
	var totalAmount float64 = 0
	for _, item := range orderItems {
		totalAmount += item.Subtotal
	}

	return &model.Order{
		Status:      enum.OrderStatusPending,
		TotalAmount: totalAmount,
		ShippingFee: shippingPrice,
		CreatedAt:   now,
		UpdatedAt:   now,

		//Copy shipping address fields
		FullName:      address.FullName,
		PhoneNumber:   address.PhoneNumber,
		Email:         address.Email,
		Street:        address.Street,
		AddressLine2:  address.AddressLine2,
		City:          address.City,
		GhnProvinceID: address.GhnProvinceID,
		GhnDistrictID: address.GhnDistrictID,
		GhnWardCode:   address.GhnWardCode,
		ProvinceName:  address.ProvinceName,
		DistrictName:  address.DistrictName,
		WardName:      address.WardName,
		UserNotes:     ptr.String(or.UserNote),

		//Order Relationships
		UserID:     userID,
		OrderItems: orderItems,
	}
}

// ===========================ORDER ITEM==============================//
type OrderItemRequest struct {
	VariantID uuid.UUID `json:"variant_id" validate:"required,uuid4" example:"69700831-4112-44fd-bf7f-07b015f56218"`
	Quantity  int       `json:"quantity" validate:"required,min=1" example:"1"`
}

// ToModel converts OrderItemRequest to OrderItem model. The purpose of "now" parameter is to set CreatedAt and UpdatedAt fields sync with Order.
func (oi *OrderItemRequest) ToModel(prdVariant model.ProductVariant, now time.Time) *model.OrderItem {

	// Build attributes description JSON
	var attrs []map[string]any
	for _, vav := range prdVariant.AttributeValues {
		m := map[string]any{
			"attribute_id": vav.AttributeID,
			"value":        vav.Value,
			"unit":         vav.Unit.String(),
		}
		if vav.Attribute != nil {
			m["ingredient"] = vav.Attribute.Ingredient
			m["description"] = vav.Attribute.Description
		}
		attrs = append(attrs, m)
	}

	var attrsJSON *datatypes.JSON
	if len(attrs) > 0 {
		b, err := json.Marshal(attrs)
		if err == nil {
			dj := datatypes.JSON(b)
			attrsJSON = &dj
		}
	}

	return &model.OrderItem{
		VariantID:             prdVariant.ID,
		Quantity:              oi.Quantity,
		Subtotal:              float64(oi.Quantity) * prdVariant.Price,
		UnitPrice:             prdVariant.Price,
		Capacity:              &prdVariant.Capacity,
		CapacityUnit:          ptr.String(prdVariant.CapacityUnit.String()),
		ContainerType:         &prdVariant.ContainerType,
		DispenserType:         &prdVariant.DispenserType,
		Uses:                  &prdVariant.Uses,
		ManufactureDate:       prdVariant.ManufactureDate,
		ExpiryDate:            prdVariant.ExpiryDate,
		Instructions:          &prdVariant.Instructions,
		AttributesDescription: attrsJSON,
		ItemStatus:            enum.OrderStatusPending,
		Weight:                prdVariant.Weight,
		Height:                prdVariant.Height,
		Length:                prdVariant.Length,
		Width:                 prdVariant.Width,
		//CreatedAt: now,
		UpdatedAt: now,
	}
}

// ===========================PAYMENT==============================// (BY GHN)

// PlaceAndPayRequest wraps OrderRequest with an optional delivery service selection
// used by the place-and-pay endpoint.
type PlaceAndPayRequest struct {
	Order      OrderRequest `json:"order" validate:"required,dive"`
	CancelURL  string       `json:"cancel_url" validate:"omitempty,url" example:"https://example.com/cancel"`
	SuccessURL string       `json:"success_url" validate:"omitempty,url" example:"https://example.com/success"`
}
