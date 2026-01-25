package enum

import (
	"database/sql/driver"
	"fmt"
)

// ContractStatus represents the status of a contract.
// Possible values are: "DRAFT", "APPROVED", "ACTIVE", "COMPLETED", "INACTIVE", "TERMINATED"
// and violation statuses: "BRAND_VIOLATED", "BRAND_PENALTY_PENDING", "BRAND_PENALTY_PAID",
// "KOL_VIOLATED", "KOL_REFUND_PENDING", "KOL_PROOF_SUBMITTED", "KOL_PROOF_REJECTED", "KOL_REFUND_APPROVED"
type ContractStatus string

const (
	ContractStatusDraft      ContractStatus = "DRAFT"
	ContractStatusApproved   ContractStatus = "APPROVED"
	ContractStatusActive     ContractStatus = "ACTIVE"
	ContractStatusCompleted  ContractStatus = "COMPLETED"
	ContractStatusInactive   ContractStatus = "INACTIVE"
	ContractStatusTerminated ContractStatus = "TERMINATED"

	// Brand Violation Path
	ContractStatusBrandViolated       ContractStatus = "BRAND_VIOLATED"
	ContractStatusBrandPenaltyPending ContractStatus = "BRAND_PENALTY_PENDING"
	ContractStatusBrandPenaltyPaid    ContractStatus = "BRAND_PENALTY_PAID"

	// KOL Violation Path
	ContractStatusKOLViolated       ContractStatus = "KOL_VIOLATED"
	ContractStatusKOLRefundPending  ContractStatus = "KOL_REFUND_PENDING"
	ContractStatusKOLProofSubmitted ContractStatus = "KOL_PROOF_SUBMITTED"
	ContractStatusKOLProofRejected  ContractStatus = "KOL_PROOF_REJECTED"
	ContractStatusKOLRefundApproved ContractStatus = "KOL_REFUND_APPROVED"
)

func (cs ContractStatus) IsValid() bool {
	switch cs {
	case ContractStatusDraft, ContractStatusApproved, ContractStatusActive,
		ContractStatusCompleted, ContractStatusInactive, ContractStatusTerminated,
		ContractStatusBrandViolated, ContractStatusBrandPenaltyPending, ContractStatusBrandPenaltyPaid,
		ContractStatusKOLViolated, ContractStatusKOLRefundPending, ContractStatusKOLProofSubmitted,
		ContractStatusKOLProofRejected, ContractStatusKOLRefundApproved:
		return true
	}
	return false
}

// IsViolationState returns true if the contract is in any violation state
func (cs ContractStatus) IsViolationState() bool {
	switch cs {
	case ContractStatusBrandViolated, ContractStatusBrandPenaltyPending, ContractStatusBrandPenaltyPaid,
		ContractStatusKOLViolated, ContractStatusKOLRefundPending, ContractStatusKOLProofSubmitted,
		ContractStatusKOLProofRejected, ContractStatusKOLRefundApproved:
		return true
	}
	return false
}

// GetViolationType returns the violation type based on contract status
func (cs ContractStatus) GetViolationType() ViolationType {
	switch cs {
	case ContractStatusBrandViolated, ContractStatusBrandPenaltyPending, ContractStatusBrandPenaltyPaid:
		return ViolationTypeBrand
	case ContractStatusKOLViolated, ContractStatusKOLRefundPending, ContractStatusKOLProofSubmitted,
		ContractStatusKOLProofRejected, ContractStatusKOLRefundApproved:
		return ViolationTypeKOL
	}
	return ""
}

func (cs *ContractStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContractStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cs = ContractStatus(s)
	return nil
}

func (cs ContractStatus) Value() (driver.Value, error) {
	return string(cs), nil
}

func (cs ContractStatus) String() string {
	return string(cs)
}
