package consumer

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/persistence"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ContentViewConsumer handles content view events from RabbitMQ
type ContentViewConsumer struct {
	kpiMetricsRepo     irepository.KPIMetricsRepository
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	cache              *persistence.ValkeyCache
	adminConfig        *config.AdminConfig
}

// NewContentViewConsumer creates a new content view consumer
func NewContentViewConsumer(
	kpiMetricsRepo irepository.KPIMetricsRepository,
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	cache *persistence.ValkeyCache,
	adminConfig *config.AdminConfig,
) *ContentViewConsumer {
	return &ContentViewConsumer{
		kpiMetricsRepo:     kpiMetricsRepo,
		contentChannelRepo: contentChannelRepo,
		cache:              cache,
		adminConfig:        adminConfig,
	}
}

// Handle processes content view messages from RabbitMQ
func (c *ContentViewConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Debug("Processing content view message")

	// Unmarshal message
	var message consumers.ContentViewMessage
	if err := json.Unmarshal(body, &message); err != nil {
		zap.L().Error("Failed to unmarshal content view message",
			zap.Error(err),
			zap.String("body", string(body)))
		return err
	}

	// Validate required fields
	if err := c.validateMessage(&message); err != nil {
		zap.L().Error("Invalid content view message", zap.Error(err))
		return err
	}

	// Skip bot views
	if message.IsBot {
		zap.L().Debug("Skipping bot view",
			zap.String("content_channel_id", message.ContentChannelID.String()),
			zap.String("user_agent", message.UserAgent))
		return nil // Don't requeue, just skip
	}

	// Always record view count
	viewMetric := &model.KPIMetrics{
		ReferenceID:   message.ContentChannelID,
		ReferenceType: enum.KPIReferenceTypeContentChannel,
		Type:          enum.KPIValueTypeViews,
		Value:         1,
		RecordedDate:  message.ViewedAt,
	}
	if err := c.kpiMetricsRepo.Add(ctx, viewMetric); err != nil {
		zap.L().Error("Failed to record view metric",
			zap.String("content_channel_id", message.ContentChannelID.String()),
			zap.Error(err))
		return err
	}

	// Update ContentChannel.Metrics (fetched and mapped)
	if err := c.incrementContentChannelViews(ctx, message.ContentChannelID, 1, false); err != nil {
		zap.L().Error("Failed to update content channel metrics",
			zap.String("content_channel_id", message.ContentChannelID.String()),
			zap.Error(err))
		// Don't fail the operation, just log error
	}

	// Check if this is a unique view
	isUnique := c.isUniqueView(ctx, &message)
	if isUnique {
		// Record unique view
		uniqueViewMetric := &model.KPIMetrics{
			ReferenceID:   message.ContentChannelID,
			ReferenceType: enum.KPIReferenceTypeContentChannel,
			Type:          enum.KPIValueTypeUniqueViews,
			Value:         1,
			RecordedDate:  message.ViewedAt,
		}
		if err := c.kpiMetricsRepo.Add(ctx, uniqueViewMetric); err != nil {
			zap.L().Error("Failed to record unique view metric",
				zap.String("content_channel_id", message.ContentChannelID.String()),
				zap.Error(err))
			// Don't fail the entire operation for unique view tracking
		}

		// Update ContentChannel.Metrics for unique view
		if err := c.incrementContentChannelViews(ctx, message.ContentChannelID, 1, true); err != nil {
			zap.L().Error("Failed to update content channel unique views",
				zap.String("content_channel_id", message.ContentChannelID.String()),
				zap.Error(err))
		}
	}

	zap.L().Info("Content view recorded successfully",
		zap.String("content_channel_id", message.ContentChannelID.String()),
		zap.String("content_id", message.ContentID.String()),
		zap.Bool("is_unique", isUnique),
		zap.Bool("authenticated", message.UserID != nil))

	return nil
}

