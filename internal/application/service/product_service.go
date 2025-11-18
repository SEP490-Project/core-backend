package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/internal/infrastructure/third_party_repository"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type productService struct {
	repository           irepository.GenericRepository[model.Product]
	variantRepo          irepository.GenericRepository[model.ProductVariant]
	taskRepo             irepository.GenericRepository[model.Task]
	brandRepo            irepository.GenericRepository[model.Brand]
	categoryRepo         irepository.GenericRepository[model.ProductCategory]
	conceptRepo          irepository.GenericRepository[model.Concept]
	limitedProductRepo   irepository.GenericRepository[model.LimitedProduct]
	variantAttributeRepo irepository.GenericRepository[model.VariantAttribute]

	imageStorage irepository_third_party.S3Storage
	rabbitmq     *rabbitmq.RabbitMQ
}

func (p productService) PublishProduct(productID uuid.UUID, isActive bool) (*responses.ProductResponseV2, error) {
	product, err := p.repository.GetByID(context.Background(), productID, nil)
	if err != nil {
		zap.L().Info("failed to get product by id", zap.String("product_id", productID.String()), zap.Error(err))
		return nil, err
	}
	// Persist is_active explicitly to allow setting false (zero value)
	if err := p.repository.UpdateByCondition(context.Background(), func(db *gorm.DB) *gorm.DB { return db.Where("id = ?", productID) }, map[string]any{"is_active": isActive}); err != nil {
		zap.L().Error("failed to update product is_active", zap.String("product_id", productID.String()), zap.Error(err))
		return nil, err
	}
	// update in-memory model for response
	product.IsActive = isActive

	resp := &responses.ProductResponseV2{}
	return resp.ToProductResponseV2(product), nil
}

func (p productService) AddConceptToLimitedProduct(ctx context.Context, limitedProductID uuid.UUID, conceptID uuid.UUID, uow irepository.UnitOfWork) (*model.LimitedProduct, error) {
	var limitedProduct *model.LimitedProduct

	err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//validate limitedProduct
		limitedProductEntity, err := uow.LimitedProducts().GetByID(ctx, limitedProductID, []string{"Concept"})
		if err != nil {
			return fmt.Errorf("failed to get limited product by id: %w", err)
		}
		//validate concept
		conceptEntity, err := uow.Concepts().GetByID(ctx, conceptID, nil)
		if err != nil {
			return fmt.Errorf("failed to get concept by id: %w", err)
		}
		//add concept to limited product
		limitedProductEntity.ConceptID = &conceptEntity.ID
		if err := uow.LimitedProducts().Update(ctx, limitedProductEntity); err != nil {
			zap.L().Info("failed to update limited product with concept", zap.Error(err))
			return err
		}

		// set return variable so caller receives updated entity
		limitedProduct = limitedProductEntity
		return nil
	})

	if err != nil {
		return nil, err
	}

	return limitedProduct, nil
}

func (p productService) AddVariantAttributeValue(ctx context.Context, variantID uuid.UUID, attributeID uuid.UUID, attributeValue requests.CreateVariantAttributeValueRequest, uow irepository.UnitOfWork) (*model.VariantAttributeValue, error) {
	var variantAttributeValue *model.VariantAttributeValue
	var variantAttribute *model.VariantAttribute

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
		variantAttributeValue = attributeValue.ToModel()
		variantAttributeValue.VariantID = variantID
		if err := uow.VariantAttributeValue().Add(ctx, variantAttributeValue); err != nil {
			zap.L().Info("failed to persist variant attribute value", zap.Error(err))
			return err
		}

		//Load attribute to return
		variantAttribute, _ = uow.VariantAttributes().GetByID(ctx, attributeID, nil)

		return nil
	})

	if err != nil {
		return nil, err
	}
	variantAttributeValue.Attribute = variantAttribute
	return variantAttributeValue, nil
}

