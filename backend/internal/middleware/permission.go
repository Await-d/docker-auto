package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"docker-auto/internal/model"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Permission defines different permission levels
type Permission string

const (
	// Basic permissions
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionDelete Permission = "delete"
	PermissionAdmin  Permission = "admin"

	// Resource-specific permissions
	PermissionContainerRead     Permission = "container:read"
	PermissionContainerWrite    Permission = "container:write"
	PermissionContainerDelete   Permission = "container:delete"
	PermissionContainerManage   Permission = "container:manage"

	PermissionImageRead         Permission = "image:read"
	PermissionImageWrite        Permission = "image:write"
	PermissionImageDelete       Permission = "image:delete"

	PermissionUserRead          Permission = "user:read"
	PermissionUserWrite         Permission = "user:write"
	PermissionUserDelete        Permission = "user:delete"
	PermissionUserManage        Permission = "user:manage"

	PermissionSystemRead        Permission = "system:read"
	PermissionSystemWrite       Permission = "system:write"
	PermissionSystemManage      Permission = "system:manage"

	PermissionScheduleRead      Permission = "schedule:read"
	PermissionScheduleWrite     Permission = "schedule:write"
	PermissionScheduleDelete    Permission = "schedule:delete"

	PermissionNotificationRead  Permission = "notification:read"
	PermissionNotificationWrite Permission = "notification:write"
)

// PermissionConfig represents permission middleware configuration
type PermissionConfig struct {
	AllowSelf      bool     // Allow users to access their own resources
	SkipPaths      []string // Paths to skip permission checking
	DefaultDeny    bool     // Default to deny if no permission found
	LogViolations  bool     // Log permission violations
}

// rolePermissions defines permissions for each role
var rolePermissions = map[model.UserRole][]Permission{
	model.UserRoleAdmin: {
		// Admin has all permissions
		PermissionRead, PermissionWrite, PermissionDelete, PermissionAdmin,
		PermissionContainerRead, PermissionContainerWrite, PermissionContainerDelete, PermissionContainerManage,
		PermissionImageRead, PermissionImageWrite, PermissionImageDelete,
		PermissionUserRead, PermissionUserWrite, PermissionUserDelete, PermissionUserManage,
		PermissionSystemRead, PermissionSystemWrite, PermissionSystemManage,
		PermissionScheduleRead, PermissionScheduleWrite, PermissionScheduleDelete,
		PermissionNotificationRead, PermissionNotificationWrite,
	},
	model.UserRoleOperator: {
		// Operator can manage containers and images, read system info
		PermissionRead, PermissionWrite,
		PermissionContainerRead, PermissionContainerWrite, PermissionContainerDelete, PermissionContainerManage,
		PermissionImageRead, PermissionImageWrite, PermissionImageDelete,
		PermissionSystemRead,
		PermissionScheduleRead, PermissionScheduleWrite, PermissionScheduleDelete,
		PermissionNotificationRead, PermissionNotificationWrite,
	},
	model.UserRoleViewer: {
		// Viewer can only read most resources
		PermissionRead,
		PermissionContainerRead,
		PermissionImageRead,
		PermissionSystemRead,
		PermissionScheduleRead,
		PermissionNotificationRead,
	},
}

// PermissionMiddleware creates a permission checking middleware
func PermissionMiddleware(requiredPermission Permission) gin.HandlerFunc {
	return PermissionMiddlewareWithConfig(requiredPermission, &PermissionConfig{
		AllowSelf:     true,
		DefaultDeny:   true,
		LogViolations: true,
	})
}

// PermissionMiddlewareWithConfig creates a permission middleware with configuration
func PermissionMiddlewareWithConfig(requiredPermission Permission, config *PermissionConfig) gin.HandlerFunc {
	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// Skip permission checking for certain paths
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// Get user from context
		user := GetUserFromContext(c)
		if user == nil {
			logrus.WithFields(logrus.Fields{
				"path":       c.Request.URL.Path,
				"method":     c.Request.Method,
				"permission": requiredPermission,
			}).Warn("Permission check failed: no authenticated user")

			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		// Check if user has the required permission
		hasPermission := checkUserPermission(user.Role, requiredPermission)

		if !hasPermission && config.AllowSelf {
			// Check if user is accessing their own resource
			hasPermission = checkSelfAccess(c, user)
		}

		if !hasPermission {
			if config.LogViolations {
				logrus.WithFields(logrus.Fields{
					"user_id":    user.UserID,
					"username":   user.Username,
					"role":       user.Role,
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"permission": requiredPermission,
					"client_ip":  c.ClientIP(),
				}).Warn("Permission denied")
			}

			c.JSON(http.StatusForbidden, utils.ErrorResponseWithDetails(
				http.StatusForbidden,
				"Insufficient permissions",
				[]utils.ErrorDetail{{Message: fmt.Sprintf("Required permission: %s", requiredPermission)}},
			))
			c.Abort()
			return
		}

		// Log successful permission check in debug mode
		logrus.WithFields(logrus.Fields{
			"user_id":    user.UserID,
			"username":   user.Username,
			"role":       user.Role,
			"permission": requiredPermission,
		}).Debug("Permission check passed")

		c.Next()
	}
}

// checkUserPermission checks if a user role has the required permission
func checkUserPermission(userRole model.UserRole, permission Permission) bool {
	permissions, exists := rolePermissions[userRole]
	if !exists {
		return false
	}

	// Check for exact permission match
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	// Check for wildcard permissions
	return checkWildcardPermissions(permissions, permission)
}

