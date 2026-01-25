package responses

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

// SuggestedTask represents a suggested task extracted from contract deliverables
type SuggestedTask struct {
	Name              string         `json:"name" example:"Create social media post"`
	Description       map[string]any `json:"description,omitempty"`
	Deadline          time.Time      `json:"deadline" example:"2024-12-31T23:59:59Z"`
	Type              enum.TaskType  `json:"type" example:"CONTENT"`
	ScopeOfWorkItemID *string        `json:"scope_of_work_item_id,omitempty" example:"SOW-12345"`
	AssignedToID      *uuid.UUID     `json:"assigned_to_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// SuggestedMilestone represents a suggested milestone with its tasks
type SuggestedMilestone struct {
	Description string          `json:"description,omitempty" example:"Initial launch of the campaign"`
	DueDate     time.Time       `json:"due_date" example:"2024-12-31T23:59:59Z"`
	Tasks       []SuggestedTask `json:"tasks"`
}

// SuggestedCampaign represents the campaign structure extracted from a contract
type SuggestedCampaign struct {
	Name        string               `json:"name" example:"New Product Launch Campaign"`
	Description *string              `json:"description,omitempty" example:"Campaign for launching the new product line."`
	StartDate   time.Time            `json:"start_date" example:"2024-11-01T00:00:00Z"`
	EndDate     time.Time            `json:"end_date" example:"2024-12-31T23:59:59Z"`
	Type        enum.ContractType    `json:"type,omitempty" example:"ADVERTISING"`
	ContractID  uuid.UUID            `json:"contract_id"`
	Milestones  []SuggestedMilestone `json:"milestones"`
}

// CampaignSuggestionResponse represents the response for campaign suggestion from a contract
type CampaignSuggestionResponse struct {
	ContractID              uuid.UUID          `json:"contract_id"`
	ContractType            string             `json:"contract_type"`
	SuggestedCampaign       *SuggestedCampaign `json:"suggested_campaign"`
	ScopeOfWorkDeliverables any                `json:"scope_of_work_deliverables,omitempty"`
}
