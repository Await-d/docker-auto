package utils

import (
	"fmt"
	"math"
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

// PasswordStrength validates password strength with comprehensive security checks
func (v *Validator) PasswordStrength(field string, value string) {
	if value == "" {
		return
	}

	// Basic character type validation
	var hasLower, hasUpper, hasNumber, hasSpecial bool
	var consecutiveChars, repeatedChars int
	var prevChar rune

	for i, char := range value {
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

		// Check for consecutive characters
		if i > 0 && char == prevChar+1 {
			consecutiveChars++
		}

		// Check for repeated characters
		if i > 0 && char == prevChar {
			repeatedChars++
		}

		prevChar = char
	}

	// Length validation
	if len(value) < 12 {
		v.AddError(field, "must be at least 12 characters long for strong security", nil)
	} else if len(value) < 8 {
		v.AddError(field, "must be at least 8 characters long", nil)
	}

	// Character type requirements
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
		v.AddError(field, "must contain at least one special character (!@#$%^&*)", nil)
	}

	// Pattern checks
	if consecutiveChars > 2 {
		v.AddError(field, "contains too many consecutive characters (abc, 123, etc.)", nil)
	}

	if repeatedChars > 2 {
		v.AddError(field, "contains too many repeated characters", nil)
	}

	// Common weak patterns
	if v.containsWeakPatterns(value) {
		v.AddError(field, "contains common weak patterns", nil)
	}

	// Dictionary word check
	if v.containsCommonWords(value) {
		v.AddError(field, "contains common dictionary words", nil)
	}

	// Keyboard pattern check
	if v.containsKeyboardPatterns(value) {
		v.AddError(field, "contains keyboard patterns (qwerty, asdf, etc.)", nil)
	}

	// Personal information patterns
	if v.containsPersonalInfoPatterns(value) {
		v.AddError(field, "should not contain personal information patterns", nil)
	}
}

// containsWeakPatterns checks for common weak password patterns
func (v *Validator) containsWeakPatterns(password string) bool {
	password = strings.ToLower(password)

	// Common weak patterns
	weakPatterns := []string{
		"password", "123456", "qwerty", "admin", "root", "user",
		"login", "welcome", "guest", "test", "demo", "temp",
		"default", "secret", "pass", "pwd", "letmein",
	}

	for _, pattern := range weakPatterns {
		if strings.Contains(password, pattern) {
			return true
		}
	}

	// Check for simple number sequences
	sequences := []string{
		"012", "123", "234", "345", "456", "567", "678", "789",
		"987", "876", "765", "654", "543", "432", "321", "210",
	}

	for _, seq := range sequences {
		if strings.Contains(password, seq) {
			return true
		}
	}

	return false
}

// containsCommonWords checks for common dictionary words
func (v *Validator) containsCommonWords(password string) bool {
	password = strings.ToLower(password)

	// Most common English words that appear in passwords
	commonWords := []string{
		"the", "and", "for", "are", "but", "not", "you", "all",
		"can", "had", "her", "was", "one", "our", "out", "day",
		"get", "has", "him", "his", "how", "man", "new", "now",
		"old", "see", "two", "way", "who", "boy", "did", "its",
		"let", "put", "say", "she", "too", "use", "love", "good",
		"make", "time", "year", "work", "first", "right", "think",
		"house", "world", "school", "family", "company", "system",
	}

	for _, word := range commonWords {
		if len(word) >= 3 && strings.Contains(password, word) {
			return true
		}
	}

	return false
}

// containsKeyboardPatterns checks for keyboard patterns
func (v *Validator) containsKeyboardPatterns(password string) bool {
	password = strings.ToLower(password)

	// Common keyboard patterns
	keyboardPatterns := []string{
		"qwerty", "qwertyui", "asdf", "asdfgh", "zxcv", "zxcvbn",
		"qazwsx", "wsxedc", "rfvtgb", "yhnujm", "ikol", "plmokn",
		"qweqwe", "asdasd", "zxczxc", "poipoi", "mnbmnb",
		"1qaz2wsx", "qweasd", "asdqwe", "zxcasd", "qwezxc",
	}

	for _, pattern := range keyboardPatterns {
		if strings.Contains(password, pattern) {
			return true
		}
	}

	// Check for simple left-right patterns
	leftRightPatterns := []string{
		"aqsw", "swde", "derf", "frtg", "gtyk", "tyui", "yuio",
		"uiop", "plko", "lkjh", "jhgf", "hgfd", "gfds", "fdsa",
	}

	for _, pattern := range leftRightPatterns {
		if strings.Contains(password, pattern) {
			return true
		}
	}

	return false
}

// containsPersonalInfoPatterns checks for personal information patterns
func (v *Validator) containsPersonalInfoPatterns(password string) bool {
	password = strings.ToLower(password)

	// Common personal info patterns
	personalPatterns := []string{
		"name", "birth", "age", "phone", "email", "address",
		"city", "state", "country", "zip", "postal", "ssn",
		"social", "security", "license", "card", "bank",
		"account", "mother", "father", "spouse", "child",
		"pet", "dog", "cat", "car", "house", "street",
	}

	for _, pattern := range personalPatterns {
		if strings.Contains(password, pattern) {
			return true
		}
	}

	// Check for date patterns (YYYY, MMDD, DDMM)
	datePatterns := []string{
		"1980", "1981", "1982", "1983", "1984", "1985", "1986", "1987", "1988", "1989",
		"1990", "1991", "1992", "1993", "1994", "1995", "1996", "1997", "1998", "1999",
		"2000", "2001", "2002", "2003", "2004", "2005", "2006", "2007", "2008", "2009",
		"2010", "2011", "2012", "2013", "2014", "2015", "2016", "2017", "2018", "2019",
		"2020", "2021", "2022", "2023", "2024",
		"0101", "0201", "0301", "0401", "0501", "0601", "0701", "0801", "0901", "1001", "1101", "1201",
		"1231", "1225", "0704", "1031", "0214", "0317", "0401", "1111", "1212",
	}

	for _, pattern := range datePatterns {
		if strings.Contains(password, pattern) {
			return true
		}
	}

	return false
}

