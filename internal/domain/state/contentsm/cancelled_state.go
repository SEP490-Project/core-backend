package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// CancelledState represents content that has been cancelled (terminal state)
// This state is reached when:
// - Contract is terminated and content cascade is triggered
// - Content is manually cancelled/deleted
// - Any other business rule requiring content cancellation
type CancelledState struct{}

// Name returns the state enum
func (s *CancelledState) Name() enum.ContentStatus {
	return enum.ContentStatusCancelled
}

// Next validates and transitions to the next state
func (s *CancelledState) Next(ctx *ContentContext, nextState ContentState) error {
	// CANCELLED is a terminal state - no transitions allowed
	return errors.New("invalid transition: CANCELLED is a terminal state, no further transitions allowed")
}

// AllowedTransitions returns valid next states (none for terminal state)
func (s *CancelledState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{}
}
