// Package model contains the domain model for the users.
package model

import (
	"core-backend/internal/domain/enum"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	ID                uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	Username          string         `json:"username" gorm:"type:varchar(255);column:username;unique;not null"`
	Email             string         `json:"email" gorm:"type:varchar(255);column:email;unique;not null"`
	PasswordHash      string         `json:"password_hash" gorm:"type:varchar(255);column:password_hash;not null"`
	FullName          string         `json:"full_name" gorm:"type:varchar(255);column:full_name;not null"`
	Phone             string         `json:"phone" gorm:"type:varchar(20);column:phone"`
	DateOfBirth       *time.Time     `json:"date_of_birth" gorm:"type:date;column:date_of_birth"`
	Role              enum.UserRole  `json:"role" gorm:"type:varchar(50);column:role;not null;check:role IN ('ADMIN', 'MARKETING_STAFF', 'CONTENT_STAFF', 'SALES_STAFF', 'CUSTOMER', 'BRAND_PARTNER')"`
	AvatarURL         *string        `json:"avatar_url" gorm:"type:text;column:avatar_url"`
	EmailEnabled      bool           `gorm:"default:true;not null" json:"email_enabled"`
	PushEnabled       bool           `gorm:"default:true;not null" json:"push_enabled"`
	IsActive          bool           `json:"is_active" gorm:"column:is_active;not null"`
	CreatedAt         *time.Time     `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         *time.Time     `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	LastLogin         *time.Time     `json:"last_login" gorm:"column:last_login"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index;auto"`
	ProfileData       datatypes.JSON `json:"profile_data" gorm:"type:jsonb"`
	IsFacebookOAuth   bool           `json:"is_facebook_oauth" gorm:"column:is_facebook_oauth;not null;default:false"`
	IsTikTokOAuth     bool           `json:"is_tiktok_oauth" gorm:"column:is_tiktok_oauth;not null;default:false"`
	OAuthMetadata     *OAuthMetadata `json:"oauth_metadata" gorm:"column:oauth_metadata;type:jsonb"`
	BankAccount       *string        `json:"bank_account" gorm:"column:bank_account"`
	BankName          *string        `json:"bank_name" gorm:"column:bank_name"`
	BankAccountHolder *string        `json:"bank_account_holder" gorm:"column:bank_account_holder"`

	// Relationships
	Sessions        []LoggedSession   `json:"sessions" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ShippingAddress []ShippingAddress `json:"shipping_addresses" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Brand           *Brand            `json:"brand" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	DeviceTokens    []DeviceToken     `json:"device_tokens" gorm:"foreignKey:UserID;corder_onstraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Notifications   []Notification    `json:"notifications" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.Role == "" {
		u.Role = enum.UserRoleCustomer
	}
	return nil
}

type OAuthMetadata struct {
	Facebook *FacebookOAuthMetadata `json:"facebook,omitempty" gorm:"type:jsonb"`
	TikTok   *TikTokOAuthMetadata   `json:"tiktok,omitempty" gorm:"type:jsonb"`
}

type FacebookOAuthMetadata struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture *struct {
		Data *struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
	Birthday  *string   `json:"birthday,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TikTokOAuthMetadata struct {
	User struct {
		OpenID          string  `json:"open_id"`                     // scope: user.info.basic
		UnionID         string  `json:"union_id"`                    // scope: user.info.basic
		AvatarURL       string  `json:"avatar_url"`                  // scope: user.info.basic
		DisplayName     string  `json:"display_name"`                // scope: user.info.basic
		BioDescription  *string `json:"bio_description,omitempty"`   // scope: user.info.profile
		ProfileDeepLink *string `json:"profile_deep_link,omitempty"` // scope: user.info.profile
		IsVerified      *bool   `json:"is_verified,omitempty"`       // scope: user.info.profile
		UserName        *string `json:"username,omitempty"`          // scope: user.info.profile
		FollowerCount   *int64  `json:"follower_count,omitempty"`    // scope: user.info.stats
		FollowingCount  *int64  `json:"following_count,omitempty"`   // scope: user.info.stats
		LikesCount      *int64  `json:"likes_count,omitempty"`       // scope: user.info.stats
		VideoCount      *int64  `json:"video_count,omitempty"`       // scope: user.info.stats
	} `json:"user"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (oauth OAuthMetadata) Value() (driver.Value, error) {
	return json.Marshal(oauth)
}

func (oauth *OAuthMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, oauth)
}
