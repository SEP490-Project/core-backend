package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type TaskService struct {
	taskRepo irepository.TaskRepository
}

// GetTaskByFilter implements iservice.TaskService.
func (t *TaskService) GetTaskByFilter(ctx context.Context, filter requests.TaskFilterRequest) ([]responses.TaskListResponse, int64, error) {
	zap.L().Info("TaskService - GetTaskByFilter called",
		zap.Any("request", filter))

	taskDTOs, total, err := t.taskRepo.GetListTasks(ctx, &filter)
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

func NewTaskService(taskRepo irepository.TaskRepository) iservice.TaskService {
	return &TaskService{taskRepo: taskRepo}
}
