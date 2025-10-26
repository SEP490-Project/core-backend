// Package model contains the domain model for the users.
package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	Username     string         `json:"username" gorm:"type:varchar(255);column:username;unique;not null"`
	Email        string         `json:"email" gorm:"type:varchar(255);column:email;unique;not null"`
	PasswordHash string         `json:"password_hash" gorm:"type:varchar(255);column:password_hash;not null"`
	FullName     string         `json:"full_name" gorm:"type:varchar(255);column:full_name;not null"`
	Phone        string         `json:"phone" gorm:"type:varchar(20);column:phone"`
	DateOfBirth  *time.Time     `json:"date_of_birth" gorm:"type:date;column:date_of_birth"`
	Role         enum.UserRole  `json:"role" gorm:"type:varchar(50);column:role;not null;check:role IN ('ADMIN', 'MARKETING_STAFF', 'CONTENT_STAFF', 'SALES_STAFF', 'CUSTOMER', 'BRAND_PARTNER')"`
	AvatarURL    *string        `json:"avatar_url" gorm:"type:text;column:avatar_url"`
	EmailEnabled bool           `gorm:"default:true;not null" json:"email_enabled"`
	PushEnabled  bool           `gorm:"default:true;not null" json:"push_enabled"`
	IsActive     bool           `json:"is_active" gorm:"column:is_active;not null"`
	CreatedAt    *time.Time     `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    *time.Time     `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	LastLogin    *time.Time     `json:"last_login" gorm:"column:last_login"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index;auto"`
	ProfileData  datatypes.JSON `json:"profile_data" gorm:"type:jsonb"`

	// Relationships
	Sessions        []LoggedSession   `json:"sessions" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ShippingAddress []ShippingAddress `json:"shipping_addresses" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Brand           *Brand            `json:"brand" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	DeviceTokens    []DeviceToken     `json:"device_tokens" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Notifications   []Notification    `json:"notifications" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
