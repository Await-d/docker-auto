package utils

import (
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// Common validation patterns
var (
	// Docker image name validation
	imageNameRegex = regexp.MustCompile(`^[a-z0-9]+(?:[._-][a-z0-9]+)*(?:/[a-z0-9]+(?:[._-][a-z0-9]+)*)*$`)

	// Docker tag validation
	tagRegex = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`)

	// Container name validation
	containerNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

	// Port validation
	portRegex = regexp.MustCompile(`^\d+$`)

	// Cron expression validation (basic)
	cronRegex = regexp.MustCompile(`^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])) (\*|([0-6]))$`)
)

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []*ValidationError

func (errs ValidationErrors) Error() string {
	if len(errs) == 0 {
		return ""
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (errs ValidationErrors) HasErrors() bool {
	return len(errs) > 0
}

// Validator provides data validation utilities
type Validator struct {
	errors ValidationErrors
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make(ValidationErrors, 0),
	}
}

// AddError adds a validation error
func (v *Validator) AddError(field, message string, value interface{}) {
	v.errors = append(v.errors, &ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors returns all validation errors
func (v *Validator) GetErrors() ValidationErrors {
	return v.errors
}

// Clear clears all validation errors
func (v *Validator) Clear() {
	v.errors = make(ValidationErrors, 0)
}

// Required validates that a field is not empty
func (v *Validator) Required(field string, value interface{}) {
	if IsEmpty(value) {
		v.AddError(field, "is required", value)
	}
}

// MinLength validates minimum string length
func (v *Validator) MinLength(field string, value string, min int) {
	if len(value) < min {
		v.AddError(field, fmt.Sprintf("must be at least %d characters long", min), value)
	}
}

// MaxLength validates maximum string length
func (v *Validator) MaxLength(field string, value string, max int) {
	if len(value) > max {
		v.AddError(field, fmt.Sprintf("must not exceed %d characters", max), value)
	}
}

// Email validates email format
func (v *Validator) Email(field string, value string) {
	if value != "" {
		if _, err := mail.ParseAddress(value); err != nil {
			v.AddError(field, "must be a valid email address", value)
		}
	}
}

// URL validates URL format
func (v *Validator) URL(field string, value string) {
	if value != "" {
		if _, err := url.Parse(value); err != nil {
			v.AddError(field, "must be a valid URL", value)
		}
	}
}

// Port validates port number
func (v *Validator) Port(field string, value interface{}) {
	var port int
	var err error

	switch val := value.(type) {
	case int:
		port = val
	case string:
		if val != "" {
			port, err = strconv.Atoi(val)
			if err != nil {
				v.AddError(field, "must be a valid port number", value)
				return
			}
		}
	default:
		v.AddError(field, "must be a valid port number", value)
		return
	}

	if port < 1 || port > 65535 {
		v.AddError(field, "must be between 1 and 65535", value)
	}
}

// IP validates IP address
func (v *Validator) IP(field string, value string) {
	if value != "" {
		if net.ParseIP(value) == nil {
			v.AddError(field, "must be a valid IP address", value)
		}
	}
}

// DockerImage validates Docker image name format
func (v *Validator) DockerImage(field string, value string) {
	if value != "" {
		// Split registry/image:tag
		parts := strings.Split(value, ":")
		imagePart := parts[0]

		// Remove registry part if present
		if strings.Contains(imagePart, "/") && (strings.Contains(imagePart, ".") || strings.Contains(imagePart, ":")) {
			parts := strings.SplitN(imagePart, "/", 2)
			if len(parts) > 1 {
				imagePart = parts[1]
			}
		}

		if !imageNameRegex.MatchString(imagePart) {
			v.AddError(field, "must be a valid Docker image name", value)
		}
	}
}

// DockerTag validates Docker tag format
func (v *Validator) DockerTag(field string, value string) {
	if value != "" && !tagRegex.MatchString(value) {
		v.AddError(field, "must be a valid Docker tag", value)
	}
}

// ContainerName validates container name format
func (v *Validator) ContainerName(field string, value string) {
	if value != "" {
		if !containerNameRegex.MatchString(value) {
			v.AddError(field, "must be a valid container name (alphanumeric, dots, dashes, underscores)", value)
		}
		if len(value) > 63 {
			v.AddError(field, "must not exceed 63 characters", value)
		}
	}
}

// CronExpression validates cron expression format
func (v *Validator) CronExpression(field string, value string) {
	if value != "" && !cronRegex.MatchString(value) {
		v.AddError(field, "must be a valid cron expression (minute hour day month weekday)", value)
	}
}

// OneOf validates that value is one of allowed values
func (v *Validator) OneOf(field string, value string, allowed []string) {
	if value != "" {
		for _, allow := range allowed {
			if value == allow {
				return
			}
		}
		v.AddError(field, fmt.Sprintf("must be one of: %s", strings.Join(allowed, ", ")), value)
	}
}

// Range validates that numeric value is within range
func (v *Validator) Range(field string, value interface{}, min, max int64) {
	var num int64
	var err error

	switch val := value.(type) {
	case int:
		num = int64(val)
	case int64:
		num = val
	case string:
		if val != "" {
			num, err = strconv.ParseInt(val, 10, 64)
			if err != nil {
				v.AddError(field, "must be a valid number", value)
				return
			}
		} else {
			return
		}
	default:
		v.AddError(field, "must be a valid number", value)
		return
	}

	if num < min || num > max {
		v.AddError(field, fmt.Sprintf("must be between %d and %d", min, max), value)
	}
}

// Pattern validates against regex pattern
func (v *Validator) Pattern(field string, value string, pattern *regexp.Regexp, message string) {
	if value != "" && !pattern.MatchString(value) {
		v.AddError(field, message, value)
	}
}

// PasswordStrength validates password strength
func (v *Validator) PasswordStrength(field string, value string) {
	if value == "" {
		return
	}

	var hasLower, hasUpper, hasNumber, hasSpecial bool

	for _, char := range value {
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if len(value) < 8 {
		v.AddError(field, "must be at least 8 characters long", nil)
	}
	if !hasLower {
		v.AddError(field, "must contain at least one lowercase letter", nil)
	}
	if !hasUpper {
		v.AddError(field, "must contain at least one uppercase letter", nil)
	}
	if !hasNumber {
		v.AddError(field, "must contain at least one number", nil)
	}
	if !hasSpecial {
		v.AddError(field, "must contain at least one special character", nil)
	}
}

// Helper functions

// IsEmpty checks if a value is considered empty
func IsEmpty(value interface{}) bool {
	if value == nil {
		return true
	}

	switch val := value.(type) {
	case string:
		return strings.TrimSpace(val) == ""
	case []string:
		return len(val) == 0
	case map[string]interface{}:
		return len(val) == 0
	case []interface{}:
		return len(val) == 0
	case int, int64, float64:
		return false // Numbers are never empty, use Range for validation
	default:
		return fmt.Sprintf("%v", value) == ""
	}
}

// ValidateStruct validates a struct using validation tags (basic implementation)
func ValidateStruct(s interface{}) ValidationErrors {
	// This is a simplified implementation
	// In a full implementation, you would use reflection to parse validation tags
	// For now, return empty errors as this would require more complex tag parsing
	return ValidationErrors{}
}

// Standalone validation functions

// IsValidEmail validates email format
func IsValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsValidURL validates URL format
func IsValidURL(rawURL string) bool {
	_, err := url.Parse(rawURL)
	return err == nil
}

// IsValidPort validates port number
func IsValidPort(port interface{}) bool {
	var p int
	var err error

	switch val := port.(type) {
	case int:
		p = val
	case string:
		p, err = strconv.Atoi(val)
		if err != nil {
			return false
		}
	default:
		return false
	}

	return p >= 1 && p <= 65535
}

// IsValidIP validates IP address
func IsValidIP(ip string) bool {
	return net.ParseIP(ip) != nil
}

// IsValidDockerImage validates Docker image name
func IsValidDockerImage(image string) bool {
	if image == "" {
		return false
	}

	parts := strings.Split(image, ":")
	imagePart := parts[0]

	// Remove registry part if present
	if strings.Contains(imagePart, "/") && (strings.Contains(imagePart, ".") || strings.Contains(imagePart, ":")) {
		parts := strings.SplitN(imagePart, "/", 2)
		if len(parts) > 1 {
			imagePart = parts[1]
		}
	}

	return imageNameRegex.MatchString(imagePart)
}

// IsValidDockerTag validates Docker tag
func IsValidDockerTag(tag string) bool {
	return tag != "" && tagRegex.MatchString(tag)
}

// IsValidContainerName validates container name
func IsValidContainerName(name string) bool {
	return name != "" && containerNameRegex.MatchString(name) && len(name) <= 63
}

// IsValidCronExpression validates cron expression
func IsValidCronExpression(cron string) bool {
	return cron != "" && cronRegex.MatchString(cron)
}