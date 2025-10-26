package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type CreateProductDTO struct {
	BrandID     uuid.UUID        `json:"brand_id" validate:"required,uuid"`
	CategoryID  uuid.UUID        `json:"category_id" validate:"required,uuid"`
	TaskID      *uuid.UUID       `json:"task_id" validate:"omitempty,uuid"`
	Name        string           `json:"name" validate:"required,min=3,max=255"`
	Description *string          `json:"description" validate:"omitempty,max=2000"`
	Price       float64          `json:"price" validate:"required,gte=0"`
	Type        enum.ProductType `json:"type" validate:"required,oneof=STANDARD LIMITED"`
}

// CreateProductVariantDTO carries variant data into the service layer.
type CreateProductVariantDTO struct {
	Price           float64            `json:"price" validate:"required,gte=0"`
	CurrentStock    int                `json:"current_stock" validate:"required,gte=0"`
	Capacity        float64            `json:"capacity" validate:"omitempty,gte=0"`
	CapacityUnit    enum.CapacityUnit  `json:"capacity_unit" validate:"required,oneof=ML L G KG OZ"`
	ContainerType   enum.ContainerType `json:"container_type" validate:"required,oneof=BOTTLE TUBE JAR STICK PENCIL COMPACT PALLETE SACHET VIAL ROLLER_BOTTLE"`
	DispenserType   enum.DispenserType `json:"dispenser_type" validate:"required,oneof=PUMP SPRAY DROPPER ROLL_ON TWIST_UP SQUEEZE NONE"`
	Uses            string             `json:"uses" validate:"required,max=5000"`
	ManufactureDate *string            `json:"manufacturing_date" validate:"omitempty,datetime=2006-01-02"`
	ExpiryDate      *string            `json:"expiry_date" validate:"omitempty,datetime=2006-01-02"`
	Instructions    string             `json:"instructions" validate:"omitempty,max=5000"`
	IsDefault       bool               `json:"is_default" gorm:"column:is_default;not null;default:false"`
}

// ToModel maps the DTO to a Product domain model.
func (d *CreateProductDTO) ToModel(createdBy uuid.UUID) *model.Product {
	if d == nil {
		return nil
	}
	return &model.Product{
		BrandID:     d.BrandID,
		CategoryID:  d.CategoryID,
		TaskID:      d.TaskID,
		Name:        d.Name,
		Description: d.Description,
		Type:        d.Type,
		CreatedByID: createdBy,
	}
}
