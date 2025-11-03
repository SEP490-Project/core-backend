package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// AffiliateLinkService defines the business logic for affiliate link management
type AffiliateLinkService interface {
	// CreateOrGet creates a new affiliate link or returns an existing one with the same context
	// This is the primary method for automatic link generation
	CreateOrGet(ctx context.Context, req *requests.CreateAffiliateLinkRequest) (*responses.AffiliateLinkResponse, error)

	// GetByHash retrieves an affiliate link by its unique hash
	GetByHash(ctx context.Context, hash string) (*model.AffiliateLink, error)

	// GetByID retrieves an affiliate link by its ID with optional preloads
	GetByID(ctx context.Context, id uuid.UUID, includes []string) (*responses.AffiliateLinkResponse, error)

	// List retrieves affiliate links with filtering and pagination
	List(ctx context.Context, req *requests.GetAffiliateLinkRequest) (*responses.AffiliateLinkListResponse, error)

	// Update updates an affiliate link's status or tracking URL
	Update(ctx context.Context, id uuid.UUID, req *requests.UpdateAffiliateLinkRequest) (*responses.AffiliateLinkResponse, error)

	// Delete soft-deletes an affiliate link
	Delete(ctx context.Context, id uuid.UUID) error

	// ValidateTrackingLink checks if tracking URL exists in contract's ScopeOfWork
	ValidateTrackingLink(ctx context.Context, contractID uuid.UUID, trackingURL string) (bool, error)

	// MarkAsExpired marks affiliate links as expired (for cron job use)
	MarkAsExpired(ctx context.Context, ids []uuid.UUID) error

	// ValidateContractStatus checks if a contract is in ACTIVE status
	ValidateContractStatus(ctx context.Context, contractID uuid.UUID) error

	// ValidateContentStatus checks if content is in POSTED status
	ValidateContentStatus(ctx context.Context, contentID uuid.UUID) error

	// ValidateAffiliateLink performs comprehensive validation of link, contract, and content status
	ValidateAffiliateLink(ctx context.Context, link *model.AffiliateLink) error
}
