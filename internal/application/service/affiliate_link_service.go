package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type affiliateLinkService struct {
	affiliateLinkRepo irepository.AffiliateLinkRepository
	contractRepo      irepository.GenericRepository[model.Contract]
	contentRepo       irepository.GenericRepository[model.Content]
	channelRepo       irepository.GenericRepository[model.Channel]
	baseURL           string // Base URL for short links (e.g., "https://domain.com")
}

func NewAffiliateLinkService(
	affiliateLinkRepo irepository.AffiliateLinkRepository,
	contractRepo irepository.GenericRepository[model.Contract],
	contentRepo irepository.GenericRepository[model.Content],
	channelRepo irepository.GenericRepository[model.Channel],
	baseURL string,
) iservice.AffiliateLinkService {
	return &affiliateLinkService{
		affiliateLinkRepo: affiliateLinkRepo,
		contractRepo:      contractRepo,
		contentRepo:       contentRepo,
		channelRepo:       channelRepo,
		baseURL:           strings.TrimSuffix(baseURL, "/"),
	}
}

// CreateOrGet creates a new affiliate link or returns an existing one
func (s *affiliateLinkService) CreateOrGet(ctx context.Context, req *requests.CreateAffiliateLinkRequest) (*responses.AffiliateLinkResponse, error) {
	startTime := time.Now()

	zap.L().Debug("CreateOrGet affiliate link started",
		zap.String("contract_id", req.ContractID.String()),
		zap.String("content_id", req.ContentID.String()),
		zap.String("channel_id", req.ChannelID.String()),
		zap.String("tracking_url", req.TrackingURL))

	// Validate that contract, content, and channel exist
	if err := s.validateReferences(ctx, req.ContractID, req.ContentID, req.ChannelID); err != nil {
		zap.L().Error("Affiliate link reference validation failed",
			zap.Error(err),
			zap.String("contract_id", req.ContractID.String()),
			zap.String("content_id", req.ContentID.String()),
			zap.String("channel_id", req.ChannelID.String()))
		return nil, err
	}

	// Check if affiliate link already exists for this combination
	existing, err := s.affiliateLinkRepo.GetByTrackingURLAndContext(
		ctx,
		req.TrackingURL,
		req.ContractID,
		req.ContentID,
		req.ChannelID,
	)

	if err == nil && existing != nil {
		// Link already exists, return it
		duration := time.Since(startTime)
		zap.L().Info("Affiliate link already exists, returning existing",
			zap.String("hash", existing.Hash),
			zap.String("link_id", existing.ID.String()),
			zap.String("contract_id", req.ContractID.String()),
			zap.String("content_id", req.ContentID.String()),
			zap.String("channel_id", req.ChannelID.String()),
			zap.Duration("duration_ms", duration),
			zap.String("operation", "affiliate_link_create_or_get"))
		return responses.AffiliateLinkResponse{}.ToResponse(existing, s.baseURL), nil
	}

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Error("Error checking for existing affiliate link",
			zap.Error(err),
			zap.String("tracking_url", req.TrackingURL))
		return nil, fmt.Errorf("failed to check existing link: %w", err)
	}

	// Generate unique hash
	hash := s.generateHash(req.TrackingURL, req.ContractID, req.ContentID, req.ChannelID)

	// Create new affiliate link
	link := &model.AffiliateLink{
		Hash:        hash,
		ContractID:  req.ContractID,
		ContentID:   req.ContentID,
		ChannelID:   req.ChannelID,
		TrackingURL: req.TrackingURL,
		Status:      enum.AffiliateLinkStatusActive,
	}

	if err := s.affiliateLinkRepo.Add(ctx, link); err != nil {
		zap.L().Error("Failed to create affiliate link",
			zap.Error(err),
			zap.String("hash", hash),
			zap.String("tracking_url", req.TrackingURL))
		return nil, fmt.Errorf("failed to create affiliate link: %w", err)
	}

	duration := time.Since(startTime)
	zap.L().Info("Created new affiliate link",
		zap.String("hash", link.Hash),
		zap.String("link_id", link.ID.String()),
		zap.String("contract_id", req.ContractID.String()),
		zap.String("content_id", req.ContentID.String()),
		zap.String("channel_id", req.ChannelID.String()),
		zap.String("tracking_url", req.TrackingURL),
		zap.Duration("duration_ms", duration),
		zap.String("operation", "affiliate_link_create"))

	return responses.AffiliateLinkResponse{}.ToResponse(link, s.baseURL), nil
}

