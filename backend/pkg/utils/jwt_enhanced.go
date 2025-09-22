package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// EnhancedJWTManager provides advanced JWT functionality with Redis support
type EnhancedJWTManager struct {
	secretKey       []byte
	expireDuration  time.Duration
	refreshDuration time.Duration
	issuer          string
	cacheManager    *CacheManager
	logger          *logrus.Logger
	config          *config.Config
}

// NewEnhancedJWTManager creates a new enhanced JWT manager
func NewEnhancedJWTManager(cfg *config.Config, cacheManager *CacheManager, logger *logrus.Logger) *EnhancedJWTManager {
	return &EnhancedJWTManager{
		secretKey:       []byte(cfg.JWT.Secret),
		expireDuration:  time.Duration(cfg.JWT.ExpireHours) * time.Hour,
		refreshDuration: time.Duration(cfg.JWT.RefreshDays) * 24 * time.Hour,
		issuer:          "docker-auto",
		cacheManager:    cacheManager,
		logger:          logger,
		config:          cfg,
	}
}

// EnhancedClaims represents enhanced JWT claims with additional security features
type EnhancedClaims struct {
	UserID       int64            `json:"user_id"`
	Username     string           `json:"username"`
	Email        string           `json:"email"`
	Role         model.UserRole   `json:"role"`
	IsActive     bool             `json:"is_active"`
	SessionID    string           `json:"session_id"`    // Unique session identifier
	DeviceID     string           `json:"device_id"`     // Device fingerprint
	IPAddress    string           `json:"ip_address"`    // Client IP
	UserAgent    string           `json:"user_agent"`    // Client user agent
	Permissions  []string         `json:"permissions"`   // User permissions
	LastActivity int64            `json:"last_activity"` // Last activity timestamp
	jwt.RegisteredClaims
}

// EnhancedRefreshClaims represents enhanced refresh token claims
type EnhancedRefreshClaims struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	SessionID string `json:"session_id"`
	DeviceID  string `json:"device_id"`
	Type      string `json:"type"` // "refresh"
	jwt.RegisteredClaims
}

// TokenInfo contains comprehensive token information
type TokenInfo struct {
	AccessToken   string                 `json:"access_token"`
	RefreshToken  string                 `json:"refresh_token"`
	TokenType     string                 `json:"token_type"`
	ExpiresIn     int64                  `json:"expires_in"`
	ExpiresAt     time.Time              `json:"expires_at"`
	SessionID     string                 `json:"session_id"`
	Permissions   []string               `json:"permissions"`
	UserInfo      map[string]interface{} `json:"user_info"`
}

// AuthContext contains authentication context information
type AuthContext struct {
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
	DeviceID  string `json:"device_id"`
}

// GenerateTokenPair generates enhanced access and refresh tokens with session management
func (ejm *EnhancedJWTManager) GenerateTokenPair(ctx context.Context, user *model.User, authContext *AuthContext) (*TokenInfo, error) {
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}

	sessionID, err := ejm.generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Get user permissions
	permissions := ejm.GetUserPermissions(user)

	// Generate access token
	accessToken, err := ejm.generateAccessToken(user, sessionID, authContext, permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := ejm.generateRefreshToken(user, sessionID, authContext)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(ejm.expireDuration)

	// Store session information in cache
	sessionData := map[string]interface{}{
		"user_id":       user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"role":          user.Role,
		"permissions":   permissions,
		"device_id":     authContext.DeviceID,
		"ip_address":    authContext.IPAddress,
		"user_agent":    authContext.UserAgent,
		"created_at":    time.Now().Unix(),
		"last_activity": time.Now().Unix(),
	}

	if ejm.cacheManager != nil {
		if err := ejm.cacheManager.SetUserSession(ctx, sessionID, user.ID, sessionData, ejm.refreshDuration); err != nil {
			ejm.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to store session data")
		}
	}

	// Log successful token generation
	ejm.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"session_id": sessionID,
		"device_id":  authContext.DeviceID,
		"ip_address": authContext.IPAddress,
	}).Info("Token pair generated successfully")

	return &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(ejm.expireDuration.Seconds()),
		ExpiresAt:    expiresAt,
		SessionID:    sessionID,
		Permissions:  permissions,
		UserInfo: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	}, nil
}

