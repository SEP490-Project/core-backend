package responses

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// region: =============== TaskResponse ===============

type TaskResponse struct {
	ID             string   `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name           string   `json:"name" example:"Design Homepage"`
	Description    any      `json:"description"`
	Deadline       string   `json:"deadline" example:"2023-12-31T23:59:59Z"`
	Type           string   `json:"type" example:"PRODUCT"`
	Status         string   `json:"status" example:"IN_PROGRESS"`
	AssignedToID   *string  `json:"assigned_to_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	AssignedToName *string  `json:"assigned_to_name,omitempty" example:"Jane Smith"`
	AssignedToRole *string  `json:"assigned_to_role,omitempty" example:"Sales Staff"`
	CreatedAt      string   `json:"created_at" example:"2023-10-01T12:00:00Z"`
	CreatedByID    string   `json:"created_by_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedByName  string   `json:"created_by_name" example:"John Doe"`
	UpdatedAt      string   `json:"updated_at" example:"2023-10-15T15:30:00Z"`
	UpdatedByID    string   `json:"updated_by_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UpdatedByName  string   `json:"updated_by_name" example:"John Doe"`
	MilestoneID    *string  `json:"milestone_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CampaignID     *string  `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractID     *string  `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductIDs     []string `json:"product_ids,omitempty" example:"[\"550e8400-e29b-41d4-a716-446655440000\", \"660e8400-e29b-41d4-a716-446655440000\"]"`
	ContentIDs     []string `json:"content_ids,omitempty" example:"[\"770e8400-e29b-41d4-a716-446655440000\", \"880e8400-e29b-41d4-a716-446655440000\"]"`
}

func (TaskResponse) ToResponse(dto *dtos.TaskDetailDTO) *TaskResponse {
	response := &TaskResponse{
		ID:          dto.ID.String(),
		Name:        dto.Name,
		Description: dto.Description,
		Deadline:    utils.FormatLocalTime(&dto.Deadline, ""),
		Type:        dto.Type.String(),
		Status:      dto.Status.String(),
		CreatedAt:   utils.FormatLocalTime(&dto.CreatedAt, ""),
		UpdatedAt:   utils.FormatLocalTime(&dto.UpdatedAt, ""),
	}

	if dto.AssignedToID != nil {
		response.AssignedToID = utils.PtrOrNil(dto.AssignedToID.String())
		response.AssignedToName = dto.AssignedToName
		response.AssignedToRole = utils.PtrOrNil(dto.AssignedToRole.String())
	}

	if dto.MilestoneID != nil {
		response.MilestoneID = utils.PtrOrNil(dto.MilestoneID.String())
	}
	if dto.CampaignID != nil {
		response.CampaignID = utils.PtrOrNil(dto.CampaignID.String())
	}
	if dto.ContractID != nil {
		response.ContractID = utils.PtrOrNil(dto.ContractID.String())
	}

	if len(dto.ContentIDs) > 0 {
		response.ContentIDs = utils.MapSlice(dto.ContentIDs, func(id uuid.UUID) string { return id.String() })
	}
	if len(dto.ProductIDs) > 0 {
		response.ProductIDs = utils.MapSlice(dto.ProductIDs, func(id uuid.UUID) string { return id.String() })
	}

	return response
}

// endregion

// region: =============== TaskListResponse ===============

// TaskListResponse represents a summarized view of a task for listing purposes.
type TaskListResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name           string  `json:"name" example:"Design Homepage"`
	Deadline       string  `json:"deadline" example:"2023-12-31T23:59:59Z"`
	Type           string  `json:"type" example:"PRODUCT"`
	Status         string  `json:"status" example:"IN_PROGRESS"`
	AssignedToID   *string `json:"assigned_to_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	AssignedToName *string `json:"assigned_to_name,omitempty" example:"Jane Smith"`
	AssignedToRole *string `json:"assigned_to_role,omitempty" example:"Sales Staff"`
	CreatedAt      string  `json:"created_at" example:"2023-10-01T12:00:00Z"`
	UpdatedAt      string  `json:"updated_at" example:"2023-10-15T15:30:00Z"`
	MilestoneID    *string `json:"milestone_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CampaignID     *string `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractID     *string `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
}

func (TaskListResponse) ToListResponse(dtos []dtos.TaskListDTO) []TaskListResponse {
	if len(dtos) == 0 {
		return []TaskListResponse{}
	}

	responses := make([]TaskListResponse, len(dtos))
	for i, dto := range dtos {
		res := &TaskListResponse{
			ID:        dto.ID.String(),
			Name:      dto.Name,
			Deadline:  dto.Deadline.String(),
			Type:      dto.Type.String(),
			Status:    dto.Status.String(),
			CreatedAt: dto.CreatedAt.String(),
			UpdatedAt: dto.UpdatedAt.String(),
		}
		if dto.AssignedToID != nil {
			res.AssignedToID = utils.PtrOrNil(dto.AssignedToID.String())
			res.AssignedToName = dto.AssignedToName
			res.AssignedToRole = utils.PtrOrNil(dto.AssignedToRole.String())
		}
		if dto.MilestoneID != nil {
			res.MilestoneID = utils.PtrOrNil(dto.MilestoneID.String())
		}
		if dto.CampaignID != nil {
			res.CampaignID = utils.PtrOrNil(dto.CampaignID.String())
		}
		if dto.ContractID != nil {
			res.ContractID = utils.PtrOrNil(dto.ContractID.String())
		}

		responses[i] = *res
	}

	return responses
}

// PaginationTaskResponse represents a paginated response for tasks.
// Only used for Swaggo swagger docs generation.
type PaginationTaskResponse PaginationResponse[TaskListResponse]

// endregion
