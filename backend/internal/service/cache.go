package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/pkg/utils"

	"github.com/sirupsen/logrus"
)

// CacheService manages all caching operations for the application
type CacheService struct {
	cache       *utils.MemoryCache
	config      *config.Config
	stats       *CacheStats
	statsMutex  sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits         int64 `json:"hits"`
	Misses       int64 `json:"misses"`
	Sets         int64 `json:"sets"`
	Deletes      int64 `json:"deletes"`
	Cleanups     int64 `json:"cleanups"`
	TotalItems   int64 `json:"total_items"`
	LastCleanup  time.Time `json:"last_cleanup"`
	StartTime    time.Time `json:"start_time"`
}

// Cache key prefixes for different data types
const (
	// Image information cache keys
	ImageInfoKeyPrefix    = "image:info:"
	ImageVersionKeyPrefix = "image:version:"
	ImageTagsKeyPrefix    = "image:tags:"

	// System configuration cache keys
	SystemConfigKeyPrefix = "system:config:"
	SystemStatsKeyPrefix  = "system:stats:"

	// Container status cache keys
	ContainerStatusKeyPrefix = "container:status:"
	ContainerListKeyPrefix   = "container:list:"

	// User session cache keys
	UserSessionKeyPrefix = "user:session:"
	UserPermissionPrefix = "user:permission:"

	// API response cache keys
	APIResponseKeyPrefix = "api:response:"

	// Docker info cache keys
	DockerInfoKeyPrefix = "docker:info:"

	// Notification cache keys
	NotificationKeyPrefix = "notification:"
)

// TTL constants for different cache types
const (
	DefaultTTL          = 30 * time.Minute
	ImageInfoTTL        = 6 * time.Hour
	SystemConfigTTL     = 5 * time.Minute
	ContainerStatusTTL  = 2 * time.Minute
	UserSessionTTL      = 24 * time.Hour
	APIResponseTTL      = 1 * time.Minute
	DockerInfoTTL       = 10 * time.Minute
	NotificationTTL     = 1 * time.Hour
)

// NewCacheService creates a new cache service instance
func NewCacheService(config *config.Config) *CacheService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &CacheService{
		cache:  utils.NewMemoryCacheWithConfig(config),
		config: config,
		stats: &CacheStats{
			StartTime: time.Now().UTC(),
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return service
}

// Start initializes and starts the cache service
func (s *CacheService) Start(ctx context.Context) error {
	if !s.config.IsCacheEnabled() {
		logrus.Info("Cache service is disabled")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"default_ttl":       time.Duration(s.config.Cache.DefaultTTLMinutes) * time.Minute,
		"image_cache_ttl":   time.Duration(s.config.Cache.ImageCacheTTLHours) * time.Hour,
		"config_cache_ttl":  time.Duration(s.config.Cache.ConfigCacheTTLMinutes) * time.Minute,
		"cleanup_interval":  time.Duration(s.config.Cache.CleanupIntervalMinutes) * time.Minute,
	}).Info("Cache service started")

	// Start background tasks
	go s.startStatisticsUpdate()
	go s.startHealthCheck()

	return nil
}

// Stop gracefully stops the cache service
func (s *CacheService) Stop() error {
	s.cancel()
	if s.cache != nil {
		s.cache.Stop()
	}
	logrus.Info("Cache service stopped")
	return nil
}

// Generic cache operations

// Set stores a value in the cache with TTL
func (s *CacheService) Set(key string, value interface{}, ttl time.Duration) error {
	if !s.config.IsCacheEnabled() {
		return nil
	}

	if ttl <= 0 {
		ttl = DefaultTTL
	}

	err := s.cache.Set(key, value, ttl)
	if err == nil {
		s.incrementStats("sets")
	}

	logrus.WithFields(logrus.Fields{
		"key": key,
		"ttl": ttl,
	}).Debug("Cache set operation")

	return err
}

// Get retrieves a value from the cache
func (s *CacheService) Get(key string) (interface{}, bool) {
	if !s.config.IsCacheEnabled() {
		return nil, false
	}

	value, exists := s.cache.Get(key)

	if exists {
		s.incrementStats("hits")
		logrus.WithField("key", key).Debug("Cache hit")
	} else {
		s.incrementStats("misses")
		logrus.WithField("key", key).Debug("Cache miss")
	}

	return value, exists
}

// Delete removes a value from the cache
func (s *CacheService) Delete(key string) {
	if !s.config.IsCacheEnabled() {
		return
	}

	s.cache.Delete(key)
	s.incrementStats("deletes")

	logrus.WithField("key", key).Debug("Cache delete operation")
}

// Clear removes all items from the cache
func (s *CacheService) Clear() {
	if !s.config.IsCacheEnabled() {
		return
	}

	s.cache.Clear()
	s.incrementStats("cleanups")

	logrus.Info("Cache cleared")
}

// Image information caching

// SetImageInfo caches image information
func (s *CacheService) SetImageInfo(image string, info *model.ImageVersion) error {
	key := buildImageInfoKey(image)
	ttl := time.Duration(s.config.Cache.ImageCacheTTLHours) * time.Hour
	return s.Set(key, info, ttl)
}

