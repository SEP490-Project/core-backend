package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BulkVariantRequest represent a compound, multi model createtion
// include: CreateProductVariantRequest, CreateProductStoryRequest, [] CreateVariantImagesRequest
//
//	@example	{
//	  "price": 29.99,
//	  "current_stock": 100,
//	  "capacity": 500,
//	  "capacity_unit": "ML",
//	  "container_type": "BOTTLE",
//	  "dispenser_type": "SPRAY",
//	  "uses": "For daily use",
//	  "manufacturing_date": "2023-10-01T00:00:00Z",
//	  "expiry_date": "2025-10-01T00:00:00Z",
//	  "instructions": "Shake well before use",
//	  "is_default": true,
//	  "story": {
//	    "variant_id": "550e8400-e29b-41d4-a716-446655440000",
//	    "content": {"description": "This is a sample story", "details": "More details here"}
//	  },
//	  "attributes": [
//	    {
//	      "variant_id": "550e8400-e29b-41d4-a716-446655440000",
//	      "attribute_id": "550e8400-e29b-41d4-a716-446655440001",
//	      "value": 10.5,
//	      "unit": "MG"
//	    }
//	  ]
//	}
type BulkVariantRequest struct {
	CreateProductVariantRequest
	Story      *CreateProductStoryRequest           `json:"story" validate:"omitempty" example:"{\"variant_id\":\"550e8400-e29b-41d4-a716-446655440000\",\"content\":{\"description\":\"This is a sample story\"}}"`
	Attributes []CreateVariantAttributeValueRequest `json:"attributes" validate:"dive" example:"[{\"variant_id\":\"550e8400-e29b-41d4-a716-446655440000\",\"attribute_id\":\"550e8400-e29b-41d4-a716-446655440001\",\"value\":10.5,\"unit\":\"MG\"}]"`
	//Images []CreateVariantImagesRequest
}

// CreateProductVariantRequest represents a variant payload when creating a product
// Date fields use RFC3339 strings; service will parse to time.Time
type CreateProductVariantRequest struct {
	Price           float64            `json:"price" form:"price" validate:"required,min=1000" example:"1000"`
	CurrentStock    *int               `json:"current_stock" form:"current_stock" validate:"omitempty" example:"100"`
	Capacity        float64            `json:"capacity" form:"capacity" validate:"omitempty,min=0" example:"500"`
	CapacityUnit    enum.CapacityUnit  `json:"capacity_unit" form:"capacity_unit" validate:"required,oneof=ML L GALLON"` // add valid units
	ContainerType   enum.ContainerType `json:"container_type" form:"container_type" validate:"required,oneof=BOTTLE BOX CAN"`
	DispenserType   enum.DispenserType `json:"dispenser_type" form:"dispenser_type" validate:"required,oneof=SPRAY PUMP"`
	Uses            string             `json:"uses" form:"uses" example:"For daily use"`
	ManufactureDate *string            `json:"manufacturing_date" form:"manufacturing_date" example:"2023-10-01T00:00:00Z"`
	ExpiryDate      *string            `json:"expiry_date" form:"expiry_date" example:"2025-10-01T00:00:00Z"`
	Instructions    string             `json:"instructions" form:"instructions" example:"Shake well before use"`
	IsDefault       bool               `json:"is_default" form:"is_default" example:"true"`
}

func (e *CreateProductVariantRequest) ToModel(productID uuid.UUID, createdBy uuid.UUID) *model.ProductVariant {
	if e == nil {
		return nil
	}

	// Parse ngày sản xuất và ngày hết hạn nếu có
	var manufactureDatePtr *time.Time
	if e.ManufactureDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *e.ManufactureDate); err == nil {
			manufactureDatePtr = &parsed
		}
	}

	var expiryDatePtr *time.Time
	if e.ExpiryDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *e.ExpiryDate); err == nil {
			expiryDatePtr = &parsed
		}
	}

	now := time.Now().UTC()

	return &model.ProductVariant{
		ProductID:       productID,
		Price:           e.Price,
		CurrentStock:    e.CurrentStock,
		Capacity:        e.Capacity,
		CapacityUnit:    e.CapacityUnit,
		ContainerType:   e.ContainerType,
		DispenserType:   e.DispenserType,
		Uses:            e.Uses,
		ManufactureDate: manufactureDatePtr,
		ExpiryDate:      expiryDatePtr,
		Instructions:    e.Instructions,
		IsDefault:       e.IsDefault,
		CreatedAt:       now,
		UpdatedAt:       now,
		DeletedAt:       gorm.DeletedAt{},
		CreatedByID:     createdBy,
		UpdatedByID:     nil,
		Story:           nil,
		AttributeValues: nil,
	}
}
