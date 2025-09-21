package model

import (
	"time"
)

type Product struct {
	ID           int              `gorm:"primaryKey;column:id"`
	BrandID      int              `gorm:"column:brand_id;not null"`
	Name         string           `gorm:"column:name;not null"`
	Description  string           `gorm:"column:description;not null"`
	Price        float64          `gorm:"column:price;not null"`
	CurrentStock int              `gorm:"column:current_stock;not null"`
	Type         string           `gorm:"column:type;not null"`
	CreatedAt    time.Time        `gorm:"column:created_at"`
	UpdatedAt    time.Time        `gorm:"column:updated_at"`
	IsDeleted    bool             `gorm:"column:is_deleted"`
	Variants     []ProductVariant `gorm:"foreignKey:ProductID"`
}

func (Product) TableName() string { return "product" }
