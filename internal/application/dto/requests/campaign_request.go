package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CreateCampaignRequest struct {
	ContractID      string    `json:"contract_id" validate:"required,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name            string    `json:"name" validate:"required,min=3,max=255" example:"Summer Sale Campaign"`
	Description     *string   `json:"description" validate:"omitempty,max=1000" example:"A campaign for the summer sale."`
	StartDate       time.Time `json:"start_date" validate:"required" example:"2023-06-01T00:00:00Z"`
	EndDate         time.Time `json:"end_date" validate:"required,gtfield=StartDate" example:"2023-08-31T23:59:59Z"`
	BudgetProjected float64   `json:"budget_projected" validate:"required,gte=0" example:"10000.50"`
	BudgetActual    float64   `json:"budget_actual" validate:"required,gte=0" example:"5000.25"`
	Type            string    `json:"type" validate:"required,oneof=ADVERTISING AFFILIATE AMBASSADOR COPRODUCE" example:"ADVERTISING"`
}

func (ccr *CreateCampaignRequest) ToModel() (*model.Campaign, error) {
	contractID, err := uuid.Parse(ccr.ContractID)
	if err != nil {
		zap.L().Error("Failed to parse ContractID", zap.Error(err))
		return nil, err
	}
	campaignType := enum.ContractType(ccr.Type)
	if !campaignType.IsValid() {
		zap.L().Error("Invalid campaign type", zap.String("type", ccr.Type))
		return nil, errors.New("invalid campaign type")
	}

	return &model.Campaign{
		ID:              uuid.New(),
		ContractID:      contractID,
		Name:            ccr.Name,
		Description:     ccr.Description,
		StartDate:       ccr.StartDate,
		EndDate:         ccr.EndDate,
		BudgetProjected: ccr.BudgetProjected,
		BudgetActual:    ccr.BudgetActual,
		Status:          enum.CampaignStatusRunning,
		Type:            campaignType,
	}, nil
}
