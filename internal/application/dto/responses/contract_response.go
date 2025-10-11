package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"

	"github.com/google/uuid"
)

// ContractResponse represents the full contract response with all details
type ContractResponse struct {
	ID               string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ParentContractID *string `json:"parent_contract_id" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Contract basic information
	Title          string `json:"title" example:"Social Media Promotion Contract"`
	ContractNumber string `json:"contract_number" example:"CONTRACT-2023-001"`
	Type           string `json:"type" example:"ADVERTISING"`
	Status         string `json:"status" example:"ACTIVE"`

	// Brand information (from relationship)
	BrandID                  string        `json:"brand_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Brand                    *BrandSummary `json:"brand,omitempty"`
	BrandTaxNumber           *string       `json:"brand_tax_number,omitempty" example:"TAX123456"`
	BrandRepresentativeName  *string       `json:"brand_representative_name,omitempty" example:"John Doe"`
	BrandRepresentativeRole  *string       `json:"brand_representative_role,omitempty" example:"CEO"`
	BrandRepresentativePhone *string       `json:"brand_representative_phone,omitempty" example:"+84901234567"`
	BrandRepresentativeEmail *string       `json:"brand_representative_email,omitempty" example:"john.doe@acme.com"`
	BrandBankName            *string       `json:"brand_bank_name,omitempty" example:"Vietcombank"`
	BrandBankAccountNumber   *string       `json:"brand_bank_account_number,omitempty" example:"0123456789"`

	// KOL/Representative information
	RepresentativeName              string  `json:"representative_name" example:"Jane Smith"`
	RepresentativeRole              *string `json:"representative_role,omitempty" example:"Influencer"`
	RepresentativePhone             *string `json:"representative_phone,omitempty" example:"+84901234567"`
	RepresentativeEmail             *string `json:"representative_email,omitempty" example:"jane.smith@example.com"`
	RepresentativeTaxNumber         *string `json:"representative_tax_number,omitempty" example:"TAX654321"`
	RepresentativeBankName          *string `json:"representative_bank_name,omitempty" example:"First National Bank"`
	RepresentativeBankAccountNumber *string `json:"representative_bank_account_number,omitempty" example:"987654321"`
	RepresentativeBankAccountHolder *string `json:"representative_bank_account_holder,omitempty" example:"Jane Smith"`

	// Contract dates
	SignedDate     string  `json:"signed_date" example:"2006-01-02 15:04:05"`
	SignedLocation *string `json:"signed_location,omitempty" example:"Springfield"`
	StartDate      string  `json:"start_date" example:"2006-01-02 15:04:05"`
	EndDate        string  `json:"end_date" example:"2006-01-02 15:04:05"`

	// Financial
	Currency *string `json:"currency,omitempty" example:"VND"`

	// Complex JSONB fields (unmarshaled)
	FinancialTerms any `json:"financial_terms"`
	ScopeOfWork    any `json:"scope_of_work"`
	LegalTerms     any `json:"legal_terms"`

	// File URLs
	ContractFileURL *string `json:"contract_file_url,omitempty" example:"https://example.com/contracts/contract.pdf"`
	ProposalFileURL *string `json:"proposal_file_url,omitempty" example:"https://example.com/proposals/proposal.pdf"`

	// Parent contract summary (if exists)
	ParentContract *ContractSummary `json:"parent_contract,omitempty"`

	// Metadata
	CreatedAt string `json:"created_at" example:"2006-01-02 15:04:05"`
	UpdatedAt string `json:"updated_at" example:"2006-01-02 15:04:05"`
}

// ContractSummary represents a brief contract summary for nested relationships
type ContractSummary struct {
	ID             string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title          string `json:"title" example:"Master Service Agreement"`
	ContractNumber string `json:"contract_number" example:"CONTRACT-2023-001"`
	Type           string `json:"type" example:"ADVERTISING"`
	Status         string `json:"status" example:"ACTIVE"`
	StartDate      string `json:"start_date" example:"2006-01-02 15:04:05"`
	EndDate        string `json:"end_date" example:"2006-01-02 15:04:05"`
}

