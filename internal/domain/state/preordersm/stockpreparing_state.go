package preordersm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// StockPreparingState represents STOCK_PREPARING pre-order status
type StockPreparingState struct{}

func (s *StockPreparingState) Name() enum.PreOrderStatus { return enum.PreOrderStatusStockPreparing }
func (s *StockPreparingState) AllowedTransitions() map[enum.PreOrderStatus]bool {
	return map[enum.PreOrderStatus]bool{
		enum.PreOrderStatusStockReady: true,
		enum.PreOrderStatusCancelled:  true,
	}
}
func (s *StockPreparingState) Next(ctx *PreOrderContext, next PreOrderState) error {
	if _, ok := s.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from STOCK_PREPARING to %s", next.Name())
}
