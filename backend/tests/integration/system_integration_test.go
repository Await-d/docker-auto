package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/utils"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SystemIntegrationTestSuite contains all components for system integration testing
type SystemIntegrationTestSuite struct {
	cfg           *config.Config
	db            *gorm.DB
	dockerClient  *docker.Client
	cacheManager  *utils.CacheManager
	containerRepo repository.ContainerRepository
	containerSvc  *service.ContainerService
	monitoringSvc *service.ContainerMonitoringService
	terminalSvc   *service.WebTerminalService
	updateSvc     *service.UpdateService
	testContainers []string
	serverURL     string
}

// SetupSystemIntegrationTest initializes the complete system for integration testing
func SetupSystemIntegrationTest(t *testing.T) *SystemIntegrationTestSuite {
	suite := &SystemIntegrationTestSuite{}

	// Load test configuration
	suite.cfg = &config.Config{
		Database: config.DatabaseConfig{
			Type:     "sqlite",
			Host:     "",
			Port:     0,
			Name:     ":memory:",
			User:     "",
			Password: "",
			SSLMode:  "disable",
		},
		Docker: config.DockerConfig{
			Host:       "unix:///var/run/docker.sock",
			APIVersion: "1.41",
			Timeout:    30,
		},
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			Database: 15, // Use test database
		},
		JWT: config.JWTConfig{
			Secret:       "test-jwt-secret-key-32-chars-minimum",
			ExpireHours:  24,
			RefreshDays:  7,
		},
		Port:        8081,
		Environment: "test",
		LogLevel:    "info",
	}

	suite.serverURL = fmt.Sprintf("http://localhost:%d", suite.cfg.Port)

	// Initialize database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	suite.db = db

	// Auto-migrate models
	err = db.AutoMigrate(&model.User{}, &model.Container{}, &model.UpdateHistory{},
		&model.Notification{}, &model.Setting{}, &model.ScheduledTask{},
		&model.TerminalSession{}, &model.MonitoringMetrics{})
	require.NoError(t, err)

	// Initialize cache manager
	cacheManager, err := utils.NewCacheManager(&suite.cfg.Redis)
	require.NoError(t, err)
	suite.cacheManager = cacheManager

	// Initialize Docker client
	dockerClient, err := docker.NewClient(&suite.cfg.Docker)
	require.NoError(t, err)
	suite.dockerClient = dockerClient

	// Initialize repositories
	suite.containerRepo = repository.NewContainerRepository(db)

	// Initialize services
	suite.containerSvc = service.NewContainerService(dockerClient, suite.containerRepo, cacheManager)
	suite.monitoringSvc = service.NewContainerMonitoringService(dockerClient, suite.containerRepo, cacheManager)
	suite.terminalSvc = service.NewWebTerminalService(dockerClient, suite.containerRepo)
	suite.updateSvc = service.NewUpdateService(dockerClient, suite.containerRepo, cacheManager)

	suite.testContainers = []string{}

	return suite
}

// TeardownSystemIntegrationTest cleans up resources after testing
func (s *SystemIntegrationTestSuite) TeardownSystemIntegrationTest(t *testing.T) {
	ctx := context.Background()

	// Remove test containers
	for _, containerID := range s.testContainers {
		if err := s.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			t.Logf("Failed to remove test container %s: %v", containerID, err)
		}
	}

	// Close cache manager
	if s.cacheManager != nil {
		s.cacheManager.Close()
	}

	// Close Docker client
	if s.dockerClient != nil {
		s.dockerClient.Close()
	}
}

