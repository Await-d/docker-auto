package controller

import (
	"time"

	"docker-auto/internal/api"
	"docker-auto/internal/config"
	"docker-auto/internal/middleware"
	"docker-auto/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RouterConfig holds configuration for the router setup
type RouterConfig struct {
	Config              *config.Config
	Logger              *logrus.Logger
	UserService         *service.UserService
	ContainerService    *service.ContainerService
	ImageService        *service.ImageService
	NotificationService *service.NotificationService
	WebSocketManager    *api.WebSocketManager
}

// SetupRoutes configures all API routes with proper middleware chains
func SetupRoutes(router *gin.Engine, cfg *RouterConfig) {
	// Apply global middleware
	setupGlobalMiddleware(router, cfg)

	// Setup API routes
	setupAPIRoutes(router, cfg)

	// Setup health check and metrics routes (no auth required)
	setupPublicRoutes(router, cfg)
}

// setupGlobalMiddleware configures global middleware that applies to all routes
func setupGlobalMiddleware(router *gin.Engine, cfg *RouterConfig) {
	// Request ID middleware for tracing
	router.Use(middleware.RequestIDMiddleware())

	// Logger middleware
	router.Use(middleware.LoggerMiddleware(cfg.Logger))

	// CORS middleware
	corsConfig := middleware.CORSConfig{
		AllowedOrigins:   []string{"*"}, // Configure based on environment
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(middleware.CORSMiddleware(corsConfig))

	// Error handling middleware
	router.Use(middleware.ErrorHandlerMiddleware(cfg.Logger))

	// Recovery middleware
	router.Use(gin.Recovery())
}

// setupPublicRoutes configures routes that don't require authentication
func setupPublicRoutes(router *gin.Engine, cfg *RouterConfig) {
	systemController := NewSystemController(cfg.Logger)

	// Health check endpoint (public)
	router.GET("/health", systemController.HealthCheck)
	router.GET("/api/health", systemController.HealthCheck)

	// API info endpoint (public)
	router.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"name":    "Docker Auto Update System API",
			"version": "v1",
			"status":  "running",
		})
	})
}

// setupAPIRoutes configures all authenticated API routes
func setupAPIRoutes(router *gin.Engine, cfg *RouterConfig) {
	// Create API v1 group
	api := router.Group("/api")

	// Apply rate limiting to API endpoints
	rateLimitConfig := middleware.RateLimitConfig{
		Requests: 100, // requests per window
		Window:   time.Minute,
		KeyFunc:  middleware.ClientIPKeyFunc,
	}
	api.Use(middleware.RateLimitMiddleware(rateLimitConfig))

	// Setup authentication routes (no auth required)
	setupAuthRoutes(api, cfg)

	// Setup authenticated routes
	setupAuthenticatedRoutes(api, cfg)
}

// setupAuthRoutes configures authentication-related routes
func setupAuthRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	userController := NewUserController(cfg.UserService, cfg.Logger)

	auth := api.Group("/auth")
	{
		// Public authentication endpoints
		auth.POST("/login", userController.Login)
		auth.POST("/refresh", userController.RefreshToken)

		// Authenticated endpoints
		authRequired := auth.Group("")
		authRequired.Use(middleware.JWTAuthMiddleware(cfg.Config.JWT.Secret))
		{
			authRequired.POST("/logout", userController.Logout)
			authRequired.GET("/profile", userController.GetProfile)
			authRequired.PUT("/profile", userController.UpdateProfile)
			authRequired.PUT("/password", userController.ChangePassword)
		}
	}
}

// setupAuthenticatedRoutes configures all routes that require authentication
func setupAuthenticatedRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	// All routes below require authentication
	protected := api.Group("")
	protected.Use(middleware.JWTAuthMiddleware(cfg.Config.JWT.Secret))
	protected.Use(middleware.RequireActiveUser())

	// Setup individual route groups
	setupUserRoutes(protected, cfg)
	setupContainerRoutes(protected, cfg)
	setupImageRoutes(protected, cfg)
	setupUpdateRoutes(protected, cfg)
	setupSystemRoutes(protected, cfg)
	setupRegistryRoutes(protected, cfg)
	setupNotificationRoutes(protected, cfg)
	setupWebSocketRoutes(api, cfg)
}

