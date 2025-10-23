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
	// Terminal state - no transitions allowed
	return errors.New("invalid transition: POSTED is a terminal state, no further transitions allowed")
}

// AllowedTransitions returns valid next states (none for terminal state)
func (s *PostedState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{}
}
