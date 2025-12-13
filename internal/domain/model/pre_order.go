package model

import (
	"core-backend/internal/domain/enum"
	"database/sql/driver"
	"encoding/json"
	"time"

	"gorm.io/datatypes"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PreOrderActionNote mirrors OrderActionNote for pre-orders
type PreOrderActionNote struct {
	UserID     uuid.UUID           `json:"user_id"`
	UserName   string              `json:"user_name"`
	UserEmail  string              `json:"user_email"`
	ActionType enum.PreOrderStatus `json:"action_type"`
	Reason     string              `json:"reason"`
	CreatedAt  time.Time           `json:"created_at"`
}

// PreOrderActionNotes wrapper type for JSONB handling
type PreOrderActionNotes []PreOrderActionNote

// Value implements driver.Valuer interface for JSONB storage
func (notes PreOrderActionNotes) Value() (driver.Value, error) {
	if notes == nil {
		return nil, nil
	}
	return json.Marshal(notes)
}

// Scan implements sql.Scanner interface for JSONB retrieval
func (notes *PreOrderActionNotes) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, notes)
}

type PreOrder struct {
	ID          uuid.UUID           `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID           `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	VariantID   uuid.UUID           `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	Quantity    int                 `json:"quantity" gorm:"column:quantity;not null"`
	UnitPrice   float64             `json:"unit_price" gorm:"column:unit_price;not null"`
	TotalAmount float64             `json:"total_amount" gorm:"column:total_amount;not null"`
	Status      enum.PreOrderStatus `json:"status" gorm:"column:status;not null;check:status in ('PENDING', 'PRE_ORDERED', 'AWAITING_RELEASE', 'AWAITING_PICKUP', 'CONFIRMED', 'CANCELLED', 'IN_TRANSIT', 'DELIVERED', 'RECEIVED')"`

	// Bank Info
	BankAccount       string `json:"user_bank_account" gorm:"column:user_bank_account;not null"`
	BankName          string `json:"user_bank_name" gorm:"column:user_bank_name;not null"`
	BankAccountHolder string `json:"user_bank_account_holder" gorm:"column:user_bank_account_holder;not null"`

	//The same as order which Copied shipping address fields
	FullName      string `json:"full_name" gorm:"column:full_name"`
	PhoneNumber   string `json:"phone_number" gorm:"column:phone_number"`
	Email         string `json:"email" gorm:"column:email"`
	Street        string `json:"street" gorm:"column:street"`
	AddressLine2  string `json:"address_line2" gorm:"column:address_line2"`
	City          string `json:"city" gorm:"column:city"`
	GhnProvinceID int    `json:"ghn_province_id" gorm:"column:ghn_province_id"`
	GhnDistrictID int    `json:"ghn_district_id" gorm:"column:ghn_district_id"`
	GhnWardCode   string `json:"ghn_ward_code" gorm:"column:ghn_ward_code"`
	ProvinceName  string `json:"province_name" gorm:"column:province_name"`
	DistrictName  string `json:"district_name" gorm:"column:district_name"`
	WardName      string `json:"ward_name" gorm:"column:ward_name"`

	IsSelfPickedUp    bool           `json:"is_self_picked_up" gorm:"column:is_self_picked_up;not null;default:false"`
	ConfirmationImage *string        `json:"confirmation_image,omitempty" gorm:"column:confirmation_image;type:text"`
	CreatedAt         time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt         gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at" swaggertype:"string"`
	UserResource      *string        `json:"user_resource,omitempty" gorm:"column:user_resource;type:text"`
	StaffResource     *string        `json:"staff_resource,omitempty" gorm:"column:staff_resource;type:text"`

	ActionNotes *PreOrderActionNotes `json:"action_notes,omitempty" gorm:"column:action_notes;type:jsonb"`
	UserNote    *string              `json:"user_note,omitempty" gorm:"column:user_note;type:text"`

	//The same as orderItem
	Capacity              *float64            `json:"capacity" gorm:"column:capacity"`
	CapacityUnit          *string             `json:"capacity_unit" gorm:"column:capacity_unit"`
	ContainerType         *enum.ContainerType `json:"container_type" gorm:"type:varchar(255);column:container_type;check:container_type in ('BOTTLE', 'TUBE', 'JAR', 'STICK', 'PENCIL', 'COMPACT', 'PALLETE', 'SACHET', 'VIAL', 'ROLLER_BOTTLE')"`
	DispenserType         *enum.DispenserType `json:"dispenser_type" gorm:"type:varchar(255);column:dispenser_type;check:dispenser_type in ('PUMP', 'SPRAY', 'DROPPER', 'ROLL_ON', 'TWIST_UP', 'SQUEEZE', 'NONE')"`
	Uses                  *string             `json:"uses" gorm:"type:text;column:uses"`
	ManufactureDate       *time.Time          `json:"manufacturing_date" gorm:"column:manufacturing_date"`
	ExpiryDate            *time.Time          `json:"expiry_date" gorm:"column:expiry_date"`
	Instructions          *string             `json:"instructions" gorm:"type:text;column:instructions"`
	AttributesDescription *datatypes.JSON     `json:"attributes_description" gorm:"column:attributes_description;type:jsonb" swaggerignore:"true"`
	Weight                int                 `json:"weight" gorm:"column:weight"` // in grams
	Height                int                 `json:"height" gorm:"column:height"` // in centimeters
	Length                int                 `json:"length" gorm:"column:length"` // in centimeters
	Width                 int                 `json:"width" gorm:"column:width"`   // in centimeters

	//product fields
	ProductName string  `json:"product_name" gorm:"column:product_name;not null"`
	Description *string `json:"description" gorm:"column:description"`
	Type        string  `json:"product_type" gorm:"column:product_type;not null"`
	IsReviewed  bool    `json:"is_reviewed" gorm:"column:is_review;default:false"`

	BrandID    *uuid.UUID `json:"brand_id" gorm:"column:brand_id;"`
	CategoryID uuid.UUID  `json:"category_id" gorm:"column:category_id;not null"`

	// Relationships
	User           *User            `json:"-" gorm:"foreignKey:UserID"`
	ProductVariant *ProductVariant  `json:"-" gorm:"foreignKey:VariantID"`
	Brand          *Brand           `json:"brand" gorm:"foreignKey:BrandID" swaggerignore:"true"`
	Category       *ProductCategory `json:"category" gorm:"foreignKey:CategoryID"`
	ProductReview  *ProductReview   `json:"review" gorm:"foreignKey:ID" swaggerignore:"true"`

	// Transient fields populated by repository (not persisted)
	PaymentID  *uuid.UUID `json:"payment_id,omitempty" gorm:"-"`
	PaymentBin *string    `json:"payment_bin,omitempty" gorm:"-"`
}

