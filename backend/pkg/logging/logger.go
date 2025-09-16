package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Logger represents the main structured logger
type Logger struct {
	mu        sync.RWMutex
	config    LogConfig
	output    io.Writer
	context   map[string]interface{}
	component string
	buffer    chan LogEntry
	done      chan bool
}

// NewLogger creates a new logger instance
func NewLogger(config LogConfig) (*Logger, error) {
	logger := &Logger{
		config:    config,
		context:   make(map[string]interface{}),
		buffer:    make(chan LogEntry, config.BufferSize),
		done:      make(chan bool),
	}

	// Set output writer
	switch config.Output {
	case "stdout", "":
		logger.output = os.Stdout
	case "stderr":
		logger.output = os.Stderr
	default:
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logger.output = file
	}

	// Start background writer
	go logger.writeLoop()

	return logger, nil
}

// WithComponent returns a new logger instance with a component name
func (l *Logger) WithComponent(component string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		config:    l.config,
		output:    l.output,
		context:   make(map[string]interface{}),
		component: component,
		buffer:    l.buffer,
		done:      l.done,
	}

	// Copy existing context
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	return newLogger
}

// WithContext returns a new logger instance with additional context
func (l *Logger) WithContext(ctx map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &Logger{
		config:    l.config,
		output:    l.output,
		context:   make(map[string]interface{}),
		component: l.component,
		buffer:    l.buffer,
		done:      l.done,
	}

	// Copy existing context
	for k, v := range l.context {
		newLogger.context[k] = v
	}

	// Add new context
	for k, v := range ctx {
		newLogger.context[k] = v
	}

	return newLogger
}

// WithField adds a single field to the logger context
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.WithContext(map[string]interface{}{key: value})
}

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithField("request_id", requestID)
}

// WithUserID adds user ID to logger context
func (l *Logger) WithUserID(userID string) *Logger {
	return l.WithField("user_id", userID)
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	l.log(DEBUG, message, nil, nil, fields...)
}

// Info logs an info message
func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	l.log(INFO, message, nil, nil, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	l.log(WARN, message, nil, nil, fields...)
}

// Error logs an error message
func (l *Logger) Error(message string, err error, fields ...map[string]interface{}) {
	l.log(ERROR, message, err, nil, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(message string, err error, fields ...map[string]interface{}) {
	l.log(FATAL, message, err, nil, fields...)
	os.Exit(1)
}

// LogPerformance logs a performance event
func (l *Logger) LogPerformance(operation string, duration time.Duration, fields ...map[string]interface{}) {
	perfFields := map[string]interface{}{
		"operation": operation,
		"duration":  duration.String(),
	}

	// Merge additional fields
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			perfFields[k] = v
		}
	}

	l.log(INFO, fmt.Sprintf("Performance: %s completed in %s", operation, duration.String()), nil, &duration, perfFields)
}

// LogSecurity logs a security event
func (l *Logger) LogSecurity(eventType, outcome string, fields ...map[string]interface{}) {
	secFields := map[string]interface{}{
		"event_type": eventType,
		"outcome":    outcome,
		"security":   true,
	}

	// Merge additional fields
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			secFields[k] = v
		}
	}

	l.log(WARN, fmt.Sprintf("Security Event: %s - %s", eventType, outcome), nil, nil, secFields)
}

// LogAudit logs an audit event
func (l *Logger) LogAudit(action, resource string, fields ...map[string]interface{}) {
	auditFields := map[string]interface{}{
		"action":   action,
		"resource": resource,
		"audit":    true,
	}

	// Merge additional fields
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			auditFields[k] = v
		}
	}

	l.log(INFO, fmt.Sprintf("Audit: %s on %s", action, resource), nil, nil, auditFields)
}

