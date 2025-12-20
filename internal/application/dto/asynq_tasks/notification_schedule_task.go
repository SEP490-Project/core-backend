package asynqtask

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

// ScheduledNotificationPayload is the payload for scheduled notification tasks
type ScheduledNotificationPayload struct {
	UserID       uuid.UUID               `json:"user_id"`
	Title        string                  `json:"title"`
	Body         string                  `json:"body"`
	Types        []enum.NotificationType `json:"type"`     // "EMAIL", "PUSH", "IN_APP", "ALL"
	Priority     string                  `json:"priority"` // "high", "normal", "low"
	ScheduledAt  time.Time               `json:"scheduled_at"`
	Data         map[string]string       `json:"data,omitempty"`
	TemplateData map[string]any          `json:"template_data,omitempty"` // For email notifications
	TemplateName string                  `json:"template_name,omitempty"` // For email notifications
	Subject      string                  `json:"subject,omitempty"`       // For email notifications

	ScheduleIDs  []uuid.UUID        `json:"schedule_id,omitempty"`
	ScheduleType *enum.ScheduleType `json:"schedule_type,omitempty"`
}
