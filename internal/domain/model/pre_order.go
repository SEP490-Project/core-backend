package model

import (
	"time"
)

type PreOrder struct {
	ID          int       `gorm:"primaryKey;column:id"`
	UserID      int       `gorm:"column:user_id;not null"`
	VariantID   int       `gorm:"column:variant_id;not null"`
	Quantity    int       `gorm:"column:quantity;not null"`
	UnitPrice   float64   `gorm:"column:unit_price;not null"`
	TotalAmount float64   `gorm:"column:total_amount;not null"`
	Status      string    `gorm:"column:status;not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (PreOrder) TableName() string { return "pre_order" }
