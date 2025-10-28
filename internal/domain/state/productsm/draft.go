package productsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type DraftState struct{}

func (s DraftState) Name() enum.ProductStatus {
	return enum.ProductStatusDraft
}

func (s DraftState) Next(ctx *ProductContext, next ProductState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		if next.Name() == enum.ProductStatusActived && ctx.IsActivable(next) {
			ctx.State = next
		} else if ctx.Product.Type == enum.ProductTypeStandard && next.Name() != enum.ProductStatusActived {
			fmt.Errorf("invalid transition for STANDARD product: %s -> %s", s.Name(), next.Name())
		} else {
			ctx.State = next
		}
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s DraftState) AllowedTransitions() map[enum.ProductStatus]struct{} {
	return map[enum.ProductStatus]struct{}{
		enum.ProductStatusSubmitted: {},
		enum.ProductStatusInactived: {},

		//only STANDARD products can be activated
		enum.ProductStatusActived: {},
	}
}
