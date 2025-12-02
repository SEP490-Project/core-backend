package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TaskService struct {
	taskRepo irepository.TaskRepository
	userRepo irepository.GenericRepository[model.User]
}

// AssignTask implements iservice.TaskService.
func (t *TaskService) AssignTask(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID, userID uuid.UUID, updatedByID uuid.UUID) (*responses.TaskResponse, error) {
	zap.L().Info("TaskService - AssignTask called",
		zap.Any("task_id", taskID),
		zap.Any("user_id", userID))

	taskRepo := uow.Tasks()

	var task *model.Task
	// Validate if tasks and users exists
	validateTaskFunc := func(ctx context.Context) error {
		var err error
		task, err = t.taskRepo.GetByID(ctx, taskID, nil)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				zap.L().Warn("TaskService - AssignTask - task not found",
					zap.Any("task_id", taskID))
				return errors.New("task not found")
			} else {
				zap.L().Error("TaskService - AssignTask - failed to get task",
					zap.Error(err))
				return err
			}
		}
		return nil
	}
	validateUserFunc := func(ctx context.Context) error {
		if exists, err := t.userRepo.ExistsByID(ctx, userID); err != nil {
			zap.L().Error("TaskService - AssignTask - user not found",
				zap.Error(err))
			return err
		} else if !exists {
			zap.L().Warn("TaskService - AssignTask - user not found",
				zap.Any("user_id", userID))
			return errors.New("user not found")
		}
		return nil
	}
	if err := utils.RunParallel(ctx, 2, validateTaskFunc, validateUserFunc); err != nil {
		zap.L().Error("TaskService - AssignTask - validation failed",
			zap.Error(err))
		return nil, err
	}

	task.AssignedToID = &userID
	task.UpdatedByID = &updatedByID
	if err := taskRepo.Update(ctx, task); err != nil {
		zap.L().Error("TaskService - AssignTask - failed to update task",
			zap.Error(err))
		return nil, err
	}
	uow.Commit()

	taskDetail, err := t.taskRepo.GetDetailTask(ctx, taskID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("TaskService - AssignTask - task not found",
				zap.Any("task_id", taskID))
			return nil, errors.New("task not found")
		}
		zap.L().Error("TaskService - AssignTask - failed to get task detail",
			zap.Error(err))
		return nil, err
	}
	zap.L().Info("TaskService - AssignTask - task assigned successfully",
		zap.Any("task_id", taskID),
		zap.Any("user_id", userID))
	return responses.TaskResponse{}.ToResponse(taskDetail), nil
}

