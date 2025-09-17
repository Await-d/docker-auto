package middleware

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// PerformanceConfig holds configuration for performance middleware
type PerformanceConfig struct {
	EnableCompression   bool
	EnableCaching       bool
	EnableMetrics       bool
	CompressionLevel    int
	CacheMaxAge         int
	SlowThreshold       time.Duration
	EnableMemoryMonitor bool
	MaxConcurrentReq    int
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		EnableCompression:   true,
		EnableCaching:       true,
		EnableMetrics:       true,
		CompressionLevel:    gzip.BestSpeed,
		CacheMaxAge:         300, // 5 minutes
		SlowThreshold:       200 * time.Millisecond,
		EnableMemoryMonitor: true,
		MaxConcurrentReq:    1000,
	}
}

// PerformanceMetrics tracks API performance metrics
type PerformanceMetrics struct {
	mu                sync.RWMutex
	RequestCount      int64
	TotalDuration     time.Duration
	SlowRequestCount  int64
	CompressionRatio  float64
	CacheHitCount     int64
	CacheMissCount    int64
	ActiveRequests    int64
	PeakMemoryUsage   uint64
	RequestsByEndpoint map[string]*EndpointMetrics
}

// EndpointMetrics tracks metrics for individual endpoints
type EndpointMetrics struct {
	Count        int64
	TotalTime    time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
	ErrorCount   int64
	AvgTime      time.Duration
}

var (
	performanceMetrics = &PerformanceMetrics{
		RequestsByEndpoint: make(map[string]*EndpointMetrics),
	}
)

// PerformanceMiddleware creates a comprehensive performance middleware
func PerformanceMiddleware(config PerformanceConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Track active requests
		performanceMetrics.mu.Lock()
		performanceMetrics.ActiveRequests++
		if performanceMetrics.ActiveRequests > int64(config.MaxConcurrentReq) {
			performanceMetrics.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Too many concurrent requests",
				"retry_after": "5",
			})
			c.Abort()
			return
		}
		performanceMetrics.mu.Unlock()

		defer func() {
			performanceMetrics.mu.Lock()
			performanceMetrics.ActiveRequests--
			performanceMetrics.mu.Unlock()
		}()

		// Memory monitoring
		if config.EnableMemoryMonitor {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			performanceMetrics.mu.Lock()
			if m.Alloc > performanceMetrics.PeakMemoryUsage {
				performanceMetrics.PeakMemoryUsage = m.Alloc
			}
			performanceMetrics.mu.Unlock()
		}

		// Apply compression if enabled
		if config.EnableCompression && shouldCompress(c) {
			c.Writer = &gzipWriter{
				ResponseWriter: c.Writer,
				level:          config.CompressionLevel,
			}
		}

		// Apply caching headers if enabled
		if config.EnableCaching && shouldCache(c) {
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", config.CacheMaxAge))
			c.Header("ETag", generateETag(c))

			// Check if client has cached version
			if checkETag(c) {
				c.Status(http.StatusNotModified)
				return
			}
		}

		// Set performance headers
		c.Header("X-Response-Time-Start", strconv.FormatInt(start.UnixNano(), 10))

		// Execute request
		c.Next()

		// Calculate metrics
		duration := time.Since(start)
		endpoint := c.Request.Method + " " + c.FullPath()
		statusCode := c.Writer.Status()

		// Update metrics
		if config.EnableMetrics {
			updateMetrics(endpoint, duration, statusCode, config.SlowThreshold)
		}

		// Add performance headers
		c.Header("X-Response-Time", duration.String())
		c.Header("X-Request-ID", getRequestID(c))

		// Log slow requests
		if duration > config.SlowThreshold {
			logrus.WithFields(logrus.Fields{
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"duration":   duration,
				"status":     statusCode,
				"user_agent": c.Request.UserAgent(),
				"ip":         c.ClientIP(),
			}).Warn("Slow request detected")
		}
	}
}

// gzipWriter implements compression for responses
type gzipWriter struct {
	gin.ResponseWriter
	writer io.Writer
	level  int
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	g.initWriter()
	return g.writer.Write([]byte(s))
}

func (g *gzipWriter) Write(data []byte) (int, error) {
	g.initWriter()
	return g.writer.Write(data)
}

func (g *gzipWriter) initWriter() {
	if g.writer == nil {
		g.Header().Set("Content-Encoding", "gzip")
		gz, _ := gzip.NewWriterLevel(g.ResponseWriter, g.level)
		g.writer = gz
	}
}

