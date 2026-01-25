package contractsm

import "core-backend/internal/domain/enum"

// BrandViolatedState represents the state when the brand has violated the contract
type BrandViolatedState struct{}

func (s *BrandViolatedState) Name() enum.ContractStatus {
	return enum.ContractStatusBrandViolated
}

func (s *BrandViolatedState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *BrandViolatedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusBrandPenaltyPending: {},
	}
}

// BrandPenaltyPendingState represents the state when brand penalty payment is pending
type BrandPenaltyPendingState struct{}

func (s *BrandPenaltyPendingState) Name() enum.ContractStatus {
	return enum.ContractStatusBrandPenaltyPending
}

func (s *BrandPenaltyPendingState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *BrandPenaltyPendingState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusBrandPenaltyPaid: {},
		enum.ContractStatusTerminated:       {}, // Escalation if payment not made
	}
}

// BrandPenaltyPaidState represents the state when brand has paid the penalty
type BrandPenaltyPaidState struct{}

func (s *BrandPenaltyPaidState) Name() enum.ContractStatus {
	return enum.ContractStatusBrandPenaltyPaid
}

func (s *BrandPenaltyPaidState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *BrandPenaltyPaidState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusTerminated: {}, // Contract is terminated after penalty paid
	}
}