// GetByHash retrieves an affiliate link by its hash
func (s *affiliateLinkService) GetByHash(ctx context.Context, hash string) (*model.AffiliateLink, error) {
	link, err := s.affiliateLinkRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("affiliate link not found: %s", hash)
		}
		return nil, err
	}
	return link, nil
}

// GetByID retrieves an affiliate link by its ID
func (s *affiliateLinkService) GetByID(ctx context.Context, id uuid.UUID, includes []string) (*responses.AffiliateLinkResponse, error) {
	link, err := s.affiliateLinkRepo.GetByID(ctx, id, includes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("affiliate link not found")
		}
		return nil, err
	}
	return responses.AffiliateLinkResponse{}.ToResponse(link, s.baseURL), nil
}

// List retrieves affiliate links with filtering
func (s *affiliateLinkService) List(ctx context.Context, req *requests.GetAffiliateLinkRequest) (*responses.AffiliateLinkListResponse, error) {
	pageSize := req.PageSize
	if pageSize == 0 {
		pageSize = 20 // Default page size
	}
	pageNumber := req.PageNumber
	if pageNumber == 0 {
		pageNumber = 1
	}

	var links []model.AffiliateLink
	var total int64
	var err error

	// Apply filters based on request
	if req.ContractID != nil {
		links, total, err = s.affiliateLinkRepo.GetByContract(ctx, *req.ContractID, []string{"Contract", "Content", "Channel"}, pageSize, pageNumber)
	} else if req.ContentID != nil {
		links, total, err = s.affiliateLinkRepo.GetByContent(ctx, *req.ContentID, []string{"Contract", "Content", "Channel"}, pageSize, pageNumber)
	} else if req.ChannelID != nil {
		links, total, err = s.affiliateLinkRepo.GetByChannel(ctx, *req.ChannelID, []string{"Contract", "Content", "Channel"}, pageSize, pageNumber)
	} else {
		// Get all with optional status filter
		links, total, err = s.affiliateLinkRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
			query := db.Where("deleted_at IS NULL")
			if req.Status != nil {
				query = query.Where("status = ?", *req.Status)
			}
			return query
		}, []string{"Contract", "Content", "Channel"}, pageSize, pageNumber)
	}

	if err != nil {
		return nil, err
	}

	// Convert to response DTOs
	linkResponses := make([]responses.AffiliateLinkResponse, len(links))
	for i, link := range links {
		linkResponses[i] = *responses.AffiliateLinkResponse{}.ToResponse(&link, s.baseURL)
	}

	return &responses.AffiliateLinkListResponse{
		Links: linkResponses,
		Pagination: responses.Pagination{
			Page:  pageNumber,
			Limit: pageSize,
			Total: total,
		},
	}, nil
}

// Update updates an affiliate link
func (s *affiliateLinkService) Update(ctx context.Context, id uuid.UUID, req *requests.UpdateAffiliateLinkRequest) (*responses.AffiliateLinkResponse, error) {
	link, err := s.affiliateLinkRepo.GetByID(ctx, id, nil)
	if err != nil {
		return nil, fmt.Errorf("affiliate link not found")
	}

	// Update fields if provided
	if req.Status != nil {
		link.Status = enum.AffiliateLinkStatus(*req.Status)
	}
	if req.TrackingURL != nil {
		link.TrackingURL = *req.TrackingURL
	}

	if err := s.affiliateLinkRepo.Update(ctx, link); err != nil {
		return nil, fmt.Errorf("failed to update affiliate link: %w", err)
	}

	return responses.AffiliateLinkResponse{}.ToResponse(link, s.baseURL), nil
}

// Delete soft-deletes an affiliate link
func (s *affiliateLinkService) Delete(ctx context.Context, id uuid.UUID) error {
	link, err := s.affiliateLinkRepo.GetByID(ctx, id, nil)
	if err != nil {
		return fmt.Errorf("affiliate link not found")
	}

	return s.affiliateLinkRepo.Delete(ctx, link)
}

