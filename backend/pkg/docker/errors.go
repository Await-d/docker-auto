package docker

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"syscall"
	"time"
)

// DockerError represents a Docker-specific error
type DockerError struct {
	Type       ErrorType `json:"type"`
	Operation  string    `json:"operation"`
	Resource   string    `json:"resource"`
	Message    string    `json:"message"`
	Underlying error     `json:"-"`
	Code       string    `json:"code,omitempty"`
	Retryable  bool      `json:"retryable"`
	Timestamp  time.Time `json:"timestamp"`
}

// ErrorType defines types of Docker errors
type ErrorType string

const (
	// Connection errors
	ErrorTypeConnection     ErrorType = "connection"
	ErrorTypePermission     ErrorType = "permission"
	ErrorTypeAuthentication ErrorType = "authentication"
	ErrorTypeTimeout        ErrorType = "timeout"

	// Resource errors
	ErrorTypeNotFound      ErrorType = "not_found"
	ErrorTypeAlreadyExists ErrorType = "already_exists"
	ErrorTypeConflict      ErrorType = "conflict"
	ErrorTypeInvalidConfig ErrorType = "invalid_config"

	// Resource constraint errors
	ErrorTypeOutOfMemory ErrorType = "out_of_memory"
	ErrorTypeOutOfDisk   ErrorType = "out_of_disk"
	ErrorTypeResourceLimit ErrorType = "resource_limit"

	// Operation errors
	ErrorTypeOperationFailed ErrorType = "operation_failed"
	ErrorTypeInvalidOperation ErrorType = "invalid_operation"
	ErrorTypeUnsupported     ErrorType = "unsupported"

	// System errors
	ErrorTypeSystemError ErrorType = "system_error"
	ErrorTypeUnknown     ErrorType = "unknown"

	// Network errors
	ErrorTypeNetworkError ErrorType = "network_error"
	ErrorTypeDNSError     ErrorType = "dns_error"

	// Registry errors
	ErrorTypeRegistryError ErrorType = "registry_error"
	ErrorTypeImagePull     ErrorType = "image_pull"
	ErrorTypeImagePush     ErrorType = "image_push"
)

// Error implements the error interface
func (e *DockerError) Error() string {
	if e.Resource != "" {
		return fmt.Sprintf("%s %s failed: %s", e.Operation, e.Resource, e.Message)
	}
	return fmt.Sprintf("%s failed: %s", e.Operation, e.Message)
}

