package contentsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// NewContentState creates the appropriate state instance based on ContentStatus enum
func NewContentState(status enum.ContentStatus) (ContentState, error) {
	switch status {
	case enum.ContentStatusDraft:
		return &DraftState{}, nil
	case enum.ContentStatusAwaitStaff:
		return &AwaitStaffState{}, nil
	case enum.ContentStatusAwaitBrand:
		return &AwaitBrandState{}, nil
	case enum.ContentStatusRejected:
		return &RejectedState{}, nil
	case enum.ContentStatusApproved:
		return &ApprovedState{}, nil
	case enum.ContentStatusPosted:
		return &PostedState{}, nil
	case enum.ContentStatusCancelled:
		return &CancelledState{}, nil
	case enum.ContentStatusScheduled:
		return &ScheduledState{}, nil
	default:
		return nil, fmt.Errorf("unknown content status: %s", status)
	}
}
