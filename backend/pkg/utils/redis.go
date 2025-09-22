package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"docker-auto/internal/config"

	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

// RedisClient wraps the redis client with additional functionality
type RedisClient struct {
	client redis.Cmdable
	config *config.RedisConfig
	logger *logrus.Logger
}

// NewRedisClient creates a new Redis client based on configuration
func NewRedisClient(cfg *config.RedisConfig, logger *logrus.Logger) (*RedisClient, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("Redis is disabled in configuration")
	}

	var client redis.Cmdable
	var err error

	if cfg.ClusterMode {
		client, err = createClusterClient(cfg)
	} else if cfg.SentinelMode {
		client, err = createSentinelClient(cfg)
	} else {
		client, err = createStandaloneClient(cfg)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Redis client: %w", err)
	}

	redisClient := &RedisClient{
		client: client,
		config: cfg,
		logger: logger,
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host":         cfg.Host,
		"port":         cfg.Port,
		"cluster_mode": cfg.ClusterMode,
		"sentinel_mode": cfg.SentinelMode,
		"tls_enabled":  cfg.TLSEnabled,
	}).Info("Redis client initialized successfully")

	return redisClient, nil
}

// createStandaloneClient creates a standalone Redis client
func createStandaloneClient(cfg *config.RedisConfig) (*redis.Client, error) {
	opts := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
	}

	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}

		if cfg.TLSCertFile != "" && cfg.TLSKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS certificates: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		opts.TLSConfig = tlsConfig
	}

	return redis.NewClient(opts), nil
}

// createClusterClient creates a Redis cluster client
func createClusterClient(cfg *config.RedisConfig) (*redis.ClusterClient, error) {
	if len(cfg.ClusterAddrs) == 0 {
		return nil, fmt.Errorf("cluster addresses are required for cluster mode")
	}

	opts := &redis.ClusterOptions{
		Addrs:        cfg.ClusterAddrs,
		Password:     cfg.Password,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxRetries:   cfg.MaxRetries,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.IdleTimeout) * time.Second,
	}

	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
		opts.TLSConfig = tlsConfig
	}

	return redis.NewClusterClient(opts), nil
}

// createSentinelClient creates a Redis sentinel client
func createSentinelClient(cfg *config.RedisConfig) (*redis.Client, error) {
	if len(cfg.SentinelAddrs) == 0 {
		return nil, fmt.Errorf("sentinel addresses are required for sentinel mode")
	}

	if cfg.MasterName == "" {
		return nil, fmt.Errorf("master name is required for sentinel mode")
	}

	opts := &redis.FailoverOptions{
		MasterName:    cfg.MasterName,
		SentinelAddrs: cfg.SentinelAddrs,
		Password:      cfg.Password,
		DB:            cfg.DB,
		PoolSize:      cfg.PoolSize,
		MinIdleConns:  cfg.MinIdleConns,
		MaxRetries:    cfg.MaxRetries,
		DialTimeout:   time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:   time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:  time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:   time.Duration(cfg.IdleTimeout) * time.Second,
	}

	if cfg.TLSEnabled {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: cfg.TLSSkipVerify,
		}
		opts.TLSConfig = tlsConfig
	}

	return redis.NewFailoverClient(opts), nil
}

// Ping tests Redis connectivity
func (r *RedisClient) Ping(ctx context.Context) error {
	result := r.client.Ping(ctx)
	return result.Err()
}

// Close closes the Redis client connection
func (r *RedisClient) Close() error {
	if closer, ok := r.client.(interface{ Close() error }); ok {
		err := closer.Close()
		if err != nil {
			r.logger.WithError(err).Error("Failed to close Redis client")
			return err
		}
		r.logger.Info("Redis client closed successfully")
	}
	return nil
}

// Set stores a key-value pair with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	result := r.client.Set(ctx, key, value, expiration)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"key":        key,
			"expiration": expiration,
		}).Error("Failed to set Redis key")
		return err
	}
	return nil
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	result := r.client.Get(ctx, key)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return "", nil // Key not found
		}
		r.logger.WithError(err).WithField("key", key).Error("Failed to get Redis key")
		return "", err
	}
	return result.Val(), nil
}

// Delete removes a key from Redis
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	result := r.client.Del(ctx, keys...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("keys", keys).Error("Failed to delete Redis keys")
		return err
	}
	return nil
}

// Exists checks if keys exist
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	result := r.client.Exists(ctx, keys...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("keys", keys).Error("Failed to check Redis key existence")
		return 0, err
	}
	return result.Val(), nil
}

// Expire sets expiration for a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	result := r.client.Expire(ctx, key, expiration)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"key":        key,
			"expiration": expiration,
		}).Error("Failed to set Redis key expiration")
		return err
	}
	return nil
}

// TTL returns the time to live for a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	result := r.client.TTL(ctx, key)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to get Redis key TTL")
		return 0, err
	}
	return result.Val(), nil
}

