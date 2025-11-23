package gormrepository

import (
	"context"
	"encoding/json"
	"time"

	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationRepository implements the notification repository interface
type NotificationRepository struct {
	*genericRepository[model.Notification]
}

// NewNotificationRepository creates a new notification repository
func NewNotificationRepository(db *gorm.DB) irepository.NotificationRepository {
	return &NotificationRepository{
		genericRepository: &genericRepository[model.Notification]{db: db},
	}
}

// FindByStatus retrieves notifications by status
func (r *NotificationRepository) FindByStatus(ctx context.Context, status enum.NotificationStatus, limit int) ([]*model.Notification, error) {
	var notifications []*model.Notification
	err := r.db.WithContext(ctx).
		Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Find(&notifications).Error
	return notifications, err
}

// FindByEmailRecipient retrieves notifications sent to a specific email
func (r *NotificationRepository) FindByEmailRecipient(ctx context.Context, email string) ([]*model.Notification, error) {
	var notifications []*model.Notification
	err := r.db.WithContext(ctx).
		Where("recipient_info->>'email' = ?", email).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

// FindFailedWithRetries retrieves failed notifications with minimum retry attempts
func (r *NotificationRepository) FindFailedWithRetries(ctx context.Context, minRetries int) ([]*model.Notification, error) {
	var notifications []*model.Notification
	err := r.db.WithContext(ctx).
		Where("jsonb_array_length(delivery_attempts) >= ?", minRetries).
		Where("status = ?", enum.NotificationStatusFailed).
		Order("created_at DESC").
		Find(&notifications).Error
	return notifications, err
}

// UpdateDeliveryAttempt appends a new delivery attempt to the JSONB array
func (r *NotificationRepository) UpdateDeliveryAttempt(ctx context.Context, id uuid.UUID, attempt model.DeliveryAttempt) error {
	// Convert attempt to JSON
	attemptJSON, err := json.Marshal(attempt)
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Update("delivery_attempts", gorm.Expr("delivery_attempts || ?::jsonb", string(attemptJSON))).Error
}

// UpdateErrorDetails updates the error details JSONB field
func (r *NotificationRepository) UpdateErrorDetails(ctx context.Context, id uuid.UUID, errorDetails model.ErrorDetails) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Update("error_details", model.JSONBErrorDetails(errorDetails)).Error
}

// UpdateStatus updates the notification status
func (r *NotificationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status enum.NotificationStatus) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// FindByUserID retrieves notifications for a specific user with pagination
func (r *NotificationRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Notification, int64, error) {
	var notifications []*model.Notification
	var total int64

	// Count total
	if err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

// CleanupOldNotifications deletes notifications older than the specified date
func (r *NotificationRepository) CleanupOldNotifications(ctx context.Context, olderThan time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", olderThan).
		Delete(&model.Notification{})
	return result.RowsAffected, result.Error
}

// CountUnread counts unread notifications for a specific user
func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID).
		Where("is_read = ?", false).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ?", id).
		Update("is_read", true).Error
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID).
		Where("is_read = ?", false).
		Update("is_read", true).Error
}