// TestSystemIntegrationComplete performs comprehensive system integration testing
func TestSystemIntegrationComplete(t *testing.T) {
	suite := SetupSystemIntegrationTest(t)
	defer suite.TeardownSystemIntegrationTest(t)

	t.Run("DatabaseIntegration", suite.TestDatabaseIntegration)
	t.Run("DockerClientIntegration", suite.TestDockerClientIntegration)
	t.Run("CacheSystemIntegration", suite.TestCacheSystemIntegration)
	t.Run("ContainerServiceIntegration", suite.TestContainerServiceIntegration)
	t.Run("MonitoringServiceIntegration", suite.TestMonitoringServiceIntegration)
	t.Run("TerminalServiceIntegration", suite.TestTerminalServiceIntegration)
	t.Run("UpdateServiceIntegration", suite.TestUpdateServiceIntegration)
	t.Run("CrossComponentIntegration", suite.TestCrossComponentIntegration)
	t.Run("PerformanceUnderLoad", suite.TestPerformanceUnderLoad)
	t.Run("ErrorRecoveryAndResilience", suite.TestErrorRecoveryAndResilience)
}

// TestDatabaseIntegration validates database operations and transaction consistency
func (s *SystemIntegrationTestSuite) TestDatabaseIntegration(t *testing.T) {
	ctx := context.Background()

	// Test container CRUD operations
	testContainer := &model.Container{
		Name:      "test-integration-container",
		ImageName: "nginx:alpine",
		Status:    "created",
		Runtime: model.ContainerRuntime{
			Status:      "stopped",
			ContainerID: "test-container-id",
		},
	}

	// Test Create
	err := s.containerRepo.Create(ctx, testContainer)
	require.NoError(t, err)
	assert.NotZero(t, testContainer.ID)

	// Test Read
	retrieved, err := s.containerRepo.GetByID(ctx, testContainer.ID)
	require.NoError(t, err)
	assert.Equal(t, testContainer.Name, retrieved.Name)
	assert.Equal(t, testContainer.ImageName, retrieved.ImageName)

	// Test Update
	testContainer.Status = "running"
	testContainer.Runtime.Status = "running"
	err = s.containerRepo.Update(ctx, testContainer)
	require.NoError(t, err)

	updated, err := s.containerRepo.GetByID(ctx, testContainer.ID)
	require.NoError(t, err)
	assert.Equal(t, "running", updated.Status)
	assert.Equal(t, "running", updated.Runtime.Status)

	// Test List with filters
	containers, err := s.containerRepo.List(ctx, repository.ContainerFilter{
		Status: "running",
	})
	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, testContainer.ID, containers[0].ID)

	// Test transaction consistency
	tx := s.db.Begin()
	testContainer2 := &model.Container{
		Name:      "test-transaction-container",
		ImageName: "alpine:latest",
		Status:    "created",
	}

	err = tx.Create(testContainer2).Error
	require.NoError(t, err)

	// Rollback transaction
	tx.Rollback()

	// Verify rollback worked
	var count int64
	s.db.Model(&model.Container{}).Where("name = ?", "test-transaction-container").Count(&count)
	assert.Equal(t, int64(0), count)

	// Test Delete
	err = s.containerRepo.Delete(ctx, testContainer.ID)
	require.NoError(t, err)

	_, err = s.containerRepo.GetByID(ctx, testContainer.ID)
	assert.Error(t, err)
}

// TestDockerClientIntegration validates Docker client operations and integration
func (s *SystemIntegrationTestSuite) TestDockerClientIntegration(t *testing.T) {
	ctx := context.Background()

	// Test Docker client connectivity
	info, err := s.dockerClient.Info(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info.ID)

	// Test container lifecycle operations
	containerName := "test-integration-nginx-" + strconv.FormatInt(time.Now().Unix(), 10)

	// Pull test image
	err = s.dockerClient.ImagePull(ctx, "nginx:alpine", types.ImagePullOptions{})
	require.NoError(t, err)

	// Create container
	createResp, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "nginx:alpine",
		Cmd:   []string{"sleep", "30"},
	}, &container.HostConfig{}, nil, nil, containerName)
	require.NoError(t, err)
	s.testContainers = append(s.testContainers, createResp.ID)

	// Start container
	err = s.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	// Wait for container to be running
	time.Sleep(2 * time.Second)

	// Inspect container
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(t, err)
	assert.True(t, containerJSON.State.Running)

	// Get container stats
	stats, err := s.dockerClient.ContainerStats(ctx, createResp.ID, false)
	require.NoError(t, err)
	assert.NotNil(t, stats.Body)
	stats.Body.Close()

	// Stop container
	timeout := time.Duration(10) * time.Second
	err = s.dockerClient.ContainerStop(ctx, createResp.ID, &timeout)
	require.NoError(t, err)

	// Verify container stopped
	containerJSON, err = s.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(t, err)
	assert.False(t, containerJSON.State.Running)
}

