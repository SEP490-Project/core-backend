package contentsm

import "core-backend/internal/domain/model"

// ContentContext maintains the current state and related entities for Content FSM
type ContentContext struct {
	State ContentState

	// Related entities needed for workflow routing logic
	ContentChannels []*model.ContentChannel
}

// SetState updates the current state
func (ctx *ContentContext) SetState(state ContentState) {
	ctx.State = state
}

// GetState returns the current state
func (ctx *ContentContext) GetState() ContentState {
	return ctx.State
}
