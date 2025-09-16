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

// CacheItem represents a cached item with expiration
type CacheItem struct {
	Value     interface{} `json:"value"`
	ExpiresAt time.Time   `json:"expires_at"`
	CreatedAt time.Time   `json:"created_at"`
}

// IsExpired checks if the cache item has expired
func (item *CacheItem) IsExpired() bool {
	return time.Now().UTC().After(item.ExpiresAt)
}

// MemoryCache implements a high-performance in-memory cache with TTL support and advanced features
type MemoryCache struct {
	data        sync.Map
	expiry      sync.Map
	defaultTTL  time.Duration
	cleanupDone chan struct{}
	mu          sync.RWMutex
	stats       CacheStats
	maxItems    int64
	evictionCh  chan string
	preloadCh   chan PreloadRequest
	metrics     *CacheMetrics
}

// PreloadRequest represents a cache preload request
type PreloadRequest struct {
	Key      string
	Loader   func() (interface{}, error)
	TTL      time.Duration
	Callback func(error)
}

// CacheMetrics tracks detailed cache performance metrics
type CacheMetrics struct {
	mu              sync.RWMutex
	HitLatencies    []time.Duration
	MissLatencies   []time.Duration
	SetLatencies    []time.Duration
	EvictionRate    float64
	MemoryUsage     int64
	LastOptimization time.Time
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits         int64 `json:"hits"`
	Misses       int64 `json:"misses"`
	Sets         int64 `json:"sets"`
	Deletes      int64 `json:"deletes"`
	Evictions    int64 `json:"evictions"`
	ItemCount    int64 `json:"item_count"`
	LastCleanup  time.Time `json:"last_cleanup"`
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	DefaultTTL      time.Duration
	CleanupInterval time.Duration
	MaxItems        int
}

// NewMemoryCache creates a new memory cache instance
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		defaultTTL:  30 * time.Minute,
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.startCleanup(5 * time.Minute)

	return cache
}

// NewMemoryCacheWithConfig creates a high-performance memory cache with advanced configuration
func NewMemoryCacheWithConfig(cfg *config.Config) *MemoryCache {
	cache := &MemoryCache{
		defaultTTL:  time.Duration(cfg.Cache.DefaultTTLMinutes) * time.Minute,
		cleanupDone: make(chan struct{}),
		maxItems:    10000, // Configurable max items
		evictionCh:  make(chan string, 1000),
		preloadCh:   make(chan PreloadRequest, 100),
		metrics: &CacheMetrics{
			HitLatencies:  make([]time.Duration, 0, 1000),
			MissLatencies: make([]time.Duration, 0, 1000),
			SetLatencies:  make([]time.Duration, 0, 1000),
		},
	}

	cleanupInterval := time.Duration(cfg.Cache.CleanupIntervalMinutes) * time.Minute

	// Start background goroutines for performance optimization
	go cache.startCleanup(cleanupInterval)
	go cache.startEvictionWorker()
	go cache.startPreloadWorker()
	go cache.startMetricsCollector()

	logrus.WithFields(logrus.Fields{
		"default_ttl":      cache.defaultTTL,
		"cleanup_interval": cleanupInterval,
		"max_items":        cache.maxItems,
		"features":          []string{"eviction_worker", "preload_worker", "metrics_collector"},
	}).Info("High-performance memory cache initialized")

	return cache
}

// Set stores a value in the cache with TTL and performance monitoring
func (c *MemoryCache) Set(key string, value interface{}, ttl time.Duration) error {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.recordLatency(latency, "set")
	}()

	if key == "" {
		return fmt.Errorf("cache key cannot be empty")
	}

	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	// Check if we need to evict items before adding new ones
	c.mu.RLock()
	currentCount := c.stats.ItemCount
	c.mu.RUnlock()

	if currentCount >= c.maxItems {
		// Trigger LRU eviction asynchronously
		c.triggerEviction()
	}

	item := &CacheItem{
		Value:     value,
		ExpiresAt: time.Now().UTC().Add(ttl),
		CreatedAt: time.Now().UTC(),
	}

	c.data.Store(key, item)
	c.expiry.Store(key, item.ExpiresAt)

	c.mu.Lock()
	c.stats.Sets++
	c.stats.ItemCount++
	c.mu.Unlock()

	return nil
}

