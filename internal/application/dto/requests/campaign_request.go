package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CampaignFilterRequest represents the request payload for filtering campaigns.
type CampaignFilterRequest struct {
	PaginationRequest
	StartDate *time.Time `json:"start_date" form:"start_date" validate:"omitempty" example:"2023-06-01T00:00:00Z"`
	EndDate   *time.Time `json:"end_date" form:"end_date" validate:"omitempty,gtfield=StartDate" example:"2023-08-31T23:59:59Z"`
	Keyword   *string    `json:"keyword" form:"keyword" validate:"omitempty,min=1,max=255" example:"Summer"`
	Status    *string    `json:"status" form:"status" validate:"omitempty,oneof=DRAFT RUNNING COMPLETED CANCELLED" example:"RUNNING"`
	Type      *string    `json:"type" form:"type" validate:"omitempty,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
}

// UpdateCampaignRequest represents the request payload for updating an existing campaign.
type UpdateCampaignRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=3,max=255" example:"Summer Sale Campaign"`
	Description *string `json:"description" validate:"omitempty,max=1000" example:"A campaign for the summer sale."`
	StartDate   *time.Time
	EndDate     *time.Time
	Type        *string `json:"type" validate:"omitempty,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`

	// Metadata fields (not exposed in JSON)
	UpdatedBy *uuid.UUID `json:"-" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

func (r UpdateCampaignRequest) ToExistingModel(existing *model.Campaign) (*model.Campaign, error) {
	if existing == nil || existing.ID == uuid.Nil {
		return nil, fmt.Errorf("existing campaign model is nil")
	}

	if r.Name != nil {
		existing.Name = *r.Name
	}
	if r.Description != nil {
		existing.Description = r.Description
	}
	if r.StartDate != nil {
		existing.StartDate = *r.StartDate
	}
	if r.EndDate != nil {
		existing.EndDate = *r.EndDate
	}
	if r.Type != nil {
		campaignType := enum.ContractType(*r.Type)
		if !campaignType.IsValid() {
			return nil, fmt.Errorf("invalid campaign type: %s", *r.Type)
		}
		existing.Type = campaignType
	}
	return existing, nil
}

// region: ======= Create Campaign Requests =======

// CreateInternalCampaignRequest represents the request payload for creating a new internal campaign.
// This is only used for Swagger docuemntation purposes.
type CreateInternalCampaignRequest struct {
	Name        string                           `json:"name" validate:"required,min=3,max=255" example:"Summer Sale Campaign"`
	Description *string                          `json:"description" validate:"omitempty,max=1000" example:"A campaign for the summer sale."`
	StartDate   time.Time                        `json:"start_date" validate:"required" example:"2023-06-01T00:00:00Z"`
	EndDate     time.Time                        `json:"end_date" validate:"required,gtfield=StartDate" example:"2023-08-31T23:59:59Z"`
	Type        string                           `json:"type" validate:"required,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Milestones  []CreateMilestoneCampaignRequest `json:"milestones" validate:"dive"`
}

