package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Order struct {
	ID          uuid.UUID        `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	UserID      uuid.UUID        `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	Status      enum.OrderStatus `json:"status" gorm:"column:status;not null;check:status in ('PENDING', 'PAID', 'REFUNDED', 'CONFIRMED', 'CANCELED', 'SHIPPED', 'IN_TRANSIT', 'DELIVERED', 'RECEIVED')"`
	TotalAmount float64          `json:"total_amount" gorm:"column:total_amount;not null"`
	AddressID   uuid.UUID        `json:"address_id" gorm:"type:uuid;column:address_id;not null"`
	CreatedAt   time.Time        `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time        `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	//DeletedAt   gorm.DeletedAt   `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	User       User            `json:"-" gorm:"foreignKey:UserID"`
	Address    ShippingAddress `json:"-" gorm:"foreignKey:AddressID"`
	OrderItems []OrderItem     `json:"order_items" gorm:"foreignKey:OrderID"` //
}

func (Order) TableName() string { return "orders" }

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.TotalAmount < 0 {
		zap.L().Warn("TotalAmount is less than 0, setting to 0")
		o.TotalAmount = 0
	}

	return nil
}
