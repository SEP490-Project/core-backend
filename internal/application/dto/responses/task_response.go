package responses

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"encoding/json"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// region: =============== TaskResponse ===============

type TaskResponse struct {
	ID             string                `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name           string                `json:"name" example:"Design Homepage"`
	Description    any                   `json:"description"`
	Deadline       string                `json:"deadline" example:"2023-12-31T23:59:59Z"`
	Type           string                `json:"type" example:"PRODUCT"`
	Status         string                `json:"status" example:"IN_PROGRESS"`
	AssignedToID   *string               `json:"assigned_to_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	AssignedToName *string               `json:"assigned_to_name,omitempty" example:"Jane Smith"`
	AssignedToRole *string               `json:"assigned_to_role,omitempty" example:"Sales Staff"`
	CreatedAt      string                `json:"created_at" example:"2023-10-01T12:00:00Z"`
	CreatedByID    string                `json:"created_by_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreatedByName  string                `json:"created_by_name" example:"John Doe"`
	UpdatedAt      string                `json:"updated_at" example:"2023-10-15T15:30:00Z"`
	UpdatedByID    string                `json:"updated_by_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UpdatedByName  string                `json:"updated_by_name" example:"John Doe"`
	MilestoneID    *string               `json:"milestone_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	MilestoneInfo  *MilestoneResponse    `json:"milestone_details,omitempty"`
	CampaignID     *string               `json:"campaign_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	CampaignInfo   *CampaignInfoResponse `json:"campaign_details,omitempty"`
	ContractID     *string               `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	BrandInfo      *BrandInfoResponse    `json:"brand_info,omitempty"`

	ProductInfos []ProductInfo `json:"product_ids,omitempty"`
	ContentInfos []ContentInfo `json:"content_ids,omitempty"`
}

type ProductInfo struct {
	ID   uuid.UUID        `json:"id"`
	Name string           `json:"name"`
	Type enum.ProductType `json:"type"`
}

type ContentInfo struct {
	ID          uuid.UUID        `json:"id"`
	Title       string           `json:"title"`
	Description *string          `json:"description,omitempty"`
	Type        enum.ContentType `json:"type"`
}

func (TaskResponse) ToResponse(dto *dtos.TaskDetailDTO) *TaskResponse {
	response := &TaskResponse{
		ID:        dto.ID.String(),
		Name:      dto.Name,
		Deadline:  utils.FormatLocalTime(&dto.Deadline, ""),
		Type:      dto.Type.String(),
		Status:    dto.Status.String(),
		CreatedAt: utils.FormatLocalTime(&dto.CreatedAt, ""),
		UpdatedAt: utils.FormatLocalTime(&dto.UpdatedAt, ""),
	}

	if len(dto.Description) > 0 {
		err := json.Unmarshal(dto.Description, &response.Description)
		if err != nil {
			zap.L().Error("Failed to unmarshal description", zap.Error(err))
		}
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

	if len(dto.ProductInfos) > 0 {
		response.ProductInfos = utils.MapSlice(dto.ProductInfos, func(info dtos.ProductInfo) ProductInfo {
			return ProductInfo{
				ID:   info.ID,
				Name: info.Name,
				Type: info.Type,
			}
		})
	}
	if len(dto.ContentInfos) > 0 {
		response.ContentInfos = utils.MapSlice(dto.ContentInfos, func(info dtos.ContentInfo) ContentInfo {
			return ContentInfo{
				ID:          info.ID,
				Title:       info.Title,
				Description: info.Description,
				Type:        info.Type,
			}
		})
	}

	if dto.MilestoneInfo != nil {
		var milestone dtos.MilestoneDTO
		_ = json.Unmarshal(dto.MilestoneInfo, &milestone)
		response.MilestoneInfo = MilestoneResponse{}.ToResponse(&milestone)
	}
	if dto.CampaignInfo != nil {
		var campaign dtos.CampaignDTO
		_ = json.Unmarshal(dto.CampaignInfo, &campaign)
		if campaign.ID != uuid.Nil {
			response.CampaignInfo = &CampaignInfoResponse{
				ID:          campaign.ID.String(),
				Name:        campaign.Name,
				Description: campaign.Description,
				StartDate:   campaign.StartDate,
				EndDate:     campaign.EndDate,
				Status:      campaign.Status.String(),
				Type:        campaign.Type.String(),
			}
		}
	}

	if dto.BrandInfo != nil {
		response.BrandInfo = &BrandInfoResponse{
			ID:      dto.BrandInfo.ID,
			Name:    dto.BrandInfo.Name,
			LogoURL: dto.BrandInfo.LogoURL,
			Status:  dto.BrandInfo.Status,
		}
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
	CampaignName   *string `json:"campaign_name,omitempty" example:"Summer Sale Campaign"`
	ContractID     *string `json:"contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ProductID      *string `json:"product_id" example:"550e8400-e29b-41d4-a716-446655440000"` // Optional field for associated product ID
	ChildStatus    *string `json:"child_status" gorm:"column:child_status"`
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
		if dto.CampaignName != nil {
			res.CampaignName = dto.CampaignName
		}
		if dto.ContractID != nil {
			res.ContractID = utils.PtrOrNil(dto.ContractID.String())
		}
		if dto.ProductID != nil {
			res.ProductID = utils.PtrOrNil(dto.ProductID.String())
		}
		if dto.ChildStatus != nil {
			childStatus := dto.ChildStatus.String()
			res.ChildStatus = &childStatus
		}

		responses[i] = *res
	}

	return responses
}

// PaginationTaskResponse represents a paginated response for tasks.
// Only used for Swaggo swagger docs generation.
type PaginationTaskResponse PaginationResponse[TaskListResponse]

// endregion
