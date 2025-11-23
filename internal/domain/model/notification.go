package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Notification represents a notification record with flexible JSONB metadata
type Notification struct {
	ID     uuid.UUID               `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID               `gorm:"type:uuid;not null;index"`
	Type   enum.NotificationType   `gorm:"type:varchar(50);not null;index"`
	Status enum.NotificationStatus `gorm:"type:varchar(50);not null;index"`
	IsRead bool                    `gorm:"default:false;not null;index"`

	// JSONB columns for flexible metadata
	DeliveryAttempts JSONBDeliveryAttempts `gorm:"type:jsonb;not null;default:'[]'"`
	RecipientInfo    JSONBRecipientInfo    `gorm:"type:jsonb;not null"`
	ContentData      JSONBContentData      `gorm:"type:jsonb;not null"`
	PlatformConfig   JSONBPlatformConfig   `gorm:"type:jsonb"`
	ErrorDetails     JSONBErrorDetails     `gorm:"type:jsonb"`

	CreatedAt *time.Time     `gorm:"autoCreateTime"`
	UpdatedAt *time.Time     `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate hook to generate UUID
func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Notification) TableName() string {
	return "notifications"
}

// ========== JSONB Type Definitions ==========

// DeliveryAttempt represents a single delivery attempt
type DeliveryAttempt struct {
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"` // "success", "failed", "retrying"
	Error     string    `json:"error,omitempty"`
}

// JSONBDeliveryAttempts is a JSONB type for delivery attempts array
type JSONBDeliveryAttempts []DeliveryAttempt

func (j *JSONBDeliveryAttempts) Scan(value any) error {
	if value == nil {
		*j = []DeliveryAttempt{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONBDeliveryAttempts")
	}
	return json.Unmarshal(bytes, &j)
}

func (j JSONBDeliveryAttempts) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal([]DeliveryAttempt{})
	}
	return json.Marshal(j)
}

// RecipientInfo stores email address or device tokens
type RecipientInfo struct {
	Email  string   `json:"email,omitempty"`
	Tokens []string `json:"tokens,omitempty"` // FCM device tokens
}

// JSONBRecipientInfo is a JSONB type for recipient info
type JSONBRecipientInfo RecipientInfo

func (j *JSONBRecipientInfo) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONBRecipientInfo")
	}
	return json.Unmarshal(bytes, (*RecipientInfo)(j))
}

func (j JSONBRecipientInfo) Value() (driver.Value, error) {
	return json.Marshal(RecipientInfo(j))
}

// ContentData stores notification content
type ContentData struct {
	Subject      string         `json:"subject,omitempty"`       // Email subject
	Body         string         `json:"body,omitempty"`          // Plain text body
	HTMLBody     string         `json:"html_body,omitempty"`     // HTML email body
	Title        string         `json:"title,omitempty"`         // Push notification title
	TemplateName string         `json:"template_name,omitempty"` // Template filename
	TemplateData map[string]any `json:"template_data,omitempty"` // Data for template rendering
	Attachments  []Attachment   `json:"attachments,omitempty"`   // Email attachments
}

// Attachment represents an email attachment
type Attachment struct {
	Filename string `json:"filename"`
	URL      string `json:"url"` // S3 URL
	MimeType string `json:"mime_type"`
}

// JSONBContentData is a JSONB type for content data
type JSONBContentData ContentData

func (j *JSONBContentData) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONBContentData")
	}
	return json.Unmarshal(bytes, (*ContentData)(j))
}

func (j JSONBContentData) Value() (driver.Value, error) {
	return json.Marshal(ContentData(j))
}

// PlatformConfig stores iOS/Android specific settings
type PlatformConfig struct {
	IOSConfig     *IOSConfig     `json:"ios_config,omitempty"`
	AndroidConfig *AndroidConfig `json:"android_config,omitempty"`
}

// IOSConfig contains iOS-specific push notification settings
type IOSConfig struct {
	Badge            *int   `json:"badge,omitempty"`             // Badge count
	Sound            string `json:"sound,omitempty"`             // Sound filename
	ContentAvailable int    `json:"content_available,omitempty"` // Silent push
}

// AndroidConfig contains Android-specific push notification settings
type AndroidConfig struct {
	ChannelID   string `json:"channel_id,omitempty"`   // Notification channel
	ClickAction string `json:"click_action,omitempty"` // Activity to open
	Color       string `json:"color,omitempty"`        // Notification color
	Priority    string `json:"priority,omitempty"`     // "high" or "normal"
}

// JSONBPlatformConfig is a JSONB type for platform config
type JSONBPlatformConfig PlatformConfig

func (j *JSONBPlatformConfig) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONBPlatformConfig")
	}
	return json.Unmarshal(bytes, (*PlatformConfig)(j))
}

func (j JSONBPlatformConfig) Value() (driver.Value, error) {
	return json.Marshal(PlatformConfig(j))
}

// ErrorDetails stores last error information
type ErrorDetails struct {
	ErrorMessage  string     `json:"error_message,omitempty"`
	ErrorCode     string     `json:"error_code,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at,omitempty"`
}

// JSONBErrorDetails is a JSONB type for error details
type JSONBErrorDetails ErrorDetails

func (j *JSONBErrorDetails) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed for JSONBErrorDetails")
	}
	return json.Unmarshal(bytes, (*ErrorDetails)(j))
}

func (j JSONBErrorDetails) Value() (driver.Value, error) {
	return json.Marshal(ErrorDetails(j))
}
