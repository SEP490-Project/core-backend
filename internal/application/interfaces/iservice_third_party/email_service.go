package iservice_third_party

import (
	"context"
)

// EmailService defines the contract for sending emails
type EmailService interface {
	// SendEmail sends an email with the specified parameters
	SendEmail(ctx context.Context, to, subject string, body *string, isHTML bool) error

	// SendTemplatedEmail renders and sends an email using a template
	SendTemplatedEmail(ctx context.Context, to, subject, templateName string, data map[string]any) error

	// Health returns the health status of the email service
	Health(ctx context.Context) ServiceHealth
}
