package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/infrastructure/persistence"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CacheHandler handles Cache (Valkey/Redis) management API endpoints
type CacheHandler struct {
	valkeyCache *persistence.ValkeyCache
}

// NewCacheHandler creates a new Cache handler
func NewCacheHandler(valkeyCache *persistence.ValkeyCache) *CacheHandler {
	return &CacheHandler{
		valkeyCache: valkeyCache,
	}
}

// GetOverview godoc
//
//	@Summary		Get Cache Overview
//	@Description	Returns cache statistics and connection info
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.CacheOverviewResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/overview [get]
func (h *CacheHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()

	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	// Ping to check connection
	err := h.valkeyCache.Ping()
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Cache connection failed: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Get cache info
	infoResult, err := h.valkeyCache.GetClient().Info(ctx).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get cache info: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Parse info result
	info := parseRedisInfo(infoResult)

	// Get database size
	dbSize, err := h.valkeyCache.GetClient().DBSize(ctx).Result()
	if err != nil {
		zap.L().Warn("Failed to get database size", zap.Error(err))
		dbSize = 0
	}

	// Get memory stats
	memoryStats, err := h.valkeyCache.GetClient().Info(ctx, "memory").Result()
	if err != nil {
		zap.L().Warn("Failed to get memory stats", zap.Error(err))
	}
	memoryInfo := parseRedisInfo(memoryStats)

	overview := responses.CacheOverviewResponse{
		Status:                   "connected",
		Version:                  info["redis_version"],
		Uptime:                   info["uptime_in_seconds"],
		ConnectedClients:         info["connected_clients"],
		UsedMemory:               memoryInfo["used_memory_human"],
		UsedMemoryPeak:           memoryInfo["used_memory_peak_human"],
		TotalKeys:                dbSize,
		TotalConnectionsReceived: info["total_connections_received"],
		TotalCommandsProcessed:   info["total_commands_processed"],
		KeyspaceHits:             info["keyspace_hits"],
		KeyspaceMisses:           info["keyspace_misses"],
		EvictedKeys:              info["evicted_keys"],
		ExpiredKeys:              info["expired_keys"],
	}

	// Calculate hit rate
	if hits, hitsOk := parseInfoInt(info["keyspace_hits"]); hitsOk {
		if misses, missesOk := parseInfoInt(info["keyspace_misses"]); missesOk {
			total := hits + misses
			if total > 0 {
				overview.HitRate = fmt.Sprintf("%.2f%%", float64(hits)/float64(total)*100)
			}
		}
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Cache overview retrieved successfully", nil, overview))
}

// GetKeys godoc
//
//	@Summary		List Cache Keys
//	@Description	Returns cache keys matching a pattern
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Param			pattern	query		string	false	"Key pattern (* for all, prefix:* for prefix match)"	default(*)
//	@Param			limit	query		int		false	"Limit number of results"								default(100)
//	@Success		200		{object}	responses.APIResponse{data=responses.CacheKeysResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/keys [get]
func (h *CacheHandler) GetKeys(c *gin.Context) {
	ctx := c.Request.Context()

	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	pattern := c.DefaultQuery("pattern", "*")
	limit := c.DefaultQuery("limit", "100")

	var limitInt int64 = 100
	fmt.Sscanf(limit, "%d", &limitInt)

	// Use SCAN instead of KEYS for better performance
	var keys []string
	var cursor uint64
	var count int64 = 0

	for {
		var scanKeys []string
		var err error

		scanKeys, cursor, err = h.valkeyCache.GetClient().Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to scan keys: "+err.Error(), http.StatusInternalServerError))
			return
		}

		keys = append(keys, scanKeys...)
		count += int64(len(scanKeys))

		// Check limit
		if count >= limitInt || cursor == 0 {
			break
		}
	}

	// Truncate to limit
	if len(keys) > int(limitInt) {
		keys = keys[:limitInt]
	}

	// Get details for each key (type, TTL)
	var keyDetails []responses.CacheKeyInfo
	for _, key := range keys {
		keyType, err := h.valkeyCache.GetClient().Type(ctx, key).Result()
		if err != nil {
			zap.L().Warn("Failed to get key type", zap.String("key", key), zap.Error(err))
			continue
		}

		ttl, err := h.valkeyCache.GetClient().TTL(ctx, key).Result()
		if err != nil {
			zap.L().Warn("Failed to get key TTL", zap.String("key", key), zap.Error(err))
			ttl = -1
		}

		var ttlSeconds int64 = -1
		if ttl > 0 {
			ttlSeconds = int64(ttl.Seconds())
		}

		keyDetails = append(keyDetails, responses.CacheKeyInfo{
			Key:  key,
			Type: keyType,
			TTL:  ttlSeconds,
		})
	}

	result := responses.CacheKeysResponse{
		Keys:       keyDetails,
		TotalCount: len(keyDetails),
		Pattern:    pattern,
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Cache keys retrieved successfully", nil, result))
}

// GetKey godoc
//
//	@Summary		Get Cache Key Value
//	@Description	Returns the value of a specific cache key
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string	true	"Cache key"
//	@Success		200	{object}	responses.APIResponse{data=responses.CacheKeyValueResponse}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/keys/{key} [get]
func (h *CacheHandler) GetKey(c *gin.Context) {
	ctx := c.Request.Context()

	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("key parameter is required", http.StatusBadRequest))
		return
	}

	// Check if key exists
	exists, err := h.valkeyCache.GetClient().Exists(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to check key existence: "+err.Error(), http.StatusInternalServerError))
		return
	}
	if exists == 0 {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Key not found", http.StatusNotFound))
		return
	}

	// Get TTL
	ttl, err := h.valkeyCache.GetClient().TTL(ctx, key).Result()
	if err != nil {
		ttl = -1
	}

	var ttlSeconds int64 = -1
	if ttl > 0 {
		ttlSeconds = int64(ttl.Seconds())
	}

	var value any
	var valueType string
	if value, valueType, err = h.valkeyCache.Get(key); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get key value: "+err.Error(), http.StatusInternalServerError))
		return
	}

	result := responses.CacheKeyValueResponse{
		Key:   key,
		Type:  valueType,
		Value: value,
		TTL:   ttlSeconds,
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Cache key value retrieved successfully", nil, result))
}

