package contractsm

import (
	"core-backend/internal/domain/enum"
)

type DraftState struct{}

func (d DraftState) Name() enum.ContractStatus {
	return enum.ContractStatusDraft
}

func (d DraftState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, d, next, func(next ContractState) bool {
		ctx.IsTerminatedAndCascade(next)
		return true
	})
}

func (d DraftState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusApproved:   {},
		enum.ContractStatusTerminated: {},
	}
}
