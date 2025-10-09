package requests

// UpdateProfileRequest represents profile update request
type UpdateProfileRequest struct {
	Username string `json:"username" validate:"omitempty,min=3,max=50,alphanum" example:"new_username"`
	Email    string `json:"email" validate:"omitempty,email" example:"new_email@example.com"`
}

// UpdateUserStatusRequest represents user status update request
type UpdateUserStatusRequest struct {
	IsActive bool `json:"is_active" validate:"required" example:"true"`
}

// UpdateUserRoleRequest represents user role update request
type UpdateUserRoleRequest struct {
	Role string `json:"role" validate:"required,oneof=ADMIN MARKETING_STAFF CONTENT_STAFF SALES_STAFF CUSTOMER BRAND_PARTNER" example:"CUSTOMER"`
}

// UserListRequest represents user list query parameters
type UserListRequest struct {
	PaginationRequest
	Search   *string `form:"search" json:"search" validate:"omitempty,max=100" example:"john"`
	Role     *string `form:"role" json:"role" validate:"omitempty,oneof=ADMIN MARKETING_STAFF CONTENT_STAFF SALES_STAFF CUSTOMER BRAND_PARTNER" example:"CUSTOMER"`
	IsActive *bool   `form:"is_active" json:"is_active" validate:"omitempty" example:"true"`
}
