package dto

import (
	"core-backend/internal/domain/model"
	"time"
)

type ProductVariantResponse struct {
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
}

// Mappers
func ToProductVariantResponse(variant *model.ProductVariant) *ProductVariantResponse {
	return &ProductVariantResponse{
		ID:              variant.ID,
		ProductID:       variant.ProductID,
		Price:           variant.Price,
		CurrentStock:    variant.CurrentStock,
		Capacity:        variant.Capacity,
		CapacityUnit:    variant.CapacityUnit,
		ContainerType:   variant.ContainerType,
		DispenserType:   variant.DispenserType,
		Uses:            variant.Uses,
		ManufactureDate: variant.ManufactureDate,
		ExpiryDate:      variant.ExpiryDate,
		Instructions:    variant.Instructions,
		IsDefault:       variant.IsDefault,
	}
}
