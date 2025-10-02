package tasksm

type TaskState interface {
	Name() string
	Next(ctx *TaskContext, next TaskState) error
	AllowedTransitions() map[string]struct{}
}

func isAllowed(current TaskState, targetState string) bool {
	_, ok := current.AllowedTransitions()[targetState]
	return ok
}

func PrintAllowedTransitions(state TaskState) []string {
	transitions := make([]string, 0, len(state.AllowedTransitions()))
	for k := range state.AllowedTransitions() {
		transitions = append(transitions, k)
	}
	return transitions
}
