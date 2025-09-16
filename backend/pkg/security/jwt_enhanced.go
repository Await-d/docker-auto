package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// JWTConfig represents enhanced JWT configuration
type JWTConfig struct {
	SecretKey          string        `json:"secret_key"`
	AccessTokenTTL     time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL    time.Duration `json:"refresh_token_ttl"`
	TokenRotation      bool          `json:"token_rotation"`
	BlacklistEnabled   bool          `json:"blacklist_enabled"`
	SecureHeaders      bool          `json:"secure_headers"`
	IssuerName         string        `json:"issuer_name"`
	MaxTokenAge        time.Duration `json:"max_token_age"`
	RotationThreshold  time.Duration `json:"rotation_threshold"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
}

// DefaultJWTConfig returns a secure default configuration
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		AccessTokenTTL:    15 * time.Minute,
		RefreshTokenTTL:   7 * 24 * time.Hour,
		TokenRotation:     true,
		BlacklistEnabled:  true,
		SecureHeaders:     true,
		IssuerName:        "docker-auto",
		MaxTokenAge:       24 * time.Hour,
		RotationThreshold: 5 * time.Minute,
		CleanupInterval:   time.Hour,
	}
}

// EnhancedClaims represents enhanced JWT claims with security features
type EnhancedClaims struct {
	UserID     int64  `json:"user_id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	Status     string `json:"status"`
	SessionID  string `json:"session_id"`
	TokenType  string `json:"token_type"` // "access" or "refresh"
	ClientIP   string `json:"client_ip"`
	UserAgent  string `json:"user_agent"`
	IssuedAt   int64  `json:"iat"`
	ExpiresAt  int64  `json:"exp"`
	NotBefore  int64  `json:"nbf"`
	Issuer     string `json:"iss"`
	Subject    string `json:"sub"`
	JTI        string `json:"jti"` // JWT ID for blacklisting
	SecurityLevel int `json:"security_level"` // 1=low, 2=medium, 3=high
	jwt.RegisteredClaims
}

// TokenBlacklist manages blacklisted tokens with memory optimization
type TokenBlacklist struct {
	tokens     map[string]time.Time // JTI -> expiration time
	mutex      sync.RWMutex
	maxSize    int
	cleanupTTL time.Duration
}

// NewTokenBlacklist creates a new token blacklist
func NewTokenBlacklist(maxSize int, cleanupTTL time.Duration) *TokenBlacklist {
	bl := &TokenBlacklist{
		tokens:     make(map[string]time.Time),
		maxSize:    maxSize,
		cleanupTTL: cleanupTTL,
	}

	// Start cleanup goroutine
	go bl.startCleanup()

	return bl
}

// Add adds a token to the blacklist
func (bl *TokenBlacklist) Add(jti string, expiration time.Time) {
	bl.mutex.Lock()
	defer bl.mutex.Unlock()

	// Check size limits
	if len(bl.tokens) >= bl.maxSize {
		bl.cleanup() // Clean up expired tokens
	}

	bl.tokens[jti] = expiration
	logrus.WithField("jti", jti).Debug("Token added to blacklist")
}

// IsBlacklisted checks if a token is blacklisted
func (bl *TokenBlacklist) IsBlacklisted(jti string) bool {
	bl.mutex.RLock()
	defer bl.mutex.RUnlock()

	expiration, exists := bl.tokens[jti]
	if !exists {
		return false
	}

	// Check if token has expired
	if time.Now().After(expiration) {
		// Remove expired token in the next cleanup
		return false
	}

	return true
}

// cleanup removes expired tokens
func (bl *TokenBlacklist) cleanup() {
	now := time.Now()
	for jti, expiration := range bl.tokens {
		if now.After(expiration) {
			delete(bl.tokens, jti)
		}
	}
	logrus.WithField("remaining_tokens", len(bl.tokens)).Debug("Blacklist cleanup completed")
}

// startCleanup starts periodic cleanup
func (bl *TokenBlacklist) startCleanup() {
	ticker := time.NewTicker(bl.cleanupTTL)
	defer ticker.Stop()

	for range ticker.C {
		bl.mutex.Lock()
		bl.cleanup()
		bl.mutex.Unlock()
	}
}

// GetStats returns blacklist statistics
func (bl *TokenBlacklist) GetStats() map[string]interface{} {
	bl.mutex.RLock()
	defer bl.mutex.RUnlock()

	activeTokens := 0
	now := time.Now()
	for _, expiration := range bl.tokens {
		if now.Before(expiration) {
			activeTokens++
		}
	}

	return map[string]interface{}{
		"total_tokens":  len(bl.tokens),
		"active_tokens": activeTokens,
		"max_size":      bl.maxSize,
	}
}

