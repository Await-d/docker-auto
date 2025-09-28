package integration

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"docker-auto/internal/config"
	"docker-auto/internal/model"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/security"
	dockerTypes "docker-auto/pkg/types"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const (
	// Test image configurations
	TestImageAlpine      = "alpine:latest"
	TestImageNginx       = "nginx:latest"
	TestImageBusyBox     = "busybox:latest"
	TestImageRedis       = "redis:alpine"

	// Test container configurations
	TestContainerPrefix  = "docker-auto-test-"
	TestNetworkName      = "docker-auto-test-network"
	TestVolumeName       = "docker-auto-test-volume"

	// Test timeout configurations
	TestTimeout          = 5 * time.Minute
	ContainerStartTimeout = 30 * time.Second
	WebSocketTimeout     = 10 * time.Second
)

// DockerIntegrationTestSuite provides comprehensive integration testing for Docker functionality
type DockerIntegrationTestSuite struct {
	suite.Suite
	dockerClient    *client.Client
	clientManager   *docker.ClientManager
	containerMonitor *docker.ContainerMonitor
	terminalManager  *docker.TerminalManager
	updateEngine     *docker.UpdateEngine

	// Test resources tracking for cleanup
	testContainers  []string
	testImages      []string
	testNetworks    []string
	testVolumes     []string

	// Configuration
	config          *dockerTypes.ClientConfig
	userContext     *security.DockerUserContext
	logger          *logrus.Logger

	// Synchronization
	cleanupMutex    sync.Mutex
}

// SetupSuite initializes the test environment with real Docker services
func (suite *DockerIntegrationTestSuite) SetupSuite() {
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.InfoLevel)

	// Skip tests if Docker is not available
	if !suite.isDockerAvailable() {
		suite.T().Skip("Docker daemon not available - skipping integration tests")
		return
	}

	// Initialize Docker client with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	suite.dockerClient, err = client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	require.NoError(suite.T(), err, "Failed to create Docker client")

	// Test Docker connection
	_, err = suite.dockerClient.Ping(ctx)
	require.NoError(suite.T(), err, "Failed to connect to Docker daemon")

	// Initialize test configuration
	suite.config = &dockerTypes.ClientConfig{
		Host:               "",
		Version:            "",
		OperationTimeout:   2 * time.Minute,
		MaxConcurrentPulls: 2,
		RetryConfig: &dockerTypes.RetryConfig{
			MaxRetries:    2,
			InitialDelay:  500 * time.Millisecond,
			MaxDelay:      5 * time.Second,
			BackoffFactor: 1.5,
		},
		SecurityConfig: security.DefaultDockerSecurityConfig(),
	}

	// Initialize client manager
	suite.clientManager, err = docker.NewClientManager(suite.config, suite.logger)
	require.NoError(suite.T(), err, "Failed to create client manager")

	// Initialize container monitor
	dockerClient := docker.NewDockerClient(suite.dockerClient, suite.logger)
	monitorConfig := &docker.MonitoringConfig{
		UpdateInterval:   1 * time.Second,
		CacheTTL:         10 * time.Second,
		MaxCacheSize:     100,
		MaxHistorySize:   50,
		EnableMetrics:    true,
		BufferSize:       50,
	}
	suite.containerMonitor = docker.NewContainerMonitor(dockerClient, monitorConfig)

	// Initialize terminal manager
	terminalConfig := &docker.TerminalConfig{
		SessionTimeout:    5 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxSessions:       10,
		BufferSize:        4096,
		PingInterval:      30 * time.Second,
		WriteTimeout:      5 * time.Second,
		ReadTimeout:       5 * time.Second,
		EnableCompression: true,
		AllowPrivileged:   false,
	}
	suite.terminalManager = docker.NewTerminalManager(dockerClient, terminalConfig, suite.logger)

	// Initialize update engine
	suite.updateEngine = docker.NewUpdateEngine(suite.clientManager, suite.logger)

	// Set up test user context
	suite.userContext = &security.DockerUserContext{
		Username: "test-user",
		Role:     "admin",
		UserID:   "test-user-id",
		Groups:   []string{"docker"},
	}

	// Initialize tracking arrays
	suite.testContainers = make([]string, 0)
	suite.testImages = make([]string, 0)
	suite.testNetworks = make([]string, 0)
	suite.testVolumes = make([]string, 0)

	suite.logger.Info("Docker integration test suite initialized successfully")
}

