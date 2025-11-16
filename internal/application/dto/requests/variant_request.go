package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"github.com/aws/smithy-go/ptr"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BulkVariantRequest represent a compound, multi model createtion
// include: CreateProductVariantRequest, CreateProductStoryRequest, [] CreateVariantImagesRequest
type BulkVariantRequest struct {
	CreateProductVariantRequest
	Story      *CreateProductStoryRequest           `json:"story" validate:"omitempty"`
	Attributes []CreateVariantAttributeValueRequest `json:"attributes" validate:"dive"`
}

// CreateProductVariantRequest represents a variant payload when creating a product
// Date fields use RFC3339 strings; service will parse to time.Time
type CreateProductVariantRequest struct {
	Price           float64            `json:"price" form:"price" validate:"required,min=1000" example:"1000"`
	InputedStock    *int               `json:"input_stock" form:"input_stock" validate:"omitempty" example:"100"`
	Capacity        float64            `json:"capacity" form:"capacity" validate:"omitempty,min=0" example:"500"`
	CapacityUnit    enum.CapacityUnit  `json:"capacity_unit" form:"capacity_unit" validate:"required,oneof=ML L G KG OZ"`
	ContainerType   enum.ContainerType `json:"container_type" form:"container_type" validate:"required,oneof=BOTTLE TUBE JAR STICK PENCIL COMPACT PALLETE SACHET VIAL ROLLER_BOTTLE" example:"BOTTLE"`
	DispenserType   enum.DispenserType `json:"dispenser_type" form:"dispenser_type" validate:"required,oneof=PUMP SPRAY DROPPER ROLL_ON TWIST_UP SQUEEZE NONE" example:"SPRAY"`
	Uses            string             `json:"uses" form:"uses" example:"For daily use"`
	ManufactureDate *string            `json:"manufacturing_date" form:"manufacturing_date" example:"2023-10-01T00:00:00Z"`
	ExpiryDate      *string            `json:"expiry_date" form:"expiry_date" example:"2025-10-01T00:00:00Z"`
	Instructions    string             `json:"instructions" form:"instructions" example:"Shake well before use"`
	Weight          int                `json:"weight" form:"weight" validate:"min=0" example:"250"` // in grams
	Height          int                `json:"height" form:"height" validate:"min=0" example:"15"`  // in centimeters
	Length          int                `json:"length" form:"length" validate:"min=0" example:"10"`  // in centimeters
	Width           int                `json:"width" form:"width" validate:"min=0" example:"5"`     //
	IsDefault       bool               `json:"is_default" form:"is_default" example:"true"`
	PreOrderLimit   *int               `json:"pre_order_limit" form:"pre_order_limit" validate:"omitempty" example:"0"`
	PreOrderCount   *int               `json:"pre_order_count" form:"pre_order_count" validate:"omitempty" example:"0"`
}

func (e *CreateProductVariantRequest) ToModel(productID uuid.UUID, createdBy uuid.UUID, productType enum.ProductType) *model.ProductVariant {
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

	resp := &model.ProductVariant{
		ID:              uuid.UUID{},
		ProductID:       productID,
		Price:           e.Price,
		CurrentStock:    e.InputedStock,
		MaxStock:        e.InputedStock,
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
		Weight:          e.Weight,
		Height:          e.Height,
		Length:          e.Length,
		Width:           e.Width,
		PreOrderLimit:   e.PreOrderLimit,
		PreOrderCount:   ptr.Int(0),
		Product:         nil,
		Story:           nil,
		AttributeValues: nil,
		Images:          nil,
	}

	if productType == enum.ProductTypeStandard {
		resp.MaxStock = nil
		resp.CurrentStock = nil
		resp.PreOrderLimit = nil
		resp.PreOrderCount = nil
		resp.CurrentStock = nil
	}

	return resp
}
