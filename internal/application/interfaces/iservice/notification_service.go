package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// NotificationService defines the interface for notification monitoring operations
type NotificationService interface {
	// GetByID retrieves a notification by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error)

	// GetByUser retrieves notifications for a specific user with pagination
	GetByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*model.Notification, int64, error)

	// GetByStatus retrieves notifications by status with pagination
	GetByStatus(ctx context.Context, status enum.NotificationStatus, page, limit int) ([]*model.Notification, int64, error)

	// GetFailedWithRetries retrieves notifications that failed after multiple retry attempts
	GetFailedWithRetries(ctx context.Context, minRetries int, page, limit int) ([]*model.Notification, int64, error)

	// GetByFilters retrieves notifications with multiple filter criteria
	GetByFilters(ctx context.Context, userID *uuid.UUID, notificationType *enum.NotificationType, status *enum.NotificationStatus, isRead *bool, startDate, endDate *string, page, limit int) ([]*model.Notification, int64, error)

	// CreateAndPublishNotification creates a notification record and publishes it to specified channels
	CreateAndPublishNotification(ctx context.Context, req *requests.PublishNotificationRequest) ([]uuid.UUID, error)

	// CreateAndPublishEmail creates an email notification record and publishes it
	CreateAndPublishEmail(ctx context.Context, req *requests.PublishEmailRequest) (uuid.UUID, error)

	// CreateAndPublishPush creates a push notification record and publishes it
	CreateAndPublishPush(ctx context.Context, req *requests.PublishPushRequest) (uuid.UUID, error)

	// CreateAndPublishInApp creates an in-app notification record and publishes it
	CreateAndPublishInApp(ctx context.Context, req *requests.PublishInAppRequest) (uuid.UUID, error)

	// RepublishFailedNotifications republishes failed notifications based on filter criteria
	RepublishFailedNotifications(ctx context.Context, req *requests.RepublishFailedNotificationRequest) (int, error)

	// MarkAsRead marks a notification as read
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error

	// MarkAllAsRead marks all notifications as read for a user
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error

	// SubscribeSSE subscribes a user to real-time notification updates
	SubscribeSSE(userID uuid.UUID) (<-chan SSEMessage, func())

	// BroadcastToUser sends a unified notification to a specific user across specified channels
	BroadcastToUser(ctx context.Context, userID uuid.UUID, title, body string, data map[string]string, channels []string) error

	// BroadcastToAll sends a unified notification to all users (optionally filtered by role)
	BroadcastToAll(ctx context.Context, title, body string, data map[string]string, role *string) error
}
