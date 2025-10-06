package contractsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

type TerminatedState struct{}

func (t TerminatedState) Name() enum.ContractStatus {
	return enum.ContractStatusTerminated
}

func (t TerminatedState) Next(ctx *ContractContext, next ContractState) error {
	return fmt.Errorf("invalid transition: %s -> %s", t.Name(), next.Name())
}

func (t TerminatedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{}
}