// Get retrieves a value from the cache with performance monitoring
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		c.recordLatency(latency, "get")
	}()

	if key == "" {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	value, exists := c.data.Load(key)
	if !exists {
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	item, ok := value.(*CacheItem)
	if !ok {
		c.Delete(key)
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	if item.IsExpired() {
		// Non-blocking eviction for better performance
		select {
		case c.evictionCh <- key:
		default:
			// Channel full, delete synchronously
			c.Delete(key)
		}
		c.mu.Lock()
		c.stats.Misses++
		c.mu.Unlock()
		return nil, false
	}

	c.mu.Lock()
	c.stats.Hits++
	c.mu.Unlock()

	return item.Value, true
}

// GetString retrieves a string value from the cache
func (c *MemoryCache) GetString(key string) (string, bool) {
	value, exists := c.Get(key)
	if !exists {
		return "", false
	}

	str, ok := value.(string)
	return str, ok
}

// GetInt retrieves an int value from the cache
func (c *MemoryCache) GetInt(key string) (int, bool) {
	value, exists := c.Get(key)
	if !exists {
		return 0, false
	}

	num, ok := value.(int)
	return num, ok
}

// GetBool retrieves a bool value from the cache
func (c *MemoryCache) GetBool(key string) (bool, bool) {
	value, exists := c.Get(key)
	if !exists {
		return false, false
	}

	b, ok := value.(bool)
	return b, ok
}

// GetJSON retrieves and unmarshals a JSON value from the cache
func (c *MemoryCache) GetJSON(key string, dest interface{}) bool {
	value, exists := c.Get(key)
	if !exists {
		return false
	}

	jsonData, ok := value.([]byte)
	if !ok {
		// Try string as well
		jsonStr, ok := value.(string)
		if !ok {
			return false
		}
		jsonData = []byte(jsonStr)
	}

	if err := json.Unmarshal(jsonData, dest); err != nil {
		logrus.WithError(err).Warn("Failed to unmarshal cached JSON")
		return false
	}

	return true
}

// SetJSON marshals and stores a JSON value in the cache
func (c *MemoryCache) SetJSON(key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.Set(key, jsonData, ttl)
}

// Delete removes a value from the cache
func (c *MemoryCache) Delete(key string) {
	if key == "" {
		return
	}

	c.data.Delete(key)
	c.expiry.Delete(key)

	c.mu.Lock()
	c.stats.Deletes++
	c.stats.ItemCount--
	if c.stats.ItemCount < 0 {
		c.stats.ItemCount = 0
	}
	c.mu.Unlock()
}

// Exists checks if a key exists in the cache (and is not expired)
func (c *MemoryCache) Exists(key string) bool {
	_, exists := c.Get(key)
	return exists
}

// Clear removes all items from the cache
func (c *MemoryCache) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		c.expiry.Delete(key)
		return true
	})

	c.mu.Lock()
	c.stats.ItemCount = 0
	c.mu.Unlock()

	logrus.Info("Cache cleared")
}

// GetStats returns cache statistics
func (c *MemoryCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// GetHitRatio returns cache hit ratio as percentage
func (c *MemoryCache) GetHitRatio() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}

	return (float64(c.stats.Hits) / float64(total)) * 100
}

// Keys returns all non-expired keys in the cache
func (c *MemoryCache) Keys() []string {
	var keys []string
	now := time.Now().UTC()

	c.data.Range(func(key, value interface{}) bool {
		keyStr, ok := key.(string)
		if !ok {
			return true
		}

		item, ok := value.(*CacheItem)
		if !ok || item.IsExpired() {
			return true
		}

		if now.Before(item.ExpiresAt) {
			keys = append(keys, keyStr)
		}

		return true
	})

	return keys
}

// GetItemCount returns the current number of items in the cache
func (c *MemoryCache) GetItemCount() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats.ItemCount
}

// startCleanup starts the background cleanup goroutine
func (c *MemoryCache) startCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.cleanupDone:
			return
		}
	}
}