// Unwrap returns the underlying error
func (e *DockerError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns whether the error is retryable
func (e *DockerError) IsRetryable() bool {
	return e.Retryable
}

// GetType returns the error type
func (e *DockerError) GetType() ErrorType {
	return e.Type
}

// GetCode returns the error code
func (e *DockerError) GetCode() string {
	return e.Code
}

// NewDockerError creates a new Docker error
func NewDockerError(errorType ErrorType, operation, resource, message string, underlying error) *DockerError {
	return &DockerError{
		Type:       errorType,
		Operation:  operation,
		Resource:   resource,
		Message:    message,
		Underlying: underlying,
		Retryable:  isRetryableError(errorType, underlying),
		Timestamp:  time.Now(),
	}
}

// WrapDockerError wraps an existing error as a Docker error
func WrapDockerError(err error, operation, resource string) *DockerError {
	if err == nil {
		return nil
	}

	// If it's already a DockerError, return it
	if dockerErr, ok := err.(*DockerError); ok {
		return dockerErr
	}

	errorType, retryable := classifyError(err)
	message := err.Error()

	return &DockerError{
		Type:       errorType,
		Operation:  operation,
		Resource:   resource,
		Message:    message,
		Underlying: err,
		Retryable:  retryable,
		Timestamp:  time.Now(),
	}
}

// classifyError classifies an error into our error taxonomy
func classifyError(err error) (ErrorType, bool) {
	if err == nil {
		return ErrorTypeUnknown, false
	}

	errStr := strings.ToLower(err.Error())

	// Check for specific error patterns
	switch {
	// Connection errors
	case strings.Contains(errStr, "connection refused"):
		return ErrorTypeConnection, true
	case strings.Contains(errStr, "no such host"):
		return ErrorTypeDNSError, true
	case strings.Contains(errStr, "timeout"):
		return ErrorTypeTimeout, true
	case strings.Contains(errStr, "permission denied"):
		return ErrorTypePermission, false
	case strings.Contains(errStr, "unauthorized"):
		return ErrorTypeAuthentication, false

	// Resource errors
	case strings.Contains(errStr, "no such container"):
		return ErrorTypeNotFound, false
	case strings.Contains(errStr, "no such image"):
		return ErrorTypeNotFound, false
	case strings.Contains(errStr, "already exists"):
		return ErrorTypeAlreadyExists, false
	case strings.Contains(errStr, "conflict"):
		return ErrorTypeConflict, false

	// System resource errors
	case strings.Contains(errStr, "out of memory"):
		return ErrorTypeOutOfMemory, false
	case strings.Contains(errStr, "no space left"):
		return ErrorTypeOutOfDisk, false
	case strings.Contains(errStr, "resource temporarily unavailable"):
		return ErrorTypeResourceLimit, true

	// Registry errors
	case strings.Contains(errStr, "pull access denied"):
		return ErrorTypeRegistryError, false
	case strings.Contains(errStr, "manifest unknown"):
		return ErrorTypeImagePull, false
	case strings.Contains(errStr, "push access denied"):
		return ErrorTypeRegistryError, false

	// Network errors
	case strings.Contains(errStr, "network"):
		return ErrorTypeNetworkError, true
	case strings.Contains(errStr, "i/o timeout"):
		return ErrorTypeTimeout, true
	}

	// Check for specific error types
	switch {
	case isNetworkError(err):
		return ErrorTypeNetworkError, true
	case isTimeoutError(err):
		return ErrorTypeTimeout, true
	case isPermissionError(err):
		return ErrorTypePermission, false
	case isSystemError(err):
		return ErrorTypeSystemError, true
	}

	return ErrorTypeUnknown, false
}

// isRetryableError determines if an error type is generally retryable
func isRetryableError(errorType ErrorType, err error) bool {
	switch errorType {
	case ErrorTypeConnection, ErrorTypeTimeout, ErrorTypeNetworkError,
		 ErrorTypeDNSError, ErrorTypeResourceLimit, ErrorTypeSystemError:
		return true
	case ErrorTypePermission, ErrorTypeAuthentication, ErrorTypeNotFound,
		 ErrorTypeAlreadyExists, ErrorTypeInvalidConfig, ErrorTypeInvalidOperation:
		return false
	default:
		// For unknown errors, check the underlying error
		return isTransientError(err)
	}
}

// Error type checking functions

// IsConnectionError checks if error is a connection error
func IsConnectionError(err error) bool {
	dockerErr, ok := err.(*DockerError)
	if ok {
		return dockerErr.Type == ErrorTypeConnection
	}
	return isNetworkError(err) || strings.Contains(strings.ToLower(err.Error()), "connection")
}

// IsTimeoutError checks if error is a timeout error
func IsTimeoutError(err error) bool {
	dockerErr, ok := err.(*DockerError)
	if ok {
		return dockerErr.Type == ErrorTypeTimeout
	}
	return isTimeoutError(err)
}

// IsNotFoundError checks if error is a not found error
func IsNotFoundError(err error) bool {
	dockerErr, ok := err.(*DockerError)
	if ok {
		return dockerErr.Type == ErrorTypeNotFound
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "no such") || strings.Contains(errStr, "not found")
}

// IsPermissionError checks if error is a permission error
func IsPermissionError(err error) bool {
	dockerErr, ok := err.(*DockerError)
	if ok {
		return dockerErr.Type == ErrorTypePermission
	}
	return isPermissionError(err)
}

// IsRetryableError checks if error is retryable
func IsRetryableError(err error) bool {
	if dockerErr, ok := err.(*DockerError); ok {
		return dockerErr.Retryable
	}
	return isTransientError(err)
}

// Specific error checking functions

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error
	if netErr, ok := err.(net.Error); ok {
		return netErr.Temporary() || netErr.Timeout()
	}

	// Check for syscall errors
	if _, ok := err.(*net.OpError); ok {
		return true
	}

	return false
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Check for net.Error timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	// Check for context timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	return strings.Contains(strings.ToLower(err.Error()), "timeout")
}

