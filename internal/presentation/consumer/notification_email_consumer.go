package consumer

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationEmailConsumer handles email notification messages from RabbitMQ
type NotificationEmailConsumer struct {
	emailService     iservice_third_party.EmailService
	notificationRepo irepository.NotificationRepository
	userService      iservice.UserService
	validator        *validator.Validate
	healthMonitor    iservice_third_party.HealthMonitor
}

// NewNotificationEmailConsumer creates a new email notification consumer
func NewNotificationEmailConsumer(
	infraRegistry *infrastructure.InfrastructureRegistry,
	dbRegistry *gormrepository.DatabaseRegistry,
	userService iservice.UserService,
) *NotificationEmailConsumer {
	return &NotificationEmailConsumer{
		emailService:     infraRegistry.EmailService,
		notificationRepo: dbRegistry.NotificationRepository,
		userService:      userService,
		validator:        validator.New(),
		healthMonitor:    infraRegistry.HealthMonitor,
	}
}

// Handle processes email notification messages
func (c *NotificationEmailConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received email notification message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.EmailNotificationMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal email notification message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if err := c.validator.Struct(msg); err != nil {
		zap.L().Error("Invalid email notification message",
			zap.Error(err),
			zap.String("notification_id", msg.NotificationID.String()))

		// Update error details in database
		c.updateNotificationError(ctx, msg.NotificationID, "validation_failed", err.Error())
		return fmt.Errorf("message validation failed: %w", err)
	}

	zap.L().Info("Processing email notification",
		zap.String("notification_id", msg.NotificationID.String()),
		zap.String("to", msg.To),
		zap.String("subject", msg.Subject),
		zap.String("priority", msg.Priority))

	// Check user notification preferences if UserID is provided
	if msg.UserID != uuid.Nil {
		emailEnabled, _, err := c.userService.GetOrCreateDefault(ctx, msg.UserID)
		if err != nil {
			zap.L().Error("Failed to get notification preferences",
				zap.String("user_id", msg.UserID.String()),
				zap.String("notification_id", msg.NotificationID.String()),
				zap.Error(err))
			// Continue with send on error (fail-open approach)
		} else if !emailEnabled {
			zap.L().Info("Email notifications disabled for user, skipping send",
				zap.String("user_id", msg.UserID.String()),
				zap.String("notification_id", msg.NotificationID.String()),
				zap.String("to", msg.To))

			// Update status to skipped (custom status - you may want to add this to enum)
			c.updateNotificationStatus(ctx, msg.NotificationID, enum.NotificationStatusSent)
			c.updateNotificationError(ctx, msg.NotificationID, "user_preference", "Email notifications disabled by user")

			// Return success (no retry needed)
			return nil
		}
	}

	// Check email service health before attempting to send
	if c.healthMonitor != nil && !c.healthMonitor.IsEmailHealthy() {
		health := c.healthMonitor.GetEmailHealth()
		zap.L().Warn("Email service is unhealthy, skipping send",
			zap.String("notification_id", msg.NotificationID.String()),
			zap.Error(health.LastError))

		// Update status to retrying (will retry when service recovers)
		c.updateNotificationStatus(ctx, msg.NotificationID, enum.NotificationStatusRetrying)
		c.updateNotificationError(ctx, msg.NotificationID, "service_unhealthy", "Email service is temporarily unavailable")

		// Return error to trigger RabbitMQ retry
		return fmt.Errorf("email service unhealthy: %w", health.LastError)
	}

	// Send email
	var err error
	if msg.TemplateName != "" {
		// Send templated email
		zap.L().Debug("Sending templated email",
			zap.String("template", msg.TemplateName),
			zap.String("notification_id", msg.NotificationID.String()))
		err = c.emailService.SendTemplatedEmail(ctx, msg.To, msg.Subject, msg.TemplateName, msg.TemplateData)
	} else if msg.HTMLBody != "" {
		// Send HTML email
		zap.L().Debug("Sending HTML email",
			zap.String("notification_id", msg.NotificationID.String()))
		err = c.emailService.SendEmail(ctx, msg.To, msg.Subject, &msg.HTMLBody, true)
	} else if msg.Body != "" {
		// Send plain text email
		zap.L().Debug("Sending plain text email",
			zap.String("notification_id", msg.NotificationID.String()))
		err = c.emailService.SendEmail(ctx, msg.To, msg.Subject, &msg.Body, false)
	} else {
		err = fmt.Errorf("no email content provided: body, html_body, or template_name required")
	}

	// Log delivery attempt
	attempt := model.DeliveryAttempt{
		Timestamp: time.Now(),
		Status:    "success",
	}

	if err != nil {
		attempt.Status = "failed"
		attempt.Error = err.Error()
		zap.L().Error("Failed to send email",
			zap.String("notification_id", msg.NotificationID.String()),
			zap.String("to", msg.To),
			zap.Error(err))

		// Update notification with error details
		c.updateNotificationError(ctx, msg.NotificationID, "smtp_error", err.Error())
		c.logDeliveryAttempt(ctx, msg.NotificationID, attempt)

		// Return error to trigger RabbitMQ retry
		return fmt.Errorf("failed to send email: %w", err)
	}

	zap.L().Info("Email notification sent successfully",
		zap.String("notification_id", msg.NotificationID.String()),
		zap.String("to", msg.To))

	// Update notification status to SENT
	c.updateNotificationStatus(ctx, msg.NotificationID, enum.NotificationStatusSent)
	c.logDeliveryAttempt(ctx, msg.NotificationID, attempt)

	return nil
}

// updateNotificationStatus updates the notification status in the database
func (c *NotificationEmailConsumer) updateNotificationStatus(ctx context.Context, notificationID uuid.UUID, status enum.NotificationStatus) {
	if err := c.notificationRepo.UpdateStatus(ctx, notificationID, status); err != nil {
		zap.L().Error("Failed to update notification status",
			zap.String("notification_id", notificationID.String()),
			zap.String("status", string(status)),
			zap.Error(err))
	}
}

// logDeliveryAttempt logs a delivery attempt to the database
func (c *NotificationEmailConsumer) logDeliveryAttempt(ctx context.Context, notificationID uuid.UUID, attempt model.DeliveryAttempt) {
	if err := c.notificationRepo.UpdateDeliveryAttempt(ctx, notificationID, attempt); err != nil {
		zap.L().Error("Failed to log delivery attempt",
			zap.String("notification_id", notificationID.String()),
			zap.Error(err))
	}
}

// updateNotificationError updates the error details in the database
func (c *NotificationEmailConsumer) updateNotificationError(ctx context.Context, notificationID uuid.UUID, errorCode, errorMessage string) {
	now := time.Now()
	errorDetails := model.ErrorDetails{
		ErrorCode:     errorCode,
		ErrorMessage:  errorMessage,
		LastAttemptAt: &now,
	}

	if err := c.notificationRepo.UpdateErrorDetails(ctx, notificationID, errorDetails); err != nil {
		zap.L().Error("Failed to update notification error details",
			zap.String("notification_id", notificationID.String()),
			zap.Error(err))
	}
}
