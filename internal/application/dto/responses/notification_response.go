package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// NotificationResponse represents a single notification with delivery details
type NotificationResponse struct {
	ID               uuid.UUID                 `json:"id"`
	UserID           uuid.UUID                 `json:"user_id"`
	Type             enum.NotificationType     `json:"type"`
	Status           enum.NotificationStatus   `json:"status"`
	DeliveryAttempts []model.DeliveryAttempt   `json:"delivery_attempts"`
	RecipientInfo    model.JSONBRecipientInfo  `json:"recipient_info"`
	ContentData      model.JSONBContentData    `json:"content_data"`
	PlatformConfig   model.JSONBPlatformConfig `json:"platform_config,omitempty"`
	ErrorDetails     model.JSONBErrorDetails   `json:"error_details,omitempty"`
	CreatedAt        *time.Time                `json:"created_at,omitempty"`
	UpdatedAt        *time.Time                `json:"updated_at,omitempty"`
}

// NotificationListResponse represents a paginated list of notifications
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	Pagination    Pagination             `json:"pagination"`
}

// ToNotificationResponse converts a notification model to response DTO
func ToNotificationResponse(notification *model.Notification) NotificationResponse {
	return NotificationResponse{
		ID:               notification.ID,
		UserID:           notification.UserID,
		Type:             notification.Type,
		Status:           notification.Status,
		DeliveryAttempts: notification.DeliveryAttempts,
		RecipientInfo:    notification.RecipientInfo,
		ContentData:      notification.ContentData,
		PlatformConfig:   notification.PlatformConfig,
		ErrorDetails:     notification.ErrorDetails,
		CreatedAt:        notification.CreatedAt,
		UpdatedAt:        notification.UpdatedAt,
	}
}

// ToNotificationListResponse converts a slice of notifications to list response DTO
func ToNotificationListResponse(notifications []*model.Notification, page, limit int, total int64) NotificationListResponse {
	notificationResponses := make([]NotificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		notificationResponses = append(notificationResponses, ToNotificationResponse(notification))
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	return NotificationListResponse{
		Notifications: notificationResponses,
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    hasNext,
			HasPrev:    hasPrev,
		},
	}
}
