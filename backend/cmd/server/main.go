package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/service"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// frontendFS is defined in embed.go or nonembed.go based on build tags

// @title Docker Auto Update System API
// @version 1.0
// @description API for Docker Auto Update System
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.docker-auto.com/support
// @contact.email support@docker-auto.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Set log level from config
	if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
		logger.SetLevel(level)
	}

	logger.Info("Starting Docker Auto Update System...")

	// Initialize database
	db, err := setupDatabase(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to setup database")
	}

	// Initialize cache manager (Redis + Memory cache)
	cacheManager, err := setupRedis(cfg, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to setup cache manager")
	}

	// Ensure proper cleanup on shutdown
	defer func() {
		if err := cacheManager.Close(); err != nil {
			logger.WithError(err).Error("Failed to close cache manager")
		}
	}()

	// Initialize HTTP server
	router := setupRouter(cfg, logger, db, cacheManager)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Server starting on port %d", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func setupDatabase(cfg *config.Config, logger *logrus.Logger) (*gorm.DB, error) {
	logger.Info("Setting up database connection...")

	db, err := utils.InitDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto migrate database
	if err := utils.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Info("Database setup completed")
	return db, nil
}

func setupRedis(cfg *config.Config, logger *logrus.Logger) (*utils.CacheManager, error) {
	logger.Info("Setting up cache manager...")

	// Initialize cache manager with both memory and Redis support
	cacheManager, err := utils.NewCacheManager(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache manager: %w", err)
	}

	// Perform health check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := cacheManager.HealthCheck(ctx); err != nil {
		logger.WithError(err).Warn("Cache health check failed, but continuing with available caches")
	}

	logger.Info("Cache manager setup completed")
	return cacheManager, nil
}

