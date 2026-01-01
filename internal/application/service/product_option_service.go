package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/persistence"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	productOptionsCacheKeyPrefix = "product_options:"
	productOptionsCacheTTL       = 5 * time.Minute
)

type ProductOptionService struct {
	repo  irepository.ProductOptionRepository
	cache *persistence.ValkeyCache
}

// NewProductOptionService creates a new ProductOptionService
func NewProductOptionService(
	repo irepository.ProductOptionRepository,
	cache *persistence.ValkeyCache,
) iservice.ProductOptionService {
	return &ProductOptionService{
		repo:  repo,
		cache: cache,
	}
}

// GetByType retrieves all product options of a specific type (with caching)
func (s *ProductOptionService) GetByType(ctx context.Context, optionType string) ([]responses.ProductOptionResponse, error) {
	zap.L().Debug("ProductOptionService.GetByType called", zap.String("type", optionType))

	// Validate option type
	if !model.ProductOptionType(optionType).IsValid() {
		return nil, errors.New("invalid option type")
	}

	cacheKey := productOptionsCacheKeyPrefix + optionType

	// Try cache first
	if cached, _, err := s.cache.Get(cacheKey); err == nil && cached != nil {
		var cachedResponses []responses.ProductOptionResponse
		if data, ok := cached.([]byte); ok {
			if err := json.Unmarshal(data, &cachedResponses); err == nil {
				zap.L().Debug("ProductOptionService.GetByType cache hit", zap.String("type", optionType))
				return cachedResponses, nil
			}
		}
		// Handle case where cached is already unmarshaled by ValkeyCache
		if data, ok := cached.([]any); ok {
			jsonBytes, _ := json.Marshal(data)
			if err := json.Unmarshal(jsonBytes, &cachedResponses); err == nil {
				zap.L().Debug("ProductOptionService.GetByType cache hit (any)", zap.String("type", optionType))
				return cachedResponses, nil
			}
		}
	}

	// Cache miss - fetch from DB
	options, err := s.repo.GetByType(ctx, model.ProductOptionType(optionType), true)
	if err != nil {
		return nil, fmt.Errorf("failed to get product options: %w", err)
	}

	result := responses.ProductOptionResponse{}.ToListResponse(options)

	// Store in cache
	if cacheData, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(cacheKey, cacheData, productOptionsCacheTTL); err != nil {
			zap.L().Warn("ProductOptionService.GetByType failed to cache",
				zap.String("type", optionType),
				zap.Error(err))
		}
	}

	zap.L().Debug("ProductOptionService.GetByType cache miss, fetched from DB",
		zap.String("type", optionType),
		zap.Int("count", len(result)))

	return result, nil
}

// GetAll retrieves all product options with optional filtering and pagination
func (s *ProductOptionService) GetAll(ctx context.Context, req *requests.ProductOptionFilterRequest) ([]responses.ProductOptionResponse, int64, error) {
	zap.L().Debug("ProductOptionService.GetAll called", zap.Any("request", req))

	var optionType *model.ProductOptionType
	if req.Type != nil && *req.Type != "" {
		t := model.ProductOptionType(*req.Type)
		if !t.IsValid() {
			return nil, 0, errors.New("invalid option type")
		}
		optionType = &t
	}

	activeOnly := true
	if req.ActiveOnly != nil {
		activeOnly = *req.ActiveOnly
	}

	pageSize := req.Limit
	if pageSize <= 0 {
		pageSize = 100
	}
	pageNumber := req.Page
	if pageNumber <= 0 {
		pageNumber = 1
	}

	options, total, err := s.repo.GetAll(ctx, optionType, activeOnly, pageSize, pageNumber)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get product options: %w", err)
	}

	result := responses.ProductOptionResponse{}.ToListResponse(options)
	return result, total, nil
}

// GetByID retrieves a product option by ID
func (s *ProductOptionService) GetByID(ctx context.Context, id uuid.UUID) (*responses.ProductOptionResponse, error) {
	zap.L().Debug("ProductOptionService.GetByID called", zap.String("id", id.String()))

	option, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get product option: %w", err)
	}

	if option == nil {
		return nil, errors.New("product option not found")
	}

	return responses.ProductOptionResponse{}.ToResponse(option), nil
}

