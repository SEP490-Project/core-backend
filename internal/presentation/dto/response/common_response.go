package response

import "net/http"

type APIResponse struct {
	Success    bool   `json:"success"`
	Status     string `json:"status,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
	Data       any    `json:"data,omitempty"`
}

type PaginationResponse struct {
	Success    bool       `json:"success"`
	Status     string     `json:"status,omitempty"`
	StatusCode int        `json:"status_code,omitempty"`
	Message    string     `json:"message,omitempty"`
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func SuccessResponse(message string, statusCode int, data any) *APIResponse {
	return &APIResponse{
		Success:    true,
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
		Data:       data,
	}
}

func ErrorResponse(message string, statusCode int) *APIResponse {
	return &APIResponse{
		Success:    false,
		Status:     http.StatusText(statusCode),
		StatusCode: statusCode,
		Message:    message,
		Data:       nil,
	}
}
