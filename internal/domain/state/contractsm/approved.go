package contractsm

import (
	"core-backend/internal/domain/enum"
)

type ApprovedState struct{}

func (t ApprovedState) Name() enum.ContractStatus {
	return enum.ContractStatusApproved
}

func (t ApprovedState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, t, next, func(next ContractState) {
		ctx.IsTerminatedAndCascade(next)
	})
}

func (t ApprovedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusActive:     {},
		enum.ContractStatusTerminated: {},
	}
}
