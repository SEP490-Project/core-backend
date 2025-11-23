package consumers

import "github.com/google/uuid"

// UnifiedNotificationMessage represents a generic notification message
// that can be processed by any notification channel (Email, Push, In-App)
type UnifiedNotificationMessage struct {
	NotificationID uuid.UUID         `json:"notification_id"`
	UserID         uuid.UUID         `json:"user_id"`
	Title          string            `json:"title"`           // Subject for Email, Title for Push/InApp
	Body           string            `json:"body"`            // HTMLBody for Email, Body for Push, Message for InApp
	Data           map[string]string `json:"data,omitempty"`  // TemplateData for Email, Data for Push/InApp
	Type           string            `json:"type,omitempty"`  // Specific type for InApp (info, warning, etc)
	TargetChannels []string          `json:"target_channels"` // List of channels to target (EMAIL, PUSH, IN_APP)
	CreatedAt      string            `json:"created_at"`
}
