// Package model contains the domain model for the users.
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LoggedSession struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	UserID            uuid.UUID      `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	RefreshTokenHash  string         `json:"-" gorm:"type:text;column:refresh_token_hash"`
	DeviceFingerprint string         `json:"device_fingerprint" gorm:"type:text;column:device_fingerprint"`
	ExpiryAt          *time.Time     `json:"expiry_at"`
	IsRevoked         bool           `json:"is_revoked" gorm:"default:false"`
	LastUsedAt        *time.Time     `json:"last_used_at" gorm:"autoUpdateTime"`
	CreatedAt         *time.Time     `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         *time.Time     `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	User User `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (LoggedSession) TableName() string { return "logged_sessions" }

func (ls *LoggedSession) BeforeCreate(tx *gorm.DB) error {
	if ls.ID == uuid.Nil {
		ls.ID = uuid.New()
	}
	return nil
}
