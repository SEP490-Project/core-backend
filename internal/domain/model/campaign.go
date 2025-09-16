package model

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Campaign struct {
	ID              uuid.UUID           `json:"id" gorm:"primaryKey"`
	ContractID      uuid.UUID           `json:"contract_id" gorm:"not null"`
	Name            string              `json:"name" gorm:"not null"`
	Description     string              `json:"description"`
	StartDate       string              `json:"start_date" gorm:"not null"`
	EndDate         string              `json:"end_date" gorm:"not null"`
	Status          enum.CampaignStatus `json:"status" gorm:"type:enum('RUNNING','COMPLETED','CANCELED');not null"`
	BudgetProjected float64             `json:"budget_projected" gorm:"not null"`
	BudgetActual    float64             `json:"budget_actual" gorm:"not null"`
	Type            enum.ContractType   `json:"type" gorm:"type:enum('ADVERTISING','AFFILIATE','AMBASSADOR','COPRODUCE');not null"`
	CreatedAt       int64               `json:"created_at" gorm:"autoCreateTime"`
}

func (c *Campaign) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
