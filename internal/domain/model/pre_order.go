package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PreOrder struct {
	ID          uuid.UUID           `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	UserID      uuid.UUID           `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	VariantID   uuid.UUID           `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	Quantity    int                 `json:"quantity" gorm:"column:quantity;not null"`
	UnitPrice   float64             `json:"unit_price" gorm:"column:unit_price;not null"`
	TotalAmount float64             `json:"total_amount" gorm:"column:total_amount;not null"`
	Status      enum.PreOrderStatus `json:"status" gorm:"column:status;not null;check:status in ('PENDING', 'PRE_ORDERED', 'AWAITING_RELEASE', 'AWAITING_PICKUP', 'CONFIRMED', 'CANCELLED', 'IN_TRANSIT', 'DELIVERED', 'RECEIVED')"`
	CreatedAt   time.Time           `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time           `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt      `json:"deleted_at" gorm:"column:deleted_at"`

	// Relationships
	User           *User           `json:"-" gorm:"foreignKey:UserID"`
	ProductVariant *ProductVariant `json:"-" gorm:"foreignKey:VariantID"`
}

func (PreOrder) TableName() string { return "pre_order" }

func (po *PreOrder) BeforeCreate(tx any) (err error) {
	if po.ID == uuid.Nil {
		po.ID = uuid.New()
	}
	if po.Quantity < 0 {
		zap.L().Warn("Quantity is less than 0, setting to 0")
		po.Quantity = 0
	}
	if po.UnitPrice < 0 {
		zap.L().Warn("UnitPrice is less than 0, setting to 0")
		po.UnitPrice = 0
	}
	if po.TotalAmount < 0 {
		zap.L().Warn("TotalAmount is less than 0, setting to 0")
		po.TotalAmount = 0
	}

	return nil
}
