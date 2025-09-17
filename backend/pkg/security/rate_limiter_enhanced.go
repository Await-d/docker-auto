package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EnhancedRateLimitConfig represents advanced rate limiting configuration
type EnhancedRateLimitConfig struct {
	// Basic rate limiting
	GlobalLimit         int           `json:"global_limit"`
	GlobalWindow        time.Duration `json:"global_window"`

	// Per-user rate limiting
	UserLimit           int           `json:"user_limit"`
	UserWindow          time.Duration `json:"user_window"`

	// Per-endpoint rate limiting
	EndpointLimits      map[string]EndpointLimit `json:"endpoint_limits"`

	// IP-based rate limiting
	IPLimit             int           `json:"ip_limit"`
	IPWindow            time.Duration `json:"ip_window"`
	SubnetLimit         int           `json:"subnet_limit"`
	SubnetWindow        time.Duration `json:"subnet_window"`

	// Burst handling
	BurstMultiplier     float64       `json:"burst_multiplier"`
	BurstWindow         time.Duration `json:"burst_window"`

	// Blacklist/Whitelist
	IPBlacklist         []string      `json:"ip_blacklist"`
	IPWhitelist         []string      `json:"ip_whitelist"`
	UserBlacklist       []int64       `json:"user_blacklist"`
	UserWhitelist       []int64       `json:"user_whitelist"`

	// Dynamic limiting
	EnableDynamicLimits bool          `json:"enable_dynamic_limits"`
	LoadThreshold       float64       `json:"load_threshold"`
	DynamicMultiplier   float64       `json:"dynamic_multiplier"`

	// Ban management
	EnableBanning       bool          `json:"enable_banning"`
	BanThreshold        int           `json:"ban_threshold"`
	BanDuration         time.Duration `json:"ban_duration"`
	MaxBanDuration      time.Duration `json:"max_ban_duration"`

	// Cleanup settings
	CleanupInterval     time.Duration `json:"cleanup_interval"`
	MaxMemoryEntries    int           `json:"max_memory_entries"`
}

// EndpointLimit represents rate limit for specific endpoint
type EndpointLimit struct {
	Limit           int           `json:"limit"`
	Window          time.Duration `json:"window"`
	Methods         []string      `json:"methods"`
	RequireAuth     bool          `json:"require_auth"`
	SkipWhitelist   bool          `json:"skip_whitelist"`
}

// DefaultEnhancedRateLimitConfig returns default configuration
func DefaultEnhancedRateLimitConfig() *EnhancedRateLimitConfig {
	return &EnhancedRateLimitConfig{
		GlobalLimit:         10000,
		GlobalWindow:        time.Hour,
		UserLimit:          1000,
		UserWindow:         time.Hour,
		IPLimit:            100,
		IPWindow:           time.Minute,
		SubnetLimit:        1000,
		SubnetWindow:       time.Hour,
		BurstMultiplier:    2.0,
		BurstWindow:        time.Minute,
		EnableDynamicLimits: true,
		LoadThreshold:      0.8,
		DynamicMultiplier:  0.5,
		EnableBanning:      true,
		BanThreshold:       10,
		BanDuration:        time.Hour,
		MaxBanDuration:     24 * time.Hour,
		CleanupInterval:    15 * time.Minute,
		MaxMemoryEntries:   100000,
		EndpointLimits: map[string]EndpointLimit{
			"/api/auth/login":    {Limit: 5, Window: time.Minute, Methods: []string{"POST"}},
			"/api/auth/register": {Limit: 3, Window: 10 * time.Minute, Methods: []string{"POST"}},
			"/api/auth/refresh":  {Limit: 10, Window: time.Minute, Methods: []string{"POST"}},
			"/api/containers":    {Limit: 100, Window: time.Minute, RequireAuth: true},
			"/api/images":        {Limit: 50, Window: time.Minute, RequireAuth: true},
		},
	}
}

// RateLimitEntry represents a rate limit entry
type RateLimitEntry struct {
	Key          string      `json:"key"`
	Count        int         `json:"count"`
	WindowStart  time.Time   `json:"window_start"`
	LastRequest  time.Time   `json:"last_request"`
	Violations   int         `json:"violations"`
	BannedUntil  *time.Time  `json:"banned_until,omitempty"`
}

