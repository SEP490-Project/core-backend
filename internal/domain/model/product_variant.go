package model

import (
	"time"
)

type ProductVariant struct {
	ID              int        `gorm:"primaryKey;column:id"`
	ProductID       int        `gorm:"column:product_id;not null"`
	Price           float64    `gorm:"column:price"`
	CurrentStock    int        `gorm:"column:current_stock;not null"`
	Capacity        float64    `gorm:"column:capacity"`
	CapacityUnit    string     `gorm:"column:capacity_unit"`
	ContainerType   string     `gorm:"column:container_type"`
	DispenserType   string     `gorm:"column:dispenser_type"`
	Uses            string     `gorm:"column:uses"`
	ManufactureDate *time.Time `gorm:"column:manufactring_date"`
	ExpiryDate      *time.Time `gorm:"column:expiry_date"`
	Instructions    string     `gorm:"column:instructions"`
	IsDefault       bool       `gorm:"column:is_default"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`
	IsDeleted       bool       `gorm:"column:is_deleted"`

	//Relationship
	Product Product `gorm:"foreignKey:ProductID"`
}

func (ProductVariant) TableName() string { return "product_variant" }
