package dtos

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

type MilestoneDTO struct {
	ID                   uuid.UUID            `json:"id" gorm:"column:id"`
	Description          *string              `json:"description" gorm:"column:description"`
	DueDate              string               `json:"due_date" gorm:"column:due_date;not null"`
	CompletedAt          *time.Time           `json:"completed_at" gorm:"column:completed_at"`
	CompletionPercentage float64              `json:"completion_percentage" gorm:"column:completion_percentage"`
	Status               enum.MilestoneStatus `json:"status" gorm:"column:status"`
	BehindSchedule       bool                 `json:"behind_schedule" gorm:"column:behind_schedule"`
}
