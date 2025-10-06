package contractsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type DraftState struct{}

func (d DraftState) Name() enum.ContractStatus {
	return enum.ContractStatusDraft
}

func (p DraftState) Next(ctx *ContractContext, next ContractState) error {

	if _, ok := p.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		ctx.IsTerminatedAndCascade(next)
		return nil
	}

	return fmt.Errorf("invalid transition: %s -> %s", p.Name(), next.Name())
}

func (p DraftState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusActive:     {},
		enum.ContractStatusTerminated: {},
	}
}
