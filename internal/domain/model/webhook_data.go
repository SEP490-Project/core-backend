package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookData stores raw webhook payloads for audit and debugging purposes
type WebhookData struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	Source       WebhookSource  `gorm:"type:varchar(50);not null"`     // Source: payos, facebook, tiktok, ghn, other
	EventType    *string        `gorm:"type:varchar(100)"`             // Type of event
	ExternalID   *string        `gorm:"type:varchar(255);index"`       // External reference ID
	RawQuery     datatypes.JSON `gorm:"type:jsonb"`                    // Raw query parameters
	RawPayload   datatypes.JSON `gorm:"type:jsonb;not null"`           // Raw webhook payload
	Processed    bool           `gorm:"default:false"`                 // Whether processed
	ProcessedAt  *time.Time     `gorm:"type:timestamp with time zone"` // When processed
	ErrorMessage *string        `gorm:"type:text"`                     // Error message if failed
	CreatedAt    *time.Time     `gorm:"type:timestamp with time zone;autoCreateTime"`
}

// TableName returns the table name for GORM
func (WebhookData) TableName() string {
	return "webhook_data"
}

// BeforeCreate generates UUID if not set
func (w *WebhookData) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// Webhook source constants

type WebhookSource string

const (
	WebhookSourcePayOS    WebhookSource = "payos"
	WebhookSourceFacebook WebhookSource = "facebook"
	WebhookSourceTikTok   WebhookSource = "tiktok"
	WebhookSourceGHN      WebhookSource = "ghn"
	WebhookSourceOther    WebhookSource = "other"
)

// MarkProcessed marks the webhook as processed
func (w *WebhookData) MarkProcessed() {
	w.Processed = true
	now := time.Now()
	w.ProcessedAt = &now
}

// MarkFailed marks the webhook as failed with an error message
func (w *WebhookData) MarkFailed(errMsg string) {
	w.Processed = false
	w.ErrorMessage = &errMsg
}
