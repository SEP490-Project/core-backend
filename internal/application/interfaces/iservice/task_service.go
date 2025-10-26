package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

type TaskService interface {
	GetTaskByFilter(ctx context.Context, filter *requests.TaskFilterRequest) ([]responses.TaskListResponse, int64, error)
	GetTaskByID(ctx context.Context, taskID uuid.UUID) (*responses.TaskResponse, error)
}
