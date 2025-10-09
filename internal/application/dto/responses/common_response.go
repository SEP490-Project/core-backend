// Package responses defines common response DTOs for API responses.
package responses

import "net/http"

const (
	TimeFormat = "2006-12-30 15:04:05"
	DateFormat = "2006-12-30"
)

// APIResponse represents a standard API response structure.
type APIResponse struct {
	Success    bool   `json:"success"`
	Status     string `json:"status,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
	Data       any    `json:"data,omitempty"`
}

// PaginationResponse represents a paginated API response structure.
type PaginationResponse[T any] struct {
	Success    bool       `json:"success"`
	Status     string     `json:"status,omitempty"`
	StatusCode int        `json:"status_code,omitempty"`
	Message    string     `json:"message,omitempty"`
	Data       []T        `json:"data"`
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

// SuccessResponse creates a success API response, defaulting to HTTP 200 OK if no status code is provided.
func SuccessResponse(message string, statusCode *int, data any) *APIResponse {
	if statusCode == nil {
		statusCode = new(int)
		*statusCode = http.StatusOK
	}
	return &APIResponse{
		Success:    true,
		Status:     http.StatusText(*statusCode),
		StatusCode: *statusCode,
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
func NewPaginationResponse[T any](
	message string,
	statusCode int,
	data []T,
	pagination Pagination,
) *PaginationResponse[T] {
	if pagination.TotalPages == 0 && pagination.Total > 0 && pagination.Limit > 0 {
		pagination.TotalPages = int((pagination.Total + int64(pagination.Limit) - 1) / int64(pagination.Limit))
	}
	if pagination.HasNext == false && pagination.Page*pagination.Limit < int(pagination.Total) {
		pagination.HasNext = true
	}
	if pagination.HasPrev == false && pagination.Page > 1 {
		pagination.HasPrev = true
	}

	return &PaginationResponse[T]{
		Success:    true,
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
		Pagination: pagination,
	}
}

func EmptyPaginationResponse[T any](
	message string,
	statusCode *int,
	page int,
	limit int,
) *PaginationResponse[T] {
	if statusCode == nil {
		statusCode = new(int)
		*statusCode = http.StatusNoContent
	}

	return &PaginationResponse[T]{
		Success:    true,
		Status:     http.StatusText(*statusCode),
		StatusCode: *statusCode,
		Message:    message,
		Data:       []T{},
		Pagination: Pagination{
			Page:       page,
			Limit:      limit,
			Total:      0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    false,
		},
	}
}
