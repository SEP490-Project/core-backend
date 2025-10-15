package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
)

type productService struct {
	repository   irepository.GenericRepository[model.Product]
	variantRepo  irepository.GenericRepository[model.ProductVariant]
	taskRepo     irepository.GenericRepository[model.Task]
	brandRepo    irepository.GenericRepository[model.Brand]
	categoryRepo irepository.GenericRepository[model.ProductCategory]
}

func (p productService) AddVariantAttributeValue(ctx context.Context, variantID uuid.UUID, attributeID uuid.UUID, attributeValue requests.CreateVariantAttributeValueRequest, uow irepository.UnitOfWork) (*model.VariantAttributeValue, error) {
	var varitantAttributeValue *model.VariantAttributeValue

	err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Validate variant
		if variantID == uuid.Nil {
			return errors.New("invalid variant id")
		}
		exists, err := uow.ProductVariant().ExistsByID(ctx, variantID)
		if err != nil {
			return fmt.Errorf("failed to check product variant existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("product variant with ID %s not found", variantID)
		}

		// Validate attribute IDs
		attrExists, err := uow.VariantAttributes().ExistsByID(ctx, attributeID)
		if err != nil {
			return fmt.Errorf("failed to check variant attribute existence: %w", err)
		}
		if !attrExists {
			return fmt.Errorf("variant attribute with ID %s not found", attributeID)
		}

		// Create VariantAttributeValue
		varitantAttributeValue = attributeValue.ToModel()
		varitantAttributeValue.VariantID = variantID
		if err := uow.VariantAttributeValue().Add(ctx, varitantAttributeValue); err != nil {
			zap.L().Info("failed to persist variant attribute value", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return varitantAttributeValue, nil
}

func (p productService) CreateVariantAttribute(ctx context.Context, createdByID uuid.UUID, attribute requests.CreateVariantAttributeRequest, uow irepository.UnitOfWork) (*model.VariantAttribute, error) {
	var variantAttribute *model.VariantAttribute

	err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Create variant attribute
		variantAttribute = attribute.ToModel(createdByID)
		if err := uow.VariantAttributes().Add(ctx, variantAttribute); err != nil {
			zap.L().Info("failed to persist variant attribute", zap.Error(err))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return variantAttribute, nil
}

func (p productService) CreateProductStory(ctx context.Context, variantID uuid.UUID, story requests.CreateProductStoryRequest, uow irepository.UnitOfWork) (*model.ProductStory, error) {
	//TODO implement me
	var productStory *model.ProductStory
	err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Validate variant
		if variantID == uuid.Nil {
			return errors.New("invalid variant id")
		}
		exists, err := uow.ProductVariant().ExistsByID(ctx, variantID)
		if err != nil {
			return fmt.Errorf("failed to check product variant existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("product variant with ID %s not found", variantID)
		}

		// Set the correct variant ID
		story.VariantID = variantID

		//Create product story
		productStory = story.ToModel()
		if err := uow.ProductStory().Add(ctx, productStory); err != nil {
			zap.L().Info("failed to persist product story", zap.Error(err))
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return productStory, nil
}

func (p productService) CreateVarianceImage(ctx context.Context, variantID uuid.UUID, image requests.CreateVariantImagesRequest, unitOfWork irepository.UnitOfWork) (*model.VariantImage, error) {
	var variantImage *model.VariantImage

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Validate variant
		if variantID == uuid.Nil {
			return errors.New("invalid variant id")
		}
		exists, err := p.variantRepo.ExistsByID(ctx, variantID)
		if err != nil {
			return fmt.Errorf("failed to check product variant existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("product variant with ID %s not found", variantID)
		}

		//Create VariantImage
		variantImage = image.ToModel()

		if err := uow.(irepository.UnitOfWork).VariantImage().Add(ctx, variantImage); err != nil {
			zap.L().Error("failed to persist variant image", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return variantImage, nil
}

func (p productService) CreateProductVariance(ctx context.Context, userID uuid.UUID, productID uuid.UUID, variant requests.CreateProductVariantRequest, unitOfWork irepository.UnitOfWork) (*model.ProductVariant, error) {
	var productVariant *model.ProductVariant

	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//Validate product
		if productID == uuid.Nil {
			return errors.New("invalid product id")
		}
		exists, err := uow.Products().ExistsByID(ctx, productID)
		if err != nil {
			return fmt.Errorf("failed to check product existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("product with ID %s not found", productID)
		}
		//Create ProductVariant
		productVariant = variant.ToModel(productID, userID)
		if err := uow.ProductVariant().Add(ctx, productVariant); err != nil {
			zap.L().Info("failed to persist product variant", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return productVariant, nil
}

func NewProductService(
	dbRegistry *gormrepository.DatabaseRegistry,
) iservice.ProductService {
	return &productService{
		repository:   dbRegistry.ProductRepository,
		variantRepo:  dbRegistry.ProductVariantRepository,
		taskRepo:     dbRegistry.TaskRepository,
		brandRepo:    dbRegistry.BrandRepository,
		categoryRepo: dbRegistry.ProductCategoryRepository,
	}
}

// CreateProduct creates a new product with default status DRAFT.
func (p productService) CreateProduct(dto *requests.CreateProductRequest, createdBy uuid.UUID) (*responses.ProductResponse, error) {
	if dto == nil {
		return nil, errors.New("nil dto")
	}
	if &dto.TaskID == nil || dto.TaskID == uuid.Nil {
		return nil, errors.New("task_id is required: product must depend on a task")
	}
	ctx := context.Background()
	// Validate task existence
	if found, err := p.taskRepo.GetByID(ctx, dto.TaskID, nil); err != nil {
		zap.L().Info("failed verifying task existence", zap.Error(err), zap.String("task_id", dto.TaskID.String()))
		return nil, errors.New("could not verify task existence")
	} else if found == nil {
		return nil, errors.New("task not found")
	} else if found.Status != enum.TaskStatusInProgress {
		return nil, errors.New("your task may expired or overdue")
	}
	// Validate brand existence
	if exists, err := p.brandRepo.ExistsByID(ctx, dto.BrandID); err != nil {
		zap.L().Info("failed verifying brand existence", zap.Error(err), zap.String("brand_id", dto.BrandID.String()))
		return nil, errors.New("could not verify brand existence")
	} else if !exists {
		return nil, errors.New("brand not found")
	}
	// Validate category existence
	if exists, err := p.categoryRepo.ExistsByID(ctx, dto.CategoryID); err != nil {
		zap.L().Info("failed verifying category existence", zap.Error(err), zap.String("category_id", dto.CategoryID.String()))
		return nil, errors.New("could not verify category existence")
	} else if !exists {
		return nil, errors.New("category not found")
	}

	entity := dto.ToModel(createdBy)
	entity.Status = enum.ProductStatusDraft

	if err := p.repository.Add(ctx, entity); err != nil {
		zap.L().Info("failed to persist product", zap.Error(err))
		return nil, err
	}

	// Reload with relations
	saved, err := p.repository.GetByID(ctx, entity.ID, []string{"Brand", "Category", "Variants"})
	if err != nil {
		zap.L().Warn("created product but failed to reload with relations", zap.Error(err))
		saved = entity
	}

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
		zap.L().Info("Failed to fetch products from repository",
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

// GetProductVariants lists variants for a product with pagination.
func (p productService) GetProductVariants(productID uuid.UUID, limit, offset int) ([]*responses.ProductVariantResponse, int, error) {
	ctx := context.Background()

	// Optionally ensure product exists
	exists, err := p.repository.ExistsByID(ctx, productID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, errors.New("product not found")
	}

	// Convert offset to pageNumber expected by repository (1-based)
	pageNumber := 1
	if limit > 0 && offset > 0 {
		pageNumber = (offset / limit) + 1
	}
	if limit <= 0 {
		limit = 10
	}

	filter := func(db *gorm.DB) *gorm.DB { return db.Where("product_id = ?", productID) }
	includes := []string{"Product"} // preload product for response Name/Type if needed

	variants, total, err := p.variantRepo.GetAll(ctx, filter, includes, limit, pageNumber)
	if err != nil {
		return nil, 0, err
	}

	res := make([]*responses.ProductVariantResponse, 0, len(variants))
	for i := range variants {
		res = append(res, responses.ProductVariantResponse{}.ToProductVariantResponse(&variants[i]))
	}
	return res, int(total), nil
}
