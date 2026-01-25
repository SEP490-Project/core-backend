package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// affiliateLinkRepository implements AffiliateLinkRepository interface
type affiliateLinkRepository struct {
	irepository.GenericRepository[model.AffiliateLink]
	db *gorm.DB
}

// NewAffiliateLinkRepository creates a new instance of AffiliateLinkRepository
func NewAffiliateLinkRepository(db *gorm.DB) irepository.AffiliateLinkRepository {
	return &affiliateLinkRepository{
		GenericRepository: NewGenericRepository[model.AffiliateLink](db),
		db:                db,
	}
}

// GetByHash retrieves an affiliate link by its unique hash
func (r *affiliateLinkRepository) GetByHash(ctx context.Context, hash string) (*model.AffiliateLink, error) {
	var link model.AffiliateLink
	err := r.db.WithContext(ctx).
		Where("hash = ?", hash).
		Where("deleted_at IS NULL").
		First(&link).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil instead of error for not found
		}
		return nil, err
	}

	return &link, nil
}

// GetByTrackingURLAndContext retrieves an affiliate link by tracking URL and context
func (r *affiliateLinkRepository) GetByTrackingURLAndContext(
	ctx context.Context,
	trackingURL string,
	contractID, contentID, channelID *uuid.UUID,
) (*model.AffiliateLink, error) {
	var link model.AffiliateLink
	query := r.db.WithContext(ctx).
		Where("tracking_url = ?", trackingURL).
		Where("deleted_at IS NULL")

	if contractID != nil {
		query = query.Where("contract_id = ?", *contractID)
	} else {
		query = query.Where("contract_id IS NULL")
	}

	if contentID != nil {
		query = query.Where("content_id = ?", *contentID)
	} else {
		query = query.Where("content_id IS NULL")
	}

	if channelID != nil {
		query = query.Where("channel_id = ?", *channelID)
	} else {
		query = query.Where("channel_id IS NULL")
	}

	err := query.First(&link).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil for not found (allow creation)
		}
		return nil, err
	}

	return &link, nil
}

// GetActiveLinks retrieves all active affiliate links
func (r *affiliateLinkRepository) GetActiveLinks(ctx context.Context, pageSize, pageNumber int) ([]model.AffiliateLink, int64, error) {
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", enum.AffiliateLinkStatusActive).
			Where("deleted_at IS NULL")
	}
	return r.GetAll(ctx, filter, nil, pageSize, pageNumber)
}

// GetByContract retrieves all affiliate links for a specific contract
func (r *affiliateLinkRepository) GetByContract(
	ctx context.Context,
	contractID uuid.UUID,
	includes []string,
	pageSize, pageNumber int,
) ([]model.AffiliateLink, int64, error) {
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", contractID).
			Where("deleted_at IS NULL")
	}
	return r.GetAll(ctx, filter, includes, pageSize, pageNumber)
}

// GetByContent retrieves all affiliate links for a specific content
func (r *affiliateLinkRepository) GetByContent(
	ctx context.Context,
	contentID uuid.UUID,
	includes []string,
	pageSize, pageNumber int,
) ([]model.AffiliateLink, int64, error) {
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("content_id = ?", contentID).
			Where("deleted_at IS NULL")
	}
	return r.GetAll(ctx, filter, includes, pageSize, pageNumber)
}

// GetByChannel retrieves all affiliate links for a specific channel
func (r *affiliateLinkRepository) GetByChannel(
	ctx context.Context,
	channelID uuid.UUID,
	includes []string,
	pageSize, pageNumber int,
) ([]model.AffiliateLink, int64, error) {
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("channel_id = ?", channelID).
			Where("deleted_at IS NULL")
	}
	return r.GetAll(ctx, filter, includes, pageSize, pageNumber)
}

// MarkAsExpired updates affiliate link status to 'expired'
func (r *affiliateLinkRepository) MarkAsExpired(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.AffiliateLink{}).
		Where("id = ?", id).
		Update("status", enum.AffiliateLinkStatusExpired).Error
}

// MarkAsInactive updates affiliate link status to 'inactive'
func (r *affiliateLinkRepository) MarkAsInactive(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.AffiliateLink{}).
		Where("id = ?", id).
		Update("status", enum.AffiliateLinkStatusInactive).Error
}

// BulkMarkAsExpired marks multiple affiliate links as expired
func (r *affiliateLinkRepository) BulkMarkAsExpired(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Model(&model.AffiliateLink{}).
		Where("id IN ?", ids).
		Update("status", enum.AffiliateLinkStatusExpired).Error
}

func (r *affiliateLinkRepository) GetIDsByCondition(ctx context.Context, filter func(*gorm.DB) *gorm.DB) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := r.db.WithContext(ctx).Model(new(model.AffiliateLink))
	if filter != nil {
		query = filter(query)
	}
	if err := query.Pluck("id", &ids).Error; err != nil {
		return nil, err
	}

	return ids, nil
}

// func (r *affiliateLinkRepository) Get
