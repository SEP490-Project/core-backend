package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Channel struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	Code        string    `json:"code" gorm:"type:varchar(100);column:code;not null;unique;index"`
	Name        string    `json:"name" gorm:"type:varchar(100);column:name;not null;unique;index"`
	Description *string   `json:"description" gorm:"type:text;column:description"`
	HomePageURL *string   `json:"home_page_url" gorm:"type:text;column:home_page_url"`
	IsActive    bool      `json:"is_active" gorm:"type:boolean;column:is_active;default:true;not null"`
	// OAuth fields
	ExternalID            *string    `json:"external_id" gorm:"type:varchar(255);column:external_id"`
	AccountName           *string    `json:"account_name" gorm:"type:varchar(255);column:account_name"`
	VaultPath             *string    `json:"vault_path" gorm:"type:text;column:vault_path"`
	HashedAccessToken     *string    `json:"-" gorm:"type:text;column:hashed_access_token"`
	HashedRefreshToken    *string    `json:"-" gorm:"type:text;column:hashed_refresh_token"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at" gorm:"column:access_token_expires_at"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at" gorm:"column:refresh_token_expires_at"`
	LastSyncedAt          *time.Time `json:"last_synced_at" gorm:"column:last_synced_at"`
	CreatedAt             time.Time  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt             *time.Time `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	ContentChannels []ContentChannel `json:"-" gorm:"foreignKey:ChannelID"`
}

func (Channel) TableName() string { return "channels" }

func (c *Channel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
