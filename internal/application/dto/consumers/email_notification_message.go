package consumers

import "github.com/google/uuid"

// EmailNotificationMessage represents the message structure for email notifications
// sent through RabbitMQ queue.notification.email
type EmailNotificationMessage struct {
	NotificationID uuid.UUID         `json:"notification_id" validate:"required"`
	UserID         uuid.UUID         `json:"user_id" validate:"required"`
	To             string            `json:"to" validate:"required,email"`
	Subject        string            `json:"subject" validate:"required,min=1,max=255"`
	Body           string            `json:"body,omitempty"`
	HTMLBody       string            `json:"html_body,omitempty"`
	TemplateName   string            `json:"template_name,omitempty"`
	TemplateData   map[string]any    `json:"template_data,omitempty"`
	Attachments    []Attachment      `json:"attachments,omitempty"`
	Priority       string            `json:"priority,omitempty" validate:"omitempty,oneof=low normal high"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// Attachment represents an email attachment with S3 URL reference
type Attachment struct {
	Filename string `json:"filename" validate:"required"`
	URL      string `json:"url" validate:"required,url"`
	MimeType string `json:"mime_type" validate:"required"`
}
