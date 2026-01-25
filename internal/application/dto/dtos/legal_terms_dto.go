package dtos

// region: ================= Legal Terms =================

// LegalTerms represents the legal terms and conditions associated with a contract.
type LegalTerms struct {
	BreachOfContract BreachOfContract `json:"breach_of_contract" validate:"required"`
	StandardTerms    StandardTerm     `json:"standard_terms" validate:"required"`
}

// endregion

// region: ================= Sub-structures for Legal Terms =================

// BreachOfContract represents the breach of contract section of the legal terms.
type BreachOfContract struct {
	Label string                 `json:"label" example:"Breach of Contract" validate:"required,min=1,max=1024"`
	Items []BreachOfContractItem `json:"items" validate:"required,dive"`
}

// BreachOfContractItem represents an individual item under the breach of contract section.
type BreachOfContractItem struct {
	Title               string   `json:"title" example:"Party A (The Brand) has breached its contract with Party B (The KOL)" validate:"required,min=1,max=1024"`
	Details             []string `json:"details,omitempty" example:"['The contract will be terminated immediately.', 'Party A will loses (forfeits) the money they have paid to Party B.']" validate:"omitempty,dive,min=1,max=2048"`
	CompensationPercent *int     `json:"compensation_percent,omitempty" example:"50" validate:"omitempty,min=0,max=100"`
}

// StandardTerm represents a standard term item in the legal terms.
type StandardTerm struct {
	Label string     `json:"label" example:"Standard Terms" validate:"required,min=1,max=1024"`
	Items []TermItem `json:"items" validate:"required,dive"`
}

// TermItem represents an individual standard term item.
type TermItem struct {
	Title       string  `json:"title" example:"Term 1: Confidentiality" validate:"required,min=1,max=1024"`
	Description *string `json:"description,omitempty" example:"Both parties agree to keep all information related to this contract confidential." validate:"omitempty,min=1,max=2048"`
}

// endregion
