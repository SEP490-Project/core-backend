package model

import (
	"time"

	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceToken represents a mobile device FCM token for push notifications
type DeviceToken struct {
	ID           uuid.UUID         `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID         `gorm:"type:uuid;not null;index"`
	Token        string            `gorm:"type:varchar(255);not null;uniqueIndex"`
	Platform     enum.PlatformType `gorm:"type:varchar(50);not null"`
	RegisteredAt *time.Time        `gorm:"not null;autoCreateTime"`
	LastUsedAt   *time.Time        `gorm:""`
	IsValid      bool              `gorm:"not null;default:true;index"`
	CreatedAt    *time.Time        `gorm:"autoCreateTime"`
	UpdatedAt    *time.Time        `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt    `gorm:"index"`

	// Relationships
	User *User `gorm:"foreignKey:UserID"`
}

// BeforeCreate hook to generate UUID
func (dt *DeviceToken) BeforeCreate(tx *gorm.DB) error {
	if dt.ID == uuid.Nil {
		dt.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (DeviceToken) TableName() string {
	return "device_tokens"
}