func (p productService) CreateVariantAttribute(ctx context.Context, createdByID uuid.UUID, attribute requests.CreateVariantAttributeRequest, uow irepository.UnitOfWork) (*model.VariantAttribute, error) {
	var variantAttribute *model.VariantAttribute

	err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Create variant attribute
		variantAttribute = attribute.ToCreationalModel(createdByID)
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

		if image.IsPrimary {
			result := uow.VariantImage().
				DB().
				WithContext(ctx).
				Model(&model.VariantImage{}).
				Where("variant_id = ?", variantID).
				Update("is_primary", false)

			if result.Error != nil {
				zap.L().Error("failed to unset primary variant images", zap.Error(result.Error))
				return fmt.Errorf("failed to unset primary variant images: %w", result.Error)
			}
		}

		//Create VariantImage
		variantImage = image.ToModel()

		if err := uow.VariantImage().Add(ctx, variantImage); err != nil {
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
	var productOfVariant *model.Product
	err := helper.WithTransaction(ctx, unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// validate dimensions
		err := p.variantDimensionValidation(variant)
		if err != nil {
			return err
		}

		//Validate product
		if productID == uuid.Nil {
			return errors.New("invalid product id")
		}
		//Limited variants of a product maximum is 5
		//Count variants
		variantCount, err := uow.ProductVariant().Count(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("product_id = ?", productID)
		})
		if err != nil {
			return fmt.Errorf("failed to count product variants: %w", err)
		} else if variantCount >= 5 {
			return errors.New("reach maximum variants for a product (5)")
		}

		// Load product now (check and propagate any error) so we can use it for response
		productOfVariant, err = uow.Products().GetByID(ctx, productID, nil)
		if err != nil {
			return fmt.Errorf("failed to load product by id: %w", err)
		}
		if productOfVariant == nil {
			return fmt.Errorf("product with ID %s not found after load", productID)
		}

		if productOfVariant.Type == enum.ProductTypeLimited {
			err := p.checkLimitedStockIntegrity(ctx, productID)
			if err != nil {
				return err
			}

			var (
				preOrderLimit = variant.PreOrderLimit
				inputStock    = variant.InputedStock
			)

			if preOrderLimit == nil || inputStock == nil {
				return fmt.Errorf("preorder_limit or input_stock cannot be empty if product was LIMITED")
			} else if *preOrderLimit > *inputStock {
				return fmt.Errorf("preorder_limit must not exceed input_stock")
			}
		}

		//Create ProductVariant
		productVariant = variant.ToModel(productID, userID, productOfVariant.Type)
		if err := uow.ProductVariant().Add(ctx, productVariant); err != nil {
			zap.L().Info("failed to persist product variant", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	//findProduct to buildResponse:
	productVariant.Product = productOfVariant
	return productVariant, nil
}

func (p productService) checkLimitedStockIntegrity(ctx context.Context, prdId uuid.UUID) error {
	//find variants by productID
	_, _, err := p.variantRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("product_id = ?", prdId)
	}, nil, 0, 0)

	if err != nil {
		return err
	}

	//totalStock := 0
	//for _, v := range variantList {
	//
	//}

	return nil
}

func (p productService) variantDimensionValidation(dto requests.CreateProductVariantRequest) error {
	const (
		maxWidth  = 200
		maxLength = 200
		maxHeight = 200
		maxWeight = 50000
	)

	var errs []string

	if dto.Width > maxWidth {
		errs = append(errs, fmt.Sprintf("width exceeds maximum limit of %d cm, your input: %d cm", maxWidth, dto.Width))
	}
	if dto.Length > maxLength {
		errs = append(errs, fmt.Sprintf("length exceeds maximum limit of %d cm, your input: %d cm", maxLength, dto.Length))
	}
	if dto.Height > maxHeight {
		errs = append(errs, fmt.Sprintf("height exceeds maximum limit of %d cm, your input: %d cm", maxHeight, dto.Height))
	}
	if dto.Weight > maxWeight {
		errs = append(errs, fmt.Sprintf("weight exceeds maximum limit of %d grams, your input: %d grams", maxWeight, dto.Weight))
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// CreateStandardProduct creates a new product with default status ACTIVE.
func (p productService) CreateStandardProduct(dto *requests.CreateStandardProductRequest, createdBy uuid.UUID) (*responses.ProductResponseV2, error) {
	ctx := context.Background()

	//Validate Request
	err := p.standardProductValidation(ctx, dto)
	if err != nil {
		return nil, err
	}

	entity := dto.ToStandardModel(createdBy)
	entity.Status = enum.ProductStatusActived

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

	resp := &responses.ProductResponseV2{}
	return resp.ToProductResponseV2(saved), nil
}

func (p productService) standardProductValidation(ctx context.Context, dto *requests.CreateStandardProductRequest) error {
	if dto == nil {
		return errors.New("null request")
	}

	// Validate brand existence
	if exists, err := p.brandRepo.ExistsByID(ctx, dto.BrandID); err != nil {
		zap.L().Info("failed verifying brand existence", zap.Error(err), zap.String("brand_id", dto.BrandID.String()))
		return errors.New("could not verify brand existence")
	} else if !exists {
		return errors.New("brand not found")
	}

	// Validate category existence
	if exists, err := p.categoryRepo.ExistsByID(ctx, dto.CategoryID); err != nil {
		zap.L().Info("failed verifying category existence", zap.Error(err), zap.String("category_id", dto.CategoryID.String()))
		return errors.New("could not verify category existence")
	} else if !exists {
		return errors.New("category not found")
	}

	return nil
}

func (p productService) CreateLimitedProduct(dto *requests.CreateLimitedProductRequest, createdBy uuid.UUID) (*responses.ProductResponseV2, error) {
	ctx := context.Background()
	err := p.limitedProductValidation(ctx, dto)
	if err != nil {
		return nil, err
	}

	entity := dto.ToProductWithLimitedModel(createdBy)
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

	resp := &responses.ProductResponseV2{}
	return resp.ToProductResponseV2(saved), nil
}

func (p productService) limitedProductValidation(ctx context.Context, dto *requests.CreateLimitedProductRequest) error {
	if dto == nil {
		return errors.New("nil dto")
	}

	if &dto.TaskID == nil || dto.TaskID == uuid.Nil {
		return errors.New("Task is required: Limited product must depend on a task")
	}
	// Validate task existence
	if found, err := p.taskRepo.GetByID(ctx, dto.TaskID, nil); err != nil {
		zap.L().Info("failed verifying task existence", zap.Error(err), zap.String("task_id", dto.TaskID.String()))
		return errors.New("could not verify task existence")
	} else if found == nil {
		return errors.New("task not found")
	} else if found.Status != enum.TaskStatusInProgress {
		return errors.New("your task may expired or overdue")
	}

	// Check if task already has a limited product
	existed, err := p.repository.Exists(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("task_id = ?", dto.TaskID)
	})
	if err != nil {
		zap.L().Info("failed checking existing limited product for task", zap.Error(err), zap.String("task_id", dto.TaskID.String()))
		return errors.New("could not verify existing limited product for task")
	} else if existed {
		return errors.New("a limited product for this task already exists")
	}

	// Validate brand existence
	if exists, err := p.brandRepo.ExistsByID(ctx, dto.BrandID); err != nil {
		zap.L().Info("failed verifying brand existence", zap.Error(err), zap.String("brand_id", dto.BrandID.String()))
		return errors.New("could not verify brand existence")
	} else if !exists {
		return errors.New("brand not found")
	}
	// Validate category existence
	if exists, err := p.categoryRepo.ExistsByID(ctx, dto.CategoryID); err != nil {
		zap.L().Info("failed verifying category existence", zap.Error(err), zap.String("category_id", dto.CategoryID.String()))
		return errors.New("could not verify category existence")
	} else if !exists {
		return errors.New("category not found")
	}
	// Additional validation logic for limited products can be added here

	return nil
}

func (p productService) GetProductsPagination(page, limit int, search, categoryID, productType string) ([]*responses.ProductResponse, int, error) {
	zap.L().Debug("Fetching products with pagination",
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.String("search", search),
		zap.String("category_id", categoryID),
		zap.String("product_type", productType),
	)

	ctx := context.Background()
	offset := (page - 1) * limit

	// --- Tạo filter chính ---
	filter := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where(`name ILIKE ?`, "%"+search+"%")
		}
		if categoryID != "" {
			cid, err := uuid.Parse(categoryID)
			if err == nil {
				db = db.Where(`category_id = ?`, cid)
			} else {
				zap.L().Warn("invalid category id filter provided, ignoring", zap.String("category_id", categoryID))
			}
		}
		if productType != "" {
			switch productType {
			case "STANDARD", "LIMITED":
				db = db.Where(`type = ?`, productType)
			default:
				zap.L().Warn("invalid product type provided, ignoring", zap.String("product_type", productType))
			}
		}

		// Only include active & published products by default
		db = db.Where("products.status = ? AND products.is_active = ?", enum.ProductStatusActived, true)
		return db.Order("products.created_at DESC").Order("products.id")
	}

	includes := []string{
		"Brand",
		"Variants",
		"Variants.Images",
		"Category",
		"Category.ParentCategory",
	}

	// === Bước 1: Lấy danh sách ID của page này ===
	var productIDs []uuid.UUID
	idFilter := filter

	// Query danh sách ID cho trang này
	err := p.repository.DB().
		WithContext(ctx).
		Model(&model.Product{}).
		Scopes(idFilter).
		Select("products.id").
		Limit(limit).
		Offset(offset).
		Pluck("products.id", &productIDs).Error
	if err != nil {
		return nil, 0, err
	}

	if len(productIDs) == 0 {
		// Không có dữ liệu ở trang này
		return []*responses.ProductResponse{}, 0, nil
	}

	// === Bước 2: Lấy total record (không preload) ===
	_, total, err := p.repository.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to count total products", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 3: Lấy thực thể kèm quan hệ theo ID ===
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("products.id IN ?", productIDs).Order("products.created_at DESC")
	}

	products, _, err := p.repository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch products with includes", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 4: Map sang DTO ===
	productResponses := make([]*responses.ProductResponse, 0, len(products))
	for i := range products {
		resp := &responses.ProductResponse{}
		productResponses = append(productResponses, resp.ToProductResponse(&products[i]))
	}

	zap.L().Info("Successfully retrieved products with pagination",
		zap.Int("returned_count", len(productResponses)),
		zap.Int("total_count", int(total)),
		zap.String("search_term", search),
	)

	return productResponses, int(total), nil
}