func setupRouter(cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) *gin.Engine {
	// Set gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Setup middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware for frontend development
	router.Use(func(c *gin.Context) {
		method := c.Request.Method

		// Allow requests from development frontend (more permissive for dev)
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control, X-CSRF-Token, X-Device-ID, X-Request-Time")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")

		if method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Setup static file serving from embedded filesystem
	setupStaticFiles(router, logger)

	// Setup basic API routes
	setupBasicAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup dashboard API routes
	setupDashboardAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup WebSocket routes
	setupWebSocketRoutes(router, cfg, logger, db, cacheManager)

	// Setup Container API routes
	setupContainerAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup Image API routes
	setupImageAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup Updates API routes
	setupUpdatesAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup Settings API routes
	setupSettingsAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup Users API routes
	setupUsersAPIRoutes(router, cfg, logger, db, cacheManager)

	// Setup System API routes
	setupSystemAPIRoutes(router, cfg, logger, db, cacheManager)

	return router
}

func setupStaticFiles(router *gin.Engine, logger *logrus.Logger) {
	// Use local filesystem directly for development
	localFrontendPath := "./frontend"
	if _, err := os.Stat(localFrontendPath); os.IsNotExist(err) {
		logger.Warn("No frontend files found at ./frontend")
		return
	}

	// Serve static files directly
	router.Static("/assets", "./frontend/assets")
	router.Static("/js", "./frontend/js")

	// Handle root route
	router.GET("/", func(c *gin.Context) {
		c.File("./frontend/index.html")
	})

	// Handle SPA routing - serve index.html for all non-API routes
	router.NoRoute(func(c *gin.Context) {
		// Skip API routes
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// Try to serve the requested file
		filePath := "./frontend" + c.Request.URL.Path
		if _, err := os.Stat(filePath); err == nil {
			c.File(filePath)
			return
		}

		// Serve index.html for SPA routing
		c.File("./frontend/index.html")
	})

	logger.Info("Static file serving configured with local files")
}

// setupBasicAPIRoutes sets up basic API routes including authentication
func setupBasicAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	api := router.Group("/api")

	// Health check endpoint
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": "2.3.0"})
	})

	// Initialize auth service
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Authentication routes
	auth := api.Group("/auth")
	{
		// Login endpoint with enhanced JWT authentication
		auth.POST("/login", func(c *gin.Context) {
			var loginReq service.LoginRequest

			if err := c.ShouldBindJSON(&loginReq); err != nil {
				c.JSON(400, gin.H{"error": "请求格式无效: " + err.Error()})
				return
			}

			// Set authentication context from request
			if loginReq.IPAddress == "" {
				loginReq.IPAddress = c.ClientIP()
			}
			if loginReq.UserAgent == "" {
				loginReq.UserAgent = c.GetHeader("User-Agent")
			}
			if loginReq.DeviceID == "" {
				loginReq.DeviceID = c.GetHeader("X-Device-ID")
				if loginReq.DeviceID == "" {
					loginReq.DeviceID = "web-" + c.ClientIP() // Generate a default device ID
				}
			}

			// Accept both username and email login
			credential := loginReq.Username
			if credential == "" {
				response := gin.H{
					"code":      400,
					"message":   "用户名不能为空",
					"success":   false,
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				}
				c.JSON(400, response)
				return
			}

			if loginReq.Password == "" {
				response := gin.H{
					"code":      400,
					"message":   "密码不能为空",
					"success":   false,
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				}
				c.JSON(400, response)
				return
			}

			// Perform login
			ctx := c.Request.Context()
			loginResponse, err := authService.Login(ctx, &loginReq)
			if err != nil {
				// Use standard API response format
				response := gin.H{
					"code":      401,
					"message":   err.Error(),
					"success":   false,
					"timestamp": time.Now().UTC().Format(time.RFC3339),
				}
				c.JSON(401, response)
				return
			}

			// Use standard API response format for success
			response := gin.H{
				"code":      200,
				"message":   "登录成功",
				"data": gin.H{
					"user":       loginResponse.User,
					"token_info": loginResponse.TokenInfo,
				},
				"success":   true,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			c.JSON(200, response)
		})

		// Token refresh endpoint
		auth.POST("/refresh", func(c *gin.Context) {
			var refreshReq struct {
				RefreshToken string `json:"refresh_token" binding:"required"`
			}

			if err := c.ShouldBindJSON(&refreshReq); err != nil {
				c.JSON(400, gin.H{"error": "请求格式无效: " + err.Error()})
				return
			}

			// Prepare auth context
			authContext := &utils.AuthContext{
				IPAddress: c.ClientIP(),
				UserAgent: c.GetHeader("User-Agent"),
				DeviceID:  c.GetHeader("X-Device-ID"),
			}

			if authContext.DeviceID == "" {
				authContext.DeviceID = "web-" + c.ClientIP()
			}

			// Refresh token
			ctx := c.Request.Context()
			tokenInfo, err := authService.RefreshToken(ctx, refreshReq.RefreshToken, authContext)
			if err != nil {
				c.JSON(401, gin.H{"error": "Token刷新失败: " + err.Error()})
				return
			}

			c.JSON(200, gin.H{
				"message":    "Token刷新成功",
				"token_info": tokenInfo,
			})
		})

		// Logout endpoint
		auth.POST("/logout", func(c *gin.Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(400, gin.H{"error": "授权头缺失"})
				return
			}

			token, err := utils.ExtractTokenFromHeader(authHeader)
			if err != nil {
				c.JSON(400, gin.H{"error": "无效的授权头格式"})
				return
			}

			// Perform logout
			ctx := c.Request.Context()
			if err := authService.Logout(ctx, token); err != nil {
				logger.WithError(err).Warn("Failed to logout")
				// Don't return error as logout should always succeed from user perspective
			}

			c.JSON(200, gin.H{"message": "登出成功"})
		})

		// Password change endpoint
		auth.POST("/change-password", func(c *gin.Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, gin.H{"error": "未授权"})
				return
			}

			token, err := utils.ExtractTokenFromHeader(authHeader)
			if err != nil {
				c.JSON(401, gin.H{"error": "无效的授权头格式"})
				return
			}

			// Validate token and get user info
			ctx := c.Request.Context()
			claims, err := authService.ValidateToken(ctx, token)
			if err != nil {
				c.JSON(401, gin.H{"error": "无效的访问令牌"})
				return
			}

			var changeReq struct {
				CurrentPassword string `json:"current_password" binding:"required"`
				NewPassword     string `json:"new_password" binding:"required,min=6"`
			}

			if err := c.ShouldBindJSON(&changeReq); err != nil {
				c.JSON(400, gin.H{"error": "请求格式无效: " + err.Error()})
				return
			}

			// Change password
			if err := authService.ChangePassword(ctx, claims.UserID, changeReq.CurrentPassword, changeReq.NewPassword); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, gin.H{"message": "密码修改成功"})
		})

		// User info endpoint
		auth.GET("/me", func(c *gin.Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, gin.H{"error": "未授权"})
				return
			}

			token, err := utils.ExtractTokenFromHeader(authHeader)
			if err != nil {
				c.JSON(401, gin.H{"error": "无效的授权头格式"})
				return
			}

			// Validate token and get user info
			ctx := c.Request.Context()
			claims, err := authService.ValidateToken(ctx, token)
			if err != nil {
				c.JSON(401, gin.H{"error": "无效的访问令牌"})
				return
			}

			c.JSON(200, gin.H{
				"user": gin.H{
					"id":           claims.UserID,
					"username":     claims.Username,
					"email":        claims.Email,
					"role":         claims.Role,
					"is_active":    claims.IsActive,
					"permissions":  claims.Permissions,
					"session_id":   claims.SessionID,
					"last_activity": claims.LastActivity,
				},
			})
		})

		// Update profile endpoint
		auth.PUT("/profile", func(c *gin.Context) {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(401, gin.H{"error": "未授权"})
				return
			}

			token, err := utils.ExtractTokenFromHeader(authHeader)
			if err != nil {
				c.JSON(401, gin.H{"error": "无效的授权头格式"})
				return
			}

			// Validate token and get user info
			ctx := c.Request.Context()
			claims, err := authService.ValidateToken(ctx, token)
			if err != nil {
				c.JSON(401, gin.H{"error": "无效的访问令牌"})
				return
			}

			var updateReq struct {
				Email    string `json:"email"`
				Username string `json:"username"`
			}

			if err := c.ShouldBindJSON(&updateReq); err != nil {
				c.JSON(400, gin.H{"error": "请求格式无效: " + err.Error()})
				return
			}

			// Update user profile (simplified version)
			// In a real implementation, you would update the database
			response := gin.H{
				"code": 200,
				"message": "用户信息更新成功",
				"data": gin.H{
					"id":         claims.UserID,
					"username":   updateReq.Username,
					"email":      updateReq.Email,
					"role":       claims.Role,
					"is_active":  claims.IsActive,
				},
				"success":   true,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			}
			c.JSON(200, response)
		})
	}

	logger.Info("Basic API routes configured")
}

