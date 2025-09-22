package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"docker-auto/internal/config"

	"github.com/sirupsen/logrus"
)

// CacheManager provides a unified interface for both memory and Redis caching
type CacheManager struct {
	memoryCache *MemoryCache
	redisClient *RedisClient
	config      *config.Config
	logger      *logrus.Logger
	enabled     bool
	mu          sync.RWMutex
}

// NewCacheManager creates a new cache manager with both memory and Redis support
func NewCacheManager(cfg *config.Config, logger *logrus.Logger) (*CacheManager, error) {
	cm := &CacheManager{
		config:  cfg,
		logger:  logger,
		enabled: cfg.Cache.Enabled,
	}

	// Always initialize memory cache
	if cfg.Cache.Enabled {
		cm.memoryCache = NewMemoryCacheWithConfig(cfg)
		logger.Info("Memory cache initialized")
	}

	// Initialize Redis if enabled
	if cfg.Redis.Enabled {
		redisClient, err := NewRedisClient(&cfg.Redis, logger)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize Redis client, using memory cache only")
		} else {
			cm.redisClient = redisClient
			logger.Info("Redis cache initialized")
		}
	}

	return cm, nil
}

// CacheLevel represents the cache level to use
type CacheLevel int

const (
	CacheLevelMemory CacheLevel = iota // Memory cache only
	CacheLevelRedis                    // Redis cache only
	CacheLevelBoth                     // Both memory and Redis (L1/L2 cache)
	CacheLevelAuto                     // Auto-select based on configuration
)

// CacheOptions provides options for cache operations
type CacheOptions struct {
	Level      CacheLevel
	TTL        time.Duration
	MemoryTTL  time.Duration // Specific TTL for memory cache (L1)
	RedisTTL   time.Duration // Specific TTL for Redis cache (L2)
	SkipMemory bool          // Skip memory cache for this operation
	SkipRedis  bool          // Skip Redis cache for this operation
}

// DefaultCacheOptions returns default cache options
func DefaultCacheOptions() *CacheOptions {
	return &CacheOptions{
		Level: CacheLevelAuto,
		TTL:   30 * time.Minute,
	}
}

// Set stores a value with the specified options
func (cm *CacheManager) Set(ctx context.Context, key string, value interface{}, opts *CacheOptions) error {
	if !cm.enabled {
		return nil
	}

	if opts == nil {
		opts = DefaultCacheOptions()
	}

	level := cm.resolveCacheLevel(opts.Level)
	memoryTTL := opts.MemoryTTL
	redisTTL := opts.RedisTTL

	if memoryTTL == 0 {
		memoryTTL = opts.TTL
	}
	if redisTTL == 0 {
		redisTTL = opts.TTL
	}

	var errors []error

	// Store in memory cache (L1)
	if level == CacheLevelMemory || level == CacheLevelBoth {
		if cm.memoryCache != nil && !opts.SkipMemory {
			if err := cm.memoryCache.Set(key, value, memoryTTL); err != nil {
				cm.logger.WithError(err).WithField("key", key).Warn("Failed to set memory cache")
				errors = append(errors, fmt.Errorf("memory cache: %w", err))
			}
		}
	}

	// Store in Redis cache (L2)
	if level == CacheLevelRedis || level == CacheLevelBoth {
		if cm.redisClient != nil && !opts.SkipRedis {
			// Serialize value for Redis
			serializedValue, err := cm.serializeValue(value)
			if err != nil {
				cm.logger.WithError(err).WithField("key", key).Warn("Failed to serialize value for Redis")
				errors = append(errors, fmt.Errorf("redis serialization: %w", err))
			} else {
				if err := cm.redisClient.Set(ctx, key, serializedValue, redisTTL); err != nil {
					cm.logger.WithError(err).WithField("key", key).Warn("Failed to set Redis cache")
					errors = append(errors, fmt.Errorf("redis cache: %w", err))
				}
			}
		}
	}

	// Return error only if all cache operations failed
	if len(errors) > 0 && len(errors) == cm.getActiveCacheCount(level, opts) {
		return fmt.Errorf("all cache operations failed: %v", errors)
	}

	return nil
}

