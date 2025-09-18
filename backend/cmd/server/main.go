package main

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"docker-auto/internal/config"
	"docker-auto/pkg/utils"

	"github.com/gin-gonic/gin"
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
	_, err = setupDatabase(cfg, logger)
	if err != nil {
		logger.Fatalf("Failed to setup database: %v", err)
	}

	// Initialize Redis (TODO: Implement Redis setup when needed)
	// redisClient, err := setupRedis(cfg, logger)
	// if err != nil {
	//	logger.Fatalf("Failed to setup Redis: %v", err)
	// }

	// Initialize repositories (TODO: Implement repository manager)
	// repos := repository.NewRepositories(db, redisClient)

	// Initialize services (TODO: Implement service manager)
	// services := service.NewServices(repos, cfg, logger)

	// Initialize HTTP server
	router := setupRouter(cfg, logger)

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

func setupRedis(cfg *config.Config, logger *logrus.Logger) (interface{}, error) {
	logger.Info("Setting up Redis connection...")

	// TODO: Implement Redis initialization when needed
	// redisClient, err := utils.InitRedis(cfg)
	// if err != nil {
	//	return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	// }

	logger.Info("Redis setup completed (placeholder)")
	return nil, nil
}

func setupRouter(cfg *config.Config, logger *logrus.Logger) *gin.Engine {
	// Set gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Setup middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Setup API routes
	apiGroup := router.Group("/api/v1")
	{
		// Health check endpoint
		apiGroup.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok", "version": "2.3.0"})
		})

		// TODO: Add other API routes here
		// apiGroup.GET("/containers", handlers.GetContainers)
		// apiGroup.POST("/containers/:id/update", handlers.UpdateContainer)
	}

	// Setup static file serving from embedded filesystem
	setupStaticFiles(router, logger)

	return router
}

func setupStaticFiles(router *gin.Engine, logger *logrus.Logger) {
	// Get the embedded filesystem for the dist directory
	distFS, err := fs.Sub(frontendFS, "frontend/dist")
	if err != nil {
		logger.Warnf("Failed to create sub filesystem for frontend: %v", err)
		return
	}

	// Serve static files
	router.StaticFS("/assets", http.FS(distFS))

	// Handle SPA routing - serve index.html for all non-API routes
	router.NoRoute(func(c *gin.Context) {
		// Skip API routes
		if c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// Try to serve the requested file
		file, err := distFS.Open(c.Request.URL.Path[1:]) // Remove leading slash
		if err == nil {
			defer file.Close()
			stat, err := file.Stat()
			if err == nil && !stat.IsDir() {
				c.FileFromFS(c.Request.URL.Path[1:], http.FS(distFS))
				return
			}
		}

		// Serve index.html for SPA routing
		c.FileFromFS("index.html", http.FS(distFS))
	})

	logger.Info("Static file serving configured with embedded frontend")
}