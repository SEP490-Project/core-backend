package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// PostedState represents published content (terminal state)
type PostedState struct{}

// Name returns the state enum
func (s *PostedState) Name() enum.ContentStatus {
	return enum.ContentStatusPosted
}

// Next validates and transitions to the next state
func (s *PostedState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: POSTED → CANCELLED only
	// Note: When cancelling POSTED content, social media unpublish should be triggered
	if nextName != enum.ContentStatusCancelled {
		return errors.New("invalid transition: POSTED can only transition to CANCELLED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *PostedState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusCancelled,
	}
}