// cleanup removes expired items from the cache
func (c *MemoryCache) cleanup() {
	now := time.Now().UTC()
	var expiredKeys []interface{}
	evicted := int64(0)

	// Collect expired keys
	c.expiry.Range(func(key, value interface{}) bool {
		expiryTime, ok := value.(time.Time)
		if !ok {
			expiredKeys = append(expiredKeys, key)
			return true
		}

		if now.After(expiryTime) {
			expiredKeys = append(expiredKeys, key)
		}

		return true
	})

	// Remove expired items
	for _, key := range expiredKeys {
		c.data.Delete(key)
		c.expiry.Delete(key)
		evicted++
	}

	// Update stats
	c.mu.Lock()
	c.stats.Evictions += evicted
	c.stats.ItemCount -= evicted
	if c.stats.ItemCount < 0 {
		c.stats.ItemCount = 0
	}
	c.stats.LastCleanup = now
	c.mu.Unlock()

	if evicted > 0 {
		logrus.WithFields(logrus.Fields{
			"evicted_items": evicted,
			"total_items":   c.stats.ItemCount,
		}).Debug("Cache cleanup completed")
	}
}

// Stop stops all cache background workers
func (c *MemoryCache) Stop() {
	close(c.cleanupDone)
	logrus.WithFields(logrus.Fields{
		"final_item_count": c.GetItemCount(),
		"final_hit_ratio":  c.GetHitRatio(),
		"memory_usage_mb":  c.estimateMemoryUsage() / 1024 / 1024,
	}).Info("High-performance memory cache stopped")
}

// GetWithFunc retrieves a value from cache or computes it using the provided function
func (c *MemoryCache) GetWithFunc(key string, fn func() (interface{}, error), ttl time.Duration) (interface{}, error) {
	// Try to get from cache first
	if value, exists := c.Get(key); exists {
		return value, nil
	}

	// Compute value
	value, err := fn()
	if err != nil {
		return nil, err
	}

	// Store in cache
	if err := c.Set(key, value, ttl); err != nil {
		logrus.WithError(err).Warn("Failed to store computed value in cache")
	}

	return value, nil
}

// SetWithContext stores a value with context for cancellation
func (c *MemoryCache) SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return c.Set(key, value, ttl)
	}
}

// GetWithContext retrieves a value with context for cancellation
func (c *MemoryCache) GetWithContext(ctx context.Context, key string) (interface{}, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
		value, exists := c.Get(key)
		return value, exists, nil
	}
}

// Refresh extends the TTL of an existing cache item
func (c *MemoryCache) Refresh(key string, ttl time.Duration) bool {
	value, exists := c.data.Load(key)
	if !exists {
		return false
	}

	item, ok := value.(*CacheItem)
	if !ok {
		return false
	}

	if item.IsExpired() {
		c.Delete(key)
		return false
	}

	// Update expiration time
	newExpiresAt := time.Now().UTC().Add(ttl)
	item.ExpiresAt = newExpiresAt
	c.expiry.Store(key, newExpiresAt)

	return true
}

// recordLatency records operation latency for performance metrics
func (c *MemoryCache) recordLatency(latency time.Duration, operation string) {
	if c.metrics == nil {
		return
	}

	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	switch operation {
	case "get":
		c.metrics.HitLatencies = append(c.metrics.HitLatencies, latency)
		if len(c.metrics.HitLatencies) > 1000 {
			c.metrics.HitLatencies = c.metrics.HitLatencies[500:]
		}
	case "set":
		c.metrics.SetLatencies = append(c.metrics.SetLatencies, latency)
		if len(c.metrics.SetLatencies) > 1000 {
			c.metrics.SetLatencies = c.metrics.SetLatencies[500:]
		}
	}
}

// triggerEviction triggers cache eviction for performance
func (c *MemoryCache) triggerEviction() {
	select {
	case c.evictionCh <- "__trigger_eviction__":
	default:
		// Channel is full, eviction is already in progress
	}
}

// startEvictionWorker starts the background eviction worker
func (c *MemoryCache) startEvictionWorker() {
	for {
		select {
		case key := <-c.evictionCh:
			if key == "__trigger_eviction__" {
				c.performLRUEviction()
			} else {
				c.Delete(key)
			}
		case <-c.cleanupDone:
			return
		}
	}
}

// performLRUEviction performs Least Recently Used eviction
func (c *MemoryCache) performLRUEviction() {
	var oldestKeys []string
	oldestTime := time.Now()

	// Find oldest items (simplified LRU implementation)
	c.data.Range(func(key, value interface{}) bool {
		item, ok := value.(*CacheItem)
		if ok && item.CreatedAt.Before(oldestTime) {
			keyStr, ok := key.(string)
			if ok {
				oldestKeys = append(oldestKeys, keyStr)
				if len(oldestKeys) > 100 { // Evict max 100 items at once
					return false
				}
			}
		}
		return true
	})

	// Delete oldest items
	for _, key := range oldestKeys {
		c.Delete(key)
	}

	if len(oldestKeys) > 0 {
		logrus.WithField("evicted_count", len(oldestKeys)).Debug("LRU eviction completed")
	}
}

