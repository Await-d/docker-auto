package logging

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestResponseWriter captures response data for logging
type RequestResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body
func (w *RequestResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

// LoggingMiddleware creates a Gin middleware for request/response logging
func LoggingMiddleware(logger *Logger, config LoggingMiddlewareConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		// Skip certain paths
		for _, path := range config.SkipPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		start := time.Now()

		// Generate request ID if not present
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Header("X-Request-ID", requestID)
		}

		// Create request-scoped logger
		reqLogger := logger.WithRequestID(requestID)

		// Add user ID if available
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				reqLogger = reqLogger.WithUserID(uid)
			}
		}

		// Store logger in context
		c.Set("logger", reqLogger)

		// Capture request body if configured
		var requestBody []byte
		if config.LogRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			// Limit body size for logging
			if len(requestBody) > config.MaxBodySize {
				requestBody = requestBody[:config.MaxBodySize]
			}
		}

		// Capture response body if configured
		var responseWriter *RequestResponseWriter
		if config.LogResponseBody {
			responseWriter = &RequestResponseWriter{
				ResponseWriter: c.Writer,
				body:          bytes.NewBuffer([]byte{}),
			}
			c.Writer = responseWriter
		}

		// Log request
		reqFields := map[string]interface{}{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"remote_addr": c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
			"referer":     c.Request.Referer(),
		}

		// Add headers (excluding sensitive ones)
		headers := make(map[string]string)
		for name, values := range c.Request.Header {
			if !contains(config.SensitiveHeaders, strings.ToLower(name)) {
				headers[name] = strings.Join(values, ", ")
			}
		}
		reqFields["headers"] = headers

		// Add request body
		if config.LogRequestBody && len(requestBody) > 0 {
			reqFields["request_body"] = string(requestBody)
		}

		reqLogger.Info("HTTP Request", reqFields)

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Prepare response fields
		respFields := map[string]interface{}{
			"status":   c.Writer.Status(),
			"size":     c.Writer.Size(),
			"duration": duration.String(),
		}

		// Add response body
		if config.LogResponseBody && responseWriter != nil {
			responseBody := responseWriter.body.String()
			if len(responseBody) > config.MaxBodySize {
				responseBody = responseBody[:config.MaxBodySize]
			}
			respFields["response_body"] = responseBody
		}

		// Add errors if any
		if len(c.Errors) > 0 {
			errors := make([]string, len(c.Errors))
			for i, err := range c.Errors {
				errors[i] = err.Error()
			}
			respFields["errors"] = errors
		}

		// Log response
		logLevel := INFO
		if c.Writer.Status() >= 400 && c.Writer.Status() < 500 {
			logLevel = WARN
		} else if c.Writer.Status() >= 500 {
			logLevel = ERROR
		}

		switch logLevel {
		case WARN:
			reqLogger.Warn("HTTP Response", respFields)
		case ERROR:
			reqLogger.Error("HTTP Response", nil, respFields)
		default:
			reqLogger.Info("HTTP Response", respFields)
		}

		// Log performance metrics
		reqLogger.LogPerformance("http_request", duration, map[string]interface{}{
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"status":        c.Writer.Status(),
			"response_size": c.Writer.Size(),
		})
	}
}

// ErrorLoggingMiddleware creates a middleware for error logging
func ErrorLoggingMiddleware(logger *Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		reqLogger := getLoggerFromContext(c, logger)

		reqLogger.Error("Panic recovered", nil, map[string]interface{}{
			"panic":  recovered,
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		})

		// Log security event for potential attacks
		reqLogger.LogSecurity("panic_recovery", "recovered", map[string]interface{}{
			"panic":      recovered,
			"ip_address": c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":      "Internal server error",
			"request_id": c.GetHeader("X-Request-ID"),
		})
	})
}

// SecurityLoggingMiddleware logs security-related events
func SecurityLoggingMiddleware(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLogger := getLoggerFromContext(c, logger)

		// Log authentication attempts
		if strings.Contains(c.Request.URL.Path, "/auth") || strings.Contains(c.Request.URL.Path, "/login") {
			reqLogger.LogSecurity("authentication_attempt", "started", map[string]interface{}{
				"path":       c.Request.URL.Path,
				"ip_address": c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			})
		}

		// Log admin access attempts
		if strings.Contains(c.Request.URL.Path, "/admin") {
			reqLogger.LogSecurity("admin_access_attempt", "started", map[string]interface{}{
				"path":       c.Request.URL.Path,
				"ip_address": c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			})
		}

		c.Next()

		// Log failed authentication
		if c.Writer.Status() == http.StatusUnauthorized {
			reqLogger.LogSecurity("authentication_failed", "failed", map[string]interface{}{
				"path":       c.Request.URL.Path,
				"ip_address": c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
				"status":     c.Writer.Status(),
			})
		}

		// Log successful authentication
		if strings.Contains(c.Request.URL.Path, "/auth") && c.Writer.Status() == http.StatusOK {
			reqLogger.LogSecurity("authentication_success", "success", map[string]interface{}{
				"path":       c.Request.URL.Path,
				"ip_address": c.ClientIP(),
				"user_agent": c.Request.UserAgent(),
			})
		}
	}
}

// AuditLoggingMiddleware logs audit events for data modifications
func AuditLoggingMiddleware(logger *Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqLogger := getLoggerFromContext(c, logger)

		// Only log modification operations
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" || c.Request.Method == "PATCH" {
			// Log start of modification
			reqLogger.LogAudit(c.Request.Method, c.Request.URL.Path, map[string]interface{}{
				"started_at": time.Now().UTC(),
				"ip_address": c.ClientIP(),
			})
		}

		c.Next()

		// Log completion of modification
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" || c.Request.Method == "PATCH" {
			outcome := "success"
			if c.Writer.Status() >= 400 {
				outcome = "failed"
			}

			reqLogger.LogAudit(c.Request.Method+"_completed", c.Request.URL.Path, map[string]interface{}{
				"outcome":      outcome,
				"status":       c.Writer.Status(),
				"completed_at": time.Now().UTC(),
				"ip_address":   c.ClientIP(),
			})
		}
	}
}

// getLoggerFromContext retrieves logger from Gin context or returns default
func getLoggerFromContext(c *gin.Context, defaultLogger *Logger) *Logger {
	if logger, exists := c.Get("logger"); exists {
		if reqLogger, ok := logger.(*Logger); ok {
			return reqLogger
		}
	}
	return defaultLogger
}

// contains checks if a slice contains a string (case-insensitive)
func contains(slice []string, item string) bool {
	item = strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == item {
			return true
		}
	}
	return false
}