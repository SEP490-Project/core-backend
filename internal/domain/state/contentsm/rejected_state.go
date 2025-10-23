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

	// Valid transitions: REJECTED → AWAIT_STAFF or AWAIT_BRAND (after corrections)
	if nextName != enum.ContentStatusAwaitStaff && nextName != enum.ContentStatusAwaitBrand {
		return errors.New("invalid transition: REJECTED can only transition back to AWAIT_STAFF or AWAIT_BRAND")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *RejectedState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusAwaitStaff,
		enum.ContentStatusAwaitBrand,
	}
}