// setupDashboardAPIRoutes sets up dashboard API routes
func setupDashboardAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service for token validation
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Dashboard API group
	dashboard := router.Group("/api/dashboard")

	// Middleware to validate authentication
	dashboard.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	})

	// System overview endpoint
	dashboard.GET("/system-overview", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"cpuUsage":          45.2,
				"memoryUsage":       68.5,
				"diskUsage":         32.1,
				"uptime":            "2 days 15 hours",
				"containersRunning": 8,
				"containersStopped": 2,
			},
		}
		c.JSON(200, response)
	})

	// Container stats endpoint
	dashboard.GET("/container-stats", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"totalContainers":   10,
				"runningContainers": 8,
				"stoppedContainers": 2,
				"pausedContainers":  0,
			},
		}
		c.JSON(200, response)
	})

	// Recent activities endpoint
	dashboard.GET("/recent-activities", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"activities": []map[string]interface{}{},
			},
		}
		c.JSON(200, response)
	})

	// Update activity endpoint
	dashboard.GET("/update-activity", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"recentUpdates":    []map[string]interface{}{},
				"pendingUpdates":   0,
				"lastUpdateTime":   time.Now().UTC().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	// Health status endpoint
	dashboard.GET("/health-status", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"services":       []map[string]interface{}{},
				"overallHealth":  "healthy",
			},
		}
		c.JSON(200, response)
	})

	// Notifications endpoint
	dashboard.GET("/notifications", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"notifications": []map[string]interface{}{},
				"unreadCount":   0,
			},
		}
		c.JSON(200, response)
	})

	// Quick actions endpoint
	dashboard.GET("/quick-actions", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"actions": []map[string]interface{}{},
			},
		}
		c.JSON(200, response)
	})

	// Resource metrics endpoint
	dashboard.GET("/resource-metrics", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"cpuData":    []map[string]interface{}{},
				"memoryData": []map[string]interface{}{},
				"diskData":   []map[string]interface{}{},
			},
		}
		c.JSON(200, response)
	})

	// Realtime metrics endpoint
	dashboard.GET("/realtime-metrics", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"metrics":   []map[string]interface{}{},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	// Security status endpoint
	dashboard.GET("/security-status", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"securityStatus":   "secure",
				"vulnerabilities":  []map[string]interface{}{},
				"lastScan":         time.Now().UTC().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	logger.Info("Dashboard API routes configured")
}

// setupWebSocketRoutes sets up WebSocket routes
func setupWebSocketRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service for token validation
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Configure WebSocket upgrader
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from localhost during development
			origin := r.Header.Get("Origin")
			return origin == "http://localhost:3000" || origin == "http://localhost:8080"
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// WebSocket endpoint
	router.GET("/api/ws", func(c *gin.Context) {
		// Get token from query parameter
		token := c.Query("token")
		if token == "" {
			c.JSON(401, gin.H{"error": "Missing token parameter"})
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "Invalid token"})
			return
		}

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.WithError(err).Error("Failed to upgrade WebSocket connection")
			return
		}
		defer conn.Close()

		logger.WithFields(logrus.Fields{
			"user":    claims.Username,
			"user_id": claims.UserID,
		}).Info("WebSocket connection established")

		// Send welcome message
		welcomeMsg := map[string]interface{}{
			"type": "welcome",
			"data": map[string]interface{}{
				"message": "WebSocket connection established",
				"user":    claims.Username,
				"time":    time.Now().Format(time.RFC3339),
			},
		}

		if err := conn.WriteJSON(welcomeMsg); err != nil {
			logger.WithError(err).Error("Failed to send welcome message")
			return
		}

		// Handle WebSocket messages
		handleWebSocketConnection(conn, claims, logger)
	})

	logger.Info("WebSocket routes configured")
}

// handleWebSocketConnection handles WebSocket connection and messages
func handleWebSocketConnection(conn *websocket.Conn, claims *utils.EnhancedClaims, logger *logrus.Logger) {
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))

	// Set pong handler for keepalive
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Start ping ticker for keepalive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Channel for graceful shutdown
	done := make(chan struct{})

	// Goroutine to handle ping messages
	go func() {
		defer close(done)
		for {
			select {
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.WithError(err).Error("Failed to send ping message")
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read messages from client
	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.WithError(err).Error("WebSocket connection error")
			}
			break
		}

		// Handle different message types
		msgType, ok := msg["type"].(string)
		if !ok {
			logger.Warn("Received message without type field")
			continue
		}

		switch msgType {
		case "ping":
			response := map[string]interface{}{
				"type": "pong",
				"data": map[string]interface{}{
					"timestamp": time.Now().Format(time.RFC3339),
				},
			}
			if err := conn.WriteJSON(response); err != nil {
				logger.WithError(err).Error("Failed to send pong response")
				break
			}

		case "subscribe":
			// Handle subscription requests (placeholder)
			subscription, _ := msg["subscription"].(string)
			response := map[string]interface{}{
				"type": "subscribed",
				"data": map[string]interface{}{
					"subscription": subscription,
					"status":       "active",
				},
			}
			if err := conn.WriteJSON(response); err != nil {
				logger.WithError(err).Error("Failed to send subscription response")
				break
			}
			logger.WithField("subscription", subscription).Info("Client subscribed")

		default:
			logger.WithField("message_type", msgType).Info("Received unknown message type")
		}
	}

	logger.WithField("user", claims.Username).Info("WebSocket connection closed")
}

