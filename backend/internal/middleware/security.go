package middleware

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityConfig represents security middleware configuration
type SecurityConfig struct {
	// HTTPS enforcement
	ForceHTTPS           bool     `json:"force_https"`
	HTTPSRedirect        bool     `json:"https_redirect"`
	TrustProxyHeaders    bool     `json:"trust_proxy_headers"`
	AllowInsecureLocal   bool     `json:"allow_insecure_local"`

	// Security headers
	EnableHSTS           bool     `json:"enable_hsts"`
	HSTSMaxAge          int      `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool    `json:"hsts_include_subdomains"`
	EnableCSP            bool     `json:"enable_csp"`
	CSPDirectives        string   `json:"csp_directives"`
	EnableXFrameOptions  bool     `json:"enable_x_frame_options"`
	XFrameOptions        string   `json:"x_frame_options"`
	EnableXSSProtection  bool     `json:"enable_xss_protection"`
	EnableContentTypeOptions bool `json:"enable_content_type_options"`
	EnableReferrerPolicy bool     `json:"enable_referrer_policy"`
	ReferrerPolicy       string   `json:"referrer_policy"`

	// CORS settings
	EnableCORS           bool     `json:"enable_cors"`
	AllowedOrigins       []string `json:"allowed_origins"`
	AllowedMethods       []string `json:"allowed_methods"`
	AllowedHeaders       []string `json:"allowed_headers"`
	AllowCredentials     bool     `json:"allow_credentials"`
	MaxAge               int      `json:"max_age"`

	// Request validation
	MaxRequestSize       int64    `json:"max_request_size"`
	BlockSuspiciousUA    bool     `json:"block_suspicious_ua"`
	SuspiciousUAPatterns []string `json:"suspicious_ua_patterns"`

	// API Key validation
	RequireAPIKey        bool     `json:"require_api_key"`
	APIKeyHeader         string   `json:"api_key_header"`
	ValidAPIKeys         []string `json:"valid_api_keys"`

	// Rate limiting integration
	EnableGlobalRateLimit bool    `json:"enable_global_rate_limit"`
	GlobalRateLimit      int     `json:"global_rate_limit"`
	GlobalRateWindow     time.Duration `json:"global_rate_window"`
}

// DefaultSecurityConfig returns a secure default configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		// HTTPS enforcement
		ForceHTTPS:         true,
		HTTPSRedirect:      true,
		TrustProxyHeaders:  true,
		AllowInsecureLocal: false,

		// Security headers
		EnableHSTS:           true,
		HSTSMaxAge:          31536000, // 1 year
		HSTSIncludeSubdomains: true,
		EnableCSP:            true,
		CSPDirectives:        "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'",
		EnableXFrameOptions:  true,
		XFrameOptions:        "DENY",
		EnableXSSProtection:  true,
		EnableContentTypeOptions: true,
		EnableReferrerPolicy: true,
		ReferrerPolicy:       "strict-origin-when-cross-origin",

		// CORS settings
		EnableCORS:        true,
		AllowedOrigins:   []string{"http://localhost:3000", "https://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours

		// Request validation
		MaxRequestSize:       10 * 1024 * 1024, // 10MB
		BlockSuspiciousUA:    true,
		SuspiciousUAPatterns: []string{
			"sqlmap", "nmap", "nikto", "w3af", "paros", "burp", "hydra",
			"python-requests", "curl", "wget",
		},

		// API Key validation
		RequireAPIKey: false,
		APIKeyHeader:  "X-API-Key",
		ValidAPIKeys:  []string{},

		// Rate limiting
		EnableGlobalRateLimit: true,
		GlobalRateLimit:      1000,
		GlobalRateWindow:     time.Hour,
	}
}

// SecurityMiddleware creates a comprehensive security middleware
func SecurityMiddleware(config *SecurityConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return func(c *gin.Context) {
		// Apply security headers first
		applySecurityHeaders(c, config)

		// Enforce HTTPS if required
		if config.ForceHTTPS {
			if err := enforceHTTPS(c, config); err != nil {
				logrus.WithError(err).Warn("HTTPS enforcement failed")
				c.JSON(http.StatusUpgradeRequired, utils.ErrorResponse(http.StatusUpgradeRequired, "HTTPS required"))
				c.Abort()
				return
			}
		}

		// Validate request size
		if config.MaxRequestSize > 0 && c.Request.ContentLength > config.MaxRequestSize {
			logrus.WithFields(logrus.Fields{
				"content_length": c.Request.ContentLength,
				"max_size":       config.MaxRequestSize,
				"client_ip":      c.ClientIP(),
			}).Warn("Request size exceeds security limit")
			c.JSON(http.StatusRequestEntityTooLarge, utils.ErrorResponse(http.StatusRequestEntityTooLarge, "Request too large"))
			c.Abort()
			return
		}

		// Block suspicious user agents
		if config.BlockSuspiciousUA {
			if isSuspiciousUserAgent(c.GetHeader("User-Agent"), config.SuspiciousUAPatterns) {
				logrus.WithFields(logrus.Fields{
					"user_agent": c.GetHeader("User-Agent"),
					"client_ip":  c.ClientIP(),
					"path":       c.Request.URL.Path,
				}).Warn("Suspicious user agent blocked")
				c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "Access denied"))
				c.Abort()
				return
			}
		}

		// Validate API key if required
		if config.RequireAPIKey {
			if err := validateAPIKey(c, config); err != nil {
				logrus.WithError(err).Warn("API key validation failed")
				c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Invalid or missing API key"))
				c.Abort()
				return
			}
		}

		// Handle CORS preflight requests
		if config.EnableCORS && c.Request.Method == "OPTIONS" {
			handleCORSPreflight(c, config)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// Validate CORS if enabled
		if config.EnableCORS && c.Request.Method != "OPTIONS" {
			if err := validateCORS(c, config); err != nil {
				logrus.WithError(err).Warn("CORS validation failed")
				c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "CORS policy violation"))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// applySecurityHeaders applies various security headers
func applySecurityHeaders(c *gin.Context, config *SecurityConfig) {
	// X-Content-Type-Options
	if config.EnableContentTypeOptions {
		c.Header("X-Content-Type-Options", "nosniff")
	}

	// X-Frame-Options
	if config.EnableXFrameOptions {
		c.Header("X-Frame-Options", config.XFrameOptions)
	}

	// X-XSS-Protection
	if config.EnableXSSProtection {
		c.Header("X-XSS-Protection", "1; mode=block")
	}

	// Content Security Policy
	if config.EnableCSP && config.CSPDirectives != "" {
		c.Header("Content-Security-Policy", config.CSPDirectives)
	}

	// Referrer Policy
	if config.EnableReferrerPolicy && config.ReferrerPolicy != "" {
		c.Header("Referrer-Policy", config.ReferrerPolicy)
	}

	// HSTS (only for HTTPS)
	if config.EnableHSTS && isHTTPS(c, config) {
		hstsValue := fmt.Sprintf("max-age=%d", config.HSTSMaxAge)
		if config.HSTSIncludeSubdomains {
			hstsValue += "; includeSubDomains"
		}
		hstsValue += "; preload"
		c.Header("Strict-Transport-Security", hstsValue)
	}

	// Additional security headers
	c.Header("X-Permitted-Cross-Domain-Policies", "none")
	c.Header("X-Download-Options", "noopen")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
}

// enforceHTTPS ensures the request is made over HTTPS
func enforceHTTPS(c *gin.Context, config *SecurityConfig) error {
	if isHTTPS(c, config) {
		return nil
	}

	// Allow insecure local connections if configured
	if config.AllowInsecureLocal && isLocalRequest(c) {
		return nil
	}

	// Redirect to HTTPS if enabled
	if config.HTTPSRedirect {
		httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
		c.Redirect(http.StatusMovedPermanently, httpsURL)
		return fmt.Errorf("redirected to HTTPS")
	}

	return fmt.Errorf("HTTPS required")
}

// isHTTPS checks if the request is made over HTTPS
func isHTTPS(c *gin.Context, config *SecurityConfig) bool {
	// Check TLS
	if c.Request.TLS != nil {
		return true
	}

	// Check proxy headers if trusted
	if config.TrustProxyHeaders {
		if proto := c.GetHeader("X-Forwarded-Proto"); proto == "https" {
			return true
		}
		if c.GetHeader("X-Forwarded-SSL") == "on" {
			return true
		}
	}

	return false
}


// isSuspiciousUserAgent checks if the user agent matches suspicious patterns
func isSuspiciousUserAgent(userAgent string, patterns []string) bool {
	if userAgent == "" {
		return true // Empty user agent is suspicious
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range patterns {
		if strings.Contains(userAgentLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// validateAPIKey validates the API key
func validateAPIKey(c *gin.Context, config *SecurityConfig) error {
	apiKey := c.GetHeader(config.APIKeyHeader)
	if apiKey == "" {
		return fmt.Errorf("API key header missing")
	}

	// Use constant-time comparison to prevent timing attacks
	for _, validKey := range config.ValidAPIKeys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validKey)) == 1 {
			return nil
		}
	}

	return fmt.Errorf("invalid API key")
}

// handleCORSPreflight handles CORS preflight requests
func handleCORSPreflight(c *gin.Context, config *SecurityConfig) {
	origin := c.GetHeader("Origin")

	// Check if origin is allowed
	if !isAllowedOrigin(origin, config.AllowedOrigins) {
		return
	}

	// Set CORS headers for preflight
	c.Header("Access-Control-Allow-Origin", origin)
	c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
	c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))

	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	if config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", config.MaxAge))
	}
}

// validateCORS validates CORS for actual requests
func validateCORS(c *gin.Context, config *SecurityConfig) error {
	origin := c.GetHeader("Origin")

	// No origin header means same-origin request
	if origin == "" {
		return nil
	}

	// Check if origin is allowed
	if !isAllowedOrigin(origin, config.AllowedOrigins) {
		return fmt.Errorf("origin not allowed: %s", origin)
	}

	// Set CORS headers for actual request
	c.Header("Access-Control-Allow-Origin", origin)

	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	return nil
}

// isAllowedOrigin checks if an origin is allowed
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard subdomains
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, "."+domain) {
				return true
			}
		}
	}
	return false
}

// CSRFProtectionMiddleware provides CSRF protection
func CSRFProtectionMiddleware(secretKey string, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for safe methods
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Get CSRF token from header or form
		token := c.GetHeader("X-CSRF-Token")
		if token == "" {
			token = c.PostForm("_csrf_token")
		}

		// Get expected token from cookie
		expectedToken, err := c.Cookie(cookieName)
		if err != nil {
			logrus.WithError(err).Warn("CSRF cookie missing")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "CSRF token missing"))
			c.Abort()
			return
		}

		// Validate token
		if !isValidCSRFToken(token, expectedToken, secretKey) {
			logrus.WithFields(logrus.Fields{
				"client_ip": c.ClientIP(),
				"path":      c.Request.URL.Path,
				"method":    c.Request.Method,
			}).Warn("CSRF token validation failed")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "CSRF token invalid"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// isValidCSRFToken validates a CSRF token
func isValidCSRFToken(token, expectedToken, secretKey string) bool {
	if token == "" || expectedToken == "" {
		return false
	}

	// Use constant-time comparison
	return subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1
}

// generateCSRFToken generates a CSRF token
func generateCSRFToken() (string, error) {
	return utils.GenerateSecureRandomString(32)
}

// SetCSRFTokenMiddleware sets CSRF token in cookie
func SetCSRFTokenMiddleware(cookieName string, secure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if token already exists
		_, err := c.Cookie(cookieName)
		if err != nil {
			// Generate new token
			token, err := generateCSRFToken()
			if err != nil {
				logrus.WithError(err).Error("Failed to generate CSRF token")
				c.Next()
				return
			}

			// Set cookie
			c.SetCookie(
				cookieName,
				token,
				3600, // 1 hour
				"/",
				"",
				secure,
				true, // HttpOnly
			)

			// Set token in header for JavaScript access
			c.Header("X-CSRF-Token", token)
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			var err error
			requestID, err = utils.GenerateSecureRandomString(16)
			if err != nil {
				logrus.WithError(err).Error("Failed to generate request ID")
				requestID = fmt.Sprintf("req-%d", time.Now().UnixNano())
			}
		}

		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)

		// Add to logger context
		logrus.WithField("request_id", requestID).Debug("Request processed")

		c.Next()
	}
}

// IPWhitelistMiddleware allows only whitelisted IPs
func IPWhitelistMiddleware(allowedIPs []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)

		allowed := false
		for _, allowedIP := range allowedIPs {
			if clientIP == allowedIP {
				allowed = true
				break
			}
			// Support CIDR notation if needed (simple check for now)
			if strings.Contains(allowedIP, "/") {
				// For full CIDR support, would need additional parsing
				continue
			}
		}

		if !allowed {
			logrus.WithFields(logrus.Fields{
				"client_ip":   clientIP,
				"path":        c.Request.URL.Path,
				"user_agent":  c.GetHeader("User-Agent"),
			}).Warn("IP not in whitelist")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "Access denied"))
			c.Abort()
			return
		}

		c.Next()
	}
}


// SecurityHeadersOnlyMiddleware applies only security headers without other checks
func SecurityHeadersOnlyMiddleware(config *SecurityConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return func(c *gin.Context) {
		applySecurityHeaders(c, config)
		c.Next()
	}
}

// StrictTransportSecurityMiddleware specifically handles HSTS
func StrictTransportSecurityMiddleware(maxAge int, includeSubdomains bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			hstsValue := fmt.Sprintf("max-age=%d", maxAge)
			if includeSubdomains {
				hstsValue += "; includeSubDomains"
			}
			hstsValue += "; preload"
			c.Header("Strict-Transport-Security", hstsValue)
		}
		c.Next()
	}
}

// ContentSecurityPolicyMiddleware specifically handles CSP
func ContentSecurityPolicyMiddleware(directives string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if directives != "" {
			c.Header("Content-Security-Policy", directives)
		}
		c.Next()
	}
}

// NoSniffMiddleware prevents MIME type sniffing
func NoSniffMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}

// ClickjackingProtectionMiddleware prevents clickjacking attacks
func ClickjackingProtectionMiddleware(policy string) gin.HandlerFunc {
	if policy == "" {
		policy = "DENY"
	}

	return func(c *gin.Context) {
		c.Header("X-Frame-Options", policy)
		c.Next()
	}
}

// URLValidationMiddleware validates URL structure and prevents malformed requests
func URLValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate URL structure
		if _, err := url.Parse(c.Request.RequestURI); err != nil {
			logrus.WithError(err).WithField("uri", c.Request.RequestURI).Warn("Invalid URL structure")
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid URL"))
			c.Abort()
			return
		}

		// Check for suspicious patterns in URL
		uri := c.Request.RequestURI
		suspiciousPatterns := []string{
			"../", "..\\", "%2e%2e", "%2f", "%5c",
			"<script", "</script>", "javascript:",
			"<iframe", "<object", "<embed",
		}

		uriLower := strings.ToLower(uri)
		for _, pattern := range suspiciousPatterns {
			if strings.Contains(uriLower, pattern) {
				logrus.WithFields(logrus.Fields{
					"uri":     uri,
					"pattern": pattern,
					"client_ip": c.ClientIP(),
				}).Warn("Suspicious URL pattern detected")
				c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid URL pattern"))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}