package dto

import (
	"time"
	"github.com/google/uuid"
)

// UserResponse represents user information in responses
type UserResponse struct {
	ID        uuid.UUID  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string     `json:"username" example:"john_doe"`
	Email     string     `json:"email" example:"john@example.com"`
	Role      string     `json:"role" example:"user"`
	IsActive  bool       `json:"is_active" example:"true"`
	CreatedAt *time.Time  `json:"created_at" example:"2023-01-01T00:00:00Z"`
	UpdatedAt *time.Time  `json:"updated_at" example:"2023-01-01T00:00:00Z"`
	LastLogin *time.Time `json:"last_login,omitempty" example:"2023-01-01T00:00:00Z"`
}

// UpdateProfileRequest represents profile update request
type UpdateProfileRequest struct {
	Username string `json:"username,omitempty" validate:"omitempty,min=3,max=50" example:"new_username"`
	Email    string `json:"email,omitempty" validate:"omitempty,email" example:"new_email@example.com"`
}

// UpdateUserStatusRequest represents user status update request
type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required" example:"true"`
}

// UpdateUserRoleRequest represents user role update request
type UpdateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=admin user moderator" example:"user"`
}

// UserListRequest represents user list query parameters
type UserListRequest struct {
	Page     int     `form:"page" example:"1"`
	Limit    int     `form:"limit" example:"10"`
	Search   string  `form:"search" example:"john"`
	Role     string  `form:"role" example:"user"`
	IsActive *bool   `form:"is_active" example:"true"`
}

// UserListResponse represents paginated user list response
type UserListResponse struct {
	Users      []*UserResponse `json:"users"`
	Total      int             `json:"total" example:"100"`
	Page       int             `json:"page" example:"1"`
	Limit      int             `json:"limit" example:"10"`
	TotalPages int             `json:"total_pages" example:"10"`
	HasNext    bool            `json:"has_next" example:"true"`
	HasPrev    bool            `json:"has_prev" example:"false"`
}