func (p productService) GetProductsPaginationV2(page, limit int, search, categoryID, productType string, productStatuses []string, isPreOrderOnly bool) ([]responses.ProductResponseV2, int, error) {
	zap.L().Debug("Fetching products with pagination",
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.String("search", search),
		zap.String("category_id", categoryID),
		zap.String("product_type", productType),
	)

	ctx := context.Background()
	offset := (page - 1) * limit

	// --- Tạo filter chính ---
	filter := func(db *gorm.DB) *gorm.DB {

		// Nếu filterPreOrder = TRUE → Bỏ hết status, type filter
		if isPreOrderOnly {
			return db.
				Joins("JOIN limited_products lp ON lp.id = products.id").
				Where("products.type = ?", enum.ProductTypeLimited).
				Where("lp.availability_start_date > NOW()").
				Order("products.created_at DESC").Order("products.id")
		}

		// ---- Normal filters ----

		if search != "" {
			db = db.Where(`name ILIKE ?`, "%"+search+"%")
		}

		if categoryID != "" {
			cid, err := uuid.Parse(categoryID)
			if err == nil {
				db = db.Where(`category_id = ?`, cid)
			} else {
				db = db.Where(`category_id = ?`, uuid.Nil)
			}
		}

		if productType != "" {
			db = db.Where(`type = ?`, productType)
		}

		// Support filtering by multiple statuses when provided
		if len(productStatuses) > 0 {
			db = db.Where("status IN ?", productStatuses)
		}

		return db.Order("products.created_at DESC").Order("products.id")
	}

	includes := []string{
		"Brand",
		"Variants",
		"Variants.Images",
		"Category",
		"Category.ParentCategory",
		"Category.ChildCategories",
		"CreatedBy",
		"UpdatedBy",
	}

	// === Bước 1: Lấy danh sách ID của page này ===
	var productIDs []uuid.UUID
	idFilter := filter

	// Query danh sách ID cho trang này
	err := p.repository.DB().
		WithContext(ctx).
		Model(&model.Product{}).
		Scopes(idFilter).
		Select("products.id").
		Limit(limit).
		Offset(offset).
		Pluck("products.id", &productIDs).Error
	if err != nil {
		return nil, 0, err
	}

	if len(productIDs) == 0 {
		// Không có dữ liệu ở trang này
		return []responses.ProductResponseV2{}, 0, nil
	}

	// === Bước 2: Lấy total record (không preload) ===
	_, total, err := p.repository.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to count total products", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 3: Lấy thực thể kèm quan hệ theo ID ===
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("products.id IN ?", productIDs).Order("products.created_at DESC")
	}

	products, _, err := p.repository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch products with includes", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 4: Map sang DTO ===
	productResponses := make([]responses.ProductResponseV2, 0, len(products))
	for i := range products {
		resp := responses.ProductResponseV2{}
		productResponses = append(productResponses, *resp.ToProductResponseV2(&products[i]))
	}

	zap.L().Info("Successfully retrieved products with pagination",
		zap.Int("returned_count", len(productResponses)),
		zap.Int("total_count", int(total)),
		zap.String("search_term", search),
	)

	return productResponses, int(total), nil
}