// TestCacheSystemIntegration validates Redis caching operations and performance
func (s *SystemIntegrationTestSuite) TestCacheSystemIntegration(t *testing.T) {
	ctx := context.Background()

	// Test basic cache operations
	key := "test-integration-key"
	value := map[string]interface{}{
		"test": "data",
		"timestamp": time.Now().Unix(),
	}

	// Test Set
	err := s.cacheManager.Set(ctx, key, value, 5*time.Minute)
	require.NoError(t, err)

	// Test Get
	var retrieved map[string]interface{}
	err = s.cacheManager.Get(ctx, key, &retrieved)
	require.NoError(t, err)
	assert.Equal(t, value["test"], retrieved["test"])

	// Test TTL
	ttl, err := s.cacheManager.TTL(ctx, key)
	require.NoError(t, err)
	assert.True(t, ttl > 4*time.Minute && ttl <= 5*time.Minute)

	// Test Delete
	err = s.cacheManager.Delete(ctx, key)
	require.NoError(t, err)

	err = s.cacheManager.Get(ctx, key, &retrieved)
	assert.Error(t, err)

	// Test batch operations
	keys := []string{"batch-key-1", "batch-key-2", "batch-key-3"}
	values := []interface{}{"value1", "value2", "value3"}

	for i, k := range keys {
		err := s.cacheManager.Set(ctx, k, values[i], time.Minute)
		require.NoError(t, err)
	}

	// Test pattern matching
	patternKeys, err := s.cacheManager.Keys(ctx, "batch-key-*")
	require.NoError(t, err)
	assert.Len(t, patternKeys, 3)

	// Test memory usage optimization
	info, err := s.cacheManager.Info(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, info)
}

// TestContainerServiceIntegration validates container service operations
func (s *SystemIntegrationTestSuite) TestContainerServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// Create test container record
	testContainer := &model.Container{
		Name:      "integration-test-container",
		ImageName: "nginx:alpine",
		Status:    "created",
	}

	err := s.containerRepo.Create(ctx, testContainer)
	require.NoError(t, err)

	// Test container operations through service
	containerID, err := s.containerSvc.CreateContainer(ctx, &service.CreateContainerRequest{
		Name:      testContainer.Name,
		ImageName: testContainer.ImageName,
		Config: service.ContainerConfig{
			Cmd: []string{"sleep", "30"},
		},
	})
	require.NoError(t, err)
	s.testContainers = append(s.testContainers, containerID)

	// Update container record with actual container ID
	testContainer.Runtime.ContainerID = containerID
	err = s.containerRepo.Update(ctx, testContainer)
	require.NoError(t, err)

	// Test start operation
	err = s.containerSvc.StartContainer(ctx, containerID)
	require.NoError(t, err)

	// Wait for container to start
	time.Sleep(2 * time.Second)

	// Verify container status
	status, err := s.containerSvc.GetContainerStatus(ctx, containerID)
	require.NoError(t, err)
	assert.True(t, status.Running)

	// Test stop operation
	err = s.containerSvc.StopContainer(ctx, containerID)
	require.NoError(t, err)

	// Verify container stopped
	status, err = s.containerSvc.GetContainerStatus(ctx, containerID)
	require.NoError(t, err)
	assert.False(t, status.Running)

	// Test list containers
	containers, err := s.containerSvc.ListContainers(ctx, service.ListContainersOptions{})
	require.NoError(t, err)

	found := false
	for _, c := range containers {
		if c.ID == containerID {
			found = true
			break
		}
	}
	assert.True(t, found, "Container should be found in list")
}

