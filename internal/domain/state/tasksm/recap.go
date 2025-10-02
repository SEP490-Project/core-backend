package tasksm

import "fmt"

type RecapState struct{}

func (r RecapState) Name() string {
	return "RECAP"
}

func (r RecapState) Next(ctx *TaskContext, next TaskState) error {
	if _, ok := r.AllowedTransitions()[next.Name()]; ok {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", r.Name(), next.Name())
}

func (r RecapState) AllowedTransitions() map[string]struct{} {
	return map[string]struct{}{
		"DONE": {},
	}
}