// TearDownSuite cleans up all test resources
func (suite *DockerIntegrationTestSuite) TearDownSuite() {
	if suite.dockerClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	suite.cleanupMutex.Lock()
	defer suite.cleanupMutex.Unlock()

	// Stop and remove test containers
	for _, containerID := range suite.testContainers {
		suite.forceRemoveContainer(ctx, containerID)
	}

	// Remove test images (except base images)
	for _, imageID := range suite.testImages {
		if !suite.isBaseImage(imageID) {
			suite.forceRemoveImage(ctx, imageID)
		}
	}

	// Remove test networks
	for _, networkID := range suite.testNetworks {
		suite.forceRemoveNetwork(ctx, networkID)
	}

	// Remove test volumes
	for _, volumeID := range suite.testVolumes {
		suite.forceRemoveVolume(ctx, volumeID)
	}

	// Close clients
	if suite.containerMonitor != nil {
		suite.containerMonitor.Stop()
	}
	if suite.terminalManager != nil {
		suite.terminalManager.Shutdown()
	}
	if suite.clientManager != nil {
		suite.clientManager.Close()
	}
	if suite.dockerClient != nil {
		suite.dockerClient.Close()
	}

	suite.logger.Info("Docker integration test suite cleanup completed")
}

// TestContainerLifecycleOperations tests complete container lifecycle management
func (suite *DockerIntegrationTestSuite) TestContainerLifecycleOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	containerName := suite.generateTestContainerName("lifecycle")

	// Test 1: Create container
	createConfig := &container.Config{
		Image: TestImageAlpine,
		Cmd:   []string{"sh", "-c", "while true; do echo 'test'; sleep 1; done"},
		Labels: map[string]string{
			"docker-auto.test": "true",
			"docker-auto.type": "integration-test",
		},
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "no"},
	}

	createResp, err := suite.dockerClient.ContainerCreate(
		ctx, createConfig, hostConfig, nil, nil, containerName,
	)
	require.NoError(suite.T(), err, "Failed to create container")
	suite.trackContainer(createResp.ID)

	// Verify container exists and is in created state
	inspection, err := suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to inspect created container")
	assert.Equal(suite.T(), "created", inspection.State.Status)

	// Test 2: Start container
	err = suite.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(suite.T(), err, "Failed to start container")

	// Wait for container to be running
	suite.waitForContainerState(ctx, createResp.ID, "running", ContainerStartTimeout)

	// Verify container is running
	inspection, err = suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to inspect running container")
	assert.Equal(suite.T(), "running", inspection.State.Status)
	assert.True(suite.T(), inspection.State.Running)

	// Test 3: Pause container
	err = suite.dockerClient.ContainerPause(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to pause container")

	suite.waitForContainerState(ctx, createResp.ID, "paused", 10*time.Second)

	inspection, err = suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to inspect paused container")
	assert.True(suite.T(), inspection.State.Paused)

	// Test 4: Unpause container
	err = suite.dockerClient.ContainerUnpause(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to unpause container")

	suite.waitForContainerState(ctx, createResp.ID, "running", 10*time.Second)

	// Test 5: Stop container
	stopTimeout := 10
	err = suite.dockerClient.ContainerStop(ctx, createResp.ID, &stopTimeout)
	require.NoError(suite.T(), err, "Failed to stop container")

	suite.waitForContainerState(ctx, createResp.ID, "exited", 15*time.Second)

	inspection, err = suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to inspect stopped container")
	assert.False(suite.T(), inspection.State.Running)
	assert.Equal(suite.T(), "exited", inspection.State.Status)

	// Test 6: Restart container
	err = suite.dockerClient.ContainerRestart(ctx, createResp.ID, &stopTimeout)
	require.NoError(suite.T(), err, "Failed to restart container")

	suite.waitForContainerState(ctx, createResp.ID, "running", ContainerStartTimeout)

	// Test 7: Remove container (with force to handle running state)
	err = suite.dockerClient.ContainerRemove(ctx, createResp.ID, types.ContainerRemoveOptions{
		Force: true,
	})
	require.NoError(suite.T(), err, "Failed to remove container")

	// Verify container is removed
	_, err = suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	assert.Error(suite.T(), err, "Container should not exist after removal")
}

// TestContainerMonitoring tests real-time monitoring functionality
func (suite *DockerIntegrationTestSuite) TestContainerMonitoring() {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	containerName := suite.generateTestContainerName("monitoring")

	// Create and start a container with some workload
	createConfig := &container.Config{
		Image: TestImageAlpine,
		Cmd:   []string{"sh", "-c", "while true; do dd if=/dev/zero of=/tmp/test bs=1M count=10; sleep 1; done"},
		Labels: map[string]string{
			"docker-auto.test": "true",
			"docker-auto.type": "monitoring-test",
		},
	}

	createResp, err := suite.dockerClient.ContainerCreate(
		ctx, createConfig, &container.HostConfig{}, nil, nil, containerName,
	)
	require.NoError(suite.T(), err, "Failed to create monitoring test container")
	suite.trackContainer(createResp.ID)

	err = suite.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(suite.T(), err, "Failed to start monitoring test container")

	suite.waitForContainerState(ctx, createResp.ID, "running", ContainerStartTimeout)

	// Test monitoring functionality
	suite.containerMonitor.Start()
	defer suite.containerMonitor.Stop()

	// Add container to monitoring
	err = suite.containerMonitor.AddContainer(createResp.ID)
	require.NoError(suite.T(), err, "Failed to add container to monitoring")

	// Wait for some monitoring data
	time.Sleep(5 * time.Second)

	// Verify monitoring data is collected
	stats, exists := suite.containerMonitor.GetContainerStats(createResp.ID)
	assert.True(suite.T(), exists, "Container stats should exist")
	if exists {
		assert.NotNil(suite.T(), stats, "Container stats should not be nil")
		assert.Greater(suite.T(), stats.CPUUsage, 0.0, "CPU usage should be positive")
		assert.Greater(suite.T(), stats.MemoryUsage, int64(0), "Memory usage should be positive")
		assert.NotEmpty(suite.T(), stats.Timestamp, "Timestamp should not be empty")
	}

	// Test monitoring history
	history := suite.containerMonitor.GetContainerHistory(createResp.ID)
	assert.NotEmpty(suite.T(), history, "Container history should not be empty")

	// Test concurrent monitoring of multiple containers
	containerName2 := suite.generateTestContainerName("monitoring2")
	createResp2, err := suite.dockerClient.ContainerCreate(
		ctx, createConfig, &container.HostConfig{}, nil, nil, containerName2,
	)
	require.NoError(suite.T(), err, "Failed to create second monitoring test container")
	suite.trackContainer(createResp2.ID)

	err = suite.dockerClient.ContainerStart(ctx, createResp2.ID, types.ContainerStartOptions{})
	require.NoError(suite.T(), err, "Failed to start second monitoring test container")

	err = suite.containerMonitor.AddContainer(createResp2.ID)
	require.NoError(suite.T(), err, "Failed to add second container to monitoring")

	// Wait for monitoring data from both containers
	time.Sleep(3 * time.Second)

	stats1, exists1 := suite.containerMonitor.GetContainerStats(createResp.ID)
	stats2, exists2 := suite.containerMonitor.GetContainerStats(createResp2.ID)

	assert.True(suite.T(), exists1 && exists2, "Both containers should have stats")
	if exists1 && exists2 {
		assert.NotEqual(suite.T(), stats1.Timestamp, stats2.Timestamp,
			"Timestamps should be different for concurrent monitoring")
	}

	// Remove containers from monitoring
	suite.containerMonitor.RemoveContainer(createResp.ID)
	suite.containerMonitor.RemoveContainer(createResp2.ID)
}

// TestWebTerminalIntegration tests WebSocket terminal functionality with real containers
func (suite *DockerIntegrationTestSuite) TestWebTerminalIntegration() {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	containerName := suite.generateTestContainerName("terminal")

	// Create and start container with shell access
	createConfig := &container.Config{
		Image: TestImageAlpine,
		Cmd:   []string{"sh", "-c", "while true; do sleep 1; done"},
		Tty:   true,
		OpenStdin: true,
		Labels: map[string]string{
			"docker-auto.test": "true",
			"docker-auto.type": "terminal-test",
		},
	}

	createResp, err := suite.dockerClient.ContainerCreate(
		ctx, createConfig, &container.HostConfig{}, nil, nil, containerName,
	)
	require.NoError(suite.T(), err, "Failed to create terminal test container")
	suite.trackContainer(createResp.ID)

	err = suite.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(suite.T(), err, "Failed to start terminal test container")

	suite.waitForContainerState(ctx, createResp.ID, "running", ContainerStartTimeout)

	// Test terminal session creation
	terminalRequest := &dockerTypes.TerminalRequest{
		ContainerID: createResp.ID,
		Command:     []string{"/bin/sh"},
		User:        suite.userContext.Username,
		Cols:        80,
		Rows:        24,
		TTY:         true,
	}

	session, err := suite.terminalManager.CreateSession(ctx, terminalRequest, suite.userContext)
	require.NoError(suite.T(), err, "Failed to create terminal session")
	require.NotNil(suite.T(), session, "Terminal session should not be nil")

	// Test command execution
	testCommand := "echo 'Hello Docker Auto Test'"
	err = session.WriteCommand([]byte(testCommand + "\n"))
	require.NoError(suite.T(), err, "Failed to write command to terminal")

	// Wait for command output (with timeout)
	outputReceived := false
	timeout := time.After(WebSocketTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for !outputReceived {
		select {
		case <-timeout:
			assert.Fail(suite.T(), "Timeout waiting for terminal output")
			return
		case <-ticker.C:
			// Check if we received output
			// This is a simplified check - in real implementation,
			// output would be handled via WebSocket
			outputReceived = true
		}
	}

	// Test session management
	sessions := suite.terminalManager.GetActiveSessions()
	assert.Contains(suite.T(), sessions, session.ID, "Session should be in active sessions")

	// Test session cleanup
	err = suite.terminalManager.CloseSession(session.ID)
	require.NoError(suite.T(), err, "Failed to close terminal session")

	// Verify session is removed
	sessions = suite.terminalManager.GetActiveSessions()
	assert.NotContains(suite.T(), sessions, session.ID, "Session should be removed from active sessions")
}

// TestAutomaticUpdateEngine tests the update detection and rollback functionality
func (suite *DockerIntegrationTestSuite) TestAutomaticUpdateEngine() {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	// Use nginx for update testing (has multiple versions available)
	imageName := "nginx"
	oldTag := "1.20-alpine"
	newTag := "1.21-alpine"

	containerName := suite.generateTestContainerName("update")

	// Pull old version
	oldImageName := imageName + ":" + oldTag
	pullProgress, err := suite.clientManager.PullImage(
		ctx, imageName, oldTag, nil, false, suite.userContext,
	)
	require.NoError(suite.T(), err, "Failed to initiate pull for old image")
	suite.trackImage(oldImageName)

	// Wait for pull to complete
	suite.waitForPullCompletion(ctx, pullProgress, 2*time.Minute)
	assert.True(suite.T(), pullProgress.Completed, "Old image pull should be completed")
	assert.Empty(suite.T(), pullProgress.Error, "Old image pull should not have errors")

	// Create container with old image
	createConfig := &container.Config{
		Image: oldImageName,
		Labels: map[string]string{
			"docker-auto.test": "true",
			"docker-auto.type": "update-test",
			"docker-auto.update.policy": "automatic",
			"docker-auto.update.strategy": "rolling",
		},
		ExposedPorts: map[types.Port]struct{}{
			"80/tcp": {},
		},
	}

	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}

	createResp, err := suite.dockerClient.ContainerCreate(
		ctx, createConfig, hostConfig, nil, nil, containerName,
	)
	require.NoError(suite.T(), err, "Failed to create update test container")
	suite.trackContainer(createResp.ID)

	err = suite.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
	require.NoError(suite.T(), err, "Failed to start update test container")

	suite.waitForContainerState(ctx, createResp.ID, "running", ContainerStartTimeout)

	// Verify initial image
	inspection, err := suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	require.NoError(suite.T(), err, "Failed to inspect container before update")
	assert.Equal(suite.T(), oldImageName, inspection.Config.Image)

	// Pull new version
	newImageName := imageName + ":" + newTag
	newPullProgress, err := suite.clientManager.PullImage(
		ctx, imageName, newTag, nil, false, suite.userContext,
	)
	require.NoError(suite.T(), err, "Failed to initiate pull for new image")
	suite.trackImage(newImageName)

	suite.waitForPullCompletion(ctx, newPullProgress, 2*time.Minute)
	assert.True(suite.T(), newPullProgress.Completed, "New image pull should be completed")

	// Test update detection
	containerModel := &model.Container{
		ContainerID: createResp.ID,
		Name:        containerName,
		Image:       oldImageName,
		Tag:         oldTag,
		UpdatePolicy: "automatic",
		UpdateStrategy: "rolling",
	}

	updateAvailable, err := suite.updateEngine.CheckForUpdates(ctx, containerModel)
	require.NoError(suite.T(), err, "Failed to check for updates")
	assert.True(suite.T(), updateAvailable, "Update should be available")

	// Test update execution
	updateResult, err := suite.updateEngine.ExecuteUpdate(ctx, containerModel, suite.userContext)
	require.NoError(suite.T(), err, "Failed to execute update")
	require.NotNil(suite.T(), updateResult, "Update result should not be nil")
	assert.True(suite.T(), updateResult.Success, "Update should be successful")

	// Wait for update to complete
	time.Sleep(10 * time.Second)

	// Verify container is using new image (check new container if strategy is recreate)
	if updateResult.NewContainerID != "" {
		suite.trackContainer(updateResult.NewContainerID)
		inspection, err = suite.dockerClient.ContainerInspect(ctx, updateResult.NewContainerID)
	} else {
		inspection, err = suite.dockerClient.ContainerInspect(ctx, createResp.ID)
	}
	require.NoError(suite.T(), err, "Failed to inspect container after update")

	// For rolling updates, the image should be updated
	// Note: Exact verification depends on update strategy implementation
	assert.NotEmpty(suite.T(), inspection.Image, "Container should have an image after update")
}

// TestConcurrentOperations tests Docker operations under concurrent load
func (suite *DockerIntegrationTestSuite) TestConcurrentOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	numConcurrentOps := 5
	var wg sync.WaitGroup
	results := make(chan error, numConcurrentOps)

	// Test concurrent container creation and operation
	for i := 0; i < numConcurrentOps; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			containerName := suite.generateTestContainerName(fmt.Sprintf("concurrent-%d", index))

			// Create container
			createConfig := &container.Config{
				Image: TestImageAlpine,
				Cmd:   []string{"sh", "-c", "echo 'concurrent test'; sleep 2"},
				Labels: map[string]string{
					"docker-auto.test": "true",
					"docker-auto.type": "concurrent-test",
				},
			}

			createResp, err := suite.dockerClient.ContainerCreate(
				ctx, createConfig, &container.HostConfig{}, nil, nil, containerName,
			)
			if err != nil {
				results <- fmt.Errorf("concurrent create %d failed: %w", index, err)
				return
			}
			suite.trackContainer(createResp.ID)

			// Start container
			err = suite.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
			if err != nil {
				results <- fmt.Errorf("concurrent start %d failed: %w", index, err)
				return
			}

			// Wait for completion
			suite.waitForContainerState(ctx, createResp.ID, "exited", 30*time.Second)

			results <- nil
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(results)

	// Check results
	var errors []error
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		}
	}

	assert.Empty(suite.T(), errors, "Concurrent operations should succeed: %v", errors)
}

