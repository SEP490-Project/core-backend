// Package responses defines common response DTOs for API responses.
package responses

import "net/http"

// APIResponse represents a standard API response structure.
type APIResponse struct {
	Success    bool   `json:"success"`
	Status     string `json:"status,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
	Data       any    `json:"data,omitempty"`
}

// PaginationResponse represents a paginated API response structure.
type PaginationResponse struct {
	Success    bool       `json:"success"`
	Status     string     `json:"status,omitempty"`
	StatusCode int        `json:"status_code,omitempty"`
	Message    string     `json:"message,omitempty"`
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

// Pagination contains pagination details.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// SuccessResponse creates a success API response.
func SuccessResponse(message string, statusCode int, data any) *APIResponse {
	return &APIResponse{
		Success:    true,
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
}

// ErrorResponse creates an error API response.
func ErrorResponse(message string, statusCode int) *APIResponse {
	return &APIResponse{
		Success:    false,
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
		Data:       nil,
	}
}

// PaginatedResponse creates a paginated API response.
func PaginatedResponse(
	message string,
	statusCode int,
	data any,
	pagination Pagination,
) *PaginationResponse {
	return &PaginationResponse{
		Success:    true,
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	}
}
