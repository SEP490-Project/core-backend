package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// ViolationResponse is the detailed response for a contract violation
type ViolationResponse struct {
	ID         uuid.UUID          `json:"id"`
	ContractID uuid.UUID          `json:"contract_id"`
	CampaignID *uuid.UUID         `json:"campaign_id,omitempty"`
	Type       enum.ViolationType `json:"type"`
	Reason     string             `json:"reason"`

	// Financial details
	PenaltyAmount       float64 `json:"penalty_amount"`
	RefundAmount        float64 `json:"refund_amount"`
	TotalPaidByBrand    float64 `json:"total_paid_by_brand"`
	CompletedMilestones int     `json:"completed_milestones"`
	TotalMilestones     int     `json:"total_milestones"`

	// Proof handling
	ProofStatus      *enum.ViolationProofStatus `json:"proof_status,omitempty"`
	ProofURL         *string                    `json:"proof_url,omitempty"`
	ProofSubmittedAt *time.Time                 `json:"proof_submitted_at,omitempty"`
	ProofSubmittedBy *uuid.UUID                 `json:"proof_submitted_by"`
	ProofReviewedAt  *time.Time                 `json:"proof_reviewed_at,omitempty"`
	ProofReviewNote  *string                    `json:"proof_review_note,omitempty"`
	ProofAttempts    int                        `json:"proof_attempts,omitempty"`
	ProofReviewedBy  *uuid.UUID                 `json:"proof_reviewed_by"`

	// Resolution
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	IsResolved bool       `json:"is_resolved"`

	// Related data
	Contract    *ContractSummaryResponse `json:"contract,omitempty"`
	Campaign    *CampaignSummaryResponse `json:"campaign,omitempty"`
	PaymentData *ViolationPaymentData    `json:"payment_data,omitempty"`

	// Audit
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ViolationListResponse is the summary response for listing violations
type ViolationListResponse struct {
	ID             uuid.UUID                  `json:"id"`
	ContractID     uuid.UUID                  `json:"contract_id"`
	ContractNumber string                     `json:"contract_number"`
	CampaignID     *uuid.UUID                 `json:"campaign_id,omitempty"`
	CampaignName   *string                    `json:"campaign_name,omitempty"`
	BrandID        uuid.UUID                  `json:"brand_id"`
	BrandName      string                     `json:"brand_name"`
	Type           enum.ViolationType         `json:"type"`
	Reason         string                     `json:"reason"`
	PenaltyAmount  float64                    `json:"penalty_amount"`
	RefundAmount   float64                    `json:"refund_amount"`
	ProofStatus    *enum.ViolationProofStatus `json:"proof_status,omitempty"`
	IsResolved     bool                       `json:"is_resolved"`
	CreatedAt      time.Time                  `json:"created_at"`
}

// ViolationCalculationResponse shows the calculated amounts for a violation
type ViolationCalculationResponse struct {
	ContractID          uuid.UUID `json:"contract_id"`
	ContractTotalValue  float64   `json:"contract_total_value"`
	TotalPaidByBrand    float64   `json:"total_paid_by_brand"`
	CompletedMilestones int       `json:"completed_milestones"`
	TotalMilestones     int       `json:"total_milestones"`

	// Brand violation specific
	PenaltyPercentage float64 `json:"penalty_percentage,omitempty"`
	PenaltyAmount     float64 `json:"penalty_amount"`

	// KOL violation specific
	RefundAmount float64 `json:"refund_amount,omitempty"`

	// Calculation breakdown
	CalculationFormula string                  `json:"calculation_formula"`
	MilestoneDetails   []MilestoneBreakdownDTO `json:"milestone_details,omitempty"`
	PaymentDetails     []PaymentBreakdownDTO   `json:"payment_details,omitempty"`
}

// MilestoneBreakdownDTO shows milestone details in calculation
type MilestoneBreakdownDTO struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Percentage      float64    `json:"percentage"`
	Status          string     `json:"status"`
	LinkedPaymentID *uuid.UUID `json:"linked_payment_id,omitempty"`
}

// PaymentBreakdownDTO shows payment details in calculation
type PaymentBreakdownDTO struct {
	ID          uuid.UUID  `json:"id"`
	Amount      float64    `json:"amount"`
	Status      string     `json:"status"`
	DueDate     time.Time  `json:"due_date"`
	MilestoneID *uuid.UUID `json:"milestone_id,omitempty"`
	IsDeposit   bool       `json:"is_deposit"`
}

type ViolationPaymentData struct {
	PaymentLinkID string  `json:"paymentLinkId"`
	OrderCode     int64   `json:"orderCode"`
	CheckoutURL   string  `json:"checkoutUrl"`
	QRCode        string  `json:"qrCode"`
	Bin           string  `json:"bin"`
	AccountNumber string  `json:"accountNumber"`
	AccountName   string  `json:"accountName"`
	ExpiredAt     int64   `json:"expiredAt"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
}

// ContractSummaryResponse is a brief contract summary for nested responses
type ContractSummaryResponse struct {
	ID             uuid.UUID `json:"id"`
	ContractNumber string    `json:"contract_number"`
	BrandID        uuid.UUID `json:"brand_id"`
	BrandName      string    `json:"brand_name"`
	TotalValue     float64   `json:"total_value"`
	Status         string    `json:"status"`
}

// CampaignSummaryResponse is a brief campaign summary for nested responses
type CampaignSummaryResponse struct {
	ID     uuid.UUID `json:"id"`
	Name   string    `json:"name"`
	Status string    `json:"status"`
}

// ToViolationResponse converts a model to response
func (ViolationResponse) ToViolationResponse(v *model.ContractViolation) *ViolationResponse {
	if v == nil {
		return nil
	}

	res := &ViolationResponse{
		ID:                  v.ID,
		ContractID:          v.ContractID,
		CampaignID:          v.CampaignID,
		Type:                v.Type,
		Reason:              v.Reason,
		PenaltyAmount:       v.PenaltyAmount,
		RefundAmount:        v.RefundAmount,
		TotalPaidByBrand:    v.TotalPaidByBrand,
		CompletedMilestones: v.CompletedMilestones,
		TotalMilestones:     v.TotalMilestones,
		ProofStatus:         v.ProofStatus,
		ProofURL:            v.ProofURL,
		ProofSubmittedAt:    v.ProofSubmittedAt,
		ProofSubmittedBy:    v.ProofSubmittedBy,
		ProofReviewedAt:     v.ProofReviewedAt,
		ProofReviewNote:     v.ProofReviewNote,
		ResolvedAt:          v.ResolvedAt,
		IsResolved:          v.ResolvedAt != nil,
		CreatedAt:           v.CreatedAt,
		UpdatedAt:           v.UpdatedAt,
	}

	if v.PaymentTransaction != nil && v.PaymentTransaction.PayOSMetadata != nil {
		meta := v.PaymentTransaction.PayOSMetadata
		res.PaymentData = &ViolationPaymentData{
			PaymentLinkID: meta.PaymentLinkID,
			OrderCode:     meta.OrderCode,
			CheckoutURL:   meta.CheckoutURL,
			QRCode:        meta.QRCode,
			Bin:           meta.Bin,
			AccountNumber: meta.AccountNumber,
			AccountName:   meta.AccountName,
			ExpiredAt:     meta.ExpiredAt,
			Amount:        meta.Amount,
			Description:   meta.Description,
		}
	}

	return res
}
