package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type InProgressState struct{}

func (i *InProgressState) Name() enum.TaskStatus { return enum.TaskStatusInProgress }

func (i *InProgressState) Next(ctx *TaskContext, next TaskState) error {

	if _, ok := i.AllowedTransitions()[next.Name()]; ok {
		if next.Name() == enum.TaskStatusRecap {
			prdStatusCheck := ctx.IsAllProductsActive()
			contentStatusCheck := ctx.IsAllContentsPosted()

			// forbid if neither products are active nor contents are posted
			if !prdStatusCheck && !contentStatusCheck {
				return fmt.Errorf(
					"cannot transition to %s: neither products are active nor contents are posted",
					next.Name(),
				)
			}
		}

		ctx.State = next
		ctx.IsCancelAndCascade(next)
		return nil
	}

	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i *InProgressState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{
		enum.TaskStatusRecap:     {},
		enum.TaskStatusCancelled: {},
	}
}
