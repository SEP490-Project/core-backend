package service

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/infrastructure/rabbitmq"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ClickTrackingService struct {
	affiliateLinkRepo        irepository.AffiliateLinkRepository
	affiliateLinkService     iservice.AffiliateLinkService
	cache                    *persistence.ValkeyCache
	rabbitmq                 *rabbitmq.RabbitMQ
	cacheHits                int64
	cacheMisses              int64
	affiliateLinkCachePrefix string
	affiliateLinkCacheTTL    time.Duration
}

func NewClickTrackingService(
	affiliateLinkRepo irepository.AffiliateLinkRepository,
	affiliateLinkService iservice.AffiliateLinkService,
	cache *persistence.ValkeyCache,
	rabbitmq *rabbitmq.RabbitMQ,
) iservice.ClickTrackingService {
	return &ClickTrackingService{
		affiliateLinkRepo:        affiliateLinkRepo,
		affiliateLinkService:     affiliateLinkService,
		cache:                    cache,
		rabbitmq:                 rabbitmq,
		cacheHits:                0,
		cacheMisses:              0,
		affiliateLinkCachePrefix: "affiliate:hash:",
		affiliateLinkCacheTTL:    24 * time.Hour,
	}
}

// ResolveHash resolves an affiliate link hash to its tracking URL using cache-first approach
func (s *ClickTrackingService) ResolveHash(ctx context.Context, hash string) (string, uuid.UUID, error) {
	startTime := time.Now()

	// Try cache first
	if s.cache != nil {
		trackingURL, err := s.getAffiliateLinkCache(hash)
		if err != nil {
			zap.L().Warn("Cache lookup failed, falling back to database",
				zap.String("hash", hash),
				zap.Error(err))
		} else if trackingURL != "" {
			// Cache hit
			s.cacheHits++
			zap.L().Debug("Cache hit for affiliate link",
				zap.String("hash", hash),
				zap.Duration("latency", time.Since(startTime)))

			// Still need to get the affiliate link ID from database
			// TODO: Consider caching ID as well in format "url|id"
			link, err := s.affiliateLinkRepo.GetByHash(ctx, hash)
			if err != nil {
				return "", uuid.Nil, err
			}
			return trackingURL, link.ID, nil
		}
	}

	// Cache miss - query database
	s.cacheMisses++
	zap.L().Debug("Cache miss for affiliate link, querying database", zap.String("hash", hash))

	link, err := s.affiliateLinkRepo.GetByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Affiliate link not found", zap.String("hash", hash))
			return "", uuid.Nil, errors.New("affiliate link not found")
		}
		zap.L().Error("Failed to resolve affiliate link hash",
			zap.String("hash", hash),
			zap.Error(err))
		return "", uuid.Nil, err
	}

	// Validate link status, contract status, and content status
	if err := s.affiliateLinkService.ValidateAffiliateLink(ctx, link); err != nil {
		zap.L().Warn("Affiliate link validation failed",
			zap.String("hash", hash),
			zap.String("status", string(link.Status)),
			zap.Error(err))
		return "", uuid.Nil, errors.New("affiliate link is inactive or expired")
	}

	// Cache the result for future lookups
	if s.cache != nil {
		if err := s.setAffiliateLinkCache(hash, link.TrackingURL); err != nil {
			zap.L().Warn("Failed to cache affiliate link",
				zap.String("hash", hash),
				zap.Error(err))
			// Don't fail the request if caching fails
		}
	}

	latency := time.Since(startTime)
	zap.L().Info("Resolved affiliate link hash",
		zap.String("hash", hash),
		zap.String("tracking_url", link.TrackingURL),
		zap.Duration("latency", latency))

	return link.TrackingURL, link.ID, nil
}

// LogClickAsync publishes a click event to RabbitMQ for async processing
func (s *ClickTrackingService) LogClickAsync(ctx context.Context, message *consumers.ClickEventMessage) error {
	if s.rabbitmq == nil {
		zap.L().Warn("RabbitMQ not configured, skipping click event logging")
		return nil // Graceful degradation - redirect still works
	}

	// Validate affiliate link before logging click event
	// Get affiliate link to perform validation
	link, err := s.affiliateLinkRepo.GetByID(ctx, message.AffiliateLinkID, []string{"Contract", "Content"})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Affiliate link not found, skipping click event",
				zap.String("affiliate_link_id", message.AffiliateLinkID.String()))
			return nil // Don't fail redirect, just skip logging
		}
		zap.L().Error("Failed to get affiliate link for validation", zap.Error(err))
		return err
	}

	// Validate link status, contract status, and content status
	if err = s.affiliateLinkService.ValidateAffiliateLink(ctx, link); err != nil {
		zap.L().Debug("Affiliate link validation failed, skipping click event",
			zap.String("affiliate_link_id", message.AffiliateLinkID.String()),
			zap.Error(err))
		return nil // Don't fail redirect, just skip logging
	}

	// Get producer for affiliate link clicks
	producer, err := s.rabbitmq.GetProducer("affiliate-link-click-producer")
	if err != nil {
		zap.L().Error("Failed to get RabbitMQ producer for click events", zap.Error(err))
		return err // Return error but don't fail the redirect
	}

	// Serialize message to JSON
	messageBytes, err := json.Marshal(message)
	if err != nil {
		zap.L().Error("Failed to serialize click event message", zap.Error(err))
		return err
	}

	// Publish to queue
	if err := producer.Publish(ctx, messageBytes); err != nil {
		zap.L().Error("Failed to publish click event to RabbitMQ",
			zap.String("affiliate_link_id", message.AffiliateLinkID.String()),
			zap.Error(err))
		return err
	}

	zap.L().Debug("Click event published to RabbitMQ",
		zap.String("affiliate_link_id", message.AffiliateLinkID.String()),
		zap.String("ip_address", message.IPAddress))

	return nil
}