// DeleteKey godoc
//
//	@Summary		Delete Cache Key
//	@Description	Deletes one or more cache keys
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CacheDeleteKeyRequest	true	"Delete key request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/keys [delete]
func (h *CacheHandler) DeleteKey(c *gin.Context) {
	ctx := c.Request.Context()

	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.CacheDeleteKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	if len(req.Keys) == 0 {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("At least one key is required", http.StatusBadRequest))
		return
	}

	deleted, err := h.valkeyCache.GetClient().Del(ctx, req.Keys...).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to delete keys: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Cache keys deleted",
		zap.Int64("count", deleted),
		zap.Strings("keys", req.Keys))

	c.JSON(http.StatusOK, responses.SuccessResponse(fmt.Sprintf("Deleted %d key(s)", deleted), nil, map[string]int64{"deleted_count": deleted}))
}

// DeleteByPattern godoc
//
//	@Summary		Delete Cache Keys by Pattern
//	@Description	Deletes all cache keys matching a pattern
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CacheDeletePatternRequest	true	"Delete pattern request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/keys/by-pattern [delete]
func (h *CacheHandler) DeleteByPattern(c *gin.Context) {
	ctx := c.Request.Context()

	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.CacheDeletePatternRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	if req.Pattern == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Pattern is required", http.StatusBadRequest))
		return
	}

	// Safety check: prevent deleting all keys accidentally
	if req.Pattern == "*" && !req.ConfirmDeleteAll {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("To delete all keys, set confirm_delete_all to true", http.StatusBadRequest))
		return
	}

	// Scan and delete matching keys
	var cursor uint64
	var deletedCount int64 = 0

	for {
		var keys []string
		var err error

		keys, cursor, err = h.valkeyCache.GetClient().Scan(ctx, cursor, req.Pattern, 100).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to scan keys: "+err.Error(), http.StatusInternalServerError))
			return
		}

		if len(keys) > 0 {
			deleted, err := h.valkeyCache.GetClient().Del(ctx, keys...).Result()
			if err != nil {
				zap.L().Warn("Failed to delete some keys", zap.Error(err))
			} else {
				deletedCount += deleted
			}
		}

		if cursor == 0 {
			break
		}
	}

	zap.L().Info("Cache keys deleted by pattern",
		zap.Int64("count", deletedCount),
		zap.String("pattern", req.Pattern))

	c.JSON(http.StatusOK, responses.SuccessResponse(fmt.Sprintf("Deleted %d key(s) matching pattern", deletedCount), nil, map[string]int64{"deleted_count": deletedCount}))
}

// SetKey godoc
//
//	@Summary		Set Cache Key
//	@Description	Sets a cache key with an optional TTL
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CacheSetKeyRequest	true	"Set key request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/keys [post]
func (h *CacheHandler) SetKey(c *gin.Context) {
	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.CacheSetKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}
	var ttl time.Duration
	if req.TTL > 0 {
		ttl = time.Duration(req.TTL) * time.Second
	}
	rawValue, err := json.Marshal(req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to marshal value: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err = h.valkeyCache.Set(req.Key, rawValue, ttl); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to set key: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Cache key set",
		zap.String("key", req.Key),
		zap.Int64("ttl", req.TTL))

	c.JSON(http.StatusOK, responses.SuccessResponse("Cache key set successfully", nil, nil))
}

// FlushDatabase godoc
//
//	@Summary		Flush Cache Database
//	@Description	Deletes all keys in the current database (USE WITH CAUTION)
//	@Tags			Admin.Cache
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CacheFlushRequest	true	"Flush request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/cache/keys/flush [delete]
func (h *CacheHandler) FlushDatabase(c *gin.Context) {
	if h.valkeyCache == nil || h.valkeyCache.GetClient() == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Cache service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.CacheFlushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	if !req.Confirm {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("To flush database, set confirm to true", http.StatusBadRequest))
		return
	}

	err := h.valkeyCache.FlushDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to flush database: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Warn("Cache database flushed - all keys deleted")

	c.JSON(http.StatusOK, responses.SuccessResponse("Database flushed successfully - all keys deleted", nil, nil))
}

// region: ============== Helper Functions ==============

func parseRedisInfo(info string) map[string]string {
	result := make(map[string]string)
	lines := strings.SplitSeq(info, "\r\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}

	return result
}

func parseInfoInt(value string) (int64, bool) {
	var result int64
	_, err := fmt.Sscanf(value, "%d", &result)
	return result, err == nil
}

// endregion
