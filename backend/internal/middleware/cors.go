package middleware

import (
	"net/http"
	"strings"
	"time"

	"docker-auto/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
}

// CORSMiddleware creates a CORS middleware with configuration
func CORSMiddleware(config *config.Config) gin.HandlerFunc {
	corsConfig := buildCORSConfig(config)

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Set CORS headers
		setCORSHeaders(c, corsConfig, origin)

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			logrus.WithFields(logrus.Fields{
				"origin": origin,
				"path":   c.Request.URL.Path,
			}).Debug("Handling CORS preflight request")

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SimpleCORSMiddleware creates a simple CORS middleware for development
func SimpleCORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RestrictiveCORSMiddleware creates a restrictive CORS middleware for production
func RestrictiveCORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		if origin != "" && isOriginAllowed(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		c.Header("Access-Control-Max-Age", "3600")

		if c.Request.Method == "OPTIONS" {
			logrus.WithFields(logrus.Fields{
				"origin":  origin,
				"allowed": isOriginAllowed(origin, allowedOrigins),
			}).Debug("Handling restrictive CORS preflight request")

			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// buildCORSConfig builds CORS configuration from app config
func buildCORSConfig(config *config.Config) *CORSConfig {
	corsConfig := &CORSConfig{
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Parse allowed origins
	if config.Security.CORSAllowedOrigins != "" {
		corsConfig.AllowOrigins = parseCSV(config.Security.CORSAllowedOrigins)
	} else {
		// Default to localhost for development
		if config.IsDevelopment() {
			corsConfig.AllowOrigins = []string{"*"}
		} else {
			corsConfig.AllowOrigins = []string{"https://localhost", "http://localhost"}
		}
	}

	// Parse allowed methods
	if config.Security.CORSAllowedMethods != "" {
		corsConfig.AllowMethods = parseCSV(config.Security.CORSAllowedMethods)
	} else {
		corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}

	// Parse allowed headers
	if config.Security.CORSAllowedHeaders != "" {
		corsConfig.AllowHeaders = parseCSV(config.Security.CORSAllowedHeaders)
	} else {
		corsConfig.AllowHeaders = []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
			"Origin",
			"Cache-Control",
			"X-File-Name",
		}
	}

	// Standard expose headers
	corsConfig.ExposeHeaders = []string{
		"Content-Length",
		"Content-Type",
		"X-Total-Count",
		"X-Page-Count",
	}

	logrus.WithFields(logrus.Fields{
		"allowed_origins": corsConfig.AllowOrigins,
		"allowed_methods": corsConfig.AllowMethods,
		"max_age":         corsConfig.MaxAge,
		"development":     config.IsDevelopment(),
	}).Info("CORS configuration initialized")

	return corsConfig
}

// setCORSHeaders sets the appropriate CORS headers
func setCORSHeaders(c *gin.Context, config *CORSConfig, origin string) {
	// Set allowed origin
	if len(config.AllowOrigins) == 1 && config.AllowOrigins[0] == "*" {
		c.Header("Access-Control-Allow-Origin", "*")
	} else if origin != "" && isOriginAllowed(origin, config.AllowOrigins) {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
	}

	// Set allowed methods
	if len(config.AllowMethods) > 0 {
		c.Header("Access-Control-Allow-Methods", strings.Join(config.AllowMethods, ", "))
	}

	// Set allowed headers
	if len(config.AllowHeaders) > 0 {
		c.Header("Access-Control-Allow-Headers", strings.Join(config.AllowHeaders, ", "))
	}

	// Set exposed headers
	if len(config.ExposeHeaders) > 0 {
		c.Header("Access-Control-Expose-Headers", strings.Join(config.ExposeHeaders, ", "))
	}

	// Set credentials
	if config.AllowCredentials {
		c.Header("Access-Control-Allow-Credentials", "true")
	}

	// Set max age
	if config.MaxAge > 0 {
		c.Header("Access-Control-Max-Age", string(rune(int(config.MaxAge.Seconds()))))
	}
}

// isOriginAllowed checks if the origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if len(allowedOrigins) == 0 {
		return false
	}

	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if strings.EqualFold(origin, allowed) {
			return true
		}
		// Support wildcard subdomains like *.example.com
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, "."+domain) || strings.EqualFold(origin, domain) {
				return true
			}
		}
	}

	return false
}

// parseCSV parses a comma-separated string into a slice
func parseCSV(input string) []string {
	if input == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// CORSValidationMiddleware validates CORS requests and logs violations
func CORSValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		method := c.Request.Method
		path := c.Request.URL.Path

		// Log CORS request details for debugging
		if origin != "" {
			logrus.WithFields(logrus.Fields{
				"origin": origin,
				"method": method,
				"path":   path,
				"type":   "cors_request",
			}).Debug("CORS request received")
		}

		// Check for potential CORS issues
		if method != "OPTIONS" && origin != "" {
			referer := c.Request.Header.Get("Referer")
			if referer != "" && !strings.HasPrefix(referer, origin) {
				logrus.WithFields(logrus.Fields{
					"origin":  origin,
					"referer": referer,
					"path":    path,
				}).Warn("Potential CORS issue: origin and referer mismatch")
			}
		}

		c.Next()
	}
}