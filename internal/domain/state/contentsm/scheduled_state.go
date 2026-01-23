package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
	"slices"
)

// ScheduledState represents content that has been rejected
type ScheduledState struct{}

// Name returns the state enum
func (s *ScheduledState) Name() enum.ContentStatus {
	return enum.ContentStatusRejected
}

// Next validates and transitions to the next state
func (s *ScheduledState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: SCHEDULED → POSTED or CANCELLED
	if !slices.Contains(s.AllowedTransitions(), nextName) {
		return errors.New("invalid transition: REJECTED can only transition to POSTED or CANCELLED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *ScheduledState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusPosted,
		enum.ContentStatusCancelled,
	}
}