// Get retrieves a value with the specified options
func (cm *CacheManager) Get(ctx context.Context, key string, opts *CacheOptions) (interface{}, bool, error) {
	if !cm.enabled {
		return nil, false, nil
	}

	if opts == nil {
		opts = DefaultCacheOptions()
	}

	level := cm.resolveCacheLevel(opts.Level)

	// Try memory cache first (L1)
	if level == CacheLevelMemory || level == CacheLevelBoth {
		if cm.memoryCache != nil && !opts.SkipMemory {
			if value, exists := cm.memoryCache.Get(key); exists {
				cm.logger.WithField("key", key).Debug("Cache hit in memory")
				return value, true, nil
			}
		}
	}

	// Try Redis cache (L2)
	if level == CacheLevelRedis || level == CacheLevelBoth {
		if cm.redisClient != nil && !opts.SkipRedis {
			serializedValue, err := cm.redisClient.Get(ctx, key)
			if err != nil {
				cm.logger.WithError(err).WithField("key", key).Warn("Failed to get from Redis cache")
			} else if serializedValue != "" {
				value, err := cm.deserializeValue(serializedValue)
				if err != nil {
					cm.logger.WithError(err).WithField("key", key).Warn("Failed to deserialize Redis value")
				} else {
					cm.logger.WithField("key", key).Debug("Cache hit in Redis")

					// Populate memory cache if using both levels
					if level == CacheLevelBoth && cm.memoryCache != nil && !opts.SkipMemory {
						memoryTTL := opts.MemoryTTL
						if memoryTTL == 0 {
							memoryTTL = opts.TTL
						}
						if err := cm.memoryCache.Set(key, value, memoryTTL); err != nil {
							cm.logger.WithError(err).WithField("key", key).Warn("Failed to populate memory cache from Redis")
						}
					}

					return value, true, nil
				}
			}
		}
	}

	return nil, false, nil
}

