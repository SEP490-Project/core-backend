package requests

// CacheDeleteKeyRequest for deleting cache keys
type CacheDeleteKeyRequest struct {
	Keys []string `json:"keys" binding:"required,min=1" example:"user:123,session:456"`
}

// CacheDeletePatternRequest for deleting cache keys by pattern
type CacheDeletePatternRequest struct {
	Pattern          string `json:"pattern" binding:"required" example:"user:*"`
	ConfirmDeleteAll bool   `json:"confirm_delete_all" binding:"omitempty" example:"false"`
}

// CacheSetKeyRequest for setting a cache key
type CacheSetKeyRequest struct {
	Key   string `json:"key" binding:"required" example:"user:123"`
	Value any    `json:"value" binding:"required"`
	TTL   int64  `json:"ttl" binding:"omitempty,min=0" example:"3600"` // TTL in seconds, 0 = no expiration
}

// CacheFlushRequest for flushing the cache database
type CacheFlushRequest struct {
	Confirm bool `json:"confirm" binding:"required" example:"true"`
}
