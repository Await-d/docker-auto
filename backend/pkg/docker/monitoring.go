package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

// ContainerMonitor manages real-time container monitoring with Docker stats API
type ContainerMonitor struct {
	client           *DockerClient
	monitoredContainers map[string]*MonitoringSession
	mu               sync.RWMutex
	metrics          *MonitoringMetrics
	cache            *MonitoringCache
	updateInterval   time.Duration
	maxHistorySize   int
	logger           *logrus.Entry
}

// MonitoringSession represents an active monitoring session for a container
type MonitoringSession struct {
	containerID    string
	containerName  string
	startTime      time.Time
	isRunning      bool
	cancel         context.CancelFunc
	metricsChan    chan *ContainerMetrics
	errorChan      chan error
	lastMetrics    *ContainerMetrics
	statsStream    chan *types.StatsJSON
	mu             sync.RWMutex
}

// MonitoringMetrics tracks performance metrics for the monitoring system itself
type MonitoringMetrics struct {
	mu                    sync.RWMutex
	ActiveSessions        int64                 `json:"active_sessions"`
	TotalSessionsStarted  int64                 `json:"total_sessions_started"`
	TotalSessionsClosed   int64                 `json:"total_sessions_closed"`
	MetricsCollected      int64                 `json:"metrics_collected"`
	ErrorsEncountered     int64                 `json:"errors_encountered"`
	CacheHits             int64                 `json:"cache_hits"`
	CacheMisses          int64                 `json:"cache_misses"`
	LastMetricsUpdate     time.Time             `json:"last_metrics_update"`
	AverageCollectionTime time.Duration         `json:"average_collection_time"`
	SessionStats          map[string]*SessionStats `json:"session_stats"`
}

// SessionStats tracks per-container session statistics
type SessionStats struct {
	ContainerID       string        `json:"container_id"`
	SessionStart      time.Time     `json:"session_start"`
	MetricsCount      int64         `json:"metrics_count"`
	ErrorCount        int64         `json:"error_count"`
	LastUpdate        time.Time     `json:"last_update"`
	AverageInterval   time.Duration `json:"average_interval"`
	IsActive          bool          `json:"is_active"`
}

// MonitoringCache provides efficient caching for monitoring data
type MonitoringCache struct {
	mu              sync.RWMutex
	containerMetrics map[string]*CachedMetrics
	systemInfo       *CachedSystemInfo
	maxCacheSize     int
	ttl              time.Duration
}

// CachedMetrics represents cached container metrics with expiry
type CachedMetrics struct {
	Metrics      *ContainerMetrics `json:"metrics"`
	Timestamp    time.Time         `json:"timestamp"`
	IsValid      bool              `json:"is_valid"`
	AccessCount  int64             `json:"access_count"`
	LastAccessed time.Time         `json:"last_accessed"`
}

// CachedSystemInfo represents cached system information
type CachedSystemInfo struct {
	Info         *types.Info `json:"info"`
	Timestamp    time.Time   `json:"timestamp"`
	IsValid      bool        `json:"is_valid"`
	AccessCount  int64       `json:"access_count"`
	LastAccessed time.Time   `json:"last_accessed"`
}

