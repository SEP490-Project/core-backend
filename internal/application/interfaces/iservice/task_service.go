package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type TaskService interface {
	// GetTaskByFilter retrieves a list of tasks based on the provided filter criteria.
	GetTaskByFilter(ctx context.Context, filter *requests.TaskFilterRequest) ([]responses.TaskListResponse, int64, error)
	// GetTaskByID retrieves detailed information about a specific task by its ID.
	GetTaskByID(ctx context.Context, taskID uuid.UUID) (*responses.TaskResponse, error)

	// CreateTask creates a new task based on the provided request data.
	CreateTask(ctx context.Context, uow irepository.UnitOfWork, createRequest *requests.CreateTaskRequest) (*responses.TaskResponse, error)

	// AssignTask assigns a task to a user.
	AssignTask(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID, userID uuid.UUID, updatedByID uuid.UUID) (*responses.TaskResponse, error)

	// UpdateTask updates an existing task with the provided update data.
	UpdateTaskByID(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID, updateRequest *requests.UpdateTaskRequest) (*responses.TaskResponse, error)

	// DeleteTask deletes a task by its ID.
	DeleteTask(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID) error
}
