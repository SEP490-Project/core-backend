package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// UserResponse represents user information in responses
type UserResponse struct {
	ID        uuid.UUID     `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string        `json:"username" example:"john_doe"`
	Email     string        `json:"email" example:"john@example.com"`
	Role      enum.UserRole `json:"role" example:"user"`
	IsActive  bool          `json:"is_active" example:"true"`
	CreatedAt *time.Time    `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt *time.Time    `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	LastLogin *time.Time    `json:"last_login,omitempty" example:"2023-01-01T00:00:00Z"`
}

func (ur *UserResponse) ToUserResponse(model *model.User) *UserResponse {
	return &UserResponse{
		ID:        model.ID,
		Username:  model.Username,
		Email:     model.Email,
		Role:      model.Role,
		IsActive:  model.IsActive,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
		LastLogin: model.LastLogin,
	}
}

// UserPaginationResponse represents a paginated response for users.
// Only used for Swaggo swagger docs generation
type UserPaginationResponse PaginationResponse[UserResponse]
