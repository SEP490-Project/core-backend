package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type TaskRepository interface {
	GenericRepository[model.Task]
	GetListTasks(ctx context.Context, filter *requests.TaskFilterRequest) ([]dtos.TaskListDTO, int64, error)
	GetDetailTask(ctx context.Context, taskID uuid.UUID) (*dtos.TaskDetailDTO, error)
}
