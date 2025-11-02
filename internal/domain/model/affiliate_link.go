package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AffiliateLink represents a unique trackable affiliate link for content+channel combinations
type AffiliateLink struct {
	ID          uuid.UUID                `json:"id" gorm:"type:uuid;primaryKey"`
	Hash        string                   `json:"hash" gorm:"type:varchar(16);uniqueIndex;not null"`                  // Base62 SHA-256 truncated (16 chars)
	ContractID  uuid.UUID                `json:"contract_id" gorm:"type:uuid;not null;index:idx_affiliate_contract"` // Reference to contract
	ContentID   uuid.UUID                `json:"content_id" gorm:"type:uuid;not null;index:idx_affiliate_content"`   // Reference to content
	ChannelID   uuid.UUID                `json:"channel_id" gorm:"type:uuid;not null;index:idx_affiliate_channel"`   // Reference to channel
	TrackingURL string                   `json:"tracking_url" gorm:"type:text;not null"`                             // Original URL from contract
	Status      enum.AffiliateLinkStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`           // active, inactive, expired
	CreatedAt   *time.Time               `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time               `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt           `json:"deleted_at,omitempty" gorm:"index"`

	// Relationships (use pointers to avoid circular dependencies)
	Contract *Contract `json:"contract,omitempty" gorm:"foreignKey:ContractID"`
	Content  *Content  `json:"content,omitempty" gorm:"foreignKey:ContentID"`
	Channel  *Channel  `json:"channel,omitempty" gorm:"foreignKey:ChannelID"`
}

func (AffiliateLink) TableName() string {
	return "affiliate_links"
}

// BeforeCreate generates UUID if not set
func (a *AffiliateLink) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// IsActive checks if the affiliate link is in active status
func (a *AffiliateLink) IsActive() bool {
	return a.Status == enum.AffiliateLinkStatusActive && a.DeletedAt.Time.IsZero()
}

// IsExpired checks if the affiliate link is in expired status
func (a *AffiliateLink) IsExpired() bool {
	return a.Status == enum.AffiliateLinkStatusExpired
}

// Deactivate sets the affiliate link status to inactive
func (a *AffiliateLink) Deactivate() {
	a.Status = enum.AffiliateLinkStatusInactive
}

// Expire sets the affiliate link status to expired
func (a *AffiliateLink) Expire() {
	a.Status = enum.AffiliateLinkStatusExpired
}

// Activate sets the affiliate link status to active
func (a *AffiliateLink) Activate() {
	a.Status = enum.AffiliateLinkStatusActive
}
