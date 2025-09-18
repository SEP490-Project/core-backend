package persistence

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"core-backend/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ValkeyCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewValkeyCache() *ValkeyCache {
	cfg := config.GetAppConfig().Cache
	
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, strconv.Itoa(cfg.Port)),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	cache := &ValkeyCache{
		client: client,
		ctx:    context.Background(),
	}

	// Test connection
	if err := cache.Ping(); err != nil {
		zap.L().Error("Failed to connect to Valkey", zap.Error(err))
		return nil
	}

	zap.L().Info("Valkey cache connected successfully", 
		zap.String("host", cfg.Host), 
		zap.Int("port", cfg.Port),
		zap.Int("db", cfg.DB))

	return cache
}

// Ping tests the connection to Valkey
func (v *ValkeyCache) Ping() error {
	return v.client.Ping(v.ctx).Err()
}

// Set stores a value with an optional expiration time
func (v *ValkeyCache) Set(key string, value interface{}, expiration time.Duration) error {
	return v.client.Set(v.ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (v *ValkeyCache) Get(key string) (string, error) {
	result := v.client.Get(v.ctx, key)
	if result.Err() == redis.Nil {
		return "", nil // Key does not exist
	}
	return result.Result()
}

// GetBytes retrieves a value as bytes by key
func (v *ValkeyCache) GetBytes(key string) ([]byte, error) {
	result := v.client.Get(v.ctx, key)
	if result.Err() == redis.Nil {
		return nil, nil // Key does not exist
	}
	if result.Err() != nil {
		return nil, result.Err()
	}
	return []byte(result.Val()), nil
}

// SetJSON stores a JSON-serializable value
func (v *ValkeyCache) SetJSON(key string, value interface{}, expiration time.Duration) error {
	return v.client.Set(v.ctx, key, value, expiration).Err()
}

// GetJSON retrieves and deserializes a JSON value
func (v *ValkeyCache) GetJSON(key string, dest interface{}) error {
	result := v.client.Get(v.ctx, key)
	if result.Err() == redis.Nil {
		return nil // Key does not exist
	}
	if result.Err() != nil {
		return result.Err()
	}
	
	// In a real implementation, you'd unmarshal JSON here
	// For now, this is a placeholder
	return nil
}

// Delete removes a key
func (v *ValkeyCache) Delete(keys ...string) error {
	return v.client.Del(v.ctx, keys...).Err()
}

// Exists checks if a key exists
func (v *ValkeyCache) Exists(key string) (bool, error) {
	count, err := v.client.Exists(v.ctx, key).Result()
	return count > 0, err
}

// Expire sets an expiration time for a key
func (v *ValkeyCache) Expire(key string, expiration time.Duration) error {
	return v.client.Expire(v.ctx, key, expiration).Err()
}

// TTL returns the time to live for a key
func (v *ValkeyCache) TTL(key string) (time.Duration, error) {
	return v.client.TTL(v.ctx, key).Result()
}

// Increment increments an integer value
func (v *ValkeyCache) Increment(key string) (int64, error) {
	return v.client.Incr(v.ctx, key).Result()
}

// IncrementBy increments a value by the given amount
func (v *ValkeyCache) IncrementBy(key string, value int64) (int64, error) {
	return v.client.IncrBy(v.ctx, key, value).Result()
}

// Decrement decrements an integer value
func (v *ValkeyCache) Decrement(key string) (int64, error) {
	return v.client.Decr(v.ctx, key).Result()
}

// DecrementBy decrements a value by the given amount
func (v *ValkeyCache) DecrementBy(key string, value int64) (int64, error) {
	return v.client.DecrBy(v.ctx, key, value).Result()
}

// SetNX sets a key only if it doesn't exist
func (v *ValkeyCache) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
	return v.client.SetNX(v.ctx, key, value, expiration).Result()
}

// MGet retrieves multiple values by keys
func (v *ValkeyCache) MGet(keys ...string) ([]interface{}, error) {
	return v.client.MGet(v.ctx, keys...).Result()
}

// MSet sets multiple key-value pairs
func (v *ValkeyCache) MSet(pairs ...interface{}) error {
	return v.client.MSet(v.ctx, pairs...).Err()
}

// Keys returns all keys matching a pattern
func (v *ValkeyCache) Keys(pattern string) ([]string, error) {
	return v.client.Keys(v.ctx, pattern).Result()
}

// FlushDB clears the current database
func (v *ValkeyCache) FlushDB() error {
	return v.client.FlushDB(v.ctx).Err()
}

// Close closes the connection to Valkey
func (v *ValkeyCache) Close() error {
	return v.client.Close()
}

// GetClient returns the underlying Redis client for advanced operations
func (v *ValkeyCache) GetClient() *redis.Client {
	return v.client
}