// AdvancedPasswordStrength provides detailed password strength analysis
type PasswordStrength struct {
	Score       int      `json:"score"`        // 0-100
	Level       string   `json:"level"`        // Weak, Fair, Good, Strong, Very Strong
	Issues      []string `json:"issues"`       // List of issues found
	Suggestions []string `json:"suggestions"`  // Suggestions for improvement
	Entropy     float64  `json:"entropy"`      // Estimated entropy bits
}

// AnalyzePasswordStrength provides comprehensive password analysis
func AnalyzePasswordStrength(password string) *PasswordStrength {
	if password == "" {
		return &PasswordStrength{
			Score:   0,
			Level:   "Very Weak",
			Issues:  []string{"Password is empty"},
			Suggestions: []string{"Create a password with at least 12 characters"},
		}
	}

	analysis := &PasswordStrength{
		Issues:      []string{},
		Suggestions: []string{},
	}

	score := 0

	// Length scoring
	length := len(password)
	if length >= 12 {
		score += 25
	} else if length >= 8 {
		score += 15
	} else {
		analysis.Issues = append(analysis.Issues, "Password is too short")
		analysis.Suggestions = append(analysis.Suggestions, "Use at least 12 characters for better security")
	}

	// Character variety scoring
	var hasLower, hasUpper, hasNumber, hasSpecial bool
	charTypes := 0

	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			if !hasLower {
				hasLower = true
				charTypes++
				score += 10
			}
		case unicode.IsUpper(char):
			if !hasUpper {
				hasUpper = true
				charTypes++
				score += 10
			}
		case unicode.IsNumber(char):
			if !hasNumber {
				hasNumber = true
				charTypes++
				score += 10
			}
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			if !hasSpecial {
				hasSpecial = true
				charTypes++
				score += 15
			}
		}
	}

	// Check for missing character types
	if !hasLower {
		analysis.Issues = append(analysis.Issues, "Missing lowercase letters")
		analysis.Suggestions = append(analysis.Suggestions, "Add lowercase letters (a-z)")
	}
	if !hasUpper {
		analysis.Issues = append(analysis.Issues, "Missing uppercase letters")
		analysis.Suggestions = append(analysis.Suggestions, "Add uppercase letters (A-Z)")
	}
	if !hasNumber {
		analysis.Issues = append(analysis.Issues, "Missing numbers")
		analysis.Suggestions = append(analysis.Suggestions, "Add numbers (0-9)")
	}
	if !hasSpecial {
		analysis.Issues = append(analysis.Issues, "Missing special characters")
		analysis.Suggestions = append(analysis.Suggestions, "Add special characters (!@#$%^&*)")
	}

	// Pattern checks
	validator := NewValidator()
	if validator.containsWeakPatterns(password) {
		score -= 15
		analysis.Issues = append(analysis.Issues, "Contains common weak patterns")
		analysis.Suggestions = append(analysis.Suggestions, "Avoid common words and sequences")
	}

	if validator.containsKeyboardPatterns(password) {
		score -= 10
		analysis.Issues = append(analysis.Issues, "Contains keyboard patterns")
		analysis.Suggestions = append(analysis.Suggestions, "Avoid keyboard sequences like 'qwerty'")
	}

	if validator.containsPersonalInfoPatterns(password) {
		score -= 10
		analysis.Issues = append(analysis.Issues, "May contain personal information")
		analysis.Suggestions = append(analysis.Suggestions, "Avoid dates, names, and personal information")
	}

	// Calculate entropy (simplified estimation)
	analysis.Entropy = calculatePasswordEntropy(password)
	if analysis.Entropy > 60 {
		score += 10
	} else if analysis.Entropy > 40 {
		score += 5
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	analysis.Score = score

	// Determine strength level
	switch {
	case score >= 90:
		analysis.Level = "Very Strong"
	case score >= 70:
		analysis.Level = "Strong"
	case score >= 50:
		analysis.Level = "Good"
	case score >= 30:
		analysis.Level = "Fair"
	default:
		analysis.Level = "Weak"
	}

	// Add general suggestions if needed
	if score < 70 {
		if len(analysis.Suggestions) == 0 {
			analysis.Suggestions = append(analysis.Suggestions, "Use a longer password with mixed character types")
		}
		analysis.Suggestions = append(analysis.Suggestions, "Consider using a passphrase with random words")
	}

	return analysis
}

// calculatePasswordEntropy estimates password entropy in bits
func calculatePasswordEntropy(password string) float64 {
	if len(password) == 0 {
		return 0
	}

	// Character set size estimation
	var charsetSize float64 = 0

	hasLower := false
	hasUpper := false
	hasDigits := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasDigits = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if hasLower {
		charsetSize += 26
	}
	if hasUpper {
		charsetSize += 26
	}
	if hasDigits {
		charsetSize += 10
	}
	if hasSpecial {
		charsetSize += 32 // Approximate number of special characters
	}

	if charsetSize == 0 {
		return 0
	}

	// Entropy = log2(charset_size^length)
	// Simplified: length * log2(charset_size)
	return float64(len(password)) * math.Log2(charsetSize)
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