package irepository

import (
	"context"
	"time"

	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// NotificationRepository defines the interface for notification data access
type NotificationRepository interface {
	GenericRepository[model.Notification]

	// FindByStatus retrieves notifications by status
	FindByStatus(ctx context.Context, status enum.NotificationStatus, limit int) ([]*model.Notification, error)

	// FindByEmailRecipient retrieves notifications sent to a specific email
	FindByEmailRecipient(ctx context.Context, email string) ([]*model.Notification, error)

	// FindFailedWithRetries retrieves failed notifications with minimum retry attempts
	FindFailedWithRetries(ctx context.Context, minRetries int) ([]*model.Notification, error)

	// UpdateDeliveryAttempt appends a new delivery attempt to the JSONB array
	UpdateDeliveryAttempt(ctx context.Context, id uuid.UUID, attempt model.DeliveryAttempt) error

	// UpdateErrorDetails updates the error details JSONB field
	UpdateErrorDetails(ctx context.Context, id uuid.UUID, errorDetails model.ErrorDetails) error

	// UpdateStatus updates the notification status
	UpdateStatus(ctx context.Context, id uuid.UUID, status enum.NotificationStatus) error

	// FindByUserID retrieves notifications for a specific user with pagination
	FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Notification, int64, error)

	// CountUnread counts unread notifications for a specific user
	CountUnread(ctx context.Context, userID uuid.UUID, notiType []enum.NotificationType) (int64, error)

	// MarkAsRead marks a notification as read
	MarkAsRead(ctx context.Context, id uuid.UUID) error

	// MarkAllAsRead marks all notifications as read for a user
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

	// CleanupOldNotifications deletes notifications older than the specified date
	CleanupOldNotifications(ctx context.Context, olderThan time.Time) (int64, error)
}
