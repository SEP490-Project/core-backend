package consumers

import "github.com/google/uuid"

// InAppNotificationMessage represents the payload for in-app notifications
type InAppNotificationMessage struct {
	NotificationID uuid.UUID         `json:"notification_id"`
	UserID         uuid.UUID         `json:"user_id"`
	Title          string            `json:"title"`
	Message        string            `json:"message"`
	Type           string            `json:"type"` // e.g. "info", "warning", "error"
	Data           map[string]string `json:"data,omitempty"`
	CreatedAt      string            `json:"created_at"`
}