// ValidateTrackingLink checks if tracking URL exists in contract's ScopeOfWork
func (s *affiliateLinkService) ValidateTrackingLink(ctx context.Context, contractID uuid.UUID, trackingURL string) (bool, error) {
	contract, err := s.contractRepo.GetByID(ctx, contractID, nil)
	if err != nil {
		return false, fmt.Errorf("contract not found")
	}

	// Parse ScopeOfWork JSONB to check for TrackingLink
	var scopeOfWork map[string]any
	if err := json.Unmarshal(contract.ScopeOfWork, &scopeOfWork); err != nil {
		zap.L().Warn("Failed to parse ScopeOfWork JSONB", zap.Error(err))
		return false, nil
	}

	// Check if TrackingLink field exists and matches
	if trackingLink, ok := scopeOfWork["TrackingLink"].(string); ok {
		return trackingLink == trackingURL, nil
	}

	return false, nil
}

// MarkAsExpired marks multiple affiliate links as expired
func (s *affiliateLinkService) MarkAsExpired(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return s.affiliateLinkRepo.BulkMarkAsExpired(ctx, ids)
}

// region: ============= Helpers methods =============

// generateHash creates a unique base62-encoded SHA-256 hash (16 chars)
func (s *affiliateLinkService) generateHash(trackingURL string, contractID, contentID, channelID uuid.UUID) string {
	// Combine inputs for hashing
	input := fmt.Sprintf("%s|%s|%s|%s", trackingURL, contractID.String(), contentID.String(), channelID.String())

	// Compute SHA-256 hash
	hash := sha256.Sum256([]byte(input))

	// Encode to base64 and take first 16 characters (URL-safe)
	encoded := base64.URLEncoding.EncodeToString(hash[:])

	// Remove padding and special characters for clean URLs
	encoded = strings.ReplaceAll(encoded, "+", "")
	encoded = strings.ReplaceAll(encoded, "/", "")
	encoded = strings.ReplaceAll(encoded, "=", "")

	// Take first 16 characters
	if len(encoded) > 16 {
		encoded = encoded[:16]
	}

	return encoded
}

func (s *affiliateLinkService) validateReferences(ctx context.Context, contractID, contentID, channelID uuid.UUID) error {
	// Check contract exists
	if _, err := s.contractRepo.GetByID(ctx, contractID, nil); err != nil {
		return fmt.Errorf("contract not found: %s", contractID)
	}

	// Check content exists
	if _, err := s.contentRepo.GetByID(ctx, contentID, nil); err != nil {
		return fmt.Errorf("content not found: %s", contentID)
	}

	// Check channel exists
	if _, err := s.channelRepo.GetByID(ctx, channelID, nil); err != nil {
		return fmt.Errorf("channel not found: %s", channelID)
	}

	return nil
}

// ValidateContractStatus checks if the contract is still active for the affiliate link
func (s *affiliateLinkService) ValidateContractStatus(ctx context.Context, contractID uuid.UUID) error {
	contract, err := s.contractRepo.GetByID(ctx, contractID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("contract not found")
		}
		return fmt.Errorf("failed to get contract: %w", err)
	}

	// Check if contract is in active status
	if contract.Status != enum.ContractStatusActive {
		return fmt.Errorf("contract is not active: status is %s", contract.Status)
	}

	return nil
}

// ValidateContentStatus checks if the content is published and active
func (s *affiliateLinkService) ValidateContentStatus(ctx context.Context, contentID uuid.UUID) error {
	content, err := s.contentRepo.GetByID(ctx, contentID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("content not found")
		}
		return fmt.Errorf("failed to get content: %w", err)
	}

	// Check if content is in posted status (published)
	if content.Status != enum.ContentStatusPosted {
		return fmt.Errorf("content is not published: status is %s", content.Status)
	}

	return nil
}

// ValidateAffiliateLink performs comprehensive validation of an affiliate link
// Returns specific error types for different validation failures
func (s *affiliateLinkService) ValidateAffiliateLink(ctx context.Context, link *model.AffiliateLink) error {
	// Check if link itself is active
	if link.Status != enum.AffiliateLinkStatusActive {
		return fmt.Errorf("affiliate link is %s", link.Status)
	}

	// Validate contract status
	if err := s.ValidateContractStatus(ctx, link.ContractID); err != nil {
		zap.L().Debug("Contract validation failed",
			zap.String("link_id", link.ID.String()),
			zap.String("contract_id", link.ContractID.String()),
			zap.Error(err))
		return fmt.Errorf("contract validation failed: %w", err)
	}

	// Validate content status
	if err := s.ValidateContentStatus(ctx, link.ContentID); err != nil {
		zap.L().Debug("Content validation failed",
			zap.String("link_id", link.ID.String()),
			zap.String("content_id", link.ContentID.String()),
			zap.Error(err))
		return fmt.Errorf("content validation failed: %w", err)
	}

	return nil
}

// endregion
