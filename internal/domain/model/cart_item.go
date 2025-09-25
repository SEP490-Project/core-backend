package model

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CartItem struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	CartID    uuid.UUID `json:"cart_id" gorm:"type:uuid;column:cart_id;not null"`
	VariantID uuid.UUID `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	Quantity  int       `json:"quantity" gorm:"type:int;column:quantity;not null"`
	Subtotal  float64   `json:"subtotal" gorm:"type:numeric(12,2);column:subtotal;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Cart           *Cart           `json:"-" gorm:"foreignKey:CartID"`
	ProductVariant *ProductVariant `json:"-" gorm:"foreignKey:VariantID"`
}

func (CartItem) TableName() string { return "cart_item" }

func (ct *CartItem) BeforeCreate(tx *gorm.DB) error {
	if ct.ID == uuid.Nil {
		ct.ID = uuid.New()
	}
	if ct.Quantity < 1 {
		zap.L().Warn("Quantity is less than 1, setting to 1")
		ct.Quantity = 1
	}
	if ct.Subtotal < 0 {
		zap.L().Warn("Subtotal is less than 0, setting to 0")
		ct.Subtotal = 0
	}

	return nil
}
