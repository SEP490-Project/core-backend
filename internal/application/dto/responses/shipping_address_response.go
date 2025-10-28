package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// ShippingAddressResponse represents shipping address information in responses
type ShippingAddressResponse struct {
	ID           string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string  `json:"user_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type         string  `json:"type" example:"SHIPPING"`
	FullName     string  `json:"full_name" example:"John Doe"`
	PhoneNumber  string  `json:"phone_number" example:"+1234567890"`
	Email        string  `json:"email" example:"john@example.com"`
	Street       string  `json:"street" example:"123 Main St"`
	AddressLine2 *string `json:"address_line_2" example:"Apt 4B"`
	City         string  `json:"city" example:"New York"`
	State        *string `json:"state" example:"NY"`
	PostalCode   string  `json:"postal_code" example:"10001"`
	Country      string  `json:"country" example:"USA"`
	Company      *string `json:"company" example:"Acme Corp"`
	IsDefault    bool    `json:"is_default" example:"false"`
	CreatedAt    string  `json:"created_at" example:"2023-12-30 15:04:05"`
	UpdatedAt    string  `json:"updated_at" example:"2023-12-30 15:04:05"`
}

// ToResponse converts ShippingAddress model to ShippingAddressResponse
func (sar ShippingAddressResponse) ToResponse(model *model.ShippingAddress) *ShippingAddressResponse {
	return &ShippingAddressResponse{
		ID:           model.ID.String(),
		UserID:       model.UserID.String(),
		Type:         model.Type.String(),
		FullName:     model.FullName,
		PhoneNumber:  *model.PhoneNumber,
		Email:        *model.Email,
		Street:       model.Street,
		AddressLine2: model.AddressLine2,
		City:         model.City,
		PostalCode:   model.PostalCode,
		Country:      *model.Country,
		IsDefault:    model.IsDefault,
		CreatedAt:    utils.FormatLocalTime(&model.CreatedAt, ""),
		UpdatedAt:    utils.FormatLocalTime(&model.UpdatedAt, ""),
	}
}

func (sar ShippingAddressResponse) ToResponseList(models []model.ShippingAddress) (responses []*ShippingAddressResponse) {
	if len(models) == 0 {
		return []*ShippingAddressResponse{}
	}
	for _, model := range models {
		responses = append(responses, sar.ToResponse(&model))
	}
	return
}