func (p productService) GetProductsPaginationV2Partial(page, limit int, search, categoryID string, productType string, isPreOrderOnly bool) ([]responses.ProductResponseV2Partial, int, error) {
	zap.L().Debug("Fetching products with pagination",
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.String("search", search),
		zap.String("category_id", categoryID),
		zap.String("product_type", productType),
	)

	ctx := context.Background()
	offset := (page - 1) * limit

	// --- Tạo filter chính ---
	filter := func(db *gorm.DB) *gorm.DB {

		// Nếu filterPreOrder = TRUE → Bỏ hết status, type filter
		if isPreOrderOnly {
			return db.
				Joins("JOIN limited_products lp ON lp.id = products.id").
				Where("products.type = ?", enum.ProductTypeLimited).
				Where("lp.availability_start_date > NOW()").
				Where("products.status = ?", enum.ProductStatusActived).
				Where("products.is_active = ?", true).
				Order("products.created_at DESC").Order("products.id")
		}

		// ---- Normal filters ----

		if search != "" {
			db = db.Where(`name ILIKE ?`, "%"+search+"%")
		}

		if categoryID != "" {
			cid, err := uuid.Parse(categoryID)
			if err == nil {
				db = db.Where(`category_id = ?`, cid)
			} else {
				db = db.Where(`category_id = ?`, uuid.Nil)
			}
		}

		if productType != "" {
			db = db.Where(`type = ?`, productType)
		}

		db = db.Where("products.status = ?", enum.ProductStatusActived).
			Where("products.is_active = ?", true)

		return db.Order("products.created_at DESC").Order("products.id")
	}

	includes := []string{
		"Brand",
		"Variants",
		"Variants.Images",
		"Category",
		"Category.ParentCategory",
		"Category.ChildCategories",
	}

	// === Bước 1: Lấy danh sách ID của page này ===
	var productIDs []uuid.UUID
	idFilter := filter

	// Query danh sách ID cho trang này
	err := p.repository.DB().
		WithContext(ctx).
		Model(&model.Product{}).
		Scopes(idFilter).
		Select("products.id").
		Limit(limit).
		Offset(offset).
		Pluck("products.id", &productIDs).Error
	if err != nil {
		return nil, 0, err
	}

	if len(productIDs) == 0 {
		// Không có dữ liệu ở trang này
		return []responses.ProductResponseV2Partial{}, 0, nil
	}

	// === Bước 2: Lấy total record (không preload) ===
	_, total, err := p.repository.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to count total products", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 3: Lấy thực thể kèm quan hệ theo ID ===
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("products.id IN ?", productIDs).Order("products.created_at DESC")
	}

	products, _, err := p.repository.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch products with includes", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 4: Map sang DTO ===
	productResponses := make([]responses.ProductResponseV2Partial, 0, len(products))
	for i := range products {
		resp := &responses.ProductResponseV2Partial{}
		productResponses = append(productResponses, *resp.ToProductResponseV2(&products[i]))
	}

	zap.L().Info("Successfully retrieved products with pagination",
		zap.Int("returned_count", len(productResponses)),
		zap.Int("total_count", int(total)),
		zap.String("search_term", search),
	)

	return productResponses, int(total), nil
}