// WarmCache preloads affiliate link hashes into cache
func (s *ClickTrackingService) WarmCache(ctx context.Context, linkIDs []uuid.UUID) error {
	if s.cache == nil {
		return errors.New("cache not configured")
	}

	if len(linkIDs) == 0 {
		return nil
	}

	zap.L().Info("Warming affiliate link cache", zap.Int("count", len(linkIDs)))

	// Fetch links from database
	links, _, err := s.affiliateLinkRepo.GetActiveLinks(ctx, 1000, 1) // Get up to 1000 active links
	if err != nil {
		zap.L().Error("Failed to fetch active links for cache warming", zap.Error(err))
		return err
	}

	// Build hash -> tracking URL map
	cacheData := make(map[string]string, len(links))
	for _, link := range links {
		cacheData[link.Hash] = link.TrackingURL
	}

	// Warm cache in batch
	if err := s.warmAffiliateLinkCache(cacheData); err != nil {
		zap.L().Error("Failed to warm affiliate link cache", zap.Error(err))
		return err
	}

	zap.L().Info("Affiliate link cache warmed successfully", zap.Int("count", len(cacheData)))
	return nil
}

// GetCacheStats returns cache hit/miss statistics
func (s *ClickTrackingService) GetCacheStats(ctx context.Context) (hits int64, misses int64, hitRate float64, err error) {
	hits = s.cacheHits
	misses = s.cacheMisses
	total := hits + misses

	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100.0
	}

	return hits, misses, hitRate, nil
}

// region: =============== Cache Helper Methods ===============

// setAffiliateLinkCache stores an affiliate link's tracking URL by hash
// Key format: affiliate:hash:{hash} → tracking_url
func (s *ClickTrackingService) setAffiliateLinkCache(hash string, trackingURL string) error {
	key := s.affiliateLinkCachePrefix + hash
	zap.L().Debug("Caching affiliate link",
		zap.String("hash", hash),
		zap.String("key", key),
		zap.Duration("ttl", s.affiliateLinkCacheTTL))

	return s.cache.Set(key, trackingURL, s.affiliateLinkCacheTTL)
}

// GetAffiliateLinkCache retrieves an affiliate link's tracking URL by hash
// Returns empty string if not found (cache miss)
func (s *ClickTrackingService) getAffiliateLinkCache(hash string) (string, error) {
	key := s.affiliateLinkCachePrefix + hash
	zap.L().Debug("Looking up affiliate link in cache", zap.String("hash", hash), zap.String("key", key))

	trackingURL, valueType, err := s.cache.Get(key)
	if err != nil {
		return "", err
	}

	if trackingURL == "" {
		zap.L().Debug("Affiliate link cache miss", zap.String("hash", hash))
		return "", nil
	}

	if valueType != "string" {
		zap.L().Warn("Affiliate link cache hit with unexpected value type", zap.String("hash", hash), zap.String("value_type", valueType))
		return "", nil
	}
	trackingURLStr, ok := trackingURL.(string)
	if !ok {
		zap.L().Warn("Affiliate link cache hit with unexpected value type", zap.String("hash", hash), zap.String("value_type", valueType))
		return "", nil
	}
	zap.L().Debug("Affiliate link cache hit", zap.String("hash", hash), zap.String("tracking_url", trackingURLStr))
	return trackingURLStr, nil
}

// deleteAffiliateLinkCache removes an affiliate link from cache
func (s *ClickTrackingService) deleteAffiliateLinkCache(hash string) error {
	key := s.affiliateLinkCachePrefix + hash
	zap.L().Debug("Deleting affiliate link from cache", zap.String("hash", hash), zap.String("key", key))
	return s.cache.Delete(key)
}

// WarmAffiliateLinkCache preloads multiple affiliate links into cache
// Used for cache warming of popular or recently created links
func (s *ClickTrackingService) warmAffiliateLinkCache(links map[string]string) error {
	if len(links) == 0 {
		return nil
	}

	zap.L().Info("Warming affiliate link cache", zap.Int("count", len(links)))

	// Use pipeline for batch operations
	client := s.cache.GetClient()
	clientContext := s.cache.GetContext()
	pipe := client.Pipeline()
	for hash, trackingURL := range links {
		key := s.affiliateLinkCachePrefix + hash
		pipe.Set(clientContext, key, trackingURL, s.affiliateLinkCacheTTL)
	}

	_, err := pipe.Exec(clientContext)
	if err != nil {
		zap.L().Error("Failed to warm affiliate link cache", zap.Error(err))
		return err
	}

	zap.L().Info("Affiliate link cache warming completed", zap.Int("count", len(links)))
	return nil
}

// GetAffiliateLinkCacheStats returns cache statistics for affiliate links
func (s *ClickTrackingService) GetAffiliateLinkCacheStats() (total int64, err error) {
	pattern := s.affiliateLinkCachePrefix + "*"
	keys, err := s.cache.Keys(pattern)
	if err != nil {
		return 0, err
	}
	return int64(len(keys)), nil
}

// endregion
