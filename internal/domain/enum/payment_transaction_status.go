package enum

import "database/sql/driver"

// PaymentTransactionStatus represents the status of a payment transaction
type PaymentTransactionStatus string

const (
	PaymentTransactionStatusPending   PaymentTransactionStatus = "PENDING"
	PaymentTransactionStatusCompleted PaymentTransactionStatus = "COMPLETED"
	PaymentTransactionStatusFailed    PaymentTransactionStatus = "FAILED"
	PaymentTransactionStatusCancelled PaymentTransactionStatus = "CANCELLED"
	PaymentTransactionStatusExpired   PaymentTransactionStatus = "EXPIRED"
)

// IsValid checks if the payment transaction status is valid
func (s PaymentTransactionStatus) IsValid() bool {
	switch s {
	case PaymentTransactionStatusPending,
		PaymentTransactionStatusCompleted,
		PaymentTransactionStatusFailed,
		PaymentTransactionStatusCancelled,
		PaymentTransactionStatusExpired:
		return true
	}
	return false
}

// String returns the string representation of the payment transaction status
func (s PaymentTransactionStatus) String() string {
	return string(s)
}

// Value implements the driver.Valuer interface for database storage
func (s PaymentTransactionStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *PaymentTransactionStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		*s = PaymentTransactionStatus(v)
	case string:
		*s = PaymentTransactionStatus(v)
	}
	
	return nil
}
