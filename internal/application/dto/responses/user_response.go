package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// region: ======= User Response =======

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
	IsBrandAccount     bool                       `json:"is_brand_account,omitempty" example:"false"`
	AvatarURL          *string                    `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	CreatedAt          string                     `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt          string                     `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	LastLogin          string                     `json:"last_login,omitempty" example:"2023-01-01T00:00:00Z"`
	CurrentLoginDevice []string                   `json:"current_login_device,omitempty" example:"[\"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3\"]"`
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
		AvatarURL:   model.AvatarURL,
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

	if model.Brand != nil {
		userResponse.IsBrandAccount = true
	}

	return
}

// endregion

// region: ======= User Info Response =======

type UserInfoResponse struct {
	ID          uuid.UUID `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username    string    `json:"username" example:"john_doe"`
	Email       string    `json:"email" example:"john@example.com"`
	Role        string    `json:"role" example:"user"`
	FullName    string    `json:"full_name" example:"John Doe"`
	Phone       string    `json:"phone" example:"+1234567890"`
	DateOfBirth string    `json:"date_of_birth" example:"1990-01-01"`
	IsActive    bool      `json:"is_active" example:"true"`
	AvatarURL   *string   `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
}

func (UserInfoResponse) ToResponse(model *model.User) *UserInfoResponse {
	return &UserInfoResponse{
		ID:          model.ID,
		Username:    model.Username,
		Email:       model.Email,
		Role:        model.Role.String(),
		FullName:    model.FullName,
		Phone:       model.Phone,
		DateOfBirth: utils.FormatLocalTime(model.DateOfBirth, utils.DateFormat),
		IsActive:    model.IsActive,
		AvatarURL:   model.AvatarURL,
	}
}

// endregion

// region: ======= User Notification Preference Response =======

// UserNotificationPreferenceResponse represents user notification preference in responses
type UserNotificationPreferenceResponse struct {
	ID           string `json:"id"`
	EmailEnabled bool   `json:"email_enabled"`
	PushEnabled  bool   `json:"push_enabled"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// ToResponse converts User model to UserNotificationPreferenceResponse
func (UserNotificationPreferenceResponse) ToResponse(model *model.User) *UserNotificationPreferenceResponse {
	return &UserNotificationPreferenceResponse{
		ID:           model.ID.String(),
		EmailEnabled: model.EmailEnabled,
		PushEnabled:  model.PushEnabled,
		CreatedAt:    utils.FormatLocalTime(model.CreatedAt, ""),
		UpdatedAt:    utils.FormatLocalTime(model.UpdatedAt, ""),
	}
}

// endregion

// region: ======= User List Response =======

type UserListResponse struct {
	ID               uuid.UUID     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username         string        `json:"username" example:"john_doe"`
	Email            string        `json:"email" example:"john@example.com"`
	Role             enum.UserRole `json:"role" example:"user"`
	FullName         string        `json:"full_name" example:"John Doe"`
	IsActive         bool          `json:"is_active" example:"true"`
	IsBrandAccount   bool          `json:"is_brand_account,omitempty" example:"false"`
	AvatarURL        *string       `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	CreatedAt        string        `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt        string        `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	NumberOfSessions int           `json:"number_of_sessions,omitempty" example:"3"`
}

func (ulr UserListResponse) ToListResponse(models []model.User) (userListResponses []*UserListResponse) {
	if len(models) == 0 {
		return []*UserListResponse{}
	}

	for _, user := range models {
		response := &UserListResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			Role:      user.Role,
			FullName:  user.FullName,
			IsActive:  user.IsActive,
			AvatarURL: user.AvatarURL,
			CreatedAt: utils.FormatLocalTime(user.CreatedAt, ""),
			UpdatedAt: utils.FormatLocalTime(user.UpdatedAt, ""),
		}
		if user.Brand != nil {
			response.IsBrandAccount = true
		}
		userListResponses = append(userListResponses, response)
	}
	return
}

func (ulr UserListResponse) ToSingleUserListResponse(model model.User) (userListResponse *UserListResponse) {
	userListResponse = &UserListResponse{
		ID:        model.ID,
		Username:  model.Username,
		Email:     model.Email,
		Role:      model.Role,
		FullName:  model.FullName,
		IsActive:  model.IsActive,
		AvatarURL: model.AvatarURL,
		CreatedAt: utils.FormatLocalTime(model.CreatedAt, ""),
		UpdatedAt: utils.FormatLocalTime(model.UpdatedAt, ""),
	}
	if model.Brand != nil {
		userListResponse.IsBrandAccount = true
	}
	return
}

// endregion

// UserPaginationResponse represents a paginated response for users.
// Only used for Swaggo swagger docs generation
type UserPaginationResponse PaginationResponse[UserListResponse]
