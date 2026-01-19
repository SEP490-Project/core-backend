package requests

// UpdateContractStateRequest represents a request to update the state of a contract to a target state.
type UpdateContractStateRequest struct {
	State string `json:"state" validate:"required,oneof=DRAFT APPROVED ACTIVE COMPLETED TERMINATED INACTIVE BRAND_VIOLATED BRAND_PENALTY_PENDING BRAND_PENALTY_PAID KOL_VIOLATED KOL_REFUND_PENDING KOL_PROOF_SUBMITTED KOL_PROOF_REJECTED KOL_REFUND_APPROVED" example:"TERMINATED"`
}
