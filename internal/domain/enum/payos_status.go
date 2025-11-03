package enum

import "database/sql/driver"

// PayOSStatus represents the payment status returned by PayOS API
type PayOSStatus string

const (
	PayOSStatusPending   PayOSStatus = "PENDING"
	PayOSStatusPaid      PayOSStatus = "PAID"
	PayOSStatusCancelled PayOSStatus = "CANCELLED"
	PayOSStatusExpired   PayOSStatus = "EXPIRED"
)

// IsValid checks if the PayOS status is valid
func (s PayOSStatus) IsValid() bool {
	switch s {
	case PayOSStatusPending,
		PayOSStatusPaid,
		PayOSStatusCancelled,
		PayOSStatusExpired:
		return true
	}
	return false
}

// String returns the string representation of the PayOS status
func (s PayOSStatus) String() string {
	return string(s)
}

// Value implements the driver.Valuer interface for database storage
func (s PayOSStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *PayOSStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		*s = PayOSStatus(v)
	case string:
		*s = PayOSStatus(v)
	}
	
	return nil
}

// MapToPaymentTransactionStatus converts PayOS status to internal PaymentTransactionStatus
func (s PayOSStatus) MapToPaymentTransactionStatus() PaymentTransactionStatus {
	switch s {
	case PayOSStatusPending:
		return PaymentTransactionStatusPending
	case PayOSStatusPaid:
		return PaymentTransactionStatusCompleted
	case PayOSStatusCancelled:
		return PaymentTransactionStatusCancelled
	case PayOSStatusExpired:
		return PaymentTransactionStatusExpired
	default:
		// Default to pending for unknown statuses
		return PaymentTransactionStatusPending
	}
}
