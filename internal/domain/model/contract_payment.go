package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContractPayment struct {
	ID                    uuid.UUID                  `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ContractID            uuid.UUID                  `json:"contract_id" gorm:"type:uuid;column:contract_id;not null"`
	MilestoneID           *uuid.UUID                 `json:"milestone_id" gorm:"type:uuid;column:milestone_id"` // Links payment to campaign milestone
	InstallmentPercentage float64                    `json:"installment_percentage" gorm:"column:installment_percentage;not null"`
	Amount                float64                    `json:"amount" gorm:"column:amount;not null"`
	BaseAmount            float64                    `json:"base_amount" gorm:"column:base_amount;not null"`
	PerformanceAmount     float64                    `json:"performance_amount" gorm:"column:performance_amount"`
	Status                enum.ContractPaymentStatus `json:"status" gorm:"column:status;not null;check:status IN ('PENDING','PAID','OVERDUE')"`
	DueDate               time.Time                  `json:"due_date" gorm:"column:due_date;not null"`
	PaymentMethod         enum.ContractPaymentMethod `json:"payment_method" gorm:"column:payment_method;not null;check:payment_method IN ('BANK_TRANSFER','CASH','CHECK')"`
	Note                  *string                    `json:"note" gorm:"type:text;column:note"`
	IsDeposit             bool                       `json:"is_deposit" gorm:"column:is_deposit;not null;default:false"`
	CreatedAt             time.Time                  `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt             time.Time                  `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	CreatedBy             *uuid.UUID                 `json:"created_by" gorm:"type:uuid;column:created_by"`
	UpdatedBy             *uuid.UUID                 `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	DeletedAt             gorm.DeletedAt             `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Payment period fields (for AFFILIATE/CO_PRODUCING contracts)
	PeriodStart *time.Time `json:"period_start" gorm:"column:period_start"`
	PeriodEnd   *time.Time `json:"period_end" gorm:"column:period_end"`

	// Calculation tracking fields
	CalculatedAt         *time.Time     `json:"calculated_at" gorm:"column:calculated_at"`
	CalculationBreakdown datatypes.JSON `json:"calculation_breakdown" gorm:"column:calculation_breakdown;type:jsonb"`

	// Payment locking fields (for payment link creation)
	LockedAmount  *float64   `json:"locked_amount" gorm:"column:locked_amount;type:decimal(15,2)"`
	LockedAt      *time.Time `json:"locked_at" gorm:"column:locked_at"`
	LockedClicks  *int64     `json:"locked_clicks" gorm:"column:locked_clicks"`
	LockedRevenue *float64   `json:"locked_revenue" gorm:"column:locked_revenue;type:decimal(15,2)"`

	// Refund workflow fields (for CO_PRODUCING contracts when net amount < 0)
	RefundAmount       float64    `json:"refund_amount" gorm:"column:refund_amount;type:decimal(15,2);default:0"`
	RefundProofURL     *string    `json:"refund_proof_url" gorm:"type:text;column:refund_proof_url"`
	RefundProofNote    *string    `json:"refund_proof_note" gorm:"type:text;column:refund_proof_note"`
	RefundSubmittedAt  *time.Time `json:"refund_submitted_at" gorm:"column:refund_submitted_at"`
	RefundSubmittedBy  *uuid.UUID `json:"refund_submitted_by" gorm:"type:uuid;column:refund_submitted_by"`
	RefundReviewedAt   *time.Time `json:"refund_reviewed_at" gorm:"column:refund_reviewed_at"`
	RefundReviewedBy   *uuid.UUID `json:"refund_reviewed_by" gorm:"type:uuid;column:refund_reviewed_by"`
	RefundRejectReason *string    `json:"refund_reject_reason" gorm:"type:text;column:refund_reject_reason"`
	RefundAttempts     int        `json:"refund_attempts" gorm:"column:refund_attempts;default:0"`

	// Relationships
	Contract        *Contract  `json:"-" gorm:"foreignKey:ContractID"`
	Milestone       *Milestone `json:"-" gorm:"foreignKey:MilestoneID"` // Related milestone (if linked)
	RefundSubmitter *User      `json:"-" gorm:"foreignKey:RefundSubmittedBy"`
	RefundReviewer  *User      `json:"-" gorm:"foreignKey:RefundReviewedBy"`
}

func (ContractPayment) TableName() string { return "contract_payments" }

func (cp *ContractPayment) BeforeCreate(tx *gorm.DB) error {
	if cp.ID == uuid.Nil {
		cp.ID = uuid.New()
	}
	if cp.Status == "" {
		cp.Status = enum.ContractPaymentStatusPending
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

// IsInRefundFlow returns true if the payment is in any refund workflow status
func (cp *ContractPayment) IsInRefundFlow() bool {
	return cp.Status.IsRefundStatus()
}

// CanSubmitRefundProof returns true if refund proof can be submitted by Marketing Staff
func (cp *ContractPayment) CanSubmitRefundProof() bool {
	return cp.Status == enum.ContractPaymentStatusKOLPending ||
		cp.Status == enum.ContractPaymentStatusKOLProofRejected
}

// CanReviewRefundProof returns true if Brand can review the proof
func (cp *ContractPayment) CanReviewRefundProof() bool {
	return cp.Status == enum.ContractPaymentStatusKOLProofSubmitted
}

// CanGeneratePaymentLink returns true if payment link can be generated
// Payment link is only allowed after the due date to prevent early payment exploitation
func (cp *ContractPayment) CanGeneratePaymentLink() bool {
	if cp.Status != enum.ContractPaymentStatusPending {
		return false
	}
	now := time.Now()
	return !now.Before(cp.DueDate)
}
