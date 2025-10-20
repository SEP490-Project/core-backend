package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
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
	DepositPercent *int   `json:"deposit_percent,omitempty" example:"30"`
	DepositAmount  *int   `json:"deposit_amount,omitempty" example:"3000000"`

	// Brand information (from relationship)
	Brand *BrandSummary `json:"brand,omitempty"`

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
	BankName                *string `json:"bank_name,omitempty" example:"Vietcombank"`
	BankAccountNumber       *string `json:"bank_account_number,omitempty" example:"0123456789"`
}

// ToContractResponse converts a model.Contract to ContractResponse
func (ContractResponse) ToContractResponse(contract *model.Contract) (*ContractResponse, error) {
	if contract == nil {
		return nil, nil
	}

	response := &ContractResponse{
		ID:                              contract.ID.String(),
		Title:                           safeString(contract.Title),
		ContractNumber:                  safeString(contract.ContractNumber),
		Type:                            string(contract.Type),
		Status:                          string(contract.Status),
		DepositPercent:                  contract.DepositPercent,
		DepositAmount:                   contract.DepositAmount,
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
	if contract.BrandID != nil && contract.Brand != nil {
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

// ContractPaginationResponse represents a paginated response for contracts
// Only used for Swaggo swagger docs generation
type ContractPaginationResponse PaginationResponse[ContractListResponse]
