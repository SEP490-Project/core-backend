// Package requests contains common request DTOs for various API endpoints.
package requests

// PaginationRequest represents common pagination parameters for API requests.
type PaginationRequest struct {
	Page      int    `form:"page" json:"page" validate:"omitempty,min=1" example:"1"`
	Limit     int    `form:"limit" json:"limit" validate:"omitempty,min=1,max=100" example:"10"`
	SortBy    string `form:"sort_by" json:"sort_by" validate:"omitempty,max=50" example:"created_at"`
	SortOrder string `form:"sort_order" json:"sort_order" validate:"omitempty,oneof=asc desc" example:"asc"`
}
