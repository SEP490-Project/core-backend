package consumers

import (
	"time"

	"github.com/google/uuid"
)

// ContentViewMessage represents the message structure for async content view tracking via RabbitMQ
type ContentViewMessage struct {
	ContentChannelID uuid.UUID  `json:"content_channel_id"`     // ContentChannel being viewed
	ContentID        uuid.UUID  `json:"content_id"`             // Parent content ID
	UserID           *uuid.UUID `json:"user_id,omitempty"`      // Authenticated user (optional)
	IPAddress        string     `json:"ip_address"`             // Client IP address
	UserAgent        string     `json:"user_agent"`             // Browser user agent
	ReferrerURL      *string    `json:"referrer_url,omitempty"` // HTTP Referer header
	SessionID        *string    `json:"session_id,omitempty"`   // Session tracking ID (optional)
	ViewedAt         time.Time  `json:"viewed_at"`              // Timestamp when view occurred
	IsBot            bool       `json:"is_bot"`                 // Bot detection flag
	DeviceType       string     `json:"device_type,omitempty"`  // mobile, tablet, desktop
	Platform         string     `json:"platform,omitempty"`     // iOS, Android, Windows, etc.
	Browser          string     `json:"browser,omitempty"`      // Chrome, Safari, Firefox, etc.
}
