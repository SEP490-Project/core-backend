package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// AffiliateLinkRepository defines the interface for affiliate link data access
type AffiliateLinkRepository interface {
	// Embed generic repository for standard CRUD operations
	GenericRepository[model.AffiliateLink]

	// GetByHash retrieves an affiliate link by its unique hash
	GetByHash(ctx context.Context, hash string) (*model.AffiliateLink, error)

	// GetByTrackingURLAndContext retrieves an affiliate link by tracking URL and context (contract+content+channel)
	GetByTrackingURLAndContext(ctx context.Context, trackingURL string, contractID, contentID, channelID *uuid.UUID) (*model.AffiliateLink, error)

	// GetActiveLinks retrieves all active affiliate links (status = 'active' and not soft-deleted)
	GetActiveLinks(ctx context.Context, pageSize, pageNumber int) ([]model.AffiliateLink, int64, error)

	// GetByContract retrieves all affiliate links for a specific contract
	GetByContract(ctx context.Context, contractID uuid.UUID, includes []string, pageSize, pageNumber int) ([]model.AffiliateLink, int64, error)

	// GetByContent retrieves all affiliate links for a specific content
	GetByContent(ctx context.Context, contentID uuid.UUID, includes []string, pageSize, pageNumber int) ([]model.AffiliateLink, int64, error)

	// GetByChannel retrieves all affiliate links for a specific channel
	GetByChannel(ctx context.Context, channelID uuid.UUID, includes []string, pageSize, pageNumber int) ([]model.AffiliateLink, int64, error)

	// MarkAsExpired updates affiliate link status to 'expired'
	MarkAsExpired(ctx context.Context, id uuid.UUID) error

	// MarkAsInactive updates affiliate link status to 'inactive'
	MarkAsInactive(ctx context.Context, id uuid.UUID) error

	// BulkMarkAsExpired marks multiple affiliate links as expired
	BulkMarkAsExpired(ctx context.Context, ids []uuid.UUID) error
}
