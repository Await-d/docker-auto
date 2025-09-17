package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogLevel represents log levels
type LogLevel string

const (
	LogLevelTrace LogLevel = "trace"
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
	LogLevelPanic LogLevel = "panic"
)

// LogFormat represents log output formats
type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

// LogConfig represents logger configuration
type LogConfig struct {
	Level      LogLevel  `json:"level" yaml:"level"`
	Format     LogFormat `json:"format" yaml:"format"`
	Output     string    `json:"output" yaml:"output"` // stdout, stderr, file path
	MaxSize    int       `json:"max_size" yaml:"max_size"`       // megabytes
	MaxAge     int       `json:"max_age" yaml:"max_age"`         // days
	MaxBackups int       `json:"max_backups" yaml:"max_backups"` // number of backups
	Compress   bool      `json:"compress" yaml:"compress"`       // compress rotated files
}

// DefaultLogConfig returns default logger configuration
func DefaultLogConfig() *LogConfig {
	return &LogConfig{
		Level:      LogLevelInfo,
		Format:     LogFormatJSON,
		Output:     "stdout",
		MaxSize:    100,   // 100MB
		MaxAge:     30,    // 30 days
		MaxBackups: 5,     // keep 5 backups
		Compress:   true,  // compress old logs
	}
}

// Logger wraps logrus.Logger with additional functionality
type Logger struct {
	*logrus.Logger
	config *LogConfig
}

// NewLogger creates a new logger with the given configuration
func NewLogger(config *LogConfig) (*Logger, error) {
	if config == nil {
		config = DefaultLogConfig()
	}

	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}
	logger.SetLevel(level)

	// Set log format
	switch config.Format {
	case LogFormatJSON:
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	case LogFormatText:
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
			DisableColors:   false,
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}

	// Set output
	output, err := getLogOutput(config)
	if err != nil {
		return nil, fmt.Errorf("failed to setup log output: %w", err)
	}
	logger.SetOutput(output)

	return &Logger{
		Logger: logger,
		config: config,
	}, nil
}