// GetImageInfo retrieves cached image information
func (s *CacheService) GetImageInfo(image string) (*model.ImageVersion, bool) {
	key := buildImageInfoKey(image)
	value, exists := s.Get(key)
	if !exists {
		return nil, false
	}

	if imageInfo, ok := value.(*model.ImageVersion); ok {
		return imageInfo, true
	}

	// If type assertion fails, remove the invalid entry
	s.Delete(key)
	return nil, false
}

// InvalidateImageCache removes image information from cache
func (s *CacheService) InvalidateImageCache(image string) {
	s.Delete(buildImageInfoKey(image))
	s.Delete(buildImageVersionKey(image))
	s.Delete(buildImageTagsKey(image))

	logrus.WithField("image", image).Info("Image cache invalidated")
}

// SetImageTags caches image tags
func (s *CacheService) SetImageTags(image string, tags []string) error {
	key := buildImageTagsKey(image)
	ttl := time.Duration(s.config.Cache.ImageCacheTTLHours) * time.Hour
	return s.Set(key, tags, ttl)
}

// GetImageTags retrieves cached image tags
func (s *CacheService) GetImageTags(image string) ([]string, bool) {
	key := buildImageTagsKey(image)
	value, exists := s.Get(key)
	if !exists {
		return nil, false
	}

	if tags, ok := value.([]string); ok {
		return tags, true
	}

	s.Delete(key)
	return nil, false
}

// System configuration caching

// SetSystemConfig caches system configuration
func (s *CacheService) SetSystemConfig(configs map[string]interface{}) error {
	ttl := time.Duration(s.config.Cache.ConfigCacheTTLMinutes) * time.Minute

	for key, value := range configs {
		cacheKey := buildSystemConfigKey(key)
		if err := s.Set(cacheKey, value, ttl); err != nil {
			logrus.WithError(err).WithField("config_key", key).Warn("Failed to cache system config")
		}
	}

	logrus.WithField("config_count", len(configs)).Debug("System configurations cached")
	return nil
}

// GetSystemConfig retrieves a cached system configuration value
func (s *CacheService) GetSystemConfig(key string) (interface{}, bool) {
	cacheKey := buildSystemConfigKey(key)
	return s.Get(cacheKey)
}

// RefreshSystemConfig invalidates all system configuration cache
func (s *CacheService) RefreshSystemConfig() error {
	// Get all keys and remove system config ones
	keys := s.cache.Keys()
	for _, key := range keys {
		if fmt.Sprintf(key)[0:len(SystemConfigKeyPrefix)] == SystemConfigKeyPrefix {
			s.Delete(key)
		}
	}

	logrus.Info("System configuration cache refreshed")
	return nil
}

// Container status caching

// SetContainerStatus caches container status
func (s *CacheService) SetContainerStatus(containerID int64, status string) error {
	key := buildContainerStatusKey(containerID)
	return s.Set(key, status, ContainerStatusTTL)
}

// GetContainerStatus retrieves cached container status
func (s *CacheService) GetContainerStatus(containerID int64) (string, bool) {
	key := buildContainerStatusKey(containerID)
	value, exists := s.Get(key)
	if !exists {
		return "", false
	}

	if status, ok := value.(string); ok {
		return status, true
	}

	s.Delete(key)
	return "", false
}

// SetContainerList caches container list
func (s *CacheService) SetContainerList(userID int64, containers []model.Container) error {
	key := buildContainerListKey(userID)
	return s.Set(key, containers, ContainerStatusTTL)
}

// GetContainerList retrieves cached container list
func (s *CacheService) GetContainerList(userID int64) ([]model.Container, bool) {
	key := buildContainerListKey(userID)
	value, exists := s.Get(key)
	if !exists {
		return nil, false
	}

	if containers, ok := value.([]model.Container); ok {
		return containers, true
	}

	s.Delete(key)
	return nil, false
}

// User session caching

// SetUserSession caches user session information
func (s *CacheService) SetUserSession(sessionID string, session *model.UserSession) error {
	key := buildUserSessionKey(sessionID)
	return s.Set(key, session, UserSessionTTL)
}

// GetUserSession retrieves cached user session
func (s *CacheService) GetUserSession(sessionID string) (*model.UserSession, bool) {
	key := buildUserSessionKey(sessionID)
	value, exists := s.Get(key)
	if !exists {
		return nil, false
	}

	if session, ok := value.(*model.UserSession); ok {
		return session, true
	}

	s.Delete(key)
	return nil, false
}

// SetUserPermissions caches user permissions
func (s *CacheService) SetUserPermissions(userID int64, permissions []string) error {
	key := buildUserPermissionKey(userID)
	return s.Set(key, permissions, UserSessionTTL)
}

// GetUserPermissions retrieves cached user permissions
func (s *CacheService) GetUserPermissions(userID int64) ([]string, bool) {
	key := buildUserPermissionKey(userID)
	value, exists := s.Get(key)
	if !exists {
		return nil, false
	}

	if permissions, ok := value.([]string); ok {
		return permissions, true
	}

	s.Delete(key)
	return nil, false
}

