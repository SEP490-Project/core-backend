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

	// Optional fields for email notifications
	HTMLBody     *string           `json:"html_body,omitempty"`
	TemplateName *string           `json:"template_name,omitempty"`
	TemplateData map[string]any    `json:"template_data,omitempty"`
	Attachments  []Attachment      `json:"attachments,omitempty"`
	Priority     *string           `json:"priority,omitempty" validate:"omitempty,oneof=low normal high"`
	Metadata     map[string]string `json:"metadata,omitempty"`

	// Optional fields for push notifications
	PlatformConfig *PlatformConfig `json:"platform_config,omitempty"`

	// Optional schedule ID if part of a scheduled notification
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty"`
}

func (m *UnifiedNotificationMessage) ToEmailRequest() *EmailNotificationMessage {
	response := &EmailNotificationMessage{
		NotificationID: m.NotificationID,
		UserID:         m.UserID,
		Subject:        m.Title,
		ScheduleID:     m.ScheduleID,
	}
	if m.HTMLBody != nil {
		response.HTMLBody = *m.HTMLBody
	}
	if m.TemplateName != nil {
		response.TemplateName = *m.TemplateName
	}
	if m.TemplateData != nil {
		response.TemplateData = m.TemplateData
	}
	if m.Attachments != nil {
		response.Attachments = m.Attachments
	}
	if m.Priority != nil {
		response.Priority = *m.Priority
	}
	if m.Metadata != nil {
		response.Metadata = m.Metadata
	}
	return response
}

func (m *UnifiedNotificationMessage) ToPushRequest() *PushNotificationMessage {
	response := &PushNotificationMessage{
		NotificationID: m.NotificationID,
		UserID:         m.UserID,
		Title:          m.Title,
		Body:           m.Body,
		Data:           m.Data,
		ScheduleID:     m.ScheduleID,
	}
	if m.PlatformConfig != nil {
		response.PlatformConfig = m.PlatformConfig
	}
	return response
}

func (m *UnifiedNotificationMessage) ToInAppRequest() *InAppNotificationMessage {
	response := &InAppNotificationMessage{
		NotificationID: m.NotificationID,
		UserID:         m.UserID,
		Title:          m.Title,
		Message:        m.Body,
		Type:           m.Type,
		Data:           m.Data,
		CreatedAt:      m.CreatedAt,
		ScheduleID:     m.ScheduleID,
	}
	if m.Type != "" {
		response.Type = "info"
	}
	return response
}
