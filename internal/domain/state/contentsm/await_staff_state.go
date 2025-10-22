package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// AwaitStaffState represents content awaiting internal staff review
type AwaitStaffState struct{}

// Name returns the state enum
func (s *AwaitStaffState) Name() enum.ContentStatus {
	return enum.ContentStatusAwaitStaff
}

// Next validates and transitions to the next state
func (s *AwaitStaffState) Next(ctx *ContentContext, nextState ContentState) error {
	nextName := nextState.Name()

	// Valid transitions: AWAIT_STAFF → APPROVED or REJECTED
	if nextName != enum.ContentStatusApproved && nextName != enum.ContentStatusRejected {
		return errors.New("invalid transition: AWAIT_STAFF can only transition to APPROVED or REJECTED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *AwaitStaffState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusApproved,
		enum.ContentStatusRejected,
	}
}
