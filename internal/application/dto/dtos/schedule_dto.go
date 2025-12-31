package dtos

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

// ScheduleDTO represents schedule data from database query with nested details
type ScheduleDTO struct {
	ScheduleID    uuid.UUID           `json:"schedule_id" gorm:"column:schedule_id"`
	ReferenceID   uuid.UUID           `json:"reference_id" gorm:"column:reference_id"`
	ReferenceType *enum.ReferenceType `json:"reference_type,omitempty" gorm:"column:reference_type"`
	Type          enum.ScheduleType   `json:"type" gorm:"column:type"`
	ScheduledAt   time.Time           `json:"scheduled_at" gorm:"column:scheduled_at"`
	Status        enum.ScheduleStatus `json:"status" gorm:"column:status"`
	RetryCount    int                 `json:"retry_count" gorm:"column:retry_count"`
	LastError     *string             `json:"last_error,omitempty" gorm:"column:last_error"`
	ExecutedAt    *time.Time          `json:"executed_at,omitempty" gorm:"column:executed_at"`
	CreatedAt     time.Time           `json:"created_at" gorm:"column:created_at"`
	CreatedBy     uuid.UUID           `json:"created_by" gorm:"column:created_by"`
	CreatedByName string              `json:"created_by_name,omitempty" gorm:"column:created_by_name"`
	UpdatedAt     *time.Time          `json:"updated_at,omitempty" gorm:"column:updated_at"`

	// Nested details based on schedule type
	ContentDetails      *ContentScheduleDetails      `json:"content_details,omitempty" gorm:"-"`
	NotificationDetails *NotificationScheduleDetails `json:"notification_details,omitempty" gorm:"-"`
}

// ContentScheduleDetails contains content-specific scheduling information
type ContentScheduleDetails struct {
	ContentID        uuid.UUID        `json:"content_id"`
	ContentChannelID uuid.UUID        `json:"content_channel_id"`
	ChannelID        uuid.UUID        `json:"channel_id"`
	ContentTitle     string           `json:"content_title"`
	ContentType      enum.ContentType `json:"content_type"`
	ChannelName      string           `json:"channel_name"`
	ChannelCode      string           `json:"channel_code"`
	ThumbnailURL     *string          `json:"thumbnail_url,omitempty"`
}

// NotificationScheduleDetails contains notification-specific scheduling information
type NotificationScheduleDetails struct {
	ID             uuid.UUID             `json:"id"`
	ToUserID       uuid.UUID             `json:"to_user_id"`
	ToUserFullName *string               `json:"to_user_fullname,omitempty"`
	ToUserEmail    *string               `json:"to_user_email,omitempty"`
	Title          string                `json:"title"`
	Type           enum.NotificationType `json:"type"`
	IsRead         bool                  `json:"is_read"`
}

// ContentScheduleRawDTO is used for raw SQL queries with content joins
type ContentScheduleRawDTO struct {
	ScheduleDTO
	ContentID    uuid.UUID        `gorm:"column:content_id"`
	ContentTitle string           `gorm:"column:content_title"`
	ContentType  enum.ContentType `gorm:"column:content_type"`
	ChannelID    uuid.UUID        `gorm:"column:channel_id"`
	ChannelName  string           `gorm:"column:channel_name"`
	ChannelCode  string           `gorm:"column:channel_code"`
	ThumbnailURL *string          `gorm:"column:thumbnail_url"`
}

// ToScheduleDTO converts raw DTO to structured ScheduleDTO
func (r *ContentScheduleRawDTO) ToScheduleDTO() *ScheduleDTO {
	dto := r.ScheduleDTO
	dto.ContentDetails = &ContentScheduleDetails{
		ContentChannelID: r.ReferenceID,
		ContentID:        r.ContentID,
		ContentTitle:     r.ContentTitle,
		ContentType:      r.ContentType,
		ChannelID:        r.ChannelID,
		ChannelName:      r.ChannelName,
		ChannelCode:      r.ChannelCode,
		ThumbnailURL:     r.ThumbnailURL,
	}
	return &dto
}