// setupContainerAPIRoutes sets up container management API routes
func setupContainerAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service for token validation
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Container API group
	containers := router.Group("/api/containers")

	// Middleware to validate authentication
	containers.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	})

	// Get all containers
	containers.GET("", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"containers": []map[string]interface{}{
					{
						"id":     "container1",
						"name":   "nginx-proxy",
						"image":  "nginx:latest",
						"status": "running",
						"ports":  []string{"80:80", "443:443"},
						"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
					},
					{
						"id":     "container2",
						"name":   "mysql-db",
						"image":  "mysql:8.0",
						"status": "running",
						"ports":  []string{"3306:3306"},
						"created_at": time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
						"updated_at": time.Now().Format(time.RFC3339),
					},
				},
				"total": 2,
				"page":  1,
				"limit": 20,
			},
		}
		c.JSON(200, response)
	})

	// Get container by ID
	containers.GET("/:id", func(c *gin.Context) {
		id := c.Param("id")
		response := gin.H{
			"success": true,
			"data": gin.H{
				"id":     id,
				"name":   "nginx-proxy",
				"image":  "nginx:latest",
				"status": "running",
				"ports":  []string{"80:80", "443:443"},
				"env":    []string{"ENV=production"},
				"volumes": []string{"/var/www:/usr/share/nginx/html"},
				"created_at": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	// Start container
	containers.POST("/:id/start", func(c *gin.Context) {
		id := c.Param("id")
		response := gin.H{
			"success": true,
			"message": fmt.Sprintf("Container %s started successfully", id),
		}
		c.JSON(200, response)
	})

	// Stop container
	containers.POST("/:id/stop", func(c *gin.Context) {
		id := c.Param("id")
		response := gin.H{
			"success": true,
			"message": fmt.Sprintf("Container %s stopped successfully", id),
		}
		c.JSON(200, response)
	})

	// Restart container
	containers.POST("/:id/restart", func(c *gin.Context) {
		id := c.Param("id")
		response := gin.H{
			"success": true,
			"message": fmt.Sprintf("Container %s restarted successfully", id),
		}
		c.JSON(200, response)
	})

	// Get container logs
	containers.GET("/:id/logs", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{
				{
					"timestamp": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					"level":     "info",
					"message":   "Server started on port 80",
				},
				{
					"timestamp": time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
					"level":     "info",
					"message":   "Processing request for /api/health",
				},
			},
		}
		c.JSON(200, response)
	})

	// Get container stats
	containers.GET("/:id/stats", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"cpu_percent":    25.5,
				"memory_usage":   "256MB",
				"memory_percent": 12.8,
				"network_io": gin.H{
					"rx_bytes": 1024000,
					"tx_bytes": 512000,
				},
				"disk_io": gin.H{
					"read_bytes":  2048000,
					"write_bytes": 1024000,
				},
				"timestamp": time.Now().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	// Container terminal via WebSocket (NEW)
	containers.GET("/:id/terminal", func(c *gin.Context) {
		id := c.Param("id")

		// Upgrade to WebSocket
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to upgrade to WebSocket"})
			return
		}
		defer conn.Close()

		// Send welcome message
		welcome := fmt.Sprintf("Connected to container %s terminal\r\nroot@%s:~$ ", id, id)
		conn.WriteMessage(websocket.TextMessage, []byte(welcome))

		// Handle WebSocket messages
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			input := string(message)
			var response string

			// Simulate terminal responses
			switch input {
			case "ls":
				response = "bin  dev  etc  home  lib  proc  root  usr  var\r\n"
			case "pwd":
				response = "/root\r\n"
			case "whoami":
				response = "root\r\n"
			case "exit":
				response = "Connection closed\r\n"
				conn.WriteMessage(websocket.TextMessage, []byte(response))
				return
			default:
				if input != "" {
					response = fmt.Sprintf("bash: %s: command not found\r\n", input)
				} else {
					response = ""
				}
			}

			response += fmt.Sprintf("root@%s:~$ ", id)
			if err := conn.WriteMessage(websocket.TextMessage, []byte(response)); err != nil {
				break
			}
		}
	})

	// Container events (NEW)
	containers.GET("/:id/events", func(c *gin.Context) {
		id := c.Param("id")

		// Generate sample events
		events := []map[string]interface{}{
			{
				"id":        "1",
				"type":      "container",
				"action":    "start",
				"time":      time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				"message":   "Container started successfully",
				"level":     "info",
				"actor": map[string]interface{}{
					"ID": id,
					"Attributes": map[string]string{
						"name":  fmt.Sprintf("container-%s", id),
						"image": "nginx:latest",
					},
				},
			},
			{
				"id":        "2",
				"type":      "container",
				"action":    "health_status",
				"time":      time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
				"message":   "Health check passed",
				"level":     "info",
				"actor": map[string]interface{}{
					"ID": id,
					"Attributes": map[string]string{
						"name":  fmt.Sprintf("container-%s", id),
						"image": "nginx:latest",
					},
				},
			},
			{
				"id":        "3",
				"type":      "container",
				"action":    "update_config",
				"time":      time.Now().Add(-15 * time.Minute).Format(time.RFC3339),
				"message":   "Container configuration updated",
				"level":     "info",
				"actor": map[string]interface{}{
					"ID": id,
					"Attributes": map[string]string{
						"name":  fmt.Sprintf("container-%s", id),
						"image": "nginx:latest",
					},
				},
			},
		}

		response := gin.H{
			"success": true,
			"data":    events,
		}
		c.JSON(200, response)
	})

	logger.Info("Container API routes configured")
}

// setupImageAPIRoutes sets up image management API routes
func setupImageAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service for token validation
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Image API group
	images := router.Group("/api/images")

	// Middleware to validate authentication
	images.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	})

	// Get all images
	images.GET("", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{
				{
					"id":         "img1",
					"repository": "nginx",
					"tag":        "latest",
					"size":       "142MB",
					"created_at": time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
				},
				{
					"id":         "img2",
					"repository": "mysql",
					"tag":        "8.0",
					"size":       "521MB",
					"created_at": time.Now().Add(-14 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		}
		c.JSON(200, response)
	})

	// Pull image
	images.POST("/pull", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "Image pull started",
		}
		c.JSON(200, response)
	})

	// Check for image updates
	images.POST("/check-updates", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{
				{
					"image":           "nginx:latest",
					"current_digest":  "sha256:abc123",
					"latest_digest":   "sha256:def456",
					"update_available": true,
				},
			},
		}
		c.JSON(200, response)
	})

	logger.Info("Image API routes configured")
}

