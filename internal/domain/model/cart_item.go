package model

import (
	"time"
)

type CartItem struct {
	ID        int       `gorm:"primaryKey;column:id"`
	CartID    int       `gorm:"column:cart_id;not null"`
	VariantID int       `gorm:"column:variant_id;not null"`
	Quantity  int       `gorm:"column:quantity;not null"`
	Subtotal  float64   `gorm:"column:subtotal;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (CartItem) TableName() string { return "cart_item" }
