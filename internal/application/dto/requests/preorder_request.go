package requests

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"time"

	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ===========================PREORDER==============================//

type PreOrderRequest struct {
	AddressID    uuid.UUID `json:"address_id" validate:"required,uuid4" example:"3fa85f64-5717-4562-b3fc-2c963f66afa6"`
	VariantID    uuid.UUID `json:"variant_id" validate:"required,uuid4" example:"69700831-4112-44fd-bf7f-07b015f56218"`
	CancelURL    string    `json:"cancel_url" validate:"omitempty,url" example:"https://example.com/cancel"`
	SuccessURL   string    `json:"success_url" validate:"omitempty,url" example:"https://example.com/success"`
	IsSelfPickup bool      `json:"is_self_pickup" validate:"bool" example:"false"`
	UserNote     *string   `json:"user_note,omitempty" validate:"omitempty" example:"Please deliver between 9 AM and 5 PM."`
}

func (p PreOrderRequest) ToModel(user model.User, address model.ShippingAddress, variant model.ProductVariant, now time.Time) *model.PreOrder {

	// Build attributes description JSON
	var attrs []map[string]any
	for _, vav := range variant.AttributeValues {
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

	return &model.PreOrder{
		UserID:                address.UserID,
		VariantID:             variant.ID,
		Quantity:              1,
		UnitPrice:             variant.Price,
		TotalAmount:           variant.Price,
		FullName:              address.FullName,
		PhoneNumber:           address.PhoneNumber,
		Email:                 address.Email,
		Street:                address.Street,
		AddressLine2:          address.AddressLine2,
		City:                  address.City,
		GhnProvinceID:         address.GhnProvinceID,
		GhnDistrictID:         address.GhnDistrictID,
		GhnWardCode:           address.GhnWardCode,
		ProvinceName:          address.ProvinceName,
		DistrictName:          address.DistrictName,
		WardName:              address.WardName,
		Capacity:              &variant.Capacity,
		CapacityUnit:          ptr.String(variant.CapacityUnit.String()),
		ContainerType:         &variant.ContainerType,
		DispenserType:         &variant.DispenserType,
		Uses:                  &variant.Uses,
		ManufactureDate:       variant.ManufactureDate,
		ExpiryDate:            variant.ExpiryDate,
		Instructions:          &variant.Instructions,
		AttributesDescription: attrsJSON,
		Weight:                variant.Weight,
		Height:                variant.Height,
		Length:                variant.Length,
		Width:                 variant.Width,
		Status:                enum.PreOrderStatusPending,
		CreatedAt:             now,
		UpdatedAt:             now,
		IsSelfPickedUp:        p.IsSelfPickup,
		UserNote:              p.UserNote,

		BankAccount:       *user.BankAccount,
		BankName:          *user.BankName,
		BankAccountHolder: *user.BankAccountHolder,

		// product info
		ProductName: variant.Product.Name,
		Description: variant.Product.Description,
		Type:        variant.Product.Type.String(),
		BrandID:     variant.Product.BrandID,
		CategoryID:  variant.Product.CategoryID,
	}
}

// ===========================PAYMENT==============================//

type PlaceAndPayPreOrderRequest struct {
	DeliveryService *dtos.DeliveryAvailableServiceDTO `json:"delivery_service,omitempty"`
	PreOrder        PreOrderRequest                   `json:"pre_order" validate:"required,dive"`
	CancelURL       string                            `json:"cancel_url" validate:"omitempty,url" example:"https://example.com/cancel"`
	SuccessURL      string                            `json:"success_url" validate:"omitempty,url" example:"https://example.com/success"`
}