func (PreOrder) TableName() string { return "pre_orders" }

func (po *PreOrder) BeforeCreate(tx *gorm.DB) (err error) {
	_ = tx
	if po.ID == uuid.Nil {
		po.ID = uuid.New()
	}
	if po.Quantity < 0 {
		zap.L().Warn("Quantity is less than 0, setting to 0")
		po.Quantity = 0
	}
	if po.UnitPrice < 0 {
		zap.L().Warn("UnitPrice is less than 0, setting to 0")
		po.UnitPrice = 0
	}
	if po.TotalAmount < 0 {
		zap.L().Warn("TotalAmount is less than 0, setting to 0")
		po.TotalAmount = 0
	}

	return nil
}

// AddActionNote appends a PreOrderActionNote to the PreOrder's ActionNotes
func (po *PreOrder) AddActionNote(note PreOrderActionNote) {
	if po.ActionNotes == nil {
		notes := make(PreOrderActionNotes, 0)
		po.ActionNotes = &notes
	}

	if note.CreatedAt.IsZero() {
		note.CreatedAt = time.Now()
	}

	*po.ActionNotes = append(*po.ActionNotes, note)
}

// GetLatestActionNote returns the latest action note or nil if none exists
func (po *PreOrder) GetLatestActionNote() *PreOrderActionNote {
	if po.ActionNotes == nil || len(*po.ActionNotes) == 0 {
		return nil
	}
	notes := *po.ActionNotes
	return &notes[len(notes)-1]
}
