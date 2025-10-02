package productsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ActivedState struct{}

func (a ActivedState) Name() enum.ProductStatus {
	return enum.ProductStatusActived
}

func (a ActivedState) Next(ctx *ProductContext, next ProductState) error {
	return fmt.Errorf("invalid transition: " + "The state is final and cannot transition to another state")
}

func (a ActivedState) AllowedTransitions() map[enum.ProductStatus]struct{} {
	return map[enum.ProductStatus]struct{}{}
}
