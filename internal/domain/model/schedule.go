package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Schedule represents a scheduled job
// Uses Asynq or RabbitMQ delayed message plugin for execution
type Schedule struct {
	ID            uuid.UUID           `json:"id" gorm:"type:uuid;primaryKey;column:id"`
	ReferenceID   *uuid.UUID          `json:"reference_id" gorm:"type:uuid;index;column:reference_id"`
	ReferenceType *enum.ReferenceType `json:"reference_type" gorm:"type:varchar(50);index;column:reference_type"`
	Type          enum.ScheduleType   `json:"type" gorm:"type:varchar(50);not null;index;column:type"`
	ScheduledAt   time.Time           `json:"scheduled_at" gorm:"not null;index;column:scheduled_at"`
	Status        enum.ScheduleStatus `json:"status" gorm:"type:varchar(30);not null;default:'PENDING';column:status"`
	RetryCount    int                 `json:"retry_count" gorm:"default:0;column:retry_count"`
	LastError     *string             `json:"last_error,omitempty" gorm:"type:text;column:last_error"`
	ExecutedAt    *time.Time          `json:"executed_at,omitempty" gorm:"column:executed_at"`
	CreatedAt     *time.Time          `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt     *time.Time          `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
	CreatedBy     uuid.UUID           `json:"created_by" gorm:"type:uuid;not null;column:created_by"`
	Metadata      datatypes.JSON      `json:"metadata,omitempty" gorm:"type:jsonb;column:metadata"`

	// Relationships
	Creator *User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL"`
}

func (Schedule) TableName() string { return "schedules" }

func (s *Schedule) BeforeCreate(_ *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// region: ============== Schedule Methods ==============

// IsPending checks if the schedule is pending
func (s *Schedule) IsPending() bool {
	return s.Status == enum.ScheduleStatusPending
}

// IsCompleted checks if the schedule is completed
func (s *Schedule) IsCompleted() bool {
	return s.Status == enum.ScheduleStatusCompleted
}

// IsCancelled checks if the schedule is cancelled
func (s *Schedule) IsCancelled() bool {
	return s.Status == enum.ScheduleStatusCancelled
}

// CanRetry checks if the schedule can be retried
func (s *Schedule) CanRetry(maxRetries int) bool {
	return s.Status == enum.ScheduleStatusFailed && s.RetryCount < maxRetries
}

// MarkProcessing marks the schedule as processing
func (s *Schedule) MarkProcessing() {
	s.Status = enum.ScheduleStatusProcessing
}

// MarkCompleted marks the schedule as completed
func (s *Schedule) MarkCompleted() {
	s.Status = enum.ScheduleStatusCompleted
	now := time.Now()
	s.ExecutedAt = &now
}

// MarkFailed marks the schedule as failed with error
func (s *Schedule) MarkFailed(err string) {
	s.Status = enum.ScheduleStatusFailed
	s.LastError = &err
}

// MarkCancelled marks the schedule as cancelled
func (s *Schedule) MarkCancelled() {
	s.Status = enum.ScheduleStatusCancelled
}

// IncrementRetry increments the retry count and records the error
func (s *Schedule) IncrementRetry(err string) {
	s.RetryCount++
	s.LastError = &err
}

// endregion
