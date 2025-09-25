package iservice

import "core-backend/internal/application/dto/responses"

type ProductService interface {
	GetProductsPagination(limit, offset int, search string) ([]*responses.ProductResponse, int, error)
	GetProductByID(id string) (*responses.ProductResponse, error)
}
