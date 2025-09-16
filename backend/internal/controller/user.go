package controller

import (
	"net/http"
	"strconv"

	"docker-auto/internal/middleware"
	"docker-auto/internal/service"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// UserController handles user-related HTTP requests
type UserController struct {
	userService *service.UserService
	logger      *logrus.Logger
}

// NewUserController creates a new user controller
func NewUserController(userService *service.UserService, logger *logrus.Logger) *UserController {
	return &UserController{
		userService: userService,
		logger:      logger,
	}
}

// Authentication endpoints

// Login godoc
// @Summary User login
// @Description Authenticate user with username/email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body service.LoginRequest true "Login credentials"
// @Success 200 {object} utils.APIResponse{data=service.LoginResponse} "Login successful"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Invalid credentials"
// @Failure 429 {object} utils.APIResponse "Rate limit exceeded"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/auth/login [post]
func (uc *UserController) Login(c *gin.Context) {
	var req service.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("client_ip", c.ClientIP()).Warn("Invalid login request format")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	// Create response builder for consistent formatting
	rb := utils.NewResponseBuilder(c)

	// Call user service to authenticate
	response, err := uc.userService.Login(c.Request.Context(), &req)
	if err != nil {
		uc.logger.WithError(err).WithFields(logrus.Fields{
			"username":  req.Username,
			"client_ip": c.ClientIP(),
		}).Warn("Login failed")
		rb.Unauthorized("Invalid credentials")
		return
	}

	uc.logger.WithFields(logrus.Fields{
		"user_id":   response.User.ID,
		"username":  response.User.Username,
		"client_ip": c.ClientIP(),
	}).Info("User logged in successfully")

	rb.SuccessWithMessage(response, "Login successful")
}

// Logout godoc
// @Summary User logout
// @Description Invalidate user session and tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse "Logout successful"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/auth/logout [post]
func (uc *UserController) Logout(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	sessionID := c.GetHeader("Session-ID") // Optional session ID for specific session logout

	rb := utils.NewResponseBuilder(c)

	if err := uc.userService.Logout(c.Request.Context(), userID, sessionID); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to logout user")
		rb.InternalServerError("Failed to logout")
		return
	}

	uc.logger.WithField("user_id", userID).Info("User logged out successfully")
	rb.SuccessWithMessage(nil, "Logout successful")
}

// GetProfile godoc
// @Summary Get current user profile
// @Description Get profile information for the authenticated user
// @Tags Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} utils.APIResponse{data=service.UserResponse} "User profile"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/auth/profile [get]
func (uc *UserController) GetProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	user, err := uc.userService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user profile")
		rb.InternalServerError("Failed to retrieve profile")
		return
	}

	// Convert to response format
	userResponse := &service.UserResponse{
		ID:        int64(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	rb.Success(userResponse)
}

// UpdateProfile godoc
// @Summary Update user profile
// @Description Update profile information for the authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.UpdateProfileRequest true "Profile update data"
// @Success 200 {object} utils.APIResponse "Profile updated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 409 {object} utils.APIResponse "Username or email already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/auth/profile [put]
func (uc *UserController) UpdateProfile(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req service.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid profile update request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := uc.userService.UpdateProfile(c.Request.Context(), userID, &req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to update user profile")
		if err.Error() == "username already exists" || err.Error() == "email already exists" {
			rb.Conflict(err.Error())
			return
		}
		rb.InternalServerError("Failed to update profile")
		return
	}

	uc.logger.WithField("user_id", userID).Info("User profile updated successfully")
	rb.SuccessWithMessage(nil, "Profile updated successfully")
}

// ChangePassword godoc
// @Summary Change user password
// @Description Change password for the authenticated user
// @Tags Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.ChangePasswordRequest true "Password change data"
// @Success 200 {object} utils.APIResponse "Password changed successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized or invalid old password"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/auth/password [put]
func (uc *UserController) ChangePassword(c *gin.Context) {
	userID, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		utils.UnauthorizedJSON(c, "Authentication required")
		return
	}

	var req service.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid password change request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := uc.userService.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to change password")
		if err.Error() == "invalid old password" {
			rb.Unauthorized("Invalid old password")
			return
		}
		rb.InternalServerError("Failed to change password")
		return
	}

	uc.logger.WithField("user_id", userID).Info("User password changed successfully")
	rb.SuccessWithMessage(nil, "Password changed successfully")
}