// setupUserRoutes configures user management routes
func setupUserRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	userController := NewUserController(cfg.UserService, cfg.Logger)

	users := api.Group("/users")
	{
		// User listing and creation (admin only)
		users.GET("", middleware.RequireAdmin(), userController.ListUsers)
		users.POST("", middleware.RequireAdmin(), userController.CreateUser)

		// Individual user operations
		userRoutes := users.Group("/:id")
		{
			// Read operations (allow self-access)
			userRoutes.GET("",
				middleware.PermissionMiddlewareWithConfig(
					middleware.PermissionUserRead,
					&middleware.PermissionConfig{AllowSelf: true, LogViolations: true},
				),
				userController.GetUser)

			// Admin-only operations
			userRoutes.PUT("", middleware.RequireAdmin(), userController.UpdateUser)
			userRoutes.DELETE("", middleware.RequireAdmin(), userController.DeleteUser)
			userRoutes.PUT("/password", middleware.RequireAdmin(), userController.ChangeUserPassword)

			// Session management
			userRoutes.GET("/sessions", middleware.RequireAdmin(), userController.GetUserSessions)
			userRoutes.DELETE("/sessions", middleware.RequireAdmin(), userController.RevokeAllUserSessions)
			userRoutes.DELETE("/sessions/:sessionId", middleware.RequireAdmin(), userController.RevokeUserSession)
		}
	}
}

// setupContainerRoutes configures container management routes
func setupContainerRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	containerController := NewContainerController(cfg.ContainerService, cfg.Logger)

	containers := api.Group("/containers")
	{
		// Container listing and creation
		containers.GET("", middleware.RequireContainerRead(), containerController.ListContainers)
		containers.POST("", middleware.RequireContainerWrite(), containerController.CreateContainer)

		// Bulk operations
		containers.POST("/bulk", middleware.RequireContainerManage(), containerController.BulkContainerOperation)
		containers.POST("/sync", middleware.RequireOperator(), containerController.SyncContainerStatus)

		// Individual container operations
		containerRoutes := containers.Group("/:id")
		{
			// Read operations
			containerRoutes.GET("", middleware.RequireContainerRead(), containerController.GetContainer)
			containerRoutes.GET("/status", middleware.RequireContainerRead(), containerController.GetContainerStatus)
			containerRoutes.GET("/logs", middleware.RequireContainerRead(), containerController.GetContainerLogs)
			containerRoutes.GET("/stats", middleware.RequireContainerRead(), containerController.GetContainerStats)

			// Write operations
			containerRoutes.PUT("", middleware.RequireContainerWrite(), containerController.UpdateContainer)
			containerRoutes.DELETE("", middleware.RequireContainerManage(), containerController.DeleteContainer)

			// Container control operations
			containerRoutes.POST("/start", middleware.RequireContainerManage(), containerController.StartContainer)
			containerRoutes.POST("/stop", middleware.RequireContainerManage(), containerController.StopContainer)
			containerRoutes.POST("/restart", middleware.RequireContainerManage(), containerController.RestartContainer)
			containerRoutes.POST("/update", middleware.RequireContainerManage(), containerController.UpdateContainerImage)
		}
	}
}

// setupImageRoutes configures image management routes
func setupImageRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	imageController := NewImageController(cfg.ImageService, cfg.Logger)

	images := api.Group("/images")
	{
		// Image listing and search
		images.GET("", middleware.RequireViewer(), imageController.ListImages)
		images.GET("/search", middleware.RequireViewer(), imageController.SearchImages)

		// Update checking operations
		images.POST("/check-updates", middleware.RequireOperator(), imageController.CheckUpdates)
		images.GET("/update-info", middleware.RequireViewer(), imageController.GetImageUpdateInfo)
		images.POST("/schedule-check", middleware.RequireOperator(), imageController.ScheduleImageCheck)

		// Image comparison
		images.POST("/compare", middleware.RequireViewer(), imageController.CompareImageVersions)

		// Individual image operations (using URL-encoded image names)
		imageRoutes := images.Group("/:name")
		{
			// Read operations
			imageRoutes.GET("/info", middleware.RequireViewer(), imageController.GetImageInfo)
			imageRoutes.GET("/versions", middleware.RequireViewer(), imageController.GetImageVersions)
			imageRoutes.GET("/security", middleware.RequireViewer(), imageController.GetImageSecurityIssues)

			// Write operations (admin/operator only)
			imageRoutes.POST("/pull", middleware.RequireOperator(), imageController.PullImage)
			imageRoutes.POST("/refresh", middleware.RequireOperator(), imageController.RefreshImageCache)
			imageRoutes.DELETE("", middleware.RequireOperator(), imageController.RemoveImage)
		}
	}
}

