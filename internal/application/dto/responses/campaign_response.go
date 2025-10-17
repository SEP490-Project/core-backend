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
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractID     string  `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractTitle  string  `json:"contract_title" example:"Q2 Marketing Contract"`
	ContractNumber string  `json:"contract_number" example:"contract_20251017_AD"`
	Name           string  `json:"name" example:"Summer Sale Campaign"`
	Description    *string `json:"description" example:"A campaign for the summer sale."`
	StartDate      string  `json:"start_date" example:"2023-06-01 00:00:00"`
	EndDate        string  `json:"end_date" example:"2023-08-31 23:59:59"`
	Status         string  `json:"status" example:"RUNNING"`
	Budget         float64 `json:"budget" example:"100000000"`
	Type           string  `json:"type" example:"ADVERTISING"`
	CreatedAt      string  `json:"created_at" example:"2023-06-01 00:00:00"`
	UpdatedAt      string  `json:"updated_at" example:"2023-06-15 12:00:00"`
}

// CampaignDetailsResponse represents the details of a campaign.
type CampaignDetailsResponse struct {
	ID                  string                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractID          string                  `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractTitle       string                  `json:"contract_title" example:"Q2 Marketing Contract"`
	ContractNumber      string                  `json:"contract_number" example:"contract_20251017_AD"`
	Name                string                  `json:"name" example:"Summer Sale Campaign"`
	Description         *string                 `json:"description" example:"A campaign for the summer sale."`
	StartDate           string                  `json:"start_date" example:"2023-06-01 00:00:00"`
	EndDate             string                  `json:"end_date" example:"2023-08-31 23:59:59"`
	Status              string                  `json:"status" example:"RUNNING"`
	Budget              float64                 `json:"budget" example:"100000000"`
	Type                string                  `json:"type" example:"ADVERTISING"`
	Milestones          []CampaignMilestoneInfo `json:"milestones"`
	NumberOfTasks       int                     `json:"number_of_tasks" example:"25"`
	PercentageCompleted float64                 `json:"percentage_completed" example:"60.5"`
	CreatedAt           string                  `json:"created_at" example:"2023-06-01 00:00:00"`
	UpdatedAt           string                  `json:"updated_at" example:"2023-06-15 12:00:00"`
}

// CampaignMilestoneInfo represents the details of milestone within a campaign.
type CampaignMilestoneInfo struct {
	ID                   string  `json:"id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	Description          string  `json:"description,omitempty" example:"Milestone for initial launch."`
	DueDate              string  `json:"due_date,omitempty" example:"2023-06-15 00:00:00"`
	CompletedAt          string  `json:"completed_at,omitempty" example:"2023-06-15 00:00:00"`
	CompletionPercentage float64 `json:"completion_percentage,omitempty" example:"60.5"`
	Status               string  `json:"status,omitempty" example:"NOT_STARTED"`
	BehindSchedule       bool    `json:"behind_schedule,omitempty" example:"false"`
	NumberOfTasks        int     `json:"number_of_tasks,omitempty" example:"25"`
}

// ToCampaignInfoResponse maps a Campaign model to a CampaignInfoResponse DTO.
// Only need basic info of the Campaign model
func (cir CampaignInfoResponse) ToCampaignInfoResponse(model *model.Campaign) *CampaignInfoResponse {
	return &CampaignInfoResponse{
		ID:             model.ID.String(),
		ContractID:     model.ContractID.String(),
		ContractTitle:  *model.Contract.Title,
		ContractNumber: *model.Contract.ContractNumber,
		Name:           model.Name,
		Description:    model.Description,
		StartDate:      model.StartDate.String(),
		EndDate:        model.EndDate.String(),
		Status:         model.Status.String(),
		Budget:         model.Budget,
		Type:           model.Type.String(),
		CreatedAt:      model.CreatedAt.String(),
		UpdatedAt:      model.UpdatedAt.String(),
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
		Budget:              model.Budget,
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
				milestoneInfo.CompletedAt = utils.FormatLocalTime(milestone.CompletedAt, "")
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
