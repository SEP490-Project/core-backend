package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Contract struct {
	ID               uuid.UUID  `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ParentContractID *uuid.UUID `json:"parent_contract_id" gorm:"type:uuid;column:parent_contract_id"`
	Title            *string    `json:"title" gorm:"type:varchar(255);column:title"`
	ContractNumber   *string    `json:"contract_number" gorm:"type:varchar(255);column:contract_number;not null;unique"`

	// Brand information
	BrandID                *uuid.UUID `json:"brand_id" gorm:"type:uuid;column:brand_id"`
	BrandBankName          *string    `json:"brand_bank_name" gorm:"type:varchar(255);column:brand_bank_name"`
	BrandBankAccountNumber *string    `json:"brand_account_number" gorm:"type:varchar(255);column:brand_account_number"`
	BrandBankAccountHolder *string    `json:"brand_account_holder" gorm:"type:varchar(100);column:brand_account_holder"`

	// KOL Representative information
	RepresentativeName              *string `json:"representative_name" gorm:"type:varchar(255);column:representative_name"`
	RepresentativeRole              *string `json:"representative_role" gorm:"type:varchar(255);column:representative_role"`
	RepresentativePhone             *string `json:"representative_phone" gorm:"type:varchar(20);column:representative_phone"`
	RepresentativeEmail             *string `json:"representative_email" gorm:"type:varchar(255);column:representative_email"`
	RepresentativeTaxNumber         *string `json:"representative_tax_number" gorm:"type:varchar(100);column:representative_tax_number"`
	RepresentativeBankName          *string `json:"representative_bank_name" gorm:"type:varchar(255);column:representative_bank_name"`
	RepresentativeBankAccountNumber *string `json:"representative_bank_account_number" gorm:"type:varchar(255);column:representative_bank_account_number"`
	RepresentativeBankAccountHolder *string `json:"representative_bank_account_holder" gorm:"type:varchar(255);column:representative_bank_account_holder"`

	// Contract complex details stored as JSONB
	FinancialTerms datatypes.JSON `json:"financial_terms" gorm:"type:jsonb;column:financial_terms;not null"`
	ScopeOfWork    datatypes.JSON `json:"scope_of_work" gorm:"type:jsonb;column:scope_of_work;not null"`
	LegalTerms     datatypes.JSON `json:"legal_terms" gorm:"type:jsonb;column:legal_terms;not null"`

	// Contract Details
	Type            enum.ContractType   `json:"type" gorm:"type:varchar(50);column:type;not null;check:type IN ('ADVERTISING', 'AFFILIATE', 'BRAND_AMBASSADOR', 'CO_PRODUCING')"`
	Status          enum.ContractStatus `json:"status" gorm:"type:varchar(50);column:status;not null;check:status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'TERMINATED')"`
	DepositPercent  *int                `json:"deposit_percent" gorm:"type:int;column:deposit_percent;check:deposit_percent >= 0 AND deposit_percent <= 100"`
	DepositAmount   *int                `json:"deposit_amount" gorm:"type:int;column:deposit_amount;check:deposit_amount >= 0"`
	IsDepositPaid   *bool               `json:"is_deposit_paid" gorm:"type:boolean;column:is_deposit_paid;default:false"`
	SignedDate      time.Time           `json:"signed_date" gorm:"column:signed_date;not null"`
	SignedLocation  *string             `json:"signed_location" gorm:"type:varchar(255);column:signed_location"`
	StartDate       time.Time           `json:"start_date" gorm:"column:start_date;not null"`
	EndDate         time.Time           `json:"end_date" gorm:"column:end_date;not null"`
	Currency        *string             `json:"currency" gorm:"type:varchar(3);column:currency;default:'VND'"`
	ContractFileURL *string             `json:"contract_file_url" gorm:"type:text;column:contract_file_url"`
	ProposalFileURL *string             `json:"proposal_file_url" gorm:"type:text;column:proposal_file_url"`
	CreatedAt       time.Time           `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time           `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt       gorm.DeletedAt      `json:"deleted_at" gorm:"column:deleted_at;index"`
	CreatedByID     uuid.UUID           `json:"created_by" gorm:"type:uuid;column:created_by;not null"`
	UpdatedByID     *uuid.UUID          `json:"updated_by" gorm:"type:uuid;column:updated_by"`

	// Relationships
	ParentContract    *Contract         `json:"parent_contract" gorm:"foreignKey:ParentContractID;references:ID"`
	ChildrenContracts []Contract        `json:"children_contracts" gorm:"foreignKey:ParentContractID;references:ID"`
	Brand             *Brand            `json:"brand" gorm:"foreignKey:BrandID;references:ID"`
	Campaign          *Campaign         `json:"campaigns" gorm:"foreignKey:ContractID;references:ID"`
	ContractPayments  []ContractPayment `json:"contract_payments" gorm:"foreignKey:ContractID;references:ID"`
}

func (Contract) TableName() string { return "contracts" }

func (c *Contract) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
