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
	//case enum.ProductStatusApproved:
	//	return &ApprovedState{}
	case enum.ProductStatusActived:
		return &ActivedState{}
	case enum.ProductStatusInactived:
		return &InActivedState{}
	default:
		return nil
	}
}

func (s *ProductContext) ForwardState(next ProductState) {
	s.State = next
	s.Product.Status = next.Name()
}
