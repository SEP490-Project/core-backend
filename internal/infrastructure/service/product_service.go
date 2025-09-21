package service

import (
	"context"
	"core-backend/internal/application/dto"
	"core-backend/internal/application/repository"
	"core-backend/internal/application/service"
	"core-backend/internal/domain/model"
	"gorm.io/gorm"
)

type productService struct {
	repository repository.GenericRepository[model.Product]
}

func NewProductService(repo repository.GenericRepository[model.Product]) service.ProductService {
	return &productService{
		repository: repo,
	}
}

func (p productService) GetProductsPagination(limit, offset int, search string) ([]*dto.ProductResponse, int, error) {
	ctx := context.Background()

	// Build filter for search
	var filter func(*gorm.DB) *gorm.DB
	if search != "" {
		filter = func(db *gorm.DB) *gorm.DB {
			return db.Where("name ILIKE ?", "%"+search+"%")
		}
	}

	// Fetch products with variants
	products, total, err := p.repository.GetAll(ctx, filter, []string{"Variants"}, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Map to DTOs
	responses := make([]*dto.ProductResponse, 0, len(products))
	for _, prod := range products {
		resp := &dto.ProductResponse{
			ID:          prod.ID,
			BrandID:     prod.BrandID,
			Name:        prod.Name,
			Description: prod.Description,
			Price:       prod.Price,
			Type:        prod.Type,
		}
		// Map variants
		for _, v := range prod.Variants {
			resp.Variants = append(resp.Variants, dto.ToProductVariantResponse(&v))
		}
		responses = append(responses, resp)
	}

	return responses, int(total), nil
}

func (p productService) GetProductByID(id string) (*dto.ProductResponse, error) {
	//TODO implement me
	panic("implement me")
}