// isUniqueView checks if this is a unique view using Valkey cache for deduplication
func (c *ContentViewConsumer) isUniqueView(ctx context.Context, event *consumers.ContentViewMessage) bool {
	if c.cache == nil {
		// If cache is not available, treat as unique
		return true
	}

	viewerKey := c.getViewerKey(event)
	cacheKey := fmt.Sprintf("content_view:%s:%s", event.ContentChannelID.String(), viewerKey)

	// Get TTL from AdminConfig (default 24 hours)
	ttlHours := c.adminConfig.ContentViewUniqueCacheTTLHours
	if ttlHours <= 0 {
		ttlHours = 24
	}
	ttl := time.Duration(ttlHours) * time.Hour

	// Check if already viewed within TTL window
	existing, err := c.cache.Get(cacheKey)
	if err != nil {
		zap.L().Warn("Failed to check cache for unique view",
			zap.String("cache_key", cacheKey),
			zap.Error(err))
		return true // Treat as unique on error
	}

	if existing != "" {
		// Already viewed within TTL window
		return false
	}

	// Mark as viewed in cache
	if err := c.cache.Set(cacheKey, "1", ttl); err != nil {
		zap.L().Warn("Failed to set cache for unique view",
			zap.String("cache_key", cacheKey),
			zap.Error(err))
	}

	return true
}

// getViewerKey generates a unique key for the viewer (UserID or hashed IP+UA)
func (c *ContentViewConsumer) getViewerKey(event *consumers.ContentViewMessage) string {
	// If authenticated, use UserID
	if event.UserID != nil && *event.UserID != uuid.Nil {
		return event.UserID.String()
	}

	// For anonymous users, hash IP + UserAgent for privacy
	data := event.IPAddress + "|" + event.UserAgent
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:16]) // Use first 16 bytes (32 hex chars)
}

// validateMessage performs basic validation on the content view message
func (c *ContentViewConsumer) validateMessage(message *consumers.ContentViewMessage) error {
	if message.ContentChannelID == uuid.Nil {
		return errors.New("content_channel_id is required")
	}

	if message.ContentID == uuid.Nil {
		return errors.New("content_id is required")
	}

	if message.IPAddress == "" {
		return errors.New("ip_address is required")
	}

	if message.UserAgent == "" {
		return errors.New("user_agent is required")
	}

	if message.ViewedAt.IsZero() {
		return errors.New("viewed_at timestamp is required")
	}

	return nil
}

// incrementContentChannelViews updates the views count in ContentChannel.Metrics
func (c *ContentViewConsumer) incrementContentChannelViews(ctx context.Context, contentChannelID uuid.UUID, count int64, isUnique bool) error {
	// Fetch content channel
	cc, err := c.contentChannelRepo.GetByID(ctx, contentChannelID, nil)
	if err != nil {
		return err
	}

	// Initialize Metrics if nil
	if cc.Metrics == nil {
		cc.Metrics = &model.ContentChannelMetrics{
			CurrentFetched: make(map[string]any),
			CurrentMapped:  make(map[enum.KPIValueType]float64),
		}
	}
	if cc.Metrics.CurrentFetched == nil {
		cc.Metrics.CurrentFetched = make(map[string]any)
	}
	if cc.Metrics.CurrentMapped == nil {
		cc.Metrics.CurrentMapped = make(map[enum.KPIValueType]float64)
	}

	// Update Mapped Metrics
	if isUnique {
		cc.Metrics.CurrentMapped[enum.KPIValueTypeUniqueViews] += float64(count)
	} else {
		cc.Metrics.CurrentMapped[enum.KPIValueTypeViews] += float64(count)

		// Update Fetched Metrics (Website specific format)
		// views_count is stored as float64 in JSON unmarshaling usually, but we set it as int64 or float64
		var currentViews float64
		if v, ok := cc.Metrics.CurrentFetched["views_count"]; ok {
			switch val := v.(type) {
			case float64:
				currentViews = val
			case int64:
				currentViews = float64(val)
			case int:
				currentViews = float64(val)
			}
		}
		cc.Metrics.CurrentFetched["views_count"] = currentViews + float64(count)
	}

	// Save updates
	return c.contentChannelRepo.Update(ctx, cc)
}
