package responses

import (
	"github.com/google/uuid"
)

// SuggestedTask represents a suggested task extracted from contract deliverables
type SuggestedTask struct {
	Name        string         `json:"name" example:"Create social media post"`
	Description map[string]any `json:"description_json,omitempty" example:"{\"details\":\"Create engaging social media content.\",\"resources\":[\"link1\",\"link2\"]}"`
	Deadline    string         `json:"deadline,omitempty" example:"2024-12-31T23:59:59Z"`
	Type        string         `json:"type,omitempty" example:"CONTENT"`
}

// SuggestedMilestone represents a suggested milestone with its tasks
type SuggestedMilestone struct {
	Description string          `json:"description,omitempty" example:"Initial launch of the campaign"`
	DueDate     string          `json:"due_date,omitempty" example:"2024-12-31T23:59:59Z"`
	Tasks       []SuggestedTask `json:"tasks"`
}

// SuggestedCampaign represents the campaign structure extracted from a contract
type SuggestedCampaign struct {
	Name        string               `json:"name" example:"New Product Launch Campaign"`
	Description string               `json:"description,omitempty" example:"Campaign for launching the new product line."`
	StartDate   string               `json:"start_date,omitempty" example:"2024-11-01T00:00:00Z"`
	EndDate     string               `json:"end_date,omitempty" example:"2024-12-31T23:59:59Z"`
	Type        string               `json:"type,omitempty" example:"ADVERTISING"`
	Milestones  []SuggestedMilestone `json:"milestones"`
}

// CampaignSuggestionResponse represents the response for campaign suggestion from a contract
type CampaignSuggestionResponse struct {
	ContractID        uuid.UUID          `json:"contract_id"`
	ContractType      string             `json:"contract_type"`
	SuggestedCampaign *SuggestedCampaign `json:"suggested_campaign"`
}
