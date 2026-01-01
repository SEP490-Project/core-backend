package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProductOptionRepository struct {
	db *gorm.DB
}

// NewProductOptionRepository creates a new ProductOptionRepository
func NewProductOptionRepository(db *gorm.DB) irepository.ProductOptionRepository {
	return &ProductOptionRepository{db: db}
}

// GetByType retrieves all product options of a specific type
func (r *ProductOptionRepository) GetByType(ctx context.Context, optionType model.ProductOptionType, activeOnly bool) ([]model.ProductOption, error) {
	var options []model.ProductOption
	query := r.db.WithContext(ctx).
		Where("type = ?", optionType).
		Where("deleted_at IS NULL")

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	err := query.Order("sort_order ASC, code ASC").Find(&options).Error
	if err != nil {
		zap.L().Error("ProductOptionRepository.GetByType failed",
			zap.String("type", string(optionType)),
			zap.Error(err))
		return nil, err
	}

	return options, nil
}

// GetByTypeAndCode retrieves a specific product option by type and code
func (r *ProductOptionRepository) GetByTypeAndCode(ctx context.Context, optionType model.ProductOptionType, code string) (*model.ProductOption, error) {
	var option model.ProductOption
	err := r.db.WithContext(ctx).
		Where("type = ? AND code = ? AND deleted_at IS NULL", optionType, code).
		First(&option).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		zap.L().Error("ProductOptionRepository.GetByTypeAndCode failed",
			zap.String("type", string(optionType)),
			zap.String("code", code),
			zap.Error(err))
		return nil, err
	}

	return &option, nil
}

// IsValidOption checks if a code is valid for the given option type
func (r *ProductOptionRepository) IsValidOption(ctx context.Context, optionType model.ProductOptionType, code string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ProductOption{}).
		Where("type = ? AND code = ? AND is_active = true AND deleted_at IS NULL", optionType, code).
		Count(&count).Error

	if err != nil {
		zap.L().Error("ProductOptionRepository.IsValidOption failed",
			zap.String("type", string(optionType)),
			zap.String("code", code),
			zap.Error(err))
		return false, err
	}

	return count > 0, nil
}

// GetAll retrieves all product options with optional filtering
func (r *ProductOptionRepository) GetAll(ctx context.Context, optionType *model.ProductOptionType, activeOnly bool, pageSize, pageNumber int) ([]model.ProductOption, int64, error) {
	var options []model.ProductOption
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ProductOption{}).Where("deleted_at IS NULL")

	if optionType != nil {
		query = query.Where("type = ?", *optionType)
	}

	if activeOnly {
		query = query.Where("is_active = ?", true)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("ProductOptionRepository.GetAll count failed", zap.Error(err))
		return nil, 0, err
	}

	// Apply pagination
	if pageSize > 0 {
		if pageSize > 100 {
			pageSize = 100
		}
		offset := (pageNumber - 1) * pageSize
		if offset < 0 {
			offset = 0
		}
		query = query.Limit(pageSize).Offset(offset)
	}

	err := query.Order("type ASC, sort_order ASC, code ASC").Find(&options).Error
	if err != nil {
		zap.L().Error("ProductOptionRepository.GetAll failed", zap.Error(err))
		return nil, 0, err
	}

	return options, total, nil
}

// GetByID retrieves a product option by ID
func (r *ProductOptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ProductOption, error) {
	var option model.ProductOption
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&option).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		zap.L().Error("ProductOptionRepository.GetByID failed",
			zap.String("id", id.String()),
			zap.Error(err))
		return nil, err
	}

	return &option, nil
}

// Create adds a new product option
func (r *ProductOptionRepository) Create(ctx context.Context, option *model.ProductOption) error {
	if err := r.db.WithContext(ctx).Create(option).Error; err != nil {
		zap.L().Error("ProductOptionRepository.Create failed", zap.Error(err))
		return err
	}
	return nil
}

// Update modifies an existing product option
func (r *ProductOptionRepository) Update(ctx context.Context, option *model.ProductOption) error {
	if err := r.db.WithContext(ctx).Save(option).Error; err != nil {
		zap.L().Error("ProductOptionRepository.Update failed", zap.Error(err))
		return err
	}
	return nil
}

// Delete soft deletes a product option
func (r *ProductOptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&model.ProductOption{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()"))

	if result.Error != nil {
		zap.L().Error("ProductOptionRepository.Delete failed",
			zap.String("id", id.String()),
			zap.Error(result.Error))
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("product option not found")
	}

	return nil
}
