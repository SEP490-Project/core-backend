package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CreateShippingAddressRequest represents the payload to create a new shipping address
type CreateShippingAddressRequest struct {
	UserID       string  `json:"user_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type         string  `json:"type" validate:"required,oneof=BILLING SHIPPING" example:"SHIPPING"`
	FullName     string  `json:"full_name" validate:"required,min=3,max=100" example:"John Doe"`
	PhoneNumber  string  `json:"phone_number" validate:"required,e164" example:"+1234567890"`
	Email        string  `json:"email" validate:"required,email" example:"john@example.com"`
	Street       string  `json:"street" validate:"required,min=5,max=200" example:"123 Main St"`
	AddressLine2 string  `json:"address_line_2" validate:"omitempty,max=200" example:"Apt 4B"`
	City         string  `json:"city" validate:"required,min=2,max=100" example:"New York"`
	PostalCode   *string `json:"postal_code" validate:"required,min=2,max=20" example:"10001"`
	Country      *string `json:"country" validate:"required,min=2,max=100" example:"USA"`
	//Company      *string `json:"company" validate:"omitempty,max=100" example:"Acme Corp"`
	IsDefault *bool `json:"is_default" validate:"omitempty" example:"false"`
}

// ToModel converts CreateShippingAddressRequest to ShippingAddress model
func (r CreateShippingAddressRequest) ToModel() (*model.ShippingAddress, error) {
	userID, err := uuid.Parse(r.UserID)
	if err != nil {
		zap.L().Error("failed to parse user ID", zap.Error(err))
		return nil, err
	}
	addressType := enum.AddressType(r.Type)
	if !addressType.IsValid() {
		zap.L().Error("invalid address type", zap.String("type", r.Type))
		return nil, err
	}
	isDefault := false
	if r.IsDefault != nil {
		isDefault = *r.IsDefault
	}

	model := &model.ShippingAddress{
		ID:           uuid.Must(uuid.NewRandom()),
		UserID:       userID,
		Type:         addressType,
		FullName:     r.FullName,
		PhoneNumber:  r.PhoneNumber,
		Email:        r.Email,
		Street:       r.Street,
		AddressLine2: r.AddressLine2,
		City:         r.City,
		PostalCode:   r.PostalCode,
		Country:      r.Country,
		IsDefault:    isDefault,
	}
	return model, nil
}

// UpdateShippingAddressRequest represents the payload to update an existing shipping address
type UpdateShippingAddressRequest struct {
	ID           *string `json:"id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Type         *string `json:"type" validate:"required,oneof=BILLING SHIPPING" example:"SHIPPING"`
	FullName     *string `json:"full_name" validate:"required,min=3,max=100" example:"John Doe"`
	PhoneNumber  *string `json:"phone_number" validate:"required,e164" example:"+1234567890"`
	Email        *string `json:"email" validate:"required,email" example:"john@example.com"`
	Street       *string `json:"street" validate:"required,min=5,max=200" example:"123 Main St"`
	AddressLine2 *string `json:"address_line_2" validate:"omitempty,max=200" example:"Apt 4B"`
	City         *string `json:"city" validate:"required,min=2,max=100" example:"New York"`
	State        *string `json:"state" validate:"omitempty,min=2,max=100" example:"NY"`
	PostalCode   *string `json:"postal_code" validate:"required,min=2,max=20" example:"10001"`
	Country      *string `json:"country" validate:"required,min=2,max=100" example:"USA"`
	Company      *string `json:"company" validate:"omitempty,max=100" example:"Acme Corp"`
	IsDefault    *bool   `json:"is_default" validate:"omitempty" example:"false"`
}

// ToExistingModel converts UpdateShippingAddressRequest to ShippingAddress model
func (r UpdateShippingAddressRequest) ToExistingModel(existing *model.ShippingAddress) (*model.ShippingAddress, error) {
	if existing == nil {
		return nil, errors.New("existing shipping address is nil")
	} else if existing.ID.String() != *r.ID {
		return nil, errors.New("ID mismatch between request and existing shipping address")
	}
	if r.Type != nil {
		addressType := enum.AddressType(*r.Type)
		if !addressType.IsValid() {
			zap.L().Error("invalid address type", zap.String("type", *r.Type))
			return nil, errors.New("invalid address type")
		} else {
			existing.Type = addressType
		}
	}
	if r.FullName != nil {
		existing.FullName = *r.FullName
	}
	if r.PhoneNumber != nil {
		existing.PhoneNumber = *r.PhoneNumber
	}
	if r.Email != nil {
		existing.Email = *r.Email
	}
	if r.Street != nil {
		existing.Street = *r.Street
	}
	if r.AddressLine2 != nil {
		existing.AddressLine2 = *r.AddressLine2
	}
	if r.City != nil {
		existing.City = *r.City
	}
	if r.PostalCode != nil {
		existing.PostalCode = r.PostalCode
	}
	if r.Country != nil {
		existing.Country = r.Country
	}
	if r.IsDefault != nil {
		existing.IsDefault = *r.IsDefault
	}
	return existing, nil
}
