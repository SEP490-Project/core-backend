package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// DraftState represents content in draft status
type DraftState struct{}

// Name returns the state enum
func (s *DraftState) Name() enum.ContentStatus {
	return enum.ContentStatusDraft
}

// Next validates and transitions to the next state
func (s *DraftState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: DRAFT → AWAIT_STAFF or AWAIT_BRAND
	if nextName != enum.ContentStatusAwaitStaff && nextName != enum.ContentStatusAwaitBrand {
		return errors.New("invalid transition: DRAFT can only transition to AWAIT_STAFF or AWAIT_BRAND")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *DraftState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusAwaitStaff,
		enum.ContentStatusAwaitBrand,
	}
}
