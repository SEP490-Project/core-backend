package consumers

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// InAppNotificationMessage represents the payload for in-app notifications
type InAppNotificationMessage struct {
	NotificationID uuid.UUID                 `json:"notification_id"`
	UserID         uuid.UUID                 `json:"user_id"`
	Title          string                    `json:"title"`
	Message        string                    `json:"message"`
	Severity       enum.NotificationSeverity `json:"severity"`
	Data           map[string]string         `json:"data,omitempty"`
	CreatedAt      string                    `json:"created_at"`

	// Optional ScheduleID if the notification is related to a scheduled task
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty"`
}