// checkWildcardPermissions checks for wildcard permission matches
func checkWildcardPermissions(userPermissions []Permission, requiredPermission Permission) bool {
	requiredStr := string(requiredPermission)

	for _, p := range userPermissions {
		pStr := string(p)

		// Check if user has admin permission (grants everything)
		if p == PermissionAdmin {
			return true
		}

		// Check resource-specific wildcard (e.g., container:* allows container:read)
		if strings.Contains(requiredStr, ":") {
			parts := strings.Split(requiredStr, ":")
			if len(parts) == 2 {
				resourceWildcard := parts[0] + ":*"
				if pStr == resourceWildcard {
					return true
				}

				// Check if user has manage permission for the resource
				managePermission := parts[0] + ":manage"
				if pStr == managePermission {
					return true
				}
			}
		}

		// Check basic permission inheritance
		if requiredPermission == PermissionRead && (p == PermissionWrite || p == PermissionDelete) {
			return true
		}
		if requiredPermission == PermissionWrite && p == PermissionDelete {
			return true
		}
	}

	return false
}

// checkSelfAccess checks if user is accessing their own resource
func checkSelfAccess(c *gin.Context, user *utils.Claims) bool {
	// Extract user ID from URL path (e.g., /api/users/123)
	path := c.Request.URL.Path
	if strings.Contains(path, "/users/") && c.Request.Method == "GET" {
		pathParts := strings.Split(path, "/")
		for i, part := range pathParts {
			if part == "users" && i+1 < len(pathParts) {
				if pathParts[i+1] == fmt.Sprintf("%d", user.UserID) {
					return true
				}
			}
		}
	}

	// Check for user ID in query parameters
	if userIDParam := c.Query("user_id"); userIDParam != "" {
		if userIDParam == fmt.Sprintf("%d", user.UserID) {
			return true
		}
	}

	return false
}


// RequireMinRole creates a middleware that requires a minimum role level
func RequireMinRole(minRole model.UserRole) gin.HandlerFunc {
	roleHierarchy := map[model.UserRole]int{
		model.UserRoleViewer:   1,
		model.UserRoleOperator: 2,
		model.UserRoleAdmin:    3,
	}

	return func(c *gin.Context) {
		user := GetUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		userLevel, userExists := roleHierarchy[user.Role]
		minLevel, minExists := roleHierarchy[minRole]

		if !userExists || !minExists || userLevel < minLevel {
			logrus.WithFields(logrus.Fields{
				"user_id":      user.UserID,
				"user_role":    user.Role,
				"min_role":     minRole,
				"path":         c.Request.URL.Path,
			}).Warn("Minimum role requirement not met")

			c.JSON(http.StatusForbidden, utils.ErrorResponseWithDetails(
				http.StatusForbidden,
				"Insufficient role level",
				[]utils.ErrorDetail{{Message: fmt.Sprintf("Minimum required role: %s", minRole)}},
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAnyRole creates a middleware that requires any of the specified roles
func RequireAnyRole(roles ...model.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			logrus.WithFields(logrus.Fields{
				"user_id":       user.UserID,
				"user_role":     user.Role,
				"allowed_roles": roles,
				"path":          c.Request.URL.Path,
			}).Warn("Role requirement not met")

			c.JSON(http.StatusForbidden, utils.ErrorResponseWithDetails(
				http.StatusForbidden,
				"Insufficient role",
				[]utils.ErrorDetail{{Message: fmt.Sprintf("Required one of roles: %v", roles)}},
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// ResourceOwnershipMiddleware checks if user owns the resource
func ResourceOwnershipMiddleware(resourceParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetUserFromContext(c)
		if user == nil {
			c.JSON(http.StatusUnauthorized, utils.ErrorResponse(http.StatusUnauthorized, "Authentication required"))
			c.Abort()
			return
		}

		// Admin can access all resources
		if user.Role == model.UserRoleAdmin {
			c.Next()
			return
		}

		// Check resource ownership
		resourceID := c.Param(resourceParam)
		if resourceID == "" {
			c.JSON(http.StatusBadRequest, utils.ErrorResponse(http.StatusBadRequest, "Resource ID required"))
			c.Abort()
			return
		}

		// This is a simplified check - in practice, you'd query the database
		// to verify ownership based on the resource type
		userIDStr := fmt.Sprintf("%d", user.UserID)
		if resourceID != userIDStr {
			logrus.WithFields(logrus.Fields{
				"user_id":     user.UserID,
				"resource_id": resourceID,
				"path":        c.Request.URL.Path,
			}).Warn("Resource ownership check failed")

			c.JSON(http.StatusForbidden, utils.ErrorResponse(http.StatusForbidden, "You can only access your own resources"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// Convenience middleware functions

// RequireAdmin requires admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(string(model.UserRoleAdmin))
}

// RequireOperator requires operator role or higher
func RequireOperator() gin.HandlerFunc {
	return RequireMinRole(model.UserRoleOperator)
}

// RequireViewer requires viewer role or higher (basically any authenticated user)
func RequireViewer() gin.HandlerFunc {
	return RequireMinRole(model.UserRoleViewer)
}

// RequireContainerRead requires container read permission
func RequireContainerRead() gin.HandlerFunc {
	return PermissionMiddleware(PermissionContainerRead)
}

// RequireContainerWrite requires container write permission
func RequireContainerWrite() gin.HandlerFunc {
	return PermissionMiddleware(PermissionContainerWrite)
}

// RequireContainerManage requires container management permission
func RequireContainerManage() gin.HandlerFunc {
	return PermissionMiddleware(PermissionContainerManage)
}

// RequireSystemManage requires system management permission
func RequireSystemManage() gin.HandlerFunc {
	return PermissionMiddleware(PermissionSystemManage)
}

// RequireUserManage requires user management permission
func RequireUserManage() gin.HandlerFunc {
	return PermissionMiddleware(PermissionUserManage)
}