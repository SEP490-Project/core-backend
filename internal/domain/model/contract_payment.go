package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContractPayment struct {
	ID                    uuid.UUID                  `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ContractID            uuid.UUID                  `json:"contract_id" gorm:"type:uuid;column:contract_id;not null"`
	InstallmentPercentage float64                    `json:"installment_percentage" gorm:"column:installment_percentage;not null"`
	Amount                float64                    `json:"amount" gorm:"column:amount;not null"`
	Status                enum.ContractPaymentStatus `json:"status" gorm:"column:status;not null;check:status IN ('PENDING','PAID','OVERDUE')"`
	DueDate               time.Time                  `json:"due_date" gorm:"column:due_date;not null"`
	PaymentMethod         enum.ContractPaymentMethod `json:"payment_method" gorm:"column:payment_method;not null;check:payment_method IN ('BANK_TRANSFER','CASH','CHECK')"`
	Note                  *string                    `json:"note" gorm:"type:text;column:note"`
	CreatedAt             time.Time                  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time                  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt             gorm.DeletedAt             `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	Contract *Contract `json:"-" gorm:"foreignKey:ContractID"`
}

func (ContractPayment) TableName() string { return "contract_payments" }

func (cp *ContractPayment) BeforeCreate(tx *gorm.DB) error {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	if cp.InstallmentPercentage < 0 {
		zap.L().Warn("InstallmentPercentage is less than 0, setting to 0")
		cp.InstallmentPercentage = 0
	}
	if cp.InstallmentPercentage > 100 {
		zap.L().Warn("InstallmentPercentage is greater than 100, setting to 100")
		cp.InstallmentPercentage = 100
	}

	return nil
}
