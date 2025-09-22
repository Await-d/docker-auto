package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/pkg/utils"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService handles authentication-related operations
type AuthService struct {
	db           *gorm.DB
	jwtManager   *utils.EnhancedJWTManager
	cacheManager *utils.CacheManager
	logger       *logrus.Logger
	config       *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(db *gorm.DB, cfg *config.Config, cacheManager *utils.CacheManager, logger *logrus.Logger) *AuthService {
	jwtManager := utils.NewEnhancedJWTManager(cfg, cacheManager, logger)
	return &AuthService{
		db:           db,
		jwtManager:   jwtManager,
		cacheManager: cacheManager,
		logger:       logger,
		config:       cfg,
	}
}


// ValidateLogin validates user credentials and returns user info if valid
func (s *AuthService) ValidateLogin(username, password string) (*model.User, error) {
	var user model.User

	// Find user by username or email
	err := s.db.Where("username = ? OR email = ?", username, username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("账户已被禁用")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Model(&user).Update("last_login_at", now)

	return &user, nil
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username  string `json:"username" binding:"required"`
	Password  string `json:"password" binding:"required"`
	DeviceID  string `json:"device_id"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	User        *model.User       `json:"user"`
	TokenInfo   *utils.TokenInfo  `json:"token_info"`
}

// Login authenticates a user and returns token information
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	// Validate user credentials
	user, err := s.ValidateLogin(req.Username, req.Password)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"username":   req.Username,
			"ip_address": req.IPAddress,
			"user_agent": req.UserAgent,
		}).Warn("Login attempt failed")
		return nil, err
	}

	// Prepare authentication context
	authContext := &utils.AuthContext{
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		DeviceID:  req.DeviceID,
	}

	// Generate token pair
	tokenInfo, err := s.jwtManager.GenerateTokenPair(ctx, user, authContext)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to generate token pair")
		return nil, fmt.Errorf("failed to generate authentication tokens: %w", err)
	}

	// Add permissions to user object for response
	user.Permissions = s.jwtManager.GetUserPermissions(user)

	// Log successful login
	s.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"ip_address": req.IPAddress,
		"device_id":  req.DeviceID,
		"session_id": tokenInfo.SessionID,
	}).Info("User logged in successfully")

	return &LoginResponse{
		User:      user,
		TokenInfo: tokenInfo,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string, authContext *utils.AuthContext) (*utils.TokenInfo, error) {
	// Validate refresh token and get user
	refreshClaims, err := s.jwtManager.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		s.logger.WithError(err).WithField("refresh_token", refreshToken[:20]+"...").Warn("Invalid refresh token")
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user from database
	var user model.User
	if err := s.db.Where("id = ? AND is_active = ?", refreshClaims.UserID, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found or inactive")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Generate new token pair
	tokenInfo, err := s.jwtManager.RefreshAccessToken(ctx, refreshToken, &user, authContext)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to refresh access token")
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    user.ID,
		"username":   user.Username,
		"session_id": refreshClaims.SessionID,
	}).Info("Token refreshed successfully")

	return tokenInfo, nil
}

// ValidateToken validates an access token and returns claims
func (s *AuthService) ValidateToken(ctx context.Context, token string) (*utils.EnhancedClaims, error) {
	claims, err := s.jwtManager.ValidateAccessToken(ctx, token)
	if err != nil {
		// Safely truncate token for logging
		tokenPreview := token
		if len(token) > 20 {
			tokenPreview = token[:20] + "..."
		}
		s.logger.WithError(err).WithField("token", tokenPreview).Debug("Token validation failed")
		return nil, err
	}

	return claims, nil
}

// Logout revokes a user's token
func (s *AuthService) Logout(ctx context.Context, token string) error {
	// Extract claims to get session info
	claims, err := s.jwtManager.ValidateAccessToken(ctx, token)
	if err != nil {
		// If token is invalid, consider logout successful
		s.logger.WithError(err).Debug("Token already invalid during logout")
		return nil
	}

	// Revoke the token
	if err := s.jwtManager.RevokeToken(ctx, token); err != nil {
		s.logger.WithError(err).WithField("user_id", claims.UserID).Warn("Failed to revoke token")
		// Don't return error as logout should always succeed from user perspective
	}

	// Revoke the session
	if err := s.jwtManager.RevokeSession(ctx, claims.SessionID); err != nil {
		s.logger.WithError(err).WithField("session_id", claims.SessionID).Warn("Failed to revoke session")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    claims.UserID,
		"username":   claims.Username,
		"session_id": claims.SessionID,
	}).Info("User logged out successfully")

	return nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	// Get user
	var user model.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database
	now := time.Now()
	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"password_hash": string(newPasswordHash),
		"updated_at":    now,
	}).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all user sessions (force re-login)
	if err := s.jwtManager.RevokeAllUserSessions(ctx, userID); err != nil {
		s.logger.WithError(err).WithField("user_id", userID).Warn("Failed to revoke all user sessions after password change")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  userID,
		"username": user.Username,
	}).Info("Password changed successfully")

	return nil
}

// AdminChangePassword changes a user's password (admin function)
func (s *AuthService) AdminChangePassword(ctx context.Context, adminUserID, targetUserID int64, newPassword string) error {
	// Verify admin user exists and has admin role
	var adminUser model.User
	if err := s.db.Where("id = ? AND role = ? AND is_active = ?", adminUserID, model.UserRoleAdmin, true).First(&adminUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("admin user not found or not authorized")
		}
		return fmt.Errorf("failed to verify admin user: %w", err)
	}

	// Get target user
	var targetUser model.User
	if err := s.db.Where("id = ?", targetUserID).First(&targetUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("target user not found")
		}
		return fmt.Errorf("failed to get target user: %w", err)
	}

	// Hash new password
	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password in database
	now := time.Now()
	if err := s.db.Model(&targetUser).Updates(map[string]interface{}{
		"password_hash": string(newPasswordHash),
		"updated_at":    now,
	}).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all target user sessions (force re-login)
	if err := s.jwtManager.RevokeAllUserSessions(ctx, targetUserID); err != nil {
		s.logger.WithError(err).WithField("user_id", targetUserID).Warn("Failed to revoke all user sessions after admin password change")
	}

	s.logger.WithFields(logrus.Fields{
		"admin_user_id":  adminUserID,
		"admin_username": adminUser.Username,
		"target_user_id": targetUserID,
		"target_username": targetUser.Username,
	}).Info("Password changed by admin successfully")

	return nil
}

// GetUserSessions returns active sessions for a user
func (s *AuthService) GetUserSessions(ctx context.Context, userID int64) ([]map[string]interface{}, error) {
	// This would require maintaining a user-to-sessions mapping
	// For now, return empty list
	// TODO: Implement user session tracking
	return []map[string]interface{}{}, nil
}

// RevokeUserSession revokes a specific user session
func (s *AuthService) RevokeUserSession(ctx context.Context, userID int64, sessionID string) error {
	// Verify user exists
	var user model.User
	if err := s.db.Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Revoke session
	if err := s.jwtManager.RevokeSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"session_id": sessionID,
	}).Info("User session revoked successfully")

	return nil
}

// GenerateToken generates a JWT token for the user (legacy method for backward compatibility)
func (s *AuthService) GenerateToken(user *model.User) (string, error) {
	// Use enhanced JWT manager
	authContext := &utils.AuthContext{
		IPAddress: "127.0.0.1",
		UserAgent: "legacy",
		DeviceID:  "legacy",
	}

	tokenInfo, err := s.jwtManager.GenerateTokenPair(context.Background(), user, authContext)
	if err != nil {
		return "", err
	}

	return tokenInfo.AccessToken, nil
}