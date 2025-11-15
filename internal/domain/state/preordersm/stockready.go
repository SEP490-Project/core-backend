package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
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
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from STOCK_READY to %s", next.Name())
}
