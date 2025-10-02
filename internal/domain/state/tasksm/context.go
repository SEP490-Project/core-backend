package tasksm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"fmt"
	"go.uber.org/zap"
)

type TaskContext struct {
	state    TaskState
	products []*model.Product
}

func NewTaskContext(initialState TaskState) *TaskContext {
	return &TaskContext{state: initialState}
}

func (ctx *TaskContext) CurrentState() TaskState {
	return ctx.state
}

func (ctx *TaskContext) SetState(state TaskState) {
	zap.L().Debug("State transition",
		zap.String("from", ctx.state.Name()),
		zap.String("to", state.Name()),
	)
	ctx.state = state
}

func (ctx *TaskContext) TransitionTo(state TaskState) error {
	if !isAllowed(ctx.state, state.Name()) {
		return errors.New("invalid state transition from " + ctx.state.Name() + " to " + state.Name() + ". Allowed: " + fmt.Sprint(PrintAllowedTransitions(ctx.state)))
	}
	return ctx.state.Next(ctx)
}

// helper
func (c *TaskContext) IsAllProductsActive() bool {
	for _, p := range c.products {
		if p.Status != enum.ProductStatus("ACTIVED") {
			return false
		}
	}
	return true
}
