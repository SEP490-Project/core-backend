package tasksm

import "fmt"

type ToDoState struct{}

func (s *ToDoState) Next(ctx *TaskContext, next TaskState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s *ToDoState) AllowedTransitions() map[string]struct{} {
	return map[string]struct{}{
		"IN_PROGRESS": {},
		"CANCELLED":   {},
	}
}

func (s *ToDoState) Name() string {
	return "TO_DO"
}
