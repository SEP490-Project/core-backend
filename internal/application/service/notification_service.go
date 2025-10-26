package service

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// NotificationService implements notification monitoring operations
type NotificationService struct {
	notificationRepo irepository.NotificationRepository
}

// NewNotificationService creates a new notification service instance
func NewNotificationService(notificationRepo irepository.NotificationRepository) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
	}
}

// GetByID retrieves a notification by its ID
func (s *NotificationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	zap.L().Info("Fetching notification by ID",
		zap.String("notification_id", id.String()))

	notification, err := s.notificationRepo.GetByID(ctx, id, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Notification not found",
				zap.String("notification_id", id.String()))
			return nil, errors.New("notification not found")
		}
		zap.L().Error("Failed to fetch notification",
			zap.String("notification_id", id.String()),
			zap.Error(err))
		return nil, errors.New("failed to fetch notification")
	}

	return notification, nil
}

// GetByUser retrieves notifications for a specific user with pagination
func (s *NotificationService) GetByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching notifications for user",
		zap.String("user_id", userID.String()),
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := s.notificationRepo.GetAll(
		ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("user_id = ?", userID).Order("created_at DESC")
		},
		nil,
		limit,
		page,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications for user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, 0, errors.New("failed to fetch notifications")
	}

	zap.L().Info("Fetched notifications for user",
		zap.String("user_id", userID.String()),
		zap.Int("count", len(notifications)),
		zap.Int64("total", total))

	// Convert []model.Notification to []*model.Notification
	notificationPtrs := make([]*model.Notification, len(notifications))
	for i := range notifications {
		notificationPtrs[i] = &notifications[i]
	}

	return notificationPtrs, total, nil
}

// GetByStatus retrieves notifications by status with pagination
func (s *NotificationService) GetByStatus(ctx context.Context, status enum.NotificationStatus, page, limit int) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching notifications by status",
		zap.String("status", string(status)),
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	if !status.IsValid() {
		return nil, 0, errors.New("invalid notification status")
	}

	notifications, total, err := s.notificationRepo.GetAll(
		ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", status).Order("created_at DESC")
		},
		nil,
		limit,
		page,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications by status",
			zap.String("status", string(status)),
			zap.Error(err))
		return nil, 0, errors.New("failed to fetch notifications")
	}

	zap.L().Info("Fetched notifications by status",
		zap.String("status", string(status)),
		zap.Int("count", len(notifications)),
		zap.Int64("total", total))

	// Convert []model.Notification to []*model.Notification
	notificationPtrs := make([]*model.Notification, len(notifications))
	for i := range notifications {
		notificationPtrs[i] = &notifications[i]
	}

	return notificationPtrs, total, nil
}

// GetFailedWithRetries retrieves notifications that failed after multiple retry attempts
func (s *NotificationService) GetFailedWithRetries(ctx context.Context, minRetries int, page, limit int) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching failed notifications with retries",
		zap.Int("min_retries", minRetries),
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, err := s.notificationRepo.FindFailedWithRetries(ctx, minRetries)
	if err != nil {
		zap.L().Error("Failed to fetch notifications with retries",
			zap.Int("min_retries", minRetries),
			zap.Error(err))
		return nil, 0, errors.New("failed to fetch failed notifications")
	}

	// Manual pagination since this uses a custom query
	total := int64(len(notifications))
	start := (page - 1) * limit
	end := start + limit

	if start >= len(notifications) {
		return []*model.Notification{}, total, nil
	}
	if end > len(notifications) {
		end = len(notifications)
	}

	paginatedNotifications := notifications[start:end]

	zap.L().Info("Fetched failed notifications with retries",
		zap.Int("count", len(paginatedNotifications)),
		zap.Int64("total", total))

	return paginatedNotifications, total, nil
}

// GetByFilters retrieves notifications with multiple filter criteria
func (s *NotificationService) GetByFilters(
	ctx context.Context,
	userID *uuid.UUID,
	notificationType *enum.NotificationType,
	status *enum.NotificationStatus,
	startDate, endDate *string,
	page, limit int,
) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching notifications with filters",
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Validate enums if provided
	if notificationType != nil && !notificationType.IsValid() {
		return nil, 0, errors.New("invalid notification type")
	}
	if status != nil && !status.IsValid() {
		return nil, 0, errors.New("invalid notification status")
	}

	// Parse dates if provided
	var startTime, endTime *time.Time
	if startDate != nil && *startDate != "" {
		t, err := time.Parse(time.RFC3339, *startDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *startDate)
			if err != nil {
				return nil, 0, errors.New("invalid start date format, use RFC3339 or YYYY-MM-DD")
			}
		}
		startTime = &t
	}
	if endDate != nil && *endDate != "" {
		t, err := time.Parse(time.RFC3339, *endDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *endDate)
			if err != nil {
				return nil, 0, errors.New("invalid end date format, use RFC3339 or YYYY-MM-DD")
			}
		}
		endTime = &t
	}

	notifications, total, err := s.notificationRepo.GetAll(
		ctx,
		func(db *gorm.DB) *gorm.DB {
			query := db

			if userID != nil {
				query = query.Where("user_id = ?", *userID)
			}
			if notificationType != nil {
				query = query.Where("type = ?", *notificationType)
			}
			if status != nil {
				query = query.Where("status = ?", *status)
			}
			if startTime != nil {
				query = query.Where("created_at >= ?", *startTime)
			}
			if endTime != nil {
				query = query.Where("created_at <= ?", *endTime)
			}

			return query.Order("created_at DESC")
		},
		nil,
		limit,
		page,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications with filters", zap.Error(err))
		return nil, 0, errors.New("failed to fetch notifications")
	}

	zap.L().Info("Fetched notifications with filters",
		zap.Int("count", len(notifications)),
		zap.Int64("total", total))

	// Convert []model.Notification to []*model.Notification
	notificationPtrs := make([]*model.Notification, len(notifications))
	for i := range notifications {
		notificationPtrs[i] = &notifications[i]
	}

	return notificationPtrs, total, nil
}
