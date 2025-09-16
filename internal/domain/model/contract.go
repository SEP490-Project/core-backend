package model

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

type Contract struct {
	ID                     uuid.UUID           `json:"id" gorm:"primaryKey"`
	BrandID                uuid.UUID           `json:"brand_id" gorm:"not null"`
	InfluencerUserID       uuid.UUID           `json:"influencer_user_id" gorm:"not null"`
	Title                  string              `json:"title" gorm:"not null"`
	Type                   enum.ContractType   `json:"type" gorm:"type:enum('ADVERTISING','AFFILIATE','AMBASSADOR','COPRODUCE');not null"`
	StartDate              string              `json:"start_date" gorm:"not null"`
	EndDate                string              `json:"end_date" gorm:"not null"`
	ScopeOfWork            string              `json:"scope_of_work" gorm:"not null"`
	CompensationAmount     float64             `json:"compensation_amount" gorm:"not null"`
	PaymentTerms           string              `json:"payment_terms" gorm:"not null"`
	CommissionRate         float64             `json:"commission_rate" gorm:"not null"`
	UsageRights            string              `json:"usage_rights" gorm:"not null"`
	ExclusivityClause      string              `json:"exclusivity_clause" gorm:"not null"`
	Confidentiality        string              `json:"confidentiality" gorm:"not null"`
	TerminationConditions  string              `json:"termination_conditions" gorm:"not null"`
	GoverningLaw           string              `json:"governing_law" gorm:"not null"`
	DisputeResolution      string              `json:"dispute_resolution" gorm:"not null"`
	ComplianceRequirements string              `json:"compliance_requirements" gorm:"not null"`
	PdfURL                 string              `json:"pdf_url" gorm:"not null"`
	Status                 enum.ContractStatus `json:"status" gorm:"type:enum('ACTIVE','EXPIRED','CANCELED');not null"`
	CreatedAt              int64               `json:"created_at" gorm:"autoCreateTime"`
	Signatures             string              `json:"signatures" gorm:"type:jsonb"`
}
