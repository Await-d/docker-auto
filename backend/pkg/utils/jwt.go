package utils

import (
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
)

// Claims represents JWT claims structure
type Claims struct {
	UserID   int64            `json:"user_id"`
	Username string           `json:"username"`
	Email    string           `json:"email"`
	Role     model.UserRole   `json:"role"`
	IsActive bool             `json:"is_active"`
	jwt.RegisteredClaims
}

// JWTManager manages JWT token operations
type JWTManager struct {
	secretKey       []byte
	expireDuration  time.Duration
	refreshDuration time.Duration
	issuer          string
}

// TokenPair represents access and refresh tokens
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int64     `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshClaims represents refresh token claims
type RefreshClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Type     string `json:"type"` // "refresh"
	jwt.RegisteredClaims
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(cfg *config.Config) *JWTManager {
	return &JWTManager{
		secretKey:       []byte(cfg.JWT.Secret),
		expireDuration:  time.Duration(cfg.JWT.ExpireHours) * time.Hour,
		refreshDuration: time.Duration(cfg.JWT.RefreshDays) * 24 * time.Hour,
		issuer:          "docker-auto",
	}
}

// GenerateJWT generates an access token for the user
func GenerateJWT(user *model.User, secret string) (string, error) {
	manager := &JWTManager{
		secretKey:      []byte(secret),
		expireDuration: 24 * time.Hour, // Default 24 hours
		issuer:         "docker-auto",
	}

	return manager.GenerateAccessToken(user)
}

// GenerateAccessToken generates an access token for the user
func (jm *JWTManager) GenerateAccessToken(user *model.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(jm.expireDuration)

	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		IsActive: user.IsActive,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"docker-auto-api"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateJTI(user.ID, now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"expires_at": expiresAt,
	}).Debug("Access token generated")

	return tokenString, nil
}

// GenerateRefreshToken generates a refresh token for the user
func (jm *JWTManager) GenerateRefreshToken(user *model.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user cannot be nil")
	}

	now := time.Now().UTC()
	expiresAt := now.Add(jm.refreshDuration)

	claims := &RefreshClaims{
		UserID:   user.ID,
		Username: user.Username,
		Type:     "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    jm.issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  []string{"docker-auto-refresh"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateJTI(user.ID, now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jm.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"expires_at": expiresAt,
	}).Debug("Refresh token generated")

	return tokenString, nil
}

// GenerateTokenPair generates both access and refresh tokens
func (jm *JWTManager) GenerateTokenPair(user *model.User) (*TokenPair, error) {
	accessToken, err := jm.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := jm.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(jm.expireDuration)

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(jm.expireDuration.Seconds()),
		ExpiresAt:    expiresAt,
	}, nil
}

// ValidateJWT validates and parses a JWT token
func ValidateJWT(tokenString string, secret string) (*Claims, error) {
	manager := &JWTManager{
		secretKey: []byte(secret),
		issuer:    "docker-auto",
	}

	return manager.ValidateAccessToken(tokenString)
}

// ValidateAccessToken validates and parses an access token
func (jm *JWTManager) ValidateAccessToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	// Additional validation
	if err := jm.validateClaims(claims); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	return claims, nil
}

// ValidateRefreshToken validates and parses a refresh token
func (jm *JWTManager) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("refresh token is empty")
	}

	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok {
		return nil, fmt.Errorf("invalid refresh token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("refresh token is invalid")
	}

	if claims.Type != "refresh" {
		return nil, fmt.Errorf("token is not a refresh token")
	}

	// Additional validation
	if err := jm.validateRefreshClaims(claims); err != nil {
		return nil, fmt.Errorf("refresh token validation failed: %w", err)
	}

	return claims, nil
}

// validateClaims performs additional validation on access token claims
func (jm *JWTManager) validateClaims(claims *Claims) error {
	now := time.Now().UTC()

	// Check issuer
	if claims.Issuer != jm.issuer {
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

	// Check role
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

// validateRefreshClaims performs additional validation on refresh token claims
func (jm *JWTManager) validateRefreshClaims(claims *RefreshClaims) error {
	now := time.Now().UTC()

	// Check issuer
	if claims.Issuer != jm.issuer {
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

	return nil
}

// RefreshAccessToken generates a new access token using a refresh token
func (jm *JWTManager) RefreshAccessToken(refreshTokenString string, user *model.User) (string, error) {
	// Validate refresh token
	refreshClaims, err := jm.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify user ID matches
	if refreshClaims.UserID != user.ID {
		return "", fmt.Errorf("refresh token user ID does not match")
	}

	// Generate new access token
	return jm.GenerateAccessToken(user)
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) {
		return "", fmt.Errorf("invalid authorization header format")
	}

	if authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}

	return token, nil
}

// GetTokenExpiration returns the expiration time of a token
func (jm *JWTManager) GetTokenExpiration(tokenString string) (*time.Time, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jm.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			expTime := time.Unix(int64(exp), 0)
			return &expTime, nil
		}
	}

	return nil, fmt.Errorf("unable to extract expiration time")
}

// IsTokenExpired checks if a token is expired without full validation
func (jm *JWTManager) IsTokenExpired(tokenString string) bool {
	expTime, err := jm.GetTokenExpiration(tokenString)
	if err != nil {
		return true
	}

	return time.Now().UTC().After(*expTime)
}

// generateJTI generates a unique JWT ID
func generateJTI(userID int64, issuedAt time.Time) string {
	return fmt.Sprintf("%d_%d", userID, issuedAt.Unix())
}

// GetUserIDFromToken extracts user ID from token without full validation
func GetUserIDFromToken(tokenString string) (int64, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &Claims{})
	if err != nil {
		return 0, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok {
		return claims.UserID, nil
	}

	return 0, fmt.Errorf("invalid token claims")
}

// CreateTokenBlacklist creates a simple in-memory token blacklist
type TokenBlacklist struct {
	cache *MemoryCache
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{
		cache: NewMemoryCache(),
	}
}

// Add adds a token to the blacklist
func (tb *TokenBlacklist) Add(tokenString string, expiration time.Time) error {
	ttl := time.Until(expiration)
	if ttl <= 0 {
		return nil // Token already expired, no need to blacklist
	}

	return tb.cache.Set(tokenString, true, ttl)
}

// IsBlacklisted checks if a token is blacklisted
func (tb *TokenBlacklist) IsBlacklisted(tokenString string) bool {
	_, exists := tb.cache.Get(tokenString)
	return exists
}

// Cleanup removes expired entries from the blacklist
func (tb *TokenBlacklist) Cleanup() {
	// The cache automatically handles cleanup of expired items
	logrus.Debug("Token blacklist cleanup completed")
}