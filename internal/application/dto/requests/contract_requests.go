package requests

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// region: ======= CreateContractRequest =======

// CreateContractRequest represents the payload for creating a new contract.
type CreateContractRequest struct {
	// Parent contract (for amendments or related contracts)
	ParentContractID *string `json:"parent_contract_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Contract basic information
	Title          string  `json:"title" validate:"required,min=2,max=255" example:"Social Media Promotion Contract"`
	Type           string  `json:"type" validate:"required,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Status         *string `json:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETED TERMINATED" example:"DRAFT"`
	DepositPercent *int    `json:"deposit_percent" validate:"omitempty,min=0,max=100" example:"30"`
	DepositAmount  *int    `json:"deposit_amount" validate:"omitempty,min=0" example:"3000000"`
	IsDepositPaid  bool    `json:"is_deposit_paid" validate:"omitempty" example:"false"`

	// Brand information (stored in contract for record-keeping)
	BrandID                string  `json:"brand_id" validate:"required,uuid4" example:"660e8400-e29b-41d4-a716-446655440000"`
	BrandBankName          *string `json:"brand_bank_name" validate:"omitempty,max=255" example:"Vietcombank"`
	BrandBankAccountNumber *string `json:"brand_bank_account_number" validate:"omitempty,max=255" example:"0123456789"`
	BrandBankAccountHolder *string `json:"brand_bank_account_holder" validate:"omitempty,max=255" example:"ABC Company Ltd."`

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
	StartDate      time.Time `json:"start_date" validate:"required,gtefield=SignedDate" example:"2023-10-02T00:00:00Z"`
	EndDate        time.Time `json:"end_date" validate:"required,gtefield=StartDate" example:"2023-12-31T23:59:59Z"`

	// Financial
	Currency *string `json:"currency" validate:"omitempty,len=3" example:"VND"`

	// Complex JSONB fields
	FinancialTerms dtos.FinancialTerms `json:"financial_terms" validate:"required"`
	ScopeOfWork    dtos.ScopeOfWork    `json:"scope_of_work" validate:"required"`
	LegalTerms     dtos.LegalTerms     `json:"legal_terms" validate:"required"`

	// File URLs
	ContractFileURL *string `json:"contract_file_url" validate:"omitempty,url" example:"https://example.com/contracts/contract.pdf"`
	ProposalFileURL *string `json:"proposal_file_url" validate:"omitempty,url" example:"https://example.com/proposals/proposal.pdf"`
}

// ToContract converts CreateContractRequest to model.Contract
func (r *CreateContractRequest) ToContract(ctx context.Context) (*model.Contract, error) {
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

	var wg sync.WaitGroup
	wg.Add(3)
	errorsChan := make(chan error)

	var financialTermsJSON []byte
	var scopeOfWorkJSON []byte
	var legalTermsJSON []byte

	// Goroutine to unmarshal FinancialTerms into specific struct based on ContractType then marshal to JSON for storage
	go func(contract *CreateContractRequest) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
		}

		var financialTermsErr error

		switch contractType {
		case enum.ContractTypeAdvertising, enum.ContractTypeAmbassador:
			var advertisingFinancialTerms *dtos.AdvertisingFinancialTerms
			advertisingFinancialTerms, financialTermsErr = contract.FinancialTerms.ConvertToAdvertisingFinancialTerms()
			if financialTermsErr != nil {
				errorsChan <- fmt.Errorf("invalid financial_terms: %v", financialTermsErr)
				return
			}

			if r.DepositAmount == nil || *r.DepositAmount <= 0 {
				calculatedAmount := int(float64((advertisingFinancialTerms.TotalCost * *r.DepositPercent) / 100))
				r.DepositAmount = &calculatedAmount
			}

			for _, schedule := range advertisingFinancialTerms.Schedules {
				if schedule.Amount == 0 {
					schedule.Amount = int(math.Round(float64((advertisingFinancialTerms.TotalCost * schedule.Percent) / 100)))
				}
			}

			financialTermsJSON, financialTermsErr = json.Marshal(advertisingFinancialTerms)
			if financialTermsErr != nil {
				errorsChan <- errors.New("invalid financial_terms: failed to marshal advertising financial terms to JSON")
				return
			}

			if contract.FinancialTerms.Schedule == nil || len(*contract.FinancialTerms.Schedule) == 0 {
				sortedSchedules := *contract.FinancialTerms.Schedule
				slices.SortFunc(sortedSchedules, func(a, b dtos.Schedule) int {
					dueDateA, _ := time.Parse(utils.DateFormat, fmt.Sprintf("%v", a.DueDate))
					dueDateB, _ := time.Parse(utils.DateFormat, fmt.Sprintf("%v", b.DueDate))

					return dueDateA.Compare(dueDateB)
				})
				for i := range sortedSchedules {
					if sortedSchedules[i].ID == nil {
						id := int8(i + 1)
						advertisingFinancialTerms.Schedules[i].ID = &id
					}
				}
				contract.FinancialTerms.Schedule = &sortedSchedules
			}
		case enum.ContractTypeCoProduce:
			var coProducingFinancialTerms *dtos.CoProducingFinancialTerms
			coProducingFinancialTerms, financialTermsErr = contract.FinancialTerms.ConvertToCoProducingFinancialTerms()
			if financialTermsErr != nil {
				errorsChan <- fmt.Errorf("invalid financial_terms: %v", financialTermsErr)
				return
			}

			r.DepositPercent = nil

			financialTermsJSON, financialTermsErr = json.Marshal(coProducingFinancialTerms)
			if financialTermsErr != nil {
				errorsChan <- errors.New("invalid financial_terms: failed to marshal co-producing financial terms to JSON")
				return
			}
		case enum.ContractTypeAffiliate:
			var affiliateFinancialTerms *dtos.AffiliateFinancialTerms
			affiliateFinancialTerms, financialTermsErr = contract.FinancialTerms.ConvertToAffiliateFinancialTerms()
			if financialTermsErr != nil {
				errorsChan <- fmt.Errorf("invalid financial_terms: %v", financialTermsErr)
				return
			}

			r.DepositPercent = nil

			financialTermsJSON, financialTermsErr = json.Marshal(affiliateFinancialTerms)
			if financialTermsErr != nil {
				errorsChan <- errors.New("invalid financial_terms: failed to marshal affiliate financial terms to JSON")
				return
			}
		}
	}(r)

	// Goroutine to unmarshal ScopeOfWork into specific struct based on ContractType then marshal to JSON for storage
	go func(contract *CreateContractRequest) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
		}
		var scopeOfWorkErr error

		scopeOfWorkDTO := &dtos.ScopeOfWorkDto{
			GeneralRequirements: contract.ScopeOfWork.GeneralRequirements,
		}

		switch contractType {
		case enum.ContractTypeAdvertising:
			var advertisingDeliverable *dtos.AdvertisingDeliverable
			advertisingDeliverable, scopeOfWorkErr = contract.ScopeOfWork.Deliverables.ToAdvertisingDeliverable()
			if scopeOfWorkErr != nil {
				errorsChan <- fmt.Errorf("invalid scope_of_work.deliverables: %v", scopeOfWorkErr)
				return
			}
			scopeOfWorkDTO.Deliverables = advertisingDeliverable

			for i := range advertisingDeliverable.AdvertisedItems {
				if advertisingDeliverable.AdvertisedItems[i].ID == nil {
					id := int8(i + 1)
					advertisingDeliverable.AdvertisedItems[i].ID = &id
				}
			}
		case enum.ContractTypeAffiliate:
			var affiliateDeliverable *dtos.AffiliateDeliverable
			affiliateDeliverable, scopeOfWorkErr = contract.ScopeOfWork.Deliverables.ToAffiliateDeliverable()
			if scopeOfWorkErr != nil {
				errorsChan <- fmt.Errorf("invalid scope_of_work.deliverables: %v", scopeOfWorkErr)
				return
			}
			scopeOfWorkDTO.Deliverables = affiliateDeliverable

			for i := range affiliateDeliverable.AdvertisedItems {
				if affiliateDeliverable.AdvertisedItems[i].ID == nil {
					id := int8(i + 1)
					affiliateDeliverable.AdvertisedItems[i].ID = &id
				}
			}
		case enum.ContractTypeAmbassador:
			var brandAmbassadorDeliverable *dtos.BrandAmbassadorDeliverable
			brandAmbassadorDeliverable, scopeOfWorkErr = contract.ScopeOfWork.Deliverables.ToBrandAmbassadorDeliverable()
			if scopeOfWorkErr != nil {
				errorsChan <- fmt.Errorf("invalid scope_of_work.deliverables: %v", scopeOfWorkErr)
				return
			}
			scopeOfWorkDTO.Deliverables = brandAmbassadorDeliverable

			for i := range brandAmbassadorDeliverable.Events {
				if brandAmbassadorDeliverable.Events[i].ID == nil {
					id := int8(i + 1)
					brandAmbassadorDeliverable.Events[i].ID = &id
				}
			}
		case enum.ContractTypeCoProduce:
			var coProducingDeliverable *dtos.CoProducingDeliverable
			coProducingDeliverable, scopeOfWorkErr = contract.ScopeOfWork.Deliverables.ToCoProducingDeliverable()
			if scopeOfWorkErr != nil {
				errorsChan <- fmt.Errorf("invalid scope_of_work.deliverables: %v", scopeOfWorkErr)
				return
			}
			scopeOfWorkDTO.Deliverables = coProducingDeliverable

			for i := range coProducingDeliverable.Concepts {
				if coProducingDeliverable.Concepts[i].ID == nil {
					id := int8(i + 1)
					coProducingDeliverable.Concepts[i].ID = &id
				}
			}
		}

		scopeOfWorkJSON, scopeOfWorkErr = json.Marshal(scopeOfWorkDTO)
		if scopeOfWorkErr != nil {
			errorsChan <- errors.New("invalid scope_of_work: failed to marshal to JSON")
		}
	}(r)

	// Goroutine to marshal LegalTerms to JSON for storage
	go func(contract *CreateContractRequest) {
		defer wg.Done()
		select {
		case <-ctx.Done():
			return
		default:
		}
		var legalTermsErr error

		legalTermsJSON, legalTermsErr = json.Marshal(contract.LegalTerms)
		if legalTermsErr != nil {
			errorsChan <- errors.New("invalid legal_terms: failed to marshal to JSON")
		}
	}(r)

	wg.Wait()
	close(errorsChan)
	var errs []error
	for err := range errorsChan {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("validation errors: %v | see logs for details", errors.Join(errs...))
	}

	return &model.Contract{
		ParentContractID: parentContractID,
		Title:            &r.Title,
		ContractNumber: utils.PtrOrNil(fmt.Sprintf("CONTRACT-%s-%s-%s",
			utils.AbbreviateString(contractType.String(), 4),
			utils.AbbreviateString(r.BrandID, 4),
			utils.GetFormattedCurrentTime(utils.TimestampStringFormat, "")),
		),
		BrandID:                         &brandID,
		BrandBankName:                   r.BrandBankName,
		BrandBankAccountNumber:          r.BrandBankAccountNumber,
		BrandBankAccountHolder:          r.BrandBankAccountHolder,
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
		DepositPercent:                  r.DepositPercent,
		DepositAmount:                   r.DepositAmount,
		IsDepositPaid:                   &r.IsDepositPaid,
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

// endregion

// region: ======= UpdateContractRequest =======

// UpdateContractRequest represents the payload for updating an existing contract.
type UpdateContractRequest struct {
	// Parent contract (for amendments or related contracts)
	ParentContractID *string `json:"parent_contract_id" validate:"omitempty,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`

	// Contract basic information
	Title          *string `json:"title" validate:"omitempty,min=2,max=255" example:"Updated Contract Title"`
	ContractNumber *string `json:"contract_number" validate:"omitempty,min=2,max=255" example:"CONTRACT-2023-001-UPDATED"`
	Type           *string `json:"type" validate:"omitempty,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Status         *string `json:"status" validate:"omitempty,oneof=DRAFT ACTIVE COMPLETED TERMINATED" example:"ACTIVE"`

	// Brand information (stored in contract for record-keeping)
	BrandID                *string `json:"brand_id" validate:"omitempty,uuid4" example:"660e8400-e29b-41d4-a716-446655440000"`
	BrandBankName          *string `json:"brand_bank_name" validate:"omitempty,max=255" example:"Vietcombank"`
	BrandBankAccountNumber *string `json:"brand_bank_account_number" validate:"omitempty,max=255" example:"0123456789"`
	BrandBankAccountHolder *string `json:"brand_bank_account_holder" validate:"omitempty,max=255" example:"ABC Company Ltd."`

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

// endregion

// region: ======= ContractFilterRequest =======

// ContractFilterRequest represents query parameters for filtering contracts
type ContractFilterRequest struct {
	PaginationRequest
	BrandID    *string `form:"brand_id" json:"brand_id" validate:"omitempty,uuid4" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type       *string `form:"type" json:"type" validate:"omitempty,oneof=ADVERTISING AFFILIATE BRAND_AMBASSADOR CO_PRODUCING" example:"ADVERTISING"`
	Status     *string `form:"status" json:"status" validate:"omitempty,oneof=DRAFT APPROVED ACTIVE COMPLETED INACTIVE TERMINATED" example:"ACTIVE"`
	Keyword    *string `form:"keyword" json:"keyword" validate:"omitempty,max=255" example:"contract title"`
	StartDate  *string `form:"start_date" json:"start_date" validate:"omitempty" example:"2023-10-01"`
	EndDate    *string `form:"end_date" json:"end_date" validate:"omitempty" example:"2023-12-31"`
	NoCampaign *bool   `form:"no_campaign" json:"no_campaign" validate:"omitempty" example:"true"`
}