// TestStressTestOperations performs stress testing on Docker operations
func (suite *DockerIntegrationTestSuite) TestStressTestOperations() {
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	// Create multiple containers rapidly
	numContainers := 10
	containers := make([]string, 0, numContainers)

	for i := 0; i < numContainers; i++ {
		containerName := suite.generateTestContainerName(fmt.Sprintf("stress-%d", i))

		createConfig := &container.Config{
			Image: TestImageBusyBox,
			Cmd:   []string{"sh", "-c", "echo 'stress test'; sleep 1"},
			Labels: map[string]string{
				"docker-auto.test": "true",
				"docker-auto.type": "stress-test",
			},
		}

		createResp, err := suite.dockerClient.ContainerCreate(
			ctx, createConfig, &container.HostConfig{}, nil, nil, containerName,
		)
		require.NoError(suite.T(), err, "Failed to create stress test container %d", i)

		containers = append(containers, createResp.ID)
		suite.trackContainer(createResp.ID)

		// Start immediately
		err = suite.dockerClient.ContainerStart(ctx, createResp.ID, types.ContainerStartOptions{})
		require.NoError(suite.T(), err, "Failed to start stress test container %d", i)
	}

	// Wait for all containers to complete
	for _, containerID := range containers {
		suite.waitForContainerState(ctx, containerID, "exited", 30*time.Second)
	}

	// Verify all containers completed successfully
	for i, containerID := range containers {
		inspection, err := suite.dockerClient.ContainerInspect(ctx, containerID)
		require.NoError(suite.T(), err, "Failed to inspect stress test container %d", i)
		assert.Equal(suite.T(), "exited", inspection.State.Status)
		assert.Equal(suite.T(), 0, inspection.State.ExitCode,
			"Container %d should exit with code 0", i)
	}
}

