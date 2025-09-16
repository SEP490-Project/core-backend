// Package model contains the domain model for the users.
package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID     `json:"id" gorm:"primaryKey"`
	Username     string        `json:"username" gorm:"unique;not null"`
	Email        string        `json:"email" gorm:"unique;not null"`
	PasswordHash string        `json:"password_hash" gorm:"not null"`
	FullName     string        `json:"full_name"`
	Phone        string        `json:"phone"`
	Role         enum.UserRole `json:"role" gorm:"not null"`
	IsActive     bool          `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	LastLogin    time.Time     `json:"last_login" gorm:"autoUpdateTime"`
	ProfileData  string        `json:"profile_data" gorm:"type:jsonb"`

	// Relationships
	Sessions []LoggedSession `json:"sessions" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
