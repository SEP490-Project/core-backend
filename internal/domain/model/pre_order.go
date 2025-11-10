package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"gorm.io/datatypes"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PreOrder struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	VariantID   uuid.UUID `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	Quantity    int       `json:"quantity" gorm:"column:quantity;not null"`
	UnitPrice   float64   `json:"unit_price" gorm:"column:unit_price;not null"`
	TotalAmount float64   `json:"total_amount" gorm:"column:total_amount;not null"`

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

	Status    enum.PreOrderStatus `json:"status" gorm:"column:status;not null;check:status in ('PENDING', 'PRE_ORDERED', 'AWAITING_RELEASE', 'AWAITING_PICKUP', 'CONFIRMED', 'CANCELLED', 'IN_TRANSIT', 'DELIVERED', 'RECEIVED')"`
	CreatedAt time.Time           `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time           `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	//DeletedAt gorm.DeletedAt      `json:"deleted_at" gorm:"column:deleted_at"swaggerignore:"true"`

	// Relationships
	User           *User           `json:"-" gorm:"foreignKey:UserID"`
	ProductVariant *ProductVariant `json:"-" gorm:"foreignKey:VariantID"`

	// Transient fields populated by repository (not persisted)
	PaymentID  *uuid.UUID `json:"payment_id,omitempty" gorm:"-"`
	PaymentBin *string    `json:"payment_bin,omitempty" gorm:"-"`
}

func (PreOrder) TableName() string { return "pre_orders" }

func (po *PreOrder) BeforeCreate(tx *gorm.DB) (err error) {
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
