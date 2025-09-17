package middleware

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogger represents request logging configuration
type RequestLogger struct {
	logger        *logrus.Logger
	skipPaths     map[string]bool
	logBody       bool
	logResponse   bool
	maxBodySize   int64
	sensitiveKeys []string
}

// LoggerConfig represents logger middleware configuration
type LoggerConfig struct {
	SkipPaths     []string
	LogBody       bool
	LogResponse   bool
	MaxBodySize   int64
	SensitiveKeys []string
}

// bodyLogWriter is a custom response writer to capture response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// LoggerMiddleware creates a request logging middleware
func LoggerMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	config := &LoggerConfig{
		SkipPaths: []string{
			"/health",
			"/metrics",
			"/favicon.ico",
		},
		LogBody:       false, // Disabled by default for security
		LogResponse:   false, // Disabled by default for performance
		MaxBodySize:   1024,  // 1KB limit for request body logging
		SensitiveKeys: []string{"password", "token", "secret", "key"},
	}

	return LoggerMiddlewareWithConfig(logger, config)
}

// LoggerMiddlewareWithConfig creates a request logging middleware with custom configuration
func LoggerMiddlewareWithConfig(logger *logrus.Logger, config *LoggerConfig) gin.HandlerFunc {
	requestLogger := &RequestLogger{
		logger:        logger,
		skipPaths:     make(map[string]bool),
		logBody:       config.LogBody,
		logResponse:   config.LogResponse,
		maxBodySize:   config.MaxBodySize,
		sensitiveKeys: config.SensitiveKeys,
	}

	// Build skip paths map for O(1) lookup
	for _, path := range config.SkipPaths {
		requestLogger.skipPaths[path] = true
	}

	return requestLogger.Handler()
}

// Handler returns the gin handler function
func (rl *RequestLogger) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain paths
		if shouldSkipLogging(c.Request.URL.Path, rl.skipPaths) {
			c.Next()
			return
		}

		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Extract user info if available
		var userID interface{}
		var username string
		if user := GetUserFromContext(c); user != nil {
			userID = user.UserID
			username = user.Username
		}

		// Capture request body if enabled
		var requestBody string
		if rl.logBody && c.Request.Body != nil {
			requestBody = rl.captureRequestBody(c)
		}

		// Capture response body if enabled
		var blw *bodyLogWriter
		if rl.logResponse {
			blw = &bodyLogWriter{
				ResponseWriter: c.Writer,
				body:          bytes.NewBufferString(""),
			}
			c.Writer = blw
		}

		// Process request
		c.Next()

		// Calculate request duration
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Build log fields
		fields := logrus.Fields{
			"method":      method,
			"path":        path,
			"status_code": statusCode,
			"latency":     latency,
			"client_ip":   clientIP,
			"user_agent":  userAgent,
			"size":        c.Writer.Size(),
		}

		// Add user info if available
		if userID != nil {
			fields["user_id"] = userID
		}
		if username != "" {
			fields["username"] = username
		}

		// Add request body if captured
		if requestBody != "" {
			fields["request_body"] = rl.sanitizeBody(requestBody)
		}

		// Add response body if captured
		if rl.logResponse && blw != nil {
			responseBody := blw.body.String()
			if responseBody != "" {
				fields["response_body"] = rl.sanitizeBody(responseBody)
			}
		}

		// Add query parameters
		if len(c.Request.URL.RawQuery) > 0 {
			fields["query"] = c.Request.URL.RawQuery
		}

		// Add error information if present
		if len(c.Errors) > 0 {
			fields["errors"] = c.Errors.String()
		}

		// Determine log level based on status code
		logLevel := rl.getLogLevel(statusCode)

		// Log the request
		rl.logger.WithFields(fields).Log(logLevel, "HTTP Request")
	}
}

