package responses

// CacheOverviewResponse represents cache (Valkey/Redis) overview
type CacheOverviewResponse struct {
	Status                   string `json:"status" example:"connected"`
	Version                  string `json:"version" example:"7.2.0"`
	Uptime                   string `json:"uptime" example:"86400"`
	ConnectedClients         string `json:"connected_clients" example:"10"`
	UsedMemory               string `json:"used_memory" example:"1.5M"`
	UsedMemoryPeak           string `json:"used_memory_peak" example:"2.0M"`
	TotalKeys                int64  `json:"total_keys" example:"1234"`
	TotalConnectionsReceived string `json:"total_connections_received" example:"5000"`
	TotalCommandsProcessed   string `json:"total_commands_processed" example:"10000"`
	KeyspaceHits             string `json:"keyspace_hits" example:"8000"`
	KeyspaceMisses           string `json:"keyspace_misses" example:"2000"`
	HitRate                  string `json:"hit_rate" example:"80.00%"`
	EvictedKeys              string `json:"evicted_keys" example:"10"`
	ExpiredKeys              string `json:"expired_keys" example:"50"`
}

// CacheKeysResponse represents a list of cache keys
type CacheKeysResponse struct {
	Keys       []CacheKeyInfo `json:"keys"`
	TotalCount int            `json:"total_count" example:"100"`
	Pattern    string         `json:"pattern" example:"user:*"`
}

// CacheKeyInfo represents information about a cache key
type CacheKeyInfo struct {
	Key  string `json:"key" example:"user:123"`
	Type string `json:"type" example:"string"`
	TTL  int64  `json:"ttl" example:"3600"` // -1 for no expiration, -2 for expired
}

// CacheKeyValueResponse represents a cache key with its value
type CacheKeyValueResponse struct {
	Key   string `json:"key" example:"user:123"`
	Type  string `json:"type" example:"string"`
	Value any    `json:"value"`
	TTL   int64  `json:"ttl" example:"3600"`
}
