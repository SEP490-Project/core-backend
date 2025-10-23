package requests

import "github.com/google/uuid"

// CampaignSuggestionRequest represents a request to generate campaign suggestions from a contract
type CampaignSuggestionRequest struct {
	ContractID uuid.UUID `json:"contract_id" binding:"required" validate:"required,uuid"`
}