// generateAccessToken creates an enhanced access token
func (ejm *EnhancedJWTManager) generateAccessToken(user *model.User, sessionID string, authContext *AuthContext, permissions []string) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ejm.expireDuration)

	claims := &EnhancedClaims{
		UserID:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Role:         user.Role,
		IsActive:     user.IsActive,
		SessionID:    sessionID,
		DeviceID:     authContext.DeviceID,
		IPAddress:    authContext.IPAddress,
		UserAgent:    authContext.UserAgent,
		Permissions:  permissions,
		LastActivity: now.Unix(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ejm.issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"docker-auto-api"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        ejm.generateJTI(user.ID, sessionID, now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(ejm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, nil
}

// generateRefreshToken creates an enhanced refresh token
func (ejm *EnhancedJWTManager) generateRefreshToken(user *model.User, sessionID string, authContext *AuthContext) (string, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ejm.refreshDuration)

	claims := &EnhancedRefreshClaims{
		UserID:    user.ID,
		Username:  user.Username,
		SessionID: sessionID,
		DeviceID:  authContext.DeviceID,
		Type:      "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    ejm.issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"docker-auto-refresh"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        ejm.generateJTI(user.ID, sessionID, now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(ejm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// ValidateAccessToken validates and parses an enhanced access token
func (ejm *EnhancedJWTManager) ValidateAccessToken(ctx context.Context, tokenString string) (*EnhancedClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &EnhancedClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ejm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*EnhancedClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Validate claims
	if err := ejm.validateEnhancedClaims(claims); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Check if token is blacklisted
	if ejm.cacheManager != nil {
		blacklisted, err := ejm.cacheManager.IsTokenBlacklisted(ctx, claims.ID)
		if err != nil {
			ejm.logger.WithError(err).WithField("token_id", claims.ID).Warn("Failed to check token blacklist")
		} else if blacklisted {
			return nil, fmt.Errorf("token has been revoked")
		}
	}

	// Validate session
	if err := ejm.validateSession(ctx, claims); err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	// Update last activity
	if ejm.cacheManager != nil {
		sessionData := map[string]interface{}{
			"last_activity": time.Now().Unix(),
		}
		if err := ejm.cacheManager.SetUserSession(ctx, claims.SessionID, claims.UserID, sessionData, ejm.refreshDuration); err != nil {
			ejm.logger.WithError(err).WithField("session_id", claims.SessionID).Warn("Failed to update session activity")
		}
	}

	return claims, nil
}

// ValidateRefreshToken validates and parses an enhanced refresh token
func (ejm *EnhancedJWTManager) ValidateRefreshToken(ctx context.Context, tokenString string) (*EnhancedRefreshClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("refresh token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &EnhancedRefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return ejm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*EnhancedRefreshClaims)
	if !ok {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("refresh token is invalid")
	}

	if claims.Type != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	// Validate refresh claims
	if err := ejm.validateEnhancedRefreshClaims(claims); err != nil {
		return nil, fmt.Errorf("refresh token validation failed: %w", err)
	}

	// Check if token is blacklisted
	if ejm.cacheManager != nil {
		blacklisted, err := ejm.cacheManager.IsTokenBlacklisted(ctx, claims.ID)
		if err != nil {
			ejm.logger.WithError(err).WithField("token_id", claims.ID).Warn("Failed to check refresh token blacklist")
		} else if blacklisted {
			return nil, fmt.Errorf("refresh token has been revoked")
		}
	}

	return claims, nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (ejm *EnhancedJWTManager) RefreshAccessToken(ctx context.Context, refreshTokenString string, user *model.User, authContext *AuthContext) (*TokenInfo, error) {
	// Validate refresh token
	refreshClaims, err := ejm.ValidateRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify user ID matches
	if refreshClaims.UserID != user.ID {
		return nil, fmt.Errorf("refresh token user ID does not match")
	}

	// Get user permissions
	permissions := ejm.GetUserPermissions(user)

	// Generate new access token with same session ID
	accessToken, err := ejm.generateAccessToken(user, refreshClaims.SessionID, authContext, permissions)
	if err != nil {
		return nil, fmt.Errorf("failed to generate new access token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(ejm.expireDuration)

	// Update session activity
	if ejm.cacheManager != nil {
		sessionData := map[string]interface{}{
			"last_activity": time.Now().Unix(),
			"ip_address":    authContext.IPAddress,
			"user_agent":    authContext.UserAgent,
		}
		if err := ejm.cacheManager.SetUserSession(ctx, refreshClaims.SessionID, user.ID, sessionData, ejm.refreshDuration); err != nil {
			ejm.logger.WithError(err).WithField("session_id", refreshClaims.SessionID).Warn("Failed to update session during refresh")
		}
	}

	ejm.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"session_id": refreshClaims.SessionID,
	}).Info("Access token refreshed successfully")

	return &TokenInfo{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString, // Keep the same refresh token
		TokenType:    "Bearer",
		ExpiresIn:    int64(ejm.expireDuration.Seconds()),
		ExpiresAt:    expiresAt,
		SessionID:    refreshClaims.SessionID,
		Permissions:  permissions,
		UserInfo: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	}, nil
}

// RevokeToken revokes a token by adding it to the blacklist
func (ejm *EnhancedJWTManager) RevokeToken(ctx context.Context, tokenString string) error {
	// Parse token to get expiration time
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return ejm.secretKey, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token for revocation: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims for revocation")
	}

	tokenID, ok := claims["jti"].(string)
	if !ok {
		return fmt.Errorf("token ID not found")
	}

	var expiration time.Time
	if exp, ok := claims["exp"].(float64); ok {
		expiration = time.Unix(int64(exp), 0)
	} else {
		return fmt.Errorf("token expiration not found")
	}

	// Add to blacklist
	if ejm.cacheManager != nil {
		if err := ejm.cacheManager.BlacklistToken(ctx, tokenID, expiration); err != nil {
			return fmt.Errorf("failed to blacklist token: %w", err)
		}
	}

	ejm.logger.WithFields(logrus.Fields{
		"token_id":   tokenID,
		"expires_at": expiration,
	}).Info("Token revoked successfully")

	return nil
}

// RevokeSession revokes all tokens for a specific session
func (ejm *EnhancedJWTManager) RevokeSession(ctx context.Context, sessionID string) error {
	if ejm.cacheManager == nil {
		return fmt.Errorf("cache manager is required for session revocation")
	}

	// Delete session data
	if err := ejm.cacheManager.DeleteUserSession(ctx, sessionID); err != nil {
		ejm.logger.WithError(err).WithField("session_id", sessionID).Warn("Failed to delete session data")
	}

	ejm.logger.WithField("session_id", sessionID).Info("Session revoked successfully")
	return nil
}

// RevokeAllUserSessions revokes all sessions for a specific user
func (ejm *EnhancedJWTManager) RevokeAllUserSessions(ctx context.Context, userID int64) error {
	// This would require maintaining a user-to-sessions mapping in Redis
	// For now, we'll log the action and let individual tokens expire
	ejm.logger.WithField("user_id", userID).Info("All user sessions revocation requested")
	return nil
}

// GetSessionInfo retrieves session information
func (ejm *EnhancedJWTManager) GetSessionInfo(ctx context.Context, sessionID string) (map[string]interface{}, error) {
	if ejm.cacheManager == nil {
		return nil, fmt.Errorf("cache manager is required for session info")
	}

	sessionData, exists, err := ejm.cacheManager.GetUserSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session info: %w", err)
	}

	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	return sessionData, nil
}

// Helper methods

func (ejm *EnhancedJWTManager) generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (ejm *EnhancedJWTManager) generateJTI(userID int64, sessionID string, issuedAt time.Time) string {
	return fmt.Sprintf("%d_%s_%d", userID, sessionID[:8], issuedAt.Unix())
}

func (ejm *EnhancedJWTManager) GetUserPermissions(user *model.User) []string {
	// Define role-based permissions
	permissions := make([]string, 0)

	switch user.Role {
	case model.UserRoleAdmin:
		permissions = []string{
			// Administrator has all permissions
			"container:read", "container:create", "container:update", "container:delete",
			"container:start", "container:stop",
			"image:read", "image:check", "image:update",
			"update:read", "update:create", "update:rollback",
			"system:read", "system:config", "system:logs",
			"user:read", "user:create", "user:update", "user:delete",
		}
	case model.UserRoleOperator:
		permissions = []string{
			// Operator permissions
			"container:read", "container:create", "container:update",
			"container:start", "container:stop",
			"image:read", "image:check", "image:update",
			"update:read", "update:create", "update:rollback",
			"system:read", "system:logs",
		}
	case model.UserRoleViewer:
		permissions = []string{
			// Viewer permissions (read-only)
			"container:read",
			"image:read",
			"update:read",
			"system:read",
		}
	default:
		permissions = []string{"system:read"} // Basic read permission
	}

	return permissions
}

func (ejm *EnhancedJWTManager) validateEnhancedClaims(claims *EnhancedClaims) error {
	now := time.Now().UTC()

	// Check issuer
	if claims.Issuer != ejm.issuer {
		return fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(now) {
		return fmt.Errorf("token has expired")
	}

	// Check not before
	if claims.NotBefore != nil && claims.NotBefore.Time.After(now) {
		return fmt.Errorf("token is not yet valid")
	}

	// Check user ID
	if claims.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	// Check username
	if claims.Username == "" {
		return fmt.Errorf("username is required")
	}

	// Check session ID
	if claims.SessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	// Validate role
	validRoles := model.GetValidRoles()
	roleValid := false
	for _, role := range validRoles {
		if claims.Role == role {
			roleValid = true
			break
		}
	}
	if !roleValid {
		return fmt.Errorf("invalid user role: %s", claims.Role)
	}

	return nil
}

func (ejm *EnhancedJWTManager) validateEnhancedRefreshClaims(claims *EnhancedRefreshClaims) error {
	now := time.Now().UTC()

	// Check issuer
	if claims.Issuer != ejm.issuer {
		return fmt.Errorf("invalid issuer: %s", claims.Issuer)
	}

	// Check expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(now) {
		return fmt.Errorf("refresh token has expired")
	}

	// Check not before
	if claims.NotBefore != nil && claims.NotBefore.Time.After(now) {
		return fmt.Errorf("refresh token is not yet valid")
	}

	// Check user ID
	if claims.UserID <= 0 {
		return fmt.Errorf("invalid user ID")
	}

	// Check username
	if claims.Username == "" {
		return fmt.Errorf("username is required")
	}

	// Check session ID
	if claims.SessionID == "" {
		return fmt.Errorf("session ID is required")
	}

	return nil
}

func (ejm *EnhancedJWTManager) validateSession(ctx context.Context, claims *EnhancedClaims) error {
	if ejm.cacheManager == nil {
		return nil // Skip session validation if cache is not available
	}

	sessionData, exists, err := ejm.cacheManager.GetUserSession(ctx, claims.SessionID)
	if err != nil {
		ejm.logger.WithError(err).WithField("session_id", claims.SessionID).Warn("Failed to validate session")
		return nil // Don't fail validation for cache errors
	}

	if !exists {
		return fmt.Errorf("session not found or expired")
	}

	// Verify session data matches token claims
	if sessionData["user_id"] != nil {
		if sessionUserID, ok := sessionData["user_id"].(float64); ok {
			if int64(sessionUserID) != claims.UserID {
				return fmt.Errorf("session user ID mismatch")
			}
		}
	}

	return nil
}

// ExtractTokenFromHeader is available in jwt.go

// GetUserIDFromToken extracts user ID from token without full validation
func (ejm *EnhancedJWTManager) GetUserIDFromToken(tokenString string) (int64, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &EnhancedClaims{})
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*EnhancedClaims); ok {
		return claims.UserID, nil
	}

	return 0, fmt.Errorf("invalid token claims")
}

// IsTokenExpired checks if a token is expired without full validation
func (ejm *EnhancedJWTManager) IsTokenExpired(tokenString string) bool {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &EnhancedClaims{})
	if err != nil {
		return true
	}

	if claims, ok := token.Claims.(*EnhancedClaims); ok {
		return claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now().UTC())
	}

	return true
}