package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"sync"

	"go.uber.org/zap"
)

// CampaignInfoResponse represents the basic information of a campaign.
type CampaignInfoResponse struct {
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

// CampaignDetailsResponse represents the details of a campaign.
type CampaignDetailsResponse struct {
	ID                  string                  `json:"id"`
	ContractID          string                  `json:"contract_id"`
	ContractTitle       string                  `json:"contract_title"`
	ContractNumber      string                  `json:"contract_number"`
	Name                string                  `json:"name"`
	Description         *string                 `json:"description"`
	StartDate           string                  `json:"start_date"`
	EndDate             string                  `json:"end_date"`
	Status              string                  `json:"status"`
	BudgetProjected     float64                 `json:"budget_projected"`
	BudgetActual        float64                 `json:"budget_actual"`
	Type                string                  `json:"type"`
	Milestones          []CampaignMilestoneInfo `json:"milestones"`
	NumberOfTasks       int                     `json:"number_of_tasks"`
	PercentageCompleted float64                 `json:"percentage_completed"`
	CreatedAt           string                  `json:"created_at"`
	UpdatedAt           string                  `json:"updated_at"`
}

// CampaignMilestoneInfo represents the details of milestone within a campaign.
type CampaignMilestoneInfo struct {
	ID                   string  `json:"id,omitempty"`
	Description          string  `json:"description,omitempty"`
	DueDate              string  `json:"due_date,omitempty"`
	CompletedAt          string  `json:"completed_at,omitempty"`
	CompletionPercentage float64 `json:"completion_percentage,omitempty"`
	Status               string  `json:"status,omitempty"`
	BehindSchedule       bool    `json:"behind_schedule,omitempty"`
	NumberOfTasks        int     `json:"number_of_tasks,omitempty"`
}

// ToCampaignInfoResponse maps a Campaign model to a CampaignInfoResponse DTO.
// Only need basic info of the Campaign model
func (cir CampaignInfoResponse) ToCampaignInfoResponse(model *model.Campaign) *CampaignInfoResponse {
	return &CampaignInfoResponse{
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

func (cir CampaignInfoResponse) ToCampaignInfoResponseList(models []model.Campaign) []*CampaignInfoResponse {
	if len(models) == 0 {
		zap.L().Warn("No campaigns found to map to CampaignInfoResponse")
		return []*CampaignInfoResponse{}
	}
	responses := make([]*CampaignInfoResponse, 0, len(models))
	for _, model := range models {
		responses = append(responses, cir.ToCampaignInfoResponse(&model))
	}
	return responses
}

// ToCampaignDetailsResponse maps a Campaign model to a CampaignDetailsResponse DTO.
// This includes detailed info from milestones and tasks
func (cdr CampaignDetailsResponse) ToCampaignDetailsResponse(model *model.Campaign) *CampaignDetailsResponse {
	var milestones []CampaignMilestoneInfo
	if len(model.Milestones) == 0 {
		zap.L().Warn("Campaign has no milestones", zap.String("campaign_id", model.ID.String()))
	} else {
		milestones = cdr.ToCampaignMilestoneInfoList(model)
	}
	totalTasks, percentageCompleted := cdr.CalculateCampaignProgress(model)

	return &CampaignDetailsResponse{
		ID:                  model.ID.String(),
		ContractID:          model.ContractID.String(),
		ContractTitle:       *model.Contract.Title,
		ContractNumber:      *model.Contract.ContractNumber,
		Name:                model.Name,
		Description:         model.Description,
		StartDate:           model.StartDate.String(),
		EndDate:             model.EndDate.String(),
		Status:              model.Status.String(),
		BudgetProjected:     model.BudgetProjected,
		BudgetActual:        model.BudgetActual,
		Type:                model.Type.String(),
		Milestones:          milestones,
		NumberOfTasks:       totalTasks,
		PercentageCompleted: percentageCompleted,
		CreatedAt:           model.CreatedAt.String(),
		UpdatedAt:           model.UpdatedAt.String(),
	}
}

func (cdr CampaignDetailsResponse) ToCampaignMilestoneInfoList(campaign *model.Campaign) []CampaignMilestoneInfo {
	milestones := campaign.Milestones
	milestoneInfoResponse := make([]CampaignMilestoneInfo, 0, len(milestones))

	var wg sync.WaitGroup
	milestoneChan := make(chan CampaignMilestoneInfo, len(milestones))

	for _, milestone := range milestones {
		// Capture the current milestone in the loop
		m := milestone
		wg.Add(1)

		go func(milestone *model.Milestone) {
			defer func() { wg.Done() }()

			milestoneInfo := CampaignMilestoneInfo{
				ID:                   milestone.ID.String(),
				Description:          *milestone.Description,
				DueDate:              milestone.DueDate.String(),
				CompletedAt:          "",
				CompletionPercentage: milestone.CompletionPercentage,
				Status:               milestone.Status.String(),
				BehindSchedule:       milestone.BehindSchedule,
				NumberOfTasks:        len(milestone.Tasks),
			}
			if milestone.CompletedAt != nil {
				milestoneInfo.CompletedAt = utils.FormatLocalTime(*milestone.CompletedAt, "")
			}

			milestoneChan <- milestoneInfo
		}(m)
	}

	wg.Wait()

	close(milestoneChan)
	for milestoneInfo := range milestoneChan {
		milestoneInfoResponse = append(milestoneInfoResponse, milestoneInfo)
	}

	return milestoneInfoResponse
}

// CalculateCampaignProgress calculates the total number of tasks and the percentage of completed tasks in a campaign.
func (cdr CampaignDetailsResponse) CalculateCampaignProgress(campaign *model.Campaign) (int, float64) {
	var totalTasks, completedTasks int

	for _, milestone := range campaign.Milestones {
		totalTasks += len(milestone.Tasks)

		for _, task := range milestone.Tasks {
			if task.Status == enum.TaskStatusDone {
				completedTasks += 1
			}
		}
	}

	return totalTasks, (float64(completedTasks) / float64(totalTasks)) * 100
}

// CampaignInfoPaginationResponse represents a paginated basic response for campaign.
// Only used for Swaggo swagger docs generation
type CampaignInfoPaginationResponse PaginationResponse[CampaignInfoResponse]

// CampaignDetailsPaginationResponse represents a paginated detailed response for campaign.
// Only used for Swaggo swagger docs generation
type CampaignDetailsPaginationResponse PaginationResponse[CampaignDetailsResponse]
