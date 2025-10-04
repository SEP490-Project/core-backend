package responses

import "core-backend/internal/domain/model"

type CampaignResponse struct {
	ID              string  `json:"id"`
	ContractID      string  `json:"contract_id"`
	ContractTitle   string  `json:"contract_title"`
	ContractNumber  string  `json:"contract_number"`
	Name            string  `json:"name"`
	Description     *string `json:"description"`
	StartDate       string  `json:"start_date"`
	EndDate         string  `json:"end_date"`
	Status          string  `json:"status"`
	BudgetProjected float64 `json:"budget_projected"`
	BudgetActual    float64 `json:"budget_actual"`
	Type            string  `json:"type"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

func (cr CampaignResponse) ToCampaignResponse(model *model.Campaign) *CampaignResponse {
	return &CampaignResponse{
		ID:              model.ID.String(),
		ContractID:      model.ContractID.String(),
		ContractTitle:   *model.Contract.Title,
		ContractNumber:  *model.Contract.ContractNumber,
		Name:            model.Name,
		Description:     model.Description,
		StartDate:       model.StartDate.String(),
		EndDate:         model.EndDate.String(),
		Status:          model.Status.String(),
		BudgetProjected: model.BudgetProjected,
		BudgetActual:    model.BudgetActual,
		Type:            model.Type.String(),
		CreatedAt:       model.CreatedAt.String(),
		UpdatedAt:       model.UpdatedAt.String(),
	}
}
