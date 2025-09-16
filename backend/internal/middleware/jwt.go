package middleware

import (
	"docker-auto/pkg/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	// ContextUserKey is the key used to store user claims in context
	ContextUserKey = "user"

	// ContextUserIDKey is the key used to store user ID in context
	ContextUserIDKey = "user_id"

	// AuthorizationHeaderKey is the key for authorization header
	AuthorizationHeaderKey = "Authorization"
)

// JWTAuthMiddleware creates a JWT authentication middleware
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from header
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Authorization header is required"))
			c.Abort()
			return
		}

		token, err := extractTokenFromHeader(authHeader)
		if err != nil {
			logrus.WithError(err).Warn("Failed to extract token from header")
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Invalid authorization header format"))
			c.Abort()
			return
		}

		// Validate token
		claims, err := utils.ValidateJWT(token, secret)
		if err != nil {
			logrus.WithError(err).WithField("token", token[:min(len(token), 20)]).Warn("Token validation failed")
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Invalid or expired token"))
			c.Abort()
			return
		}

		// Store user information in context
		c.Set(ContextUserKey, claims)
		c.Set(ContextUserIDKey, claims.UserID)

		logrus.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
			"role":     claims.Role,
			"path":     c.Request.URL.Path,
			"method":   c.Request.Method,
		}).Debug("JWT authentication successful")

		c.Next()
	}
}

// JWTAuthOptionalMiddleware creates an optional JWT authentication middleware
// It validates the token if present but doesn't abort the request if missing
func JWTAuthOptionalMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Next()
			return
		}

		token, err := extractTokenFromHeader(authHeader)
		if err != nil {
			logrus.WithError(err).Debug("Failed to extract token from header")
			c.Next()
			return
		}

		// Validate token
		claims, err := utils.ValidateJWT(token, secret)
		if err != nil {
			logrus.WithError(err).Debug("Token validation failed")
			c.Next()
			return
		}

		// Store user information in context
		c.Set(ContextUserKey, claims)
		c.Set(ContextUserIDKey, claims.UserID)

		logrus.WithFields(logrus.Fields{
			"user_id":  claims.UserID,
			"username": claims.Username,
		}).Debug("Optional JWT authentication successful")

		c.Next()
	}
}

// extractTokenFromHeader extracts the JWT token from the Authorization header
func extractTokenFromHeader(authHeader string) (string, error) {
	return utils.ExtractTokenFromHeader(authHeader)
}

// getUserFromContext retrieves user claims from the gin context
func getUserFromContext(c *gin.Context) *utils.Claims {
	if claims, exists := c.Get(ContextUserKey); exists {
		if userClaims, ok := claims.(*utils.Claims); ok {
			return userClaims
		}
	}
	return nil
}

// GetUserFromContext retrieves user claims from the gin context (exported)
func GetUserFromContext(c *gin.Context) *utils.Claims {
	return getUserFromContext(c)
}

// GetUserIDFromContext retrieves user ID from the gin context
func GetUserIDFromContext(c *gin.Context) (int64, bool) {
	if userID, exists := c.Get(ContextUserIDKey); exists {
		if id, ok := userID.(int64); ok {
			return id, true
		}
	}
	return 0, false
}

// RequireAuth ensures that the request is authenticated
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Authentication required"))
			c.Abort()
			return
		}
		c.Next()
	}
}

// RequireActiveUser ensures that the authenticated user is active
func RequireActiveUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := getUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Authentication required"))
			c.Abort()
			return
		}

		if user.Status != "active" {
			logrus.WithFields(logrus.Fields{
				"user_id": user.UserID,
				"status":  user.Status,
			}).Warn("Inactive user attempted access")
			c.JSON(http.StatusForbidden, utils.ErrorResponse("Account is not active"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenBlacklistMiddleware checks if the token is blacklisted
func TokenBlacklistMiddleware(blacklist *utils.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := extractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		if blacklist.IsBlacklisted(token) {
			logrus.WithField("token", token[:min(len(token), 20)]).Warn("Blacklisted token used")
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse("Token has been revoked"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}