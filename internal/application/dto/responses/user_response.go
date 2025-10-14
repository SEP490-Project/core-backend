package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// UserResponse represents user information in responses
type UserResponse struct {
	ID                 uuid.UUID                  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username           string                     `json:"username" example:"john_doe"`
	Email              string                     `json:"email" example:"john@example.com"`
	Role               enum.UserRole              `json:"role" example:"user"`
	FullName           string                     `json:"full_name" example:"John Doe"`
	Phone              string                     `json:"phone" example:"+1234567890"`
	DateOfBirth        string                     `json:"date_of_birth" example:"1990-01-01"`
	IsActive           bool                       `json:"is_active" example:"true"`
	CreatedAt          string                     `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt          string                     `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	LastLogin          string                     `json:"last_login,omitempty" example:"2023-01-01T00:00:00Z"`
	CurrentLoginDevice []*string                  `json:"current_login_device,omitempty" example:"[\"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3\"]"`
	NumberOfSessions   int                        `json:"number_of_sessions,omitempty" example:"3"`
	ShippingAddress    []*ShippingAddressResponse `json:"shipping_address,omitempty"`
}

// ToUserResponse converts User model to UserResponse
func (ur *UserResponse) ToUserResponse(model *model.User) (userResponse *UserResponse) {
	userResponse = &UserResponse{
		ID:          model.ID,
		Username:    model.Username,
		Email:       model.Email,
		Role:        model.Role,
		FullName:    model.FullName,
		Phone:       model.Phone,
		DateOfBirth: utils.FormatLocalTime(model.DateOfBirth, utils.DateFormat),
		IsActive:    model.IsActive,
		CreatedAt:   utils.FormatLocalTime(model.CreatedAt, ""),
		UpdatedAt:   utils.FormatLocalTime(model.UpdatedAt, ""),
		LastLogin:   utils.FormatLocalTime(model.LastLogin, ""),
	}

	if len(model.ShippingAddress) > 0 {
		userResponse.ShippingAddress = ShippingAddressResponse{}.ToResponseList(model.ShippingAddress)
	}

	if len(model.Sessions) > 0 {
		loggedSessions := LoggedDeviceListResponse{}.ToResponseList(model.Sessions)
		userResponse.CurrentLoginDevice = append(userResponse.CurrentLoginDevice, loggedSessions...)
		userResponse.NumberOfSessions = len(model.Sessions)
	}

	return
}

// UserPaginationResponse represents a paginated response for users.
// Only used for Swaggo swagger docs generation
type UserPaginationResponse PaginationResponse[UserResponse]
