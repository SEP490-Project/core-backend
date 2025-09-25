package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Contract struct {
	ID              uuid.UUID           `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	BrandID         uuid.UUID           `json:"brand_id" gorm:"type:uuid;column:brand_id;not null"`
	StaffID         uuid.UUID           `json:"staff_id" gorm:"type:uuid;column:staff_id;not null"`
	Title           string              `json:"title" gorm:"type:varchar(255);column:title;not null"`
	Type            enum.ContractType   `json:"type" gorm:"type:varchar(50);column:type;not null"`
	StartDate       time.Time           `json:"start_date" gorm:"type:timestamp;column:start_date;not null"`
	EndDate         time.Time           `json:"end_date" gorm:"type:timestamp;column:end_date;not null"`
	Status          enum.ContractStatus `json:"status" gorm:"type:varchar(50);column:status;not null"`
	ContractFileURL string              `json:"contract_file_url" gorm:"type:text;column:contract_file_url;not null"`
	ProposalFileURL *string             `json:"proposal_file_url" gorm:"type:text;column:proposal_file_url"`

	CompensationAmount *float64 `json:"compensation_amount" gorm:"type:numeric(12,2);column:compensation_amount"`
	PaymentTerms       *string  `json:"payment_terms" gorm:"type:text;column:payment_terms"`

	// Advertising specific fields (used when Type is Advertising)
	Deliverables datatypes.JSON `json:"deliverables" gorm:"type:jsonb;column:deliverables"`
	UsageRights  *string        `json:"usage_rights" gorm:"type:text;column:usage_rights"`

	// Affiliate specific fields (used when Type is Affiliate)
	CommissionRate     float64 `json:"commission_rate" gorm:"type:numeric(5,2);column:commission_rate"`
	CookieDurationDays int     `json:"cookie_duration_days" gorm:"type:int;column:cookie_duration_days"`
	PayoutThreshold    float64 `json:"payout_threshold" gorm:"type:numeric(12,2);column:payout_threshold"`

	// Brand Ambassador specific fields (used when Type is BrandAmbassador)
	MonthlyRetainer       float64 `json:"monthly_retainer" gorm:"type:numeric(12,2);column:monthly_retainer"`
	RequiredPostsPerMonth int     `json:"required_posts_per_month" gorm:"type:int;column:required_posts_per_month"`

	// Co-Producing specific fields (used when Type is CoProducing)
	RevenueSharePercentage float64 `json:"revenue_share_percentage" gorm:"type:numeric(5,2);column:revenue_share_percentage"`
	IPOwnership            string  `json:"ip_ownership" gorm:"type:text;column:ip_ownership"`

	ParentContractID *uuid.UUID `json:"parent_contract_id" gorm:"type:uuid;column:parent_contract_id"`

	CreatedAt time.Time      `json:"created_at" gorm:"type:timestamp;column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"type:timestamp;column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at"`

	// Relationships (always use struct field names, not column names)
	ParentContract *Contract  `json:"-" gorm:"foreignKey:ParentContractID"`
	ChildContracts []Contract `json:"-" gorm:"foreignKey:ParentContractID"`
	Brand          *Brand     `json:"-" gorm:"foreignKey:BrandID"`
	Staff          *User      `json:"-" gorm:"foreignKey:StaffID"`
	Campaign       *Campaign  `json:"-" gorm:"foreignKey:ContractID"`
}

func (Contract) TableName() string { return "contracts" }

func (c *Contract) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.CompensationAmount == nil {
		c.CompensationAmount = new(float64)
	}
	if c.PaymentTerms == nil {
		c.PaymentTerms = new(string)
	}
	return nil
}
