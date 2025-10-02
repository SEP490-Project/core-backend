package productsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type RevisionState struct{}

func (r RevisionState) Name() enum.ProductStatus {
	return enum.ProductStatusRevision
}

func (r RevisionState) Next(ctx *ProductContext, next ProductState) error {
	if _, ok := r.AllowedTransitions()[next.Name()]; ok {
		ctx.SetState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", r.Name(), next.Name())
}

func (r RevisionState) AllowedTransitions() map[enum.ProductStatus]struct{} {
	return map[enum.ProductStatus]struct{}{
		enum.ProductStatusDraft: {},
	}
}
