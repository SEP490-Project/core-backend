package productsm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
)

type ProductState interface {
	Name() enum.ProductStatus
	Next(ctx *ProductContext, next ProductState) error
	AllowedTransitions() map[enum.ProductStatus]struct{}
}

type ProductContext struct {
	Product *model.Product
	state   ProductState
}

func NewProductContext(product *model.Product, state ProductState) *ProductContext {
	return &ProductContext{Product: product, state: state}
}

func (ctx *ProductContext) SetState(state ProductState) {
	ctx.state = state
	ctx.Product.Status = state.Name()
}

func (ctx *ProductContext) State() ProductState {
	return ctx.state
}