// setupUpdatesAPIRoutes sets up update management API routes
func setupUpdatesAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service for token validation
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Updates API group
	updates := router.Group("/api/updates")

	// Middleware to validate authentication
	updates.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	})

	// Check for updates
	updates.GET("/check", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"available_updates": []map[string]interface{}{
					{
						"id":              "update1",
						"container_name":  "nginx-proxy",
						"current_version": "nginx:1.20",
						"latest_version":  "nginx:1.22",
						"size":            "42MB",
						"security_patches": 2,
						"created_at":      time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
					},
				},
				"total": 1,
			},
		}
		c.JSON(200, response)
	})

	// Get update history
	updates.GET("/history", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"updates": []map[string]interface{}{
					{
						"id":              "hist1",
						"container_name":  "mysql-db",
						"from_version":    "mysql:8.0.30",
						"to_version":      "mysql:8.0.32",
						"status":          "completed",
						"started_at":      time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
						"completed_at":    time.Now().Add(-23 * time.Hour).Format(time.RFC3339),
					},
				},
				"total": 1,
				"page":  1,
				"limit": 20,
			},
		}
		c.JSON(200, response)
	})

	// Get running updates
	updates.GET("/running", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{},
		}
		c.JSON(200, response)
	})

	// Get scheduled updates
	updates.GET("/scheduled", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{
				{
					"id":              "sched1",
					"container_name":  "nginx-proxy",
					"scheduled_time":  time.Now().Add(24 * time.Hour).Format(time.RFC3339),
					"update_version":  "nginx:1.22",
					"created_by":      "admin",
				},
			},
		}
		c.JSON(200, response)
	})

	// Get update policies
	updates.GET("/policies", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{
				{
					"id":               "policy1",
					"name":             "Auto Update Production",
					"schedule":         "0 2 * * 0", // Weekly on Sunday at 2 AM
					"auto_rollback":    true,
					"maintenance_mode": true,
					"created_at":       time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		}
		c.JSON(200, response)
	})

	// Create update policy
	updates.POST("/policies", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "Update policy created successfully",
			"data": gin.H{
				"id":               "policy2",
				"name":             "New Policy",
				"schedule":         "0 3 * * 1",
				"auto_rollback":    false,
				"maintenance_mode": false,
				"created_at":       time.Now().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	// Get update settings
	updates.GET("/settings", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"auto_update_enabled":    true,
				"update_check_interval":  "24h",
				"maintenance_window":     "02:00-06:00",
				"auto_rollback_enabled":  true,
				"notification_enabled":   true,
				"backup_before_update":   true,
				"parallel_updates":       2,
				"update_timeout":         "30m",
			},
		}
		c.JSON(200, response)
	})

	// Update settings
	updates.PUT("/settings", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "Update settings saved successfully",
		}
		c.JSON(200, response)
	})

	// Start update
	updates.POST("/:id/start", func(c *gin.Context) {
		id := c.Param("id")
		response := gin.H{
			"success": true,
			"message": fmt.Sprintf("Update %s started successfully", id),
			"data": gin.H{
				"update_id": id,
				"status":    "running",
				"started_at": time.Now().Format(time.RFC3339),
			},
		}
		c.JSON(200, response)
	})

	// Cancel running update
	updates.POST("/running/:id/cancel", func(c *gin.Context) {
		id := c.Param("id")
		response := gin.H{
			"success": true,
			"message": fmt.Sprintf("Update %s cancelled successfully", id),
		}
		c.JSON(200, response)
	})

	// Get update templates
	updates.GET("/templates", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": []map[string]interface{}{
				{
					"id":          "template1",
					"name":        "Standard Update",
					"description": "Standard container update with rollback",
					"steps":       []string{"backup", "pull", "stop", "start", "verify"},
					"created_at":  time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
		}
		c.JSON(200, response)
	})

	// Get notifications
	updates.GET("/notifications", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"notifications": []map[string]interface{}{
					{
						"id":        "notif1",
						"type":      "update_completed",
						"title":     "Update Completed",
						"message":   "Container nginx-proxy updated successfully",
						"read":      false,
						"created_at": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					},
				},
				"total":    1,
				"unread":   1,
			},
		}
		c.JSON(200, response)
	})

	logger.Info("Updates API routes configured")
}

