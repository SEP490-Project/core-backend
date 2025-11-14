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

	// GetContractTrackingLinkByTaskID retrieves the contract tracking link associated with the given task ID
	// and only if the contract is of type 'AFFILIATE'.
	GetContractTrackingLinkByTaskID(ctx context.Context, taskID uuid.UUID) (string, uuid.UUID, error)
}