// Delete removes a key from cache
func (cm *CacheManager) Delete(ctx context.Context, key string, opts *CacheOptions) error {
	if !cm.enabled {
		return nil
	}

	if opts == nil {
		opts = DefaultCacheOptions()
	}

	level := cm.resolveCacheLevel(opts.Level)
	var errors []error

	// Delete from memory cache
	if level == CacheLevelMemory || level == CacheLevelBoth {
		if cm.memoryCache != nil && !opts.SkipMemory {
			cm.memoryCache.Delete(key)
		}
	}

	// Delete from Redis cache
	if level == CacheLevelRedis || level == CacheLevelBoth {
		if cm.redisClient != nil && !opts.SkipRedis {
			if err := cm.redisClient.Delete(ctx, key); err != nil {
				cm.logger.WithError(err).WithField("key", key).Warn("Failed to delete from Redis cache")
				errors = append(errors, fmt.Errorf("redis delete: %w", err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache delete errors: %v", errors)
	}

	return nil
}

// Exists checks if a key exists in cache
func (cm *CacheManager) Exists(ctx context.Context, key string, opts *CacheOptions) (bool, error) {
	if !cm.enabled {
		return false, nil
	}

	if opts == nil {
		opts = DefaultCacheOptions()
	}

	level := cm.resolveCacheLevel(opts.Level)

	// Check memory cache first
	if level == CacheLevelMemory || level == CacheLevelBoth {
		if cm.memoryCache != nil && !opts.SkipMemory {
			if cm.memoryCache.Exists(key) {
				return true, nil
			}
		}
	}

	// Check Redis cache
	if level == CacheLevelRedis || level == CacheLevelBoth {
		if cm.redisClient != nil && !opts.SkipRedis {
			count, err := cm.redisClient.Exists(ctx, key)
			if err != nil {
				return false, err
			}
			return count > 0, nil
		}
	}

	return false, nil
}

// Clear clears all cache entries
func (cm *CacheManager) Clear(ctx context.Context, opts *CacheOptions) error {
	if !cm.enabled {
		return nil
	}

	if opts == nil {
		opts = DefaultCacheOptions()
	}

	level := cm.resolveCacheLevel(opts.Level)
	var errors []error

	// Clear memory cache
	if level == CacheLevelMemory || level == CacheLevelBoth {
		if cm.memoryCache != nil && !opts.SkipMemory {
			cm.memoryCache.Clear()
		}
	}

	// Clear Redis cache (use FLUSHDB for the current database)
	if level == CacheLevelRedis || level == CacheLevelBoth {
		if cm.redisClient != nil && !opts.SkipRedis {
			client := cm.redisClient.GetClient()
			if err := client.FlushDB(ctx).Err(); err != nil {
				cm.logger.WithError(err).Warn("Failed to clear Redis cache")
				errors = append(errors, fmt.Errorf("redis flush: %w", err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache clear errors: %v", errors)
	}

	return nil
}

// GetWithFunc retrieves a value or computes it using the provided function
func (cm *CacheManager) GetWithFunc(ctx context.Context, key string, fn func() (interface{}, error), opts *CacheOptions) (interface{}, error) {
	if !cm.enabled {
		return fn()
	}

	// Try to get from cache first
	if value, exists, err := cm.Get(ctx, key, opts); err != nil {
		cm.logger.WithError(err).WithField("key", key).Warn("Cache get error in GetWithFunc")
	} else if exists {
		return value, nil
	}

	// Compute value
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := cm.Set(ctx, key, value, opts); err != nil {
		cm.logger.WithError(err).WithField("key", key).Warn("Failed to store computed value in cache")
	}

	return value, nil
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if cm.memoryCache != nil {
		stats["memory"] = cm.memoryCache.GetAdvancedStats()
	}

	if cm.redisClient != nil {
		stats["redis"] = cm.redisClient.GetStats()
	}

	stats["enabled"] = cm.enabled
	stats["redis_enabled"] = cm.redisClient != nil
	stats["memory_enabled"] = cm.memoryCache != nil

	return stats
}

// HealthCheck performs health checks on all cache backends
func (cm *CacheManager) HealthCheck(ctx context.Context) error {
	var errors []error

	// Check Redis if enabled
	if cm.redisClient != nil {
		if err := cm.redisClient.HealthCheck(ctx); err != nil {
			errors = append(errors, fmt.Errorf("redis health check failed: %w", err))
		}
	}

	// Memory cache doesn't need health check as it's in-process

	if len(errors) > 0 {
		return fmt.Errorf("cache health check failed: %v", errors)
	}

	return nil
}

// Close closes all cache connections
func (cm *CacheManager) Close() error {
	var errors []error

	if cm.memoryCache != nil {
		cm.memoryCache.Stop()
	}

	if cm.redisClient != nil {
		if err := cm.redisClient.Close(); err != nil {
			errors = append(errors, fmt.Errorf("redis close error: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache close errors: %v", errors)
	}

	return nil
}

// resolveCacheLevel determines the actual cache level to use
func (cm *CacheManager) resolveCacheLevel(level CacheLevel) CacheLevel {
	if level == CacheLevelAuto {
		if cm.redisClient != nil && cm.memoryCache != nil {
			return CacheLevelBoth
		} else if cm.redisClient != nil {
			return CacheLevelRedis
		} else if cm.memoryCache != nil {
			return CacheLevelMemory
		}
	}
	return level
}

// getActiveCacheCount returns the number of active cache backends for error handling
func (cm *CacheManager) getActiveCacheCount(level CacheLevel, opts *CacheOptions) int {
	count := 0
	resolvedLevel := cm.resolveCacheLevel(level)

	if (resolvedLevel == CacheLevelMemory || resolvedLevel == CacheLevelBoth) && cm.memoryCache != nil && !opts.SkipMemory {
		count++
	}

	if (resolvedLevel == CacheLevelRedis || resolvedLevel == CacheLevelBoth) && cm.redisClient != nil && !opts.SkipRedis {
		count++
	}

	return count
}

// serializeValue serializes a value for Redis storage
func (cm *CacheManager) serializeValue(value interface{}) (string, error) {
	switch v := value.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return fmt.Sprintf("%v", v), nil
	default:
		// Use JSON for complex types
		jsonBytes, err := json.Marshal(value)
		if err != nil {
			return "", fmt.Errorf("failed to marshal value: %w", err)
		}
		return string(jsonBytes), nil
	}
}

// deserializeValue deserializes a value from Redis
func (cm *CacheManager) deserializeValue(serialized string) (interface{}, error) {
	// Try to unmarshal as JSON first
	var jsonValue interface{}
	if err := json.Unmarshal([]byte(serialized), &jsonValue); err == nil {
		return jsonValue, nil
	}

	// Return as string if JSON unmarshal fails
	return serialized, nil
}

// JWT Token Blacklist Support

// BlacklistToken adds a JWT token to the blacklist
func (cm *CacheManager) BlacklistToken(ctx context.Context, tokenID string, expiration time.Time) error {
	key := fmt.Sprintf("jwt_blacklist:%s", tokenID)
	ttl := time.Until(expiration)
	if ttl <= 0 {
		return nil // Token already expired
	}

	opts := &CacheOptions{
		Level: CacheLevelAuto,
		TTL:   ttl,
	}

	return cm.Set(ctx, key, true, opts)
}

// IsTokenBlacklisted checks if a JWT token is blacklisted
func (cm *CacheManager) IsTokenBlacklisted(ctx context.Context, tokenID string) (bool, error) {
	key := fmt.Sprintf("jwt_blacklist:%s", tokenID)
	opts := &CacheOptions{Level: CacheLevelAuto}

	exists, err := cm.Exists(ctx, key, opts)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// Session Management Support

// SetUserSession stores user session data
func (cm *CacheManager) SetUserSession(ctx context.Context, sessionID string, userID int64, sessionData map[string]interface{}, ttl time.Duration) error {
	key := fmt.Sprintf("user_session:%s", sessionID)

	data := map[string]interface{}{
		"user_id":    userID,
		"session_id": sessionID,
		"created_at": time.Now().Unix(),
		"data":       sessionData,
	}

	opts := &CacheOptions{
		Level: CacheLevelAuto,
		TTL:   ttl,
	}

	return cm.Set(ctx, key, data, opts)
}

// GetUserSession retrieves user session data
func (cm *CacheManager) GetUserSession(ctx context.Context, sessionID string) (map[string]interface{}, bool, error) {
	key := fmt.Sprintf("user_session:%s", sessionID)
	opts := &CacheOptions{Level: CacheLevelAuto}

	value, exists, err := cm.Get(ctx, key, opts)
	if err != nil || !exists {
		return nil, exists, err
	}

	sessionData, ok := value.(map[string]interface{})
	if !ok {
		return nil, false, fmt.Errorf("invalid session data format")
	}

	return sessionData, true, nil
}

// DeleteUserSession removes user session data
func (cm *CacheManager) DeleteUserSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("user_session:%s", sessionID)
	opts := &CacheOptions{Level: CacheLevelAuto}

	return cm.Delete(ctx, key, opts)
}

// Distributed Lock Support

// AcquireLock attempts to acquire a distributed lock
func (cm *CacheManager) AcquireLock(ctx context.Context, lockKey string, ownerID string, ttl time.Duration) (bool, error) {
	if cm.redisClient == nil {
		return false, fmt.Errorf("Redis is required for distributed locks")
	}

	key := fmt.Sprintf("lock:%s", lockKey)
	client := cm.redisClient.GetClient()

	// Use SET with NX (only if not exists) and EX (expiration)
	result := client.SetNX(ctx, key, ownerID, ttl)
	if err := result.Err(); err != nil {
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	acquired := result.Val()
	if acquired {
		cm.logger.WithFields(logrus.Fields{
			"lock_key":  lockKey,
			"owner_id":  ownerID,
			"ttl":       ttl,
		}).Debug("Distributed lock acquired")
	}

	return acquired, nil
}

// ReleaseLock releases a distributed lock
func (cm *CacheManager) ReleaseLock(ctx context.Context, lockKey string, ownerID string) error {
	if cm.redisClient == nil {
		return fmt.Errorf("Redis is required for distributed locks")
	}

	key := fmt.Sprintf("lock:%s", lockKey)
	client := cm.redisClient.GetClient()

	// Use Lua script to ensure atomic release (only release if we own the lock)
	luaScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result := client.Eval(ctx, luaScript, []string{key}, ownerID)
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	released := result.Val().(int64) == 1
	if released {
		cm.logger.WithFields(logrus.Fields{
			"lock_key": lockKey,
			"owner_id": ownerID,
		}).Debug("Distributed lock released")
	}

	return nil
}

// RefreshLock extends the TTL of a distributed lock
func (cm *CacheManager) RefreshLock(ctx context.Context, lockKey string, ownerID string, ttl time.Duration) error {
	if cm.redisClient == nil {
		return fmt.Errorf("Redis is required for distributed locks")
	}

	key := fmt.Sprintf("lock:%s", lockKey)
	client := cm.redisClient.GetClient()

	// Use Lua script to ensure atomic refresh (only refresh if we own the lock)
	luaScript := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("EXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result := client.Eval(ctx, luaScript, []string{key}, ownerID, int(ttl.Seconds()))
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	refreshed := result.Val().(int64) == 1
	if !refreshed {
		return fmt.Errorf("lock not owned by %s or already expired", ownerID)
	}

	return nil
}