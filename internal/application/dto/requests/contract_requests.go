package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// CreateContractRequest represents the payload for creating a new contract.
type CreateContractRequest struct {
	// Parent contract (for amendments or related contracts)
	ParentContractID *string `json:"parent_contract_id" validate:"omitempty,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Contract basic information
	Title          string  `json:"title" validate:"required,min=2,max=255" example:"Social Media Promotion Contract"`
	ContractNumber string  `json:"contract_number" validate:"required,min=2,max=255" example:"CONTRACT-2023-001"`
	BrandID        string  `json:"brand_id" validate:"required,uuid4" example:"660e8400-e29b-41d4-a716-446655440000"`
	Type           string  `json:"type" validate:"required,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Status         *string `json:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETED TERMINATED" example:"DRAFT"`

	// Brand information (stored in contract for record-keeping)
	BrandTaxNumber           *string `json:"brand_tax_number" validate:"omitempty,max=100" example:"TAX123456"`
	BrandRepresentativeName  *string `json:"brand_representative_name" validate:"omitempty,max=255" example:"John Doe"`
	BrandRepresentativeRole  *string `json:"brand_representative_role" validate:"omitempty,max=255" example:"CEO"`
	BrandRepresentativePhone *string `json:"brand_representative_phone" validate:"omitempty,e164" example:"+84901234567"`
	BrandRepresentativeEmail *string `json:"brand_representative_email" validate:"omitempty,email,max=255" example:"john.doe@acme.com"`
	BrandBankName            *string `json:"brand_bank_name" validate:"omitempty,max=255" example:"Vietcombank"`
	BrandBankAccountNumber   *string `json:"brand_bank_account_number" validate:"omitempty,max=255" example:"0123456789"`

	// KOL/Representative information (the other party in the contract)
	RepresentativeName              string  `json:"representative_name" validate:"required,min=2,max=255" example:"Jane Smith"`
	RepresentativeRole              *string `json:"representative_role" validate:"omitempty,max=255" example:"Influencer"`
	RepresentativePhone             *string `json:"representative_phone" validate:"omitempty,e164" example:"+84901234567"`
	RepresentativeEmail             *string `json:"representative_email" validate:"omitempty,email,max=255" example:"jane.smith@example.com"`
	RepresentativeTaxNumber         *string `json:"representative_tax_number" validate:"omitempty,max=100" example:"TAX654321"`
	RepresentativeBankName          *string `json:"representative_bank_name" validate:"omitempty,max=255" example:"First National Bank"`
	RepresentativeBankAccountNumber *string `json:"representative_bank_account_number" validate:"omitempty,max=255" example:"987654321"`
	RepresentativeBankAccountHolder *string `json:"representative_bank_account_holder" validate:"omitempty,max=255" example:"Jane Smith"`

	// Contract dates
	SignedDate     time.Time `json:"signed_date" validate:"required" example:"2023-10-01T12:00:00Z"`
	SignedLocation *string   `json:"signed_location" validate:"omitempty,max=255" example:"Springfield"`
	StartDate      time.Time `json:"start_date" validate:"required,gtefield=SignedDate" example:"2023-10-01T00:00:00Z"`
	EndDate        time.Time `json:"end_date" validate:"required,gtefield=StartDate" example:"2023-12-31T23:59:59Z"`

	// Financial
	Currency *string `json:"currency" validate:"omitempty,len=3" example:"VND"`

	// Complex JSONB fields
	FinancialTerms any `json:"financial_terms" validate:"required"`
	ScopeOfWork    any `json:"scope_of_work" validate:"required"`
	LegalTerms     any `json:"legal_terms" validate:"required"`

	// File URLs
	ContractFileURL *string `json:"contract_file_url" validate:"omitempty,url" example:"https://example.com/contracts/contract.pdf"`
	ProposalFileURL *string `json:"proposal_file_url" validate:"omitempty,url" example:"https://example.com/proposals/proposal.pdf"`
}

// ToContract converts CreateContractRequest to model.Contract
func (r *CreateContractRequest) ToContract() (*model.Contract, error) {
	// Validate contract type
	contractType := enum.ContractType(r.Type)
	if !contractType.IsValid() {
		return nil, errors.New("invalid contract type: must be one of ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING")
	}

	// Parse BrandID
	brandID, err := uuid.Parse(r.BrandID)
	if err != nil {
		return nil, errors.New("invalid brand_id: must be a valid UUID")
	}

	// Parse ParentContractID if provided
	var parentContractID *uuid.UUID
	if r.ParentContractID != nil && *r.ParentContractID != "" {
		var parsed uuid.UUID
		parsed, err = uuid.Parse(*r.ParentContractID)
		if err != nil {
			return nil, errors.New("invalid parent_contract_id: must be a valid UUID")
		}
		parentContractID = &parsed
	}

	// Determine status (default to DRAFT if not provided)
	status := enum.ContractStatusDraft
	if r.Status != nil {
		contractStatus := enum.ContractStatus(*r.Status)
		if !contractStatus.IsValid() {
			return nil, errors.New("invalid status: must be one of DRAFT, ACTIVE, COMPLETED, TERMINATED")
		}
		status = contractStatus
	}

	// Marshal complex fields to JSON
	financialTermsJSON, err := json.Marshal(r.FinancialTerms)
	if err != nil {
		return nil, errors.New("invalid financial_terms: failed to marshal to JSON")
	}

	scopeOfWorkJSON, err := json.Marshal(r.ScopeOfWork)
	if err != nil {
		return nil, errors.New("invalid scope_of_work: failed to marshal to JSON")
	}

	legalTermsJSON, err := json.Marshal(r.LegalTerms)
	if err != nil {
		return nil, errors.New("invalid legal_terms: failed to marshal to JSON")
	}

	return &model.Contract{
		ParentContractID:                parentContractID,
		Title:                           &r.Title,
		ContractNumber:                  &r.ContractNumber,
		BrandID:                         &brandID,
		BrandTaxNumber:                  r.BrandTaxNumber,
		BrandRepresentativeName:         r.BrandRepresentativeName,
		BrandRepresentativeRole:         r.BrandRepresentativeRole,
		BrandRepresentativePhone:        r.BrandRepresentativePhone,
		BrandRepresentativeEmail:        r.BrandRepresentativeEmail,
		BrandBankName:                   r.BrandBankName,
		BrandBankAccountNumber:          r.BrandBankAccountNumber,
		RepresentativeName:              &r.RepresentativeName,
		RepresentativeRole:              r.RepresentativeRole,
		RepresentativePhone:             r.RepresentativePhone,
		RepresentativeEmail:             r.RepresentativeEmail,
		RepresentativeTaxNumber:         r.RepresentativeTaxNumber,
		RepresentativeBankName:          r.RepresentativeBankName,
		RepresentativeBankAccountNumber: r.RepresentativeBankAccountNumber,
		RepresentativeBankAccountHolder: r.RepresentativeBankAccountHolder,
		Type:                            contractType,
		Status:                          status,
		SignedDate:                      r.SignedDate,
		SignedLocation:                  r.SignedLocation,
		StartDate:                       r.StartDate,
		EndDate:                         r.EndDate,
		Currency:                        r.Currency,
		FinancialTerms:                  financialTermsJSON,
		ScopeOfWork:                     scopeOfWorkJSON,
		LegalTerms:                      legalTermsJSON,
		ContractFileURL:                 r.ContractFileURL,
		ProposalFileURL:                 r.ProposalFileURL,
	}, nil
}

// UpdateContractRequest represents the payload for updating an existing contract.
type UpdateContractRequest struct {
	// Parent contract (for amendments or related contracts)
	ParentContractID *string `json:"parent_contract_id" validate:"omitempty,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Contract basic information
	Title          *string `json:"title" validate:"omitempty,min=2,max=255" example:"Updated Contract Title"`
	ContractNumber *string `json:"contract_number" validate:"omitempty,min=2,max=255" example:"CONTRACT-2023-001-UPDATED"`
	BrandID        *string `json:"brand_id" validate:"omitempty,uuid4" example:"660e8400-e29b-41d4-a716-446655440000"`
	Type           *string `json:"type" validate:"omitempty,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Status         *string `json:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETED TERMINATED" example:"ACTIVE"`

	// Brand information (stored in contract for record-keeping)
	BrandTaxNumber           *string `json:"brand_tax_number" validate:"omitempty,max=100" example:"TAX123456"`
	BrandRepresentativeName  *string `json:"brand_representative_name" validate:"omitempty,max=255" example:"John Doe"`
	BrandRepresentativeRole  *string `json:"brand_representative_role" validate:"omitempty,max=255" example:"CEO"`
	BrandRepresentativePhone *string `json:"brand_representative_phone" validate:"omitempty,e164" example:"+84901234567"`
	BrandRepresentativeEmail *string `json:"brand_representative_email" validate:"omitempty,email,max=255" example:"john.doe@acme.com"`
	BrandBankName            *string `json:"brand_bank_name" validate:"omitempty,max=255" example:"Vietcombank"`
	BrandBankAccountNumber   *string `json:"brand_bank_account_number" validate:"omitempty,max=255" example:"0123456789"`

	// KOL/Representative information
	RepresentativeName              *string `json:"representative_name" validate:"omitempty,min=2,max=255" example:"Jane Smith"`
	RepresentativeRole              *string `json:"representative_role" validate:"omitempty,max=255" example:"Influencer"`
	RepresentativePhone             *string `json:"representative_phone" validate:"omitempty,e164" example:"+84901234567"`
	RepresentativeEmail             *string `json:"representative_email" validate:"omitempty,email,max=255" example:"jane.smith@example.com"`
	RepresentativeTaxNumber         *string `json:"representative_tax_number" validate:"omitempty,max=100" example:"TAX654321"`
	RepresentativeBankName          *string `json:"representative_bank_name" validate:"omitempty,max=255" example:"First National Bank"`
	RepresentativeBankAccountNumber *string `json:"representative_bank_account_number" validate:"omitempty,max=255" example:"987654321"`
	RepresentativeBankAccountHolder *string `json:"representative_bank_account_holder" validate:"omitempty,max=255" example:"Jane Smith"`

	// Contract dates
	SignedDate     *time.Time `json:"signed_date" validate:"omitempty" example:"2023-10-01T12:00:00Z"`
	SignedLocation *string    `json:"signed_location" validate:"omitempty,max=255" example:"Springfield"`
	StartDate      *time.Time `json:"start_date" validate:"omitempty" example:"2023-10-01T00:00:00Z"`
	EndDate        *time.Time `json:"end_date" validate:"omitempty" example:"2023-12-31T23:59:59Z"`

	// Financial
	Currency *string `json:"currency" validate:"omitempty,len=3" example:"USD"`

	// Complex JSONB fields (optional for updates)
	FinancialTerms any `json:"financial_terms" validate:"omitempty"`
	ScopeOfWork    any `json:"scope_of_work" validate:"omitempty"`
	LegalTerms     any `json:"legal_terms" validate:"omitempty"`

	// File URLs
	ContractFileURL *string `json:"contract_file_url" validate:"omitempty,url" example:"https://example.com/contracts/contract.pdf"`
	ProposalFileURL *string `json:"proposal_file_url" validate:"omitempty,url" example:"https://example.com/proposals/proposal.pdf"`
}