func (p productService) GetProductDetail(id uuid.UUID) (*responses.ProductDetailResponse, error) {
	ctx := context.Background()
	res := responses.ProductDetailResponse{}

	// Use actual struct field names for nested preloads to avoid unsupported relation errors
	includes := []string{
		"Brand",
		"Brand.User",
		"Category",
		"Category.ParentCategory",
		"Variants",
		"Variants.Images",
		"Variants.AttributeValues",
		"Variants.AttributeValues.Attribute",
		"Variants.Story",
		"Limited",
		"Limited.Concept",
	}

	product, err := p.repository.GetByID(ctx, id, includes)
	if err != nil {
		zap.L().Info("failed to get product by id", zap.String("product_id", id.String()), zap.Error(err))
		return nil, err
	}
	if product == nil {
		return nil, errors.New("product not found")
	}

	return res.ToProductDetailResponse(product), nil
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

	exists, err := p.repository.ExistsByID(ctx, productID)
	if err != nil {
		return nil, 0, err
	}
	if !exists {
		return nil, 0, errors.New("product not found")
	}

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

func (p productService) GetVariantAttributePaginationPartial(limit, page int, search string) ([]responses.VariantAttributeResponse, int, error) {
	ctx := context.Background()
	pageNum := limit
	pageSize := page
	if pageSize <= 0 {
		pageNum = page
		pageSize = limit
	}
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	// --- Tạo filter chính ---
	filter := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			// VariantAttribute model uses Ingredient column
			db = db.Where(`ingredient ILIKE ?`, "%"+search+"%")
		}
		return db.Order("variant_attributes.created_at DESC").Order("variant_attributes.id")
	}

	includes := []string{}

	var variantAttributeIDs []uuid.UUID
	idFilter := filter

	// Query danh sách ID cho trang này
	err := p.variantAttributeRepo.DB().
		WithContext(ctx).
		Model(&model.VariantAttribute{}).
		Scopes(idFilter).
		Select("variant_attributes.id").
		Limit(pageSize).
		Offset(offset).
		Pluck("variant_attributes.id", &variantAttributeIDs).Error
	if err != nil {
		return nil, 0, err
	}

	if len(variantAttributeIDs) == 0 {
		return []responses.VariantAttributeResponse{}, 0, nil
	}

	// === Bước 2: Lấy total record (không preload) ===
	// Use variantAttributeRepo for counting
	// Build count scope without ORDER to avoid DISTINCT+ORDER BY issues
	countScope := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where(`ingredient ILIKE ?`, "%"+search+"%")
		}
		return db
	}
	var total int64
	if err := p.variantAttributeRepo.DB().WithContext(ctx).Model(&model.VariantAttribute{}).Scopes(countScope).Count(&total).Error; err != nil {
		zap.L().Error("Failed to count variant attributes", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 3: Lấy thực thể kèm quan hệ theo ID ===
	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("variant_attributes.id IN ?", variantAttributeIDs).Order("variant_attributes.created_at DESC")
	}

	variantAttributes, _, err := p.variantAttributeRepo.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch products with includes", zap.Error(err))
		return nil, 0, err
	}

	// === Bước 4: Map sang DTO ===
	variantAttributeResp := make([]responses.VariantAttributeResponse, 0, len(variantAttributes))
	for i := range variantAttributes {
		resp := responses.VariantAttributeResponse{}
		variantAttributeResp = append(variantAttributeResp, resp.ToVariantAttributeResponse(variantAttributes[i]))
	}

	zap.L().Info("Successfully retrieved products with pagination",
		zap.Int("returned_count", len(variantAttributeResp)),
		zap.Int("total_count", int(total)),
		zap.String("search_term", search),
	)

	return variantAttributeResp, int(total), nil
}