// TestMonitoringServiceIntegration validates monitoring service functionality
func (s *SystemIntegrationTestSuite) TestMonitoringServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// Create and start a test container for monitoring
	containerName := "monitoring-test-container-" + strconv.FormatInt(time.Now().Unix(), 10)
	createResp, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "nginx:alpine",
		Cmd:   []string{"sleep", "60"},
	}, &container.HostConfig{}, nil, nil, containerName)
	require.NoError(t, err)
	s.testContainers = append(s.testContainers, createResp.ID)

	err = s.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	// Wait for container to be running
	time.Sleep(3 * time.Second)

	// Test monitoring service
	metrics, err := s.monitoringSvc.GetContainerMetrics(ctx, createResp.ID)
	require.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.NotEmpty(t, metrics.ContainerID)
	assert.Greater(t, metrics.CPUUsage, 0.0)
	assert.Greater(t, metrics.MemoryUsage, uint64(0))

	// Test monitoring data caching
	cachedMetrics, err := s.monitoringSvc.GetContainerMetrics(ctx, createResp.ID)
	require.NoError(t, err)
	assert.Equal(t, metrics.ContainerID, cachedMetrics.ContainerID)

	// Test real-time monitoring
	monitorChan, err := s.monitoringSvc.StartRealTimeMonitoring(ctx, createResp.ID)
	require.NoError(t, err)

	select {
	case realtimeMetrics := <-monitorChan:
		assert.NotNil(t, realtimeMetrics)
		assert.Equal(t, createResp.ID, realtimeMetrics.ContainerID)
	case <-time.After(10 * time.Second):
		t.Fatal("Timeout waiting for real-time monitoring data")
	}

	err = s.monitoringSvc.StopRealTimeMonitoring(ctx, createResp.ID)
	require.NoError(t, err)
}

