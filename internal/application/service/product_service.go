package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type productService struct {
	repository irepository.GenericRepository[model.Product]
}

func NewProductService(repo irepository.GenericRepository[model.Product]) iservice.ProductService {
	return &productService{
		repository: repo,
	}
}

// CreateProduct creates a new product with default status DRAFT.
func (p productService) CreateProduct(dto *requests.CreateProductDTO, createdBy uuid.UUID) (*responses.ProductResponse, error) {
	if dto == nil {
		return nil, errors.New("nil dto")
	}

	ctx := context.Background()
	entity := dto.ToModel(createdBy)
	// Set default workflow state
	entity.Status = enum.ProductStatusDraft

	if err := p.repository.Add(ctx, entity); err != nil {
		zap.L().Error("failed to persist product", zap.Error(err))
		return nil, err
	}

	// Reload with relations for response mapping
	saved, err := p.repository.GetByID(ctx, entity.ID, []string{"Brand", "Category", "Variants"})
	if err != nil {
		zap.L().Warn("created product but failed to reload with relations", zap.Error(err))
		// Fallback to entity if reload fails
		saved = entity
	}

	// Guard nil description for mapper
	if saved.Description == nil {
		empty := ""
		saved.Description = &empty
	}

	resp := &responses.ProductResponse{}
	return resp.ToProductResponse(saved), nil
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
	products, total, err := p.repository.GetAll(ctx, filter, []string{"Variants", "Category", "Category.ParentCategory"}, limit, offset)
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
		// ensure non-nil description for mapper
		if prod.Description == nil {
			empty := ""
			prod.Description = &empty
		}
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

func isStaffRole(role string) bool {
	if role == "ADMIN" {
		return true
	}
	return strings.HasSuffix(role, "_STAFF")
}

func (p productService) GetProductsByTask(taskID uuid.UUID, requestingUserID uuid.UUID, userRole string, limit, offset int) ([]*responses.ProductOverviewResponse, int, error) {
	ctx := context.Background()

	// Convert offset to pageNumber expected by repository (1-based)
	pageNumber := 1
	if limit > 0 && offset > 0 {
		pageNumber = (offset / limit) + 1
	}
	if limit <= 0 {
		limit = 10
	}

	// Build filter for products of given task
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("task_id = ?", taskID)
	}

	includes := []string{
		"Brand.User",
		"Category",
		"Variants",
		"Task.Milestone.Campaign.Contract.Brand.User",
	}

	products, total, err := p.repository.GetAll(ctx, filter, includes, limit, pageNumber)
	if err != nil {
		return nil, 0, err
	}

	// Authorization: allow staff roles; else ensure ownership via brand user chain.
	if !isStaffRole(userRole) {
		authorized := false
		for _, prod := range products {
			if prod.Task != nil && prod.Task.Milestone != nil && prod.Task.Milestone.Campaign != nil && prod.Task.Milestone.Campaign.Contract != nil && prod.Task.Milestone.Campaign.Contract.Brand != nil && prod.Task.Milestone.Campaign.Contract.Brand.UserID != nil {
				if *prod.Task.Milestone.Campaign.Contract.Brand.UserID == requestingUserID {
					authorized = true
					break
				}
			} else if prod.Brand != nil && prod.Brand.UserID != nil && *prod.Brand.UserID == requestingUserID { // fallback direct brand ownership
				authorized = true
				break
			}
		}
		if !authorized {
			return nil, 0, errors.New("forbidden: not authorized to view products for this task")
		}
	}

	// Map to overview responses
	ptrProducts := make([]*model.Product, 0, len(products))
	for i := range products { // need pointers
		ptrProducts = append(ptrProducts, &products[i])
	}
	overview := responses.ToOverviewList(ptrProducts)

	return overview, int(total), nil
}