// EnhancedRateLimiter implements advanced rate limiting with multiple strategies
type EnhancedRateLimiter struct {
	config          *EnhancedRateLimitConfig
	entries         map[string]*RateLimitEntry
	globalStats     *GlobalStats
	mutex           sync.RWMutex
	cleanupTicker   *time.Ticker
	systemLoad      float64
	systemLoadMutex sync.RWMutex
}

// GlobalStats represents global rate limiting statistics
type GlobalStats struct {
	TotalRequests     int64     `json:"total_requests"`
	BlockedRequests   int64     `json:"blocked_requests"`
	BannedIPs         int       `json:"banned_ips"`
	ActiveEntries     int       `json:"active_entries"`
	LastCleanup       time.Time `json:"last_cleanup"`
	AverageLoad       float64   `json:"average_load"`
}

// NewEnhancedRateLimiter creates a new enhanced rate limiter
func NewEnhancedRateLimiter(config *EnhancedRateLimitConfig) *EnhancedRateLimiter {
	if config == nil {
		config = DefaultEnhancedRateLimitConfig()
	}

	rl := &EnhancedRateLimiter{
		config: config,
		entries: make(map[string]*RateLimitEntry),
		globalStats: &GlobalStats{
			LastCleanup: time.Now(),
		},
		systemLoad: 0.0,
	}

	// Start cleanup goroutine
	rl.startCleanup()

	// Start system load monitoring
	rl.startLoadMonitoring()

	return rl
}

// CheckLimit checks if a request should be allowed
func (rl *EnhancedRateLimiter) CheckLimit(ctx *RateLimitContext) (*RateLimitResult, error) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.globalStats.TotalRequests++

	// Check IP blacklist
	if rl.isBlacklisted(ctx) {
		rl.globalStats.BlockedRequests++
		return &RateLimitResult{
			Allowed:     false,
			Reason:      "IP blacklisted",
			RetryAfter:  time.Hour,
		}, nil
	}

	// Check IP whitelist (bypass other checks)
	if rl.isWhitelisted(ctx) {
		return &RateLimitResult{
			Allowed:    true,
			Reason:     "IP whitelisted",
		}, nil
	}

	// Check if IP/User is banned
	if banned := rl.checkBanned(ctx); banned != nil {
		rl.globalStats.BlockedRequests++
		return banned, nil
	}

	// Apply dynamic limits based on system load
	currentLimits := rl.calculateDynamicLimits()

	// Check global rate limit
	if globalResult := rl.checkGlobalLimit(currentLimits); !globalResult.Allowed {
		rl.globalStats.BlockedRequests++
		rl.recordViolation(ctx)
		return globalResult, nil
	}

	// Check endpoint-specific limits
	if endpointResult := rl.checkEndpointLimit(ctx, currentLimits); !endpointResult.Allowed {
		rl.globalStats.BlockedRequests++
		rl.recordViolation(ctx)
		return endpointResult, nil
	}

	// Check IP-based limits
	if ipResult := rl.checkIPLimit(ctx, currentLimits); !ipResult.Allowed {
		rl.globalStats.BlockedRequests++
		rl.recordViolation(ctx)
		return ipResult, nil
	}

	// Check user-based limits (if authenticated)
	if ctx.UserID > 0 {
		if userResult := rl.checkUserLimit(ctx, currentLimits); !userResult.Allowed {
			rl.globalStats.BlockedRequests++
			rl.recordViolation(ctx)
			return userResult, nil
		}
	}

	// Check subnet limits
	if subnetResult := rl.checkSubnetLimit(ctx, currentLimits); !subnetResult.Allowed {
		rl.globalStats.BlockedRequests++
		rl.recordViolation(ctx)
		return subnetResult, nil
	}

	// Update counters for allowed request
	rl.updateCounters(ctx)

	return &RateLimitResult{
		Allowed:    true,
		Remaining:  rl.calculateRemaining(ctx, currentLimits),
		ResetTime:  rl.calculateResetTime(ctx),
	}, nil
}

// RateLimitContext represents the context for rate limiting
type RateLimitContext struct {
	IP         string    `json:"ip"`
	UserID     int64     `json:"user_id"`
	Endpoint   string    `json:"endpoint"`
	Method     string    `json:"method"`
	UserAgent  string    `json:"user_agent"`
	Timestamp  time.Time `json:"timestamp"`
	IsAuth     bool      `json:"is_auth"`
}

