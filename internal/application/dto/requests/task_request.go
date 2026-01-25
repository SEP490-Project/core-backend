package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidTaskType   = errors.New("invalid task type")
	ErrInvalidTaskStatus = errors.New("invalid task status")
)

// region: ======= Task Filter Request =======

// TaskFilterRequest represents the filtering criteria for retrieving tasks.
type TaskFilterRequest struct {
	PaginationRequest
	CreatedByID      *string `form:"created_by_id" json:"created_by" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AssignedToID     *string `form:"assigned_to_id" json:"assigned_to" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	MilestoneID      *string `form:"milestone_id" json:"milestone_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	CampaignID       *string `form:"campaign_id" json:"campaign_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ContractID       *string `form:"contract_id" json:"contract_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	DeadlineFromDate *string `form:"deadline_from_date" json:"start_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-01"`
	DeadlineToDate   *string `form:"deadline_to_date" json:"end_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-31"`
	UpdatedFromDate  *string `form:"updated_from_date" json:"updated_start_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-01"`
	UpdatedToDate    *string `form:"updated_to_date" json:"updated_end_date" validate:"omitempty,datetime=2006-01-02" example:"2023-10-31"`
	Status           *string `form:"status" json:"status" validate:"omitempty,oneof=TODO IN_PROGRESS CANCELLED RECAP DONE" example:"TODO"`
	Type             *string `form:"type" json:"type" validate:"omitempty,oneof=PRODUCT CONTENT EVENT OTHER" example:"OTHER"`
	HasContent       *bool   `form:"has_content" json:"has_content"`
	HasProduct       *bool   `form:"has_product" json:"has_product"`
}

// endregion

// region: ======= Create Task Requests =======

// CreateTaskRequest represents the payload for creating a new task.
type CreateTaskRequest struct {
	MilestoneID       string  `json:"milestone_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name              string  `json:"name" validate:"required,max=255" example:"Design Social Media Posts"`
	Description       any     `json:"description" validate:"required"` // JSON format
	Deadline          string  `json:"deadline" validate:"required,datetime=2006-01-02 15:04:05" example:"2023-10-15 17:00:00"`
	Type              string  `json:"type" validate:"required,oneof=PRODUCT CONTENT EVENT OTHER" example:"CONTENT"`
	Status            string  `json:"status" validate:"required,oneof=TODO IN_PROGRESS CANCELLED RECAP DONE" example:"TODO"`
	AssignedToID      *string `json:"assigned_to" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ScopeOfWorkItemID *string `json:"scope_of_work_item_id" validate:"omitempty,max=50" example:"1"`

	CreatedByID string `json:"-" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// ToModel converts CreateTaskRequest to Task model.
func (ctr CreateTaskRequest) ToModel() (*model.Task, error) {
	convertedModel := &model.Task{
		ID:   uuid.New(),
		Name: ctr.Name,
	}
	if ctr.ScopeOfWorkItemID != nil {
		convertedModel.ScopeOfWorkItemID = ctr.ScopeOfWorkItemID
	}
	if rawDescription, err := json.Marshal(ctr.Description); err == nil {
		convertedModel.Description = rawDescription
	} else {
		return nil, err
	}

	if deadline, err := utils.ParseLocalTime(ctr.Deadline, "2006-01-02 15:04:05"); err != nil {
		convertedModel.Deadline = *deadline
	} else {
		return nil, err
	}
	if taskType := enum.TaskType(ctr.Type); taskType.IsValid() {
		convertedModel.Type = taskType
	} else {
		return nil, ErrInvalidTaskType
	}
	if taskStatus := enum.TaskStatus(ctr.Status); taskStatus.IsValid() {
		convertedModel.Status = taskStatus
	} else {
		return nil, ErrInvalidTaskStatus
	}
	if milestoneID, err := uuid.Parse(ctr.MilestoneID); err == nil {
		convertedModel.MilestoneID = milestoneID
	} else {
		return nil, err
	}
	if ctr.AssignedToID != nil {
		if assignedToID, err := uuid.Parse(*ctr.AssignedToID); err == nil {
			convertedModel.AssignedToID = &assignedToID
		} else {
			return nil, err
		}
	}
	if createdByID, err := uuid.Parse(ctr.CreatedByID); err == nil {
		convertedModel.CreatedByID = createdByID
	} else {
		return nil, err
	}

	return convertedModel, nil
}

// endregion

// region: ======= Update Task Requests =======

// UpdateTaskRequest represents the payload for updating an existing task.
type UpdateTaskRequest struct {
	MilestoneID  *string `json:"milestone_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name         *string `json:"name" validate:"omitempty,max=255" example:"Design Social Media Posts"`
	Description  *any    `json:"description" validate:"omitempty"` // JSON format
	Deadline     *string `json:"deadline" validate:"omitempty,datetime=2006-01-02 15:04:05" example:"2023-10-15 17:00:00"`
	Type         *string `json:"type" validate:"omitempty,oneof=PRODUCT CONTENT EVENT OTHER" example:"CONTENT"`
	Status       *string `json:"status" validate:"omitempty,oneof=TODO IN_PROGRESS CANCELLED RECAP DONE" example:"TODO"`
	AssignedToID *string `json:"assigned_to" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`

	ID          string `json:"-"`
	UpdatedByID string `json:"-"`
}

func (utr UpdateTaskRequest) ToExistingModel(task *model.Task) (*model.Task, error) {
	if task == nil {
		return nil, errors.New("task model does not exist")
	}
	if utr.Name != nil {
		task.Name = *utr.Name
	}
	if utr.Description != nil {
		if rawDescription, err := json.Marshal(*utr.Description); err == nil {
			task.Description = rawDescription
		} else {
			return nil, err
		}
	}
	if utr.Deadline != nil {
		if deadline, err := utils.ParseLocalTime(*utr.Deadline, "2006-01-02 15:04:05"); err == nil {
			task.Deadline = *deadline
		} else {
			return nil, err
		}
	}
	if utr.Type != nil {
		if taskType := enum.TaskType(*utr.Type); taskType.IsValid() {
			task.Type = taskType
		} else {
			return nil, ErrInvalidTaskType
		}
	}
	if utr.Status != nil {
		if taskStatus := enum.TaskStatus(*utr.Status); taskStatus.IsValid() {
			task.Status = taskStatus
		} else {
			return nil, ErrInvalidTaskStatus
		}
	}
	if utr.MilestoneID != nil {
		if milestoneID, err := uuid.Parse(*utr.MilestoneID); err == nil {
			task.MilestoneID = milestoneID
		} else {
			return nil, err
		}
	}
	if utr.AssignedToID != nil {
		if assignedToID, err := uuid.Parse(*utr.AssignedToID); err == nil {
			task.AssignedToID = &assignedToID
		} else {
			return nil, err
		}
	}
	return task, nil
}

// endregion
