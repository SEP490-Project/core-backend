package model

import (
	"time"

	"github.com/google/uuid"
)

type PaymentTransaction struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ReferenceID     uuid.UUID `gorm:"type:uuid;not null" json:"reference_id"`
	ReferenceType   string    `gorm:"type:varchar(50);not null" json:"reference_type"`
	Amount          *float64  `gorm:"type:decimal(15,2);not null" json:"amount"`
	Method          string    `gorm:"type:varchar(50);not null" json:"method"`
	Status          string    `gorm:"type:varchar(50);not null" json:"status"`
	TransactionDate time.Time `gorm:"type:timestamptz;default:current_timestamp" json:"transaction_date"`
	GatewayRef      string    `gorm:"type:varchar(255)" json:"gateway_ref"`
	UpdatedAt       time.Time `gorm:"type:timestamptz;default:current_timestamp" json:"updated_at"`
}
