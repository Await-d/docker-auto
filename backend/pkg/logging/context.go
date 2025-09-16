package logging

import (
	"context"
	"time"
)

// ContextKey is a custom type for context keys
type ContextKey string

const (
	LoggerKey    ContextKey = "logger"
	RequestIDKey ContextKey = "request_id"
	UserIDKey    ContextKey = "user_id"
	ComponentKey ContextKey = "component"
)

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}

// GetLogger retrieves a logger from context or returns a default logger
func GetLogger(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(LoggerKey).(*Logger); ok {
		return logger
	}

	// Return default logger if not found
	config := LogConfig{
		Level:  INFO,
		Format: "json",
		Output: "stdout",
	}
	logger, _ := NewLogger(config)
	return logger
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// WithComponent adds a component name to the context
func WithComponent(ctx context.Context, component string) context.Context {
	return context.WithValue(ctx, ComponentKey, component)
}

// GetComponent retrieves component name from context
func GetComponent(ctx context.Context) string {
	if component, ok := ctx.Value(ComponentKey).(string); ok {
		return component
	}
	return ""
}

// EnhanceLogger adds context information to a logger
func EnhanceLogger(ctx context.Context, logger *Logger) *Logger {
	enhancedLogger := logger

	if requestID := GetRequestID(ctx); requestID != "" {
		enhancedLogger = enhancedLogger.WithRequestID(requestID)
	}

	if userID := GetUserID(ctx); userID != "" {
		enhancedLogger = enhancedLogger.WithUserID(userID)
	}

	if component := GetComponent(ctx); component != "" {
		enhancedLogger = enhancedLogger.WithComponent(component)
	}

	return enhancedLogger
}

// LogOperation logs the start and end of an operation with timing
func LogOperation(ctx context.Context, logger *Logger, operation string, fn func() error) error {
	start := time.Now()
	enhancedLogger := EnhanceLogger(ctx, logger)

	enhancedLogger.Info("Operation started", map[string]interface{}{
		"operation": operation,
	})

	err := fn()
	duration := time.Since(start)

	if err != nil {
		enhancedLogger.Error("Operation failed", err, map[string]interface{}{
			"operation": operation,
			"duration":  duration.String(),
		})
	} else {
		enhancedLogger.LogPerformance(operation, duration)
		enhancedLogger.Info("Operation completed", map[string]interface{}{
			"operation": operation,
			"duration":  duration.String(),
		})
	}

	return err
}

// LogOperationWithResult logs an operation with result
func LogOperationWithResult[T any](ctx context.Context, logger *Logger, operation string, fn func() (T, error)) (T, error) {
	start := time.Now()
	enhancedLogger := EnhanceLogger(ctx, logger)

	enhancedLogger.Info("Operation started", map[string]interface{}{
		"operation": operation,
	})

	result, err := fn()
	duration := time.Since(start)

	if err != nil {
		enhancedLogger.Error("Operation failed", err, map[string]interface{}{
			"operation": operation,
			"duration":  duration.String(),
		})
	} else {
		enhancedLogger.LogPerformance(operation, duration)
		enhancedLogger.Info("Operation completed", map[string]interface{}{
			"operation": operation,
			"duration":  duration.String(),
		})
	}

	return result, err
}

// LogDatabaseOperation logs database operations with query information
func LogDatabaseOperation(ctx context.Context, logger *Logger, query string, args []interface{}, fn func() error) error {
	start := time.Now()
	enhancedLogger := EnhanceLogger(ctx, logger)

	enhancedLogger.Debug("Database query started", map[string]interface{}{
		"query": query,
		"args":  args,
	})

	err := fn()
	duration := time.Since(start)

	if err != nil {
		enhancedLogger.Error("Database query failed", err, map[string]interface{}{
			"query":    query,
			"args":     args,
			"duration": duration.String(),
		})
	} else {
		enhancedLogger.LogPerformance("database_query", duration, map[string]interface{}{
			"query": query,
		})
		enhancedLogger.Debug("Database query completed", map[string]interface{}{
			"query":    query,
			"duration": duration.String(),
		})
	}

	return err
}

// LogHTTPClient logs HTTP client requests
func LogHTTPClient(ctx context.Context, logger *Logger, method, url string, statusCode int, duration time.Duration, err error) {
	enhancedLogger := EnhanceLogger(ctx, logger)

	fields := map[string]interface{}{
		"method":      method,
		"url":         url,
		"status_code": statusCode,
		"duration":    duration.String(),
	}

	if err != nil {
		enhancedLogger.Error("HTTP client request failed", err, fields)
	} else {
		enhancedLogger.LogPerformance("http_client_request", duration, fields)

		if statusCode >= 400 {
			enhancedLogger.Warn("HTTP client request returned error status", fields)
		} else {
			enhancedLogger.Debug("HTTP client request completed", fields)
		}
	}
}