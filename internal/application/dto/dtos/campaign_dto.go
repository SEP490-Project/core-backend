package dtos

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

type CampaignDTO struct {
	ID          uuid.UUID           `json:"id" gorm:"column:id;primaryKey"`
	Name        string              `json:"name" gorm:"column:name"`
	Description *string             `json:"description" gorm:"column:description"`
	StartDate   string              `json:"start_date" gorm:"column:start_date"`
	EndDate     string              `json:"end_date" gorm:"column:end_date"`
	Status      enum.CampaignStatus `json:"status" gorm:"column:status;not null"`
	Type        enum.ContractType   `json:"type" gorm:"column:type;not null"`
}
