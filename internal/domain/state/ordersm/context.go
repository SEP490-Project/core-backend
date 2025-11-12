package ordersm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"time"

	"github.com/aws/smithy-go/ptr"
	"go.uber.org/zap"
)

type OrderContext struct {
	State OrderState
	Order *model.Order
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
		if reason == nil {
			reason = ptr.String("Order being cancelled by User")
		}
		note.ActionType = enum.OrderStatusCancelled
		note.Reason = *reason
	case enum.OrderStatusRefunded:
		if reason == nil {
			reason = ptr.String("Order being cancelled by user and refunded")
		}
		note.ActionType = enum.OrderStatusRefunded
		note.Reason = *reason
	default:
		msg := fmt.Sprintf("Unhandled state for generating action note: %s", s.State.Name())
		zap.L().Info(msg)
		return nil
	}
	return note
}
