package iservice

import (
	"context"
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
	GetByFilters(ctx context.Context, userID *uuid.UUID, notificationType *enum.NotificationType, status *enum.NotificationStatus, startDate, endDate *string, page, limit int) ([]*model.Notification, int64, error)
}
