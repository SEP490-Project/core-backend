package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Milestone struct {
	ID                   uuid.UUID            `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	CampaignID           uuid.UUID            `json:"campaign_id" gorm:"type:uuid;column:campaign_id;not null"`
	Description          *string              `json:"description" gorm:"type:text;column:description"`
	BudgetPercent        int                  `json:"budget_percent" gorm:"column:budget_percent"`
	BudgetAmount         float64              `json:"budget_amount" gorm:"column:budget_amount"`
	DueDate              time.Time            `json:"due_date" gorm:"type:timestamp;column:due_date;not null"`
	CompletedAt          *time.Time           `json:"completed_at" gorm:"type:timestamp;column:completed_at"`
	CompletionPercentage float64              `json:"completion_percentage" gorm:"column:completion_percentage"`
	Status               enum.MilestoneStatus `json:"status" gorm:"column:status;not null;check:status in ('NOT_STARTED', 'ON_GOING', 'CANCELLED', 'COMPLETED')"`
	BehindSchedule       bool                 `json:"behind_schedule" gorm:"column:behind_schedule"`
	CreatedAt            time.Time            `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt            time.Time            `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt            gorm.DeletedAt       `json:"deleted_at" gorm:"column:deleted_at;index"`
	CreatedByID          uuid.UUID            `json:"created_by" gorm:"type:uuid;column:created_by;not null"`
	UpdatedByID          *uuid.UUID           `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	// Relationships
	Campaign *Campaign `json:"-" gorm:"foreignKey:CampaignID"`
	Tasks    []*Task   `json:"-" gorm:"foreignKey:MilestoneID"`
}

func (Milestone) TableName() string { return "milestones" }

func (m *Milestone) BeforeCreate(tx *gorm.DB) (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	if m.CompletionPercentage < 0 {
		zap.L().Warn("CompletionPercentage is less than 0, setting to 0")
		m.CompletionPercentage = 0
	} else if m.CompletionPercentage > 100 {
		zap.L().Warn("CompletionPercentage is greater than 100, setting to 100")
		m.CompletionPercentage = 100
	}
	return
}
