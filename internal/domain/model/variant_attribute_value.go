package model

import (
	"time"
)

type VariantAttributeValue struct {
	ID          int       `gorm:"primaryKey;column:id"`
	VariantID   int       `gorm:"column:variant_id;not null"`
	AttributeID int       `gorm:"column:attribute_id;not null"`
	Value       float64   `gorm:"column:value;not null"`
	Unit        string    `gorm:"column:unit"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
	IsDeleted   bool      `gorm:"column:is_deleted"`
}

func (VariantAttributeValue) TableName() string { return "variant_attribute_value" }
