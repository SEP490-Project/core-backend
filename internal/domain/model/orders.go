package model

import (
	"time"
)

type Orders struct {
	ID          int       `gorm:"primaryKey;column:id"`
	UserID      int       `gorm:"column:user_id;not null"`
	Status      string    `gorm:"column:status;not null"`
	TotalAmount float64   `gorm:"column:total_amount;not null"`
	AddressID   int       `gorm:"column:address_id;not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (Orders) TableName() string { return "orders" }