// RateLimitResult represents the result of rate limiting check
type RateLimitResult struct {
	Allowed      bool          `json:"allowed"`
	Remaining    int           `json:"remaining"`
	ResetTime    time.Time     `json:"reset_time"`
	RetryAfter   time.Duration `json:"retry_after"`
	Reason       string        `json:"reason"`
	Headers      map[string]string `json:"headers"`
}

// calculateDynamicLimits adjusts limits based on system load
func (rl *EnhancedRateLimiter) calculateDynamicLimits() *EnhancedRateLimitConfig {
	if !rl.config.EnableDynamicLimits {
		return rl.config
	}

	rl.systemLoadMutex.RLock()
	currentLoad := rl.systemLoad
	rl.systemLoadMutex.RUnlock()

	// If system load is high, reduce limits
	if currentLoad > rl.config.LoadThreshold {
		multiplier := rl.config.DynamicMultiplier

		// Create adjusted config
		adjustedConfig := *rl.config
		adjustedConfig.GlobalLimit = int(float64(rl.config.GlobalLimit) * multiplier)
		adjustedConfig.UserLimit = int(float64(rl.config.UserLimit) * multiplier)
		adjustedConfig.IPLimit = int(float64(rl.config.IPLimit) * multiplier)

		logrus.WithFields(logrus.Fields{
			"load":       currentLoad,
			"threshold":  rl.config.LoadThreshold,
			"multiplier": multiplier,
		}).Debug("Applied dynamic rate limits due to high load")

		return &adjustedConfig
	}

	return rl.config
}

// checkGlobalLimit checks global rate limit
func (rl *EnhancedRateLimiter) checkGlobalLimit(config *EnhancedRateLimitConfig) *RateLimitResult {
	key := "global"
	return rl.checkLimit(key, config.GlobalLimit, config.GlobalWindow, "Global rate limit exceeded")
}

// checkEndpointLimit checks endpoint-specific rate limit
func (rl *EnhancedRateLimiter) checkEndpointLimit(ctx *RateLimitContext, config *EnhancedRateLimitConfig) *RateLimitResult {
	endpointConfig, exists := config.EndpointLimits[ctx.Endpoint]
	if !exists {
		return &RateLimitResult{Allowed: true}
	}

	// Check if method is restricted
	if len(endpointConfig.Methods) > 0 {
		methodAllowed := false
		for _, method := range endpointConfig.Methods {
			if method == ctx.Method {
				methodAllowed = true
				break
			}
		}
		if !methodAllowed {
			return &RateLimitResult{Allowed: true}
		}
	}

	// Check if auth is required and user is authenticated
	if endpointConfig.RequireAuth && !ctx.IsAuth {
		return &RateLimitResult{Allowed: true}
	}

	key := fmt.Sprintf("endpoint:%s:%s", ctx.Endpoint, ctx.IP)
	if ctx.UserID > 0 {
		key = fmt.Sprintf("endpoint:%s:user:%d", ctx.Endpoint, ctx.UserID)
	}

	return rl.checkLimit(key, endpointConfig.Limit, endpointConfig.Window, fmt.Sprintf("Endpoint %s rate limit exceeded", ctx.Endpoint))
}

// checkIPLimit checks IP-based rate limit
func (rl *EnhancedRateLimiter) checkIPLimit(ctx *RateLimitContext, config *EnhancedRateLimitConfig) *RateLimitResult {
	key := fmt.Sprintf("ip:%s", ctx.IP)
	return rl.checkLimit(key, config.IPLimit, config.IPWindow, "IP rate limit exceeded")
}

// checkUserLimit checks user-based rate limit
func (rl *EnhancedRateLimiter) checkUserLimit(ctx *RateLimitContext, config *EnhancedRateLimitConfig) *RateLimitResult {
	key := fmt.Sprintf("user:%d", ctx.UserID)
	return rl.checkLimit(key, config.UserLimit, config.UserWindow, "User rate limit exceeded")
}