// log is the internal logging method
func (l *Logger) log(level LogLevel, message string, err error, duration *time.Duration, fields ...map[string]interface{}) {
	if level < l.config.Level {
		return
	}

	l.mu.RLock()
	defer l.mu.RUnlock()

	entry := LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level.String(),
		Message:   message,
		Component: l.component,
		Context:   make(map[string]interface{}),
		Duration:  duration,
	}

	// Copy logger context
	for k, v := range l.context {
		entry.Context[k] = v
	}

	// Extract request and user IDs from context
	if requestID, ok := l.context["request_id"].(string); ok {
		entry.RequestID = requestID
	}
	if userID, ok := l.context["user_id"].(string); ok {
		entry.UserID = userID
	}

	// Add additional fields
	for _, fieldMap := range fields {
		for k, v := range fieldMap {
			entry.Context[k] = v
		}
	}

	// Add error information
	if err != nil {
		errStr := err.Error()
		entry.Error = &errStr

		// Add stack trace for errors
		if level >= ERROR {
			stack := getStackTrace()
			entry.Stack = &stack
		}
	}

	// Send to buffer
	select {
	case l.buffer <- entry:
	default:
		// Buffer full, write directly to avoid blocking
		l.writeEntry(entry)
	}
}

// writeLoop processes log entries from buffer
func (l *Logger) writeLoop() {
	for {
		select {
		case entry := <-l.buffer:
			l.writeEntry(entry)
		case <-l.done:
			// Process remaining entries
			for {
				select {
				case entry := <-l.buffer:
					l.writeEntry(entry)
				default:
					return
				}
			}
		}
	}
}

// writeEntry writes a log entry to the output
func (l *Logger) writeEntry(entry LogEntry) {
	var output []byte
	var err error

	switch l.config.Format {
	case "json", "":
		output, err = json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
			return
		}
		output = append(output, '\n')
	case "text":
		output = []byte(l.formatTextEntry(entry))
	default:
		output = []byte(l.formatTextEntry(entry))
	}

	_, err = l.output.Write(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write log entry: %v\n", err)
	}
}

// formatTextEntry formats a log entry as human-readable text
func (l *Logger) formatTextEntry(entry LogEntry) string {
	var parts []string

	// Timestamp
	parts = append(parts, entry.Timestamp.Format("2006-01-02 15:04:05"))

	// Level
	parts = append(parts, fmt.Sprintf("[%s]", entry.Level))

	// Component
	if entry.Component != "" {
		parts = append(parts, fmt.Sprintf("[%s]", entry.Component))
	}

	// Request ID
	if entry.RequestID != "" {
		parts = append(parts, fmt.Sprintf("[req:%s]", entry.RequestID))
	}

	// User ID
	if entry.UserID != "" {
		parts = append(parts, fmt.Sprintf("[user:%s]", entry.UserID))
	}

	// Message
	parts = append(parts, entry.Message)

	// Duration
	if entry.Duration != nil {
		parts = append(parts, fmt.Sprintf("(duration: %s)", entry.Duration.String()))
	}

	// Error
	if entry.Error != nil {
		parts = append(parts, fmt.Sprintf("error: %s", *entry.Error))
	}

	// Context
	if len(entry.Context) > 0 {
		contextParts := make([]string, 0, len(entry.Context))
		for k, v := range entry.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", k, v))
		}
		parts = append(parts, fmt.Sprintf("{%s}", strings.Join(contextParts, ", ")))
	}

	return strings.Join(parts, " ") + "\n"
}

// getStackTrace returns the current stack trace
func getStackTrace() string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var stack []string
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}

	return strings.Join(stack, "\n")
}

// Close closes the logger and flushes remaining logs
func (l *Logger) Close() error {
	close(l.done)
	time.Sleep(100 * time.Millisecond) // Allow time for remaining logs to be written

	if closer, ok := l.output.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// FromContext extracts logger from context
func FromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value("logger").(*Logger); ok {
		return logger
	}
	// Return default logger if not found in context
	config := LogConfig{
		Level:  INFO,
		Format: "json",
		Output: "stdout",
	}
	logger, _ := NewLogger(config)
	return logger
}

// ToContext adds logger to context
func ToContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}