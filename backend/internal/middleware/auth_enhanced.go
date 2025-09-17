package middleware

import (
	"docker-auto/pkg/security"
	"docker-auto/pkg/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthConfig represents authentication configuration
type AuthConfig struct {
	JWTManager          *security.EnhancedJWTManager
	RequireSecureHeaders bool
	AllowInsecureLocal   bool
	TokenRotationEnabled bool
	SessionValidation    bool
	IPValidation         bool
	UserAgentValidation  bool
	SecurityLevel        int // 1=low, 2=medium, 3=high
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig(jwtManager *security.EnhancedJWTManager) *AuthConfig {
	return &AuthConfig{
		JWTManager:          jwtManager,
		RequireSecureHeaders: true,
		AllowInsecureLocal:   false,
		TokenRotationEnabled: true,
		SessionValidation:    true,
		IPValidation:         true,
		UserAgentValidation:  true,
		SecurityLevel:        2,
	}
}

// EnhancedAuthMiddleware creates an enhanced authentication middleware
func EnhancedAuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract and validate security headers first
		if config.RequireSecureHeaders {
			if err := validateSecurityHeaders(c, config); err != nil {
				logrus.WithError(err).Warn("Security headers validation failed")
				if !config.AllowInsecureLocal || !isLocalRequest(c) {
					c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Security requirements not met"))
					c.Abort()
					return
				}
			}
		}

		// Extract token from header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authorization header is required"))
			c.Abort()
			return
		}

		token, err := extractBearerToken(authHeader)
		if err != nil {
			logrus.WithError(err).Warn("Failed to extract bearer token")
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Invalid authorization header format"))
			c.Abort()
			return
		}

		// Get client context
		clientIP := getClientIP(c)
		userAgent := c.GetHeader("User-Agent")

		// Validate token with enhanced security
		claims, err := config.JWTManager.ValidateToken(token, clientIP, userAgent)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"client_ip":  clientIP,
				"user_agent": userAgent[:min(len(userAgent), 50)],
				"path":       c.Request.URL.Path,
			}).Warn("Token validation failed")

			// Determine appropriate error response
			if strings.Contains(err.Error(), "blacklisted") {
				c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Token has been revoked"))
			} else if strings.Contains(err.Error(), "session") {
				c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Session invalid or expired"))
			} else {
				c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Invalid or expired token"))
			}
			c.Abort()
			return
		}

		// Additional security validations
		if err := performSecurityValidations(c, claims, config); err != nil {
			logrus.WithError(err).WithField("user_id", claims.UserID).Warn("Security validation failed")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, err.Error()))
			c.Abort()
			return
		}

		// Store enhanced user information in context
		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set("security_level", claims.SecurityLevel)

		// Check for token rotation requirement
		if config.TokenRotationEnabled && shouldRotateToken(claims) {
			c.Header("X-Token-Rotation-Required", "true")
			c.Header("X-Token-Age", time.Since(time.Unix(claims.IssuedAt, 0)).String())
		}

		// Log successful authentication
		logrus.WithFields(logrus.Fields{
			"user_id":        claims.UserID,
			"username":       claims.Username,
			"role":           claims.Role,
			"session_id":     claims.SessionID,
			"security_level": claims.SecurityLevel,
			"path":           c.Request.URL.Path,
			"method":         c.Request.Method,
			"client_ip":      clientIP,
		}).Debug("Enhanced authentication successful")

		c.Next()
	}
}

// OptionalEnhancedAuthMiddleware creates an optional enhanced authentication middleware
func OptionalEnhancedAuthMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		token, err := extractBearerToken(authHeader)
		if err != nil {
			c.Next()
			return
		}

		clientIP := getClientIP(c)
		userAgent := c.GetHeader("User-Agent")

		claims, err := config.JWTManager.ValidateToken(token, clientIP, userAgent)
		if err != nil {
			// Log but don't abort
			logrus.WithError(err).Debug("Optional token validation failed")
			c.Next()
			return
		}

		// Store user information if valid
		c.Set("user", claims)
		c.Set("user_id", claims.UserID)
		c.Set("session_id", claims.SessionID)
		c.Set("security_level", claims.SecurityLevel)

		c.Next()
	}
}

