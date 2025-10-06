package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Campaign struct {
	ID              uuid.UUID           `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ContractID      uuid.UUID           `json:"contract_id" gorm:"type:uuid;column:contract_id;not null"`
	Name            string              `json:"name" gorm:"type:varchar(255);column:name;not null"`
	Description     *string             `json:"description" gorm:"type:text;column:description"`
	StartDate       time.Time           `json:"start_date" gorm:"type:timestamp;column:start_date;not null"`
	EndDate         time.Time           `json:"end_date" gorm:"type:timestamp;column:end_date;not null"`
	Status          enum.CampaignStatus `json:"status" gorm:"type:varchar(50);column:status;not null;check:status IN ('RUNNING','COMPLETED','CANCELED')"`
	BudgetProjected float64             `json:"budget_projected" gorm:"not null"`
	BudgetActual    float64             `json:"budget_actual" gorm:"not null"`
	Type            enum.ContractType   `json:"type" gorm:"type:varchar(50);column:type;not null;check:type IN ('ADVERTISING','AFFILIATE','AMBASSADOR','COPRODUCE')"`
	CreatedAt       time.Time           `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time           `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt       gorm.DeletedAt      `json:"deleted_at" gorm:"column:deleted_at;index"`
	CreatedByID     uuid.UUID           `json:"created_by" gorm:"type:uuid;column:created_by;not null"`
	UpdatedByID     *uuid.UUID          `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	// Relationships
	Contract   *Contract    `json:"-" gorm:"foreignKey:ContractID"`
	Milestones []*Milestone `json:"milestones" gorm:"foreignKey:CampaignID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (Campaign) TableName() string { return "campaigns" }

func (c *Campaign) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.BudgetProjected < 0 {
		zap.L().Warn("BudgetProjected is less than 0, setting to 0")
		c.BudgetProjected = 0
	}
	if c.BudgetActual < 0 {
		zap.L().Warn("BudgetActual is less than 0, setting to 0")
		c.BudgetActual = 0
	}

	return nil
}
