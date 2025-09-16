package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorType represents different types of errors
type ErrorType string

const (
	ErrorTypeValidation   ErrorType = "validation"
	ErrorTypeDatabase     ErrorType = "database"
	ErrorTypeExternal     ErrorType = "external"
	ErrorTypePermission   ErrorType = "permission"
	ErrorTypeRateLimit    ErrorType = "rate_limit"
	ErrorTypeInternal     ErrorType = "internal"
	ErrorTypeNotFound     ErrorType = "not_found"
	ErrorTypeUnauthorized ErrorType = "unauthorized"
)

// AppError represents an application error with additional context
type AppError struct {
	Type       ErrorType `json:"type"`
	Message    string    `json:"message"`
	Details    string    `json:"details,omitempty"`
	StatusCode int       `json:"status_code"`
	Internal   error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Internal != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Internal)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewAppError creates a new application error
func NewAppError(errorType ErrorType, message string, statusCode int) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: statusCode,
	}
}

// NewAppErrorWithDetails creates a new application error with details
func NewAppErrorWithDetails(errorType ErrorType, message, details string, statusCode int) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		Details:    details,
		StatusCode: statusCode,
	}
}

// NewAppErrorWithCause creates a new application error with underlying cause
func NewAppErrorWithCause(errorType ErrorType, message string, statusCode int, cause error) *AppError {
	return &AppError{
		Type:       errorType,
		Message:    message,
		StatusCode: statusCode,
		Internal:   cause,
	}
}

// ErrorHandlerMiddleware creates an error handling middleware
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return ErrorHandlerMiddlewareWithConfig(&ErrorConfig{
		EnableStackTrace: false,
		EnableDetails:    false,
	})
}

// ErrorConfig represents error handler configuration
type ErrorConfig struct {
	EnableStackTrace bool
	EnableDetails    bool
	Logger          *logrus.Logger
}

// ErrorHandlerMiddlewareWithConfig creates an error handling middleware with configuration
func ErrorHandlerMiddlewareWithConfig(config *ErrorConfig) gin.HandlerFunc {
	if config.Logger == nil {
		config.Logger = logrus.StandardLogger()
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			handlePanicRecovery(c, err, config)
		} else {
			handlePanicRecovery(c, recovered, config)
		}
	})
}

// handlePanicRecovery handles panic recovery and error formatting
func handlePanicRecovery(c *gin.Context, err interface{}, config *ErrorConfig) {
	// Log the panic with stack trace
	stackTrace := string(debug.Stack())

	logFields := logrus.Fields{
		"panic":      err,
		"path":       c.Request.URL.Path,
		"method":     c.Request.Method,
		"client_ip":  c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}

	// Add user info if available
	if user := GetUserFromContext(c); user != nil {
		logFields["user_id"] = user.UserID
		logFields["username"] = user.Username
	}

	// Add stack trace if enabled
	if config.EnableStackTrace {
		logFields["stack_trace"] = stackTrace
	}

	config.Logger.WithFields(logFields).Error("Panic recovered")

	// Prepare error response
	var response *utils.APIResponse

	if config.EnableDetails {
		response = utils.ErrorResponseWithDetails(http.StatusInternalServerError, "Internal server error", []utils.ErrorDetail{
			{Message: fmt.Sprintf("Panic: %v", err)},
		})
	} else {
		response = utils.ErrorResponse(http.StatusInternalServerError, "Internal server error")
	}

	c.JSON(http.StatusInternalServerError, response)
}

// GlobalErrorHandler handles all errors in a centralized way
func GlobalErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			handleError(c, err.Err)
			return
		}
	}
}

// handleError processes different types of errors
func handleError(c *gin.Context, err error) {
	var statusCode int
	var response *utils.APIResponse

	// Check if it's an AppError
	if ae, ok := err.(*AppError); ok {
		statusCode = ae.StatusCode

		logFields := logrus.Fields{
			"error_type": ae.Type,
			"message":    ae.Message,
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
		}

		if ae.Internal != nil {
			logFields["internal_error"] = ae.Internal.Error()
		}

		// Log based on error type
		switch ae.Type {
		case ErrorTypeValidation:
			logrus.WithFields(logFields).Warn("Validation error")
		case ErrorTypePermission, ErrorTypeUnauthorized:
			logrus.WithFields(logFields).Warn("Permission error")
		case ErrorTypeNotFound:
			logrus.WithFields(logFields).Info("Resource not found")
		case ErrorTypeRateLimit:
			logrus.WithFields(logFields).Warn("Rate limit exceeded")
		default:
			logrus.WithFields(logFields).Error("Application error")
		}

		// Build response
		response = map[string]interface{}{
			"error": map[string]interface{}{
				"type":    ae.Type,
				"message": ae.Message,
			},
		}

		if ae.Details != "" {
			response["error"].(map[string]interface{})["details"] = ae.Details
		}

	} else {
		// Handle other error types
		statusCode, response = handleGenericError(c, err)
	}

	c.JSON(statusCode, response)
}