// RequireRole creates a middleware that requires specific roles
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetEnhancedUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		roleAllowed := false
		for _, role := range allowedRoles {
			if user.Role == role {
				roleAllowed = true
				break
			}
		}

		if !roleAllowed {
			logrus.WithFields(logrus.Fields{
				"user_id":       user.UserID,
				"user_role":     user.Role,
				"required_roles": allowedRoles,
			}).Warn("Insufficient role permissions")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "Insufficient permissions"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireSecurityLevel creates a middleware that requires minimum security level
func RequireSecurityLevel(minLevel int) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetEnhancedUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		if user.SecurityLevel < minLevel {
			logrus.WithFields(logrus.Fields{
				"user_id":              user.UserID,
				"current_level":        user.SecurityLevel,
				"required_level":       minLevel,
			}).Warn("Insufficient security level")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "Higher security level required"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireActiveStatus ensures user has active status
func RequireActiveStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetEnhancedUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		if user.Status != "active" {
			logrus.WithFields(logrus.Fields{
				"user_id": user.UserID,
				"status":  user.Status,
			}).Warn("Inactive user attempted access")
			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "Account is not active"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenRefreshMiddleware handles automatic token refresh
func TokenRefreshMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		refreshToken := c.GetHeader("X-Refresh-Token")
		if refreshToken == "" {
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Refresh token required"))
			c.Abort()
			return
		}

		clientIP := getClientIP(c)
		userAgent := c.GetHeader("User-Agent")

		newAccessToken, newRefreshToken, err := config.JWTManager.RefreshTokens(refreshToken, clientIP, userAgent)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"client_ip":  clientIP,
				"user_agent": userAgent[:min(len(userAgent), 50)],
			}).Warn("Token refresh failed")
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Token refresh failed"))
			c.Abort()
			return
		}

		// Return new tokens
		c.JSON(http.StatusOK, utils.SuccessResponseWithMessage(map[string]interface{}{
			"access_token":  newAccessToken,
			"refresh_token": newRefreshToken,
			"token_type":    "Bearer",
		}, "Tokens refreshed"))
	}
}

// LogoutMiddleware handles secure logout
func LogoutMiddleware(config *AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusOK, utils.SuccessResponseWithMessage(nil, "Logged out"))
			return
		}

		token, err := extractBearerToken(authHeader)
		if err != nil {
			c.JSON(http.StatusOK, utils.SuccessResponseWithMessage(nil, "Logged out"))
			return
		}

		// Revoke the token
		err = config.JWTManager.RevokeToken(token)
		if err != nil {
			logrus.WithError(err).Warn("Failed to revoke token during logout")
		}

		logrus.Info("User logged out successfully")
		c.JSON(http.StatusOK, utils.SuccessResponseWithMessage(nil, "Logged out successfully"))
	}
}

// Helper functions

// GetEnhancedUserFromContext retrieves enhanced user claims from context
func GetEnhancedUserFromContext(c *gin.Context) *security.EnhancedClaims {
	if claims, exists := c.Get("user"); exists {
		if userClaims, ok := claims.(*security.EnhancedClaims); ok {
			return userClaims
		}
	}
	return nil
}

// extractBearerToken extracts token from Bearer authorization header
func extractBearerToken(authHeader string) (string, error) {
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", fmt.Errorf("invalid authorization header format")
	}
	return strings.TrimPrefix(authHeader, bearerPrefix), nil
}

// getClientIP gets the real client IP address
func getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header first
	xff := c.GetHeader("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := c.GetHeader("X-Real-IP")
	if xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr
	return c.ClientIP()
}

// validateSecurityHeaders validates required security headers
func validateSecurityHeaders(c *gin.Context, config *AuthConfig) error {
	// Check for HTTPS in production
	if config.SecurityLevel >= 2 {
		if c.GetHeader("X-Forwarded-Proto") != "https" && c.Request.TLS == nil {
			if !config.AllowInsecureLocal || !isLocalRequest(c) {
				return fmt.Errorf("HTTPS required")
			}
		}
	}

	// Validate Content-Type for POST/PUT requests
	if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
		contentType := c.GetHeader("Content-Type")
		if contentType == "" {
			return fmt.Errorf("Content-Type header required")
		}
	}

	return nil
}

// isLocalRequest checks if request is from localhost
func isLocalRequest(c *gin.Context) bool {
	clientIP := getClientIP(c)
	return clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost"
}

// performSecurityValidations performs additional security validations
func performSecurityValidations(c *gin.Context, claims *security.EnhancedClaims, config *AuthConfig) error {
	// Validate user status
	if claims.Status != "active" {
		return fmt.Errorf("account is not active")
	}

	// Validate token type (should be access token)
	if claims.TokenType != "access" {
		return fmt.Errorf("invalid token type")
	}

	// Additional IP validation for high security
	if config.SecurityLevel >= 3 && config.IPValidation {
		currentIP := getClientIP(c)
		if claims.ClientIP != currentIP {
			return fmt.Errorf("IP address mismatch")
		}
	}

	// User agent validation for high security
	if config.SecurityLevel >= 3 && config.UserAgentValidation {
		currentUA := c.GetHeader("User-Agent")
		if claims.UserAgent != currentUA {
			return fmt.Errorf("user agent mismatch")
		}
	}

	return nil
}

// shouldRotateToken determines if token should be rotated
func shouldRotateToken(claims *security.EnhancedClaims) bool {
	issuedAt := time.Unix(claims.IssuedAt, 0)
	// Rotate if token is older than 5 minutes for high security
	if claims.SecurityLevel >= 3 {
		return time.Since(issuedAt) > 5*time.Minute
	}
	// Rotate if token is older than 15 minutes for medium security
	if claims.SecurityLevel >= 2 {
		return time.Since(issuedAt) > 15*time.Minute
	}
	// Rotate if token is older than 1 hour for low security
	return time.Since(issuedAt) > time.Hour
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}