// setupUpdateRoutes configures update management routes
func setupUpdateRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	updateController := NewUpdateController(cfg.ContainerService, cfg.ImageService, cfg.Logger)

	updates := api.Group("/updates")
	{
		// Update history and status
		updates.GET("/history", middleware.RequireViewer(), updateController.GetUpdateHistory)
		updates.GET("/status", middleware.RequireViewer(), updateController.GetUpdateStatus)
		updates.GET("/metrics", middleware.RequireViewer(), updateController.GetUpdateMetrics)
		updates.GET("/available", middleware.RequireViewer(), updateController.CheckAvailableUpdates)

		// Update operations
		updates.POST("/batch", middleware.RequireOperator(), updateController.TriggerBatchUpdate)
		updates.POST("/schedule", middleware.RequireOperator(), updateController.ScheduleUpdate)

		// Individual update operations
		updateRoutes := updates.Group("/:id")
		{
			updateRoutes.GET("", middleware.RequireViewer(), updateController.GetUpdateDetails)
			updateRoutes.POST("/cancel", middleware.RequireOperator(), updateController.CancelUpdate)
		}

		// Rollback operations
		updates.POST("/rollback/:id", middleware.RequireOperator(), updateController.RollbackUpdate)
	}
}

// setupSystemRoutes configures system management routes
func setupSystemRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	systemController := NewSystemController(cfg.Logger)

	system := api.Group("/system")
	{
		// System information (viewer access)
		system.GET("/info", middleware.RequireViewer(), systemController.GetSystemInfo)
		system.GET("/health", systemController.HealthCheck) // No auth required for health
		system.GET("/metrics", middleware.RequireViewer(), systemController.GetSystemMetrics)

		// System configuration (admin only)
		system.GET("/config", middleware.RequireAdmin(), systemController.GetSystemConfig)
		system.PUT("/config", middleware.RequireAdmin(), systemController.UpdateSystemConfig)

		// System operations (admin only)
		system.POST("/restart", middleware.RequireAdmin(), systemController.RestartService)
		system.GET("/logs", middleware.RequireAdmin(), systemController.GetLogs)
	}
}

// setupRegistryRoutes configures registry management routes
func setupRegistryRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	registryController := NewRegistryController(cfg.ImageService, cfg.Logger)

	registries := api.Group("/registries")
	{
		// Registry listing and creation
		registries.GET("", middleware.RequireViewer(), registryController.ListRegistries)
		registries.POST("", middleware.RequireAdmin(), registryController.CreateRegistry)

		// Individual registry operations
		registryRoutes := registries.Group("/:id")
		{
			// Read operations
			registryRoutes.GET("", middleware.RequireViewer(), registryController.GetRegistry)
			registryRoutes.GET("/info", middleware.RequireViewer(), registryController.GetRegistryInfo)
			registryRoutes.GET("/stats", middleware.RequireViewer(), registryController.GetRegistryStatistics)
			registryRoutes.GET("/search", middleware.RequireViewer(), registryController.SearchRegistryImages)

			// Write operations (admin only)
			registryRoutes.PUT("", middleware.RequireAdmin(), registryController.UpdateRegistry)
			registryRoutes.DELETE("", middleware.RequireAdmin(), registryController.DeleteRegistry)

			// Registry operations
			registryRoutes.POST("/test", middleware.RequireOperator(), registryController.TestRegistryConnection)
			registryRoutes.POST("/default", middleware.RequireAdmin(), registryController.SetDefaultRegistry)
		}
	}
}

// Additional helper functions for advanced routing features

// SetupWebhookRoutes configures webhook endpoints (if needed)
func SetupWebhookRoutes(router *gin.Engine, cfg *RouterConfig) {
	webhooks := router.Group("/webhooks")
	{
		// Registry webhooks for push notifications
		webhooks.POST("/registry/:registryId", func(c *gin.Context) {
			// Handle registry push webhooks
			c.JSON(200, gin.H{"status": "received"})
		})

		// Docker Hub webhook
		webhooks.POST("/dockerhub", func(c *gin.Context) {
			// Handle Docker Hub push notifications
			c.JSON(200, gin.H{"status": "received"})
		})
	}
}

// SetupSwaggerRoutes configures Swagger documentation routes
func SetupSwaggerRoutes(router *gin.Engine, cfg *RouterConfig) {
	// Only enable in development environment
	if cfg.Config.Environment == "development" {
		// Swagger documentation would be served here
		// router.GET("/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		router.GET("/docs", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "API documentation not yet implemented",
				"api_info": gin.H{
					"name":    "Docker Auto Update System API",
					"version": "v1",
					"base_path": "/api",
				},
			})
		})
	}
}

