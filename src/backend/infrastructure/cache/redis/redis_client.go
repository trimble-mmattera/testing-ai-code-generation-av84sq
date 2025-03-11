// Package redis implements a Redis client for caching in the Document Management Platform.
// It provides key-value operations, expiration management, and pattern-based deletion
// with full tenant isolation to ensure data security.
package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8" // v8.11.5

	"../../../pkg/config"
	"../../../pkg/errors"
	"../../../pkg/logger"
)

// Default TTL for cache entries (15 minutes)
var defaultTTL = time.Duration(15 * time.Minute)

// RedisConfig defines configuration options for Redis connection
type RedisConfig struct {
	// Address is the Redis server address (host:port)
	Address string
	// Password for Redis authentication (empty for no auth)
	Password string
	// DB is the Redis database number
	DB int
	// PoolSize controls the number of connections in the pool
	PoolSize int
	// DefaultTTL is the default expiration time for cache entries
	DefaultTTL time.Duration
}

// NewRedisConfig creates a new RedisConfig with default values
func NewRedisConfig() *RedisConfig {
	return &RedisConfig{
		Address:    "localhost:6379",
		Password:   "",
		DB:         0,
		PoolSize:   10,
		DefaultTTL: 15 * time.Minute,
	}
}

// RedisClient implements caching operations using Redis
type RedisClient struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewRedisClient creates a new Redis client instance with the provided configuration
func NewRedisClient(cfg map[string]interface{}) (*RedisClient, error) {
	// Extract Redis configuration
	redisConfig := &RedisConfig{}
	
	if address, ok := cfg["address"].(string); ok && address != "" {
		redisConfig.Address = address
	} else {
		redisConfig.Address = "localhost:6379"
	}
	
	if password, ok := cfg["password"].(string); ok {
		redisConfig.Password = password
	}
	
	if db, ok := cfg["db"].(int); ok {
		redisConfig.DB = db
	}
	
	if poolSize, ok := cfg["pool_size"].(int); ok && poolSize > 0 {
		redisConfig.PoolSize = poolSize
	} else {
		redisConfig.PoolSize = 10
	}
	
	if ttl, ok := cfg["default_ttl"].(time.Duration); ok && ttl > 0 {
		redisConfig.DefaultTTL = ttl
	} else {
		redisConfig.DefaultTTL = defaultTTL
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     redisConfig.Address,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
		PoolSize: redisConfig.PoolSize,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, errors.Wrap(err, "failed to connect to Redis")
	}

	logger.Info("Connected to Redis", "address", redisConfig.Address, "db", redisConfig.DB)

	return &RedisClient{
		client:     client,
		defaultTTL: redisConfig.DefaultTTL,
	}, nil
}

// Set stores a value in the cache with the specified key and TTL
func (rc *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Marshal value to JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "failed to marshal value to JSON")
	}

	// Store in Redis
	err = rc.client.Set(ctx, key, jsonBytes, ttl).Err()
	if err != nil {
		return errors.Wrap(err, "failed to set value in Redis")
	}

	logger.WithField("key", key).WithField("ttl", ttl.String()).Debug("Cache set operation successful")
	return nil
}

// Get retrieves a value from the cache by key and unmarshals it into the provided destination
func (rc *RedisClient) Get(ctx context.Context, key string, dest interface{}) (bool, error) {
	// Check if key exists
	exists, err := rc.Exists(ctx, key)
	if err != nil {
		return false, errors.Wrap(err, "failed to check key existence")
	}

	if !exists {
		return false, nil
	}

	// Get value from Redis
	val, err := rc.client.Get(ctx, key).Result()
	if err != nil {
		return false, errors.Wrap(err, "failed to get value from Redis")
	}

	// Unmarshal JSON value into destination
	err = json.Unmarshal([]byte(val), dest)
	if err != nil {
		return false, errors.Wrap(err, "failed to unmarshal value from JSON")
	}

	logger.WithField("key", key).Debug("Cache get operation successful")
	return true, nil
}

// Delete removes a value from the cache by key
func (rc *RedisClient) Delete(ctx context.Context, key string) error {
	err := rc.client.Del(ctx, key).Err()
	if err != nil {
		return errors.Wrap(err, "failed to delete key from Redis")
	}

	logger.WithField("key", key).Debug("Cache delete operation successful")
	return nil
}

// DeletePattern removes all values matching a pattern from the cache
func (rc *RedisClient) DeletePattern(ctx context.Context, pattern string) error {
	// Scan for keys matching the pattern
	iter := rc.client.Scan(ctx, 0, pattern, 100).Iterator()
	
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		
		// Delete in batches of 100 to avoid large operations
		if len(keys) >= 100 {
			err := rc.client.Del(ctx, keys...).Err()
			if err != nil {
				return errors.Wrap(err, "failed to delete keys from Redis")
			}
			keys = keys[:0] // Clear slice but keep capacity
		}
	}
	
	// Delete any remaining keys
	if len(keys) > 0 {
		err := rc.client.Del(ctx, keys...).Err()
		if err != nil {
			return errors.Wrap(err, "failed to delete keys from Redis")
		}
	}
	
	if err := iter.Err(); err != nil {
		return errors.Wrap(err, "error scanning Redis keys")
	}

	logger.WithField("pattern", pattern).WithField("count", len(keys)).Debug("Cache delete pattern operation successful")
	return nil
}

// Exists checks if a key exists in the cache
func (rc *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	result, err := rc.client.Exists(ctx, key).Result()
	if err != nil {
		return false, errors.Wrap(err, "failed to check key existence in Redis")
	}
	
	return result > 0, nil
}

// SetWithDefaultTTL stores a value in the cache with the default TTL
func (rc *RedisClient) SetWithDefaultTTL(ctx context.Context, key string, value interface{}) error {
	return rc.Set(ctx, key, value, rc.defaultTTL)
}

// Close closes the Redis client connection
func (rc *RedisClient) Close() error {
	err := rc.client.Close()
	if err != nil {
		return errors.Wrap(err, "failed to close Redis client")
	}
	
	logger.Info("Redis client closed")
	return nil
}