// checkSubnetLimit checks subnet-based rate limit
func (rl *EnhancedRateLimiter) checkSubnetLimit(ctx *RateLimitContext, config *EnhancedRateLimitConfig) *RateLimitResult {
	// Extract subnet from IP
	ip := net.ParseIP(ctx.IP)
	if ip == nil {
		return &RateLimitResult{Allowed: true}
	}

	var subnet *net.IPNet
	if ip.To4() != nil {
		// IPv4 - use /24 subnet
		_, subnet, _ = net.ParseCIDR(ip.String() + "/24")
	} else {
		// IPv6 - use /64 subnet
		_, subnet, _ = net.ParseCIDR(ip.String() + "/64")
	}

	if subnet == nil {
		return &RateLimitResult{Allowed: true}
	}

	key := fmt.Sprintf("subnet:%s", subnet.String())
	return rl.checkLimit(key, config.SubnetLimit, config.SubnetWindow, "Subnet rate limit exceeded")
}

// checkLimit is a generic function to check rate limits
func (rl *EnhancedRateLimiter) checkLimit(key string, limit int, window time.Duration, reason string) *RateLimitResult {
	now := time.Now()
	entry, exists := rl.entries[key]

	if !exists {
		// Create new entry
		entry = &RateLimitEntry{
			Key:         key,
			Count:       1,
			WindowStart: now,
			LastRequest: now,
		}
		rl.entries[key] = entry
		return &RateLimitResult{
			Allowed:   true,
			Remaining: limit - 1,
			ResetTime: now.Add(window),
		}
	}

	// Check if window has expired
	if now.Sub(entry.WindowStart) >= window {
		// Reset window
		entry.Count = 1
		entry.WindowStart = now
		entry.LastRequest = now
		return &RateLimitResult{
			Allowed:   true,
			Remaining: limit - 1,
			ResetTime: now.Add(window),
		}
	}

	// Check burst handling
	burstLimit := int(float64(limit) * rl.config.BurstMultiplier)
	timeSinceLastRequest := now.Sub(entry.LastRequest)

	// Allow burst if within burst window and under burst limit
	if timeSinceLastRequest <= rl.config.BurstWindow && entry.Count < burstLimit {
		entry.Count++
		entry.LastRequest = now
		return &RateLimitResult{
			Allowed:   true,
			Remaining: max(0, limit-entry.Count),
			ResetTime: entry.WindowStart.Add(window),
		}
	}

	// Check normal limit
	if entry.Count >= limit {
		resetTime := entry.WindowStart.Add(window)
		retryAfter := resetTime.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}

		return &RateLimitResult{
			Allowed:    false,
			Remaining:  0,
			ResetTime:  resetTime,
			RetryAfter: retryAfter,
			Reason:     reason,
		}
	}

	// Allow request
	entry.Count++
	entry.LastRequest = now
	return &RateLimitResult{
		Allowed:   true,
		Remaining: limit - entry.Count,
		ResetTime: entry.WindowStart.Add(window),
	}
}

// isBlacklisted checks if IP or user is blacklisted
func (rl *EnhancedRateLimiter) isBlacklisted(ctx *RateLimitContext) bool {
	// Check IP blacklist
	for _, blacklistedIP := range rl.config.IPBlacklist {
		if ctx.IP == blacklistedIP {
			return true
		}
	}

	// Check user blacklist
	if ctx.UserID > 0 {
		for _, blacklistedUser := range rl.config.UserBlacklist {
			if ctx.UserID == blacklistedUser {
				return true
			}
		}
	}

	return false
}

// isWhitelisted checks if IP or user is whitelisted
func (rl *EnhancedRateLimiter) isWhitelisted(ctx *RateLimitContext) bool {
	// Check IP whitelist
	for _, whitelistedIP := range rl.config.IPWhitelist {
		if ctx.IP == whitelistedIP {
			return true
		}
	}

	// Check user whitelist
	if ctx.UserID > 0 {
		for _, whitelistedUser := range rl.config.UserWhitelist {
			if ctx.UserID == whitelistedUser {
				return true
			}
		}
	}

	return false
}