// startPreloadWorker starts the background cache preload worker
func (c *MemoryCache) startPreloadWorker() {
	for {
		select {
		case req := <-c.preloadCh:
			c.handlePreloadRequest(req)
		case <-c.cleanupDone:
			return
		}
	}
}

// handlePreloadRequest handles a cache preload request
func (c *MemoryCache) handlePreloadRequest(req PreloadRequest) {
	value, err := req.Loader()
	if err != nil {
		if req.Callback != nil {
			req.Callback(err)
		}
		return
	}

	if err := c.Set(req.Key, value, req.TTL); err != nil {
		if req.Callback != nil {
			req.Callback(err)
		}
		return
	}

	if req.Callback != nil {
		req.Callback(nil)
	}
}

// PreloadAsync asynchronously preloads a cache entry
func (c *MemoryCache) PreloadAsync(key string, loader func() (interface{}, error), ttl time.Duration, callback func(error)) {
	req := PreloadRequest{
		Key:      key,
		Loader:   loader,
		TTL:      ttl,
		Callback: callback,
	}

	select {
	case c.preloadCh <- req:
	default:
		// Channel full, execute synchronously
		c.handlePreloadRequest(req)
	}
}

// startMetricsCollector starts the background metrics collector
func (c *MemoryCache) startMetricsCollector() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collectMetrics()
		case <-c.cleanupDone:
			return
		}
	}
}

// collectMetrics collects and analyzes cache performance metrics
func (c *MemoryCache) collectMetrics() {
	if c.metrics == nil {
		return
	}

	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	// Calculate average latencies
	var avgHitLatency, avgSetLatency time.Duration
	if len(c.metrics.HitLatencies) > 0 {
		var total time.Duration
		for _, lat := range c.metrics.HitLatencies {
			total += lat
		}
		avgHitLatency = total / time.Duration(len(c.metrics.HitLatencies))
	}

	if len(c.metrics.SetLatencies) > 0 {
		var total time.Duration
		for _, lat := range c.metrics.SetLatencies {
			total += lat
		}
		avgSetLatency = total / time.Duration(len(c.metrics.SetLatencies))
	}

	// Log performance metrics
	c.mu.RLock()
	hitRatio := c.GetHitRatio()
	itemCount := c.stats.ItemCount
	c.mu.RUnlock()

	logrus.WithFields(logrus.Fields{
		"hit_ratio":         hitRatio,
		"item_count":        itemCount,
		"avg_hit_latency":   avgHitLatency,
		"avg_set_latency":   avgSetLatency,
		"max_items":         c.maxItems,
		"memory_usage_mb":   c.estimateMemoryUsage() / 1024 / 1024,
	}).Debug("Cache performance metrics")

	c.metrics.LastOptimization = time.Now()
}

// estimateMemoryUsage estimates the memory usage of the cache
func (c *MemoryCache) estimateMemoryUsage() int64 {
	var memoryUsage int64
	c.data.Range(func(key, value interface{}) bool {
		// Rough estimation: key + value + metadata
		memoryUsage += int64(len(fmt.Sprintf("%v", key))) + 100 // Key + metadata overhead
		if item, ok := value.(*CacheItem); ok {
			memoryUsage += int64(len(fmt.Sprintf("%v", item.Value))) + 100 // Value + CacheItem overhead
		}
		return true
	})
	return memoryUsage
}

// GetAdvancedStats returns detailed cache statistics
func (c *MemoryCache) GetAdvancedStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]interface{}{
		"basic_stats":    c.stats,
		"hit_ratio":      c.GetHitRatio(),
		"memory_usage":   c.estimateMemoryUsage(),
		"max_items":      c.maxItems,
		"default_ttl":    c.defaultTTL,
	}

	if c.metrics != nil {
		c.metrics.mu.RLock()
		stats["metrics"] = map[string]interface{}{
			"hit_latency_samples":  len(c.metrics.HitLatencies),
			"set_latency_samples":  len(c.metrics.SetLatencies),
			"last_optimization":    c.metrics.LastOptimization,
		}
		c.metrics.mu.RUnlock()
	}

	return stats
}