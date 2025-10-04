package productsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type InActivedState struct{}

func (a InActivedState) Name() enum.ProductStatus {
	return enum.ProductStatusActived
}

func (a InActivedState) Next(ctx *ProductContext, next ProductState) error {
	return fmt.Errorf("invalid transition: " + "The state is final and cannot transition to another state")
}

func (a InActivedState) AllowedTransitions() map[enum.ProductStatus]struct{} {
	return map[enum.ProductStatus]struct{}{}
}