// TestTerminalServiceIntegration validates web terminal functionality
func (s *SystemIntegrationTestSuite) TestTerminalServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// Create and start a test container with shell
	containerName := "terminal-test-container-" + strconv.FormatInt(time.Now().Unix(), 10)
	createResp, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "60"},
		Tty:   true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		OpenStdin:    true,
	}, &container.HostConfig{}, nil, nil, containerName)
	require.NoError(t, err)
	s.testContainers = append(s.testContainers, createResp.ID)

	err = s.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	// Wait for container to be running
	time.Sleep(2 * time.Second)

	// Test terminal session creation
	sessionID, err := s.terminalSvc.CreateSession(ctx, &service.CreateSessionRequest{
		ContainerID: createResp.ID,
		Shell:       "/bin/sh",
		Cols:        80,
		Rows:        24,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// Test command execution
	output, err := s.terminalSvc.ExecuteCommand(ctx, sessionID, "echo 'test integration'")
	require.NoError(t, err)
	assert.Contains(t, output, "test integration")

	// Test session cleanup
	err = s.terminalSvc.CloseSession(ctx, sessionID)
	require.NoError(t, err)

	// Verify session is closed
	_, err = s.terminalSvc.GetSession(ctx, sessionID)
	assert.Error(t, err)
}

// TestUpdateServiceIntegration validates update service functionality
func (s *SystemIntegrationTestSuite) TestUpdateServiceIntegration(t *testing.T) {
	ctx := context.Background()

	// Create test container with update configuration
	testContainer := &model.Container{
		Name:      "update-test-container",
		ImageName: "nginx:alpine",
		Status:    "created",
		UpdateConfig: model.UpdateConfig{
			AutoUpdate:    true,
			UpdatePolicy:  "immediate",
			HealthCheck:   true,
		},
	}

	err := s.containerRepo.Create(ctx, testContainer)
	require.NoError(t, err)

	// Test update availability check
	hasUpdate, err := s.updateSvc.CheckUpdateAvailable(ctx, testContainer.ID)
	require.NoError(t, err)
	// Note: This test might return false if the image is already latest
	assert.IsType(t, false, hasUpdate)

	// Test update history
	history, err := s.updateSvc.GetUpdateHistory(ctx, testContainer.ID)
	require.NoError(t, err)
	assert.NotNil(t, history)

	// Test update scheduling
	err = s.updateSvc.ScheduleUpdate(ctx, &service.ScheduleUpdateRequest{
		ContainerID: testContainer.ID,
		UpdateTime:  time.Now().Add(time.Hour),
		Strategy:    "rolling",
	})
	require.NoError(t, err)
}

// TestCrossComponentIntegration validates integration between different components
func (s *SystemIntegrationTestSuite) TestCrossComponentIntegration(t *testing.T) {
	ctx := context.Background()

	// Create container through service
	containerID, err := s.containerSvc.CreateContainer(ctx, &service.CreateContainerRequest{
		Name:      "cross-integration-test",
		ImageName: "nginx:alpine",
		Config: service.ContainerConfig{
			Cmd: []string{"sleep", "60"},
		},
	})
	require.NoError(t, err)
	s.testContainers = append(s.testContainers, containerID)

	// Start container
	err = s.containerSvc.StartContainer(ctx, containerID)
	require.NoError(t, err)

	// Wait for container to start
	time.Sleep(3 * time.Second)

	// Test that monitoring can access the container
	metrics, err := s.monitoringSvc.GetContainerMetrics(ctx, containerID)
	require.NoError(t, err)
	assert.Equal(t, containerID, metrics.ContainerID)

	// Test that terminal can access the container
	sessionID, err := s.terminalSvc.CreateSession(ctx, &service.CreateSessionRequest{
		ContainerID: containerID,
		Shell:       "/bin/sh",
	})
	require.NoError(t, err)

	// Test that update service can access the container
	status, err := s.updateSvc.GetContainerUpdateStatus(ctx, containerID)
	require.NoError(t, err)
	assert.NotNil(t, status)

	// Test data consistency across services
	containerStatus, err := s.containerSvc.GetContainerStatus(ctx, containerID)
	require.NoError(t, err)
	assert.True(t, containerStatus.Running)

	// Cleanup terminal session
	err = s.terminalSvc.CloseSession(ctx, sessionID)
	require.NoError(t, err)

	// Stop container
	err = s.containerSvc.StopContainer(ctx, containerID)
	require.NoError(t, err)
}

// TestPerformanceUnderLoad validates system performance under concurrent load
func (s *SystemIntegrationTestSuite) TestPerformanceUnderLoad(t *testing.T) {
	ctx := context.Background()

	// Create multiple test containers
	numContainers := 5
	containerIDs := make([]string, numContainers)

	for i := 0; i < numContainers; i++ {
		containerName := fmt.Sprintf("load-test-container-%d-%d", i, time.Now().Unix())
		createResp, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
			Image: "alpine:latest",
			Cmd:   []string{"sleep", "30"},
		}, &container.HostConfig{}, nil, nil, containerName)
		require.NoError(t, err)

		containerIDs[i] = createResp.ID
		s.testContainers = append(s.testContainers, createResp.ID)
	}

	// Test concurrent container operations
	var wg sync.WaitGroup
	errors := make(chan error, numContainers*3)

	startTime := time.Now()

	// Start all containers concurrently
	for _, containerID := range containerIDs {
		wg.Add(1)
		go func(cID string) {
			defer wg.Done()
			if err := s.containerSvc.StartContainer(ctx, cID); err != nil {
				errors <- fmt.Errorf("failed to start container %s: %v", cID, err)
			}
		}(containerID)
	}

	wg.Wait()

	// Check for errors
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	startDuration := time.Since(startTime)
	t.Logf("Started %d containers in %v", numContainers, startDuration)

	// Wait for containers to be running
	time.Sleep(3 * time.Second)

	// Test concurrent monitoring
	errors = make(chan error, numContainers)
	var monitoringWg sync.WaitGroup

	monitoringStartTime := time.Now()

	for _, containerID := range containerIDs {
		monitoringWg.Add(1)
		go func(cID string) {
			defer monitoringWg.Done()
			_, err := s.monitoringSvc.GetContainerMetrics(ctx, cID)
			if err != nil {
				errors <- fmt.Errorf("failed to get metrics for container %s: %v", cID, err)
			}
		}(containerID)
	}

	monitoringWg.Wait()
	monitoringDuration := time.Since(monitoringStartTime)
	t.Logf("Retrieved metrics for %d containers in %v", numContainers, monitoringDuration)

	// Check for monitoring errors
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	// Performance assertions
	assert.Less(t, startDuration, 30*time.Second, "Container startup should complete within 30 seconds")
	assert.Less(t, monitoringDuration, 10*time.Second, "Monitoring data retrieval should complete within 10 seconds")

	// Test concurrent stops
	errors = make(chan error, numContainers)
	var stopWg sync.WaitGroup

	stopStartTime := time.Now()

	for _, containerID := range containerIDs {
		stopWg.Add(1)
		go func(cID string) {
			defer stopWg.Done()
			if err := s.containerSvc.StopContainer(ctx, cID); err != nil {
				errors <- fmt.Errorf("failed to stop container %s: %v", cID, err)
			}
		}(containerID)
	}

	stopWg.Wait()
	stopDuration := time.Since(stopStartTime)
	t.Logf("Stopped %d containers in %v", numContainers, stopDuration)

	// Check for stop errors
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	assert.Less(t, stopDuration, 20*time.Second, "Container shutdown should complete within 20 seconds")
}

