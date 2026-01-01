package requests

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// PublishNotificationRequest represents a request to publish a notification to one or many channels
type PublishNotificationRequest struct {
	UserID   uuid.UUID                  `json:"user_id" validate:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Types    []enum.NotificationType    `json:"channels" validate:"required,min=1,dive,oneof=EMAIL PUSH IN_APP" example:"EMAIL,PUSH,IN_APP"`
	Title    string                     `json:"title" validate:"required,min=1,max=255" example:"Test Notification"`
	Body     string                     `json:"body" validate:"required,min=1" example:"This is a test notification message"`
	Data     map[string]string          `json:"data,omitempty" example:"key1:value1,key2:value2"`
	Severity *enum.NotificationSeverity `json:"severity,omitempty" validate:"omitempty" example:"INFO"`

	// Email-specific fields (used when EMAIL channel is included)
	CustomReceiver    *string        `json:"custom_receivers,omitempty" validate:"omitempty,email" example:"abc@gmail.com"`
	EmailSubject      *string        `json:"email_subject,omitempty" validate:"omitempty,min=1,max=255" example:"Test Email Subject"`
	EmailTemplateName *string        `json:"email_template_name,omitempty" validate:"omitempty,min=1" example:"task_assigned"`
	EmailTemplateData map[string]any `json:"email_template_data,omitempty"`
	EmailHTMLBody     *string        `json:"email_html_body,omitempty" validate:"omitempty,min=1"`

	// In case the notification is published through scheduling
	ScheduleIDsByChannels map[enum.NotificationType]uuid.UUID `json:"schedule_id,omitempty" validate:"omitempty"`
}

func (r *PublishNotificationRequest) ToEmailRequest() *PublishEmailRequest {
	resp := &PublishEmailRequest{
		UserID:  r.UserID,
		Subject: r.Title,
	}
	if r.EmailSubject != nil {
		resp.Subject = *r.EmailSubject
	}
	if r.EmailTemplateName != nil {
		resp.TemplateName = r.EmailTemplateName
	}
	if r.EmailTemplateData != nil {
		resp.TemplateData = r.EmailTemplateData
	}
	if r.EmailHTMLBody != nil {
		resp.HTMLBody = r.EmailHTMLBody
	}
	tempID, ok := r.ScheduleIDsByChannels[enum.NotificationTypeEmail]
	if ok {
		resp.ScheduleID = &tempID
	}
	return resp
}

func (r *PublishNotificationRequest) ToPushRequest() *PublishPushRequest {
	resp := &PublishPushRequest{UserID: r.UserID}
	if r.Title != "" {
		resp.Title = r.Title
	}
	if r.Body != "" {
		resp.Body = r.Body
	}
	if r.Data != nil {
		resp.Data = r.Data
	}
	tempID, ok := r.ScheduleIDsByChannels[enum.NotificationTypePush]
	if ok {
		resp.ScheduleID = &tempID
	}
	return resp
}

func (r *PublishNotificationRequest) ToInAppRequest() *PublishInAppRequest {
	resp := &PublishInAppRequest{UserID: r.UserID}
	if r.Title != "" {
		resp.Title = r.Title
	}
	if r.Body != "" {
		resp.Body = r.Body
	}
	if r.Data != nil {
		resp.Data = r.Data
	}
	tempID, ok := r.ScheduleIDsByChannels[enum.NotificationTypeInApp]
	if ok {
		resp.ScheduleID = &tempID
	}
	if r.Severity != nil {
		resp.Severity = *r.Severity
	} else {
		resp.Severity = enum.NotificationSeverityInfo
	}
	return resp
}

// PublishEmailRequest represents a request to publish an email notification
type PublishEmailRequest struct {
	UserID  uuid.UUID `json:"user_id" validate:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	To      string    `json:"to" validate:"required,email" example:"user@example.com"`
	Subject string    `json:"subject" validate:"required,min=1,max=255" example:"Test Email Subject"`

	// Either use template or provide body directly
	TemplateName *string        `json:"template_name,omitempty" validate:"omitempty,min=1" example:"task_assigned"`
	TemplateData map[string]any `json:"template_data,omitempty"`
	HTMLBody     *string        `json:"html_body,omitempty" validate:"omitempty,min=1"`

	Priority string            `json:"priority,omitempty" validate:"omitempty,oneof=low normal high" example:"normal"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// In case the notification is published through scheduling
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty" validate:"omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// PublishPushRequest represents a request to publish a push notification
type PublishPushRequest struct {
	UserID uuid.UUID         `json:"user_id" validate:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Title  string            `json:"title" validate:"required,min=1,max=255" example:"Test Push Notification"`
	Body   string            `json:"body" validate:"required,min=1" example:"This is a test push notification"`
	Data   map[string]string `json:"data,omitempty"`

	// Platform-specific configurations (optional)
	IOSBadge               *int    `json:"ios_badge,omitempty" example:"1"`
	IOSSound               *string `json:"ios_sound,omitempty" example:"default"`
	AndroidPriority        *string `json:"android_priority,omitempty" validate:"omitempty,oneof=min low default high max" example:"high"`
	AndroidNotificationTag *string `json:"android_notification_tag,omitempty" example:"group1"`

	// In case the notification is published through scheduling
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty" validate:"omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// PublishInAppRequest represents a request to publish an in-app notification
type PublishInAppRequest struct {
	UserID   uuid.UUID                 `json:"user_id" validate:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Title    string                    `json:"title" validate:"required,min=1,max=255" example:"Test In-App Notification"`
	Body     string                    `json:"body" validate:"required,min=1" example:"This is a test in-app notification"`
	Data     map[string]string         `json:"data,omitempty"`
	Severity enum.NotificationSeverity `json:"severity,omitempty" validate:"omitempty" example:"INFO"`

	// In case the notification is published through scheduling
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty" validate:"omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// RepublishFailedNotificationRequest represents a request to republish failed notifications
type RepublishFailedNotificationRequest struct {
	NotificationIDs []uuid.UUID `json:"notification_ids,omitempty" validate:"omitempty,dive,required" example:"123e4567-e89b-12d3-a456-426614174000"`
	MinRetries      *int        `json:"min_retries,omitempty" validate:"omitempty,min=1,max=10" example:"3"`
	Type            *string     `json:"type,omitempty" validate:"omitempty,oneof=EMAIL PUSH" example:"EMAIL"`
}