// checkBanned checks if IP or user is banned
func (rl *EnhancedRateLimiter) checkBanned(ctx *RateLimitContext) *RateLimitResult {
	if !rl.config.EnableBanning {
		return nil
	}

	// Check IP ban
	key := fmt.Sprintf("ban:ip:%s", ctx.IP)
	if entry, exists := rl.entries[key]; exists && entry.BannedUntil != nil {
		if time.Now().Before(*entry.BannedUntil) {
			retryAfter := entry.BannedUntil.Sub(time.Now())
			return &RateLimitResult{
				Allowed:    false,
				RetryAfter: retryAfter,
				Reason:     fmt.Sprintf("IP banned until %s", entry.BannedUntil.Format(time.RFC3339)),
			}
		} else {
			// Ban expired, remove it
			delete(rl.entries, key)
		}
	}

	// Check user ban
	if ctx.UserID > 0 {
		key := fmt.Sprintf("ban:user:%d", ctx.UserID)
		if entry, exists := rl.entries[key]; exists && entry.BannedUntil != nil {
			if time.Now().Before(*entry.BannedUntil) {
				retryAfter := entry.BannedUntil.Sub(time.Now())
				return &RateLimitResult{
					Allowed:    false,
					RetryAfter: retryAfter,
					Reason:     fmt.Sprintf("User banned until %s", entry.BannedUntil.Format(time.RFC3339)),
				}
			} else {
				// Ban expired, remove it
				delete(rl.entries, key)
			}
		}
	}

	return nil
}

// recordViolation records a rate limit violation and handles banning
func (rl *EnhancedRateLimiter) recordViolation(ctx *RateLimitContext) {
	if !rl.config.EnableBanning {
		return
	}

	now := time.Now()

	// Record IP violation
	ipViolationKey := fmt.Sprintf("violations:ip:%s", ctx.IP)
	ipEntry, exists := rl.entries[ipViolationKey]
	if !exists {
		ipEntry = &RateLimitEntry{
			Key:         ipViolationKey,
			Violations:  1,
			WindowStart: now,
		}
		rl.entries[ipViolationKey] = ipEntry
	} else {
		ipEntry.Violations++
		ipEntry.LastRequest = now
	}

	// Check if IP should be banned
	if ipEntry.Violations >= rl.config.BanThreshold {
		rl.banIP(ctx.IP, ipEntry.Violations)
	}

	// Record user violation (if authenticated)
	if ctx.UserID > 0 {
		userViolationKey := fmt.Sprintf("violations:user:%d", ctx.UserID)
		userEntry, exists := rl.entries[userViolationKey]
		if !exists {
			userEntry = &RateLimitEntry{
				Key:         userViolationKey,
				Violations:  1,
				WindowStart: now,
			}
			rl.entries[userViolationKey] = userEntry
		} else {
			userEntry.Violations++
			userEntry.LastRequest = now
		}

		// Check if user should be banned
		if userEntry.Violations >= rl.config.BanThreshold {
			rl.banUser(ctx.UserID, userEntry.Violations)
		}
	}
}

// banIP bans an IP address
func (rl *EnhancedRateLimiter) banIP(ip string, violations int) {
	// Calculate ban duration based on violations
	banDuration := rl.config.BanDuration * time.Duration(violations)
	if banDuration > rl.config.MaxBanDuration {
		banDuration = rl.config.MaxBanDuration
	}

	banUntil := time.Now().Add(banDuration)

	banKey := fmt.Sprintf("ban:ip:%s", ip)
	rl.entries[banKey] = &RateLimitEntry{
		Key:         banKey,
		Violations:  violations,
		BannedUntil: &banUntil,
	}

	rl.globalStats.BannedIPs++

	logrus.WithFields(logrus.Fields{
		"ip":          ip,
		"violations":  violations,
		"ban_duration": banDuration,
		"ban_until":   banUntil,
	}).Warn("IP address banned due to rate limit violations")
}

// banUser bans a user
func (rl *EnhancedRateLimiter) banUser(userID int64, violations int) {
	// Calculate ban duration based on violations
	banDuration := rl.config.BanDuration * time.Duration(violations)
	if banDuration > rl.config.MaxBanDuration {
		banDuration = rl.config.MaxBanDuration
	}

	banUntil := time.Now().Add(banDuration)

	banKey := fmt.Sprintf("ban:user:%d", userID)
	rl.entries[banKey] = &RateLimitEntry{
		Key:         banKey,
		Violations:  violations,
		BannedUntil: &banUntil,
	}

	logrus.WithFields(logrus.Fields{
		"user_id":     userID,
		"violations":  violations,
		"ban_duration": banDuration,
		"ban_until":   banUntil,
	}).Warn("User banned due to rate limit violations")
}

// updateCounters updates counters for allowed requests
func (rl *EnhancedRateLimiter) updateCounters(ctx *RateLimitContext) {
	// This is handled in checkLimit functions
	// Additional metrics can be added here if needed
}

