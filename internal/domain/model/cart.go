package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Cart struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	User     *User      `json:"-" gorm:"foreignKey:UserID"`
	CartItem []CartItem `json:"cart_items" gorm:"foreignKey:CartID"`
}

func (Cart) TableName() string { return "cart" }

func (c *Cart) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	return nil
}
