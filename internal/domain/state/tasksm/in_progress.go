package tasksm

import "fmt"

type InProgressState struct{}

func (i InProgressState) Name() string {
	return "IN_PROGRESS"
}

func (i InProgressState) Next(ctx TaskContext, next TaskState) error {
	if _, ok := i.AllowedTransitions()[next.Name()]; ok && ctx.IsAllProductsActive() {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i InProgressState) AllowedTransitions() map[string]struct{} {
	return map[string]struct{}{
		"RECAP": {},
	}
}
