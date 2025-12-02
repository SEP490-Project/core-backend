package ordersm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"time"

	"github.com/aws/smithy-go/ptr"
	"go.uber.org/zap"
)

type OrderContext struct {
	State    OrderState
	Order    *model.Order
	ActionBy *model.User
}

func (s *OrderContext) GenerateActionNote(user *model.User, reason *string) *model.OrderActionNote {
	note := &model.OrderActionNote{
		UserID:    user.ID,
		UserName:  user.FullName,
		UserEmail: user.Email,
		CreatedAt: time.Time{},
	}

	switch s.State.Name() {
	case enum.OrderStatusCancelled:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being cancelled by User")
		}
		note.ActionType = enum.OrderStatusCancelled
		note.Reason = *reason
	case enum.OrderStatusPaid:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being paid by User")
		}
		note.ActionType = enum.OrderStatusPaid
		note.Reason = *reason
	case enum.OrderStatusRefundRequested:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being request refund by user")
		}
		note.ActionType = enum.OrderStatusRefundRequested
		note.Reason = *reason
	case enum.OrderStatusRefunded:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being refunded by Staff")
		}
		note.ActionType = enum.OrderStatusRefunded
		note.Reason = *reason
	case enum.OrderStatusConfirmed:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being confirmed by Staff")
		}
		note.ActionType = enum.OrderStatusConfirmed
		note.Reason = *reason
	case enum.OrderStatusShipped:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being shipped by Staff")
		}
		note.ActionType = enum.OrderStatusShipped
		note.Reason = *reason
	case enum.OrderStatusInTransit:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being in transit by Delivery Service")
		}
		note.ActionType = enum.OrderStatusInTransit
		note.Reason = *reason
	case enum.OrderStatusDelivered:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being delivered by Delivery Service")
		}
		note.ActionType = enum.OrderStatusDelivered
		note.Reason = *reason
	case enum.OrderStatusReceived:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being received by User")
		}
		note.ActionType = enum.OrderStatusReceived
		note.Reason = *reason
	case enum.OrderStatusCompensateRequested:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being request compensation by user")
		}
		note.ActionType = enum.OrderStatusCompensateRequested
		note.Reason = *reason
	case enum.OrderStatusCompensated:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order being compensated by Staff")
		}
		note.ActionType = enum.OrderStatusCompensated
		note.Reason = *reason
	case enum.OrderStatusAwaitingPickUp:
		if !utils.NotEmptyOrNil(reason) {
			reason = ptr.String("Order are ready to picked up by User")
		}
		note.ActionType = enum.OrderStatusAwaitingPickUp
		note.Reason = *reason
	default:
		msg := fmt.Sprintf("Unhandled state for generating action note: %s", s.State.Name())

		zap.L().Info(msg)
		return nil
	}
	return note
}

func (s *OrderContext) ForwardState(next OrderState) {
	s.State = next
	s.Order.Status = next.Name()
}
