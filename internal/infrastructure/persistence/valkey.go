package persistence

import (
	"context"
	"core-backend/config"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ValkeyCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewValkeyCache() *ValkeyCache {
	zap.L().Info("Initializing Valkey cache connection")

	cfg := config.GetAppConfig().Cache
	zap.L().Debug("Valkey configuration loaded",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.Int("db", cfg.DB))

	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%s", cfg.Host, strconv.Itoa(cfg.Port)),
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        20,
		MinIdleConns:    5,
		ConnMaxLifetime: 10 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		MaxRetries:      3,
	})

	cache := &ValkeyCache{
		client: client,
		ctx:    context.Background(),
	}

	// Test connection in background to avoid blocking startup
	// The readiness probe should handle traffic gating until this succeeds
	go func() {
		zap.L().Debug("Testing Valkey connection in background")

		retryOpts := utils.RetryOptions{
			MaxAttempts:       30,
			BaseBackoff:       1 * time.Second,
			BackoffMultiplier: 1.0,
			AttemptTimeout:    3 * time.Second,
		}

		err := utils.RunWithRetry(context.Background(), retryOpts, func(ctx context.Context) error {
			return client.Ping(ctx).Err()
		})

		if err != nil {
			zap.L().Error("Failed to connect to Valkey after background retries",
				zap.String("host", cfg.Host),
				zap.Int("port", cfg.Port),
				zap.Error(err))
		} else {
			zap.L().Info("Valkey connection established successfully",
				zap.String("host", cfg.Host),
				zap.Int("port", cfg.Port))
		}
	}()

	return cache
}

// Ping tests the connection to Valkey
func (v *ValkeyCache) Ping() error {
	return v.client.Ping(v.ctx).Err()
}

// Set stores a value with an optional expiration time
func (v *ValkeyCache) Set(key string, value any, expiration time.Duration) error {
	zap.L().Debug("Setting cache key",
		zap.String("key", key),
		zap.Duration("expiration", expiration))

	switch val := value.(type) {

	// -----------------------------
	// Primitives → STRING
	// -----------------------------
	case string, []byte, int, int64, float64, bool:
		return v.setString(key, val, expiration)

	// -----------------------------
	// map[string]string → HASH
	// -----------------------------
	case map[string]string:
		return v.setHash(key, val, expiration)

	// -----------------------------
	// map[string]any → HASH if flat
	// -----------------------------
	case map[string]any:
		return v.setHash(key, utils.FlattenMapToString(val), expiration)

	// -----------------------------
	// []string → LIST
	// -----------------------------
	case []string:
		return v.setList(key, val, expiration)

	// -----------------------------
	// Fallback → JSON STRING
	// -----------------------------
	default:
		return v.setJSON(key, value, expiration)
	}
}

// Get retrieves a value by key
func (v *ValkeyCache) Get(key string) (value any, valueType string, err error) {
	zap.L().Debug("Getting cache key", zap.String("key", key))

	keyType, err := v.client.Type(v.ctx, key).Result()
	if err != nil {
		zap.L().Error("Failed to get key type",
			zap.String("key", key),
			zap.Error(err))
		return nil, "", err
	}

	switch keyType {

	case "none":
		// Key does not exist
		zap.L().Debug("Cache key not found", zap.String("key", key))
		return nil, "", nil

	// -----------------------------
	// STRING
	// -----------------------------
	case "string":
		val, err := v.client.Get(v.ctx, key).Result()
		if err != nil {
			return nil, keyType, err
		}

		// Attempt JSON decode
		var decoded any
		if json.Unmarshal([]byte(val), &decoded) == nil {
			return decoded, keyType, nil
		}

		// Fallback → raw string
		return val, keyType, nil

	// -----------------------------
	// HASH
	// -----------------------------
	case "hash":
		val, err := v.client.HGetAll(v.ctx, key).Result()
		if err != nil {
			return nil, keyType, err
		}
		return val, keyType, nil

	// -----------------------------
	// LIST
	// -----------------------------
	case "list":
		val, err := v.client.LRange(v.ctx, key, 0, -1).Result()
		if err != nil {
			return nil, keyType, err
		}
		return val, keyType, nil

	default:
		return nil, keyType, fmt.Errorf("unsupported redis type: %s", keyType)
	}
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
func (v *ValkeyCache) SetJSON(key string, value any, expiration time.Duration) error {
	return v.client.Set(v.ctx, key, value, expiration).Err()
}

// GetJSON retrieves and deserializes a JSON value
func (v *ValkeyCache) GetJSON(key string, dest any) error {
	reflect.TypeOf(dest).Kind()
	if reflect.TypeOf(dest).Kind() != reflect.Pointer {
		panic("dest must be a pointer")
	}

	result := v.client.Get(v.ctx, key)
	if result.Err() == redis.Nil {
		return nil // Key does not exist
	}
	if result.Err() != nil {
		return result.Err()
	}

	err := json.Unmarshal([]byte(result.Val()), dest)
	if err != nil {
		return err
	}
	return nil
}

// Delete removes a key
func (v *ValkeyCache) Delete(keys ...string) error {
	zap.L().Debug("Deleting cache keys", zap.Strings("keys", keys))

	err := v.client.Del(v.ctx, keys...).Err()
	if err != nil {
		zap.L().Error("Failed to delete cache keys",
			zap.Strings("keys", keys),
			zap.Error(err))
	} else {
		zap.L().Debug("Successfully deleted cache keys", zap.Strings("keys", keys))
	}

	return err
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
func (v *ValkeyCache) SetNX(key string, value any, expiration time.Duration) (bool, error) {
	return v.client.SetNX(v.ctx, key, value, expiration).Result()
}

// MGet retrieves multiple values by keys
func (v *ValkeyCache) MGet(keys ...string) ([]any, error) {
	return v.client.MGet(v.ctx, keys...).Result()
}

// MSet sets multiple key-value pairs
func (v *ValkeyCache) MSet(pairs ...any) error {
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

func (v *ValkeyCache) GetContext() context.Context {
	return v.ctx
}

// region: ============== Helper Functions ==============

func (v *ValkeyCache) setString(key string, value any, expiration time.Duration) error {
	return v.client.Set(v.ctx, key, value, expiration).Err()
}

func (v *ValkeyCache) setHash(key string, value map[string]string, expiration time.Duration) error {
	pipe := v.client.Pipeline()

	pipe.Del(v.ctx, key)
	pipe.HSet(v.ctx, key, value)

	if expiration > 0 {
		pipe.Expire(v.ctx, key, expiration)
	}

	_, err := pipe.Exec(v.ctx)
	return err
}

func (v *ValkeyCache) setList(key string, values []string, expiration time.Duration) error {
	pipe := v.client.Pipeline()

	pipe.Del(v.ctx, key)
	for _, value := range values {
		pipe.RPush(v.ctx, key, value)
	}

	if expiration > 0 {
		pipe.Expire(v.ctx, key, expiration)
	}

	_, err := pipe.Exec(v.ctx)
	return err
}

func (v *ValkeyCache) setJSON(key string, value any, expiration time.Duration) error {
	rawValue, err := json.Marshal(value)
	if err != nil {
		zap.L().Error("Failed to marshal value",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	return v.client.Set(v.ctx, key, rawValue, expiration).Err()
}

// endregion
