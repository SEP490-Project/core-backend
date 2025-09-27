package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type VariantAttributeValue struct {
	ID          uuid.UUID          `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	VariantID   uuid.UUID          `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	AttributeID uuid.UUID          `json:"attribute_id" gorm:"type:uuid;column:attribute_id;not null"`
	Value       float64            `json:"value" gorm:"column:value;not null"`
	Unit        enum.AttributeUnit `json:"unit" gorm:"column:unit;not null;check:unit in ('%', 'MG', 'G', 'ML', 'L', 'IU', 'PPM', 'NONE')"`
	CreatedAt   time.Time          `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time          `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt     `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	Variant   *ProductVariant   `gorm:"foreignKey:VariantID"`
	Attribute *VariantAttribute `gorm:"foreignKey:AttributeID"`
}

func (VariantAttributeValue) TableName() string { return "variant_attribute_value" }

func (vav *VariantAttributeValue) BeforeCreate(tx *gorm.DB) error {
	if vav.ID == uuid.Nil {
		vav.ID = uuid.New()
	}
	if vav.Value < 0 {
		zap.L().Warn("VariantAttributeValue Value is less than 0, setting to 0")
		vav.Value = 0
	}

	return nil
}
