package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// UserResponse represents user information in responses
type UserResponse struct {
	ID        uuid.UUID     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string        `json:"username" example:"john_doe"`
	Email     string        `json:"email" example:"john@example.com"`
	Role      enum.UserRole `json:"role" example:"user"`
	IsActive  bool          `json:"is_active" example:"true"`
	CreatedAt string        `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt string        `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	LastLogin string        `json:"last_login,omitempty" example:"2023-01-01T00:00:00Z"`
}

func (ur *UserResponse) ToUserResponse(model *model.User) (userResponse *UserResponse) {
	userResponse = &UserResponse{
		ID:        model.ID,
		Username:  model.Username,
		Email:     model.Email,
		Role:      model.Role,
		IsActive:  model.IsActive,
		CreatedAt: utils.FormatLocalTime(model.CreatedAt, ""),
		UpdatedAt: utils.FormatLocalTime(model.UpdatedAt, ""),
		LastLogin: utils.FormatLocalTime(model.LastLogin, ""),
	}

	return
}

// UserPaginationResponse represents a paginated response for users.
// Only used for Swaggo swagger docs generation
type UserPaginationResponse PaginationResponse[UserResponse]
