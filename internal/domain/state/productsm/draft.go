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
		if err := s.statePrerequisite(ctx, next); err != nil {
			return err
		}
		ctx.ForwardState(next)
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

func (s DraftState) statePrerequisite(ctx *ProductContext, nextState ProductState) error {
	prd := ctx.Product
	if prd.Type == enum.ProductTypeStandard && nextState.Name() != enum.ProductStatusActived {
		return fmt.Errorf("STANDARD products must transition to ACTIVED state from SUBMITTED")
	} else if prd.Type != enum.ProductTypeStandard && nextState.Name() == enum.ProductStatusActived {
		return fmt.Errorf("only STANDARD products can transition to ACTIVED state from SUBMITTED")
	}

	switch nextState.Name() {
	case enum.ProductStatusActived:
		if prd.Type != enum.ProductTypeStandard {
			return fmt.Errorf("only STANDARD products can transition to ACTIVED state from SUBMITTED")
		}
		if prd.Variants == nil || len(prd.Variants) == 0 {
			return fmt.Errorf("cannot activate product without variants")
		}
	case enum.ProductStatusSubmitted:
		if prd.Variants == nil || len(prd.Variants) == 0 {
			return fmt.Errorf("cannot submit product without variants")
		}
	}
	return nil
}
