package dtos

import (
	"core-backend/internal/domain/enum"
	"encoding/json"
)

// region: Financial Tersm structure, this is the struct used for the request body

// FinancialTerms defines the financial terms for different contract types
// This is a general purpose struct used in the requst body, and will be unmarshalled to specific struct based on the type
// To see the individual structures for each contract type, refers to the structs in the responses package
// [dtos.AdvertisingFinancialTerms] for ADVERTISEMENT and BRAND_AMBASSADOR type
// [dtos.AffiliateFinancialTerms] for AFFILIATE type
// [dtos.CoProducingFinancialTerms] for CO_PRODUCING type
type FinancialTerms struct {
	// Common fields
	Model string `json:"model,omitempty" validate:"oneof=FIXED LEVELS SHARE" example:"FIXED"` // FIXED, LEVELS, or SHARE

	// If the model is "FIXED" and the CONTRACT_TYPE is "ADVERTISING" or "BRAND_AMBASSADOR"
	PaymentMethod *string         `json:"payment_method,omitempty" example:"BANK_TRANSFER" validate:"omitempty,oneof=BANK_TRANSFER CREDIT_CARD"` // Default to BANK_TRANSFER
	TotalCost     *int            `json:"total_cost,omitempty" example:"10000000" validate:"omitempty,min=0"`
	CostBreakdown *map[string]int `json:"cost_breakdown,omitempty" validate:"omitempty,dive,min=0"`
	Schedule      *[]Schedule     `json:"schedule,omitempty" validate:"omitempty,dive"`

	// If the model is "LEVELS" and the CONTRACT_TYPE is "AFFILIATE"
	BasePerClick   *int               `json:"base_per_click,omitempty" example:"1000" validate:"omitempty,min=0"`
	Levels         *[]Level           `json:"levels,omitempty" validate:"omitempty,dive"`
	TaxWithholding *TaxWithholding    `json:"tax_withholding,omitempty" validate:"omitempty"`
	PaymentCycle   *enum.PaymentCycle `json:"payment_cycle,omitempty" example:"MONTHLY" validate:"omitempty,oneof=MONTHLY QUARTERLY ANNUALLY"`
	//	@swaggertype	object
	PaymentDate *any `json:"payment_date,omitempty" example:"Change this depending on PaymentCycle" validate:"omitempty"` // Change this depending on PaymentCycle

	// If the model is "SHARE" and the CONTRACT_TYPE is "CO_PRODUCING"
	CompanyPercent          *int               `json:"profit_split_company_percent,omitempty" example:"60" validate:"omitempty,min=0,max=100"`
	KolPercent              *int               `json:"profit_split_kol_percent,omitempty" example:"40" validate:"omitempty,min=0,max=100"`
	ProfitDistributionCycle *enum.PaymentCycle `json:"profit_distribution_cycle,omitempty" example:"QUARTERLY" validate:"omitempty,oneof=MONTHLY QUARTERLY ANNUALLY"`
	//	@swaggertype	object
	ProfitDistributionDate *any `json:"profit_distribution_date,omitempty" example:"2023-12-31" validate:"omitempty"` // Change this depending on ProfitDistributionCycle
}

// endregion

// region : ================ Financial Terms for different contract types =================

// AdvertisingFinancialTerms contains financial details specific to the ADVERTISEMENT and BRAND_AMBASSADOR contract types
// The Schedule field is an array of Schedule struct, each representing a milestone payment schedule, and will be used to
// generate each [model.ContractPayment] record
type AdvertisingFinancialTerms struct {
	Model         string         `json:"model" example:"FIXED" validate:"oneof=FIXED"`
	PaymentMethod string         `json:"payment_method" example:"BANK_TRANSFER" validate:"oneof=BANK_TRANSFER CREDIT_CARD"`
	TotalCost     int            `json:"total_cost" example:"10000000" validate:"min=0"`
	CostBreakdown map[string]int `json:"cost_breakdown" validate:"dive,keys,max=255,endkeys,min=0"`
	Schedules     []Schedule     `json:"schedule" validate:"dive"`
}

// AffiliateFinancialTerms contains financial details specific to the AFFILIATE contract type
// The PaymentDate field can be of different types based on the PaymentCycle:
// - For "MONTHLY" cycle, it can be a string representing the day of the month (e.g., "5" for the 5th day), or "start" for the first day of the month, or "end" for the last day of the month.
// - For "QUARTERLY" cycle, it can be a array contains 4 PaymentDate struct representing specific dates (day, month, year) for each quarter.
// - For "ANNUALLY" cycle, it can be a time.Time object representing the exact date of payment.
type AffiliateFinancialTerms struct {
	Model          string            `json:"model" example:"COMMISSION" validate:"oneof=LEVELS"`
	BasePerClick   int               `json:"base_per_click" example:"1000" validate:"min=0"`
	Levels         []Level           `json:"levels" validate:"dive"`
	PaymentCycle   enum.PaymentCycle `json:"payment_cycle" example:"MONTHLY" validate:"oneof=MONTHLY QUARTERLY ANNUALLY"`
	PaymentDate    any               `json:"payment_date" example:"2023-11-05" validate:""` // Change this depending on PaymentCycle
	TaxWithholding TaxWithholding    `json:"tax_withholding" validate:"required"`
}