func (p productService) GetVariantAttributePagination(limit, page int, search string) ([]model.VariantAttribute, int, error) {
	ctx := context.Background()
	pageNum := limit
	pageSize := page
	if pageSize <= 0 {
		pageNum = page
		pageSize = limit
	}
	if pageNum < 1 {
		pageNum = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	offset := (pageNum - 1) * pageSize

	filter := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where(`ingredient ILIKE ?`, "%"+search+"%")
		}
		return db.Order("variant_attributes.created_at DESC").Order("variant_attributes.id")
	}

	includes := []string{}

	var variantAttributeIDs []uuid.UUID
	// Query IDs for this page
	err := p.variantAttributeRepo.DB().WithContext(ctx).Model(&model.VariantAttribute{}).Scopes(filter).Select("variant_attributes.id").Limit(pageSize).Offset(offset).Pluck("variant_attributes.id", &variantAttributeIDs).Error
	if err != nil {
		return nil, 0, err
	}
	if len(variantAttributeIDs) == 0 {
		return []model.VariantAttribute{}, 0, nil
	}

	// Count
	countScope := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where(`ingredient ILIKE ?`, "%"+search+"%")
		}
		return db
	}
	var total int64
	if err := p.variantAttributeRepo.DB().WithContext(ctx).Model(&model.VariantAttribute{}).Scopes(countScope).Count(&total).Error; err != nil {
		zap.L().Error("Failed to count variant attributes", zap.Error(err))
		return nil, 0, err
	}

	finalFilter := func(db *gorm.DB) *gorm.DB {
		return db.Where("variant_attributes.id IN ?", variantAttributeIDs).Order("variant_attributes.created_at DESC")
	}

	variantAttributes, _, err := p.variantAttributeRepo.GetAll(ctx, finalFilter, includes, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch variant attributes", zap.Error(err))
		return nil, 0, err
	}

	return variantAttributes, int(total), nil
}

