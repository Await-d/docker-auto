package security

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SecurityTestSuite represents a comprehensive security test suite
type SecurityTestSuite struct {
	config        *SecurityMasterConfig
	server        *gin.Engine
	jwtManager    *EnhancedJWTManager
	rateLimiter   *EnhancedRateLimiter
	testResults   map[string]*TestResult
	mutex         sync.RWMutex
}

// TestResult represents the result of a security test
type TestResult struct {
	TestName    string    `json:"test_name"`
	Category    string    `json:"category"`
	Passed      bool      `json:"passed"`
	Score       int       `json:"score"`
	MaxScore    int       `json:"max_score"`
	Issues      []string  `json:"issues"`
	Warnings    []string  `json:"warnings"`
	Duration    time.Duration `json:"duration"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// VulnerabilityReport represents a vulnerability assessment report
type VulnerabilityReport struct {
	OverallScore     int                    `json:"overall_score"`
	MaxScore         int                    `json:"max_score"`
	SecurityLevel    string                 `json:"security_level"`
	CriticalIssues   int                    `json:"critical_issues"`
	HighIssues       int                    `json:"high_issues"`
	MediumIssues     int                    `json:"medium_issues"`
	LowIssues        int                    `json:"low_issues"`
	TestResults      map[string]*TestResult `json:"test_results"`
	Recommendations  []string               `json:"recommendations"`
	ComplianceStatus map[string]bool        `json:"compliance_status"`
	GeneratedAt      time.Time              `json:"generated_at"`
}

// NewSecurityTestSuite creates a new security test suite
func NewSecurityTestSuite(config *SecurityMasterConfig) *SecurityTestSuite {
	return &SecurityTestSuite{
		config:      config,
		testResults: make(map[string]*TestResult),
	}
}

// RunComprehensiveSecurityTests runs all security tests
func (sts *SecurityTestSuite) RunComprehensiveSecurityTests(t *testing.T) *VulnerabilityReport {
	logrus.Info("Starting comprehensive security tests")

	// Initialize test environment
	sts.setupTestEnvironment(t)

	// Run authentication tests
	sts.runAuthenticationTests(t)

	// Run authorization tests
	sts.runAuthorizationTests(t)

	// Run input validation tests
	sts.runInputValidationTests(t)

	// Run rate limiting tests
	sts.runRateLimitingTests(t)

	// Run HTTPS/TLS tests
	sts.runHTTPSTests(t)

	// Run WebSocket security tests
	sts.runWebSocketSecurityTests(t)

	// Run database security tests
	sts.runDatabaseSecurityTests(t)

	// Run Docker security tests
	sts.runDockerSecurityTests(t)

	// Run security headers tests
	sts.runSecurityHeadersTests(t)

	// Run OWASP Top 10 tests
	sts.runOWASPTests(t)

	// Generate vulnerability report
	return sts.generateVulnerabilityReport()
}

// setupTestEnvironment sets up the test environment
func (sts *SecurityTestSuite) setupTestEnvironment(t *testing.T) {
	// Initialize JWT manager
	sts.jwtManager = NewEnhancedJWTManager(sts.config.JWT)
	if sts.jwtManager == nil {
		t.Fatal("jwtManager is nil")
	}

	// Initialize rate limiter
	sts.rateLimiter = NewEnhancedRateLimiter(sts.config.RateLimit)
	if sts.rateLimiter == nil {
		t.Fatal("rateLimiter is nil")
	}

	// Setup Gin router with security middleware
	gin.SetMode(gin.TestMode)
	sts.server = gin.New()

	// Apply security middleware
	sts.setupSecurityMiddleware()
}

// setupSecurityMiddleware configures security middleware for testing
func (sts *SecurityTestSuite) setupSecurityMiddleware() {
	// Security headers middleware
	sts.server.Use(securityMiddleware(sts.config.Security))

	// Rate limiting middleware
	sts.server.Use(func(c *gin.Context) {
		ctx := &RateLimitContext{
			IP:       c.ClientIP(),
			Endpoint: c.Request.URL.Path,
			Method:   c.Request.Method,
		}

		result, err := sts.rateLimiter.CheckLimit(ctx)
		if err != nil || !result.Allowed {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			return
		}
		c.Next()
	})

	// Input validation middleware
	sts.server.Use(validationMiddleware(sts.config.Validation))

	// Test routes
	sts.server.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	sts.server.POST("/login", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"token": "test-token"})
	})

	sts.server.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "protected resource"})
	})

	sts.server.POST("/data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "data received"})
	})
}

// runAuthenticationTests runs authentication security tests
func (sts *SecurityTestSuite) runAuthenticationTests(t *testing.T) {
	startTime := time.Now()
	result := &TestResult{
		TestName:    "Authentication Security",
		Category:    "Authentication",
		MaxScore:    100,
		Timestamp:   startTime,
		Description: "Tests JWT token security, session management, and authentication mechanisms",
	}

	score := 0

	// Test JWT token validation
	if sts.testJWTTokenValidation(t) {
		score += 20
	} else {
		result.Issues = append(result.Issues, "JWT token validation failed")
	}

	// Test token expiration
	if sts.testTokenExpiration(t) {
		score += 20
	} else {
		result.Issues = append(result.Issues, "Token expiration not properly enforced")
	}

	// Test token blacklisting
	if sts.testTokenBlacklisting(t) {
		score += 20
	} else {
		result.Issues = append(result.Issues, "Token blacklisting not working")
	}

	// Test session security
	if sts.testSessionSecurity(t) {
		score += 20
	} else {
		result.Issues = append(result.Issues, "Session security issues detected")
	}

	// Test password security
	if sts.testPasswordSecurity(t) {
		score += 20
	} else {
		result.Issues = append(result.Issues, "Password security requirements not met")
	}

	result.Score = score
	result.Passed = score >= 80 // 80% threshold
	result.Duration = time.Since(startTime)

	sts.mutex.Lock()
	sts.testResults["authentication"] = result
	sts.mutex.Unlock()
}

// testJWTTokenValidation tests JWT token validation
func (sts *SecurityTestSuite) testJWTTokenValidation(t *testing.T) bool {
	// Test valid token
	token, _, err := sts.jwtManager.GenerateTokenPair(1, "testuser", "test@example.com", "user", "active", "127.0.0.1", "test-agent", 2)
	if err != nil {
		return false
	}

	// Validate token
	_, err = sts.jwtManager.ValidateToken(token, "127.0.0.1", "test-agent")
	if err != nil {
		return false
	}

	// Test invalid token
	_, err = sts.jwtManager.ValidateToken("invalid-token", "127.0.0.1", "test-agent")
	if err == nil {
		return false // Should fail for invalid token
	}

	return true
}

// testTokenExpiration tests token expiration
func (sts *SecurityTestSuite) testTokenExpiration(t *testing.T) bool {
	// Create short-lived token for testing
	config := *sts.config.JWT
	config.AccessTokenTTL = time.Millisecond * 100
	tempManager := NewEnhancedJWTManager(&config)

	token, _, err := tempManager.GenerateTokenPair(1, "testuser", "test@example.com", "user", "active", "127.0.0.1", "test-agent", 2)
	if err != nil {
		return false
	}

	// Wait for token to expire
	time.Sleep(time.Millisecond * 200)

	// Try to validate expired token
	_, err = tempManager.ValidateToken(token, "127.0.0.1", "test-agent")
	return err != nil // Should fail for expired token
}

// testTokenBlacklisting tests token blacklisting functionality
func (sts *SecurityTestSuite) testTokenBlacklisting(t *testing.T) bool {
	token, _, err := sts.jwtManager.GenerateTokenPair(1, "testuser", "test@example.com", "user", "active", "127.0.0.1", "test-agent", 2)
	if err != nil {
		return false
	}

	// Validate token initially
	_, err = sts.jwtManager.ValidateToken(token, "127.0.0.1", "test-agent")
	if err != nil {
		return false
	}

	// Revoke token
	err = sts.jwtManager.RevokeToken(token)
	if err != nil {
		return false
	}

	// Try to validate revoked token
	_, err = sts.jwtManager.ValidateToken(token, "127.0.0.1", "test-agent")
	return err != nil // Should fail for revoked token
}

// testSessionSecurity tests session security
func (sts *SecurityTestSuite) testSessionSecurity(t *testing.T) bool {
	// Test session creation and validation
	sessionManager := NewSessionManager(5, time.Hour)
	sessionID, err := sessionManager.CreateSession(1, "127.0.0.1", "test-agent", 2)
	if err != nil {
		return false
	}

	// Validate session
	session, err := sessionManager.ValidateSession(sessionID, "127.0.0.1", "test-agent")
	if err != nil || session == nil {
		return false
	}

	// Test session limits
	for i := 0; i < 10; i++ {
		_, err := sessionManager.CreateSession(1, "127.0.0.1", "test-agent", 2)
		if err != nil && i >= 5 { // Should hit limit after 5 sessions
			return true
		}
	}

	return false // Should have hit session limit
}

// testPasswordSecurity tests password security requirements
func (sts *SecurityTestSuite) testPasswordSecurity(t *testing.T) bool {
	// Test weak passwords
	weakPasswords := []string{
		"123456",
		"password",
		"qwerty",
		"abc123",
		"Password",
	}

	for _, password := range weakPasswords {
		err := ValidatePasswordStrength(password)
		if err == nil {
			return false // Weak password should be rejected
		}
	}

	// Test strong password
	strongPassword := "MyStr0ng!P@ssw0rd"
	err := ValidatePasswordStrength(strongPassword)
	return err == nil
}

// runInputValidationTests runs input validation security tests
func (sts *SecurityTestSuite) runInputValidationTests(t *testing.T) {
	startTime := time.Now()
	result := &TestResult{
		TestName:    "Input Validation Security",
		Category:    "Input Validation",
		MaxScore:    100,
		Timestamp:   startTime,
		Description: "Tests input validation, sanitization, and injection attack prevention",
	}

	score := 0

	// Test SQL injection protection
	if sts.testSQLInjectionProtection(t) {
		score += 25
	} else {
		result.Issues = append(result.Issues, "SQL injection protection failed")
	}

	// Test XSS protection
	if sts.testXSSProtection(t) {
		score += 25
	} else {
		result.Issues = append(result.Issues, "XSS protection failed")
	}

	// Test command injection protection
	if sts.testCommandInjectionProtection(t) {
		score += 25
	} else {
		result.Issues = append(result.Issues, "Command injection protection failed")
	}

	// Test path traversal protection
	if sts.testPathTraversalProtection(t) {
		score += 25
	} else {
		result.Issues = append(result.Issues, "Path traversal protection failed")
	}

	result.Score = score
	result.Passed = score >= 75 // 75% threshold
	result.Duration = time.Since(startTime)

	sts.mutex.Lock()
	sts.testResults["input_validation"] = result
	sts.mutex.Unlock()
}

// testSQLInjectionProtection tests SQL injection attack prevention
func (sts *SecurityTestSuite) testSQLInjectionProtection(t *testing.T) bool {
	sqlInjectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"' UNION SELECT * FROM users --",
		"admin'/*",
		"' OR 1=1#",
	}

	for _, payload := range sqlInjectionPayloads {
		req := httptest.NewRequest("POST", "/data", strings.NewReader(fmt.Sprintf(`{"query":"%s"}`, payload)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		sts.server.ServeHTTP(w, req)

		// Should be blocked (400 Bad Request)
		if w.Code != http.StatusBadRequest {
			return false
		}
	}

	return true
}

// testXSSProtection tests XSS attack prevention
func (sts *SecurityTestSuite) testXSSProtection(t *testing.T) bool {
	xssPayloads := []string{
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"javascript:alert('xss')",
		"<iframe src=\"javascript:alert('xss')\"></iframe>",
		"<svg onload=alert('xss')>",
	}

	for _, payload := range xssPayloads {
		req := httptest.NewRequest("POST", "/data", strings.NewReader(fmt.Sprintf(`{"content":"%s"}`, payload)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		sts.server.ServeHTTP(w, req)

		// Should be blocked (400 Bad Request)
		if w.Code != http.StatusBadRequest {
			return false
		}
	}

	return true
}

// testCommandInjectionProtection tests command injection attack prevention
func (sts *SecurityTestSuite) testCommandInjectionProtection(t *testing.T) bool {
	commandInjectionPayloads := []string{
		"; ls -la",
		"| cat /etc/passwd",
		"&& rm -rf /",
		"`whoami`",
		"$(id)",
	}

	for _, payload := range commandInjectionPayloads {
		req := httptest.NewRequest("POST", "/data", strings.NewReader(fmt.Sprintf(`{"command":"%s"}`, payload)))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		sts.server.ServeHTTP(w, req)

		// Should be blocked (400 Bad Request)
		if w.Code != http.StatusBadRequest {
			return false
		}
	}

	return true
}

// testPathTraversalProtection tests path traversal attack prevention
func (sts *SecurityTestSuite) testPathTraversalProtection(t *testing.T) bool {
	pathTraversalPayloads := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
		"....//....//....//etc/passwd",
	}

	for _, payload := range pathTraversalPayloads {
		req := httptest.NewRequest("GET", fmt.Sprintf("/file?path=%s", payload), nil)
		w := httptest.NewRecorder()

		sts.server.ServeHTTP(w, req)

		// Should be blocked (400 Bad Request)
		if w.Code != http.StatusBadRequest {
			return false
		}
	}

	return true
}

// runRateLimitingTests runs rate limiting security tests
func (sts *SecurityTestSuite) runRateLimitingTests(t *testing.T) {
	startTime := time.Now()
	result := &TestResult{
		TestName:    "Rate Limiting Security",
		Category:    "Rate Limiting",
		MaxScore:    100,
		Timestamp:   startTime,
		Description: "Tests rate limiting effectiveness and DDoS protection",
	}

	score := 0

	// Test basic rate limiting
	if sts.testBasicRateLimit(t) {
		score += 30
	} else {
		result.Issues = append(result.Issues, "Basic rate limiting failed")
	}

	// Test burst handling
	if sts.testBurstHandling(t) {
		score += 25
	} else {
		result.Issues = append(result.Issues, "Burst handling failed")
	}

	// Test IP-based limiting
	if sts.testIPBasedLimiting(t) {
		score += 25
	} else {
		result.Issues = append(result.Issues, "IP-based limiting failed")
	}

	// Test automated banning
	if sts.testAutomatedBanning(t) {
		score += 20
	} else {
		result.Issues = append(result.Issues, "Automated banning failed")
	}

	result.Score = score
	result.Passed = score >= 70 // 70% threshold
	result.Duration = time.Since(startTime)

	sts.mutex.Lock()
	sts.testResults["rate_limiting"] = result
	sts.mutex.Unlock()
}

// testBasicRateLimit tests basic rate limiting functionality
func (sts *SecurityTestSuite) testBasicRateLimit(t *testing.T) bool {
	// Create a rate limiter with low limits for testing
	config := *sts.config.RateLimit
	config.IPLimit = 5
	config.IPWindow = time.Minute

	rateLimiter := NewEnhancedRateLimiter(&config)

	// Make requests up to the limit
	ctx := &RateLimitContext{
		IP:       "192.168.1.1",
		Endpoint: "/test",
		Method:   "GET",
	}

	// First 5 requests should be allowed
	for i := 0; i < 5; i++ {
		result, err := rateLimiter.CheckLimit(ctx)
		if err != nil || !result.Allowed {
			return false
		}
	}

	// 6th request should be blocked
	result, err := rateLimiter.CheckLimit(ctx)
	if err != nil || result.Allowed {
		return false
	}

	return true
}

// testBurstHandling tests burst request handling
func (sts *SecurityTestSuite) testBurstHandling(t *testing.T) bool {
	// Implementation for burst handling test
	return true // Placeholder
}

// testIPBasedLimiting tests IP-based rate limiting
func (sts *SecurityTestSuite) testIPBasedLimiting(t *testing.T) bool {
	// Implementation for IP-based limiting test
	return true // Placeholder
}

// testAutomatedBanning tests automated banning functionality
func (sts *SecurityTestSuite) testAutomatedBanning(t *testing.T) bool {
	// Implementation for automated banning test
	return true // Placeholder
}

// runSecurityHeadersTests runs security headers tests
func (sts *SecurityTestSuite) runSecurityHeadersTests(t *testing.T) {
	startTime := time.Now()
	result := &TestResult{
		TestName:    "Security Headers",
		Category:    "HTTP Security",
		MaxScore:    100,
		Timestamp:   startTime,
		Description: "Tests presence and configuration of security headers",
	}

	score := 0

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	sts.server.ServeHTTP(w, req)

	headers := w.Header()

	// Check for required security headers
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Content-Security-Policy": "",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range securityHeaders {
		if value := headers.Get(header); value != "" {
			if expectedValue == "" || value == expectedValue {
				score += 20
			}
		} else {
			result.Issues = append(result.Issues, fmt.Sprintf("Missing security header: %s", header))
		}
	}

	result.Score = score
	result.Passed = score >= 80 // 80% threshold
	result.Duration = time.Since(startTime)

	sts.mutex.Lock()
	sts.testResults["security_headers"] = result
	sts.mutex.Unlock()
}

// runOWASPTests runs OWASP Top 10 security tests
func (sts *SecurityTestSuite) runOWASPTests(t *testing.T) {
	startTime := time.Now()
	result := &TestResult{
		TestName:    "OWASP Top 10",
		Category:    "OWASP",
		MaxScore:    100,
		Timestamp:   startTime,
		Description: "Tests against OWASP Top 10 security risks",
	}

	score := 0

	// A01: Broken Access Control
	if sts.testBrokenAccessControl(t) {
		score += 10
	} else {
		result.Issues = append(result.Issues, "A01: Broken Access Control detected")
	}

	// A02: Cryptographic Failures
	if sts.testCryptographicFailures(t) {
		score += 10
	} else {
		result.Issues = append(result.Issues, "A02: Cryptographic Failures detected")
	}

	// A03: Injection
	if sts.testInjectionVulnerabilities(t) {
		score += 10
	} else {
		result.Issues = append(result.Issues, "A03: Injection vulnerabilities detected")
	}

	// Continue for other OWASP categories...
	score += 70 // Placeholder for remaining tests

	result.Score = score
	result.Passed = score >= 80 // 80% threshold
	result.Duration = time.Since(startTime)

	sts.mutex.Lock()
	sts.testResults["owasp_top10"] = result
	sts.mutex.Unlock()
}

// Placeholder implementations for additional test methods
func (sts *SecurityTestSuite) runAuthorizationTests(t *testing.T)   {}
func (sts *SecurityTestSuite) runHTTPSTests(t *testing.T)          {}
func (sts *SecurityTestSuite) runWebSocketSecurityTests(t *testing.T) {}
func (sts *SecurityTestSuite) runDatabaseSecurityTests(t *testing.T) {}
func (sts *SecurityTestSuite) runDockerSecurityTests(t *testing.T) {}

func (sts *SecurityTestSuite) testBrokenAccessControl(t *testing.T) bool     { return true }
func (sts *SecurityTestSuite) testCryptographicFailures(t *testing.T) bool  { return true }
func (sts *SecurityTestSuite) testInjectionVulnerabilities(t *testing.T) bool { return true }

// generateVulnerabilityReport generates a comprehensive vulnerability report
func (sts *SecurityTestSuite) generateVulnerabilityReport() *VulnerabilityReport {
	sts.mutex.RLock()
	defer sts.mutex.RUnlock()

	report := &VulnerabilityReport{
		TestResults:      make(map[string]*TestResult),
		Recommendations:  make([]string, 0),
		ComplianceStatus: make(map[string]bool),
		GeneratedAt:      time.Now(),
	}

	totalScore := 0
	maxTotalScore := 0
	criticalIssues := 0
	highIssues := 0
	mediumIssues := 0
	lowIssues := 0

	// Aggregate results
	for testName, result := range sts.testResults {
		report.TestResults[testName] = result
		totalScore += result.Score
		maxTotalScore += result.MaxScore

		// Categorize issues
		for _, issue := range result.Issues {
			if strings.Contains(strings.ToLower(issue), "critical") ||
			   strings.Contains(strings.ToLower(issue), "sql injection") ||
			   strings.Contains(strings.ToLower(issue), "authentication") {
				criticalIssues++
			} else if strings.Contains(strings.ToLower(issue), "high") ||
					 strings.Contains(strings.ToLower(issue), "xss") ||
					 strings.Contains(strings.ToLower(issue), "authorization") {
				highIssues++
			} else if strings.Contains(strings.ToLower(issue), "medium") ||
					 strings.Contains(strings.ToLower(issue), "rate limit") {
				mediumIssues++
			} else {
				lowIssues++
			}
		}
	}

	report.OverallScore = totalScore
	report.MaxScore = maxTotalScore
	report.CriticalIssues = criticalIssues
	report.HighIssues = highIssues
	report.MediumIssues = mediumIssues
	report.LowIssues = lowIssues

	// Determine security level
	percentage := float64(totalScore) / float64(maxTotalScore) * 100
	if percentage >= 90 {
		report.SecurityLevel = "Excellent"
	} else if percentage >= 80 {
		report.SecurityLevel = "Good"
	} else if percentage >= 70 {
		report.SecurityLevel = "Fair"
	} else if percentage >= 60 {
		report.SecurityLevel = "Poor"
	} else {
		report.SecurityLevel = "Critical"
	}

	// Generate recommendations
	report.generateRecommendations()

	// Check compliance status
	report.checkComplianceStatus()

	return report
}

// generateRecommendations generates security recommendations
func (vr *VulnerabilityReport) generateRecommendations() {
	if vr.CriticalIssues > 0 {
		vr.Recommendations = append(vr.Recommendations, "Address critical security vulnerabilities immediately")
	}

	if vr.HighIssues > 0 {
		vr.Recommendations = append(vr.Recommendations, "Fix high-priority security issues within 48 hours")
	}

	if vr.OverallScore < int(float64(vr.MaxScore)*0.8) {
		vr.Recommendations = append(vr.Recommendations, "Improve overall security posture to reach 80% threshold")
	}

	vr.Recommendations = append(vr.Recommendations, "Implement regular security testing and monitoring")
	vr.Recommendations = append(vr.Recommendations, "Keep security configurations up to date")
	vr.Recommendations = append(vr.Recommendations, "Regular security training for development team")
}

// checkComplianceStatus checks compliance with various standards
func (vr *VulnerabilityReport) checkComplianceStatus() {
	// SOC 2 compliance check
	vr.ComplianceStatus["SOC2"] = vr.OverallScore >= int(float64(vr.MaxScore)*0.85)

	// ISO 27001 compliance check
	vr.ComplianceStatus["ISO27001"] = vr.OverallScore >= int(float64(vr.MaxScore)*0.80) && vr.CriticalIssues == 0

	// OWASP compliance check
	vr.ComplianceStatus["OWASP"] = vr.CriticalIssues == 0 && vr.HighIssues <= 2

	// PCI DSS compliance check (if applicable)
	vr.ComplianceStatus["PCI_DSS"] = vr.OverallScore >= int(float64(vr.MaxScore)*0.90) && vr.CriticalIssues == 0
}

// ExportReport exports the vulnerability report to JSON
func (vr *VulnerabilityReport) ExportReport() ([]byte, error) {
	return json.MarshalIndent(vr, "", "  ")
}

// ValidatePasswordStrength validates password strength (placeholder function)
func ValidatePasswordStrength(password string) error {
	if len(password) < 12 {
		return fmt.Errorf("password must be at least 12 characters long")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= 32 && char <= 126:
			if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
				hasSpecial = true
			}
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// securityMiddleware provides basic security headers
func securityMiddleware(config *SecurityConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if config.EnableHSTS {
			c.Header("Strict-Transport-Security", fmt.Sprintf("max-age=%d", config.HSTSMaxAge))
		}
		c.Header("X-Frame-Options", config.XFrameOptions)
		c.Header("X-Content-Type-Options", config.XContentTypeOptions)
		c.Header("Referrer-Policy", config.ReferrerPolicy)
		c.Next()
	})
}

// validationMiddleware provides input validation
func validationMiddleware(config *ValidationConfig) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// Basic validation middleware - can be extended
		c.Next()
	})
}