// calculateRemaining calculates remaining requests for the most restrictive limit
func (rl *EnhancedRateLimiter) calculateRemaining(ctx *RateLimitContext, config *EnhancedRateLimitConfig) int {
	// Return the minimum remaining from all applicable limits
	remaining := config.GlobalLimit

	if ipEntry, exists := rl.entries[fmt.Sprintf("ip:%s", ctx.IP)]; exists {
		ipRemaining := config.IPLimit - ipEntry.Count
		if ipRemaining < remaining {
			remaining = ipRemaining
		}
	}

	if ctx.UserID > 0 {
		if userEntry, exists := rl.entries[fmt.Sprintf("user:%d", ctx.UserID)]; exists {
			userRemaining := config.UserLimit - userEntry.Count
			if userRemaining < remaining {
				remaining = userRemaining
			}
		}
	}

	return max(0, remaining)
}

// calculateResetTime calculates when the limits will reset
func (rl *EnhancedRateLimiter) calculateResetTime(ctx *RateLimitContext) time.Time {
	// Return the earliest reset time from all applicable limits
	resetTime := time.Now().Add(rl.config.GlobalWindow)

	if ipEntry, exists := rl.entries[fmt.Sprintf("ip:%s", ctx.IP)]; exists {
		ipResetTime := ipEntry.WindowStart.Add(rl.config.IPWindow)
		if ipResetTime.Before(resetTime) {
			resetTime = ipResetTime
		}
	}

	if ctx.UserID > 0 {
		if userEntry, exists := rl.entries[fmt.Sprintf("user:%d", ctx.UserID)]; exists {
			userResetTime := userEntry.WindowStart.Add(rl.config.UserWindow)
			if userResetTime.Before(resetTime) {
				resetTime = userResetTime
			}
		}
	}

	return resetTime
}

// GetStats returns comprehensive rate limiter statistics
func (rl *EnhancedRateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	// Count active entries by type
	activeIPs := 0
	activeUsers := 0
	activeEndpoints := 0
	bannedIPs := 0
	bannedUsers := 0

	now := time.Now()
	for key, entry := range rl.entries {
		if strings.HasPrefix(key, "ip:") {
			activeIPs++
		} else if strings.HasPrefix(key, "user:") {
			activeUsers++
		} else if strings.HasPrefix(key, "endpoint:") {
			activeEndpoints++
		} else if strings.HasPrefix(key, "ban:ip:") && entry.BannedUntil != nil && now.Before(*entry.BannedUntil) {
			bannedIPs++
		} else if strings.HasPrefix(key, "ban:user:") && entry.BannedUntil != nil && now.Before(*entry.BannedUntil) {
			bannedUsers++
		}
	}

	rl.systemLoadMutex.RLock()
	currentLoad := rl.systemLoad
	rl.systemLoadMutex.RUnlock()

	return map[string]interface{}{
		"global_stats": map[string]interface{}{
			"total_requests":   rl.globalStats.TotalRequests,
			"blocked_requests": rl.globalStats.BlockedRequests,
			"block_rate":       float64(rl.globalStats.BlockedRequests) / float64(max(1, int(rl.globalStats.TotalRequests))),
		},
		"active_entries": map[string]interface{}{
			"total":     len(rl.entries),
			"ips":       activeIPs,
			"users":     activeUsers,
			"endpoints": activeEndpoints,
		},
		"bans": map[string]interface{}{
			"banned_ips":   bannedIPs,
			"banned_users": bannedUsers,
		},
		"system": map[string]interface{}{
			"current_load":      currentLoad,
			"load_threshold":    rl.config.LoadThreshold,
			"dynamic_limits":    rl.config.EnableDynamicLimits,
			"memory_entries":    len(rl.entries),
			"max_memory_entries": rl.config.MaxMemoryEntries,
		},
		"config": map[string]interface{}{
			"global_limit":  rl.config.GlobalLimit,
			"user_limit":    rl.config.UserLimit,
			"ip_limit":      rl.config.IPLimit,
			"subnet_limit":  rl.config.SubnetLimit,
			"ban_enabled":   rl.config.EnableBanning,
			"ban_threshold": rl.config.BanThreshold,
		},
	}
}

// startCleanup starts the cleanup goroutine
func (rl *EnhancedRateLimiter) startCleanup() {
	rl.cleanupTicker = time.NewTicker(rl.config.CleanupInterval)

	go func() {
		for range rl.cleanupTicker.C {
			rl.cleanup()
		}
	}()
}