// CoProducingFinancialTerms contains financial details specific to the CO_PRODUCING contract type
// The ProfittDistributionDate field can be of different types based on the ProfitDistributionCycle:
// - For "MONTHLY" cycle, it can be a string representing the day of the month (e.g., "5" for the 5th day), or "start" for the first day of the month, or "end" for the last day of the month.
// - For "QUARTERLY" cycle, it can be a array contains 4 PaymentDate struct representing specific dates (day, month, year) for each quarter.
// - For "ANNUALLY" cycle, it can be a time.Time object representing the exact date of profit distribution.
type CoProducingFinancialTerms struct {
	Model                   string            `json:"model" example:"PROFIT_SHARING" validate:"oneof=SHARE"`
	CompanyPercent          int               `json:"profit_split_company_percent" example:"60" validate:"min=0,max=100"`
	KolPercent              int               `json:"profit_split_kol_percent" example:"40" validate:"min=0,max=100"`
	ProfitDistributionCycle enum.PaymentCycle `json:"profit_distribution_cycle" example:"QUARTERLY" validate:"oneof=MONTHLY QUARTERLY ANNUALLY"`
	ProfitDistributionDate  any               `json:"profit_distribution_date" example:"2023-12-31" validate:""` // Change this depending on ProfitDistributionCycle
}

// endregion

// region: ================ Sub-structures for Financial Terms =================

// Schedule struct represents a milestone payment schedule used in AdvertisingFinancialTerms
type Schedule struct {
	ID        *int8  `json:"id" example:"1" validate:"omitempty,gt=0"`
	Milestone string `json:"milestone" example:"Initial payment" validate:"required,max=255"`
	Percent   int    `json:"percent" example:"30" validate:"required,min=0,max=100"`
	Amount    int    `json:"amount" example:"3000000" validate:"required,min=0"`
	DueDate   string `json:"due_date" example:"2023-10-15" validate:"required,datetime=2006-01-02"`
}

// PaymentDate struct is used to represent specific dates for quarterly profit distribution or payment dates in the AffiliateFinancialTerms and CoProducingFinancialTerms structs
type PaymentDate struct {
	ID    int8  `json:"id" validate:"omitempty,min=0,max=4"`
	Day   int8  `json:"day" validate:"omitempty,min=1,max=31"`
	Month int8  `json:"month" validate:"omitempty,min=1,max=12"`
	Year  int16 `json:"year" validate:"omitempty,min=1000,max=2999"`
}

// Level struct represents a payment level used in AffiliateFinancialTerms
// If the total clicks exceed the MaxClicks for a level, the excess clicks will be charged at the next level's Multiplier rate
type Level struct {
	Level      int     `json:"level" example:"1" validate:"required,min=1"`
	MaxClicks  int64   `json:"max_clicks" example:"1000" validate:"required,min=0"`
	Multiplier float32 `json:"multiplier" example:"1.0" validate:"required,gte=1"`
}

// TaxWithholding struct represents the tax withholding details used in AffiliateFinancialTerms
// The Threshold field is the minimum earnings before tax withholding applies
// The RatePercent field is the percentage of tax to be withheld from earnings above the threshold
type TaxWithholding struct {
	Threshold   int `json:"threshold" example:"10000000" validate:"min=0"`
	RatePercent int `json:"rate_percent" example:"10" validate:"min=0,max=100"`
}

// endregion

// region : ================ Helper Conversion Methods =================

// ConvertToAdvertisingFinancialTerms converts the general FinancialTerms struct to AdvertisingFinancialTerms struct through JSON marshal/unmarshal
func (f *FinancialTerms) ConvertToAdvertisingFinancialTerms() (*AdvertisingFinancialTerms, error) {
	rawFinancialTerms, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	var advertisingTerms AdvertisingFinancialTerms
	err = json.Unmarshal(rawFinancialTerms, &advertisingTerms)
	if err != nil {
		return nil, err
	}

	return &advertisingTerms, nil
}

// ConvertToAffiliateFinancialTerms converts the general FinancialTerms struct to AffiliateFinancialTerms struct through JSON marshal/unmarshal
func (f *FinancialTerms) ConvertToAffiliateFinancialTerms() (*AffiliateFinancialTerms, error) {
	rawFinancialTerms, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	var affiliateTerms AffiliateFinancialTerms
	err = json.Unmarshal(rawFinancialTerms, &affiliateTerms)
	if err != nil {
		return nil, err
	}

	return &affiliateTerms, nil
}

// ConvertToCoProducingFinancialTerms converts the general FinancialTerms struct to CoProducingFinancialTerms struct through JSON marshal/unmarshal
func (f *FinancialTerms) ConvertToCoProducingFinancialTerms() (*CoProducingFinancialTerms, error) {
	rawFinancialTerms, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	var coProducingTerms CoProducingFinancialTerms
	err = json.Unmarshal(rawFinancialTerms, &coProducingTerms)
	if err != nil {
		return nil, err
	}

	return &coProducingTerms, nil
}

// endregion