// BrandSummary represents a brief brand summary for contract response
type BrandSummary struct {
	ID                      string  `json:"id" example:"660e8400-e29b-41d4-a716-446655440000"`
	Name                    string  `json:"name" example:"Acme Corp"`
	ContactEmail            string  `json:"contact_email" example:"contact@acme.com"`
	ContactPhone            string  `json:"contact_phone" example:"+84901234567"`
	Address                 *string `json:"address,omitempty" example:"123 Main St, Springfield"`
	LogoURL                 *string `json:"logo_url,omitempty" example:"https://example.com/logo.png"`
	TaxNumber               *string `json:"tax_number,omitempty" example:"TAX123456"`
	RepresentativeName      *string `json:"representative_name,omitempty" example:"John Doe"`
	RepresentativeRole      *string `json:"representative_role,omitempty" example:"CEO"`
	RepresentativePhone     *string `json:"representative_phone,omitempty" example:"+84901234567"`
	RepresentativeEmail     *string `json:"representative_email,omitempty" example:"john.doe@acme.com"`
	RepresentativeCitizenID *string `json:"representative_citizen_id,omitempty" example:"1234567890"`
}

// ToContractResponse converts a model.Contract to ContractResponse
func ToContractResponse(contract *model.Contract) (*ContractResponse, error) {
	if contract == nil {
		return nil, nil
	}

	response := &ContractResponse{
		ID:                              contract.ID.String(),
		Title:                           safeString(contract.Title),
		ContractNumber:                  safeString(contract.ContractNumber),
		Type:                            string(contract.Type),
		Status:                          string(contract.Status),
		BrandBankName:                   contract.BrandBankName,
		BrandBankAccountNumber:          contract.BrandBankAccountNumber,
		RepresentativeName:              safeString(contract.RepresentativeName),
		RepresentativeRole:              contract.RepresentativeRole,
		RepresentativePhone:             contract.RepresentativePhone,
		RepresentativeEmail:             contract.RepresentativeEmail,
		RepresentativeTaxNumber:         contract.RepresentativeTaxNumber,
		RepresentativeBankName:          contract.RepresentativeBankName,
		RepresentativeBankAccountNumber: contract.RepresentativeBankAccountNumber,
		RepresentativeBankAccountHolder: contract.RepresentativeBankAccountHolder,
		SignedDate:                      utils.FormatLocalTime(&contract.SignedDate, ""),
		SignedLocation:                  contract.SignedLocation,
		StartDate:                       utils.FormatLocalTime(&contract.StartDate, ""),
		EndDate:                         utils.FormatLocalTime(&contract.EndDate, ""),
		Currency:                        contract.Currency,
		ContractFileURL:                 contract.ContractFileURL,
		ProposalFileURL:                 contract.ProposalFileURL,
		CreatedAt:                       utils.FormatLocalTime(&contract.CreatedAt, ""),
		UpdatedAt:                       utils.FormatLocalTime(&contract.UpdatedAt, ""),
	}

	// Set ParentContractID
	if contract.ParentContractID != nil {
		parentID := contract.ParentContractID.String()
		response.ParentContractID = &parentID
	}

	// Set BrandID
	if contract.BrandID != nil {
		response.BrandID = contract.BrandID.String()
	}

	// Unmarshal JSONB fields
	if len(contract.FinancialTerms) > 0 {
		var financialTerms any
		if err := json.Unmarshal(contract.FinancialTerms, &financialTerms); err == nil {
			response.FinancialTerms = financialTerms
		}
	}

	if len(contract.ScopeOfWork) > 0 {
		var scopeOfWork any
		if err := json.Unmarshal(contract.ScopeOfWork, &scopeOfWork); err == nil {
			response.ScopeOfWork = scopeOfWork
		}
	}

	if len(contract.LegalTerms) > 0 {
		var legalTerms any
		if err := json.Unmarshal(contract.LegalTerms, &legalTerms); err == nil {
			response.LegalTerms = legalTerms
		}
	}

	// Add Brand information if loaded
	if contract.Brand != nil {
		tempBrand := contract.Brand
		response.Brand = &BrandSummary{
			ID:                      tempBrand.ID.String(),
			Name:                    tempBrand.Name,
			ContactEmail:            tempBrand.ContactEmail,
			ContactPhone:            tempBrand.ContactPhone,
			Address:                 tempBrand.Address,
			LogoURL:                 tempBrand.LogoURL,
			TaxNumber:               tempBrand.TaxNumber,
			RepresentativeName:      tempBrand.RepresentativeName,
			RepresentativeRole:      tempBrand.RepresentativeRole,
			RepresentativePhone:     tempBrand.RepresentativePhone,
			RepresentativeEmail:     tempBrand.RepresentativeEmail,
			RepresentativeCitizenID: tempBrand.RepresentativeCitizenID,
		}
	}

	// Add ParentContract summary if loaded
	if contract.ParentContract != nil {
		response.ParentContract = &ContractSummary{
			ID:             contract.ParentContract.ID.String(),
			Title:          safeString(contract.ParentContract.Title),
			ContractNumber: safeString(contract.ParentContract.ContractNumber),
			Type:           string(contract.ParentContract.Type),
			Status:         string(contract.ParentContract.Status),
			StartDate:      utils.FormatLocalTime(&contract.ParentContract.StartDate, ""),
			EndDate:        utils.FormatLocalTime(&contract.ParentContract.EndDate, ""),
		}
	}

	return response, nil
}