// captureRequestBody captures and returns the request body
func (rl *RequestLogger) captureRequestBody(c *gin.Context) string {
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return ""
	}

	// Limit body size to prevent memory issues
	bodyBytes, err := io.ReadAll(io.LimitReader(c.Request.Body, rl.maxBodySize))
	if err != nil {
		rl.logger.WithError(err).Warn("Failed to read request body")
		return ""
	}

	// Restore the body for the next middleware
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return string(bodyBytes)
}

// sanitizeBody removes sensitive information from request/response body
func (rl *RequestLogger) sanitizeBody(body string) string {
	if body == "" || len(rl.sensitiveKeys) == 0 {
		return body
	}

	sanitized := body
	for _, key := range rl.sensitiveKeys {
		// Simple pattern to hide sensitive values
		// This is a basic implementation; in production, you might want more sophisticated sanitization
		if strings.Contains(strings.ToLower(body), strings.ToLower(key)) {
			sanitized = strings.ReplaceAll(sanitized, key, "***REDACTED***")
		}
	}

	return sanitized
}

// getLogLevel determines the appropriate log level based on status code
func (rl *RequestLogger) getLogLevel(statusCode int) logrus.Level {
	switch {
	case statusCode >= 500:
		return logrus.ErrorLevel
	case statusCode >= 400:
		return logrus.WarnLevel
	case statusCode >= 300:
		return logrus.InfoLevel
	default:
		return logrus.InfoLevel
	}
}

// shouldSkipLogging checks if logging should be skipped for a given path
func shouldSkipLogging(path string, skipPaths map[string]bool) bool {
	if skipPaths == nil {
		return false
	}
	return skipPaths[path]
}

// SimpleLoggerMiddleware creates a simple logger middleware for development
func SimpleLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format("02/Jan/2006:15:04:05 -0700"),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	})
}

// AccessLogMiddleware creates an access log middleware for production
func AccessLogMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// Log in Common Log Format (CLF) style
		logger.WithFields(logrus.Fields{
			"client_ip":   c.ClientIP(),
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": c.Writer.Status(),
			"size":        c.Writer.Size(),
			"latency":     time.Since(start),
			"user_agent":  c.Request.UserAgent(),
			"referer":     c.Request.Referer(),
		}).Info("access")
	}
}

// ErrorLogMiddleware specifically logs errors with detailed information
func ErrorLogMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only log if there are errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.WithFields(logrus.Fields{
					"path":        c.Request.URL.Path,
					"method":      c.Request.Method,
					"client_ip":   c.ClientIP(),
					"status_code": c.Writer.Status(),
					"error_type":  err.Type,
				}).Error(err.Error())
			}
		}
	}
}

// MetricsLogMiddleware logs metrics-specific information
func MetricsLogMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		// Log performance metrics
		if latency > 5*time.Second {
			logger.WithFields(logrus.Fields{
				"path":        c.Request.URL.Path,
				"method":      c.Request.Method,
				"latency":     latency,
				"status_code": c.Writer.Status(),
				"type":        "slow_request",
			}).Warn("Slow request detected")
		}

		// Log large responses
		if c.Writer.Size() > 1024*1024 { // 1MB
			logger.WithFields(logrus.Fields{
				"path":        c.Request.URL.Path,
				"method":      c.Request.Method,
				"size":        c.Writer.Size(),
				"type":        "large_response",
			}).Info("Large response detected")
		}
	}
}

// DebugLogMiddleware provides detailed logging for debugging
func DebugLogMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if logger.Level != logrus.DebugLevel {
			c.Next()
			return
		}

		// Log request headers
		logger.WithFields(logrus.Fields{
			"headers": c.Request.Header,
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
			"type":    "request_headers",
		}).Debug("Request details")

		c.Next()

		// Log response headers
		logger.WithFields(logrus.Fields{
			"headers":     c.Writer.Header(),
			"status_code": c.Writer.Status(),
			"type":        "response_headers",
		}).Debug("Response details")
	}
}