// ValidateOption validates if a code is valid for the given option type
func (s *ProductOptionService) ValidateOption(ctx context.Context, optionType model.ProductOptionType, code string) error {
	if code == "" {
		return nil // Empty code is valid (optional field)
	}

	valid, err := s.repo.IsValidOption(ctx, optionType, code)
	if err != nil {
		return fmt.Errorf("failed to validate option: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid %s: %s", optionType, code)
	}

	return nil
}

// Create creates a new product option (Admin only)
func (s *ProductOptionService) Create(ctx context.Context, uow irepository.UnitOfWork, req *requests.CreateProductOptionRequest) (*responses.ProductOptionResponse, error) {
	zap.L().Info("ProductOptionService.Create called",
		zap.String("type", req.Type),
		zap.String("code", req.Code))

	// Check if code already exists for this type
	existing, err := s.repo.GetByTypeAndCode(ctx, model.ProductOptionType(req.Type), req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing option: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("option with code '%s' already exists for type '%s'", req.Code, req.Type)
	}

	option := req.ToModel()

	if err := s.repo.Create(ctx, option); err != nil {
		return nil, fmt.Errorf("failed to create product option: %w", err)
	}

	// Invalidate cache for this type
	s.InvalidateCache(req.Type)

	zap.L().Info("ProductOptionService.Create successfully created product option",
		zap.String("id", option.ID.String()),
		zap.String("type", req.Type),
		zap.String("code", req.Code))

	return responses.ProductOptionResponse{}.ToResponse(option), nil
}

// Update updates an existing product option (Admin only)
func (s *ProductOptionService) Update(ctx context.Context, uow irepository.UnitOfWork, id uuid.UUID, req *requests.UpdateProductOptionRequest) (*responses.ProductOptionResponse, error) {
	zap.L().Info("ProductOptionService.Update called", zap.String("id", id.String()))

	option, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get product option: %w", err)
	}
	if option == nil {
		return nil, errors.New("product option not found")
	}

	// If changing code, check for duplicates
	if req.Code != nil && *req.Code != option.Code {
		existing, err := s.repo.GetByTypeAndCode(ctx, option.Type, *req.Code)
		if err != nil {
			return nil, fmt.Errorf("failed to check existing option: %w", err)
		}
		if existing != nil && existing.ID != id {
			return nil, fmt.Errorf("option with code '%s' already exists for type '%s'", *req.Code, option.Type)
		}
	}

	// Apply updates
	req.ApplyToModel(option)

	if err := s.repo.Update(ctx, option); err != nil {
		return nil, fmt.Errorf("failed to update product option: %w", err)
	}

	// Invalidate cache for this type
	s.InvalidateCache(string(option.Type))

	zap.L().Info("ProductOptionService.Update successfully updated product option",
		zap.String("id", id.String()))

	return responses.ProductOptionResponse{}.ToResponse(option), nil
}

// Delete soft deletes a product option (Admin only)
func (s *ProductOptionService) Delete(ctx context.Context, uow irepository.UnitOfWork, id uuid.UUID) error {
	zap.L().Info("ProductOptionService.Delete called", zap.String("id", id.String()))

	option, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get product option: %w", err)
	}
	if option == nil {
		return errors.New("product option not found")
	}

	optionType := string(option.Type)

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete product option: %w", err)
	}

	// Invalidate cache for this type
	s.InvalidateCache(optionType)

	zap.L().Info("ProductOptionService.Delete successfully deleted product option",
		zap.String("id", id.String()))

	return nil
}

// InvalidateCache invalidates the cache for a specific option type
func (s *ProductOptionService) InvalidateCache(optionType string) {
	cacheKey := productOptionsCacheKeyPrefix + optionType
	if err := s.cache.Delete(cacheKey); err != nil {
		zap.L().Warn("ProductOptionService.InvalidateCache failed",
			zap.String("type", optionType),
			zap.Error(err))
	}

	// Also invalidate "all" cache if it exists
	allCacheKey := productOptionsCacheKeyPrefix + "all"
	if err := s.cache.Delete(allCacheKey); err != nil {
		zap.L().Debug("ProductOptionService.InvalidateCache 'all' cache not found or failed",
			zap.Error(err))
	}
}