// ContractListResponse represents a simplified contract for list views
type ContractListResponse struct {
	ID             string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Title          string  `json:"title" example:"Social Media Promotion Contract"`
	ContractNumber string  `json:"contract_number" example:"CONTRACT-2023-001"`
	Type           string  `json:"type" example:"ADVERTISING"`
	Status         string  `json:"status" example:"ACTIVE"`
	BrandID        string  `json:"brand_id" example:"660e8400-e29b-41d4-a716-446655440000"`
	BrandName      *string `json:"brand_name,omitempty" example:"Acme Corp"`
	StartDate      string  `json:"start_date" example:"2006-01-02 15:04:05"`
	EndDate        string  `json:"end_date" example:"2006-01-02 15:04:05"`
	SignedDate     string  `json:"signed_date" example:"2006-01-02 15:04:05"`
	CreatedAt      string  `json:"created_at" example:"2006-01-02 15:04:05"`
	UpdatedAt      string  `json:"updated_at" example:"2006-01-02 15:04:05"`
}

// ToContractListResponse converts a model.Contract to ContractListResponse
func ToContractListResponse(contract *model.Contract) *ContractListResponse {
	if contract == nil {
		return nil
	}

	response := &ContractListResponse{
		ID:             contract.ID.String(),
		Title:          safeString(contract.Title),
		ContractNumber: safeString(contract.ContractNumber),
		Type:           string(contract.Type),
		Status:         string(contract.Status),
		StartDate:      utils.FormatLocalTime(&contract.StartDate, ""),
		EndDate:        utils.FormatLocalTime(&contract.EndDate, ""),
		SignedDate:     utils.FormatLocalTime(&contract.SignedDate, ""),
		CreatedAt:      utils.FormatLocalTime(&contract.CreatedAt, ""),
		UpdatedAt:      utils.FormatLocalTime(&contract.UpdatedAt, ""),
	}

	// Set BrandID
	if contract.BrandID != nil {
		response.BrandID = contract.BrandID.String()
	}

	// Set BrandName if Brand is loaded
	if contract.Brand != nil {
		response.BrandName = &contract.Brand.Name
	}

	return response
}

// Helper function to safely dereference string pointers
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ============================================================================
// Below are structures for complex JSONB fields for documentation and type safety
// ============================================================================

type AdvertisingFinancialTerms struct {
	Model         string            `json:"model" example:"FIXED"`
	PaymentMethod string            `json:"payment_method" example:"BANK_TRANSFER"`
	TotalCost     int               `json:"total_cost" example:"10000000"`
	CostBreakdown map[string]int    `json:"cost_breakdown"`
	Schedule      []Schedule        `json:"schedule"`
	PaymentCycle  enum.PaymentCycle `json:"payment_cycle" example:"MONTHLY"`
	PaymentDate   string            `json:"payment_date" example:"2023-11-01"`
}