// handleGenericError handles non-AppError types
func handleGenericError(c *gin.Context, err error) (int, *utils.APIResponse) {
	errMsg := err.Error()

	logrus.WithFields(logrus.Fields{
		"error": errMsg,
		"path":  c.Request.URL.Path,
		"method": c.Request.Method,
	}).Error("Generic error")

	// Pattern matching for common errors
	switch {
	case strings.Contains(errMsg, "validation"):
		return http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Validation failed")
	case strings.Contains(errMsg, "not found"):
		return http.StatusNotFound, utils.ErrorResponse(http.StatusNotFound, "Resource not found")
	case strings.Contains(errMsg, "unauthorized"):
		return http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Unauthorized")
	case strings.Contains(errMsg, "forbidden"):
		return http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "Forbidden")
	case strings.Contains(errMsg, "conflict"):
		return http.StatusConflict, utils.ErrorResponse(http.StatusConflict, "Conflict")
	default:
		return http.StatusInternalServerError, utils.ErrorResponse(http.StatusInternalServerError, "Internal server error")
	}
}

// ValidationErrorMiddleware specifically handles validation errors
func ValidationErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Process validation errors
		for _, err := range c.Errors {
			if err.Type == gin.ErrorTypeBind {
				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
					"path":  c.Request.URL.Path,
				}).Warn("Validation error")

				c.JSON(http.StatusBadRequest, utils.ErrorResponseWithDetails(
					"Validation failed",
					err.Error(),
				))
				return
			}
		}
	}
}

// NotFoundHandler handles 404 errors
func NotFoundHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		logrus.WithFields(logrus.Fields{
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Warn("Route not found")

		c.JSON(http.StatusNotFound, utils.ErrorResponse("Route not found"))
	}
}

// MethodNotAllowedHandler handles 405 errors
func MethodNotAllowedHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		logrus.WithFields(logrus.Fields{
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"client_ip":  c.ClientIP(),
		}).Warn("Method not allowed")

		c.JSON(http.StatusMethodNotAllowed, utils.ErrorResponse("Method not allowed"))
	}
}

// DatabaseErrorHandler handles database-specific errors
func DatabaseErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			errMsg := strings.ToLower(err.Error())

			if strings.Contains(errMsg, "database") ||
			   strings.Contains(errMsg, "sql") ||
			   strings.Contains(errMsg, "connection") {

				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
					"path":  c.Request.URL.Path,
					"type":  "database_error",
				}).Error("Database error")

				c.JSON(http.StatusInternalServerError, utils.ErrorResponse("Database error"))
				return
			}
		}
	}
}

// TimeoutErrorHandler handles timeout errors
func TimeoutErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, err := range c.Errors {
			if strings.Contains(strings.ToLower(err.Error()), "timeout") {
				logrus.WithFields(logrus.Fields{
					"error": err.Error(),
					"path":  c.Request.URL.Path,
					"type":  "timeout_error",
				}).Warn("Request timeout")

				c.JSON(http.StatusRequestTimeout, utils.ErrorResponse("Request timeout"))
				return
			}
		}
	}
}

// AbortWithError aborts the request with a custom error
func AbortWithError(c *gin.Context, appErr *AppError) {
	c.Error(appErr)
	c.Abort()
}

// AbortWithValidationError aborts with a validation error
func AbortWithValidationError(c *gin.Context, message string) {
	err := NewAppError(ErrorTypeValidation, message, http.StatusBadRequest)
	AbortWithError(c, err)
}

// AbortWithNotFoundError aborts with a not found error
func AbortWithNotFoundError(c *gin.Context, resource string) {
	message := fmt.Sprintf("%s not found", resource)
	err := NewAppError(ErrorTypeNotFound, message, http.StatusNotFound)
	AbortWithError(c, err)
}

// AbortWithUnauthorizedError aborts with an unauthorized error
func AbortWithUnauthorizedError(c *gin.Context, message string) {
	if message == "" {
		message = "Unauthorized access"
	}
	err := NewAppError(ErrorTypeUnauthorized, message, http.StatusUnauthorized)
	AbortWithError(c, err)
}

// AbortWithPermissionError aborts with a permission error
func AbortWithPermissionError(c *gin.Context, message string) {
	if message == "" {
		message = "Insufficient permissions"
	}
	err := NewAppError(ErrorTypePermission, message, http.StatusForbidden)
	AbortWithError(c, err)
}

// AbortWithInternalError aborts with an internal server error
func AbortWithInternalError(c *gin.Context, message string, cause error) {
	err := NewAppErrorWithCause(ErrorTypeInternal, message, http.StatusInternalServerError, cause)
	AbortWithError(c, err)
}