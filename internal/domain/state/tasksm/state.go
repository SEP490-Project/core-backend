package tasksm

import "core-backend/internal/domain/enum"

type TaskState interface {
	Name() enum.TaskStatus
	Next(ctx *TaskContext, next TaskState) error
	AllowedTransitions() map[enum.TaskStatus]struct{}
}

func NewTaskState(status enum.TaskStatus) TaskState {
	switch status {
	case enum.TaskStatusToDo:
		return &ToDoState{}
	case enum.TaskStatusInProgress:
		return &InProgressState{}
	case enum.TaskStatusRecap:
		return &RecapState{}
	case enum.TaskStatusDone:
		return &DoneState{}
	case enum.TaskStatusCancelled:
		return &CancelledState{}
	default:
		return nil
	}
}

func isAllowed(current TaskState, targetState enum.TaskStatus) bool {
	_, ok := current.AllowedTransitions()[targetState]
	return ok
}

func PrintAllowedTransitions(state TaskState) []string {
	transitions := make([]string, 0, len(state.AllowedTransitions()))
	for k := range state.AllowedTransitions() {
		transitions = append(transitions, k.String())
	}
	return transitions
}
