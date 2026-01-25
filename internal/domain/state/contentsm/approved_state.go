package contentsm

import (
	"core-backend/internal/domain/enum"
	"errors"
	"slices"

	"go.uber.org/zap"
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

	// // Valid transitions: APPROVED → POSTED, SCHEDULED, or CANCELLED
	if !slices.Contains(s.AllowedTransitions(), nextName) {
		zap.L().Warn("invalid state transition attempted",
			zap.String("from", s.Name().String()),
			zap.String("to", nextName.String()),
			zap.Any("allowed_transitions", s.AllowedTransitions()))
		return errors.New("invalid transition: APPROVED can only transition to POSTED, SCHEDULED, or CANCELLED")
	}

	ctx.SetState(nextState)
	return nil
}

// AllowedTransitions returns valid next states
func (s *ApprovedState) AllowedTransitions() []enum.ContentStatus {
	return []enum.ContentStatus{
		enum.ContentStatusPosted,
		enum.ContentStatusScheduled,
		enum.ContentStatusCancelled,
	}
}
