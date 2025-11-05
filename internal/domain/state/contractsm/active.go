package contractsm

import (
	"core-backend/internal/domain/enum"
)

type ActiveState struct{}

func (a ActiveState) Name() enum.ContractStatus {
	return enum.ContractStatusActive
}

func (a ActiveState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, a, next, func(next ContractState) bool {
		ctx.IsTerminatedAndCascade(next)
		return true
	})
}

func (a ActiveState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusTerminated: {},
	}
}