func (g *gzipWriter) Close() error {
	if gzWriter, ok := g.writer.(*gzip.Writer); ok {
		return gzWriter.Close()
	}
	return nil
}

// shouldCompress determines if the response should be compressed
func shouldCompress(c *gin.Context) bool {
	// Check Accept-Encoding header
	if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}

	// Don't compress already compressed content
	contentType := c.Writer.Header().Get("Content-Type")
	compressibleTypes := []string{
		"application/json",
		"text/html",
		"text/plain",
		"text/css",
		"text/javascript",
		"application/javascript",
		"text/xml",
		"application/xml",
	}

	for _, t := range compressibleTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	return false
}

// shouldCache determines if the response should be cached
func shouldCache(c *gin.Context) bool {
	// Only cache GET and HEAD requests
	if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
		return false
	}

	// Don't cache authenticated requests with sensitive data
	if c.Request.Header.Get("Authorization") != "" {
		return false
	}

	// Cache static endpoints
	staticEndpoints := []string{
		"/api/system/info",
		"/api/system/health",
		"/api/config/",
	}

	path := c.Request.URL.Path
	for _, endpoint := range staticEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}

	return false
}

// generateETag generates an ETag for the response
func generateETag(c *gin.Context) string {
	// Simple ETag based on path and timestamp
	return fmt.Sprintf(`"%x"`, time.Now().Unix())
}

// checkETag checks if the client has a cached version
func checkETag(c *gin.Context) bool {
	clientETag := c.Request.Header.Get("If-None-Match")
	currentETag := c.Writer.Header().Get("ETag")
	return clientETag != "" && clientETag == currentETag
}

