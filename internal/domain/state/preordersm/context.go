package preordersm

import (
	"core-backend/internal/domain/model"
)

// PreOrderContext holds the current state and related data for FSM
type PreOrderContext struct {
	State          PreOrderState
	LimitedProduct *model.LimitedProduct
}