// MonitoringConfig configures the container monitor behavior
type MonitoringConfig struct {
	UpdateInterval   time.Duration `json:"update_interval"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	MaxCacheSize     int           `json:"max_cache_size"`
	MaxHistorySize   int           `json:"max_history_size"`
	EnableMetrics    bool          `json:"enable_metrics"`
	BufferSize       int           `json:"buffer_size"`
}

// DefaultMonitoringConfig returns the default monitoring configuration
func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		UpdateInterval:   2 * time.Second,
		CacheTTL:         30 * time.Second,
		MaxCacheSize:     1000,
		MaxHistorySize:   100,
		EnableMetrics:    true,
		BufferSize:       100,
	}
}

// NewContainerMonitor creates a new container monitor with production-grade features
func NewContainerMonitor(client *DockerClient, config *MonitoringConfig) *ContainerMonitor {
	if config == nil {
		config = DefaultMonitoringConfig()
	}

	monitor := &ContainerMonitor{
		client:              client,
		monitoredContainers: make(map[string]*MonitoringSession),
		updateInterval:      config.UpdateInterval,
		maxHistorySize:      config.MaxHistorySize,
		logger:              logrus.WithField("component", "container_monitor"),
		metrics: &MonitoringMetrics{
			SessionStats:      make(map[string]*SessionStats),
			LastMetricsUpdate: time.Now(),
		},
		cache: &MonitoringCache{
			containerMetrics: make(map[string]*CachedMetrics),
			maxCacheSize:     config.MaxCacheSize,
			ttl:              config.CacheTTL,
		},
	}

	// Start cleanup goroutine for expired cache entries
	go monitor.cacheCleanupWorker()

	monitor.logger.WithFields(logrus.Fields{
		"update_interval":  config.UpdateInterval,
		"cache_ttl":        config.CacheTTL,
		"max_cache_size":   config.MaxCacheSize,
		"max_history_size": config.MaxHistorySize,
	}).Info("Container monitor initialized")

	return monitor
}

// StartMonitoring starts monitoring for a specific container
func (m *ContainerMonitor) StartMonitoring(ctx context.Context, containerID string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already monitoring this container
	if session, exists := m.monitoredContainers[containerID]; exists {
		if session.isRunning {
			return fmt.Errorf("container %s is already being monitored", containerID)
		}
	}

	// Get container information
	containerInfo, err := m.client.GetContainer(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to get container info: %w", err)
	}

	// Create monitoring context with cancel function
	monitorCtx, cancel := context.WithCancel(ctx)

	// Create monitoring session
	session := &MonitoringSession{
		containerID:   containerID,
		containerName: containerInfo.Name,
		startTime:     time.Now(),
		isRunning:     true,
		cancel:        cancel,
		metricsChan:   make(chan *ContainerMetrics, 100),
		errorChan:     make(chan error, 10),
		statsStream:   make(chan *types.StatsJSON, 100),
	}

	m.monitoredContainers[containerID] = session

	// Update monitoring metrics
	m.metrics.mu.Lock()
	m.metrics.ActiveSessions++
	m.metrics.TotalSessionsStarted++
	m.metrics.SessionStats[containerID] = &SessionStats{
		ContainerID:  containerID,
		SessionStart: time.Now(),
		IsActive:     true,
	}
	m.metrics.mu.Unlock()

	// Start monitoring goroutine
	go m.monitorContainer(monitorCtx, session)

	m.logger.WithFields(logrus.Fields{
		"container_id":   containerID,
		"container_name": containerInfo.Name,
	}).Info("Started monitoring container")

	return nil
}

// StopMonitoring stops monitoring for a specific container
func (m *ContainerMonitor) StopMonitoring(containerID string) error {
	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.monitoredContainers[containerID]
	if !exists {
		return fmt.Errorf("container %s is not being monitored", containerID)
	}

	// Cancel the monitoring context
	session.cancel()
	session.mu.Lock()
	session.isRunning = false
	session.mu.Unlock()

	// Clean up channels
	close(session.metricsChan)
	close(session.errorChan)
	close(session.statsStream)

	// Remove from monitored containers
	delete(m.monitoredContainers, containerID)

	// Update monitoring metrics
	m.metrics.mu.Lock()
	m.metrics.ActiveSessions--
	m.metrics.TotalSessionsClosed++
	if stats, exists := m.metrics.SessionStats[containerID]; exists {
		stats.IsActive = false
	}
	m.metrics.mu.Unlock()

	m.logger.WithField("container_id", containerID).Info("Stopped monitoring container")

	return nil
}

// GetContainerMetrics retrieves cached or fresh container metrics
func (m *ContainerMonitor) GetContainerMetrics(ctx context.Context, containerID string) (*ContainerMetrics, error) {
	if containerID == "" {
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	// Try to get from cache first
	m.cache.mu.RLock()
	if cached, exists := m.cache.containerMetrics[containerID]; exists {
		if cached.IsValid && time.Since(cached.Timestamp) < m.cache.ttl {
			cached.AccessCount++
			cached.LastAccessed = time.Now()
			m.cache.mu.RUnlock()

			// Update cache metrics
			m.metrics.mu.Lock()
			m.metrics.CacheHits++
			m.metrics.mu.Unlock()

			return cached.Metrics, nil
		}
	}
	m.cache.mu.RUnlock()

	// Cache miss - get fresh metrics
	m.metrics.mu.Lock()
	m.metrics.CacheMisses++
	m.metrics.mu.Unlock()

	start := time.Now()
	metrics, err := m.client.GetContainerMetrics(ctx, containerID)
	if err != nil {
		m.metrics.mu.Lock()
		m.metrics.ErrorsEncountered++
		m.metrics.mu.Unlock()
		return nil, fmt.Errorf("failed to get container metrics: %w", err)
	}

	// Update collection time metrics
	collectTime := time.Since(start)
	m.metrics.mu.Lock()
	m.metrics.MetricsCollected++
	if m.metrics.AverageCollectionTime == 0 {
		m.metrics.AverageCollectionTime = collectTime
	} else {
		m.metrics.AverageCollectionTime = (m.metrics.AverageCollectionTime + collectTime) / 2
	}
	m.metrics.LastMetricsUpdate = time.Now()
	m.metrics.mu.Unlock()

	// Cache the metrics
	m.cacheContainerMetrics(containerID, metrics)

	return metrics, nil
}

// GetAllMonitoredContainers returns metrics for all monitored containers
func (m *ContainerMonitor) GetAllMonitoredContainers(ctx context.Context) (map[string]*ContainerMetrics, error) {
	m.mu.RLock()
	containerIDs := make([]string, 0, len(m.monitoredContainers))
	for containerID := range m.monitoredContainers {
		containerIDs = append(containerIDs, containerID)
	}
	m.mu.RUnlock()

	result := make(map[string]*ContainerMetrics)
	for _, containerID := range containerIDs {
		metrics, err := m.GetContainerMetrics(ctx, containerID)
		if err != nil {
			m.logger.WithError(err).WithField("container_id", containerID).Warn("Failed to get metrics for monitored container")
			continue
		}
		result[containerID] = metrics
	}

	return result, nil
}

// GetMonitoringMetrics returns monitoring system performance metrics
func (m *ContainerMonitor) GetMonitoringMetrics() *MonitoringMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// Create a deep copy of metrics
	metrics := &MonitoringMetrics{
		ActiveSessions:        m.metrics.ActiveSessions,
		TotalSessionsStarted:  m.metrics.TotalSessionsStarted,
		TotalSessionsClosed:   m.metrics.TotalSessionsClosed,
		MetricsCollected:      m.metrics.MetricsCollected,
		ErrorsEncountered:     m.metrics.ErrorsEncountered,
		CacheHits:             m.metrics.CacheHits,
		CacheMisses:           m.metrics.CacheMisses,
		LastMetricsUpdate:     m.metrics.LastMetricsUpdate,
		AverageCollectionTime: m.metrics.AverageCollectionTime,
		SessionStats:          make(map[string]*SessionStats),
	}

	// Copy session stats
	for k, v := range m.metrics.SessionStats {
		metrics.SessionStats[k] = &SessionStats{
			ContainerID:     v.ContainerID,
			SessionStart:    v.SessionStart,
			MetricsCount:    v.MetricsCount,
			ErrorCount:      v.ErrorCount,
			LastUpdate:      v.LastUpdate,
			AverageInterval: v.AverageInterval,
			IsActive:        v.IsActive,
		}
	}

	return metrics
}

// IsMonitoring checks if a container is being monitored
func (m *ContainerMonitor) IsMonitoring(containerID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.monitoredContainers[containerID]
	if !exists {
		return false
	}

	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.isRunning
}

// GetMonitoredContainerIDs returns list of all monitored container IDs
func (m *ContainerMonitor) GetMonitoredContainerIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.monitoredContainers))
	for id := range m.monitoredContainers {
		ids = append(ids, id)
	}
	return ids
}

// monitorContainer runs the monitoring loop for a specific container
func (m *ContainerMonitor) monitorContainer(ctx context.Context, session *MonitoringSession) {
	defer func() {
		m.logger.WithField("container_id", session.containerID).Debug("Monitoring goroutine finished")
	}()

	ticker := time.NewTicker(m.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics, err := m.client.GetContainerMetrics(ctx, session.containerID)
			if err != nil {
				session.errorChan <- fmt.Errorf("failed to get metrics: %w", err)

				// Update error count
				m.metrics.mu.Lock()
				m.metrics.ErrorsEncountered++
				if stats, exists := m.metrics.SessionStats[session.containerID]; exists {
					stats.ErrorCount++
				}
				m.metrics.mu.Unlock()
				continue
			}

			// Update session last metrics
			session.mu.Lock()
			session.lastMetrics = metrics
			session.mu.Unlock()

			// Send metrics to channel (non-blocking)
			select {
			case session.metricsChan <- metrics:
			default:
				// Channel full, skip this update
				m.logger.WithField("container_id", session.containerID).Warn("Metrics channel full, skipping update")
			}

			// Cache the metrics
			m.cacheContainerMetrics(session.containerID, metrics)

			// Update session statistics
			m.metrics.mu.Lock()
			m.metrics.MetricsCollected++
			if stats, exists := m.metrics.SessionStats[session.containerID]; exists {
				stats.MetricsCount++
				stats.LastUpdate = time.Now()
				if stats.MetricsCount > 1 {
					interval := time.Since(stats.SessionStart) / time.Duration(stats.MetricsCount)
					stats.AverageInterval = interval
				}
			}
			m.metrics.mu.Unlock()
		}
	}
}

// cacheContainerMetrics stores container metrics in cache
func (m *ContainerMonitor) cacheContainerMetrics(containerID string, metrics *ContainerMetrics) {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()

	// Check cache size limit
	if len(m.cache.containerMetrics) >= m.cache.maxCacheSize {
		// Remove oldest entries (simple LRU)
		var oldestKey string
		var oldestTime time.Time
		for key, cached := range m.cache.containerMetrics {
			if oldestKey == "" || cached.LastAccessed.Before(oldestTime) {
				oldestKey = key
				oldestTime = cached.LastAccessed
			}
		}
		delete(m.cache.containerMetrics, oldestKey)
	}

	m.cache.containerMetrics[containerID] = &CachedMetrics{
		Metrics:      metrics,
		Timestamp:    time.Now(),
		IsValid:      true,
		AccessCount:  1,
		LastAccessed: time.Now(),
	}
}

// cacheCleanupWorker periodically cleans up expired cache entries
func (m *ContainerMonitor) cacheCleanupWorker() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredCache()
		}
	}
}

// cleanupExpiredCache removes expired entries from cache
func (m *ContainerMonitor) cleanupExpiredCache() {
	m.cache.mu.Lock()
	defer m.cache.mu.Unlock()

	now := time.Now()
	for containerID, cached := range m.cache.containerMetrics {
		if now.Sub(cached.Timestamp) > m.cache.ttl {
			cached.IsValid = false
			delete(m.cache.containerMetrics, containerID)
		}
	}

	// Clean up system info cache
	if m.cache.systemInfo != nil && now.Sub(m.cache.systemInfo.Timestamp) > m.cache.ttl {
		m.cache.systemInfo.IsValid = false
		m.cache.systemInfo = nil
	}
}

// Close gracefully shuts down the container monitor
func (m *ContainerMonitor) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop all monitoring sessions
	for containerID := range m.monitoredContainers {
		if err := m.StopMonitoring(containerID); err != nil {
			m.logger.WithError(err).WithField("container_id", containerID).Warn("Failed to stop monitoring session during shutdown")
		}
	}

	m.logger.Info("Container monitor shut down")
	return nil
}

// GetCacheStats returns cache performance statistics
func (m *ContainerMonitor) GetCacheStats() map[string]interface{} {
	m.cache.mu.RLock()
	defer m.cache.mu.RUnlock()

	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	return map[string]interface{}{
		"cache_size":      len(m.cache.containerMetrics),
		"max_cache_size":  m.cache.maxCacheSize,
		"cache_ttl":       m.cache.ttl.String(),
		"cache_hits":      m.metrics.CacheHits,
		"cache_misses":    m.metrics.CacheMisses,
		"hit_ratio":       float64(m.metrics.CacheHits) / float64(m.metrics.CacheHits + m.metrics.CacheMisses),
	}
}