package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContentSchedule represents a scheduled content publishing job
// Uses RabbitMQ delayed message plugin for execution
type ContentSchedule struct {
	ID               uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;column:id"`
	ContentChannelID uuid.UUID           `json:"content_channel_id" gorm:"type:uuid;not null;uniqueIndex;column:content_channel_id"`
	ScheduledAt      time.Time           `json:"scheduled_at" gorm:"not null;index;column:scheduled_at"`
	Status           enum.ScheduleStatus `json:"status" gorm:"type:varchar(30);not null;default:'PENDING';column:status"`
	RetryCount       int                 `json:"retry_count" gorm:"default:0;column:retry_count"`
	LastError        *string             `json:"last_error,omitempty" gorm:"type:text;column:last_error"`
	ExecutedAt       *time.Time          `json:"executed_at,omitempty" gorm:"column:executed_at"`
	CreatedAt        *time.Time          `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt        *time.Time          `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	CreatedBy        uuid.UUID           `json:"created_by" gorm:"type:uuid;not null;column:created_by"`

	// Relationships
	ContentChannel *ContentChannel `json:"content_channel,omitempty" gorm:"foreignKey:ContentChannelID;constraint:OnDelete:CASCADE"`
	Creator        *User           `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL"`
}

func (ContentSchedule) TableName() string { return "content_schedules" }

func (cs *ContentSchedule) BeforeCreate(_ *gorm.DB) error {
	if cs.ID == uuid.Nil {
		cs.ID = uuid.New()
	}
	return nil
}

// IsPending checks if the schedule is pending
func (cs *ContentSchedule) IsPending() bool {
	return cs.Status == enum.ScheduleStatusPending
}

// IsCompleted checks if the schedule is completed
func (cs *ContentSchedule) IsCompleted() bool {
	return cs.Status == enum.ScheduleStatusCompleted
}

// IsCancelled checks if the schedule is cancelled
func (cs *ContentSchedule) IsCancelled() bool {
	return cs.Status == enum.ScheduleStatusCancelled
}

// CanRetry checks if the schedule can be retried
func (cs *ContentSchedule) CanRetry(maxRetries int) bool {
	return cs.RetryCount < maxRetries && cs.Status != enum.ScheduleStatusCancelled
}

// MarkProcessing marks the schedule as processing
func (cs *ContentSchedule) MarkProcessing() {
	cs.Status = enum.ScheduleStatusProcessing
}

// MarkCompleted marks the schedule as completed
func (cs *ContentSchedule) MarkCompleted() {
	now := time.Now()
	cs.Status = enum.ScheduleStatusCompleted
	cs.ExecutedAt = &now
}

// MarkFailed marks the schedule as failed with error
func (cs *ContentSchedule) MarkFailed(err string) {
	cs.Status = enum.ScheduleStatusFailed
	cs.LastError = &err
}

// MarkCancelled marks the schedule as cancelled
func (cs *ContentSchedule) MarkCancelled() {
	cs.Status = enum.ScheduleStatusCancelled
}

// IncrementRetry increments the retry count and records the error
func (cs *ContentSchedule) IncrementRetry(err string) {
	cs.RetryCount++
	cs.LastError = &err
	cs.Status = enum.ScheduleStatusPending
}