// getLogOutput returns the appropriate io.Writer based on configuration
func getLogOutput(config *LogConfig) (io.Writer, error) {
	switch strings.ToLower(config.Output) {
	case "stdout":
		return os.Stdout, nil
	case "stderr":
		return os.Stderr, nil
	case "":
		return os.Stdout, nil
	default:
		// File output with rotation
		if err := os.MkdirAll(filepath.Dir(config.Output), 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		return &lumberjack.Logger{
			Filename:   config.Output,
			MaxSize:    config.MaxSize,
			MaxAge:     config.MaxAge,
			MaxBackups: config.MaxBackups,
			Compress:   config.Compress,
		}, nil
	}
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// WithField creates a logger with a single additional field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithError creates a logger with an error field
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.Logger.WithError(err)
}

// WithComponent creates a logger with a component field
func (l *Logger) WithComponent(component string) *logrus.Entry {
	return l.WithField("component", component)
}

// WithRequestID creates a logger with a request ID field
func (l *Logger) WithRequestID(requestID string) *logrus.Entry {
	return l.WithField("request_id", requestID)
}

// WithUserID creates a logger with a user ID field
func (l *Logger) WithUserID(userID interface{}) *logrus.Entry {
	return l.WithField("user_id", userID)
}

// GetConfig returns the logger configuration
func (l *Logger) GetConfig() *LogConfig {
	return l.config
}

// SetLevel changes the logger level
func (l *Logger) SetLevel(level LogLevel) error {
	logrusLevel, err := logrus.ParseLevel(string(level))
	if err != nil {
		return fmt.Errorf("invalid log level %s: %w", level, err)
	}
	l.Logger.SetLevel(logrusLevel)
	l.config.Level = level
	return nil
}

// Structured logging methods

// LogHTTPRequest logs HTTP request details
func (l *Logger) LogHTTPRequest(method, path, userAgent, clientIP string, statusCode int, duration time.Duration) {
	l.WithFields(logrus.Fields{
		"method":      method,
		"path":        path,
		"user_agent":  userAgent,
		"client_ip":   clientIP,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"type":        "http_request",
	}).Info("HTTP request processed")
}

// LogDatabaseQuery logs database query details
func (l *Logger) LogDatabaseQuery(query string, args []interface{}, duration time.Duration, err error) {
	fields := logrus.Fields{
		"query":       query,
		"args":        args,
		"duration_ms": duration.Milliseconds(),
		"type":        "database_query",
	}

	if err != nil {
		l.WithFields(fields).WithError(err).Error("Database query failed")
	} else {
		l.WithFields(fields).Debug("Database query executed")
	}
}

// LogDockerOperation logs Docker operations
func (l *Logger) LogDockerOperation(operation, containerID, image string, duration time.Duration, err error) {
	fields := logrus.Fields{
		"operation":    operation,
		"container_id": containerID,
		"image":        image,
		"duration_ms":  duration.Milliseconds(),
		"type":         "docker_operation",
	}

	if err != nil {
		l.WithFields(fields).WithError(err).Error("Docker operation failed")
	} else {
		l.WithFields(fields).Info("Docker operation completed")
	}
}

// LogSecurityEvent logs security-related events
func (l *Logger) LogSecurityEvent(event, userID, clientIP, details string) {
	l.WithFields(logrus.Fields{
		"event":     event,
		"user_id":   userID,
		"client_ip": clientIP,
		"details":   details,
		"type":      "security_event",
	}).Warn("Security event detected")
}

// LogPerformanceMetric logs performance metrics
func (l *Logger) LogPerformanceMetric(metric string, value float64, unit string, tags map[string]string) {
	fields := logrus.Fields{
		"metric": metric,
		"value":  value,
		"unit":   unit,
		"type":   "performance_metric",
	}

	for k, v := range tags {
		fields[k] = v
	}

	l.WithFields(fields).Info("Performance metric recorded")
}

// LogBusinessEvent logs business logic events
func (l *Logger) LogBusinessEvent(event, entityType, entityID string, metadata map[string]interface{}) {
	fields := logrus.Fields{
		"event":       event,
		"entity_type": entityType,
		"entity_id":   entityID,
		"type":        "business_event",
	}

	for k, v := range metadata {
		fields[k] = v
	}

	l.WithFields(fields).Info("Business event occurred")
}

// Helper functions for common logging patterns

// LogStartup logs application startup information
func (l *Logger) LogStartup(appName, version, env string, port int) {
	l.WithFields(logrus.Fields{
		"app_name": appName,
		"version":  version,
		"env":      env,
		"port":     port,
		"type":     "startup",
	}).Info("Application starting")
}

// LogShutdown logs application shutdown information
func (l *Logger) LogShutdown(appName string, reason string) {
	l.WithFields(logrus.Fields{
		"app_name": appName,
		"reason":   reason,
		"type":     "shutdown",
	}).Info("Application shutting down")
}

// LogError logs error with context
func (l *Logger) LogError(err error, context string, fields map[string]interface{}) {
	logFields := logrus.Fields{
		"context": context,
		"type":    "error",
	}

	for k, v := range fields {
		logFields[k] = v
	}

	l.WithFields(logFields).WithError(err).Error("Error occurred")
}

// Global logger instance
var defaultLogger *Logger

// InitDefaultLogger initializes the default global logger
func InitDefaultLogger(config *LogConfig) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// GetDefaultLogger returns the default global logger
func GetDefaultLogger() *Logger {
	if defaultLogger == nil {
		// Initialize with default config if not already initialized
		logger, _ := NewLogger(DefaultLogConfig())
		defaultLogger = logger
	}
	return defaultLogger
}

// Global convenience functions using default logger

// Log logs a message at the given level
func Log(level LogLevel, msg string) {
	logger := GetDefaultLogger()
	switch level {
	case LogLevelTrace:
		logger.Trace(msg)
	case LogLevelDebug:
		logger.Debug(msg)
	case LogLevelInfo:
		logger.Info(msg)
	case LogLevelWarn:
		logger.Warn(msg)
	case LogLevelError:
		logger.Error(msg)
	case LogLevelFatal:
		logger.Fatal(msg)
	case LogLevelPanic:
		logger.Panic(msg)
	}
}

// Debug logs a debug message
func Debug(msg string) {
	GetDefaultLogger().Debug(msg)
}

// Info logs an info message
func Info(msg string) {
	GetDefaultLogger().Info(msg)
}

// Warn logs a warning message
func Warn(msg string) {
	GetDefaultLogger().Warn(msg)
}

// Error logs an error message
func Error(msg string) {
	GetDefaultLogger().Error(msg)
}

// Fatal logs a fatal message and exits
func Fatal(msg string) {
	GetDefaultLogger().Fatal(msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, args ...interface{}) {
	GetDefaultLogger().Debugf(format, args...)
}

// Infof logs a formatted info message
func Infof(format string, args ...interface{}) {
	GetDefaultLogger().Infof(format, args...)
}

// Warnf logs a formatted warning message
func Warnf(format string, args ...interface{}) {
	GetDefaultLogger().Warnf(format, args...)
}

// Errorf logs a formatted error message
func Errorf(format string, args ...interface{}) {
	GetDefaultLogger().Errorf(format, args...)
}

// Fatalf logs a formatted fatal message and exits
func Fatalf(format string, args ...interface{}) {
	GetDefaultLogger().Fatalf(format, args...)
}

// WithFields creates a logger entry with fields using default logger
func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetDefaultLogger().WithFields(fields)
}

// WithField creates a logger entry with a field using default logger
func WithField(key string, value interface{}) *logrus.Entry {
	return GetDefaultLogger().WithField(key, value)
}

// WithError creates a logger entry with error using default logger
func WithError(err error) *logrus.Entry {
	return GetDefaultLogger().WithError(err)
}