package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ValidationConfig represents validation configuration
type ValidationConfig struct {
	MaxRequestSize       int64             `json:"max_request_size"`
	AllowedMimeTypes     []string          `json:"allowed_mime_types"`
	SQLInjectionProtection bool            `json:"sql_injection_protection"`
	XSSProtection        bool              `json:"xss_protection"`
	PathTraversalProtection bool          `json:"path_traversal_protection"`
	CommandInjectionProtection bool       `json:"command_injection_protection"`
	InputSanitization    bool              `json:"input_sanitization"`
	ValidationRules      map[string]ValidationRule `json:"validation_rules"`
	CustomValidators     map[string]ValidatorFunc  `json:"-"`
}

// ValidationRule represents a validation rule for a specific field
type ValidationRule struct {
	Required     bool     `json:"required"`
	MinLength    int      `json:"min_length"`
	MaxLength    int      `json:"max_length"`
	Pattern      string   `json:"pattern"`
	AllowedValues []string `json:"allowed_values"`
	DataType     string   `json:"data_type"` // string, int, float, bool, email, url, ip
	CustomValidator string `json:"custom_validator"`
}

// ValidatorFunc represents a custom validator function
type ValidatorFunc func(value interface{}) error

// DefaultValidationConfig returns a secure default configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxRequestSize:             10 * 1024 * 1024, // 10MB
		AllowedMimeTypes:          []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data"},
		SQLInjectionProtection:    true,
		XSSProtection:             true,
		PathTraversalProtection:   true,
		CommandInjectionProtection: true,
		InputSanitization:         true,
		ValidationRules:           make(map[string]ValidationRule),
		CustomValidators:          make(map[string]ValidatorFunc),
	}
}

// InputSanitizer handles input sanitization
type InputSanitizer struct {
	sqlPatterns     []*regexp.Regexp
	xssPatterns     []*regexp.Regexp
	commandPatterns []*regexp.Regexp
	pathPatterns    []*regexp.Regexp
}

// NewInputSanitizer creates a new input sanitizer
func NewInputSanitizer() *InputSanitizer {
	return &InputSanitizer{
		sqlPatterns:     compileSQLPatterns(),
		xssPatterns:     compileXSSPatterns(),
		commandPatterns: compileCommandPatterns(),
		pathPatterns:    compilePathPatterns(),
	}
}

