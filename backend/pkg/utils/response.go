package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse represents a standardized API response format
type APIResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Success   bool        `json:"success"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
}

// Meta contains additional metadata for the response
type Meta struct {
	Pagination *Pagination `json:"pagination,omitempty"`
	Count      int64       `json:"count,omitempty"`
	Total      int64       `json:"total,omitempty"`
	Duration   string      `json:"duration,omitempty"`
	Version    string      `json:"version,omitempty"`
}

// Pagination contains pagination information
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	APIResponse
	Pagination *Pagination `json:"pagination,omitempty"`
}

// ErrorDetail provides detailed error information
type ErrorDetail struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// APIError represents an error response with details
type APIError struct {
	APIResponse
	Details []ErrorDetail `json:"details,omitempty"`
	Stack   string        `json:"stack,omitempty"` // Only in development
}

// ResponseBuilder helps build API responses
type ResponseBuilder struct {
	ctx       *gin.Context
	startTime time.Time
}

// NewResponseBuilder creates a new response builder
func NewResponseBuilder(ctx *gin.Context) *ResponseBuilder {
	return &ResponseBuilder{
		ctx:       ctx,
		startTime: time.Now(),
	}
}

// SuccessResponse creates a successful API response
func SuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Code:      http.StatusOK,
		Message:   "Success",
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
	}
}

// SuccessResponseWithMessage creates a successful API response with custom message
func SuccessResponseWithMessage(data interface{}, message string) *APIResponse {
	return &APIResponse{
		Code:      http.StatusOK,
		Message:   message,
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
	}
}

// ErrorResponse creates an error API response
func ErrorResponse(code int, message string) *APIResponse {
	return &APIResponse{
		Code:      code,
		Message:   message,
		Success:   false,
		Timestamp: time.Now().UTC(),
	}
}

// ErrorResponseWithDetails creates an error API response with details
func ErrorResponseWithDetails(code int, message string, details []ErrorDetail) *APIError {
	return &APIError{
		APIResponse: APIResponse{
			Code:      code,
			Message:   message,
			Success:   false,
			Timestamp: time.Now().UTC(),
		},
		Details: details,
	}
}

// Success sends a successful response
func (rb *ResponseBuilder) Success(data interface{}) {
	response := &APIResponse{
		Code:      http.StatusOK,
		Message:   "Success",
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
		RequestID: rb.getRequestID(),
		Meta:      rb.buildMeta(),
	}

	rb.ctx.JSON(http.StatusOK, response)
}

// SuccessWithMessage sends a successful response with custom message
func (rb *ResponseBuilder) SuccessWithMessage(data interface{}, message string) {
	response := &APIResponse{
		Code:      http.StatusOK,
		Message:   message,
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
		RequestID: rb.getRequestID(),
		Meta:      rb.buildMeta(),
	}

	rb.ctx.JSON(http.StatusOK, response)
}

// SuccessWithPagination sends a successful response with pagination
func (rb *ResponseBuilder) SuccessWithPagination(data interface{}, pagination *Pagination) {
	meta := rb.buildMeta()
	meta.Pagination = pagination

	response := &APIResponse{
		Code:      http.StatusOK,
		Message:   "Success",
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
		RequestID: rb.getRequestID(),
		Meta:      meta,
	}

	rb.ctx.JSON(http.StatusOK, response)
}

// Created sends a 201 Created response
func (rb *ResponseBuilder) Created(data interface{}) {
	response := &APIResponse{
		Code:      http.StatusCreated,
		Message:   "Created successfully",
		Data:      data,
		Success:   true,
		Timestamp: time.Now().UTC(),
		RequestID: rb.getRequestID(),
		Meta:      rb.buildMeta(),
	}

	rb.ctx.JSON(http.StatusCreated, response)
}

// NoContent sends a 204 No Content response
func (rb *ResponseBuilder) NoContent() {
	rb.ctx.Status(http.StatusNoContent)
}

// Error sends an error response
func (rb *ResponseBuilder) Error(code int, message string) {
	response := &APIResponse{
		Code:      code,
		Message:   message,
		Success:   false,
		Timestamp: time.Now().UTC(),
		RequestID: rb.getRequestID(),
		Meta:      rb.buildMeta(),
	}

	rb.ctx.JSON(code, response)
}

// ErrorWithDetails sends an error response with details
func (rb *ResponseBuilder) ErrorWithDetails(code int, message string, details []ErrorDetail) {
	response := &APIError{
		APIResponse: APIResponse{
			Code:      code,
			Message:   message,
			Success:   false,
			Timestamp: time.Now().UTC(),
			RequestID: rb.getRequestID(),
			Meta:      rb.buildMeta(),
		},
		Details: details,
	}

	rb.ctx.JSON(code, response)
}

// BadRequest sends a 400 Bad Request response
func (rb *ResponseBuilder) BadRequest(message string) {
	rb.Error(http.StatusBadRequest, message)
}

// BadRequestWithDetails sends a 400 Bad Request response with details
func (rb *ResponseBuilder) BadRequestWithDetails(message string, details []ErrorDetail) {
	rb.ErrorWithDetails(http.StatusBadRequest, message, details)
}

// Unauthorized sends a 401 Unauthorized response
func (rb *ResponseBuilder) Unauthorized(message string) {
	if message == "" {
		message = "Unauthorized"
	}
	rb.Error(http.StatusUnauthorized, message)
}

// Forbidden sends a 403 Forbidden response
func (rb *ResponseBuilder) Forbidden(message string) {
	if message == "" {
		message = "Forbidden"
	}
	rb.Error(http.StatusForbidden, message)
}

// NotFound sends a 404 Not Found response
func (rb *ResponseBuilder) NotFound(message string) {
	if message == "" {
		message = "Resource not found"
	}
	rb.Error(http.StatusNotFound, message)
}

// Conflict sends a 409 Conflict response
func (rb *ResponseBuilder) Conflict(message string) {
	if message == "" {
		message = "Resource conflict"
	}
	rb.Error(http.StatusConflict, message)
}

// UnprocessableEntity sends a 422 Unprocessable Entity response
func (rb *ResponseBuilder) UnprocessableEntity(message string, details []ErrorDetail) {
	if message == "" {
		message = "Validation failed"
	}
	rb.ErrorWithDetails(http.StatusUnprocessableEntity, message, details)
}

// InternalServerError sends a 500 Internal Server Error response
func (rb *ResponseBuilder) InternalServerError(message string) {
	if message == "" {
		message = "Internal server error"
	}
	rb.Error(http.StatusInternalServerError, message)
}

// ServiceUnavailable sends a 503 Service Unavailable response
func (rb *ResponseBuilder) ServiceUnavailable(message string) {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	rb.Error(http.StatusServiceUnavailable, message)
}

// getRequestID extracts request ID from context
func (rb *ResponseBuilder) getRequestID() string {
	if requestID, exists := rb.ctx.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}

// buildMeta builds response metadata
func (rb *ResponseBuilder) buildMeta() *Meta {
	duration := time.Since(rb.startTime)

	meta := &Meta{
		Duration: duration.String(),
		Version:  "v1", // Default version, can be extracted from context
	}

	// Extract version from context if available
	if version, exists := rb.ctx.Get("api_version"); exists {
		if v, ok := version.(string); ok {
			meta.Version = v
		}
	}

	return meta
}

// Global convenience functions for quick responses

// SuccessJSON sends a successful JSON response
func SuccessJSON(ctx *gin.Context, data interface{}) {
	NewResponseBuilder(ctx).Success(data)
}

// SuccessJSONWithMessage sends a successful JSON response with message
func SuccessJSONWithMessage(ctx *gin.Context, data interface{}, message string) {
	NewResponseBuilder(ctx).SuccessWithMessage(data, message)
}

// CreatedJSON sends a 201 Created JSON response
func CreatedJSON(ctx *gin.Context, data interface{}) {
	NewResponseBuilder(ctx).Created(data)
}

// ErrorJSON sends an error JSON response
func ErrorJSON(ctx *gin.Context, code int, message string) {
	NewResponseBuilder(ctx).Error(code, message)
}

// BadRequestJSON sends a 400 Bad Request JSON response
func BadRequestJSON(ctx *gin.Context, message string) {
	NewResponseBuilder(ctx).BadRequest(message)
}

// UnauthorizedJSON sends a 401 Unauthorized JSON response
func UnauthorizedJSON(ctx *gin.Context, message string) {
	NewResponseBuilder(ctx).Unauthorized(message)
}

// ForbiddenJSON sends a 403 Forbidden JSON response
func ForbiddenJSON(ctx *gin.Context, message string) {
	NewResponseBuilder(ctx).Forbidden(message)
}

// NotFoundJSON sends a 404 Not Found JSON response
func NotFoundJSON(ctx *gin.Context, message string) {
	NewResponseBuilder(ctx).NotFound(message)
}

// ConflictJSON sends a 409 Conflict JSON response
func ConflictJSON(ctx *gin.Context, message string) {
	NewResponseBuilder(ctx).Conflict(message)
}

// InternalServerErrorJSON sends a 500 Internal Server Error JSON response
func InternalServerErrorJSON(ctx *gin.Context, message string) {
	NewResponseBuilder(ctx).InternalServerError(message)
}

// ValidationErrorJSON sends a 422 Unprocessable Entity response for validation errors
func ValidationErrorJSON(ctx *gin.Context, details []ErrorDetail) {
	NewResponseBuilder(ctx).UnprocessableEntity("Validation failed", details)
}

// CreatePagination creates pagination metadata
func CreatePagination(page, pageSize int, totalItems int64) *Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	totalPages := int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	return &Pagination{
		Page:       page,
		Limit:      pageSize,
		Total:      totalItems,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// PaginatedSuccessResponse creates a paginated success response
func PaginatedSuccessResponse(data interface{}, total int64, page, limit int) *PaginatedResponse {
	pagination := CreatePagination(page, limit, total)
	return &PaginatedResponse{
		APIResponse: APIResponse{
			Code:      http.StatusOK,
			Message:   "Success",
			Data:      data,
			Success:   true,
			Timestamp: time.Now().UTC(),
		},
		Pagination: pagination,
	}
}

// PaginatedSuccessJSON sends a successful response with pagination
func PaginatedSuccessJSON(ctx *gin.Context, data interface{}, page, pageSize int, totalItems int64) {
	pagination := CreatePagination(page, pageSize, totalItems)
	NewResponseBuilder(ctx).SuccessWithPagination(data, pagination)
}

// NewErrorDetail creates a new error detail
func NewErrorDetail(field, message, code string) ErrorDetail {
	return ErrorDetail{
		Field:   field,
		Message: message,
		Code:    code,
	}
}

// NewValidationError creates a validation error detail
func NewValidationError(field, message string) ErrorDetail {
	return ErrorDetail{
		Field:   field,
		Message: message,
		Code:    "VALIDATION_ERROR",
	}
}

// Common error messages
const (
	ErrMsgInvalidJSON       = "Invalid JSON format"
	ErrMsgMissingField      = "Required field is missing"
	ErrMsgInvalidField      = "Invalid field value"
	ErrMsgUnauthorized      = "Authentication required"
	ErrMsgForbidden         = "Access denied"
	ErrMsgNotFound          = "Resource not found"
	ErrMsgConflict          = "Resource already exists"
	ErrMsgInternalError     = "Internal server error"
	ErrMsgServiceUnavailable = "Service temporarily unavailable"
	ErrMsgRateLimitExceeded = "Rate limit exceeded"
	ErrMsgInvalidCredentials = "Invalid credentials"
	ErrMsgExpiredToken      = "Token has expired"
	ErrMsgInvalidToken      = "Invalid token"
)

// HTTP status code constants
const (
	StatusSuccessCreated = http.StatusCreated
	StatusSuccessOK      = http.StatusOK
	StatusSuccessNoContent = http.StatusNoContent
	StatusBadRequest     = http.StatusBadRequest
	StatusUnauthorized   = http.StatusUnauthorized
	StatusForbidden      = http.StatusForbidden
	StatusNotFound       = http.StatusNotFound
	StatusConflict       = http.StatusConflict
	StatusUnprocessableEntity = http.StatusUnprocessableEntity
	StatusInternalServerError = http.StatusInternalServerError
	StatusServiceUnavailable  = http.StatusServiceUnavailable
)