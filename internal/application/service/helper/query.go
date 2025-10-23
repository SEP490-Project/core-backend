package helper

import (
	"core-backend/internal/application/dto/requests"
	"fmt"
	"strings"
)

// ConvertToSortString converts PaginationRequest's SortBy and SortOrder to a SQL-compatible sort string
func ConvertToSortString(paginationRequest requests.PaginationRequest) string {
	sortBy := paginationRequest.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := strings.ToLower(paginationRequest.SortOrder)
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	return fmt.Sprintf("%s %s", sortBy, sortOrder)
}
