package productsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type SubmittedState struct{}

func (s SubmittedState) Name() enum.ProductStatus {
	return enum.ProductStatusSubmitted
}

func (s SubmittedState) Next(ctx *ProductContext, next ProductState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", s.Name(), next.Name())
}

func (s SubmittedState) AllowedTransitions() map[enum.ProductStatus]struct{} {
	return map[enum.ProductStatus]struct{}{
		//Change business rule: Submitted products after brand approve -> when to Actived directly
		//enum.ProductStatusApproved:  {},
		enum.ProductStatusActived:   {},
		enum.ProductStatusRevision:  {},
		enum.ProductStatusInactived: {},
	}
}