// CreateCampaignRequest represents the request payload for creating a new campaign.
// It embeds CreateInternalCampaignRequest to include all necessary fields.
type CreateCampaignRequest struct {
	ContractID string `json:"contract_id,omitempty" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CreateInternalCampaignRequest
}

// CreateMilestoneCampaignRequest represents the request payload for creating a new milestone campaign.
type CreateMilestoneCampaignRequest struct {
	Description *string                     `json:"description" validate:"omitempty,max=1000" example:"Milestone for initial launch."`
	DueDate     time.Time                   `json:"due_date" validate:"required" example:"2023-06-15T00:00:00Z"`
	Tasks       []CreateTaskCampaignRequest `json:"tasks" validate:"dive"`
}

// CreateTaskCampaignRequest represents the request payload for creating a new task campaign.
type CreateTaskCampaignRequest struct {
	Name string `json:"name" validate:"required,min=3,max=255" example:"Design Banner Ads"`
	//	@Example	{"details":"Create banner ads for the summer sale.","resources":["link1","link2"]}
	Description  any       `json:"description" validate:"omitempty"`
	Deadline     time.Time `json:"deadline" validate:"required" example:"2023-06-10T00:00:00Z"`
	Type         string    `json:"type" validate:"required,oneof=PRODUCT CONTENT EVENT OTHER" example:"PRODUCT"`
	AssignedToID *string   `json:"assigned_to" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ToModel converts the CreateCampaignRequest to a Campaign model.
func (ccr *CreateCampaignRequest) ToModel(userID uuid.UUID) (modelResult *model.Campaign, totalTasksCount int, err error) {
	contractID := uuid.Nil
	if ccr.ContractID != "" {
		contractID, err = uuid.Parse(ccr.ContractID)
		if err != nil {
			zap.L().Error("Failed to parse ContractID", zap.Error(err))
			return nil, 0, err
		}
	}
	campaignType := enum.ContractType(ccr.Type)
	if !campaignType.IsValid() {
		zap.L().Error("Invalid campaign type", zap.String("type", ccr.Type))
		return nil, 0, errors.New("invalid campaign type")
	}

	modelResult = &model.Campaign{
		ID:          uuid.New(),
		ContractID:  contractID,
		Name:        ccr.Name,
		Description: ccr.Description,
		StartDate:   ccr.StartDate,
		EndDate:     ccr.EndDate,
		Status:      enum.CampaignRunning,
		Type:        campaignType,
		CreatedByID: userID,
	}

	modelResult.Milestones, totalTasksCount, err = ccr.ToMilestoneModels(userID, modelResult.ID)
	if err != nil {
		zap.L().Error("Failed to convert milestones", zap.Error(err))
		return nil, 0, err
	}

	return
}

// ToMilestoneModels converts the milestones in the CreateCampaignRequest to Milestone models.
func (ccr *CreateCampaignRequest) ToMilestoneModels(userID uuid.UUID, campaignID uuid.UUID) ([]*model.Milestone, int, error) {
	milestoneRequest := ccr.Milestones
	milestoneLen := len(milestoneRequest)
	if milestoneLen == 0 {
		zap.L().Warn("No milestones provided in the campaign request")
		return nil, 0, nil
	}
	milestoneModels := make([]*model.Milestone, milestoneLen)
	totalTasksCount := 0

	var wg sync.WaitGroup

	for i, milestoneReq := range milestoneRequest {
		// Capture loop variables to avoid race conditions
		ir, request := i, milestoneReq
		totalTasksCount += len(request.Tasks)
		wg.Add(1)

		go func(i int, mr CreateMilestoneCampaignRequest) {
			defer wg.Done()

			milestone := &model.Milestone{
				ID:                   uuid.New(),
				CampaignID:           campaignID,
				Description:          mr.Description,
				DueDate:              mr.DueDate,
				CompletedAt:          nil,
				CompletionPercentage: 0,
				Status:               enum.MilestoneStatusNotStarted,
				BehindSchedule:       false,
				CreatedByID:          userID,
			}
			taskModels, err := ccr.ToTaskModels(userID, milestone.ID, &milestoneReq)
			if err != nil {
				zap.L().Error("Failed to convert tasks for milestone", zap.Int("index", i), zap.Error(err))
			} else {
				zap.L().Debug("Converted tasks for milestone", zap.Int("index", i), zap.Int("task_count", len(taskModels)))
				milestone.Tasks = taskModels
			}

			milestoneModels[i] = milestone
			zap.L().Debug("Converted milestone", zap.Int("index", i))
		}(ir, request)
	}

	wg.Wait()

	return milestoneModels, totalTasksCount, nil
}

// ToTaskModels converts the tasks in a CreateMilestoneCampaignRequest to Task models.
func (ccr *CreateCampaignRequest) ToTaskModels(
	userID uuid.UUID,
	milestoneID uuid.UUID,
	milestoneRequest *CreateMilestoneCampaignRequest,
) ([]*model.Task, error) {
	taskRequests := milestoneRequest.Tasks
	taskLen := len(taskRequests)
	if taskLen == 0 {
		zap.L().Warn("No tasks provided in the milestone request", zap.String("milestone_id", milestoneID.String()))
		return nil, nil
	}
	taskModels := make([]*model.Task, 0, taskLen)

	for _, tr := range taskRequests {
		taskModel := &model.Task{
			ID:           uuid.New(),
			MilestoneID:  milestoneID,
			Name:         tr.Name,
			Description:  nil,
			Deadline:     tr.Deadline,
			Type:         enum.TaskTypeOther,  // Default
			Status:       enum.TaskStatusToDo, // Default
			AssignedToID: nil,
			CreatedByID:  userID,
		}

		if tr.Description != nil {
			descBytes, err := json.Marshal(tr.Description)
			if err != nil {
				zap.L().Error("Failed to marshal task description, skipping description",
					zap.Any("description", tr.Description),
					zap.String("milestone_id", milestoneID.String()),
					zap.String("task_name", tr.Name),
					zap.Error(err))
			} else {
				taskModel.Description = descBytes
			}
		}

		if tr.AssignedToID != nil {
			assignedToID, err := uuid.Parse(*tr.AssignedToID)
			if err != nil {
				zap.L().Error("Failed to parse AssignedToID, skipping assignment",
					zap.String("assigned_to_id", *tr.AssignedToID),
					zap.String("milestone_id", milestoneID.String()),
					zap.String("task_name", tr.Name),
					zap.Error(err))
			} else {
				taskModel.AssignedToID = &assignedToID
			}
		}

		taskType := enum.TaskType(tr.Type)
		if !taskType.IsValid() {
			zap.L().Error("Invalid task type, defaulting to OTHER",
				zap.String("type", tr.Type),
				zap.String("task_name", tr.Name))
		} else {
			taskModel.Type = taskType
		}

		taskModels = append(taskModels, taskModel)
	}

	return taskModels, nil
}

// endregion

// region: ======= Custom Validators =======

// ValidateCreateCampaignRequest is a custom validator for CreateCampaignRequest.
// It ensures that the task types are valid for the given campaign type.
// "ADVERTISING": {"CONTENT", "OTHER"},
// "AFFILIATE":   {"CONTENT", "OTHER"},
// "COPRODUCE":   {"PRODUCT", "CONTENT", "OTHER"},
// "AMBASSADOR":  {"EVENT", "OTHER"},
func ValidateCreateCampaignRequest(sl validator.StructLevel) {
	campaign := sl.Current().Interface().(CreateCampaignRequest)

	allowedTasks := map[enum.ContractType][]enum.TaskType{
		enum.ContractTypeAdvertising: {enum.TaskTypeContent, enum.TaskTypeOther},
		enum.ContractTypeAffiliate:   {enum.TaskTypeContent, enum.TaskTypeOther},
		enum.ContractTypeAmbassador:  {enum.TaskTypeEvent, enum.TaskTypeOther},
		enum.ContractTypeCoProduce:   {enum.TaskTypeProduct, enum.TaskTypeContent, enum.TaskTypeOther},
	}

	campaignType := enum.ContractType(campaign.Type)
	if !campaignType.IsValid() {
		sl.ReportError(campaign.Type, "type", "Type", "invalidCampaignType", "")
		return
	}
	allowedTaskTypes, exists := allowedTasks[campaignType]
	if !exists {
		sl.ReportError(campaign.Type, "type", "Type", "invalidCampaignType", "")
		return
	}

	// Build a set of allowed task types for quick lookup
	allowedSet := make(map[string]struct{}, len(allowedTaskTypes))
	for _, t := range allowedTaskTypes {
		allowedSet[t.String()] = struct{}{}
	}

	for i, milestone := range campaign.Milestones {
		for ti, task := range milestone.Tasks {
			fieldPath := fmt.Sprintf("milestones[%d].tasks[%d].type", i, ti)
			taskType := enum.TaskType(task.Type)
			if !taskType.IsValid() {
				sl.ReportError(task.Type, fieldPath, "Type", "invalidTaskType", "")
				continue
			}
			if _, ok := allowedSet[taskType.String()]; !ok {
				sl.ReportError(task.Type, fieldPath, "Type", "taskTypeNotAllowedForCampaignType", campaign.Type)
			}
		}
	}
}

// endregion
