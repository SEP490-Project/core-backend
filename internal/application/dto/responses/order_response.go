package responses

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// OrderResponse is a sanitized DTO for returning orders to API clients
type OrderResponse struct {
	ID             uuid.UUID                   `json:"id"`
	UserID         uuid.UUID                   `json:"user_id"`
	Status         string                      `json:"status"`
	TotalAmount    float64                     `json:"total_amount"`
	FullName       string                      `json:"full_name"`
	PhoneNumber    string                      `json:"phone_number"`
	Email          string                      `json:"email"`
	Street         string                      `json:"street"`
	City           string                      `json:"city"`
	ShippingFee    int                         `json:"shipping_fee"`
	CreatedAt      time.Time                   `json:"created_at"`
	UpdatedAt      time.Time                   `json:"updated_at"`
	IsSelfPickedUp bool                        `json:"is_self_picked_up"`
	OrderType      string                      `json:"order_type"`
	GHNOrderCode   *string                     `json:"ghn_order_code,omitempty"`
	Payment        *PaymentTransactionResponse `json:"payment_transaction,omitempty"`
	OrderItems     []model.OrderItem           `json:"order_items,omitempty"`
}

func (OrderResponse) ToResponse(o *model.Order, pt *model.PaymentTransaction) *OrderResponse {
	if o == nil {
		return nil
	}
	resp := &OrderResponse{
		ID:             o.ID,
		UserID:         o.UserID,
		Status:         string(o.Status),
		TotalAmount:    o.TotalAmount,
		FullName:       o.FullName,
		PhoneNumber:    o.PhoneNumber,
		Email:          o.Email,
		Street:         o.Street,
		City:           o.City,
		ShippingFee:    o.ShippingFee,
		CreatedAt:      o.CreatedAt,
		UpdatedAt:      o.UpdatedAt,
		IsSelfPickedUp: o.IsSelfPickedUp,
		OrderType:      o.OrderType,
		GHNOrderCode:   o.GHNOrderCode,
		OrderItems:     o.OrderItems,
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
