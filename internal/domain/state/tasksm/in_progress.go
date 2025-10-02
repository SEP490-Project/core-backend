package tasksm

import (
	"core-backend/internal/domain/enum"
	"fmt"
	"go.uber.org/zap"
)

type InProgressState struct{}

func (i *InProgressState) Name() enum.TaskStatus { return enum.TaskStatusInProgress }

func (i *InProgressState) Next(ctx *TaskContext, next TaskState) error {
	// Log state of product activation using structured logging
	prdStatusCheck := ctx.IsAllProductsActive()
	zap.L().Info("is_all_products_active", zap.Bool("value", prdStatusCheck))

	if !prdStatusCheck {
		return fmt.Errorf("cannot transition to %s: not all products are active", next.Name())
	}

	if _, ok := i.AllowedTransitions()[next.Name()]; ok && prdStatusCheck {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", i.Name(), next.Name())
}

func (i *InProgressState) AllowedTransitions() map[enum.TaskStatus]struct{} {
	return map[enum.TaskStatus]struct{}{
		enum.TaskStatusRecap: {},
	}
}
