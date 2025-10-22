package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

// ===========================ORDER==============================//
type OrderRequest struct {
	AddressID uuid.UUID          `json:"address_id" validate:"required,uuid4"`
	Items     []OrderItemRequest `json:"items" validate:"required,dive,required"`
}

func (or *OrderRequest) ToModel(userID uuid.UUID, orderItems []model.OrderItem, now time.Time) *model.Order {

	//Calc total amount:
	var totalAmount float64 = 0
	for _, item := range orderItems {
		totalAmount += item.Subtotal
	}

	return &model.Order{
		UserID:      userID,
		Status:      enum.OrderStatusPending,
		TotalAmount: totalAmount,
		AddressID:   or.AddressID,
		CreatedAt:   now,
		UpdatedAt:   now,
		OrderItems:  orderItems,
	}
}

// ===========================ORDER ITEM==============================//
type OrderItemRequest struct {
	VariantID string `json:"variant_id" validate:"required,uuid4"`
	Quantity  int    `json:"quantity" validate:"required,min=1"`
	//Subtotal  int    `json:"subtotal" validate:"required,min=1"`

	//Copy of ProductVariant fields
	ProductVariantID uuid.UUID `json:"product_variant_id" validate:"required,uuid4"`

	//UnitPrice             float64          `json:"unit_price" validate:"required,min=1"`
	//Capacity              *float64         `json:"capacity" validate:"omitempty"`
	//CapacityUnit          *string          `json:"capacity_unit" validate:"omitempty"`
	//ContainerType         *string          `json:"container_type" validate:"omitempty"`
	//DispenserType         *string          `json:"dispenser_type" validate:"omitempty"`
	//Uses                  *string          `json:"uses" validate:"omitempty"`
	//ManufactureDate       *string          `json:"manufacture_date" validate:"omitempty,datetime=2006-01-02"`
	//ExpiryDate            *string          `json:"expiry_date" validate:"omitempty,datetime=2006-01-02"`
	//Instructions          *string          `json:"instructions" validate:"omitempty"`
	//Attribute-Value -> parsed into this
	AttributesDescription *datatypes.JSON `json:"attributes_description" validate:"omitempty,json"`

	//ItemStatus enum.OrderStatus

}

// ToModel converts OrderItemRequest to OrderItem model. The purpose of "now" parameter is to set CreatedAt and UpdatedAt fields sync with Order.
func (oi *OrderItemRequest) ToModel(prdVariant model.ProductVariant, now time.Time) *model.OrderItem {
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
		AttributesDescription: nil,
		ItemStatus:            enum.OrderStatusPending,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}
