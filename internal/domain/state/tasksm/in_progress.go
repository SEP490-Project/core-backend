package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type InProgressState struct{}

func (i *InProgressState) Name() enum.TaskStatus { return enum.TaskStatusInProgress }

func (i *InProgressState) Next(ctx *TaskContext, next TaskState) error {
	// Log state of product activation using structured logging
	prdStatusCheck := ctx.IsAllProductsActive()
	contentStatusCheck := ctx.IsAllContentsPosted()

	if !prdStatusCheck {
		return fmt.Errorf("cannot transition to %s: not all products are active", next.Name())
	}

	if !contentStatusCheck {
		return fmt.Errorf("cannot transition to %s: not all contents are posted", next.Name())
	}

	if _, ok := i.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}

	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i *InProgressState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{
		enum.TaskStatusRecap: {},
	}
}