// CreateTask implements iservice.TaskService.
func (t *TaskService) CreateTask(ctx context.Context, uow irepository.UnitOfWork, createRequest *requests.CreateTaskRequest) (*responses.TaskResponse, error) {
	zap.L().Info("TaskService - CreateTask called",
		zap.Any("request", createRequest))

	taskRepo := uow.Tasks()

	creatingTask, err := (*createRequest).ToModel()
	if err != nil {
		zap.L().Error("TaskService - CreateTask - failed to convert request to model",
			zap.Error(err))
		return nil, err
	}

	// Link to ScopeOfWork
	if err = t.linkTaskToScopeOfWork(ctx, uow, creatingTask, createRequest.ScopeOfWorkItemID); err != nil {
		zap.L().Warn("Failed to link task to scope of work", zap.Error(err))
		// We don't fail the request, just log warning
	}

	if err = taskRepo.Add(ctx, creatingTask); err != nil {
		zap.L().Error("TaskService - CreateTask - failed to create task",
			zap.Error(err))
		return nil, err
	}

	var createdTask *dtos.TaskDetailDTO
	createdTask, err = taskRepo.GetDetailTask(ctx, creatingTask.ID)
	if err != nil {
		zap.L().Error("TaskService - CreateTask - failed to get created task",
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("TaskService - CreateTask - task created successfully",
		zap.Any("task_id", createdTask.ID))
	return responses.TaskResponse{}.ToResponse(createdTask), nil
}

// DeleteTask implements iservice.TaskService.
func (t *TaskService) DeleteTask(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID) error {
	zap.L().Info("TaskService - DeleteTask called",
		zap.Any("task_id", taskID))

	taskRepo := uow.Tasks()

	if exists, err := taskRepo.ExistsByID(ctx, taskID); err != nil {
		zap.L().Error("TaskService - DeleteTask - failed to check task existence",
			zap.Error(err))
		return err
	} else if !exists {
		zap.L().Warn("TaskService - DeleteTask - task not found",
			zap.Any("task_id", taskID))
		return errors.New("task not found")
	}

	if err := taskRepo.DeleteByID(ctx, taskID); err != nil {
		zap.L().Error("TaskService - DeleteTask - failed to delete task",
			zap.Error(err))
		return err
	}

	zap.L().Info("TaskService - DeleteTask - task deleted successfully",
		zap.Any("task_id", taskID))
	return nil
}

// UpdateTaskByID implements iservice.TaskService.
func (t *TaskService) UpdateTaskByID(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID, updateRequest *requests.UpdateTaskRequest) (*responses.TaskResponse, error) {
	zap.L().Info("TaskService - UpdateTaskByID called",
		zap.Any("task_id", taskID),
		zap.Any("request", updateRequest))

	taskRepo := uow.Tasks()

	existingTask, err := taskRepo.GetByID(ctx, taskID, nil)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Warn("TaskService - UpdateTaskByID - task not found",
				zap.Any("task_id", taskID))
			return nil, errors.New("task not found")
		} else {
			zap.L().Error("TaskService - UpdateTaskByID - failed to get task",
				zap.Error(err))
			return nil, err
		}
	}

	updatingTask, err := (*updateRequest).ToExistingModel(existingTask)
	if err != nil {
		zap.L().Error("TaskService - UpdateTaskByID - failed to convert request to model",
			zap.Error(err))
		return nil, err
	}

	if err = taskRepo.Update(ctx, updatingTask); err != nil {
		zap.L().Error("TaskService - UpdateTaskByID - failed to update task",
			zap.Error(err))
		return nil, err
	}

	var updatedTask *dtos.TaskDetailDTO
	updatedTask, err = taskRepo.GetDetailTask(ctx, taskID)
	if err != nil {
		zap.L().Error("TaskService - UpdateTaskByID - failed to get updated task",
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("TaskService - UpdateTaskByID - task updated successfully",
		zap.Any("task_id", taskID))
	return responses.TaskResponse{}.ToResponse(updatedTask), nil
}

// GetTaskByFilter implements iservice.TaskService.
func (t *TaskService) GetTaskByFilter(ctx context.Context, filter *requests.TaskFilterRequest) ([]responses.TaskListResponse, int64, error) {
	zap.L().Info("TaskService - GetTaskByFilter called",
		zap.Any("request", filter))

	taskDTOs, total, err := t.taskRepo.GetListTasks(ctx, filter)
	if err != nil {
		zap.L().Error("TaskService - GetTaskByFilter failed",
			zap.Error(err))
		return []responses.TaskListResponse{}, 0, err
	}

	zap.L().Debug("TaskService - GetTaskByFilter - taskDTOs", zap.Any("taskDTOs", taskDTOs))
	return responses.TaskListResponse{}.ToListResponse(taskDTOs), total, nil
}

// GetTaskByID implements iservice.TaskService.
func (t *TaskService) GetTaskByID(ctx context.Context, taskID uuid.UUID) (*responses.TaskResponse, error) {
	zap.L().Info("TaskService - GetTaskByID called",
		zap.Any("task_id", taskID))

	task, err := t.taskRepo.GetDetailTask(ctx, taskID)
	if err != nil {
		zap.L().Error("TaskService - GetTaskByID failed",
			zap.Error(err))
		return nil, err
	} else if task == nil {
		zap.L().Warn("TaskService - GetTaskByID - task not found",
			zap.Any("task_id", taskID))
		return nil, errors.New("task not found")
	}

	return responses.TaskResponse{}.ToResponse(task), nil
}

func (t *TaskService) linkTaskToScopeOfWork(ctx context.Context, uow irepository.UnitOfWork, task *model.Task, requestedItemID *string) error {
	// 1. Get Milestone -> Campaign -> Contract
	milestoneRepo := uow.Milestones()
	milestone, err := milestoneRepo.GetByID(ctx, task.MilestoneID, []string{"Campaign"})
	if err != nil {
		return err
	}
	if milestone.Campaign == nil {
		return errors.New("milestone has no campaign")
	}

	contractRepo := uow.Contracts()
	contract, err := contractRepo.GetByID(ctx, milestone.Campaign.ContractID, nil)
	if err != nil {
		return err
	}

	// 2. Parse ScopeOfWork
	var sow dtos.ScopeOfWork
	if err := json.Unmarshal(contract.ScopeOfWork, &sow); err != nil {
		return err
	}

	// 3. Find and Link
	matched := false
	var itemID string
	if requestedItemID != nil {
		itemID = *requestedItemID
	}

	// Helper to check match
	checkMatch := func(id *int8, name string, targetType enum.TaskType) bool {
		if id == nil {
			return false
		}
		if itemID != "" {
			return fmt.Sprintf("%d", *id) == itemID
		}
		// Only match if task type matches target type
		if task.Type != targetType {
			return false
		}
		return name == task.Name
	}

	// Iterate AdvertisedItems (CONTENT)
	if sow.Deliverables.AdvertisedItems != nil {
		for i := range sow.Deliverables.AdvertisedItems {
			item := &sow.Deliverables.AdvertisedItems[i]
			if checkMatch(item.ID, item.Name, enum.TaskTypeContent) {
				item.TaskIDs = append(item.TaskIDs, task.ID)
				task.ScopeOfWorkItemID = utils.PtrOrNil(fmt.Sprintf("%d", *item.ID))
				matched = true
				goto Found
			}
		}
	}

	// Iterate Events (EVENT)
	if sow.Deliverables.Events != nil {
		for i := range sow.Deliverables.Events {
			item := &sow.Deliverables.Events[i]
			if checkMatch(item.ID, item.Name, enum.TaskTypeEvent) {
				item.TaskIDs = append(item.TaskIDs, task.ID)
				task.ScopeOfWorkItemID = utils.PtrOrNil(fmt.Sprintf("%d", *item.ID))
				matched = true
				goto Found
			}
		}
	}

	// Iterate Products (PRODUCT)
	if sow.Deliverables.Products != nil {
		for i := range sow.Deliverables.Products {
			item := &sow.Deliverables.Products[i]
			if checkMatch(item.ID, item.Name, enum.TaskTypeProduct) {
				item.TaskIDs = append(item.TaskIDs, task.ID)
				task.ScopeOfWorkItemID = utils.PtrOrNil(fmt.Sprintf("%d", *item.ID))
				matched = true
				goto Found
			}
		}
	}

	// Iterate Concepts (CONTENT)
	if sow.Deliverables.Concepts != nil {
		for i := range sow.Deliverables.Concepts {
			item := &sow.Deliverables.Concepts[i]
			if checkMatch(item.ID, item.Name, enum.TaskTypeContent) {
				item.TaskIDs = append(item.TaskIDs, task.ID)
				task.ScopeOfWorkItemID = utils.PtrOrNil(fmt.Sprintf("%d", *item.ID))
				matched = true
				goto Found
			}
		}
	}

Found:
	if matched {
		// Update Contract
		newSOWBytes, err := json.Marshal(sow)
		if err != nil {
			return err
		}
		contract.ScopeOfWork = newSOWBytes
		if err := contractRepo.Update(ctx, contract); err != nil {
			return err
		}
	}

	return nil
}

func NewTaskService(taskRepo irepository.TaskRepository, userRepo irepository.GenericRepository[model.User]) iservice.TaskService {
	return &TaskService{
		taskRepo: taskRepo,
		userRepo: userRepo,
	}
}
