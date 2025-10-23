package contentsm

import "core-backend/internal/domain/enum"

// ContentState defines the interface for all content states in the FSM
type ContentState interface {
	// Name returns the enum name of the current state
	Name() enum.ContentStatus

	// Next transitions to the next state with validation
	Next(ctx *ContentContext, nextState ContentState) error

	// AllowedTransitions returns the list of valid next states
	AllowedTransitions() []enum.ContentStatus
}
