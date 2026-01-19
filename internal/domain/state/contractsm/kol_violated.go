package contractsm

import "core-backend/internal/domain/enum"

// KOLViolatedState represents the state when the KOL has violated the contract
type KOLViolatedState struct{}

func (s *KOLViolatedState) Name() enum.ContractStatus {
	return enum.ContractStatusKOLViolated
}

func (s *KOLViolatedState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *KOLViolatedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusKOLRefundPending: {},
	}
}

// KOLRefundPendingState represents the state when KOL refund is pending
type KOLRefundPendingState struct{}

func (s *KOLRefundPendingState) Name() enum.ContractStatus {
	return enum.ContractStatusKOLRefundPending
}

func (s *KOLRefundPendingState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *KOLRefundPendingState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusKOLProofSubmitted: {},
		enum.ContractStatusTerminated:        {}, // Escalation if no proof submitted
	}
}

// KOLProofSubmittedState represents the state when KOL has submitted refund proof
type KOLProofSubmittedState struct{}

func (s *KOLProofSubmittedState) Name() enum.ContractStatus {
	return enum.ContractStatusKOLProofSubmitted
}

func (s *KOLProofSubmittedState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *KOLProofSubmittedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusKOLRefundApproved: {},
		enum.ContractStatusKOLProofRejected:  {},
	}
}

// KOLProofRejectedState represents the state when KOL refund proof is rejected
type KOLProofRejectedState struct{}

func (s *KOLProofRejectedState) Name() enum.ContractStatus {
	return enum.ContractStatusKOLProofRejected
}

func (s *KOLProofRejectedState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *KOLProofRejectedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusKOLProofSubmitted: {}, // Can resubmit
		enum.ContractStatusTerminated:        {}, // Escalation if max attempts reached
	}
}

// KOLRefundApprovedState represents the state when KOL refund is approved
type KOLRefundApprovedState struct{}

func (s *KOLRefundApprovedState) Name() enum.ContractStatus {
	return enum.ContractStatusKOLRefundApproved
}

func (s *KOLRefundApprovedState) Next(ctx *ContractContext, next ContractState) error {
	return transition(ctx, s, next, nil)
}

func (s *KOLRefundApprovedState) AllowedTransitions() map[enum.ContractStatus]struct{} {
	return map[enum.ContractStatus]struct{}{
		enum.ContractStatusTerminated: {}, // Contract is terminated after refund approved
	}
}