// ValidationMiddleware creates a comprehensive validation middleware
func ValidationMiddleware(config *ValidationConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultValidationConfig()
	}

	sanitizer := NewInputSanitizer()

	return func(c *gin.Context) {
		// Validate request size
		if c.Request.ContentLength > config.MaxRequestSize {
			logrus.WithFields(logrus.Fields{
				"content_length": c.Request.ContentLength,
				"max_size":       config.MaxRequestSize,
				"path":           c.Request.URL.Path,
				"client_ip":      c.ClientIP(),
			}).Warn("Request size exceeds limit")
			c.JSON(http.StatusRequestEntityTooLarge, utils.ErrorResponse(http.StatusRequestEntityTooLarge, "Request too large"))
			c.Abort()
			return
		}

		// Validate Content-Type for requests with body
		if hasRequestBody(c.Request.Method) {
			if err := validateContentType(c, config.AllowedMimeTypes); err != nil {
				logrus.WithError(err).Warn("Content-Type validation failed")
				c.JSON(http.StatusUnsupportedMediaType, utils.ErrorResponse(http.StatusUnsupportedMediaType, "Unsupported content type"))
				c.Abort()
				return
			}
		}

		// Validate and sanitize URL parameters
		if err := validateAndSanitizeQuery(c, config, sanitizer); err != nil {
			logrus.WithError(err).Warn("Query parameter validation failed")
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid query parameters"))
			c.Abort()
			return
		}

		// Validate and sanitize path parameters
		if err := validateAndSanitizePath(c, config, sanitizer); err != nil {
			logrus.WithError(err).Warn("Path parameter validation failed")
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid path parameters"))
			c.Abort()
			return
		}

		// Validate and sanitize request body
		if hasRequestBody(c.Request.Method) {
			if err := validateAndSanitizeBody(c, config, sanitizer); err != nil {
				logrus.WithError(err).Warn("Request body validation failed")
				c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid request body"))
				c.Abort()
				return
			}
		}

		// Validate headers
		if err := validateHeaders(c, config, sanitizer); err != nil {
			logrus.WithError(err).Warn("Header validation failed")
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid headers"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// hasRequestBody checks if the HTTP method typically has a request body
func hasRequestBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

// validateContentType validates the Content-Type header
func validateContentType(c *gin.Context, allowedTypes []string) error {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return fmt.Errorf("Content-Type header is required")
	}

	// Extract main content type (ignore charset and other parameters)
	mainType := strings.Split(contentType, ";")[0]
	mainType = strings.TrimSpace(mainType)

	for _, allowed := range allowedTypes {
		if mainType == allowed {
			return nil
		}
	}

	return fmt.Errorf("content type %s not allowed", mainType)
}

// validateAndSanitizeQuery validates and sanitizes URL query parameters
func validateAndSanitizeQuery(c *gin.Context, config *ValidationConfig, sanitizer *InputSanitizer) error {
	query := c.Request.URL.Query()

	for key, values := range query {
		for i, value := range values {
			// Perform security checks
			if err := sanitizer.ValidateInput(value, config); err != nil {
				return fmt.Errorf("invalid query parameter %s: %w", key, err)
			}

			// Sanitize if enabled
			if config.InputSanitization {
				sanitized := sanitizer.SanitizeInput(value, config)
				values[i] = sanitized
			}
		}
		query[key] = values
	}

	// Update the URL with sanitized query
	c.Request.URL.RawQuery = query.Encode()

	return nil
}

// validateAndSanitizePath validates and sanitizes path parameters
func validateAndSanitizePath(c *gin.Context, config *ValidationConfig, sanitizer *InputSanitizer) error {
	// Get path parameters from Gin context
	for _, param := range c.Params {
		// Validate path parameter
		if err := sanitizer.ValidateInput(param.Value, config); err != nil {
			return fmt.Errorf("invalid path parameter %s: %w", param.Key, err)
		}

		// Check for path traversal attempts
		if config.PathTraversalProtection {
			if sanitizer.ContainsPathTraversal(param.Value) {
				return fmt.Errorf("path traversal detected in parameter %s", param.Key)
			}
		}
	}

	return nil
}

// validateAndSanitizeBody validates and sanitizes request body
func validateAndSanitizeBody(c *gin.Context, config *ValidationConfig, sanitizer *InputSanitizer) error {
	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Restore body for subsequent handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	if len(body) == 0 {
		return nil // Empty body is allowed
	}

	contentType := c.GetHeader("Content-Type")
	mainType := strings.Split(contentType, ";")[0]

	switch mainType {
	case "application/json":
		return validateAndSanitizeJSON(body, config, sanitizer)
	case "application/x-www-form-urlencoded":
		return validateAndSanitizeForm(string(body), config, sanitizer)
	case "multipart/form-data":
		// Multipart form data is handled separately if needed
		return nil
	default:
		// For other content types, perform basic validation
		return sanitizer.ValidateInput(string(body), config)
	}
}

// validateAndSanitizeJSON validates and sanitizes JSON request body
func validateAndSanitizeJSON(body []byte, config *ValidationConfig, sanitizer *InputSanitizer) error {
	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Recursively validate and sanitize JSON structure
	if err := validateJSONValue(data, config, sanitizer); err != nil {
		return fmt.Errorf("JSON validation failed: %w", err)
	}

	return nil
}

// validateJSONValue recursively validates JSON values
func validateJSONValue(value interface{}, config *ValidationConfig, sanitizer *InputSanitizer) error {
	switch v := value.(type) {
	case string:
		return sanitizer.ValidateInput(v, config)
	case map[string]interface{}:
		for key, val := range v {
			// Validate key
			if err := sanitizer.ValidateInput(key, config); err != nil {
				return fmt.Errorf("invalid JSON key %s: %w", key, err)
			}
			// Validate value
			if err := validateJSONValue(val, config, sanitizer); err != nil {
				return err
			}
		}
	case []interface{}:
		for i, val := range v {
			if err := validateJSONValue(val, config, sanitizer); err != nil {
				return fmt.Errorf("invalid array element at index %d: %w", i, err)
			}
		}
	}
	return nil
}

// validateAndSanitizeForm validates form-urlencoded data
func validateAndSanitizeForm(body string, config *ValidationConfig, sanitizer *InputSanitizer) error {
	values, err := url.ParseQuery(body)
	if err != nil {
		return fmt.Errorf("invalid form data: %w", err)
	}

	for key, vals := range values {
		for _, val := range vals {
			if err := sanitizer.ValidateInput(val, config); err != nil {
				return fmt.Errorf("invalid form field %s: %w", key, err)
			}
		}
	}

	return nil
}

// validateHeaders validates request headers
func validateHeaders(c *gin.Context, config *ValidationConfig, sanitizer *InputSanitizer) error {
	// Validate specific headers that might contain user input
	suspiciousHeaders := []string{
		"User-Agent",
		"Referer",
		"X-Forwarded-For",
		"X-Real-IP",
		"X-Custom-Header",
	}

	for _, header := range suspiciousHeaders {
		value := c.GetHeader(header)
		if value != "" {
			if err := sanitizer.ValidateInput(value, config); err != nil {
				return fmt.Errorf("invalid header %s: %w", header, err)
			}
		}
	}

	return nil
}

// ValidateInput performs comprehensive input validation
func (s *InputSanitizer) ValidateInput(input string, config *ValidationConfig) error {
	// Check for SQL injection patterns
	if config.SQLInjectionProtection && s.ContainsSQLInjection(input) {
		return fmt.Errorf("potential SQL injection detected")
	}

	// Check for XSS patterns
	if config.XSSProtection && s.ContainsXSS(input) {
		return fmt.Errorf("potential XSS detected")
	}

	// Check for command injection patterns
	if config.CommandInjectionProtection && s.ContainsCommandInjection(input) {
		return fmt.Errorf("potential command injection detected")
	}

	// Check for path traversal patterns
	if config.PathTraversalProtection && s.ContainsPathTraversal(input) {
		return fmt.Errorf("potential path traversal detected")
	}

	return nil
}

// SanitizeInput sanitizes user input
func (s *InputSanitizer) SanitizeInput(input string, config *ValidationConfig) string {
	sanitized := input

	// Remove/escape potentially dangerous characters
	if config.SQLInjectionProtection {
		sanitized = s.sanitizeSQL(sanitized)
	}

	if config.XSSProtection {
		sanitized = s.sanitizeXSS(sanitized)
	}

	if config.CommandInjectionProtection {
		sanitized = s.sanitizeCommand(sanitized)
	}

	if config.PathTraversalProtection {
		sanitized = s.sanitizePath(sanitized)
	}

	return sanitized
}

// ContainsSQLInjection checks for SQL injection patterns
func (s *InputSanitizer) ContainsSQLInjection(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	for _, pattern := range s.sqlPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ContainsXSS checks for XSS patterns
func (s *InputSanitizer) ContainsXSS(input string) bool {
	input = strings.ToLower(input)

	for _, pattern := range s.xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ContainsCommandInjection checks for command injection patterns
func (s *InputSanitizer) ContainsCommandInjection(input string) bool {
	for _, pattern := range s.commandPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// ContainsPathTraversal checks for path traversal patterns
func (s *InputSanitizer) ContainsPathTraversal(input string) bool {
	for _, pattern := range s.pathPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// sanitizeSQL removes/escapes SQL injection patterns
func (s *InputSanitizer) sanitizeSQL(input string) string {
	// Escape single quotes
	input = strings.ReplaceAll(input, "'", "''")
	// Remove common SQL keywords in comments
	input = regexp.MustCompile(`(?i)--.*$`).ReplaceAllString(input, "")
	input = regexp.MustCompile(`(?i)/\*.*?\*/`).ReplaceAllString(input, "")
	return input
}

// sanitizeXSS removes/escapes XSS patterns
func (s *InputSanitizer) sanitizeXSS(input string) string {
	// HTML entity encoding for dangerous characters
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#x27;")
	input = strings.ReplaceAll(input, "&", "&amp;")
	return input
}

// sanitizeCommand removes command injection patterns
func (s *InputSanitizer) sanitizeCommand(input string) string {
	// Remove dangerous characters
	dangerousChars := []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "[", "]"}
	for _, char := range dangerousChars {
		input = strings.ReplaceAll(input, char, "")
	}
	return input
}

// sanitizePath removes path traversal patterns
func (s *InputSanitizer) sanitizePath(input string) string {
	// Clean path and prevent traversal
	cleaned := filepath.Clean(input)
	// Remove leading path separators and dots
	cleaned = strings.TrimPrefix(cleaned, "/")
	cleaned = strings.TrimPrefix(cleaned, "\\")
	cleaned = strings.TrimPrefix(cleaned, ".")
	return cleaned
}

// Pattern compilation functions

func compileSQLPatterns() []*regexp.Regexp {
	patterns := []string{
		`(?i)\bunion\s+select\b`,
		`(?i)\bselect\s+.*\bfrom\b`,
		`(?i)\binsert\s+into\b`,
		`(?i)\bdelete\s+from\b`,
		`(?i)\bdrop\s+table\b`,
		`(?i)\bupdate\s+.*\bset\b`,
		`(?i)\bor\s+1\s*=\s*1\b`,
		`(?i)\band\s+1\s*=\s*1\b`,
		`(?i)'.*or.*'.*=.*'`,
		`(?i)\bexec\s*\(\s*`,
		`(?i)\bsp_\w+\b`,
		`(?i)--`,
		`(?i)/\*.*\*/`,
		`(?i)\bxp_\w+\b`,
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiled[i] = regexp.MustCompile(pattern)
	}
	return compiled
}

func compileXSSPatterns() []*regexp.Regexp {
	patterns := []string{
		`(?i)<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>`,
		`(?i)<.*\bon\w+\s*=.*>`,
		`(?i)javascript:`,
		`(?i)<iframe\b`,
		`(?i)<object\b`,
		`(?i)<embed\b`,
		`(?i)<link\b`,
		`(?i)<meta\b`,
		`(?i)<form\b`,
		`(?i)document\.cookie`,
		`(?i)document\.write`,
		`(?i)eval\s*\(`,
		`(?i)expression\s*\(`,
		`(?i)vbscript:`,
		`(?i)data:`,
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiled[i] = regexp.MustCompile(pattern)
	}
	return compiled
}

func compileCommandPatterns() []*regexp.Regexp {
	patterns := []string{
		`[;&|]`,
		`\$\(.*\)`,
		"`.*`",
		`\|\s*\w+`,
		`>\s*\/`,
		`<\s*\/`,
		`\bnc\b.*-e`,
		`\bnetcat\b.*-e`,
		`\bwget\b.*\|`,
		`\bcurl\b.*\|`,
		`\bchmod\b.*\+x`,
		`\/bin\/.*sh`,
		`\bpython\b.*-c`,
		`\bperl\b.*-e`,
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiled[i] = regexp.MustCompile(pattern)
	}
	return compiled
}

func compilePathPatterns() []*regexp.Regexp {
	patterns := []string{
		`\.\./`,
		`\.\.\\`,
		`\.\./.*\.\./`,
		`\.\.\\.*\.\.\\`,
		`/etc/passwd`,
		`/etc/shadow`,
		`/proc/self/environ`,
		`\.\..*\/.*\.\.`,
		`%2e%2e%2f`,
		`%2e%2e/`,
		`..%2f`,
		`%2e%2e%5c`,
	}

	compiled := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		compiled[i] = regexp.MustCompile(pattern)
	}
	return compiled
}

// FieldValidationMiddleware creates middleware for specific field validation
func FieldValidationMiddleware(rules map[string]ValidationRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only validate JSON bodies
		if c.GetHeader("Content-Type") != "application/json" {
			c.Next()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Failed to read request body"))
			c.Abort()
			return
		}

		// Restore body
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		var data map[string]interface{}
		if err := json.Unmarshal(body, &data); err != nil {
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid JSON format"))
			c.Abort()
			return
		}

		// Validate fields according to rules
		for fieldName, rule := range rules {
			if err := validateField(fieldName, data[fieldName], rule); err != nil {
				logrus.WithError(err).WithField("field", fieldName).Warn("Field validation failed")
				c.JSON(http.StatusBadRequest, utils.ErrorResponseWithDetails(
					http.StatusBadRequest,
					fmt.Sprintf("Validation failed for field '%s'", fieldName),
					[]utils.ErrorDetail{{Message: err.Error()}},
				))
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// validateField validates a single field according to its rule
func validateField(fieldName string, value interface{}, rule ValidationRule) error {
	// Check required fields
	if rule.Required && (value == nil || value == "") {
		return fmt.Errorf("field %s is required", fieldName)
	}

	if value == nil {
		return nil // Field is not required and is nil
	}

	// Convert value to string for length and pattern validation
	strValue := fmt.Sprintf("%v", value)

	// Check length constraints
	if rule.MinLength > 0 && len(strValue) < rule.MinLength {
		return fmt.Errorf("field %s must be at least %d characters long", fieldName, rule.MinLength)
	}

	if rule.MaxLength > 0 && len(strValue) > rule.MaxLength {
		return fmt.Errorf("field %s must be at most %d characters long", fieldName, rule.MaxLength)
	}

	// Check pattern matching
	if rule.Pattern != "" {
		matched, err := regexp.MatchString(rule.Pattern, strValue)
		if err != nil {
			return fmt.Errorf("invalid pattern for field %s: %w", fieldName, err)
		}
		if !matched {
			return fmt.Errorf("field %s does not match required pattern", fieldName)
		}
	}

	// Check allowed values
	if len(rule.AllowedValues) > 0 {
		allowed := false
		for _, allowedValue := range rule.AllowedValues {
			if strValue == allowedValue {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("field %s has invalid value", fieldName)
		}
	}

	// Check data type
	if rule.DataType != "" {
		if err := validateDataType(strValue, rule.DataType); err != nil {
			return fmt.Errorf("field %s has invalid data type: %w", fieldName, err)
		}
	}

	return nil
}

// validateDataType validates the data type of a value
func validateDataType(value, dataType string) error {
	switch dataType {
	case "string":
		return nil // Already a string
	case "int":
		_, err := strconv.Atoi(value)
		return err
	case "float":
		_, err := strconv.ParseFloat(value, 64)
		return err
	case "bool":
		_, err := strconv.ParseBool(value)
		return err
	case "email":
		return validateEmail(value)
	case "url":
		return validateURL(value)
	case "ip":
		return validateIP(value)
	default:
		return fmt.Errorf("unknown data type: %s", dataType)
	}
}

// validateEmail validates email format
func validateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// validateURL validates URL format
func validateURL(urlStr string) error {
	_, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}
	return nil
}

// validateIP validates IP address format
func validateIP(ip string) error {
	// Simple IP validation (IPv4)
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipRegex.MatchString(ip) {
		return fmt.Errorf("invalid IP format")
	}

	// Validate each octet
	parts := strings.Split(ip, ".")
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return fmt.Errorf("invalid IP address")
		}
	}
	return nil
}

// FileUploadValidationMiddleware validates file uploads
func FileUploadValidationMiddleware(maxSize int64, allowedTypes []string, allowedExtensions []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process multipart form data
		if !strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
			c.Next()
			return
		}

		err := c.Request.ParseMultipartForm(maxSize)
		if err != nil {
			logrus.WithError(err).Warn("Failed to parse multipart form")
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Invalid multipart form"))
			c.Abort()
			return
		}

		for fieldName, files := range c.Request.MultipartForm.File {
			for _, file := range files {
				// Check file size
				if file.Size > maxSize {
					c.JSON(http.StatusRequestEntityTooLarge,
						utils.ErrorResponseWithDetails(http.StatusRequestEntityTooLarge, "File too large", []utils.ErrorDetail{{Message: fmt.Sprintf("Field: %s, Size: %d", fieldName, file.Size)}}))
					c.Abort()
					return
				}

				// Check file extension
				ext := strings.ToLower(filepath.Ext(file.Filename))
				if !isAllowedExtension(ext, allowedExtensions) {
					c.JSON(http.StatusUnsupportedMediaType,
						utils.ErrorResponseWithDetails(http.StatusUnsupportedMediaType, "File type not allowed", []utils.ErrorDetail{{Message: fmt.Sprintf("Field: %s, Extension: %s", fieldName, ext)}}))
					c.Abort()
					return
				}

				// Check MIME type if provided
				if len(allowedTypes) > 0 {
					contentType := file.Header.Get("Content-Type")
					if !isAllowedMimeType(contentType, allowedTypes) {
						c.JSON(http.StatusUnsupportedMediaType,
							utils.ErrorResponseWithDetails(http.StatusUnsupportedMediaType, "MIME type not allowed", []utils.ErrorDetail{{Message: fmt.Sprintf("Field: %s, Type: %s", fieldName, contentType)}}))
						c.Abort()
						return
					}
				}

				// Check for potentially dangerous filenames
				if containsDangerousPath(file.Filename) {
					c.JSON(http.StatusBadRequest,
						utils.ErrorResponseWithDetails(http.StatusBadRequest, "Dangerous filename detected", []utils.ErrorDetail{{Message: fmt.Sprintf("Field: %s, Filename: %s", fieldName, file.Filename)}}))
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

// Helper functions for file upload validation

func isAllowedExtension(ext string, allowedExtensions []string) bool {
	for _, allowed := range allowedExtensions {
		if ext == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}

func isAllowedMimeType(mimeType string, allowedTypes []string) bool {
	for _, allowed := range allowedTypes {
		if mimeType == allowed {
			return true
		}
	}
	return false
}

func containsDangerousPath(filename string) bool {
	// Check for path traversal in filename
	if strings.Contains(filename, "..") {
		return true
	}

	// Check for absolute paths
	if strings.HasPrefix(filename, "/") || strings.HasPrefix(filename, "\\") {
		return true
	}

	// Check for dangerous characters
	dangerousChars := []rune{'<', '>', ':', '"', '|', '?', '*'}
	for _, char := range filename {
		for _, dangerous := range dangerousChars {
			if char == dangerous {
				return true
			}
		}
		// Check for control characters
		if unicode.IsControl(char) {
			return true
		}
	}

	return false
}