// RefreshToken godoc
// @Summary Refresh access token
// @Description Generate new access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body map[string]string true "Refresh token"
// @Success 200 {object} utils.APIResponse{data=service.TokenResponse} "Token refreshed"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Invalid or expired refresh token"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/auth/refresh [post]
func (uc *UserController) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("client_ip", c.ClientIP()).Warn("Invalid refresh token request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	response, err := uc.userService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		uc.logger.WithError(err).WithField("client_ip", c.ClientIP()).Warn("Token refresh failed")
		rb.Unauthorized("Invalid or expired refresh token")
		return
	}

	uc.logger.WithField("client_ip", c.ClientIP()).Info("Token refreshed successfully")
	rb.Success(response)
}

// User management endpoints (Admin only)

// CreateUser godoc
// @Summary Create a new user
// @Description Create a new user account (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body service.CreateUserRequest true "User creation data"
// @Success 201 {object} utils.APIResponse{data=service.UserResponse} "User created successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 409 {object} utils.APIResponse "Username or email already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users [post]
func (uc *UserController) CreateUser(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).Warn("Invalid create user request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	user, err := uc.userService.CreateUser(c.Request.Context(), &req)
	if err != nil {
		uc.logger.WithError(err).WithField("username", req.Username).Error("Failed to create user")
		if err.Error() == "user with username or email already exists" {
			rb.Conflict("Username or email already exists")
			return
		}
		rb.InternalServerError("Failed to create user")
		return
	}

	// Convert to response format
	userResponse := &service.UserResponse{
		ID:        int64(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	uc.logger.WithField("user_id", user.ID).Info("User created successfully")
	rb.Created(userResponse)
}

// ListUsers godoc
// @Summary List users
// @Description Get paginated list of users with filtering
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Param search query string false "Search by username or email"
// @Param role query string false "Filter by role"
// @Param is_active query boolean false "Filter by active status"
// @Param sort_by query string false "Sort field" default(created_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} utils.APIResponse{data=[]service.UserResponse} "Users list"
// @Failure 400 {object} utils.APIResponse "Invalid request parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users [get]
func (uc *UserController) ListUsers(c *gin.Context) {
	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	role := c.Query("role")
	isActiveStr := c.Query("is_active")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Validate pagination parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Parse boolean parameter
	var isActive *bool
	if isActiveStr != "" {
		if isActiveStr == "true" {
			val := true
			isActive = &val
		} else if isActiveStr == "false" {
			val := false
			isActive = &val
		}
	}

	// Build filter
	filter := &service.UserFilter{
		Search:    search,
		Role:      role,
		IsActive:  isActive,
		SortBy:    sortBy,
		SortOrder: sortOrder,
		Page:      page,
		Limit:     limit,
	}

	rb := utils.NewResponseBuilder(c)

	users, total, err := uc.userService.ListUsers(c.Request.Context(), filter)
	if err != nil {
		uc.logger.WithError(err).Error("Failed to list users")
		rb.InternalServerError("Failed to retrieve users")
		return
	}

	// Convert to response format
	userResponses := make([]*service.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = &service.UserResponse{
			ID:        int64(user.ID),
			Username:  user.Username,
			Email:     user.Email,
			Role:      string(user.Role),
			AvatarURL: user.AvatarURL,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		}
	}

	// Create pagination metadata
	pagination := utils.CreatePagination(page, limit, total)

	rb.SuccessWithPagination(userResponses, pagination)
}

// GetUser godoc
// @Summary Get user by ID
// @Description Get detailed information about a specific user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.APIResponse{data=service.UserResponse} "User details"
// @Failure 400 {object} utils.APIResponse "Invalid user ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "User not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id} [get]
func (uc *UserController) GetUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	user, err := uc.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user")
		rb.NotFound("User not found")
		return
	}

	// Convert to response format
	userResponse := &service.UserResponse{
		ID:        int64(user.ID),
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		AvatarURL: user.AvatarURL,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	rb.Success(userResponse)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update user information (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body service.UpdateUserRequest true "User update data"
// @Success 200 {object} utils.APIResponse "User updated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "User not found"
// @Failure 409 {object} utils.APIResponse "Username or email already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id} [put]
func (uc *UserController) UpdateUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid update user request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := uc.userService.UpdateUser(c.Request.Context(), userID, &req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to update user")
		if err.Error() == "user not found" {
			rb.NotFound("User not found")
			return
		}
		if err.Error() == "username already exists" || err.Error() == "email already exists" {
			rb.Conflict(err.Error())
			return
		}
		rb.InternalServerError("Failed to update user")
		return
	}

	uc.logger.WithField("user_id", userID).Info("User updated successfully")
	rb.SuccessWithMessage(nil, "User updated successfully")
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user account (admin only)
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.APIResponse "User deleted successfully"
// @Failure 400 {object} utils.APIResponse "Invalid user ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "User not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id} [delete]
func (uc *UserController) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	// Deactivate user instead of hard delete for audit purposes
	if err := uc.userService.DeactivateUser(c.Request.Context(), userID); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to delete user")
		if err.Error() == "user not found" {
			rb.NotFound("User not found")
			return
		}
		rb.InternalServerError("Failed to delete user")
		return
	}

	uc.logger.WithField("user_id", userID).Info("User deleted successfully")
	rb.SuccessWithMessage(nil, "User deleted successfully")
}

// ChangeUserPassword godoc
// @Summary Change user password
// @Description Change password for a specific user (admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param request body map[string]string true "New password"
// @Success 200 {object} utils.APIResponse "Password changed successfully"
// @Failure 400 {object} utils.APIResponse "Invalid request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "User not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id}/password [put]
func (uc *UserController) ChangeUserPassword(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	var req struct {
		NewPassword string `json:"new_password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Warn("Invalid password change request")
		utils.BadRequestJSON(c, "Invalid request format: "+err.Error())
		return
	}

	rb := utils.NewResponseBuilder(c)

	// TODO: Implement admin password change in service
	// For now, return not implemented
	rb.Error(http.StatusNotImplemented, "Admin password change not yet implemented")
}

// GetUserSessions godoc
// @Summary Get user active sessions
// @Description Get list of active sessions for a user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.APIResponse{data=[]service.SessionInfo} "User sessions"
// @Failure 400 {object} utils.APIResponse "Invalid user ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "User not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id}/sessions [get]
func (uc *UserController) GetUserSessions(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	sessions, err := uc.userService.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to get user sessions")
		rb.InternalServerError("Failed to retrieve sessions")
		return
	}

	// Convert to response format
	sessionResponses := make([]*service.SessionInfo, len(sessions))
	for i, session := range sessions {
		sessionResponses[i] = &service.SessionInfo{
			ID:        session.ID,
			UserID:    session.UserID,
			IPAddress: session.IPAddress,
			UserAgent: session.UserAgent,
			ExpiresAt: session.ExpiresAt,
			CreatedAt: session.CreatedAt,
		}
	}

	rb.Success(sessionResponses)
}

// RevokeUserSession godoc
// @Summary Revoke user session
// @Description Revoke a specific user session
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param sessionId path string true "Session ID"
// @Success 200 {object} utils.APIResponse "Session revoked successfully"
// @Failure 400 {object} utils.APIResponse "Invalid parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "Session not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id}/sessions/{sessionId} [delete]
func (uc *UserController) RevokeUserSession(c *gin.Context) {
	sessionID := c.Param("sessionId")
	if sessionID == "" {
		utils.BadRequestJSON(c, "Session ID is required")
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := uc.userService.RevokeSession(c.Request.Context(), sessionID); err != nil {
		uc.logger.WithError(err).WithField("session_id", sessionID).Error("Failed to revoke session")
		if err.Error() == "session not found" {
			rb.NotFound("Session not found")
			return
		}
		rb.InternalServerError("Failed to revoke session")
		return
	}

	uc.logger.WithField("session_id", sessionID).Info("Session revoked successfully")
	rb.SuccessWithMessage(nil, "Session revoked successfully")
}

// RevokeAllUserSessions godoc
// @Summary Revoke all user sessions
// @Description Revoke all active sessions for a user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 200 {object} utils.APIResponse "All sessions revoked successfully"
// @Failure 400 {object} utils.APIResponse "Invalid user ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 403 {object} utils.APIResponse "Forbidden"
// @Failure 404 {object} utils.APIResponse "User not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /api/users/{id}/sessions [delete]
func (uc *UserController) RevokeAllUserSessions(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		utils.BadRequestJSON(c, "Invalid user ID")
		return
	}

	rb := utils.NewResponseBuilder(c)

	if err := uc.userService.RevokeAllSessions(c.Request.Context(), userID); err != nil {
		uc.logger.WithError(err).WithField("user_id", userID).Error("Failed to revoke all sessions")
		rb.InternalServerError("Failed to revoke sessions")
		return
	}

	uc.logger.WithField("user_id", userID).Info("All user sessions revoked successfully")
	rb.SuccessWithMessage(nil, "All sessions revoked successfully")
}