func isPermissionError(err error) bool {
	if err == nil {
		return false
	}

	// Check for syscall permission errors
	if errno, ok := err.(syscall.Errno); ok {
		return errno == syscall.EACCES || errno == syscall.EPERM
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "permission denied") ||
		   strings.Contains(errStr, "access denied") ||
		   strings.Contains(errStr, "unauthorized")
}

func isSystemError(err error) bool {
	if err == nil {
		return false
	}

	// Check for syscall errors
	if _, ok := err.(syscall.Errno); ok {
		return true
	}

	return false
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}

	// Network errors are generally transient
	if isNetworkError(err) {
		return true
	}

	// Timeout errors are transient
	if isTimeoutError(err) {
		return true
	}

	// Some system errors are transient
	if errno, ok := err.(syscall.Errno); ok {
		switch errno {
		case syscall.EAGAIN, syscall.EINTR:
			return true
		}
	}

	errStr := strings.ToLower(err.Error())
	transientKeywords := []string{
		"temporary",
		"retry",
		"busy",
		"unavailable",
		"overloaded",
	}

	for _, keyword := range transientKeywords {
		if strings.Contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// Error aggregation for batch operations

// ErrorCollector collects multiple errors
type ErrorCollector struct {
	errors []error
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]error, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// Count returns the number of errors
func (ec *ErrorCollector) Count() int {
	return len(ec.errors)
}

// Errors returns all collected errors
func (ec *ErrorCollector) Errors() []error {
	return ec.errors
}

// Error returns a combined error message
func (ec *ErrorCollector) Error() error {
	if len(ec.errors) == 0 {
		return nil
	}

	if len(ec.errors) == 1 {
		return ec.errors[0]
	}

	var messages []string
	for _, err := range ec.errors {
		messages = append(messages, err.Error())
	}

	return fmt.Errorf("multiple errors occurred: %s", strings.Join(messages, "; "))
}

// Retry logic and error handling

// RetryConfig defines retry configuration
type RetryConfig struct {
	MaxRetries    int
	InitialDelay  time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	RetryableFunc func(error) bool
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableFunc: IsRetryableError,
	}
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation func() error

// Retry executes an operation with retry logic
func Retry(operation RetryableOperation, config RetryConfig) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on the last attempt
		if attempt == config.MaxRetries {
			break
		}

		// Check if error is retryable
		if !config.RetryableFunc(err) {
			break
		}

		// Calculate delay for next attempt
		delay := time.Duration(float64(config.InitialDelay) *
			math.Pow(config.BackoffFactor, float64(attempt)))
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}

		time.Sleep(delay)
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// WithRetry wraps a function with retry logic
func WithRetry(fn func() error, maxRetries int) error {
	config := DefaultRetryConfig()
	config.MaxRetries = maxRetries
	return Retry(fn, config)
}

// Common Docker operation errors

// Container operation errors

func NewContainerNotFoundError(containerID string) *DockerError {
	return NewDockerError(
		ErrorTypeNotFound,
		"inspect",
		"container",
		fmt.Sprintf("container %s not found", containerID),
		nil,
	)
}

func NewContainerAlreadyExistsError(name string) *DockerError {
	return NewDockerError(
		ErrorTypeAlreadyExists,
		"create",
		"container",
		fmt.Sprintf("container with name %s already exists", name),
		nil,
	)
}

func NewContainerStopTimeoutError(containerID string, timeout int) *DockerError {
	return NewDockerError(
		ErrorTypeTimeout,
		"stop",
		"container",
		fmt.Sprintf("container %s failed to stop within %d seconds", containerID, timeout),
		nil,
	)
}

// Image operation errors

