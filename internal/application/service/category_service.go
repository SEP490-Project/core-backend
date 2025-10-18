package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type productCategoryService struct {
	categoryRepository irepository.GenericRepository[model.ProductCategory]
}

func (c productCategoryService) GetAllCategories(limit, offset int, search string, deleted string) ([]*responses.ProductCategoryResponse, int64, error) {
	ctx := context.Background()

	filter := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			return db.Where("name ILIKE ?", "%"+search+"%")
		}

		if deleted == "true" {
			db.Unscoped().Where("deleted_at IS NOT NULL")
		} else if deleted == "false" {
			db.Where("deleted_at IS NULL")
		}
		return db.Order("product_categories.Created_at DESC")
	}

	includes := []string{"ParentCategory", "ChildCategories"}
	categories, total, err := c.categoryRepository.GetAll(ctx, filter, includes, limit, offset)

	if err != nil {
		return nil, 0, err
	}

	categoryResponses := make([]*responses.ProductCategoryResponse, 0, len(categories))
	for _, category := range categories {
		categoryResponse := (&responses.ProductCategoryResponse{}).ToModelResponse(&category)
		categoryResponses = append(categoryResponses, categoryResponse)
	}

	return categoryResponses, total, nil
}

func (c productCategoryService) CreateCategory(request requests.CreateProductCategoryRequest) (*responses.ProductCategoryResponse, error) {
	ctx := context.Background()
	categoryModel := request.ToModel()

	//Check parent category existence
	if request.ParentCategoryID != nil {
		isExist, err := c.categoryRepository.ExistsByID(ctx, request.ParentCategoryID)
		if err != nil || !isExist {
			zap.L().Debug("Parent Category Not Found", zap.Error(err))
			return nil, errors.New("parent Category Not Found")
		}
	}

	isParent, _ := c.isParentCategory(categoryModel.ParentCategoryID)
	if !isParent {
		return nil, errors.New("this category is a child of another, cannot set as parent category")
	}

	if err := c.categoryRepository.Add(ctx, categoryModel); err != nil {
		zap.L().Info("Failed to create category", zap.Error(err))
		return nil, err
	}

	categoryResponse := (&responses.ProductCategoryResponse{}).ToModelResponse(categoryModel)
	return categoryResponse, nil
}

func (c productCategoryService) AddParentCategory(currentID uuid.UUID, parentID uuid.UUID) (*responses.ProductCategoryResponse, error) {
	ctx := context.Background()

	//check category existence
	found, err := c.categoryRepository.GetByID(ctx, currentID, []string{"ParentCategory", "ChildCategories"})
	if err != nil {
		return nil, err
	} else if found == nil {
		return nil, errors.New("category not found")
	}
	//check parent category existence
	if parentID == uuid.Nil {
		found.ParentCategoryID = nil
		found.ParentCategory = nil //build response
	} else {
		parentFound, err := c.categoryRepository.GetByID(ctx, parentID, []string{})
		if err != nil {
			return nil, err
		} else if parentFound == nil {
			return nil, errors.New("parent category not found")
		} else if parentFound.ParentCategoryID != nil {
			return nil, errors.New("this category is a child of another, cannot set as parent category")
		}

		found.ParentCategoryID = &parentID
	}

	found.UpdatedAt = time.Now()
	if err := c.categoryRepository.DB().
		Model(found).
		Select("parent_category_id", "updated_at").
		Updates(found).Error; err != nil {
		return nil, err
	}

	categoryResponse := (&responses.ProductCategoryResponse{}).ToModelResponse(found)
	return categoryResponse, nil
}

func (c productCategoryService) DeleteCategory(ctx context.Context, categoryID uuid.UUID, uow irepository.UnitOfWork) error {
	return helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		//check category existence
		found, err := c.categoryRepository.GetByID(ctx, categoryID, []string{"ParentCategory", "ChildCategories"})
		if err != nil {
			return err
		} else if found == nil {
			return errors.New("category not found")
		}

		// mark deleted
		found.DeletedAt.Time = time.Now()
		found.DeletedAt.Valid = true
		found.UpdatedAt = time.Now()
		if found.ParentCategoryID != nil {
			found.ParentCategoryID = nil
		}

		db := uow.ProductCategory().DB()
		if db == nil {
			return errors.New("database handle is nil")
		}

		if err := db.Model(found).
			Select("deleted_at", "updated_at").
			Updates(found).Error; err != nil {
			return err
		}

		if len(found.ChildCategories) == 0 {
			return nil
		}

		if err := db.Model(&model.ProductCategory{}).
			Where("parent_category_id = ?", found.ID).
			Updates(map[string]any{
				"parent_category_id": nil,
				"updated_at":         time.Now(),
			}).Error; err != nil {
			zap.L().Error("Failed to remove parent category from child categories during parent deletion", zap.Error(err))
			return err
		}

		return nil
	})
}

func NewProductCategoryService(repository irepository.GenericRepository[model.ProductCategory]) iservice.ProductCategoryService {
	return &productCategoryService{
		categoryRepository: repository,
	}
}

// isParentCategory checks if parentID is an ancestor of categoryID
// note: this function is not used checking the parent category existence!
func (c productCategoryService) isParentCategory(categoryID *uuid.UUID) (bool, error) {
	if categoryID == nil {
		return false, errors.New("category is nil")
	}
	ctx := context.Background()

	//find by condition to check each parent category
	found, err := c.categoryRepository.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("parent_category_id = ?", categoryID)
	},
		[]string{"ParentCategory"},
	)
	if err != nil {
		return false, err
	}

	//is this completely no relation
	found1, err := c.categoryRepository.GetByID(ctx, categoryID, []string{"ParentCategory"})
	if err != nil {
		return false, err
	}

	if found == nil || found1.ParentCategoryID == nil {
		return true, nil
	} else {
		return false, nil
	}
}
