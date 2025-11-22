package responses

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// OrderResponse is a sanitized DTO for returning orders to API clients
type OrderResponse struct {
	ID                uuid.UUID                   `json:"id"`
	UserID            uuid.UUID                   `json:"user_id"`
	Status            string                      `json:"status"`
	TotalAmount       float64                     `json:"total_amount"`
	FullName          string                      `json:"full_name"`
	PhoneNumber       string                      `json:"phone_number"`
	Email             string                      `json:"email"`
	Street            string                      `json:"street"`
	AddressLine2      string                      `json:"address_line2"`
	City              string                      `json:"city"`
	GhnProvinceID     int                         `json:"ghn_province_id" gorm:"column:ghn_province_id"`
	GhnDistrictID     int                         `json:"ghn_district_id" gorm:"column:ghn_district_id"`
	GhnWardCode       string                      `json:"ghn_ward_code" gorm:"column:ghn_ward_code"`
	ProvinceName      string                      `json:"province_name" gorm:"column:province_name"`
	DistrictName      string                      `json:"district_name" gorm:"column:district_name"`
	WardName          string                      `json:"ward_name" gorm:"column:ward_name"`
	ShippingFee       int                         `json:"shipping_fee"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	IsSelfPickedUp    bool                        `json:"is_self_picked_up"`
	ConfirmationImage *string                     `json:"confirmation_image"`
	UserResource      *string                     `json:"user_resource,omitempty"`
	StaffResource     *string                     `json:"staff_resource,omitempty"`
	OrderType         string                      `json:"order_type"`
	GHNOrderCode      *string                     `json:"ghn_order_code,omitempty"`
	ActionNotes       *model.OrderActionNotes     `json:"action_notes,omitempty"`
	UserNote          *string                     `json:"user_note,omitempty"`
	Payment           *PaymentTransactionResponse `json:"payment_transaction,omitempty"`
	OrderItems        []model.OrderItem           `json:"order_items,omitempty"`
}

func (OrderResponse) ToResponse(o *model.Order, pt *model.PaymentTransaction) *OrderResponse {
	if o == nil {
		return nil
	}
	resp := &OrderResponse{
		ID:                o.ID,
		UserID:            o.UserID,
		Status:            string(o.Status),
		TotalAmount:       o.TotalAmount,
		FullName:          o.FullName,
		PhoneNumber:       o.PhoneNumber,
		Email:             o.Email,
		Street:            o.Street,
		AddressLine2:      o.AddressLine2,
		City:              o.City,
		GhnProvinceID:     o.GhnProvinceID,
		GhnDistrictID:     o.GhnDistrictID,
		GhnWardCode:       o.GhnWardCode,
		ProvinceName:      o.ProvinceName,
		DistrictName:      o.DistrictName,
		WardName:          o.WardName,
		ShippingFee:       o.ShippingFee,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
		IsSelfPickedUp:    o.IsSelfPickedUp,
		ConfirmationImage: o.ConfirmationImage,
		UserResource:      o.UserResource,
		StaffResource:     o.StaffResource,
		ActionNotes:       o.ActionNotes,
		UserNote:          o.UserNote,
		OrderType:         o.OrderType,
		GHNOrderCode:      o.GHNOrderCode,
		OrderItems:        o.OrderItems,
	}
	if pt != nil {
		resp.Payment = PaymentTransactionResponse{}.ToResponse(pt)
	}
	return resp
}

func (OrderResponse) ToResponseList(source []model.Order, payments map[uuid.UUID]model.PaymentTransaction) []OrderResponse {
	if len(source) == 0 {
		return []OrderResponse{}
	}
	res := make([]OrderResponse, 0, len(source))
	for _, o := range source {
		var pt *model.PaymentTransaction
		if p, ok := payments[o.ID]; ok {
			pt = &p
		}
		res = append(res, *OrderResponse{}.ToResponse(&o, pt))
	}
	return res
}
