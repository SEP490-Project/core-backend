package tasksm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"go.uber.org/zap"
)

type TaskContext struct {
	state    TaskState
	products []*model.Product
}

func NewTaskContext(initialState TaskState, product []*model.Product) *TaskContext {
	return &TaskContext{state: initialState}
}

func (ctx *TaskContext) State() TaskState {
	return ctx.state
}

func (ctx *TaskContext) SetState(state TaskState) {
	ctx.state = state
}

// helper
func (c *TaskContext) IsAllProductsActive() bool {
	if c.products == nil || len(c.products) == 0 {
		return false
	}

	for _, p := range c.products {
		zap.L().Info("Product Status", zap.String("status", p.Status.String()))
		zap.L().Info("Product Status Check", zap.Bool("is_active", p.Status == enum.ProductStatus("ACTIVED")))
		if p.Status != enum.ProductStatus("ACTIVED") {
			return false
		}
	}
	return true
}
