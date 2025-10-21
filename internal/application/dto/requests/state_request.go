package requests

// UpdateContractStateRequest represents a request to update the state of a contract to a target state.
type UpdateContractStateRequest struct {
	State string `json:"state" validate:"required,oneof=DRAFT APPROVED ACTIVE COMPLETED TERMINATED INACTIVE" example:"TERMINATED"`
}
