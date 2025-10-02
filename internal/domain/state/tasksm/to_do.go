package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ToDoState struct{}

func (s *ToDoState) Next(ctx *TaskContext, next TaskState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s *ToDoState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{
		enum.TaskStatusInProgress: {},
		enum.TaskStatusCancelled:  {},
	}
}

func (s *ToDoState) Name() enum.TaskStatus {
	return enum.TaskStatusToDo
}
