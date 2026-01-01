package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// ApprovedState represents approved content ready for publishing
type ApprovedState struct{}

// Name returns the state enum
func (s *ApprovedState) Name() enum.ContentStatus {
	return enum.ContentStatusApproved
}

// Next validates and transitions to the next state
func (s *ApprovedState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: APPROVED → POSTED or CANCELLED
	if nextName != enum.ContentStatusPosted && nextName != enum.ContentStatusCancelled {
		return errors.New("invalid transition: APPROVED can only transition to POSTED or CANCELLED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *ApprovedState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusPosted,
		enum.ContentStatusCancelled,
	}
}
