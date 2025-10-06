package contractsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type CompletedState struct{}

func (c CompletedState) Name() enum.ContractStatus {
	return enum.ContractStatusCompleted
}

func (c CompletedState) Next(ctx *ContractContext, next ContractState) error {
	return fmt.Errorf("invalid transition: The state is final and cannot transition to another state")
}

func (c CompletedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{}
}
