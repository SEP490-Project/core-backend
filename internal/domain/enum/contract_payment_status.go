package enum

import (
	"database/sql/driver"
	"fmt"
)

type ContractPaymentStatus string

const (
	ContractPaymentStatusNotStarted ContractPaymentStatus = "NOT_STARTED"
	ContractPaymentStatusPending    ContractPaymentStatus = "PENDING"
	ContractPaymentStatusPaid       ContractPaymentStatus = "PAID"
	ContractPaymentStatusOverdue    ContractPaymentStatus = "OVERDUE"
	ContractPaymentStatusTerminated ContractPaymentStatus = "TERMINATED"

	// Refund workflow statuses (for CO_PRODUCING contracts when net amount < 0)
	ContractPaymentStatusKOLPending        ContractPaymentStatus = "KOL_PENDING"         // System owes brand, awaiting refund proof
	ContractPaymentStatusKOLProofSubmitted ContractPaymentStatus = "KOL_PROOF_SUBMITTED" // Proof submitted, awaiting brand review
	ContractPaymentStatusKOLProofRejected  ContractPaymentStatus = "KOL_PROOF_REJECTED"  // Brand rejected proof, resubmission needed
	ContractPaymentStatusKOLRefundApproved ContractPaymentStatus = "KOL_REFUND_APPROVED" // Brand approved, refund complete (terminal)
)

func (cps ContractPaymentStatus) IsValid() bool {
	switch cps {
	case ContractPaymentStatusNotStarted,
		ContractPaymentStatusPending,
		ContractPaymentStatusPaid,
		ContractPaymentStatusOverdue,
		ContractPaymentStatusTerminated,
		ContractPaymentStatusKOLPending,
		ContractPaymentStatusKOLProofSubmitted,
		ContractPaymentStatusKOLProofRejected,
		ContractPaymentStatusKOLRefundApproved:
		return true
	}
	return false
}

// IsRefundStatus returns true if this is any of the refund workflow statuses
func (cps ContractPaymentStatus) IsRefundStatus() bool {
	switch cps {
	case ContractPaymentStatusKOLPending,
		ContractPaymentStatusKOLProofSubmitted,
		ContractPaymentStatusKOLProofRejected,
		ContractPaymentStatusKOLRefundApproved:
		return true
	}
	return false
}

// IsTerminalStatus returns true if this is a final status (payment complete)
func (cps ContractPaymentStatus) IsTerminalStatus() bool {
	switch cps {
	case ContractPaymentStatusPaid,
		ContractPaymentStatusTerminated,
		ContractPaymentStatusKOLRefundApproved:
		return true
	}
	return false
}

// IsAwaitingPayment returns true if payment is expected from brand
func (cps ContractPaymentStatus) IsAwaitingPayment() bool {
	return cps == ContractPaymentStatusPending || cps == ContractPaymentStatusOverdue
}

// IsAwaitingRefund returns true if refund proof submission is expected from marketing staff
func (cps ContractPaymentStatus) IsAwaitingRefund() bool {
	return cps == ContractPaymentStatusKOLPending || cps == ContractPaymentStatusKOLProofRejected
}

func (cps *ContractPaymentStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan ContractPaymentStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*cps = ContractPaymentStatus(s)
	return nil
}

func (cps ContractPaymentStatus) Value() (driver.Value, error) {
	return string(cps), nil
}

func (cps ContractPaymentStatus) String() string { return string(cps) }
