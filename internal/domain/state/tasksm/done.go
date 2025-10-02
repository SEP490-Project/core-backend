package tasksm

import "fmt"

type DoneState struct{}

func (d DoneState) Name() string {
	return "DONE"
}

func (d DoneState) Next(ctx *TaskContext) error {
	return fmt.Errorf("invalid transition: " + "The state is final and cannot transition to another state")
}

func (d DoneState) AllowedTransitions() map[string]struct{} {
	return map[string]struct{}{}
}
