package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimiter implements a sliding window rate limiter
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
	keyFunc  KeyFunction
}

// KeyFunction defines how to extract the rate limiting key from the request
type KeyFunction func(c *gin.Context) string

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Limit      int           // Number of requests allowed
	Window     time.Duration // Time window for the limit
	KeyFunc    KeyFunction   // Function to extract rate limiting key
	SkipPaths  []string      // Paths to skip rate limiting
	Message    string        // Custom rate limit exceeded message
	Headers    bool          // Whether to include rate limit headers
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		keyFunc:  IPBasedKeyFunction,
	}

	// Start cleanup goroutine
	go rl.startCleanup()

	return rl
}

// NewRateLimiterWithConfig creates a rate limiter with configuration
func NewRateLimiterWithConfig(config *RateLimitConfig) *RateLimiter {
	if config.KeyFunc == nil {
		config.KeyFunc = IPBasedKeyFunction
	}

	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    config.Limit,
		window:   config.Window,
		keyFunc:  config.KeyFunc,
	}

	go rl.startCleanup()

	return rl
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return RateLimitMiddlewareWithConfig(limiter, &RateLimitConfig{
		Message: "Too many requests",
		Headers: true,
	})
}

// RateLimitMiddlewareWithConfig creates a rate limiting middleware with configuration
func RateLimitMiddlewareWithConfig(limiter *RateLimiter, config *RateLimitConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip rate limiting for certain paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Extract rate limiting key
		key := limiter.keyFunc(c)
		if key == "" {
			c.Next()
			return
		}

		// Check rate limit
		allowed, remaining, resetTime := limiter.Allow(key)

		// Add rate limit headers if enabled
		if config.Headers {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetTime.Unix(), 10))
		}

		if !allowed {
			logrus.WithFields(logrus.Fields{
				"key":       key,
				"path":      c.Request.URL.Path,
				"method":    c.Request.Method,
				"client_ip": c.ClientIP(),
				"limit":     limiter.limit,
				"window":    limiter.window,
			}).Warn("Rate limit exceeded")

			message := config.Message
			if message == "" {
				message = "Too many requests"
			}

			c.JSON(http.StatusTooManyRequests, utils.ErrorResponseWithDetails(
				message,
				fmt.Sprintf("Rate limit: %d requests per %v", limiter.limit, limiter.window),
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// Allow checks if a request is allowed and returns the current state
func (rl *RateLimiter) Allow(key string) (allowed bool, remaining int, resetTime time.Time) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get existing requests for this key
	requests, exists := rl.requests[key]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// Remove requests outside the window
	validRequests := make([]time.Time, 0, len(requests))
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we can allow this request
	if len(validRequests) >= rl.limit {
		remaining = 0
		resetTime = validRequests[0].Add(rl.window)
		return false, remaining, resetTime
	}

	// Allow the request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests

	remaining = rl.limit - len(validRequests)
	if len(validRequests) > 0 {
		resetTime = validRequests[0].Add(rl.window)
	} else {
		resetTime = now.Add(rl.window)
	}

	return true, remaining, resetTime
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	totalKeys := len(rl.requests)
	totalRequests := 0
	activeKeys := 0
	now := time.Now()
	windowStart := now.Add(-rl.window)

	for _, requests := range rl.requests {
		validCount := 0
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validCount++
			}
		}
		if validCount > 0 {
			activeKeys++
		}
		totalRequests += validCount
	}

	return map[string]interface{}{
		"total_keys":     totalKeys,
		"active_keys":    activeKeys,
		"total_requests": totalRequests,
		"limit":          rl.limit,
		"window_seconds": int(rl.window.Seconds()),
	}
}

// startCleanup starts a goroutine to periodically clean up expired entries
func (rl *RateLimiter) startCleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes expired requests from memory
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	for key, requests := range rl.requests {
		validRequests := make([]time.Time, 0, len(requests))
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.requests, key)
		} else {
			rl.requests[key] = validRequests
		}
	}
}

// Key extraction functions