// setupSettingsAPIRoutes sets up system settings API routes
func setupSettingsAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service for token validation
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Settings API group
	settings := router.Group("/api/settings")

	// Middleware to validate authentication and admin role
	settings.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Check admin role for settings access
		if claims.Role != "admin" {
			c.JSON(403, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("permissions", claims.Permissions)
		c.Next()
	})

	// Get system settings
	settings.GET("/system", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"app_name":             "Docker Auto Update System",
				"app_version":          "v2.4.0",
				"timezone":             "Asia/Shanghai",
				"language":             "zh-CN",
				"auto_cleanup":         true,
				"backup_retention":     30,
				"log_level":            "info",
				"max_concurrent_ops":   5,
				"health_check_interval": "30s",
			},
		}
		c.JSON(200, response)
	})

	// Update system settings
	settings.PUT("/system", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "系统设置已更新",
		}
		c.JSON(200, response)
	})

	// Get Docker settings
	settings.GET("/docker", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"docker_host":      "unix:///var/run/docker.sock",
				"registry_mirrors": []string{"https://registry.docker-cn.com"},
				"insecure_registries": []string{},
				"storage_driver":    "overlay2",
				"log_driver":        "json-file",
				"max_log_size":      "10m",
				"max_log_files":     3,
			},
		}
		c.JSON(200, response)
	})

	// Update Docker settings
	settings.PUT("/docker", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "Docker设置已更新",
		}
		c.JSON(200, response)
	})

	// Get notification settings
	settings.GET("/notifications", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"email_enabled":    true,
				"email_smtp_host":  "smtp.gmail.com",
				"email_smtp_port":  587,
				"email_username":   "admin@example.com",
				"webhook_enabled":  false,
				"webhook_url":      "",
				"slack_enabled":    false,
				"slack_webhook":    "",
			},
		}
		c.JSON(200, response)
	})

	// Update notification settings
	settings.PUT("/notifications", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "通知设置已更新",
		}
		c.JSON(200, response)
	})

	// Get security settings
	settings.GET("/security", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"session_timeout":     "24h",
				"password_policy":     "strong",
				"two_factor_enabled":  false,
				"login_attempts":      5,
				"lockout_duration":    "30m",
				"audit_log_enabled":   true,
				"audit_log_retention": 90,
			},
		}
		c.JSON(200, response)
	})

	// Update security settings
	settings.PUT("/security", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "安全设置已更新",
		}
		c.JSON(200, response)
	})

	// Get backup settings
	settings.GET("/backup", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"auto_backup":      true,
				"backup_schedule":  "0 2 * * *", // Daily at 2 AM
				"backup_location":  "/var/backups/docker-auto",
				"retention_days":   30,
				"compression":      true,
				"encrypt_backups":  false,
			},
		}
		c.JSON(200, response)
	})

	// Update backup settings
	settings.PUT("/backup", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "备份设置已更新",
		}
		c.JSON(200, response)
	})

	// Get monitoring settings
	settings.GET("/monitoring", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"data": gin.H{
				"metrics_enabled":     true,
				"metrics_interval":    "30s",
				"prometheus_enabled":  false,
				"prometheus_port":     9090,
				"alert_cpu_threshold": 80,
				"alert_mem_threshold": 85,
				"alert_disk_threshold": 90,
			},
		}
		c.JSON(200, response)
	})

	// Update monitoring settings
	settings.PUT("/monitoring", func(c *gin.Context) {
		response := gin.H{
			"success": true,
			"message": "监控设置已更新",
		}
		c.JSON(200, response)
	})

	logger.Info("Settings API routes configured")
}