// API response caching

// SetAPIResponse caches API response
func (s *CacheService) SetAPIResponse(endpoint string, response interface{}) error {
	key := buildAPIResponseKey(endpoint)
	return s.Set(key, response, APIResponseTTL)
}

// GetAPIResponse retrieves cached API response
func (s *CacheService) GetAPIResponse(endpoint string) (interface{}, bool) {
	key := buildAPIResponseKey(endpoint)
	return s.Get(key)
}

// Docker information caching

// SetDockerInfo caches Docker daemon information
func (s *CacheService) SetDockerInfo(info interface{}) error {
	key := DockerInfoKeyPrefix + "daemon"
	return s.Set(key, info, DockerInfoTTL)
}

// GetDockerInfo retrieves cached Docker daemon information
func (s *CacheService) GetDockerInfo() (interface{}, bool) {
	key := DockerInfoKeyPrefix + "daemon"
	return s.Get(key)
}

// Statistics and monitoring

// GetStats returns current cache statistics
func (s *CacheService) GetStats() *CacheStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	// Get current cache stats
	cacheStats := s.cache.GetStats()

	return &CacheStats{
		Hits:        s.stats.Hits + cacheStats.Hits,
		Misses:      s.stats.Misses + cacheStats.Misses,
		Sets:        s.stats.Sets + cacheStats.Sets,
		Deletes:     s.stats.Deletes + cacheStats.Deletes,
		Cleanups:    s.stats.Cleanups + cacheStats.Evictions,
		TotalItems:  cacheStats.ItemCount,
		LastCleanup: cacheStats.LastCleanup,
		StartTime:   s.stats.StartTime,
	}
}

// GetHitRatio returns cache hit ratio as percentage
func (s *CacheService) GetHitRatio() float64 {
	stats := s.GetStats()
	total := stats.Hits + stats.Misses
	if total == 0 {
		return 0
	}
	return (float64(stats.Hits) / float64(total)) * 100
}

// GetMemoryUsage returns estimated memory usage in bytes
func (s *CacheService) GetMemoryUsage() int64 {
	// This is a rough estimation based on item count
	// In a real implementation, you might want more accurate memory tracking
	stats := s.GetStats()
	return stats.TotalItems * 1024 // Assume 1KB per item average
}

// Utility functions

// incrementStats safely increments statistics counters
func (s *CacheService) incrementStats(operation string) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	switch operation {
	case "hits":
		s.stats.Hits++
	case "misses":
		s.stats.Misses++
	case "sets":
		s.stats.Sets++
	case "deletes":
		s.stats.Deletes++
	case "cleanups":
		s.stats.Cleanups++
	}
}

// startStatisticsUpdate starts a goroutine to periodically update cache statistics
func (s *CacheService) startStatisticsUpdate() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats := s.GetStats()
			logrus.WithFields(logrus.Fields{
				"hits":       stats.Hits,
				"misses":     stats.Misses,
				"hit_ratio":  s.GetHitRatio(),
				"total_items": stats.TotalItems,
			}).Debug("Cache statistics update")
		case <-s.ctx.Done():
			return
		}
	}
}

// startHealthCheck starts a goroutine to monitor cache health
func (s *CacheService) startHealthCheck() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.performHealthCheck()
		case <-s.ctx.Done():
			return
		}
	}
}

// performHealthCheck checks cache health and logs warnings if needed
func (s *CacheService) performHealthCheck() {
	stats := s.GetStats()
	hitRatio := s.GetHitRatio()

	// Log warning if hit ratio is very low
	if hitRatio < 50 && stats.Hits+stats.Misses > 100 {
		logrus.WithField("hit_ratio", hitRatio).Warn("Low cache hit ratio detected")
	}

	// Log info about memory usage
	memoryUsage := s.GetMemoryUsage()
	if memoryUsage > 100*1024*1024 { // 100MB
		logrus.WithFields(logrus.Fields{
			"memory_usage_mb": memoryUsage / 1024 / 1024,
			"total_items":     stats.TotalItems,
		}).Info("High cache memory usage")
	}
}

// Key building functions

func buildImageInfoKey(image string) string {
	return ImageInfoKeyPrefix + image
}

func buildImageVersionKey(image string) string {
	return ImageVersionKeyPrefix + image
}

func buildImageTagsKey(image string) string {
	return ImageTagsKeyPrefix + image
}

func buildSystemConfigKey(key string) string {
	return SystemConfigKeyPrefix + key
}

func buildContainerStatusKey(containerID int64) string {
	return ContainerStatusKeyPrefix + strconv.FormatInt(containerID, 10)
}

func buildContainerListKey(userID int64) string {
	return ContainerListKeyPrefix + strconv.FormatInt(userID, 10)
}

func buildUserSessionKey(sessionID string) string {
	return UserSessionKeyPrefix + sessionID
}

func buildUserPermissionKey(userID int64) string {
	return UserPermissionPrefix + strconv.FormatInt(userID, 10)
}

func buildAPIResponseKey(endpoint string) string {
	return APIResponseKeyPrefix + endpoint
}