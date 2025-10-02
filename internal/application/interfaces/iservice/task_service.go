package iservice

import (
	"core-backend/internal/domain/enum"
	"github.com/google/uuid"
)

type TaskService interface {
	MoveTaskToState(taskID uuid.UUID, targetState enum.TaskStatus) error
}