type Schedule struct {
	Milestone string `json:"milestone" example:"Initial payment"`
	Percent   int    `json:"percent" example:"30"`
	Amount    int    `json:"amount" example:"3000000"`
	DueDate   string `json:"due_date" example:"2023-10-15"`
}

type AffiliateFinancialTerms struct {
	Model          string            `json:"model" example:"COMMISSION"`
	BasePerClick   int               `json:"base_per_click" example:"1000"`
	Levels         []Level           `json:"levels"`
	PaymentCycle   enum.PaymentCycle `json:"payment_cycle" example:"MONTHLY"`
	PaymentDate    string            `json:"payment_date" example:"2023-11-05"`
	TaxWithholding TaxWithholding    `json:"tax_withholding"`
}

type Level struct {
	Level      int     `json:"level" example:"1"`
	MinClicks  int     `json:"min_clicks" example:"1000"`
	Multiplier float64 `json:"multiplier" example:"1.5"`
}

type TaxWithholding struct {
	Threshold   int `json:"threshold" example:"10000000"`
	RatePercent int `json:"rate_percent" example:"10"`
}

type CoProducingFinancialTerms struct {
	Model                   string              `json:"model" example:"PROFIT_SHARING"`
	CapitalContribution     CapitalContribution `json:"capital_contribution"`
	CompanyPercent          int                 `json:"profit_split_company_percent" example:"60"`
	KolPercent              int                 `json:"profit_split_kol_percent" example:"40"`
	ProfitDistributionCycle enum.PaymentCycle   `json:"profit_distribution_cycle" example:"QUARTERLY"`
	ProfitDistributionDate  string              `json:"profit_distribution_date" example:"2023-12-31"`
}

type CapitalContribution struct {
	Company ContributionDescription `json:"company"`
	Kol     ContributionDescription `json:"kol"`
}

type ContributionDescription struct {
	Description string `json:"description" example:"Equipment and studio"`
	Value       int    `json:"value" example:"50000000"`
}

type ScopeOfWork struct {
	Description           string            `json:"description" example:"Create and publish social media content"`
	Products              []string          `json:"products" example:"Product A,Product B"`
	TechnicalRequirements string            `json:"technical_requirements" example:"4K video, professional lighting"`
	Deliverables          []Deliverable     `json:"deliverables"`
	BrandingRestrictions  []string          `json:"branding_restrictions" example:"No competitor brands"`
	CoProductionRoles     map[string]string `json:"co_production_roles,omitempty"`
}

type Deliverable struct {
	Type        string    `json:"type" example:"VIDEO"`
	ChannelID   uuid.UUID `json:"channel_id" example:"770e8400-e29b-41d4-a716-446655440000"`
	ChannelName string    `json:"channel_name" example:"YouTube"`
	ChannelLink string    `json:"channel_link" example:"https://youtube.com/channel/xyz"`
	Quantity    int       `json:"quantity" example:"5"`
	Deadline    string    `json:"deadline" example:"2023-11-30"`
}

// Legal Terms Structures

type LegalTerms struct {
	Penalties                    Penalties         `json:"penalties"`
	ForceMajeureNotificationDays int               `json:"force_majeure_notification_days" example:"7"`
	DisputeResolutionCourt       string            `json:"dispute_resolution_court" example:"Ho Chi Minh City Court"`
	Confidentiality              bool              `json:"confidentiality" example:"true"`
	ManagementBoard              []ManagementBoard `json:"management_board"`
	NumberOfCopies               int               `json:"number_of_copies" example:"2"`
}

type Penalties struct {
	LateDeliveryPercentPerDay float64 `json:"late_delivery_percent_per_day" example:"0.5"`
	LatePaymentPercentPerDay  float64 `json:"late_payment_percent_per_day" example:"0.3"`
	BreachOfContractPercent   int     `json:"breach_of_contract_percent" example:"20"`
	NonDeliveryPenaltyPercent int     `json:"non_delivery_penalty_percent" example:"30"`
}

type ManagementBoard struct {
	Name         string `json:"name" example:"John Doe"`
	Representing string `json:"representing" example:"Brand"`
}

// ContractPaginationResponse represents a paginated response for contracts
// Only used for Swaggo swagger docs generation
type ContractPaginationResponse PaginationResponse[ContractListResponse]