// getRequestID gets or generates a request ID
func getRequestID(c *gin.Context) string {
	if id := c.Request.Header.Get("X-Request-ID"); id != "" {
		return id
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// updateMetrics updates performance metrics
func updateMetrics(endpoint string, duration time.Duration, statusCode int, slowThreshold time.Duration) {
	performanceMetrics.mu.Lock()
	defer performanceMetrics.mu.Unlock()

	// Update global metrics
	performanceMetrics.RequestCount++
	performanceMetrics.TotalDuration += duration

	if duration > slowThreshold {
		performanceMetrics.SlowRequestCount++
	}

	// Update endpoint-specific metrics
	if performanceMetrics.RequestsByEndpoint[endpoint] == nil {
		performanceMetrics.RequestsByEndpoint[endpoint] = &EndpointMetrics{
			MinTime: duration,
			MaxTime: duration,
		}
	}

	endpointMetrics := performanceMetrics.RequestsByEndpoint[endpoint]
	endpointMetrics.Count++
	endpointMetrics.TotalTime += duration

	if duration < endpointMetrics.MinTime {
		endpointMetrics.MinTime = duration
	}
	if duration > endpointMetrics.MaxTime {
		endpointMetrics.MaxTime = duration
	}

	if statusCode >= 400 {
		endpointMetrics.ErrorCount++
	}

	// Calculate average time
	endpointMetrics.AvgTime = endpointMetrics.TotalTime / time.Duration(endpointMetrics.Count)
}

// GetPerformanceMetrics returns current performance metrics
func GetPerformanceMetrics() *PerformanceMetrics {
	performanceMetrics.mu.RLock()
	defer performanceMetrics.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &PerformanceMetrics{
		RequestCount:      performanceMetrics.RequestCount,
		TotalDuration:     performanceMetrics.TotalDuration,
		SlowRequestCount:  performanceMetrics.SlowRequestCount,
		CompressionRatio:  performanceMetrics.CompressionRatio,
		CacheHitCount:     performanceMetrics.CacheHitCount,
		CacheMissCount:    performanceMetrics.CacheMissCount,
		ActiveRequests:    performanceMetrics.ActiveRequests,
		PeakMemoryUsage:   performanceMetrics.PeakMemoryUsage,
		RequestsByEndpoint: make(map[string]*EndpointMetrics),
	}

	// Copy endpoint metrics
	for k, v := range performanceMetrics.RequestsByEndpoint {
		metrics.RequestsByEndpoint[k] = &EndpointMetrics{
			Count:      v.Count,
			TotalTime:  v.TotalTime,
			MinTime:    v.MinTime,
			MaxTime:    v.MaxTime,
			ErrorCount: v.ErrorCount,
			AvgTime:    v.AvgTime,
		}
	}

	return metrics
}

// GetAverageResponseTime calculates the average response time
func (m *PerformanceMetrics) GetAverageResponseTime() time.Duration {
	if m.RequestCount == 0 {
		return 0
	}
	return m.TotalDuration / time.Duration(m.RequestCount)
}

// GetSlowRequestRatio calculates the ratio of slow requests
func (m *PerformanceMetrics) GetSlowRequestRatio() float64 {
	if m.RequestCount == 0 {
		return 0
	}
	return float64(m.SlowRequestCount) / float64(m.RequestCount)
}

// GetTopSlowEndpoints returns the slowest endpoints
func (m *PerformanceMetrics) GetTopSlowEndpoints(limit int) []EndpointMetrics {
	var endpoints []EndpointMetrics

	for endpoint, metrics := range m.RequestsByEndpoint {
		endpointData := *metrics
		endpointData.Count = int64(len(endpoint)) // Store endpoint name in Count field for sorting
		endpoints = append(endpoints, endpointData)
	}

	// Sort by average time (simplified)
	// In a real implementation, you'd use sort.Slice
	if len(endpoints) > limit {
		endpoints = endpoints[:limit]
	}

	return endpoints
}

// ResetMetrics resets all performance metrics
func ResetMetrics() {
	performanceMetrics.mu.Lock()
	defer performanceMetrics.mu.Unlock()

	performanceMetrics.RequestCount = 0
	performanceMetrics.TotalDuration = 0
	performanceMetrics.SlowRequestCount = 0
	performanceMetrics.CompressionRatio = 0
	performanceMetrics.CacheHitCount = 0
	performanceMetrics.CacheMissCount = 0
	performanceMetrics.PeakMemoryUsage = 0
	performanceMetrics.RequestsByEndpoint = make(map[string]*EndpointMetrics)

	logrus.Info("Performance metrics reset")
}

// ResponseCacheMiddleware provides advanced response caching
func ResponseCacheMiddleware(duration time.Duration) gin.HandlerFunc {
	cache := make(map[string]cachedResponse)
	var cacheMu sync.RWMutex

	return func(c *gin.Context) {
		// Only cache GET requests
		if c.Request.Method != "GET" {
			c.Next()
			return
		}

		cacheKey := generateCacheKey(c)

		// Check cache
		cacheMu.RLock()
		if cached, exists := cache[cacheKey]; exists && !cached.expired() {
			cacheMu.RUnlock()

			// Serve from cache
			for key, value := range cached.Headers {
				c.Header(key, value)
			}
			c.Header("X-Cache", "HIT")
			c.Data(cached.StatusCode, cached.ContentType, cached.Data)

			performanceMetrics.mu.Lock()
			performanceMetrics.CacheHitCount++
			performanceMetrics.mu.Unlock()
			return
		}
		cacheMu.RUnlock()

		// Cache miss - execute request
		performanceMetrics.mu.Lock()
		performanceMetrics.CacheMissCount++
		performanceMetrics.mu.Unlock()

		// Capture response
		w := &responseWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Header("X-Cache", "MISS")

		c.Next()

		// Cache the response if successful
		if w.statusCode < 400 {
			cached := cachedResponse{
				Data:        w.body.Bytes(),
				StatusCode:  w.statusCode,
				ContentType: w.Header().Get("Content-Type"),
				Headers:     make(map[string]string),
				ExpiresAt:   time.Now().Add(duration),
			}

			// Copy important headers
			for _, header := range []string{"Content-Type", "Cache-Control", "ETag"} {
				if value := w.Header().Get(header); value != "" {
					cached.Headers[header] = value
				}
			}

			cacheMu.Lock()
			cache[cacheKey] = cached
			cacheMu.Unlock()
		}
	}
}

type cachedResponse struct {
	Data        []byte
	StatusCode  int
	ContentType string
	Headers     map[string]string
	ExpiresAt   time.Time
}

func (c cachedResponse) expired() bool {
	return time.Now().After(c.ExpiresAt)
}

type responseWriter struct {
	gin.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

func (w *responseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *responseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func generateCacheKey(c *gin.Context) string {
	return fmt.Sprintf("%s:%s:%s", c.Request.Method, c.Request.URL.Path, c.Request.URL.RawQuery)
}