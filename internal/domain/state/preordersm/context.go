package preordersm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"time"

	"github.com/aws/smithy-go/ptr"
)

// PreOrderContext holds the current state and related data for FSM
type PreOrderContext struct {
	State          PreOrderState
	PreOrder       *model.PreOrder
	LimitedProduct *model.LimitedProduct
	ActionBy       *model.User
}

func (s *PreOrderContext) GenerateActionNote(user *model.User, reason *string) *model.PreOrderActionNote {
	note := &model.PreOrderActionNote{
		UserID:    user.ID,
		UserName:  user.FullName,
		UserEmail: user.Email,
		CreatedAt: time.Time{},
	}

	switch s.State.Name() {
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
	case enum.PreOrderStatusShipped:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being shipped By GHN")
		}
		note.ActionType = enum.PreOrderStatusShipped
		note.Reason = *reason
	case enum.PreOrderStatusAwaitingPickup:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being changed By System")
		}
		note.ActionType = enum.PreOrderStatusAwaitingPickup
		note.Reason = *reason
	case enum.PreOrderStatusInTransit:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being in transit by Delivery Service")
		}
		note.ActionType = enum.PreOrderStatusInTransit
		note.Reason = *reason
	case enum.PreOrderStatusDelivered:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Your Pre-order has been delivered")
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
	case enum.PreOrderStatusRefundRequest:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being request refund by User")
		}
		note.ActionType = enum.PreOrderStatusRefundRequest
		note.Reason = *reason
	case enum.PreOrderStatusRefunded:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Pre-order being refunded by Staff")
		}
		note.ActionType = enum.PreOrderStatusRefunded
		note.Reason = *reason
	}
	return note
}

func (s *PreOrderContext) ForwardState(next PreOrderState) {
	s.State = next
	s.PreOrder.Status = next.Name()
}