// SessionManager manages user sessions with security controls
type SessionManager struct {
	sessions       map[string]*SessionInfo
	userSessions   map[int64][]string // UserID -> SessionIDs
	mutex          sync.RWMutex
	maxSessions    int
	sessionTimeout time.Duration
}

// SessionInfo represents session information
type SessionInfo struct {
	SessionID     string    `json:"session_id"`
	UserID        int64     `json:"user_id"`
	ClientIP      string    `json:"client_ip"`
	UserAgent     string    `json:"user_agent"`
	CreatedAt     time.Time `json:"created_at"`
	LastActivity  time.Time `json:"last_activity"`
	AccessCount   int64     `json:"access_count"`
	SecurityLevel int       `json:"security_level"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(maxSessions int, timeout time.Duration) *SessionManager {
	sm := &SessionManager{
		sessions:       make(map[string]*SessionInfo),
		userSessions:   make(map[int64][]string),
		maxSessions:    maxSessions,
		sessionTimeout: timeout,
	}

	// Start cleanup goroutine
	go sm.startCleanup()

	return sm
}

// CreateSession creates a new session
func (sm *SessionManager) CreateSession(userID int64, clientIP, userAgent string, securityLevel int) (string, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Generate session ID
	sessionID, err := generateSecureID()
	if err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Check concurrent session limits
	if userSessions, exists := sm.userSessions[userID]; exists {
		if len(userSessions) >= sm.maxSessions {
			// Remove oldest session
			oldestSessionID := userSessions[0]
			sm.removeSession(oldestSessionID, userID)
			logrus.WithFields(logrus.Fields{
				"user_id":           userID,
				"removed_session":   oldestSessionID,
				"new_session":       sessionID,
			}).Warn("Session limit exceeded, removed oldest session")
		}
	}

	// Create session info
	sessionInfo := &SessionInfo{
		SessionID:     sessionID,
		UserID:        userID,
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		CreatedAt:     time.Now(),
		LastActivity:  time.Now(),
		AccessCount:   1,
		SecurityLevel: securityLevel,
	}

	// Store session
	sm.sessions[sessionID] = sessionInfo
	if sm.userSessions[userID] == nil {
		sm.userSessions[userID] = make([]string, 0)
	}
	sm.userSessions[userID] = append(sm.userSessions[userID], sessionID)

	logrus.WithFields(logrus.Fields{
		"user_id":        userID,
		"session_id":     sessionID,
		"client_ip":      clientIP,
		"security_level": securityLevel,
	}).Info("New session created")

	return sessionID, nil
}

// ValidateSession validates a session and updates activity
func (sm *SessionManager) ValidateSession(sessionID string, clientIP, userAgent string) (*SessionInfo, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check timeout
	if time.Since(session.LastActivity) > sm.sessionTimeout {
		sm.removeSession(sessionID, session.UserID)
		return nil, fmt.Errorf("session expired")
	}

	// Validate client context (optional strict mode)
	if session.SecurityLevel >= 3 {
		if session.ClientIP != clientIP || session.UserAgent != userAgent {
			logrus.WithFields(logrus.Fields{
				"session_id":        sessionID,
				"expected_ip":       session.ClientIP,
				"actual_ip":         clientIP,
				"expected_agent":    session.UserAgent,
				"actual_agent":      userAgent,
			}).Warn("Session context mismatch detected")
			return nil, fmt.Errorf("session context mismatch")
		}
	}

	// Update activity
	session.LastActivity = time.Now()
	session.AccessCount++

	return session, nil
}

// RevokeSession revokes a specific session
func (sm *SessionManager) RevokeSession(sessionID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found")
	}

	sm.removeSession(sessionID, session.UserID)
	logrus.WithFields(logrus.Fields{
		"session_id": sessionID,
		"user_id":    session.UserID,
	}).Info("Session revoked")

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user
func (sm *SessionManager) RevokeAllUserSessions(userID int64) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	userSessions, exists := sm.userSessions[userID]
	if !exists {
		return nil // No sessions to revoke
	}

	for _, sessionID := range userSessions {
		delete(sm.sessions, sessionID)
	}
	delete(sm.userSessions, userID)

	logrus.WithFields(logrus.Fields{
		"user_id":        userID,
		"revoked_count":  len(userSessions),
	}).Info("All user sessions revoked")

	return nil
}

// removeSession removes a session (internal use)
func (sm *SessionManager) removeSession(sessionID string, userID int64) {
	delete(sm.sessions, sessionID)

	if userSessions, exists := sm.userSessions[userID]; exists {
		for i, sid := range userSessions {
			if sid == sessionID {
				sm.userSessions[userID] = append(userSessions[:i], userSessions[i+1:]...)
				break
			}
		}
		if len(sm.userSessions[userID]) == 0 {
			delete(sm.userSessions, userID)
		}
	}
}

// startCleanup starts periodic session cleanup
func (sm *SessionManager) startCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sm.cleanupExpiredSessions()
	}
}

// cleanupExpiredSessions removes expired sessions
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	now := time.Now()
	expiredSessions := make([]string, 0)

	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastActivity) > sm.sessionTimeout {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	for _, sessionID := range expiredSessions {
		session := sm.sessions[sessionID]
		sm.removeSession(sessionID, session.UserID)
	}

	if len(expiredSessions) > 0 {
		logrus.WithField("cleaned_sessions", len(expiredSessions)).Info("Expired sessions cleaned up")
	}
}

// GetSessionStats returns session statistics
func (sm *SessionManager) GetSessionStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return map[string]interface{}{
		"total_sessions": len(sm.sessions),
		"total_users":    len(sm.userSessions),
		"max_sessions":   sm.maxSessions,
		"timeout_minutes": int(sm.sessionTimeout.Minutes()),
	}
}

// generateSecureID generates a cryptographically secure ID
func generateSecureID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// EnhancedJWTManager manages JWT tokens with enhanced security
type EnhancedJWTManager struct {
	config         *JWTConfig
	blacklist      *TokenBlacklist
	sessionManager *SessionManager
	secretKey      []byte
}

// NewEnhancedJWTManager creates a new enhanced JWT manager
func NewEnhancedJWTManager(config *JWTConfig) *EnhancedJWTManager {
	if config == nil {
		config = DefaultJWTConfig()
	}

	return &EnhancedJWTManager{
		config:         config,
		blacklist:      NewTokenBlacklist(10000, config.CleanupInterval),
		sessionManager: NewSessionManager(5, 24*time.Hour),
		secretKey:      []byte(config.SecretKey),
	}
}

// GenerateTokenPair generates access and refresh token pair
func (ejm *EnhancedJWTManager) GenerateTokenPair(userID int64, username, email, role, status, clientIP, userAgent string, securityLevel int) (accessToken, refreshToken string, err error) {
	// Create session
	sessionID, err := ejm.sessionManager.CreateSession(userID, clientIP, userAgent, securityLevel)
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %w", err)
	}

	now := time.Now()

	// Generate access token
	accessJTI, err := generateSecureID()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate access token JTI: %w", err)
	}

	accessClaims := &EnhancedClaims{
		UserID:        userID,
		Username:      username,
		Email:         email,
		Role:          role,
		Status:        status,
		SessionID:     sessionID,
		TokenType:     "access",
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		IssuedAt:      now.Unix(),
		ExpiresAt:     now.Add(ejm.config.AccessTokenTTL).Unix(),
		NotBefore:     now.Unix(),
		Issuer:        ejm.config.IssuerName,
		Subject:       username,
		JTI:           accessJTI,
		SecurityLevel: securityLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ejm.config.AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    ejm.config.IssuerName,
			Subject:   username,
			ID:        accessJTI,
		},
	}

	accessToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(ejm.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshJTI, err := generateSecureID()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate refresh token JTI: %w", err)
	}

	refreshClaims := &EnhancedClaims{
		UserID:        userID,
		Username:      username,
		Email:         email,
		Role:          role,
		Status:        status,
		SessionID:     sessionID,
		TokenType:     "refresh",
		ClientIP:      clientIP,
		UserAgent:     userAgent,
		IssuedAt:      now.Unix(),
		ExpiresAt:     now.Add(ejm.config.RefreshTokenTTL).Unix(),
		NotBefore:     now.Unix(),
		Issuer:        ejm.config.IssuerName,
		Subject:       username,
		JTI:           refreshJTI,
		SecurityLevel: securityLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ejm.config.RefreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    ejm.config.IssuerName,
			Subject:   username,
			ID:        refreshJTI,
		},
	}

	refreshToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(ejm.secretKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":        userID,
		"session_id":     sessionID,
		"access_jti":     accessJTI,
		"refresh_jti":    refreshJTI,
		"security_level": securityLevel,
	}).Info("Token pair generated")

	return accessToken, refreshToken, nil
}

// ValidateToken validates a JWT token with enhanced security checks
func (ejm *EnhancedJWTManager) ValidateToken(tokenString, clientIP, userAgent string) (*EnhancedClaims, error) {
	// Parse token
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
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is blacklisted
	if ejm.config.BlacklistEnabled && ejm.blacklist.IsBlacklisted(claims.JTI) {
		return nil, fmt.Errorf("token is blacklisted")
	}

	// Validate session
	sessionInfo, err := ejm.sessionManager.ValidateSession(claims.SessionID, clientIP, userAgent)
	if err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	// Update claims with latest session info
	claims.ClientIP = sessionInfo.ClientIP
	claims.UserAgent = sessionInfo.UserAgent

	// Check token rotation requirement
	if ejm.config.TokenRotation && claims.TokenType == "access" {
		issuedAt := time.Unix(claims.IssuedAt, 0)
		if time.Since(issuedAt) > ejm.config.RotationThreshold {
			logrus.WithFields(logrus.Fields{
				"jti":      claims.JTI,
				"user_id":  claims.UserID,
				"age":      time.Since(issuedAt),
			}).Info("Token requires rotation")
		}
	}

	return claims, nil
}

// RefreshTokens refreshes access and refresh tokens
func (ejm *EnhancedJWTManager) RefreshTokens(refreshTokenString, clientIP, userAgent string) (newAccessToken, newRefreshToken string, err error) {
	// Validate refresh token
	claims, err := ejm.ValidateToken(refreshTokenString, clientIP, userAgent)
	if err != nil {
		return "", "", fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.TokenType != "refresh" {
		return "", "", fmt.Errorf("token is not a refresh token")
	}

	// Blacklist old refresh token
	if ejm.config.BlacklistEnabled {
		ejm.blacklist.Add(claims.JTI, time.Unix(claims.ExpiresAt, 0))
	}

	// Generate new token pair
	newAccessToken, newRefreshToken, err = ejm.GenerateTokenPair(
		claims.UserID,
		claims.Username,
		claims.Email,
		claims.Role,
		claims.Status,
		clientIP,
		userAgent,
		claims.SecurityLevel,
	)

	if err != nil {
		return "", "", fmt.Errorf("failed to generate new token pair: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id":    claims.UserID,
		"old_jti":    claims.JTI,
		"session_id": claims.SessionID,
	}).Info("Tokens refreshed")

	return newAccessToken, newRefreshToken, nil
}

// RevokeToken revokes a token by adding it to blacklist
func (ejm *EnhancedJWTManager) RevokeToken(tokenString string) error {
	token, err := jwt.ParseWithClaims(tokenString, &EnhancedClaims{}, func(token *jwt.Token) (interface{}, error) {
		return ejm.secretKey, nil
	})

	if err != nil {
		return fmt.Errorf("failed to parse token for revocation: %w", err)
	}

	claims, ok := token.Claims.(*EnhancedClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	// Add to blacklist
	if ejm.config.BlacklistEnabled {
		ejm.blacklist.Add(claims.JTI, time.Unix(claims.ExpiresAt, 0))
	}

	// Revoke session if it's an access token
	if claims.TokenType == "access" {
		err = ejm.sessionManager.RevokeSession(claims.SessionID)
		if err != nil {
			logrus.WithError(err).Warn("Failed to revoke session")
		}
	}

	logrus.WithFields(logrus.Fields{
		"jti":        claims.JTI,
		"user_id":    claims.UserID,
		"token_type": claims.TokenType,
	}).Info("Token revoked")

	return nil
}

// RevokeAllUserTokens revokes all tokens for a user
func (ejm *EnhancedJWTManager) RevokeAllUserTokens(userID int64) error {
	// Revoke all user sessions
	err := ejm.sessionManager.RevokeAllUserSessions(userID)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	logrus.WithField("user_id", userID).Info("All user tokens revoked")
	return nil
}

// GetStats returns comprehensive JWT manager statistics
func (ejm *EnhancedJWTManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"blacklist":     ejm.blacklist.GetStats(),
		"sessions":      ejm.sessionManager.GetSessionStats(),
		"config": map[string]interface{}{
			"access_token_ttl_minutes":  int(ejm.config.AccessTokenTTL.Minutes()),
			"refresh_token_ttl_hours":   int(ejm.config.RefreshTokenTTL.Hours()),
			"token_rotation_enabled":    ejm.config.TokenRotation,
			"blacklist_enabled":         ejm.config.BlacklistEnabled,
			"rotation_threshold_minutes": int(ejm.config.RotationThreshold.Minutes()),
		},
	}
}