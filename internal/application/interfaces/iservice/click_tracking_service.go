package iservice

import (
	"context"
	"core-backend/internal/application/dto/consumers"

	"github.com/google/uuid"
)

// ClickTrackingService handles affiliate link resolution and click event logging
type ClickTrackingService interface {
	// ResolveHash resolves an affiliate link hash to its tracking URL
	// Uses cache-first approach with database fallback for optimal performance
	// Returns (trackingURL, affiliateLinkID, error)
	ResolveHash(ctx context.Context, hash string) (string, uuid.UUID, error)

	// LogClickAsync publishes a click event message to RabbitMQ for async processing
	// Non-blocking operation - failures are logged but don't affect redirect
	LogClickAsync(ctx context.Context, message *consumers.ClickEventMessage) error

	// WarmCache preloads affiliate link hashes into Valkey cache
	// Used for cache warming of active/popular links
	WarmCache(ctx context.Context, linkIDs []uuid.UUID) error

	// GetCacheStats returns cache hit/miss statistics for monitoring
	GetCacheStats(ctx context.Context) (hits int64, misses int64, hitRate float64, err error)
}