// cleanup removes expired entries
func (rl *EnhancedRateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range rl.entries {
		shouldRemove := false

		// Remove expired ban entries
		if entry.BannedUntil != nil && now.After(*entry.BannedUntil) {
			shouldRemove = true
		}

		// Remove old window entries
		if entry.BannedUntil == nil {
			var window time.Duration
			if strings.HasPrefix(key, "ip:") {
				window = rl.config.IPWindow
			} else if strings.HasPrefix(key, "user:") {
				window = rl.config.UserWindow
			} else if strings.HasPrefix(key, "endpoint:") {
				window = time.Hour // Default window for endpoints
			} else {
				window = rl.config.GlobalWindow
			}

			if now.Sub(entry.WindowStart) > window*2 {
				shouldRemove = true
			}
		}

		if shouldRemove {
			delete(rl.entries, key)
			removed++
		}
	}

	// If memory usage is still high, remove oldest entries
	if len(rl.entries) > rl.config.MaxMemoryEntries {
		// Sort by last request time and remove oldest
		type entryWithTime struct {
			key  string
			time time.Time
		}

		var entries []entryWithTime
		for key, entry := range rl.entries {
			entries = append(entries, entryWithTime{key: key, time: entry.LastRequest})
		}

		// Remove oldest entries beyond max memory limit
		toRemove := len(rl.entries) - rl.config.MaxMemoryEntries
		for i := 0; i < toRemove && i < len(entries); i++ {
			delete(rl.entries, entries[i].key)
			removed++
		}
	}

	rl.globalStats.LastCleanup = now
	rl.globalStats.ActiveEntries = len(rl.entries)

	if removed > 0 {
		logrus.WithField("removed_entries", removed).Debug("Rate limiter cleanup completed")
	}
}

// startLoadMonitoring starts system load monitoring
func (rl *EnhancedRateLimiter) startLoadMonitoring() {
	if !rl.config.EnableDynamicLimits {
		return
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			// Simple load calculation based on request rate
			// In a real implementation, this could use system metrics
			rl.systemLoadMutex.Lock()

			currentRequests := float64(rl.globalStats.TotalRequests)
			if currentRequests > 0 {
				blockRate := float64(rl.globalStats.BlockedRequests) / currentRequests
				rl.systemLoad = blockRate
			} else {
				rl.systemLoad = 0.0
			}

			rl.systemLoadMutex.Unlock()
		}
	}()
}

// Stop stops the rate limiter and cleanup goroutines
func (rl *EnhancedRateLimiter) Stop() {
	if rl.cleanupTicker != nil {
		rl.cleanupTicker.Stop()
	}
}

// AddToBlacklist adds IP or user to blacklist
func (rl *EnhancedRateLimiter) AddToBlacklist(ip string, userID int64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if ip != "" {
		rl.config.IPBlacklist = append(rl.config.IPBlacklist, ip)
		logrus.WithField("ip", ip).Info("IP added to blacklist")
	}

	if userID > 0 {
		rl.config.UserBlacklist = append(rl.config.UserBlacklist, userID)
		logrus.WithField("user_id", userID).Info("User added to blacklist")
	}
}

// RemoveFromBlacklist removes IP or user from blacklist
func (rl *EnhancedRateLimiter) RemoveFromBlacklist(ip string, userID int64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if ip != "" {
		for i, blacklistedIP := range rl.config.IPBlacklist {
			if blacklistedIP == ip {
				rl.config.IPBlacklist = append(rl.config.IPBlacklist[:i], rl.config.IPBlacklist[i+1:]...)
				logrus.WithField("ip", ip).Info("IP removed from blacklist")
				break
			}
		}
	}

	if userID > 0 {
		for i, blacklistedUser := range rl.config.UserBlacklist {
			if blacklistedUser == userID {
				rl.config.UserBlacklist = append(rl.config.UserBlacklist[:i], rl.config.UserBlacklist[i+1:]...)
				logrus.WithField("user_id", userID).Info("User removed from blacklist")
				break
			}
		}
	}
}

// Helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// generateKey generates a hash key for rate limiting
func generateKey(components ...string) string {
	h := sha256.New()
	for _, component := range components {
		h.Write([]byte(component + ":"))
	}
	return hex.EncodeToString(h.Sum(nil))[:16] // Use first 16 chars for efficiency
}