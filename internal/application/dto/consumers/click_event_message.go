package consumers

import (
	"time"

	"github.com/google/uuid"
)

// ClickEventMessage represents the message structure for async click event logging via RabbitMQ
type ClickEventMessage struct {
	AffiliateLinkID uuid.UUID  `json:"affiliate_link_id"`
	UserID          *uuid.UUID `json:"user_id,omitempty"`      // Authenticated user (optional)
	IPAddress       string     `json:"ip_address"`             // Client IP address
	UserAgent       string     `json:"user_agent"`             // Browser user agent
	ReferrerURL     *string    `json:"referrer_url,omitempty"` // HTTP Referer header
	SessionID       *string    `json:"session_id,omitempty"`   // Session tracking ID (optional)
	ClickedAt       time.Time  `json:"clicked_at"`             // Timestamp when click occurred
	IsBot           bool       `json:"is_bot"`                 // Bot detection flag
	DeviceType      string     `json:"device_type,omitempty"`  // mobile, tablet, desktop
	Platform        string     `json:"platform,omitempty"`     // iOS, Android, Windows, etc.
	Browser         string     `json:"browser,omitempty"`      // Chrome, Safari, Firefox, etc.
	Country         string     `json:"country,omitempty"`      // GeoIP country code
	City            string     `json:"city,omitempty"`         // GeoIP city name
}
