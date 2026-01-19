package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ContractViolation represents a contract violation record with financial details
type ContractViolation struct {
	ID         uuid.UUID          `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	ContractID uuid.UUID          `json:"contract_id" gorm:"type:uuid;column:contract_id;not null;index"`
	CampaignID *uuid.UUID         `json:"campaign_id" gorm:"type:uuid;column:campaign_id;index"` // Null if no campaign created yet
	Type       enum.ViolationType `json:"type" gorm:"type:varchar(20);column:type;not null;check:type IN ('BRAND','KOL')"`
	Reason     string             `json:"reason" gorm:"type:text;column:reason;not null"`

	// Financial details
	PenaltyAmount       float64 `json:"penalty_amount" gorm:"column:penalty_amount;type:decimal(15,2);not null;default:0"`
	RefundAmount        float64 `json:"refund_amount" gorm:"column:refund_amount;type:decimal(15,2);not null;default:0"`
	TotalPaidByBrand    float64 `json:"total_paid_by_brand" gorm:"column:total_paid_by_brand;type:decimal(15,2);not null;default:0"`
	CompletedMilestones int     `json:"completed_milestones" gorm:"column:completed_milestones;not null;default:0"`
	TotalMilestones     int     `json:"total_milestones" gorm:"column:total_milestones;not null;default:0"`

	// Calculation breakdown stored as JSONB for auditing
	CalculationBreakdown datatypes.JSON `json:"calculation_breakdown" gorm:"column:calculation_breakdown;type:jsonb"`

	// Proof handling (for KOL refund proof)
	ProofStatus      *enum.ViolationProofStatus `json:"proof_status" gorm:"type:varchar(20);column:proof_status;check:proof_status IN ('PENDING','APPROVED','REJECTED')"`
	ProofURL         *string                    `json:"proof_url" gorm:"type:text;column:proof_url"`
	ProofSubmittedAt *time.Time                 `json:"proof_submitted_at" gorm:"column:proof_submitted_at"`
	ProofSubmittedBy *uuid.UUID                 `json:"proof_submitted_by" gorm:"type:uuid;column:proof_submitted_by"`
	ProofReviewedAt  *time.Time                 `json:"proof_reviewed_at" gorm:"column:proof_reviewed_at"`
	ProofReviewedBy  *uuid.UUID                 `json:"proof_reviewed_by" gorm:"type:uuid;column:proof_reviewed_by"`
	ProofReviewNote  *string                    `json:"proof_review_note" gorm:"type:text;column:proof_review_note"`
	ProofAttempts    int                        `json:"proof_attempts" gorm:"column:proof_attempts;not null;default:0"`

	// Resolution tracking
	ResolvedAt *time.Time `json:"resolved_at" gorm:"column:resolved_at"`
	ResolvedBy *uuid.UUID `json:"resolved_by" gorm:"type:uuid;column:resolved_by"`

	// Payment transaction for brand penalty payment
	PaymentTransactionID *uuid.UUID `json:"payment_transaction_id" gorm:"type:uuid;column:payment_transaction_id"`

	// Audit fields
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	CreatedBy *uuid.UUID     `json:"created_by" gorm:"type:uuid;column:created_by"`
	UpdatedBy *uuid.UUID     `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	Contract           *Contract           `json:"-" gorm:"foreignKey:ContractID"`
	Campaign           *Campaign           `json:"-" gorm:"foreignKey:CampaignID"`
	PaymentTransaction *PaymentTransaction `json:"-" gorm:"foreignKey:PaymentTransactionID"`
}

func (ContractViolation) TableName() string { return "contract_violations" }

func (cv *ContractViolation) BeforeCreate(tx *gorm.DB) error {
	if cv.ID == uuid.Nil {
		cv.ID = uuid.New()
	}
	return nil
}

// CalculationBreakdownData represents the detailed breakdown of violation calculations
type CalculationBreakdownData struct {
	ContractTotalValue     float64                  `json:"contract_total_value"`
	PenaltyPercentage      float64                  `json:"penalty_percentage"`
	CompletedMilestoneIDs  []string                 `json:"completed_milestone_ids"`
	IncompleteMilestoneIDs []string                 `json:"incomplete_milestone_ids"`
	PaidPaymentIDs         []string                 `json:"paid_payment_ids"`
	PendingPaymentIDs      []string                 `json:"pending_payment_ids"`
	MilestoneDetails       []MilestoneBreakdownItem `json:"milestone_details"`
	PaymentDetails         []PaymentBreakdownItem   `json:"payment_details"`
	CalculationFormula     string                   `json:"calculation_formula"`
	CalculatedAt           time.Time                `json:"calculated_at"`
}

// MilestoneBreakdownItem represents a milestone in the calculation breakdown
type MilestoneBreakdownItem struct {
	MilestoneID     string  `json:"milestone_id"`
	MilestoneName   string  `json:"milestone_name"`
	Percentage      float64 `json:"percentage"`
	Status          string  `json:"status"`
	LinkedPaymentID *string `json:"linked_payment_id,omitempty"`
}

// PaymentBreakdownItem represents a payment in the calculation breakdown
type PaymentBreakdownItem struct {
	PaymentID   string    `json:"payment_id"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	DueDate     time.Time `json:"due_date"`
	MilestoneID *string   `json:"milestone_id,omitempty"`
}

// IsResolved returns true if the violation has been resolved
func (cv *ContractViolation) IsResolved() bool {
	return cv.ResolvedAt != nil
}

// IsBrandViolation returns true if this is a brand violation
func (cv *ContractViolation) IsBrandViolation() bool {
	return cv.Type == enum.ViolationTypeBrand
}

// IsKOLViolation returns true if this is a KOL violation
func (cv *ContractViolation) IsKOLViolation() bool {
	return cv.Type == enum.ViolationTypeKOL
}

// HasProofSubmitted returns true if KOL has submitted proof
func (cv *ContractViolation) HasProofSubmitted() bool {
	return cv.ProofURL != nil && *cv.ProofURL != ""
}

// IsProofApproved returns true if proof has been approved
func (cv *ContractViolation) IsProofApproved() bool {
	return cv.ProofStatus != nil && *cv.ProofStatus == enum.ViolationProofStatusApproved
}

// IsProofRejected returns true if proof has been rejected
func (cv *ContractViolation) IsProofRejected() bool {
	return cv.ProofStatus != nil && *cv.ProofStatus == enum.ViolationProofStatusRejected
}
