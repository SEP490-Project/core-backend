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
	MaxRetries    int                 `json:"max_retries" gorm:"column:max_retries"`
	LastError     *string             `json:"last_error,omitempty" gorm:"column:last_error"`
	ExecutedAt    *time.Time          `json:"executed_at,omitempty" gorm:"column:executed_at"`
	CreatedAt     time.Time           `json:"created_at" gorm:"column:created_at"`
	CreatedBy     uuid.UUID           `json:"created_by" gorm:"column:created_by"`
	CreatedByName string              `json:"created_by_name,omitempty" gorm:"column:created_by_name"`
	UpdatedAt     *time.Time          `json:"updated_at,omitempty" gorm:"column:updated_at"`

	// Nested details based on schedule type
	ContentDetails  *ContentScheduleDetails  `json:"content_details,omitempty" gorm:"-"`
	ContractDetails *ContractScheduleDetails `json:"contract_details,omitempty" gorm:"-"`
}

// ContentScheduleDetails contains content-specific scheduling information
type ContentScheduleDetails struct {
	ContentChannelID uuid.UUID        `json:"content_channel_id"`
	ContentID        uuid.UUID        `json:"content_id"`
	ContentTitle     string           `json:"content_title"`
	ContentType      enum.ContentType `json:"content_type"`
	ChannelID        uuid.UUID        `json:"channel_id"`
	ChannelName      string           `json:"channel_name"`
	ChannelCode      string           `json:"channel_code"`
	Platform         string           `json:"platform,omitempty"`
	ThumbnailURL     *string          `json:"thumbnail_url,omitempty"`
}

// ContractScheduleDetails contains contract-specific scheduling information
type ContractScheduleDetails struct {
	ContractID     uuid.UUID `json:"contract_id"`
	ContractNumber string    `json:"contract_number"`
	BrandID        uuid.UUID `json:"brand_id"`
	BrandName      string    `json:"brand_name"`
}

// ContentScheduleRawDTO is used for raw SQL queries with content joins
type ContentScheduleRawDTO struct {
	ScheduleID       uuid.UUID           `gorm:"column:schedule_id"`
	ReferenceID      uuid.UUID           `gorm:"column:reference_id"`
	ReferenceType    *enum.ReferenceType `gorm:"column:reference_type"`
	Type             enum.ScheduleType   `gorm:"column:type"`
	ScheduledAt      time.Time           `gorm:"column:scheduled_at"`
	Status           enum.ScheduleStatus `gorm:"column:status"`
	RetryCount       int                 `gorm:"column:retry_count"`
	MaxRetries       int                 `gorm:"column:max_retries"`
	LastError        *string             `gorm:"column:last_error"`
	ExecutedAt       *time.Time          `gorm:"column:executed_at"`
	CreatedAt        time.Time           `gorm:"column:created_at"`
	CreatedBy        uuid.UUID           `gorm:"column:created_by"`
	CreatedByName    string              `gorm:"column:created_by_name"`
	UpdatedAt        *time.Time          `gorm:"column:updated_at"`
	ContentChannelID uuid.UUID           `gorm:"column:content_channel_id"`
	ContentID        uuid.UUID           `gorm:"column:content_id"`
	ContentTitle     string              `gorm:"column:content_title"`
	ContentType      enum.ContentType    `gorm:"column:content_type"`
	ChannelID        uuid.UUID           `gorm:"column:channel_id"`
	ChannelName      string              `gorm:"column:channel_name"`
	ChannelCode      string              `gorm:"column:channel_code"`
	Platform         string              `gorm:"column:platform"`
	ThumbnailURL     *string             `gorm:"column:thumbnail_url"`
}

// ToScheduleDTO converts raw DTO to structured ScheduleDTO
func (r *ContentScheduleRawDTO) ToScheduleDTO() *ScheduleDTO {
	return &ScheduleDTO{
		ScheduleID:    r.ScheduleID,
		ReferenceID:   r.ReferenceID,
		ReferenceType: r.ReferenceType,
		Type:          r.Type,
		ScheduledAt:   r.ScheduledAt,
		Status:        r.Status,
		RetryCount:    r.RetryCount,
		MaxRetries:    r.MaxRetries,
		LastError:     r.LastError,
		ExecutedAt:    r.ExecutedAt,
		CreatedAt:     r.CreatedAt,
		CreatedBy:     r.CreatedBy,
		CreatedByName: r.CreatedByName,
		UpdatedAt:     r.UpdatedAt,
		ContentDetails: &ContentScheduleDetails{
			ContentChannelID: r.ContentChannelID,
			ContentID:        r.ContentID,
			ContentTitle:     r.ContentTitle,
			ContentType:      r.ContentType,
			ChannelID:        r.ChannelID,
			ChannelName:      r.ChannelName,
			ChannelCode:      r.ChannelCode,
			Platform:         r.Platform,
			ThumbnailURL:     r.ThumbnailURL,
		},
	}
}