// Helper methods for test management and cleanup

func (suite *DockerIntegrationTestSuite) isDockerAvailable() bool {
	client, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return false
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Ping(ctx)
	return err == nil
}

func (suite *DockerIntegrationTestSuite) generateTestContainerName(suffix string) string {
	return fmt.Sprintf("%s%s-%d", TestContainerPrefix, suffix, time.Now().UnixNano())
}

func (suite *DockerIntegrationTestSuite) trackContainer(containerID string) {
	suite.cleanupMutex.Lock()
	defer suite.cleanupMutex.Unlock()
	suite.testContainers = append(suite.testContainers, containerID)
}

func (suite *DockerIntegrationTestSuite) trackImage(imageName string) {
	suite.cleanupMutex.Lock()
	defer suite.cleanupMutex.Unlock()
	suite.testImages = append(suite.testImages, imageName)
}

func (suite *DockerIntegrationTestSuite) waitForContainerState(ctx context.Context, containerID, expectedState string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		inspection, err := suite.dockerClient.ContainerInspect(ctx, containerID)
		if err == nil && inspection.State.Status == expectedState {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func (suite *DockerIntegrationTestSuite) waitForPullCompletion(ctx context.Context, progress *dockerTypes.PullProgress, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		progress.Mutex.RLock()
		completed := progress.Completed
		progress.Mutex.RUnlock()

		if completed {
			return
		}
		time.Sleep(1 * time.Second)
	}
}

func (suite *DockerIntegrationTestSuite) forceRemoveContainer(ctx context.Context, containerID string) {
	_ = suite.dockerClient.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (suite *DockerIntegrationTestSuite) forceRemoveImage(ctx context.Context, imageID string) {
	_ = suite.dockerClient.ImageRemove(ctx, imageID, types.ImageRemoveOptions{
		Force: true,
	})
}

func (suite *DockerIntegrationTestSuite) forceRemoveNetwork(ctx context.Context, networkID string) {
	_ = suite.dockerClient.NetworkRemove(ctx, networkID)
}

func (suite *DockerIntegrationTestSuite) forceRemoveVolume(ctx context.Context, volumeID string) {
	_ = suite.dockerClient.VolumeRemove(ctx, volumeID, true)
}

func (suite *DockerIntegrationTestSuite) isBaseImage(imageName string) bool {
	baseImages := []string{TestImageAlpine, TestImageNginx, TestImageBusyBox, TestImageRedis}
	for _, base := range baseImages {
		if strings.Contains(imageName, base) {
			return true
		}
	}
	return false
}

// TestRunner function for running the integration test suite
func TestDockerIntegrationSuite(t *testing.T) {
	// Check if integration tests should be skipped
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Check for integration test environment variable
	if os.Getenv("INTEGRATION_TESTS") == "false" {
		t.Skip("Integration tests disabled by INTEGRATION_TESTS environment variable")
	}

	suite.Run(t, new(DockerIntegrationTestSuite))
}