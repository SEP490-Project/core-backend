package productsm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
)

type ProductContext struct {
	State   ProductState
	Product model.Product
}

type ProductState interface {
	Name() enum.ProductStatus
	Next(ctx *ProductContext, next ProductState) error
	AllowedTransitions() map[enum.ProductStatus]struct{}
}

func NewProductState(status enum.ProductStatus) ProductState {
	switch status {
	case enum.ProductStatusDraft:
		return &DraftState{}
	case enum.ProductStatusSubmitted:
		return &SubmittedState{}
	case enum.ProductStatusRevision:
		return &RevisionState{}
	case enum.ProductStatusApproved:
		return &ApprovedState{}
	case enum.ProductStatusActived:
		return &ActivedState{}
	case enum.ProductStatusInactived:
		return &InActivedState{}
	default:
		return nil
	}
}

// IsActivable checks if the product can transition into ACTIVED state.
// For STANDARD products we allow activation from DRAFT or APPROVED states.
func (c *ProductContext) IsActivable(state ProductState) bool {
	if state.Name() != enum.ProductStatusActived {
		return false
	}

	if c.Product.Type != enum.ProductTypeStandard {
		return false
	}

	current := c.State.Name()
	// allow activation from Draft or Approved for STANDARD products
	if current == enum.ProductStatusDraft || current == enum.ProductStatusApproved {
		return true
	}

	return false
}
