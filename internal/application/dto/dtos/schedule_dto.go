package dtos

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

// ScheduleDTO represents schedule data from database query
type ScheduleDTO struct {
	ScheduleID       uuid.UUID           `json:"schedule_id" gorm:"column:schedule_id"`
	ReferenceID      uuid.UUID           `json:"reference_id" gorm:"column:reference_id"`
	Type             enum.ScheduleType   `json:"type" gorm:"column:type"`
	ContentChannelID uuid.UUID           `json:"content_channel_id,omitempty" gorm:"column:content_channel_id"`
	ContentID        uuid.UUID           `json:"content_id,omitempty" gorm:"column:content_id"`
	ContentTitle     string              `json:"content_title,omitempty" gorm:"column:content_title"`
	ContentType      enum.ContentType    `json:"content_type,omitempty" gorm:"column:content_type"`
	ChannelID        uuid.UUID           `json:"channel_id,omitempty" gorm:"column:channel_id"`
	ChannelName      string              `json:"channel_name,omitempty" gorm:"column:channel_name"`
	ChannelCode      string              `json:"channel_code,omitempty" gorm:"column:channel_code"`
	ScheduledAt      time.Time           `json:"scheduled_at" gorm:"column:scheduled_at"`
	Status           enum.ScheduleStatus `json:"status" gorm:"column:status"`
	RetryCount       int                 `json:"retry_count" gorm:"column:retry_count"`
	LastError        *string             `json:"last_error" gorm:"column:last_error"`
	ExecutedAt       *time.Time          `json:"executed_at" gorm:"column:executed_at"`
	CreatedAt        time.Time           `json:"created_at" gorm:"column:created_at"`
	CreatedBy        uuid.UUID           `json:"created_by" gorm:"column:created_by"`
	CreatedByName    string              `json:"created_by_name" gorm:"column:created_by_name"`
}
