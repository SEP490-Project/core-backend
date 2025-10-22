package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// AwaitBrandState represents content awaiting brand partner review
type AwaitBrandState struct{}

// Name returns the state enum
func (s *AwaitBrandState) Name() enum.ContentStatus {
	return enum.ContentStatusAwaitBrand
}

// Next validates and transitions to the next state
func (s *AwaitBrandState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: AWAIT_BRAND → APPROVED or REJECTED
	if nextName != enum.ContentStatusApproved && nextName != enum.ContentStatusRejected {
		return errors.New("invalid transition: AWAIT_BRAND can only transition to APPROVED or REJECTED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *AwaitBrandState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusApproved,
		enum.ContentStatusRejected,
	}
}
