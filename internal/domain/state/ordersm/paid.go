package ordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type PaidState struct{}

func (p PaidState) Name() enum.OrderStatus {
	return enum.OrderStatusPaid
}

func (p PaidState) Next(ctx *OrderContext, next OrderState) error {
	if ctx.ActionBy == nil {
		return fmt.Errorf("action user is required for state transition")
	}
	if _, ok := p.AllowedTransitions()[next.Name()]; ok {
		if p.isRoleStaff(ctx.ActionBy.Role) && next.Name() != enum.OrderStatusRefunded {
			return fmt.Errorf("user role %s not allowed to transition from %s to %s", ctx.ActionBy.Role, p.Name(), next.Name())
		} else if !p.isRoleStaff(ctx.ActionBy.Role) && next.Name() == enum.OrderStatusRefunded {
			return fmt.Errorf("user role %s not allowed to transition from %s to %s", ctx.ActionBy.Role, p.Name(), next.Name())
		}
		ctx.ForwardState(next)
		return nil
	}
	return fmt.Errorf("invalid transition: %s -> %s", p.Name(), next.Name())
}

func (p PaidState) AllowedTransitions() map[enum.OrderStatus]struct{} {
	return map[enum.OrderStatus]struct{}{
		enum.OrderStatusConfirmed:       {},
		enum.OrderStatusRefunded:        {},
		enum.OrderStatusRefundRequested: {},
	}
}

func (p PaidState) isRoleStaff(role enum.UserRole) bool {
	allowedRole := []enum.UserRole{
		enum.UserRoleAdmin,
		enum.UserRoleSalesStaff,
	}
	for _, r := range allowedRole {
		if role == r {
			return true
		}
	}
	return false
}