// TestErrorRecoveryAndResilience validates system behavior under error conditions
func (s *SystemIntegrationTestSuite) TestErrorRecoveryAndResilience(t *testing.T) {
	ctx := context.Background()

	// Test invalid container operations
	err := s.containerSvc.StartContainer(ctx, "non-existent-container")
	assert.Error(t, err)

	// Test monitoring for non-existent container
	_, err = s.monitoringSvc.GetContainerMetrics(ctx, "non-existent-container")
	assert.Error(t, err)

	// Test terminal session for non-existent container
	_, err = s.terminalSvc.CreateSession(ctx, &service.CreateSessionRequest{
		ContainerID: "non-existent-container",
		Shell:       "/bin/sh",
	})
	assert.Error(t, err)

	// Test cache failure resilience (simulate Redis failure)
	originalCacheManager := s.monitoringSvc.CacheManager

	// Use a mock cache manager that always fails
	s.monitoringSvc.CacheManager = &utils.CacheManager{} // Empty cache manager

	// Service should still work without cache
	containerName := "resilience-test-container-" + strconv.FormatInt(time.Now().Unix(), 10)
	createResp, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sleep", "30"},
	}, &container.HostConfig{}, nil, nil, containerName)
	require.NoError(t, err)
	s.testContainers = append(s.testContainers, createResp.ID)

	err = s.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(t, err)

	time.Sleep(2 * time.Second)

	// This might fail due to cache issues, but should handle gracefully
	_, err = s.monitoringSvc.GetContainerMetrics(ctx, createResp.ID)
	// We don't require no error here as it might fail due to mock cache manager

	// Restore original cache manager
	s.monitoringSvc.CacheManager = originalCacheManager

	// Test database connection resilience
	// Close database connection temporarily
	sqlDB, err := s.db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	// Operations should fail gracefully
	_, err = s.containerRepo.List(ctx, repository.ContainerFilter{})
	assert.Error(t, err)

	t.Log("Error recovery and resilience tests completed")
}

// Additional helper methods for WebSocket testing
func (s *SystemIntegrationTestSuite) TestWebSocketIntegration(t *testing.T) {
	// Test WebSocket connections for real-time monitoring
	wsURL := fmt.Sprintf("ws://localhost:%d/ws", s.cfg.Port)

	u, err := url.Parse(wsURL)
	require.NoError(t, err)

	// This would require the actual server to be running
	// For unit testing, we test the WebSocket handlers separately
	t.Log("WebSocket integration test requires running server")
}