// HSet sets field in hash
func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	result := r.client.HSet(ctx, key, values...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to set Redis hash field")
		return err
	}
	return nil
}

// HGet gets field from hash
func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	result := r.client.HGet(ctx, key, field)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return "", nil // Field not found
		}
		r.logger.WithError(err).WithFields(logrus.Fields{
			"key":   key,
			"field": field,
		}).Error("Failed to get Redis hash field")
		return "", err
	}
	return result.Val(), nil
}

// HGetAll gets all fields from hash
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result := r.client.HGetAll(ctx, key)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to get all Redis hash fields")
		return nil, err
	}
	return result.Val(), nil
}

// HDel deletes fields from hash
func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}

	result := r.client.HDel(ctx, key, fields...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"key":    key,
			"fields": fields,
		}).Error("Failed to delete Redis hash fields")
		return err
	}
	return nil
}

// SAdd adds members to set
func (r *RedisClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	result := r.client.SAdd(ctx, key, members...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to add to Redis set")
		return err
	}
	return nil
}

// SRem removes members from set
func (r *RedisClient) SRem(ctx context.Context, key string, members ...interface{}) error {
	result := r.client.SRem(ctx, key, members...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to remove from Redis set")
		return err
	}
	return nil
}

// SIsMember checks if member is in set
func (r *RedisClient) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	result := r.client.SIsMember(ctx, key, member)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithFields(logrus.Fields{
			"key":    key,
			"member": member,
		}).Error("Failed to check Redis set membership")
		return false, err
	}
	return result.Val(), nil
}

// SMembers returns all members of set
func (r *RedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	result := r.client.SMembers(ctx, key)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to get Redis set members")
		return nil, err
	}
	return result.Val(), nil
}

// ZAdd adds members to sorted set
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	result := r.client.ZAdd(ctx, key, members...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to add to Redis sorted set")
		return err
	}
	return nil
}

// ZRem removes members from sorted set
func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	result := r.client.ZRem(ctx, key, members...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to remove from Redis sorted set")
		return err
	}
	return nil
}

// ZRange returns range of members from sorted set
func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	result := r.client.ZRange(ctx, key, start, stop)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to get Redis sorted set range")
		return nil, err
	}
	return result.Val(), nil
}

// LPush pushes elements to the left of list
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	result := r.client.LPush(ctx, key, values...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to left push to Redis list")
		return err
	}
	return nil
}

// RPush pushes elements to the right of list
func (r *RedisClient) RPush(ctx context.Context, key string, values ...interface{}) error {
	result := r.client.RPush(ctx, key, values...)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to right push to Redis list")
		return err
	}
	return nil
}

// LPop pops element from left of list
func (r *RedisClient) LPop(ctx context.Context, key string) (string, error) {
	result := r.client.LPop(ctx, key)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return "", nil // List is empty
		}
		r.logger.WithError(err).WithField("key", key).Error("Failed to left pop from Redis list")
		return "", err
	}
	return result.Val(), nil
}

// RPop pops element from right of list
func (r *RedisClient) RPop(ctx context.Context, key string) (string, error) {
	result := r.client.RPop(ctx, key)
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return "", nil // List is empty
		}
		r.logger.WithError(err).WithField("key", key).Error("Failed to right pop from Redis list")
		return "", err
	}
	return result.Val(), nil
}

// LLen returns the length of list
func (r *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	result := r.client.LLen(ctx, key)
	if err := result.Err(); err != nil {
		r.logger.WithError(err).WithField("key", key).Error("Failed to get Redis list length")
		return 0, err
	}
	return result.Val(), nil
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisClient) GetClient() redis.Cmdable {
	return r.client
}

// GetStats returns Redis client pool statistics
func (r *RedisClient) GetStats() interface{} {
	if client, ok := r.client.(*redis.Client); ok {
		return client.PoolStats()
	}
	if clusterClient, ok := r.client.(*redis.ClusterClient); ok {
		return clusterClient.PoolStats()
	}
	return nil
}

// HealthCheck performs a comprehensive health check
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	// Test basic connectivity
	if err := r.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Test basic operations
	testKey := "health_check_" + fmt.Sprintf("%d", time.Now().UnixNano())
	testValue := "test_value"

	// Test SET
	if err := r.Set(ctx, testKey, testValue, time.Minute); err != nil {
		return fmt.Errorf("set operation failed: %w", err)
	}

	// Test GET
	retrievedValue, err := r.Get(ctx, testKey)
	if err != nil {
		return fmt.Errorf("get operation failed: %w", err)
	}

	if retrievedValue != testValue {
		return fmt.Errorf("value mismatch: expected %s, got %s", testValue, retrievedValue)
	}

	// Test DELETE
	if err := r.Delete(ctx, testKey); err != nil {
		return fmt.Errorf("delete operation failed: %w", err)
	}

	r.logger.Debug("Redis health check completed successfully")
	return nil
}