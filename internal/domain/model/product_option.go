package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductOptionType represents the type of product option
type ProductOptionType string

const (
	ProductOptionTypeCapacityUnit  ProductOptionType = "CAPACITY_UNIT"
	ProductOptionTypeContainerType ProductOptionType = "CONTAINER_TYPE"
	ProductOptionTypeDispenserType ProductOptionType = "DISPENSER_TYPE"
	ProductOptionTypeAttributeUnit ProductOptionType = "ATTRIBUTE_UNIT"
)

// IsValid checks if the ProductOptionType is valid
func (t ProductOptionType) IsValid() bool {
	switch t {
	case ProductOptionTypeCapacityUnit, ProductOptionTypeContainerType,
		ProductOptionTypeDispenserType, ProductOptionTypeAttributeUnit:
		return true
	}
	return false
}

// ProductOption represents a configurable option value for products
// This replaces hardcoded enums like CapacityUnit, ContainerType, DispenserType, AttributeUnit
type ProductOption struct {
	ID          uuid.UUID         `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Type        ProductOptionType `json:"type" gorm:"type:varchar(50);not null"`
	Code        string            `json:"code" gorm:"type:varchar(50);not null"`
	Name        string            `json:"name" gorm:"type:varchar(100);not null"`
	Description *string           `json:"description" gorm:"type:text"`
	SortOrder   int               `json:"sort_order" gorm:"default:0"`
	IsActive    bool              `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt    `json:"deleted_at" gorm:"index"`
}

// TableName specifies the table name for GORM
func (ProductOption) TableName() string {
	return "product_options"
}

// BeforeCreate hook to generate UUID if not set
func (p *ProductOption) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
