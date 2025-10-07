package contractsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type ActiveState struct{}

func (a ActiveState) Name() enum.ContractStatus {
	return enum.ContractStatusActive
}

func (a ActiveState) Next(ctx *ContractContext, next ContractState) error {
	if _, ok := a.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsTerminatedAndCascade(next)
		return nil
	}

	return fmt.Errorf("invalid transition: %s -> %s", a.Name(), next.Name())
}

func (a ActiveState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusTerminated: {},
	}
}
