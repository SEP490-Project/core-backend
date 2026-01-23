package model

import (
	"core-backend/internal/domain/enum"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PayOSMetadata represents the PayOS-specific data stored in payment_transactions.payos_metadata JSONB column
type PayOSMetadata struct {
	PaymentLinkID      string             `json:"payment_link_id"`
	OrderCode          int64              `json:"order_code"`
	CheckoutURL        string             `json:"checkout_url"`
	QRCode             string             `json:"qr_code"`
	Bin                string             `json:"bin"`
	AccountNumber      string             `json:"account_number"`
	AccountName        string             `json:"account_name"`
	ExpiredAt          int64              `json:"expired_at"`
	Amount             float64            `json:"amount"`
	Description        string             `json:"description"`
	Currency           string             `json:"currency,omitempty"`
	Transactions       []PayOSTransaction `json:"transactions,omitempty"`
	CancelledAt        *time.Time         `json:"cancelled_at,omitempty"`
	CancellationReason *string            `json:"cancellation_reason,omitempty"`
}

// PayOSTransaction represents a transaction detail from PayOS
type PayOSTransaction struct {
	Amount                 int       `json:"amount"`
	Description            string    `json:"description"`
	AccountNumber          string    `json:"account_number"`
	Reference              string    `json:"reference"`
	TransactionDateTime    time.Time `json:"transaction_date_time"`
	CounterAccountBankID   string    `json:"counter_account_bank_id,omitempty"`
	CounterAccountBankName string    `json:"counter_account_bank_name,omitempty"`
	CounterAccountName     string    `json:"counter_account_name,omitempty"`
	CounterAccountNumber   string    `json:"counter_account_number,omitempty"`
	VirtualAccountName     string    `json:"virtual_account_name,omitempty"`
	VirtualAccountNumber   string    `json:"virtual_account_number,omitempty"`
}

// Value implements driver.Valuer interface for JSONB storage
func (m PayOSMetadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements sql.Scanner interface for JSONB retrieval
func (m *PayOSMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, m)
}

type PaymentTransaction struct {
	ID              uuid.UUID                            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReferenceID     uuid.UUID                            `gorm:"type:uuid;not null" json:"reference_id"`
	ReferenceType   enum.PaymentTransactionReferenceType `gorm:"type:varchar(50);not null" json:"reference_type"`
	PayerID         *uuid.UUID                           `gorm:"type:uuid;column:payer_id" json:"payer_id"`
	ReceivedByID    *uuid.UUID                           `gorm:"type:uuid;column:received_by_id" json:"received_by_id"`
	Amount          *float64                             `gorm:"type:decimal(15,2);not null" json:"amount"`
	Method          string                               `gorm:"type:varchar(50);not null" json:"method"`
	Status          enum.PaymentTransactionStatus        `gorm:"type:varchar(50);not null" json:"status"`
	TransactionDate time.Time                            `gorm:"type:timestamptz;default:current_timestamp" json:"transaction_date"`
	GatewayRef      string                               `gorm:"type:varchar(255)" json:"gateway_ref"`
	GatewayID       string                               `gorm:"type:varchar(255)" json:"gateway_id"`
	PayOSMetadata   *PayOSMetadata                       `gorm:"column:payos_metadata;type:jsonb" json:"payos_metadata,omitempty"`
	CreatedAt       time.Time                            `gorm:"type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time                            `gorm:"type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (PaymentTransaction) TableName() string {
	return "payment_transactions"
}

// BeforeCreate hook to ensure ID is set
func (pt *PaymentTransaction) BeforeCreate(tx *gorm.DB) error {
	if pt.ID == uuid.Nil {
		pt.ID = uuid.New()
	}
	return nil
}
