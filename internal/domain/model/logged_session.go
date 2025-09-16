// Package model contains the domain model for the users.
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoggedSession struct {
	ID                uuid.UUID `json:"id" gorm:"primaryKey"`
	UserID            uuid.UUID `json:"user_id" gorm:"not null;index"`
	RefreshTokenHash  string    `json:"-" gorm:"not null"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	CreatedAt         *time.Time `json:"created_at" gorm:"autoCreateTime"`
	ExpiryAt          *time.Time `json:"expiry_at"`
	IsRevoked         bool      `json:"is_revoked" gorm:"default:false"`
	LastUsedAt        *time.Time `json:"last_used_at" gorm:"autoUpdateTime"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ls *LoggedSession) BeforeCreate(tx *gorm.DB) error {
	if ls.ID == uuid.Nil {
		ls.ID = uuid.New()
	}
	return nil
}