// IPBasedKeyFunction extracts the client IP as the rate limiting key
func IPBasedKeyFunction(c *gin.Context) string {
	return c.ClientIP()
}

// UserBasedKeyFunction extracts the user ID as the rate limiting key
func UserBasedKeyFunction(c *gin.Context) string {
	if user := GetUserFromContext(c); user != nil {
		return fmt.Sprintf("user:%d", user.UserID)
	}
	// Fallback to IP if no user is authenticated
	return IPBasedKeyFunction(c)
}

// PathBasedKeyFunction combines IP and path for rate limiting
func PathBasedKeyFunction(c *gin.Context) string {
	return fmt.Sprintf("%s:%s", c.ClientIP(), c.Request.URL.Path)
}

// CustomKeyFunction allows custom key extraction with multiple factors
func CustomKeyFunction(factors ...string) KeyFunction {
	return func(c *gin.Context) string {
		parts := make([]string, 0, len(factors))

		for _, factor := range factors {
			switch factor {
			case "ip":
				parts = append(parts, c.ClientIP())
			case "user":
				if user := GetUserFromContext(c); user != nil {
					parts = append(parts, fmt.Sprintf("user:%d", user.UserID))
				}
			case "path":
				parts = append(parts, c.Request.URL.Path)
			case "method":
				parts = append(parts, c.Request.Method)
			case "user_agent":
				parts = append(parts, c.Request.UserAgent())
			}
		}

		if len(parts) == 0 {
			return c.ClientIP() // Fallback to IP
		}

		return fmt.Sprintf("%s", parts)
	}
}

// Pre-configured rate limiters

// NewStrictRateLimiter creates a strict rate limiter (10 requests per minute)
func NewStrictRateLimiter() *RateLimiter {
	return NewRateLimiter(10, time.Minute)
}

// NewModerateRateLimiter creates a moderate rate limiter (60 requests per minute)
func NewModerateRateLimiter() *RateLimiter {
	return NewRateLimiter(60, time.Minute)
}

// NewLenientRateLimiter creates a lenient rate limiter (300 requests per minute)
func NewLenientRateLimiter() *RateLimiter {
	return NewRateLimiter(300, time.Minute)
}

// NewAPIRateLimiter creates a rate limiter for API endpoints (100 requests per minute)
func NewAPIRateLimiter() *RateLimiter {
	return NewRateLimiter(100, time.Minute)
}

// NewAuthRateLimiter creates a rate limiter for authentication endpoints (5 requests per minute)
func NewAuthRateLimiter() *RateLimiter {
	limiter := NewRateLimiter(5, time.Minute)
	limiter.keyFunc = IPBasedKeyFunction // Use IP-based limiting for auth
	return limiter
}

// Specialized middleware functions

// AuthRateLimitMiddleware creates rate limiting specifically for authentication endpoints
func AuthRateLimitMiddleware() gin.HandlerFunc {
	limiter := NewAuthRateLimiter()
	config := &RateLimitConfig{
		Message: "Too many authentication attempts",
		Headers: true,
	}
	return RateLimitMiddlewareWithConfig(limiter, config)
}

// APIRateLimitMiddleware creates rate limiting for general API endpoints
func APIRateLimitMiddleware() gin.HandlerFunc {
	limiter := NewAPIRateLimiter()
	limiter.keyFunc = UserBasedKeyFunction // Use user-based limiting for API
	config := &RateLimitConfig{
		Message: "API rate limit exceeded",
		Headers: true,
		SkipPaths: []string{
			"/health",
			"/metrics",
		},
	}
	return RateLimitMiddlewareWithConfig(limiter, config)
}

// GlobalRateLimitMiddleware creates a global rate limiter for all endpoints
func GlobalRateLimitMiddleware() gin.HandlerFunc {
	limiter := NewModerateRateLimiter()
	config := &RateLimitConfig{
		Message: "Rate limit exceeded",
		Headers: true,
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
	}
	return RateLimitMiddlewareWithConfig(limiter, config)
}