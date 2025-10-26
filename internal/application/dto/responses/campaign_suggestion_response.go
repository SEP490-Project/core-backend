package responses

import (
	"github.com/google/uuid"
)

// SuggestedTask represents a suggested task extracted from contract deliverables
type SuggestedTask struct {
	Name            string                 `json:"name"`
	DescriptionJSON map[string]interface{} `json:"description_json,omitempty"`
}

// SuggestedMilestone represents a suggested milestone with its tasks
type SuggestedMilestone struct {
	Name  string          `json:"name"`
	Tasks []SuggestedTask `json:"tasks"`
}

// SuggestedCampaign represents the campaign structure extracted from a contract
type SuggestedCampaign struct {
	Name       string               `json:"name"`
	Milestones []SuggestedMilestone `json:"milestones"`
}

// CampaignSuggestionResponse represents the response for campaign suggestion from a contract
type CampaignSuggestionResponse struct {
	ContractID        uuid.UUID          `json:"contract_id"`
	ContractType      string             `json:"contract_type"`
	SuggestedCampaign *SuggestedCampaign `json:"suggested_campaign"`
}
