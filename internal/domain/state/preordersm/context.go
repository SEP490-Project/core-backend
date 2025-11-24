package preordersm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"github.com/aws/smithy-go/ptr"
	"time"
)

// PreOrderContext holds the current state and related data for FSM
type PreOrderContext struct {
	State          PreOrderState
	PreOrder       *model.PreOrder
	LimitedProduct *model.LimitedProduct
	ActionBy       *model.User
}

func (p *PreOrderContext) GenerateActionNote(user *model.User, reason *string) *model.PreOrderActionNote {
	note := &model.PreOrderActionNote{
		UserID:    user.ID,
		UserName:  user.FullName,
		UserEmail: user.Email,
		CreatedAt: time.Time{},
	}

	switch p.State.Name() {
	case enum.PreOrderStatusCancelled:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being expired by System")
		}
		note.ActionType = enum.PreOrderStatusCancelled
		note.Reason = *reason
	case enum.PreOrderStatusPaid:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being paid by User")
		}
		note.ActionType = enum.PreOrderStatusPaid
		note.Reason = *reason
	case enum.PreOrderStatusPreOrdered:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being Censored By Staff")
		}
		note.ActionType = enum.PreOrderStatusPreOrdered
		note.Reason = *reason
	case enum.PreOrderStatusAwaitingPickup:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being changed By System")
		}
		note.ActionType = enum.PreOrderStatusAwaitingPickup
		note.Reason = *reason
	case enum.PreOrderStatusInTransit:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being changed By System")
		}
		note.ActionType = enum.PreOrderStatusInTransit
		note.Reason = *reason
	case enum.PreOrderStatusDelivered:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being confirmed delivered By Staff")
		}
		note.ActionType = enum.PreOrderStatusDelivered
		note.Reason = *reason
	case enum.PreOrderStatusCompensateRequest:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being request compensate by User")
		}
		note.ActionType = enum.PreOrderStatusCompensateRequest
		note.Reason = *reason
	case enum.PreOrderStatusCompensated:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being compensated by Staff")
		}
		note.ActionType = enum.PreOrderStatusCompensated
		note.Reason = *reason
	case enum.PreOrderStatusReceived:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being received by User")
		}
		note.ActionType = enum.PreOrderStatusReceived
		note.Reason = *reason
	}
	return note
}

func (s *PreOrderContext) ForwardState(next PreOrderState) {
	s.State = next
	s.PreOrder.Status = next.Name()
}