// SetupDevRoutes configures development-only routes
func SetupDevRoutes(router *gin.Engine, cfg *RouterConfig) {
	if cfg.Config.Environment == "development" {
		dev := router.Group("/dev")
		{
			// Development endpoints for testing
			dev.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "pong"})
			})

			dev.POST("/echo", func(c *gin.Context) {
				var body interface{}
				c.ShouldBindJSON(&body)
				c.JSON(200, gin.H{"echo": body})
			})
		}
	}
}

// SetupCompleteRouter sets up the complete router with all routes and middleware
func SetupCompleteRouter(cfg *RouterConfig) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Config.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Setup main routes
	SetupRoutes(router, cfg)

	// Setup additional routes based on configuration
	if cfg.Config.Environment == "development" {
		SetupSwaggerRoutes(router, cfg)
		SetupDevRoutes(router, cfg)
	}

	// Setup webhook routes if webhooks are enabled
	// SetupWebhookRoutes(router, cfg)

	cfg.Logger.Info("Router setup completed with all endpoints configured")

	return router
}

// setupNotificationRoutes configures notification management routes
func setupNotificationRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	notificationController := NewNotificationController(cfg.NotificationService, cfg.Logger)

	notifications := api.Group("/notifications")
	{
		// Notification listing and statistics
		notifications.GET("", middleware.RequireViewer(), notificationController.GetNotifications)
		notifications.GET("/unread", middleware.RequireViewer(), notificationController.GetUnreadNotifications)
		notifications.GET("/count", middleware.RequireViewer(), notificationController.GetNotificationCount)
		notifications.GET("/stats", middleware.RequireViewer(), notificationController.GetNotificationStats)

		// Notification management
		notifications.POST("/mark-read", middleware.RequireViewer(), notificationController.MarkAsRead)
		notifications.POST("/mark-all-read", middleware.RequireViewer(), notificationController.MarkAllAsRead)

		// Individual notification operations
		notificationRoutes := notifications.Group("/:id")
		{
			notificationRoutes.GET("", middleware.RequireViewer(), notificationController.GetNotification)
			notificationRoutes.PUT("/read", middleware.RequireViewer(), notificationController.MarkNotificationAsRead)
			notificationRoutes.DELETE("", middleware.RequireViewer(), notificationController.DeleteNotification)
		}

		// Admin-only operations
		adminNotifications := notifications.Group("/admin")
		adminNotifications.Use(middleware.RequireAdmin())
		{
			adminNotifications.POST("/broadcast", notificationController.BroadcastNotification)
			adminNotifications.GET("/templates", notificationController.GetNotificationTemplates)
			adminNotifications.POST("/templates", notificationController.CreateNotificationTemplate)
			adminNotifications.DELETE("/cleanup", notificationController.CleanupOldNotifications)
		}
	}
}

// setupWebSocketRoutes configures WebSocket routes
func setupWebSocketRoutes(api *gin.RouterGroup, cfg *RouterConfig) {
	// WebSocket endpoint for real-time communication
	// Note: WebSocket upgrade happens without standard JWT middleware
	// Authentication is handled within the WebSocket handler via query parameter
	api.GET("/ws", cfg.WebSocketManager.HandleWebSocket)

	// WebSocket management endpoints (authenticated)
	wsManagement := api.Group("/ws")
	wsManagement.Use(middleware.JWTAuthMiddleware(cfg.Config.JWT.Secret))
	wsManagement.Use(middleware.RequireActiveUser())
	{
		wsManagement.GET("/stats", middleware.RequireAdmin(), func(c *gin.Context) {
			stats := cfg.WebSocketManager.GetStats()
			c.JSON(200, gin.H{"data": stats})
		})

		wsManagement.GET("/connections", middleware.RequireAdmin(), func(c *gin.Context) {
			connections := cfg.WebSocketManager.GetConnections()
			connectionData := make([]gin.H, len(connections))
			for i, conn := range connections {
				connectionData[i] = gin.H{
					"id":         conn.ID,
					"user_id":    conn.UserID,
					"last_ping":  conn.LastPing,
					"connected":  !conn.IsClosed(),
				}
			}
			c.JSON(200, gin.H{"data": connectionData})
		})

		wsManagement.POST("/cleanup", middleware.RequireAdmin(), func(c *gin.Context) {
			cfg.WebSocketManager.CleanupInactiveConnections()
			c.JSON(200, gin.H{"message": "Cleanup completed"})
		})
	}
}