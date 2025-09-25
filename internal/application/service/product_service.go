package service

import (
	"context"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type productService struct {
	repository irepository.GenericRepository[model.Product]
}

func NewProductService(repo irepository.GenericRepository[model.Product]) iservice.ProductService {
	return &productService{
		repository: repo,
	}
}

func (p productService) GetProductsPagination(limit, offset int, search string) ([]*responses.ProductResponse, int, error) {
	zap.L().Debug("Fetching products with pagination",
		zap.Int("limit", limit),
		zap.Int("offset", offset),
		zap.String("search", search))

	ctx := context.Background()

	// Build filter for search
	var filter func(*gorm.DB) *gorm.DB
	if search != "" {
		filter = func(db *gorm.DB) *gorm.DB {
			return db.Where("name ILIKE ?", "%"+search+"%")
		}
		zap.L().Debug("Applied search filter to product query",
			zap.String("search_term", search))
	}

	// Fetch products with variants
	products, total, err := p.repository.GetAll(ctx, filter, []string{"Variants"}, limit, offset)
	if err != nil {
		zap.L().Error("Failed to fetch products from repository",
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.Error(err))
		return nil, 0, err
	}

	zap.L().Debug("Successfully fetched products from repository",
		zap.Int("products_count", len(products)),
		zap.Int64("total_products", total))

	// Map to DTOs
	productResponses := make([]*responses.ProductResponse, 0, len(products))
	for _, prod := range products {
		resp := &responses.ProductResponse{}
		productResponses = append(productResponses, resp.ToProductResponse(&prod))
	}

	zap.L().Info("Successfully retrieved products with pagination",
		zap.Int("returned_count", len(productResponses)),
		zap.Int("total_count", int(total)),
		zap.String("search_term", search))

	return productResponses, int(total), nil
}

func (p productService) GetProductByID(id string) (*responses.ProductResponse, error) {
	zap.L().Debug("Fetching product by ID - method not implemented",
		zap.String("product_id", id))
	// TODO: implement me
	panic("implement me")
}