// endregion

// region: ======== Custom Validator Functions =======

// CreateContractRequestValidator performs custom validation for CreateContractRequest
// It handle complex validations for specific JSON fields based on contract type and other business rules. More specifically:
// - For financial_terms, it validates the structure and values based on the contract type (e.g., ADVERTISING, CO_PRODUCING, AFFICLIATE).
//   - For ADVERTISING and BRAND_AMBASSADOR types, it ensures the financial model is valid, the total cost breakdown matches the total amount, and the schedules are in ascending order and within the contract period.
//   - For CO_PRODUCING type, it ensures the profit distribution cycle and dates are valid and within the contract period.
//   - For AFFILIATE type, it ensure the level values are valid and the payment cycle and dates are valid and within the contract period.
//
// - It also cross-validates deposit_amount and deposit_percent to ensure consistency with the total cost in financial_terms.
//
// - For scope_of_work, it validates the structure and values based on the contract type (e.g., ADVERTISING, CO_PRODUCING, AFFICLIATE).
func CreateContractRequestValidator(sl validator.StructLevel) {
	contract := sl.Current().Interface().(CreateContractRequest)

	// Validate deposit percent and amount
	if contract.DepositAmount == nil && contract.DepositPercent == nil {
		sl.ReportError(contract.DepositAmount, "deposit_amount", "DepositAmount", "depositinfo", "at least one of deposit_amount or deposit_percent must be provided")
	}
	if *contract.DepositPercent > 50 {
		sl.ReportError(contract.DepositPercent, "deposit_percent", "DepositPercent", "maxdepositpercent", "deposit_percent cannot exceed 50%")
	}

	contractType := enum.ContractType(contract.Type)
	if !contractType.IsValid() {
		sl.ReportError(contract.Type, "type", "Type", "contracttype", "")
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	type ValidateError struct {
		CurrentValue any
		JSONName     string
		StructName   string
		Tag          string
		Param        string
		Error        error
	}
	errorsChan := make(chan ValidateError, 30)

	// Goroutine to validate FinancialTerms
	go func(contract CreateContractRequest, contractType enum.ContractType) {
		defer wg.Done()

		rawFinancialTerms, err := json.Marshal(contract.FinancialTerms)
		if err != nil {
			errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms", StructName: "FinancialTerms", Tag: "financialterms", Param: "", Error: err}
			return
		} else if len(rawFinancialTerms) == 0 {
			errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms", StructName: "FinancialTerms", Tag: "financialterms", Param: "", Error: errors.New("financial_terms cannot be empty")}
			return
		}

		// Validate ability to unmarshal FinancialTerms based on contract type
		switch contractType {
		case enum.ContractTypeAdvertising, enum.ContractTypeAmbassador:
			var financialTerms dtos.AdvertisingFinancialTerms
			if err = json.Unmarshal([]byte(json.RawMessage(rawFinancialTerms)), &financialTerms); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms", StructName: "FinancialTerms", Tag: "financialterms_advertising", Param: "", Error: err}
				break
			}

			if err = sl.Validator().Struct(financialTerms); err != nil {
				sl.ReportError(contract.FinancialTerms, "financial_terms", "FinancialTerms", "financialterms_advertising", "")
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms", StructName: "FinancialTerms", Tag: "financialterms_advertising", Param: "", Error: err}
			}

			if contract.DepositAmount != nil && contract.DepositPercent != nil {
				calculatedDepositAmount := int(math.Round(float64((financialTerms.TotalCost * (*contract.DepositPercent)) / 100)))
				if *contract.DepositAmount != calculatedDepositAmount {
					errorsChan <- ValidateError{CurrentValue: contract.DepositAmount, JSONName: "deposit_amount", StructName: "DepositAmount", Tag: "depositamount", Param: "", Error: fmt.Errorf("deposit_amount does not match deposit_percent and total_cost. Expected %d but got %d", calculatedDepositAmount, *contract.DepositAmount)}
				}
			}

			// Validate Model field is valid enum
			financialModel := enum.FinancialTermsModel(financialTerms.Model)
			if !financialModel.IsValid() {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.model", StructName: "Model", Tag: "financialtermsmodel", Param: "", Error: errors.New("invalid financial_terms.model: must be one of FIXED, LEVELS, or SHARE")}
			}

			// Validate TotalBreakdown equals TotalAmount
			costBreakDownValues := utils.GetValues(financialTerms.CostBreakdown)
			costBreakDownSum := 0
			for _, v := range costBreakDownValues {
				costBreakDownSum += v
			}
			if costBreakDownSum != financialTerms.TotalCost {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.cost_breakdown", StructName: "CostBreakdown", Tag: "financialtermscostbreakdown", Param: "", Error: errors.New("invalid financial_terms.cost_breakdown: total cost must equal financial_terms.total_cost")}
			}

			// Validate if the total Schedules percentage equals too 100% and Amount are calculate correctly
			// The calculated data will includes deposit amount if provided
			sortedFinancialSchedules := financialTerms.Schedules
			slices.SortFunc(sortedFinancialSchedules, func(a, b dtos.Schedule) int {
				dueDateA, _ := time.Parse(utils.DateFormat, fmt.Sprintf("%v", a.DueDate))
				dueDateB, _ := time.Parse(utils.DateFormat, fmt.Sprintf("%v", b.DueDate))

				return dueDateA.Compare(dueDateB)
			})
			totalPercent := 0
			errorValue := 0
			totalCalculatedAmount := 0
			isAmountExisted := false
			if contract.DepositAmount != nil {
				totalPercent += int((float64(*contract.DepositAmount) / float64(financialTerms.TotalCost)) * 100)
				totalCalculatedAmount += *contract.DepositAmount
			} else if contract.DepositPercent != nil {
				totalPercent += *contract.DepositPercent
				calculated := int(math.Round(float64((financialTerms.TotalCost * *contract.DepositPercent) / 100)))
				if calculated != 0 {
					isAmountExisted = true
				}
				totalCalculatedAmount += calculated
			}
			for _, schedule := range sortedFinancialSchedules {
				totalPercent += schedule.Percent
				calculated := int(math.Round(float64((financialTerms.TotalCost * schedule.Percent) / 100)))
				if calculated != 0 {
					isAmountExisted = true
				}
				totalCalculatedAmount += calculated
				errorValue += int(math.Abs(float64(schedule.Amount - calculated)))

				if errorValue > 5000 {
					errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.schedules.amount", StructName: "Amount", Tag: "financialtermsschedulesamount", Param: "", Error: errors.New("financial_terms.schedules.amount must be correctly calculated based on financial_terms.total_cost and financial_terms.schedules.percent")}
				} else if schedule.Amount != calculated {
					errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.schedules.amount", StructName: "Amount", Tag: "financialtermsschedulesamount", Param: "", Error: errors.New("financial_terms.schedules.amount must be equal to ")}
				}
			}
			if totalPercent != 100 {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.schedules.percent", StructName: "Percent", Tag: "financialtermsschedulespercent", Param: "", Error: errors.New("invalid financial_terms.schedules: total percent must equal 100%")}
			}
			if isAmountExisted && int(math.Abs(float64(financialTerms.TotalCost-totalCalculatedAmount))) != errorValue {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.total_cost", StructName: "TotalCost", Tag: "financialtermstotalcost", Param: "", Error: errors.New("invalid financial_terms.total_cost: total cost must be correctly calculated based on financial_terms.schedules.amount")}
			}

		case enum.ContractTypeCoProduce:
			var financialTerms dtos.CoProducingFinancialTerms
			if err = json.Unmarshal([]byte(json.RawMessage(rawFinancialTerms)), &financialTerms); err != nil {
				sl.ReportError(contract.FinancialTerms, "financial_terms", "FinancialTerms", "financialterms", "")
			} else {
				financialTerms := enum.FinancialTermsModel(financialTerms.Model)
				if financialTerms != enum.FinancialTermsModelShare {
					sl.ReportError(contract.FinancialTerms, "financial_terms.model", "Model", "financialtermsmodel", "")
				}
			}

			if err = sl.Validator().Struct(financialTerms); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms", StructName: "FinancialTerms", Tag: "financialterms_coproducing", Param: "", Error: err}
			}

			if contract.DepositAmount == nil || *contract.DepositAmount <= 0 {
				errorsChan <- ValidateError{CurrentValue: contract.DepositAmount, JSONName: "deposit_amount", StructName: "DepositAmount", Tag: "depositamount", Param: "", Error: errors.New("deposit_amount must be provided and greater than 0 for Co-Producing contracts")}
			}

			// Validate TotalBreakdown equals TotalAmount
			costBreakDownValues := utils.GetValues(financialTerms.CostBreakdown)
			costBreakDownSum := 0
			for _, v := range costBreakDownValues {
				costBreakDownSum += v
			}
			if costBreakDownSum != financialTerms.TotalCost {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.cost_breakdown", StructName: "CostBreakdown", Tag: "financialtermscostbreakdown", Param: "", Error: errors.New("invalid financial_terms.cost_breakdown: total cost must equal financial_terms.total_cost")}
			}

			// Validate the ProfitDistributionDate of the first and last Distribution Cycle are within the contract period
			profitDistributionCycle := enum.PaymentCycle(financialTerms.ProfitDistributionCycle)
			if !profitDistributionCycle.IsValid() {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribution_cycle", StructName: "PaymentCycle", Tag: "financialtermspaymentcycle", Param: "", Error: errors.New("invalid financial_terms.profit_distribution_cycle: must be one of MONTHLY, QUARTERLY, or ANNUALLY")}
			} else {
				contractStartDate := contract.StartDate
				contractEndDate := contract.EndDate

				//Validate PaymentDate datatype based on PaymentCycle
				switch profitDistributionCycle {
				case enum.PaymentCycleMonthly:
					distributionDayStr, ok := financialTerms.ProfitDistributionDate.(string)
					if !ok {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: for MONTHLY profit_distribution_cycle, profit_distribbution_date must be a string")}
						break
					}
					var paymentDay int
					paymentDay, err = strconv.Atoi(distributionDayStr)
					if err != nil || paymentDay < 1 || paymentDay > 31 {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: for MONTHLY profit_distribution_cycle, profit_distribbution_date must be a string representing a day of the month (1-31)")}
						break
					}

					firstYear, firstMonth, _ := contractStartDate.Date()
					lastYear, lastMonth, _ := contractEndDate.Date()
					firstPaymentDate := time.Date(firstYear, firstMonth, paymentDay, 0, 0, 0, 0, time.Local)
					lastPaymentDate := time.Date(lastYear, lastMonth, paymentDay, 0, 0, 0, 0, time.Local)
					if firstPaymentDate.Before(contractStartDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: the first payment date must be on or after the contract start_date")}
					}
					if lastPaymentDate.After(contractEndDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: the last payment date must be on or before the contract end_date")}
					}

				case enum.PaymentCycleQuarterly:
					distributionDateQuarterlyInterface, ok := financialTerms.ProfitDistributionDate.([]any)
					if !ok {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: for QUARTERLY profit_distribution_cycle, profit_distribbution_date must be an array 4 PaymentDate objects")}
					}
					var distributionDateQuarterly []dtos.PaymentDate
					for _, pd := range distributionDateQuarterlyInterface {
						var rawBytes []byte
						rawBytes, err = json.Marshal(pd)
						if err != nil {
							errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: failed to marshal PaymentDate object")}
							break
						}
						var paymentDateObj dtos.PaymentDate
						if err = json.Unmarshal(rawBytes, &paymentDateObj); err != nil {
							errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: failed to unmarshal to PaymentDate object")}
							break
						}
						distributionDateQuarterly = append(distributionDateQuarterly, paymentDateObj)
					}

					// Convert PaymentDate to time.Time slices with sorting by date
					quarterDateLen := len(distributionDateQuarterly)
					paymentQuarterSlices := make([]time.Time, quarterDateLen)
					for i, pd := range distributionDateQuarterly {
						slice := time.Date(int(pd.Year), time.Month(pd.Month), int(pd.Day), 0, 0, 0, 0, time.Local)
						paymentQuarterSlices[i] = slice
					}
					slices.SortFunc(paymentQuarterSlices, func(a, b time.Time) int { return a.Compare(b) })
					for i := 1; i < quarterDateLen; i++ {
						if paymentQuarterSlices[i].Equal(paymentQuarterSlices[i-1]) {
							errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: payment dates must be unique")}
						}
					}

					// Validate the first and last PaymentDate are within contract period
					if paymentQuarterSlices[0].Before(contractStartDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: the first payment date must be on or after the contract start_date")}
					}
					if paymentQuarterSlices[quarterDateLen-1].After(contractEndDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: the last payment date must be on or before the contract end_date")}
					}

				case enum.PaymentCycleAnnually:
					var distributionDate time.Time
					distributionDate, err = time.Parse("2006-01-02", fmt.Sprintf("%v", financialTerms.ProfitDistributionDate))
					if err != nil {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: for ANNUALLY profit_distribution_cycle, profit_distribbution_date must be a string in 'YYYY-MM-DD' format")}
						break
					}

					startYear, _, _ := contractStartDate.Date()
					endYear, _, _ := contractEndDate.Date()
					firstPaymentDate := time.Date(startYear, distributionDate.Month(), distributionDate.Day(), 0, 0, 0, 0, time.Local)
					lastPaymentDate := time.Date(endYear, distributionDate.Month(), distributionDate.Day(), 0, 0, 0, 0, time.Local)
					if firstPaymentDate.Before(contractStartDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: the first payment date must be on or after the contract start_date")}
					}
					if lastPaymentDate.After(contractEndDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.profit_distribbution_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.profit_distribbution_date: the last payment date must be on or before the contract end_date")}
					}
				}
			}

			// Validate the KolPercent and the CompanyPercent sum to 100%
			if financialTerms.KolPercent+financialTerms.CompanyPercent != 100 {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.kol_percent", StructName: "KolPercent", Tag: "financialtermskolpercent", Param: "", Error: errors.New("invalid financial_terms.kol_percent and financial_terms.company_percent: must sum to 100%")}
			}

		case enum.ContractTypeAffiliate:
			var financialTerms dtos.AffiliateFinancialTerms
			if err = json.Unmarshal([]byte(json.RawMessage(rawFinancialTerms)), &financialTerms); err != nil {
				sl.ReportError(contract.FinancialTerms, "financial_terms", "FinancialTerms", "financialterms", "")
			} else {
				financialTerms := enum.FinancialTermsModel(financialTerms.Model)
				if financialTerms != enum.FinancialTermsModelLevels {
					sl.ReportError(contract.FinancialTerms, "financial_terms.model", "Model", "financialtermsmodel", "")
				}
			}

			if err = sl.Validator().Struct(financialTerms); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms", StructName: "FinancialTerms", Tag: "financialterms_affiliate", Param: "", Error: err}
			}

			// Validate TotalBreakdown equals TotalAmount
			costBreakDownValues := utils.GetValues(financialTerms.CostBreakdown)
			costBreakDownSum := 0
			for _, v := range costBreakDownValues {
				costBreakDownSum += v
			}
			if costBreakDownSum != financialTerms.TotalCost {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.cost_breakdown", StructName: "CostBreakdown", Tag: "financialtermscostbreakdown", Param: "", Error: errors.New("invalid financial_terms.cost_breakdown: total cost must equal financial_terms.total_cost")}
			}

			if contract.DepositAmount == nil || *contract.DepositAmount <= 0 {
				errorsChan <- ValidateError{CurrentValue: contract.DepositAmount, JSONName: "deposit_amount", StructName: "DepositAmount", Tag: "depositamount", Param: "", Error: errors.New("deposit_amount must be provided and greater than 0 for AFFILIATE contracts")}
			}

			// Validate Levels to ensure max_clicks are in ascending order
			sortedLevels := financialTerms.Levels
			slices.SortFunc(sortedLevels, func(a, b dtos.Level) int { return a.Level - b.Level })
			for i := 1; i < len(sortedLevels); i++ {
				level1 := sortedLevels[i-1]
				level2 := sortedLevels[i]
				if level1.MaxClicks >= level2.MaxClicks {
					errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.levels", StructName: "Levels", Tag: "financialtermslevels", Param: "", Error: fmt.Errorf("invalid financial_terms.levels: max_clicks of level %d must be greater than level %d", sortedLevels[i].Level, sortedLevels[i-1].Level)}
				}
				if level1.Multiplier >= level2.Multiplier {
					errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.levels", StructName: "Levels", Tag: "financialtermslevels", Param: "", Error: fmt.Errorf("invalid financial_terms.levels: multiplier of level %d must be greater than level %d", sortedLevels[i].Level, sortedLevels[i-1].Level)}
				}
			}

			// Validate the PaymentDate of the first and last Payment Cycle are within the contract period
			paymentCycle := enum.PaymentCycle(financialTerms.PaymentCycle)
			if !paymentCycle.IsValid() {
				errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_cycle", StructName: "PaymentCycle", Tag: "financialtermspaymentcycle", Param: "", Error: errors.New("invalid financial_terms.payment_cycle: must be one of MONTHLY, QUARTERLY, or ANNUALLY")}
			} else {
				contractStartDate := contract.StartDate
				contractEndDate := contract.EndDate

				//Validate PaymentDate datatype based on PaymentCycle
				switch paymentCycle {
				case enum.PaymentCycleMonthly:
					paymentDayStr, ok := financialTerms.PaymentDate.(string)
					if !ok {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: for MONTHLY payment_cycle, payment_date must be a string")}
						break
					}
					paymentDay, err := strconv.Atoi(paymentDayStr)
					if err != nil || paymentDay < 1 || paymentDay > 31 {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: for MONTHLY payment_cycle, payment_date must be a string representing a day of the month (1-31)")}
						break
					}

					firstYear, firstMonth, _ := contractStartDate.Date()
					lastYear, lastMonth, _ := contractEndDate.Date()
					firstPaymentDate := time.Date(firstYear, firstMonth, paymentDay, 0, 0, 0, 0, time.Local)
					lastPaymentDate := time.Date(lastYear, lastMonth, paymentDay, 0, 0, 0, 0, time.Local)
					if firstPaymentDate.Before(contractStartDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: the first payment date must be on or after the contract start_date")}
					}
					if lastPaymentDate.After(contractEndDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: the last payment date must be on or before the contract end_date")}
					}
				case enum.PaymentCycleQuarterly:
					paymentDateQuarterly, ok := financialTerms.PaymentDate.([]dtos.PaymentDate)
					if !ok {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: for QUARTERLY payment_cycle, payment_date must be an array of PaymentDate objects")}
					}

					// Convert PaymentDate to time.Time slices with sorting by date
					quarterDateLen := len(paymentDateQuarterly)
					paymentQuarterSlices := make([]time.Time, quarterDateLen)
					for i, pd := range paymentDateQuarterly {
						slice := time.Date(int(pd.Year), time.Month(pd.Month), int(pd.Day), 0, 0, 0, 0, time.Local)
						if i == 0 {
							paymentQuarterSlices[0] = slice
							continue
						}
						for _, existing := range paymentQuarterSlices {
							if existing.Equal(slice) {
								errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: duplicate PaymentDate objects")}
							} else if existing.After(slice) {
								copy(paymentQuarterSlices[i+1:], paymentQuarterSlices[i:])
								paymentQuarterSlices[i] = slice
								break
							}
						}
					}

					// Validate the first and last PaymentDate are within contract period
					if paymentQuarterSlices[0].Before(contractStartDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: the first payment date must be on or after the contract start_date")}
					}
					if paymentQuarterSlices[3].After(contractEndDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: the last payment date must be on or before the contract end_date")}
					}
				case enum.PaymentCycleAnnually:
					paymentDate, err := time.Parse("2006-01-02", fmt.Sprintf("%v", financialTerms.PaymentDate))
					if err != nil {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: for ANNUALLY payment_cycle, payment_date must be a string in 'YYYY-MM-DD' format")}
						break
					}

					startYear, _, _ := contractStartDate.Date()
					endYear, _, _ := contractEndDate.Date()
					firstPaymentDate := time.Date(startYear, paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, time.Local)
					lastPaymentDate := time.Date(endYear, paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, time.Local)
					if firstPaymentDate.Before(contractStartDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: the first payment date must be on or after the contract start_date")}
					}
					if lastPaymentDate.After(contractEndDate) {
						errorsChan <- ValidateError{CurrentValue: contract.FinancialTerms, JSONName: "financial_terms.payment_date", StructName: "PaymentDate", Tag: "financialtermspaymentdate", Param: "", Error: errors.New("invalid financial_terms.payment_date: the last payment date must be on or before the contract end_date")}
					}
				}
			}
		}
	}(contract, contractType)

	// Goroutine to validate ScopeOfWork
	go func(contract CreateContractRequest, contractType enum.ContractType) {
		defer wg.Done()

		rawDeliverables, err := json.Marshal(contract.ScopeOfWork.Deliverables)
		if err != nil {
			errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables", StructName: "Deliverables", Tag: "deliverables", Param: "", Error: err}
			return
		} else if len(rawDeliverables) == 0 {
			errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables", StructName: "Deliverables", Tag: "deliverables", Param: "", Error: errors.New("scope_of_work.deliverables cannot be empty")}
		}
		switch contractType {
		case enum.ContractTypeAdvertising:
			var deliverables dtos.AdvertisingDeliverable
			if err = json.Unmarshal([]byte(json.RawMessage(rawDeliverables)), &deliverables); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables", StructName: "Deliverables", Tag: "deliverables_advertising", Param: "", Error: err}
			}
		case enum.ContractTypeAffiliate:
			var deliverables dtos.AffiliateDeliverable
			if err = json.Unmarshal([]byte(json.RawMessage(rawDeliverables)), &deliverables); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables", StructName: "Deliverables", Tag: "deliverables", Param: "", Error: err}
			}

			// Validate each AdvertisedItem platform is one of the tracking platforms defined in the Deliverables Platform field
			trackingPlatforms := deliverables.Platform
			for i, advertised := range deliverables.AdvertisedItems {
				if !slices.Contains(trackingPlatforms, advertised.Platform) {
					errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables.advertised_items.platform", StructName: "Platform", Tag: "platform", Param: "", Error: fmt.Errorf("invalid scope_of_work.deliverables.advertised_items[%d].platform: platform %s must be one of the tracking platforms in scope_of_work.deliverables.platform", i, advertised.Platform)}
				}
			}
		case enum.ContractTypeAmbassador:
			var deliverables dtos.BrandAmbassadorDeliverable
			if err = json.Unmarshal([]byte(json.RawMessage(rawDeliverables)), &deliverables); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables", StructName: "Deliverables", Tag: "deliverables", Param: "", Error: err}
			}

		case enum.ContractTypeCoProduce:
			var deliverables dtos.CoProducingDeliverable
			if err = json.Unmarshal([]byte(json.RawMessage(rawDeliverables)), &deliverables); err != nil {
				errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables", StructName: "Deliverables", Tag: "deliverables", Param: "", Error: err}
			}

			// Validate each Concept's ProductID exists in the Products array
			productIDs := utils.UniqueSliceMapper(deliverables.Products, func(product dtos.CoProducingProduct) int8 { return *product.ID })
			for i, concept := range deliverables.Concepts {
				if !slices.Contains(productIDs, concept.ProductID) {
					errorsChan <- ValidateError{CurrentValue: contract.ScopeOfWork, JSONName: "scope_of_work.deliverables.concepts.product_id", StructName: "ProductID", Tag: "product_id", Param: "", Error: fmt.Errorf("invalid scope_of_work.deliverables.concepts[%d].product_id: product_id %d must be one of the product IDs in scope_of_work.deliverables.products", i, concept.ProductID)}
				}
			}
		}
	}(contract, contractType)

	wg.Wait()

	close(errorsChan)
	if len(errorsChan) > 0 {
		zapFields := make([]zap.Field, 0, len(errorsChan))

		errorsList := make([]ValidateError, 0, len(errorsChan))
		for errors := range errorsChan {
			errorsList = append(errorsList, errors)
		}

		for _, error := range errorsList {
			// Param is not used in the validation error, so it is used as a workaround for custom error messages
			customParamMsg := error.Error.Error()
			sl.ReportError(error.CurrentValue, error.JSONName, error.StructName, error.Tag, customParamMsg)
			zapFields = append(zapFields, zap.Error(error.Error))
		}
		zap.L().Warn("CreateContractRequestValidator found validation errors", zapFields...)
	}
}

// endregion
