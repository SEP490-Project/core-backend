package service

import "core-backend/internal/application/dto"

type ProductService interface {
	GetProductsPagination(limit, offset int, search string) ([]*dto.ProductResponse, int, error)
	GetProductByID(id string) (*dto.ProductResponse, error)
}
