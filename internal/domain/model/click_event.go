package model

import (
	"net"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ClickEvent represents an individual click event in the TimescaleDB hypertable
// IMPORTANT: This table uses a composite primary key (ID, ClickedAt) for TimescaleDB partitioning
type ClickEvent struct {
	ID              uuid.UUID  `json:"id" gorm:"primaryKey;type:uuid"`                                                       // Part of composite PK
	AffiliateLinkID uuid.UUID  `json:"affiliate_link_id" gorm:"type:uuid;not null;index:idx_click_events_affiliate_link_id"` // Foreign key
	UserID          *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index:idx_click_events_user_id"`                    // Nullable for anonymous clicks
	ClickedAt       time.Time  `json:"clicked_at" gorm:"primaryKey;not null;index:idx_click_events_clicked_at"`              // Partition key - Part of composite PK
	IPAddress       *string    `json:"ip_address,omitempty" gorm:"type:inet"`                                                // Store hashed or anonymized IP
	UserAgent       *string    `json:"user_agent,omitempty" gorm:"type:text"`                                                // Browser user agent
	ReferrerURL     *string    `json:"referrer_url,omitempty" gorm:"type:text"`                                              // Source page URL
	SessionID       *string    `json:"session_id,omitempty" gorm:"type:varchar(255);index:idx_click_events_session_id"`      // Session identifier

	// Relationship (optional, lazy load)
	AffiliateLink *AffiliateLink `json:"affiliate_link,omitempty" gorm:"foreignKey:AffiliateLinkID"`
	User          *User          `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

func (ClickEvent) TableName() string {
	return "click_events"
}

// BeforeCreate generates UUID and sets ClickedAt if not provided
func (c *ClickEvent) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.ClickedAt.IsZero() {
		c.ClickedAt = time.Now()
	}
	return nil
}

// IsAuthenticated checks if the click is from an authenticated user
func (c *ClickEvent) IsAuthenticated() bool {
	return c.UserID != nil
}

// HasSession checks if the click has a session identifier
func (c *ClickEvent) HasSession() bool {
	return c.SessionID != nil && *c.SessionID != ""
}

// AnonymizeIP truncates the IP address for privacy.
// For IPv4, it sets the last octet to 0 (e.g., "192.168.1.123" -> "192.168.1.0").
// For IPv6, it sets the last 64 bits to 0.
func (c *ClickEvent) AnonymizeIP() {
	// 1. Check if the IPAddress field is nil or empty to avoid panics.
	if c.IPAddress == nil || *c.IPAddress == "" {
		return
	}

	// 2. Parse the string into a net.IP object. Handles both IPv4 and IPv6.
	ip := net.ParseIP(*c.IPAddress)
	if ip == nil {
		// Invalid IP — skip anonymization to avoid corrupting data.
		zap.L().Warn("Invalid IP address format, skipping anonymization", zap.String("ip_address", *c.IPAddress))
		return
	}

	// 3. Apply anonymization mask depending on address type.
	if ipv4 := ip.To4(); ipv4 != nil {
		// IPv4: zero out last 8 bits.
		mask := net.CIDRMask(24, 32)
		ip = ipv4.Mask(mask)
	} else {
		// IPv6: zero out last 64 bits.
		mask := net.CIDRMask(64, 128)
		ip = ip.Mask(mask)
	}

	// 4. Store the anonymized IP as a string.
	anonymized := ip.String()
	c.IPAddress = &anonymized
}