func (p productService) GetTop5NewestProducts() (*responses.ProductResponseTop5Newest, error) {
	ctx := context.Background()

	stdFilter := func(db *gorm.DB) *gorm.DB {
		db = db.Where(`type = ?`, enum.ProductTypeStandard)

		//filter valid only
		db.Where(`status = ?`, enum.ProductStatusActived).Where(``)
		return db.Order("products.created_at DESC").Order("products.id")
	}
	limitFilter := func(db *gorm.DB) *gorm.DB {
		db = db.Where(`type = ?`, enum.ProductTypeLimited)
		return db.Order("products.created_at DESC").Order("products.id")
	}

	includes := []string{
		"Brand",
		"Variants",
		"Variants.Images",
		"Category",
		"Category.ParentCategory",
	}

	//Get 5 newest standard products
	stdProducts, _, err := p.repository.GetAll(ctx, stdFilter, includes, 5, 1)
	if err != nil {
		zap.L().Error("Failed to fetch top 5 newest products", zap.Error(err))
		return nil, err
	}

	limitedProducts, _, err := p.repository.GetAll(ctx, limitFilter, includes, 5, 1)
	if err != nil {
		zap.L().Error("Failed to fetch top 5 newest limited products", zap.Error(err))
		return nil, err
	}

	stdProductResp := make([]responses.ProductResponseV2Partial, 0, 5)
	limitedProductResp := make([]responses.ProductResponseV2Partial, 0, 5)
	prdMapper := &responses.ProductResponseV2Partial{}
	for i := range 5 {
		if i < len(stdProducts) {
			stdItem := prdMapper.ToProductResponseV2(&stdProducts[i])
			stdProductResp = append(stdProductResp, *stdItem)
		}
		if i < len(limitedProducts) {
			limitedItem := prdMapper.ToProductResponseV2(&limitedProducts[i])
			limitedProductResp = append(limitedProductResp, *limitedItem)
		}
	}

	resp := &responses.ProductResponseTop5Newest{
		Standard: stdProductResp,
		Limited:  limitedProductResp,
	}

	return resp, nil
}

