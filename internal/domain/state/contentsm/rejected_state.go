package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// RejectedState represents content that has been rejected
type RejectedState struct{}

// Name returns the state enum
func (s *RejectedState) Name() enum.ContentStatus {
	return enum.ContentStatusRejected
}

// Next validates and transitions to the next state
func (s *RejectedState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: REJECTED → AWAIT_STAFF, AWAIT_BRAND (after corrections), or CANCELLED
	if nextName != enum.ContentStatusAwaitStaff && nextName != enum.ContentStatusAwaitBrand && nextName != enum.ContentStatusCancelled {
		return errors.New("invalid transition: REJECTED can only transition to AWAIT_STAFF, AWAIT_BRAND, or CANCELLED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *RejectedState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusAwaitStaff,
		enum.ContentStatusAwaitBrand,
		enum.ContentStatusCancelled,
	}
}
