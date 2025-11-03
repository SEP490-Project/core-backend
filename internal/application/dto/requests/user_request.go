package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UpdateProfileRequest represents profile update request
type UpdateProfileRequest struct {
	Username        *string                        `json:"username" validate:"omitempty,min=3,max=50,alphanum" example:"new_username"`
	FullName        *string                        `json:"full_name" validate:"omitempty,min=3,max=100" example:"John Doe"`
	Phone           *string                        `json:"phone" validate:"omitempty,e164" example:"+1234567890"`
	DateOfBirth     *time.Time                     `json:"date_of_birth" validate:"omitempty" example:"1990-01-01"`
	AvatarURL       *string                        `json:"avatar_url" validate:"omitempty,url" example:"https://example.com/avatar.jpg"`
	ShippingAddress []*UpdateAddressProfileRequest `json:"shipping_address" validate:"omitempty,dive"`
}

type UpdateAddressProfileRequest struct {
	ID           *string `json:"id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       *string `json:"user_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
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

// ToExistingProfile converts UpdateProfileRequest to existing model
// Skipping Username as it is required to check for uniqueness in the database separately
func (upr UpdateProfileRequest) ToExistingProfile(
	userModel *model.User,
) (
	profile *model.User,
	modifyingAddresses []model.ShippingAddress,
) {
	if userModel == nil {
		zap.L().Warn("UpdateProfileRequest.ToExistingProfile: model is nil")
		return nil, nil
	}
	if upr.FullName != nil {
		userModel.FullName = *upr.FullName
	}
	if upr.Phone != nil {
		userModel.Phone = *upr.Phone
	}
	if upr.DateOfBirth != nil {
		userModel.DateOfBirth = upr.DateOfBirth
	}
	if upr.AvatarURL != nil {
		userModel.AvatarURL = upr.AvatarURL
	}
	if len(upr.ShippingAddress) > 0 {
		// Create existing addresses map for quick lookup
		addressesMap := make(map[uuid.UUID]*model.ShippingAddress)
		for _, address := range userModel.ShippingAddress {
			addressesMap[address.ID] = &address
		}

		for _, addrReq := range upr.ShippingAddress {
			var modifiedAddr *model.ShippingAddress
			if addrReq.ID != nil {
				addressID, err := uuid.Parse(*addrReq.ID)
				if err != nil {
					zap.L().Warn("UpdateProfileRequest.ToExistingProfile: parsing address ID from requests Failed",
						zap.String("id", *addrReq.ID),
						zap.Error(err),
					)
					continue
				}

				var exists bool
				modifiedAddr, exists = addressesMap[addressID]
				if !exists {
					zap.L().Warn("UpdateProfileRequest.ToExistingProfile: provided address ID in request not found in existing addresses",
						zap.String("address.id", *addrReq.ID))
				}
			} else if addrReq.ID == nil {
				modifiedAddr = &model.ShippingAddress{
					ID:        uuid.Must(uuid.NewRandom()),
					UserID:    userModel.ID,
					IsDefault: false,
				}
			}
			convertedAddr, err := addrReq.ToExistingModel(modifiedAddr)
			if err != nil {
				zap.L().Warn("UpdateProfileRequest.ToExistingProfile: converting address from request to model failed",
					zap.String("address.id", func() string {
						if addrReq.ID != nil {
							return *addrReq.ID
						} else {
							return "new address"
						}
					}()),
					zap.Error(err))
			}
			modifyingAddresses = append(modifyingAddresses, *convertedAddr)
		}
	}

	return
}

// ToExistingModel converts UpdateAddressProfileRequest to existing model
func (uapr UpdateAddressProfileRequest) ToExistingModel(existing *model.ShippingAddress) (*model.ShippingAddress, error) {
	if existing == nil {
		return nil, errors.New("existing shipping address is nil")
	} else if existing.ID.String() != *uapr.ID {
		return nil, errors.New("ID mismatch between request and existing shipping address")
	}
	if uapr.Type != nil {
		addressType := enum.AddressType(*uapr.Type)
		if !addressType.IsValid() {
			zap.L().Error("invalid address type", zap.String("type", *uapr.Type))
			return nil, errors.New("invalid address type")
		} else {
			existing.Type = addressType
		}
	}
	if uapr.FullName != nil {
		existing.FullName = *uapr.FullName
	}
	if uapr.PhoneNumber != nil {
		existing.PhoneNumber = *uapr.PhoneNumber
	}
	if uapr.Email != nil {
		existing.Email = *uapr.Email
	}
	if uapr.Street != nil {
		existing.Street = *uapr.Street
	}
	if uapr.AddressLine2 != nil {
		existing.AddressLine2 = *uapr.AddressLine2
	}
	if uapr.City != nil {
		existing.City = *uapr.City
	}
	if uapr.PostalCode != nil {
		existing.PostalCode = uapr.PostalCode
	}
	if uapr.Country != nil {
		existing.Country = uapr.Country
	}
	if uapr.IsDefault != nil {
		existing.IsDefault = *uapr.IsDefault
	}
	return existing, nil
}

// UpdateUserStatusRequest represents user status update request
type UpdateUserStatusRequest struct {
	IsActive *bool `json:"is_active" validate:"required" example:"true"`
}

// UpdateUserRoleRequest represents user role update request
type UpdateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=ADMIN MARKETING_STAFF CONTENT_STAFF SALES_STAFF CUSTOMER BRAND_PARTNER" example:"CUSTOMER"`
}

// UserFilterRequest represents user list query parameters
type UserFilterRequest struct {
	PaginationRequest
	Search         *string `form:"search" json:"search" validate:"omitempty,max=100" example:"john"`
	Role           *string `form:"role" json:"role" validate:"omitempty,oneof=ADMIN MARKETING_STAFF CONTENT_STAFF SALES_STAFF CUSTOMER BRAND_PARTNER" example:"CUSTOMER"`
	IsActive       *bool   `form:"is_active" json:"is_active" validate:"omitempty" example:"true"`
	IsBrandAccount *bool   `form:"is_brand_account" json:"is_brand_account" validate:"omitempty" example:"true"`
}

// UserNotificationPreferenceRequest represents a request to update notification preferences
type UserNotificationPreferenceRequest struct {
	EmailEnabled *bool `json:"email_enabled" validate:"omitempty"`
	PushEnabled  *bool `json:"push_enabled" validate:"omitempty"`
}
