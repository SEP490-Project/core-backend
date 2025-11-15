package model

import (
	"core-backend/internal/domain/enum"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrderActionNote struct {
	UserID     uuid.UUID        `json:"user_id"`
	UserName   string           `json:"user_name"`
	UserEmail  string           `json:"user_email"`
	ActionType enum.OrderStatus `json:"action_type"`
	Reason     string           `json:"reason"`
	CreatedAt  time.Time        `json:"created_at"`
}

// OrderActionNotes wrapper type for JSONB handling
type OrderActionNotes []OrderActionNote

// Value implements driver.Valuer interface for JSONB storage
func (notes OrderActionNotes) Value() (driver.Value, error) {
	if notes == nil {
		return nil, nil
	}
	return json.Marshal(notes)
}

// Scan implements sql.Scanner interface for JSONB retrieval
func (notes *OrderActionNotes) Scan(value any) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, notes)
}

type Order struct {
	ID          uuid.UUID        `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	UserID      uuid.UUID        `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	Status      enum.OrderStatus `json:"status" gorm:"column:status;not null"`
	TotalAmount float64          `json:"total_amount" gorm:"column:total_amount;not null"`

	// Copied shipping address fields (migration moved from a foreign key to flat columns)
	FullName          string    `json:"full_name" gorm:"column:full_name"`
	PhoneNumber       string    `json:"phone_number" gorm:"column:phone_number"`
	Email             string    `json:"email" gorm:"column:email"`
	Street            string    `json:"street" gorm:"column:street"`
	AddressLine2      string    `json:"address_line2" gorm:"column:address_line2"`
	City              string    `json:"city" gorm:"column:city"`
	GhnProvinceID     int       `json:"ghn_province_id" gorm:"column:ghn_province_id"`
	GhnDistrictID     int       `json:"ghn_district_id" gorm:"column:ghn_district_id"`
	GhnWardCode       string    `json:"ghn_ward_code" gorm:"column:ghn_ward_code"`
	ProvinceName      string    `json:"province_name" gorm:"column:province_name"`
	DistrictName      string    `json:"district_name" gorm:"column:district_name"`
	WardName          string    `json:"ward_name" gorm:"column:ward_name"`
	ShippingFee       int       `json:"shipping_fee" gorm:"column:shipping_fee;default:0"`
	CreatedAt         time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	IsSelfPickedUp    bool      `json:"is_self_picked_up" gorm:"column:is_self_picked_up;not null;default:false"`
	ConfirmationImage *string   `json:"confirmation_image,omitempty" gorm:"column:confirmation_image;type:text"`
	OrderType         string    `json:"order_type" gorm:"column:order_type;type:text;default:'STANDARD'"`

	ActionNotes  *OrderActionNotes `json:"action_notes,omitempty" gorm:"column:action_notes;type:jsonb"`
	UserNote     *string           `json:"user_note,omitempty" gorm:"column:user_note;type:text"`
	GHNOrderCode *string           `json:"ghn_order_code" gorm:"column:ghn_order_code;type:text"`
	// Relationships
	User       User        `json:"-" gorm:"foreignKey:UserID"`
	OrderItems []OrderItem `json:"order_items" gorm:"foreignKey:OrderID"`

	// Transient fields populated by repository (not persisted)
	PaymentID  *uuid.UUID `json:"payment_id,omitempty" gorm:"-"`
	PaymentBin *string    `json:"payment_bin,omitempty" gorm:"-"`
}

func (Order) TableName() string { return "orders" }

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	_ = tx
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.TotalAmount < 0 {
		zap.L().Warn("TotalAmount is less than 0, setting to 0")
		o.TotalAmount = 0
	}

	return nil
}

func (o *Order) AddActionNote(note OrderActionNote) {
	if o.ActionNotes == nil {
		notes := make(OrderActionNotes, 0)
		o.ActionNotes = &notes
	}

	if note.CreatedAt.IsZero() {
		note.CreatedAt = time.Now()
	}

	*o.ActionNotes = append(*o.ActionNotes, note)
}