// setupUsersAPIRoutes sets up user management API routes
func setupUsersAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// Users API routes with authentication
	users := router.Group("/api/users")
	users.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Check if user is admin for user management
		if claims.Role != "admin" {
			c.JSON(403, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	})
	{
		// List all users
		users.GET("", func(c *gin.Context) {
			response := gin.H{
				"success": true,
				"data": []gin.H{
					{
						"id":          1,
						"username":    "admin",
						"email":       "admin@example.com",
						"role":        "admin",
						"is_active":   true,
						"permissions": []string{"users:read", "users:write", "users:delete", "containers:read", "containers:write", "containers:delete", "images:read", "images:write", "images:delete", "settings:read", "settings:write", "logs:read", "metrics:read", "system:admin", "read"},
						"created_at":  "2025-01-01T00:00:00Z",
						"updated_at":  "2025-01-01T00:00:00Z",
					},
					{
						"id":          2,
						"username":    "operator",
						"email":       "operator@example.com",
						"role":        "operator",
						"is_active":   true,
						"permissions": []string{"containers:read", "containers:write", "images:read", "logs:read", "metrics:read", "read"},
						"created_at":  "2025-01-01T00:00:00Z",
						"updated_at":  "2025-01-01T00:00:00Z",
					},
					{
						"id":          3,
						"username":    "viewer",
						"email":       "viewer@example.com",
						"role":        "viewer",
						"is_active":   true,
						"permissions": []string{"containers:read", "images:read", "logs:read", "metrics:read", "read"},
						"created_at":  "2025-01-01T00:00:00Z",
						"updated_at":  "2025-01-01T00:00:00Z",
					},
				},
			}
			c.JSON(200, response)
		})

		// Create new user
		users.POST("", func(c *gin.Context) {
			var req struct {
				Username string `json:"username" binding:"required"`
				Email    string `json:"email" binding:"required"`
				Password string `json:"password" binding:"required"`
				Role     string `json:"role" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": "请求格式无效: " + err.Error()})
				return
			}

			response := gin.H{
				"success": true,
				"message": "用户创建成功",
				"data": gin.H{
					"id":          4,
					"username":    req.Username,
					"email":       req.Email,
					"role":        req.Role,
					"is_active":   true,
					"created_at":  "2025-01-01T00:00:00Z",
					"updated_at":  "2025-01-01T00:00:00Z",
				},
			}
			c.JSON(201, response)
		})

		// Get specific user
		users.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			response := gin.H{
				"success": true,
				"data": gin.H{
					"id":          id,
					"username":    "user" + id,
					"email":       "user" + id + "@example.com",
					"role":        "viewer",
					"is_active":   true,
					"permissions": []string{"containers:read", "images:read", "logs:read", "metrics:read", "read"},
					"created_at":  "2025-01-01T00:00:00Z",
					"updated_at":  "2025-01-01T00:00:00Z",
				},
			}
			c.JSON(200, response)
		})

		// Update user
		users.PUT("/:id", func(c *gin.Context) {
			id := c.Param("id")
			response := gin.H{
				"success": true,
				"message": "用户信息已更新",
				"data": gin.H{
					"id":         id,
					"updated_at": "2025-01-01T00:00:00Z",
				},
			}
			c.JSON(200, response)
		})

		// Delete user
		users.DELETE("/:id", func(c *gin.Context) {
			id := c.Param("id")
			response := gin.H{
				"success": true,
				"message": "用户已删除",
				"data": gin.H{
					"id": id,
				},
			}
			c.JSON(200, response)
		})
	}

	logger.Info("Users API routes configured")
}

// setupSystemAPIRoutes sets up system management API routes
func setupSystemAPIRoutes(router *gin.Engine, cfg *config.Config, logger *logrus.Logger, db *gorm.DB, cacheManager *utils.CacheManager) {
	// Initialize auth service
	authService := service.NewAuthService(db, cfg, cacheManager, logger)

	// System API routes with authentication
	system := router.Group("/api/system")
	system.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)
		c.Next()
	})
	{
		// Get system configuration
		system.GET("/config", func(c *gin.Context) {
			// Check admin role
			roleRaw, _ := c.Get("userRole")
			userID, _ := c.Get("userID")

			// Convert role to string for comparison
			var roleStr string
			if userRole, ok := roleRaw.(model.UserRole); ok {
				roleStr = string(userRole)
			} else if str, ok := roleRaw.(string); ok {
				roleStr = str
			}

			// Debug: return user info for troubleshooting
			if roleStr != "admin" {
				c.JSON(403, gin.H{
					"error": "需要管理员权限",
					"debug_info": gin.H{
						"current_role": roleStr,
						"current_role_type": fmt.Sprintf("%T", roleRaw),
						"current_user_id": userID,
						"required_role": "admin",
					},
				})
				return
			}

			response := gin.H{
				"success": true,
				"data": gin.H{
					"app_name":        "Docker Auto Update System",
					"version":         "1.0.0",
					"environment":     cfg.Environment,
					"log_level":       cfg.LogLevel,
					"port":            cfg.Port,
					"database_type":   "sqlite",
					"cache_enabled":   false,
					"jwt_expire_hours": cfg.JWT.ExpireHours,
					"auto_update":     true,
					"max_concurrent_updates": 5,
					"update_schedule": "0 2 * * *",
				},
			}
			c.JSON(200, response)
		})

		// Update system configuration
		system.PUT("/config", func(c *gin.Context) {
			// Check admin role
			roleRaw, _ := c.Get("userRole")

			// Convert role to string for comparison
			var roleStr string
			if userRole, ok := roleRaw.(model.UserRole); ok {
				roleStr = string(userRole)
			} else if str, ok := roleRaw.(string); ok {
				roleStr = str
			}

			if roleStr != "admin" {
				c.JSON(403, gin.H{"error": "需要管理员权限"})
				return
			}

			response := gin.H{
				"success": true,
				"message": "系统配置已更新",
			}
			c.JSON(200, response)
		})

		// Export system configuration
		system.POST("/config/export", func(c *gin.Context) {
			// Check admin role
			roleRaw, _ := c.Get("userRole")

			// Convert role to string for comparison
			var roleStr string
			if userRole, ok := roleRaw.(model.UserRole); ok {
				roleStr = string(userRole)
			} else if str, ok := roleRaw.(string); ok {
				roleStr = str
			}

			if roleStr != "admin" {
				c.JSON(403, gin.H{"error": "需要管理员权限"})
				return
			}

			// Create export data
			exportData := gin.H{
				"export_time": time.Now().UTC().Format(time.RFC3339),
				"app_name":    "Docker Auto Update System",
				"version":     "1.0.0",
				"environment": cfg.Environment,
				"configuration": gin.H{
					"log_level":       cfg.LogLevel,
					"port":            cfg.Port,
					"database_type":   "sqlite",
					"cache_enabled":   false,
					"jwt_expire_hours": cfg.JWT.ExpireHours,
					"auto_update":     true,
					"max_concurrent_updates": 5,
					"update_schedule": "0 2 * * *",
				},
				"system_info": gin.H{
					"platform":   "linux/amd64",
					"go_version": "go1.21",
				},
			}

			// Set headers for file download
			c.Header("Content-Type", "application/json")
			c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=docker-auto-config-%s.json", time.Now().Format("20060102-150405")))

			response := gin.H{
				"success": true,
				"data":    exportData,
			}
			c.JSON(200, response)
		})

		// Get system information
		system.GET("/info", func(c *gin.Context) {
			response := gin.H{
				"success": true,
				"data": gin.H{
					"app_name":     "Docker Auto Update System",
					"version":      "1.0.0",
					"build_time":   "2025-01-01T00:00:00Z",
					"go_version":   "go1.21",
					"platform":     "linux/amd64",
					"uptime":       "24h30m",
					"start_time":   "2025-01-01T00:00:00Z",
				},
			}
			c.JSON(200, response)
		})

		// Get system metrics
		system.GET("/metrics", func(c *gin.Context) {
			response := gin.H{
				"success": true,
				"data": gin.H{
					"cpu_usage":       45.2,
					"memory_usage":    67.8,
					"disk_usage":      23.1,
					"network_in":      1024000,
					"network_out":     2048000,
					"active_containers": 12,
					"total_containers":  15,
					"running_updates":   2,
					"pending_updates":   5,
				},
			}
			c.JSON(200, response)
		})
	}

	// Initialize log service
	logService := service.NewLogService(logger)

	// Setup logs API routes (direct implementation in main to avoid circular imports)
	logs := router.Group("/api/logs")
	logs.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "未授权"})
			c.Abort()
			return
		}

		token, err := utils.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的授权头格式"})
			c.Abort()
			return
		}

		// Validate token
		ctx := c.Request.Context()
		claims, err := authService.ValidateToken(ctx, token)
		if err != nil {
			c.JSON(401, gin.H{"error": "无效的访问令牌"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("userID", claims.UserID)
		c.Set("userRole", claims.Role)

		// Check admin role for logs access
		roleStr := string(claims.Role)

		if roleStr != "admin" {
			c.JSON(403, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}
		c.Next()
	})

	{
		// Main logs endpoint
		logs.GET("", func(c *gin.Context) {
			level := c.Query("level")
			limit := c.DefaultQuery("limit", "100")
			sinceStr := c.Query("since")
			component := c.Query("component")
			search := c.Query("search")

			// Parse limit
			limitInt := 100
			if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 1000 {
				limitInt = l
			}

			// Parse since parameter
			var since *time.Time
			if sinceStr != "" {
				if parsed, err := time.Parse(time.RFC3339, sinceStr); err == nil {
					since = &parsed
				}
			}

			// Create log filter
			filter := service.LogFilter{
				Level:     level,
				Component: component,
				Since:     since,
				Search:    search,
				Limit:     limitInt,
			}

			// Get logs using the log service
			systemLogs, err := logService.GetSystemLogs(c.Request.Context(), filter)
			if err != nil {
				logger.WithError(err).Error("Failed to retrieve system logs")
				c.JSON(500, gin.H{"error": "Failed to retrieve system logs"})
				return
			}

			// Return logs in format expected by frontend
			response := gin.H{
				"data":  systemLogs,  // Frontend expects "data" field
				"total": len(systemLogs),
				"page":  1,
				"limit": limitInt,
				"count": len(systemLogs),
				"filter": gin.H{
					"level":     level,
					"component": component,
					"since":     since,
					"search":    search,
					"limit":     limitInt,
				},
				"log_files": logService.GetLogFiles(),
			}

			c.JSON(200, response)
		})

		// Log statistics endpoint
		logs.GET("/stats", func(c *gin.Context) {
			sinceStr := c.DefaultQuery("since", "24h")

			// Parse since parameter
			var since time.Time
			if strings.HasSuffix(sinceStr, "h") {
				if hours, err := strconv.Atoi(strings.TrimSuffix(sinceStr, "h")); err == nil {
					since = time.Now().Add(-time.Duration(hours) * time.Hour)
				} else {
					since = time.Now().Add(-24 * time.Hour)
				}
			} else if strings.HasSuffix(sinceStr, "d") {
				if days, err := strconv.Atoi(strings.TrimSuffix(sinceStr, "d")); err == nil {
					since = time.Now().Add(-time.Duration(days) * 24 * time.Hour)
				} else {
					since = time.Now().Add(-24 * time.Hour)
				}
			} else {
				if parsed, err := time.Parse(time.RFC3339, sinceStr); err == nil {
					since = parsed
				} else {
					since = time.Now().Add(-24 * time.Hour)
				}
			}

			stats, err := logService.GetLogStatistics(c.Request.Context(), since)
			if err != nil {
				logger.WithError(err).Error("Failed to get log statistics")
				c.JSON(500, gin.H{"error": "Failed to get log statistics"})
				return
			}

			c.JSON(200, stats)
		})

		// Log files endpoint
		logs.GET("/files", func(c *gin.Context) {
			logFiles := logService.GetLogFiles()
			var fileInfos []gin.H

			for _, filePath := range logFiles {
				fileInfo := gin.H{
					"path": filePath,
					"name": filepath.Base(filePath),
				}

				// Get file statistics if file exists
				if stat, err := os.Stat(filePath); err == nil {
					fileInfo["size"] = stat.Size()
					fileInfo["modified"] = stat.ModTime()
					fileInfo["exists"] = true
				} else {
					fileInfo["exists"] = false
					fileInfo["size"] = 0
				}

				fileInfos = append(fileInfos, fileInfo)
			}

			response := gin.H{
				"files": fileInfos,
				"count": len(fileInfos),
			}

			c.JSON(200, response)
		})

		// Additional endpoints to match frontend API expectations
		logs.GET("/application", func(c *gin.Context) {
			c.Request.URL.RawQuery += "&component=app"
			// Redirect to main logs endpoint
			c.Redirect(301, "/api/logs?"+c.Request.URL.RawQuery)
		})

		logs.GET("/audit", func(c *gin.Context) {
			c.Request.URL.RawQuery += "&component=audit"
			// Redirect to main logs endpoint
			c.Redirect(301, "/api/logs?"+c.Request.URL.RawQuery)
		})

		logs.GET("/errors", func(c *gin.Context) {
			c.Request.URL.RawQuery += "&level=error"
			// Redirect to main logs endpoint
			c.Redirect(301, "/api/logs?"+c.Request.URL.RawQuery)
		})

		// Search endpoint (same as main logs endpoint with search)
		logs.GET("/search", func(c *gin.Context) {
			// Redirect to main logs endpoint
			c.Redirect(301, "/api/logs?"+c.Request.URL.RawQuery)
		})

		// Export functionality (placeholder)
		logs.GET("/export", func(c *gin.Context) {
			c.JSON(501, gin.H{"error": "Export functionality not yet implemented"})
		})

		// Cleanup functionality (placeholder)
		logs.POST("/cleanup", func(c *gin.Context) {
			c.JSON(501, gin.H{"error": "Cleanup functionality not yet implemented"})
		})
	}

	logger.Info("System API routes configured")
}