// ApplyToContract applies the update request to an existing contract model
func (r *UpdateContractRequest) ApplyToContract(contract *model.Contract) error {
	if contract == nil {
		return errors.New("contract cannot be nil")
	}

	// Update ParentContractID
	if r.ParentContractID != nil {
		if *r.ParentContractID == "" {
			contract.ParentContractID = nil
		} else {
			parsed, err := uuid.Parse(*r.ParentContractID)
			if err != nil {
				return errors.New("invalid parent_contract_id: must be a valid UUID")
			}
			contract.ParentContractID = &parsed
		}
	}

	// Update basic fields
	if r.Title != nil {
		contract.Title = r.Title
	}
	if r.ContractNumber != nil {
		contract.ContractNumber = r.ContractNumber
	}

	// Update BrandID
	if r.BrandID != nil {
		brandID, err := uuid.Parse(*r.BrandID)
		if err != nil {
			return errors.New("invalid brand_id: must be a valid UUID")
		}
		contract.BrandID = &brandID
	}

	// Update Type
	if r.Type != nil {
		contractType := enum.ContractType(*r.Type)
		if !contractType.IsValid() {
			return errors.New("invalid type: must be one of ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING")
		}
		contract.Type = contractType
	}

	// Update Status
	if r.Status != nil {
		contractStatus := enum.ContractStatus(*r.Status)
		if !contractStatus.IsValid() {
			return errors.New("invalid status: must be one of DRAFT, ACTIVE, COMPLETED, TERMINATED")
		}
		contract.Status = contractStatus
	}

	// Update brand information (stored in contract)
	if r.BrandTaxNumber != nil {
		contract.BrandTaxNumber = r.BrandTaxNumber
	}
	if r.BrandRepresentativeName != nil {
		contract.BrandRepresentativeName = r.BrandRepresentativeName
	}
	if r.BrandRepresentativeRole != nil {
		contract.BrandRepresentativeRole = r.BrandRepresentativeRole
	}
	if r.BrandRepresentativePhone != nil {
		contract.BrandRepresentativePhone = r.BrandRepresentativePhone
	}
	if r.BrandRepresentativeEmail != nil {
		contract.BrandRepresentativeEmail = r.BrandRepresentativeEmail
	}
	if r.BrandBankName != nil {
		contract.BrandBankName = r.BrandBankName
	}
	if r.BrandBankAccountNumber != nil {
		contract.BrandBankAccountNumber = r.BrandBankAccountNumber
	}

	// Update representative information
	if r.RepresentativeName != nil {
		contract.RepresentativeName = r.RepresentativeName
	}
	if r.RepresentativeRole != nil {
		contract.RepresentativeRole = r.RepresentativeRole
	}
	if r.RepresentativePhone != nil {
		contract.RepresentativePhone = r.RepresentativePhone
	}
	if r.RepresentativeEmail != nil {
		contract.RepresentativeEmail = r.RepresentativeEmail
	}
	if r.RepresentativeTaxNumber != nil {
		contract.RepresentativeTaxNumber = r.RepresentativeTaxNumber
	}
	if r.RepresentativeBankName != nil {
		contract.RepresentativeBankName = r.RepresentativeBankName
	}
	if r.RepresentativeBankAccountNumber != nil {
		contract.RepresentativeBankAccountNumber = r.RepresentativeBankAccountNumber
	}
	if r.RepresentativeBankAccountHolder != nil {
		contract.RepresentativeBankAccountHolder = r.RepresentativeBankAccountHolder
	}

	// Update dates
	if r.SignedDate != nil {
		contract.SignedDate = *r.SignedDate
	}
	if r.SignedLocation != nil {
		contract.SignedLocation = r.SignedLocation
	}
	if r.StartDate != nil {
		contract.StartDate = *r.StartDate
	}
	if r.EndDate != nil {
		contract.EndDate = *r.EndDate
	}

	// Update currency
	if r.Currency != nil {
		contract.Currency = r.Currency
	}

	// Update complex JSONB fields
	if r.FinancialTerms != nil {
		financialTermsJSON, err := json.Marshal(r.FinancialTerms)
		if err != nil {
			return errors.New("invalid financial_terms: failed to marshal to JSON")
		}
		contract.FinancialTerms = financialTermsJSON
	}

	if r.ScopeOfWork != nil {
		scopeOfWorkJSON, err := json.Marshal(r.ScopeOfWork)
		if err != nil {
			return errors.New("invalid scope_of_work: failed to marshal to JSON")
		}
		contract.ScopeOfWork = scopeOfWorkJSON
	}

	if r.LegalTerms != nil {
		legalTermsJSON, err := json.Marshal(r.LegalTerms)
		if err != nil {
			return errors.New("invalid legal_terms: failed to marshal to JSON")
		}
		contract.LegalTerms = legalTermsJSON
	}

	// Update file URLs
	if r.ContractFileURL != nil {
		contract.ContractFileURL = r.ContractFileURL
	}
	if r.ProposalFileURL != nil {
		contract.ProposalFileURL = r.ProposalFileURL
	}

	return nil
}

// ContractFilterRequest represents query parameters for filtering contracts
type ContractFilterRequest struct {
	PaginationRequest
	BrandID   *string    `form:"brand_id" json:"brand_id" validate:"omitempty,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type      *string    `form:"type" json:"type" validate:"omitempty,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Status    *string    `form:"status" json:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETED TERMINATED" example:"ACTIVE"`
	Keyword   *string    `form:"keyword" json:"keyword" validate:"omitempty,max=255" example:"contract title"`
	StartDate *time.Time `form:"start_date" json:"start_date" validate:"omitempty" example:"2023-10-01T00:00:00Z"`
	EndDate   *time.Time `form:"end_date" json:"end_date" validate:"omitempty" example:"2023-12-31T23:59:59Z"`
}