func NewImageNotFoundError(imageName string) *DockerError {
	return NewDockerError(
		ErrorTypeNotFound,
		"inspect",
		"image",
		fmt.Sprintf("image %s not found", imageName),
		nil,
	)
}

func NewImagePullError(imageName string, underlying error) *DockerError {
	return NewDockerError(
		ErrorTypeImagePull,
		"pull",
		"image",
		fmt.Sprintf("failed to pull image %s", imageName),
		underlying,
	)
}

func NewRegistryAuthError(registry string) *DockerError {
	return NewDockerError(
		ErrorTypeAuthentication,
		"authenticate",
		"registry",
		fmt.Sprintf("authentication failed for registry %s", registry),
		nil,
	)
}

// Connection errors

func NewDockerDaemonConnectionError(host string, underlying error) *DockerError {
	return NewDockerError(
		ErrorTypeConnection,
		"connect",
		"daemon",
		fmt.Sprintf("failed to connect to Docker daemon at %s", host),
		underlying,
	)
}

func NewDockerPermissionError(operation string) *DockerError {
	return NewDockerError(
		ErrorTypePermission,
		operation,
		"docker",
		"permission denied - make sure user has access to Docker daemon",
		nil,
	)
}

// Error context and debugging

// ErrorContext provides additional context for debugging errors
type ErrorContext struct {
	Operation   string                 `json:"operation"`
	Resource    string                 `json:"resource"`
	Parameters  map[string]interface{} `json:"parameters"`
	Environment map[string]string      `json:"environment"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewErrorContext creates a new error context
func NewErrorContext(operation, resource string) *ErrorContext {
	return &ErrorContext{
		Operation:   operation,
		Resource:    resource,
		Parameters:  make(map[string]interface{}),
		Environment: make(map[string]string),
		Timestamp:   time.Now(),
	}
}

// AddParameter adds a parameter to the context
func (ec *ErrorContext) AddParameter(key string, value interface{}) {
	ec.Parameters[key] = value
}

// AddEnvironment adds environment information to the context
func (ec *ErrorContext) AddEnvironment(key, value string) {
	ec.Environment[key] = value
}

// Error diagnostics and debugging

// DiagnoseDockerError provides detailed diagnosis of Docker errors
func DiagnoseDockerError(err error) map[string]interface{} {
	diagnosis := make(map[string]interface{})

	if err == nil {
		diagnosis["status"] = "no_error"
		return diagnosis
	}

	diagnosis["error_message"] = err.Error()
	diagnosis["timestamp"] = time.Now()

	// Check if it's a DockerError
	if dockerErr, ok := err.(*DockerError); ok {
		diagnosis["type"] = dockerErr.Type
		diagnosis["operation"] = dockerErr.Operation
		diagnosis["resource"] = dockerErr.Resource
		diagnosis["retryable"] = dockerErr.Retryable
		diagnosis["code"] = dockerErr.Code
	} else {
		errorType, retryable := classifyError(err)
		diagnosis["type"] = errorType
		diagnosis["retryable"] = retryable
	}

	// Add suggestions based on error type
	diagnosis["suggestions"] = getSuggestionsForError(err)

	return diagnosis
}

// getSuggestionsForError provides troubleshooting suggestions
func getSuggestionsForError(err error) []string {
	var suggestions []string

	if IsConnectionError(err) {
		suggestions = append(suggestions, []string{
			"Check if Docker daemon is running",
			"Verify DOCKER_HOST environment variable",
			"Check firewall and network connectivity",
			"Ensure user has permission to access Docker socket",
		}...)
	}

	if IsPermissionError(err) {
		suggestions = append(suggestions, []string{
			"Add user to docker group: sudo usermod -aG docker $USER",
			"Check Docker socket permissions: ls -la /var/run/docker.sock",
			"Try running with sudo (not recommended for production)",
		}...)
	}

	if IsTimeoutError(err) {
		suggestions = append(suggestions, []string{
			"Increase timeout values",
			"Check system load and resources",
			"Verify network connectivity",
		}...)
	}

	return suggestions
}

