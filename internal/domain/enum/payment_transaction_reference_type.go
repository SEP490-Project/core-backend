package enum

import "database/sql/driver"

// PaymentTransactionReferenceType represents the type of reference for a payment transaction
type PaymentTransactionReferenceType string

const (
	PaymentTransactionReferenceTypeOrder           PaymentTransactionReferenceType = "ORDER"
	PaymentTransactionReferenceTypeContractPayment PaymentTransactionReferenceType = "CONTRACT_PAYMENT"
	PaymentTransactionReferenceTypePreOrder        PaymentTransactionReferenceType = "PREORDER"
)

// IsValid checks if the payment transaction status is valid
func (rt PaymentTransactionReferenceType) IsValid() bool {
	switch rt {
	case PaymentTransactionReferenceTypeOrder,
		PaymentTransactionReferenceTypeContractPayment,
		PaymentTransactionReferenceTypePreOrder:
		return true
	}
	return false
}

// String returns the string representation of the payment transaction status
func (rt PaymentTransactionReferenceType) String() string {
	return string(rt)
}

// Value implements the driver.Valuer interface for database storage
func (rt PaymentTransactionReferenceType) Value() (driver.Value, error) {
	return string(rt), nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (rt *PaymentTransactionReferenceType) Scan(value any) error {
	if value == nil {
		*rt = ""
		return nil
	}

	switch v := value.(type) {
	case []byte:
		*rt = PaymentTransactionReferenceType(v)
	case string:
		*rt = PaymentTransactionReferenceType(v)
	}

	return nil
}