func (p productService) UpdateVariantImage(ctx context.Context, variantImageID uuid.UUID, image requests.UpdateVariantImagesRequest, uow irepository.UnitOfWork) (*model.VariantImage, error) {
	var variantImage *model.VariantImage

	if err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {

		//Load existing variant image
		var err error
		variantImage, err = uow.VariantImage().GetByID(ctx, variantImageID, nil)
		if err != nil {
			return fmt.Errorf("failed to load variant image by id: %w", err)
		}
		if variantImage == nil {
			return fmt.Errorf("variant image with ID %s not found after load", variantImageID)
		}

		//Update fields
		image.ToModel(variantImage)

		if err := uow.VariantImage().Update(ctx, variantImage); err != nil {
			zap.L().Error("failed to update variant image", zap.Error(err))
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return variantImage, nil
}

func (p productService) UpdateVariantImageAsync(ctx context.Context, userID, variantImageID uuid.UUID, filePath *string, image requests.UpdateVariantImagesRequest, uow irepository.UnitOfWork) (*model.VariantImage, error) {
	var variantImage *model.VariantImage

	err := helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		var uploadedURL *string
		if filePath != nil {
			file, err := os.Open(*filePath)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			fileName := filepath.Base(*filePath)
			key := fmt.Sprintf("%s/%s", userID.String(), fileName)

			if err := p.imageStorage.Put(ctx, key, file, "application/octet-stream"); err != nil {
				return fmt.Errorf("failed to upload file: %w", err)
			}

			url := p.imageStorage.BuildUrl(key)
			uploadedURL = &url
		}

		// Load variant image
		variantImage, err := uow.VariantImage().GetByID(ctx, variantImageID, nil)
		if err != nil {
			return fmt.Errorf("failed to load variant image by id: %w", err)
		}
		if variantImage == nil {
			return fmt.Errorf("variant image with ID %s not found after load", variantImageID)
		}

		// Gán URL mới
		if uploadedURL != nil {
			variantImage.ImageURL = *uploadedURL
		}

		// Cập nhật dữ liệu mới vào model
		image.ToModel(variantImage)

		if err := uow.VariantImage().Update(ctx, variantImage); err != nil {
			zap.L().Error("failed to update variant image", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return variantImage, nil
}

func (p productService) BuildFileURL(fileName string) string {
	return p.imageStorage.BuildUrl(fileName)
}

func NewProductService(
	dbRegistry *gormrepository.DatabaseRegistry,
	storage3rd *third_party_repository.ThirdPartyStorageRegistry,
	rabbitmq *rabbitmq.RabbitMQ,
) iservice.ProductService {
	return &productService{
		repository:           dbRegistry.ProductRepository,
		variantRepo:          dbRegistry.ProductVariantRepository,
		taskRepo:             dbRegistry.TaskRepository,
		brandRepo:            dbRegistry.BrandRepository,
		categoryRepo:         dbRegistry.ProductCategoryRepository,
		conceptRepo:          dbRegistry.ConceptRepository,
		limitedProductRepo:   dbRegistry.LimitedProductRepository,
		variantAttributeRepo: dbRegistry.VariantAttributeRepository,
		imageStorage:         storage3rd.S3Storage,
		rabbitmq:             rabbitmq,
	}
}
