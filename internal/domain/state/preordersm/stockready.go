package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
	"time"
)

// StockReadyState represents STOCK_READY pre-order status
type StockReadyState struct{}

func (s *StockReadyState) Name() enum.PreOrderStatus { return enum.PreOrderStatusStockReady }
func (s *StockReadyState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusAwaitingPickup: true,
		enum.PreOrderStatusInTransit:      true,
		enum.PreOrderStatusCancelled:      true,
	}
}
func (s *StockReadyState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		err := s.statePrerequisite(ctx, next)
		if err != nil {
			return err
		}
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from STOCK_READY to %s", next.Name())
}

func (s *StockReadyState) statePrerequisite(ctx *PreOrderContext, nextState PreOrderState) error {
	isSelfPick := ctx.PreOrder.IsSelfPickedUp

	if isSelfPick && nextState.Name() == enum.PreOrderStatusInTransit {
		return fmt.Errorf("self-pickup pre-orders cannot transition to IN_TRANSIT")
	}

	if !isSelfPick && nextState.Name() == enum.PreOrderStatusAwaitingPickup {
		return fmt.Errorf("non-self-pickup pre-orders cannot transition to AWAITING_PICKUP")
	}

	isBeforeStartDate := time.Now().Before(ctx.LimitedProduct.AvailabilityStartDate)
	isAfterEndDate := time.Now().After(ctx.LimitedProduct.AvailabilityEndDate)
	if isBeforeStartDate && isAfterEndDate {
		return fmt.Errorf("cannot transition pre-order outside of availability window")
	}

	// Stock is already ready, no action needed
	return nil
}
