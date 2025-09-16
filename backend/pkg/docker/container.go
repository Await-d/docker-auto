package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"

	"docker-auto/internal/model"
)

// Container lifecycle operations

// CreateContainer creates a new Docker container with the specified configuration
func (d *DockerClient) CreateContainer(ctx context.Context, config *ContainerCreateConfig) (*container.CreateResponse, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if config == nil {
		return nil, fmt.Errorf("container config cannot be nil")
	}

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid container config: %w", err)
	}

	// Convert to Docker API types
	containerConfig, hostConfig, networkingConfig, err := config.ToDockerConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to convert config: %w", err)
	}

	// Create container
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, config.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	return &resp, nil
}

// StartContainer starts a Docker container by ID
func (d *DockerClient) StartContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	err := d.client.ContainerStart(ctx, containerID, types.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}

	return nil
}

// StopContainer stops a Docker container by ID with optional timeout
func (d *DockerClient) StopContainer(ctx context.Context, containerID string, timeout *int) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	var stopTimeout *int
	if timeout != nil {
		stopTimeout = timeout
	}

	err := d.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: stopTimeout})
	if err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	return nil
}

// RestartContainer restarts a Docker container by ID with optional timeout
func (d *DockerClient) RestartContainer(ctx context.Context, containerID string, timeout *int) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	var restartTimeout *int
	if timeout != nil {
		restartTimeout = timeout
	}

	err := d.client.ContainerRestart(ctx, containerID, container.StopOptions{Timeout: restartTimeout})
	if err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}

	return nil
}

// PauseContainer pauses a Docker container by ID
func (d *DockerClient) PauseContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	err := d.client.ContainerPause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to pause container %s: %w", containerID, err)
	}

	return nil
}

// UnpauseContainer unpauses a Docker container by ID
func (d *DockerClient) UnpauseContainer(ctx context.Context, containerID string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	err := d.client.ContainerUnpause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to unpause container %s: %w", containerID, err)
	}

	return nil
}

// RemoveContainer removes a Docker container by ID
func (d *DockerClient) RemoveContainer(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	err := d.client.ContainerRemove(ctx, containerID, options)
	if err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	return nil
}

// KillContainer kills a Docker container by ID with optional signal
func (d *DockerClient) KillContainer(ctx context.Context, containerID string, signal string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	if signal == "" {
		signal = "SIGKILL"
	}

	err := d.client.ContainerKill(ctx, containerID, signal)
	if err != nil {
		return fmt.Errorf("failed to kill container %s: %w", containerID, err)
	}

	return nil
}

// Container information and inspection

// GetContainer gets detailed information about a Docker container by ID
func (d *DockerClient) GetContainer(ctx context.Context, containerID string) (*types.ContainerJSON, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return nil, fmt.Errorf("container ID cannot be empty")
	}

	containerJSON, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	return &containerJSON, nil
}

// ListContainers lists Docker containers with optional filters
func (d *DockerClient) ListContainers(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	containers, err := d.client.ContainerList(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// ListAllContainers lists all Docker containers (running and stopped)
func (d *DockerClient) ListAllContainers(ctx context.Context) ([]types.Container, error) {
	return d.ListContainers(ctx, types.ContainerListOptions{All: true})
}

// ListRunningContainers lists only running Docker containers
func (d *DockerClient) ListRunningContainers(ctx context.Context) ([]types.Container, error) {
	return d.ListContainers(ctx, types.ContainerListOptions{All: false})
}

// FindContainerByName finds a container by name
func (d *DockerClient) FindContainerByName(ctx context.Context, name string) (*types.Container, error) {
	if name == "" {
		return nil, fmt.Errorf("container name cannot be empty")
	}

	// Add leading slash if not present (Docker container names start with /)
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	containers, err := d.ListAllContainers(ctx)
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		for _, containerName := range container.Names {
			if containerName == name {
				return &container, nil
			}
		}
	}

	return nil, fmt.Errorf("container with name %s not found", name)
}

// FindContainersByImage finds containers by image name
func (d *DockerClient) FindContainersByImage(ctx context.Context, imageName string) ([]types.Container, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	filterArgs := filters.NewArgs()
	filterArgs.Add("ancestor", imageName)

	options := types.ContainerListOptions{
		All:     true,
		Filters: filterArgs,
	}

	return d.ListContainers(ctx, options)
}

// Container status checking

// IsContainerRunning checks if a container is running
func (d *DockerClient) IsContainerRunning(ctx context.Context, containerID string) (bool, error) {
	containerJSON, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return false, err
	}

	return containerJSON.State.Running, nil
}

// GetContainerStatus gets the current status of a container
func (d *DockerClient) GetContainerStatus(ctx context.Context, containerID string) (model.ContainerStatus, error) {
	containerJSON, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return model.ContainerStatusUnknown, err
	}

	return d.mapDockerStateToModelStatus(containerJSON.State), nil
}

// mapDockerStateToModelStatus maps Docker container state to model status
func (d *DockerClient) mapDockerStateToModelStatus(state *types.ContainerState) model.ContainerStatus {
	if state.Running {
		return model.ContainerStatusRunning
	}
	if state.Paused {
		return model.ContainerStatusPaused
	}
	if state.Restarting {
		return model.ContainerStatusRestarting
	}
	if state.Dead {
		return model.ContainerStatusDead
	}
	if state.ExitCode != 0 {
		return model.ContainerStatusExited
	}
	return model.ContainerStatusStopped
}

// WaitForContainer waits for a container to exit
func (d *DockerClient) WaitForContainer(ctx context.Context, containerID string) (<-chan container.WaitResponse, <-chan error) {
	if ctx == nil {
		ctx = context.Background()
	}

	return d.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
}

// WaitForContainerWithTimeout waits for a container to exit with timeout
func (d *DockerClient) WaitForContainerWithTimeout(ctx context.Context, containerID string, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	statusCh, errCh := d.WaitForContainer(ctx, containerID)

	select {
	case <-statusCh:
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for container %s to exit", containerID)
	}
}

// Container execution

// ExecInContainer executes a command in a running container
func (d *DockerClient) ExecInContainer(ctx context.Context, containerID string, cmd []string, options types.ExecConfig) (types.HijackedResponse, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return types.HijackedResponse{}, fmt.Errorf("container ID cannot be empty")
	}

	if len(cmd) == 0 {
		return types.HijackedResponse{}, fmt.Errorf("command cannot be empty")
	}

	// Set default exec config
	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Override with provided options
	if options.User != "" {
		execConfig.User = options.User
	}
	if options.WorkingDir != "" {
		execConfig.WorkingDir = options.WorkingDir
	}
	if options.Env != nil {
		execConfig.Env = options.Env
	}
	if options.AttachStdin {
		execConfig.AttachStdin = true
	}

	// Create exec instance
	execIDResp, err := d.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return types.HijackedResponse{}, fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Start exec
	resp, err := d.client.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return types.HijackedResponse{}, fmt.Errorf("failed to attach to exec instance: %w", err)
	}

	return resp, nil
}

// ExecSimpleCommand executes a simple command and returns output
func (d *DockerClient) ExecSimpleCommand(ctx context.Context, containerID string, cmd []string) (string, string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	execConfig := types.ExecConfig{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	// Create exec instance
	execIDResp, err := d.client.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to create exec instance: %w", err)
	}

	// Attach to exec
	resp, err := d.client.ContainerExecAttach(ctx, execIDResp.ID, types.ExecStartCheck{})
	if err != nil {
		return "", "", fmt.Errorf("failed to attach to exec instance: %w", err)
	}
	defer resp.Close()

	// Read output
	var stdout, stderr strings.Builder
	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to read exec output: %w", err)
	}

	return stdout.String(), stderr.String(), nil
}

// Container copying

// CopyToContainer copies data to a container
func (d *DockerClient) CopyToContainer(ctx context.Context, containerID, dstPath string, content io.Reader, options types.CopyToContainerOptions) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	if dstPath == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	err := d.client.CopyToContainer(ctx, containerID, dstPath, content, options)
	if err != nil {
		return fmt.Errorf("failed to copy to container %s: %w", containerID, err)
	}

	return nil
}

// CopyFromContainer copies data from a container
func (d *DockerClient) CopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return nil, types.ContainerPathStat{}, fmt.Errorf("container ID cannot be empty")
	}

	if srcPath == "" {
		return nil, types.ContainerPathStat{}, fmt.Errorf("source path cannot be empty")
	}

	reader, stat, err := d.client.CopyFromContainer(ctx, containerID, srcPath)
	if err != nil {
		return nil, types.ContainerPathStat{}, fmt.Errorf("failed to copy from container %s: %w", containerID, err)
	}

	return reader, stat, nil
}

// Container updates and management

// UpdateContainer updates container configuration
func (d *DockerClient) UpdateContainer(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return container.ContainerUpdateOKBody{}, fmt.Errorf("container ID cannot be empty")
	}

	resp, err := d.client.ContainerUpdate(ctx, containerID, updateConfig)
	if err != nil {
		return container.ContainerUpdateOKBody{}, fmt.Errorf("failed to update container %s: %w", containerID, err)
	}

	return resp, nil
}

// RenameContainer renames a container
func (d *DockerClient) RenameContainer(ctx context.Context, containerID, newName string) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return fmt.Errorf("container ID cannot be empty")
	}

	if newName == "" {
		return fmt.Errorf("new name cannot be empty")
	}

	err := d.client.ContainerRename(ctx, containerID, newName)
	if err != nil {
		return fmt.Errorf("failed to rename container %s to %s: %w", containerID, newName, err)
	}

	return nil
}

// GetContainerSize gets the size of a container's filesystem
func (d *DockerClient) GetContainerSize(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = d.WithTimeout(context.Background())
		defer cancel()
	}

	if containerID == "" {
		return types.ContainerJSON{}, fmt.Errorf("container ID cannot be empty")
	}

	containerJSON, _, err := d.client.ContainerInspectWithRaw(ctx, containerID, true)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to inspect container %s with size: %w", containerID, err)
	}

	return containerJSON, nil
}

// Container health checks

// GetContainerHealth gets the health status of a container
func (d *DockerClient) GetContainerHealth(ctx context.Context, containerID string) (*types.Health, error) {
	containerJSON, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	if containerJSON.State.Health == nil {
		return nil, fmt.Errorf("container %s does not have health check configured", containerID)
	}

	return containerJSON.State.Health, nil
}

// IsContainerHealthy checks if a container is healthy
func (d *DockerClient) IsContainerHealthy(ctx context.Context, containerID string) (bool, error) {
	health, err := d.GetContainerHealth(ctx, containerID)
	if err != nil {
		return false, err
	}

	return health.Status == types.Healthy, nil
}

// WaitForHealthy waits for a container to become healthy
func (d *DockerClient) WaitForHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container %s to become healthy", containerID)
		case <-ticker.C:
			healthy, err := d.IsContainerHealthy(ctx, containerID)
			if err != nil {
				// If health check is not configured, consider it healthy
				if strings.Contains(err.Error(), "does not have health check configured") {
					return nil
				}
				return err
			}
			if healthy {
				return nil
			}
		}
	}
}

// Container utility functions

// ContainerExists checks if a container exists
func (d *DockerClient) ContainerExists(ctx context.Context, containerID string) (bool, error) {
	_, err := d.GetContainer(ctx, containerID)
	if err != nil {
		if IsContainerNotFoundError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsContainerNotFoundError checks if an error is a container not found error
func IsContainerNotFoundError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "No such container")
}

// GetContainerImage gets the image name of a container
func (d *DockerClient) GetContainerImage(ctx context.Context, containerID string) (string, error) {
	containerJSON, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return "", err
	}

	return containerJSON.Config.Image, nil
}

// GetContainerPorts gets the port mappings of a container
func (d *DockerClient) GetContainerPorts(ctx context.Context, containerID string) (map[string]string, error) {
	containerJSON, err := d.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	ports := make(map[string]string)
	for containerPort, hostBindings := range containerJSON.NetworkSettings.Ports {
		if len(hostBindings) > 0 {
			ports[hostBindings[0].HostPort] = string(containerPort)
		}
	}

	return ports, nil
}

// Parallel Container Operations for Performance Optimization

// BulkStartContainers starts multiple containers in parallel
func (d *DockerClient) BulkStartContainers(ctx context.Context, containerIDs []string, config BulkOperationConfig) []ParallelOperationResult {
	if config.MaxConcurrency <= 0 {
		config = DefaultBulkConfig()
	}

	return d.executeParallelOperation(ctx, containerIDs, "start", config, func(ctx context.Context, containerID string) error {
		return d.StartContainer(ctx, containerID)
	})
}

// BulkStopContainers stops multiple containers in parallel
func (d *DockerClient) BulkStopContainers(ctx context.Context, containerIDs []string, config BulkOperationConfig) []ParallelOperationResult {
	if config.MaxConcurrency <= 0 {
		config = DefaultBulkConfig()
	}

	return d.executeParallelOperation(ctx, containerIDs, "stop", config, func(ctx context.Context, containerID string) error {
		return d.StopContainer(ctx, containerID, &config.Timeout)
	})
}

// BulkRestartContainers restarts multiple containers in parallel
func (d *DockerClient) BulkRestartContainers(ctx context.Context, containerIDs []string, config BulkOperationConfig) []ParallelOperationResult {
	if config.MaxConcurrency <= 0 {
		config = DefaultBulkConfig()
	}

	return d.executeParallelOperation(ctx, containerIDs, "restart", config, func(ctx context.Context, containerID string) error {
		return d.RestartContainer(ctx, containerID, &config.Timeout)
	})
}

// BulkRemoveContainers removes multiple containers in parallel
func (d *DockerClient) BulkRemoveContainers(ctx context.Context, containerIDs []string, force bool, config BulkOperationConfig) []ParallelOperationResult {
	if config.MaxConcurrency <= 0 {
		config = DefaultBulkConfig()
	}

	return d.executeParallelOperation(ctx, containerIDs, "remove", config, func(ctx context.Context, containerID string) error {
		toptions := types.ContainerRemoveOptions{Force: force}
		return d.RemoveContainer(ctx, containerID, toptions)
	})
}

// BulkGetContainerStats gets stats for multiple containers in parallel
func (d *DockerClient) BulkGetContainerStats(ctx context.Context, containerIDs []string, config BulkOperationConfig) (map[string]*types.StatsJSON, []ParallelOperationResult) {
	if config.MaxConcurrency <= 0 {
		config = DefaultBulkConfig()
	}

	statsMap := make(map[string]*types.StatsJSON)
	statsMu := sync.RWMutex{}

	results := d.executeParallelOperation(ctx, containerIDs, "stats", config, func(ctx context.Context, containerID string) error {
		stats, err := d.GetContainerStats(ctx, containerID)
		if err != nil {
			return err
		}

		statsMu.Lock()
		statsMap[containerID] = stats
		statsMu.Unlock()
		return nil
	})

	return statsMap, results
}

// BulkInspectContainers inspects multiple containers in parallel
func (d *DockerClient) BulkInspectContainers(ctx context.Context, containerIDs []string, config BulkOperationConfig) (map[string]*types.ContainerJSON, []ParallelOperationResult) {
	if config.MaxConcurrency <= 0 {
		config = DefaultBulkConfig()
	}

	containersMap := make(map[string]*types.ContainerJSON)
	containersMu := sync.RWMutex{}

	results := d.executeParallelOperation(ctx, containerIDs, "inspect", config, func(ctx context.Context, containerID string) error {
		containerJSON, err := d.GetContainer(ctx, containerID)
		if err != nil {
			return err
		}

		containersMu.Lock()
		containersMap[containerID] = containerJSON
		containersMu.Unlock()
		return nil
	})

	return containersMap, results
}

// executeParallelOperation executes an operation on multiple containers in parallel
func (d *DockerClient) executeParallelOperation(
	ctx context.Context,
	containerIDs []string,
	operation string,
	config BulkOperationConfig,
	opFunc func(context.Context, string) error,
) []ParallelOperationResult {
	if ctx == nil {
		ctx = context.Background()
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(config.Timeout)*time.Second)
	defer cancel()

	// Initialize results
	results := make([]ParallelOperationResult, len(containerIDs))
	for i, containerID := range containerIDs {
		results[i] = ParallelOperationResult{
			ContainerID: containerID,
			Operation:   operation,
		}
	}

	// Create worker pool
	semaphore := make(chan struct{}, config.MaxConcurrency)
	var wg sync.WaitGroup
	resultsMu := sync.RWMutex{}
	completed := 0

	logrus.WithFields(logrus.Fields{
		"operation":      operation,
		"container_count": len(containerIDs),
		"max_concurrency": config.MaxConcurrency,
		"timeout":        config.Timeout,
	}).Info("Starting parallel container operation")

	start := time.Now()

	// Process each container
	for i, containerID := range containerIDs {
		wg.Add(1)
		go func(index int, cID string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Execute operation with timing
			opStart := time.Now()
			err := opFunc(ctx, cID)
			duration := time.Since(opStart)

			// Update result
			resultsMu.Lock()
			if err != nil {
				results[index].Error = err.Error()
				results[index].Success = false
			} else {
				results[index].Success = true
			}
			results[index].Duration = duration
			completed++

			// Call progress callback
			if config.ProgressCallback != nil {
				config.ProgressCallback(completed, len(containerIDs))
			}
			resultsMu.Unlock()

			// Log individual operation
			if err != nil {
				logrus.WithFields(logrus.Fields{
					"container_id": cID,
					"operation":    operation,
					"duration":     duration,
					"error":        err,
				}).Error("Container operation failed")

				// Fail fast if configured
				if config.FailFast {
					cancel()
				}
			} else {
				logrus.WithFields(logrus.Fields{
					"container_id": cID,
					"operation":    operation,
					"duration":     duration,
				}).Debug("Container operation completed")
			}
		}(i, containerID)
	}

	// Wait for all operations to complete
	wg.Wait()

	// Calculate summary statistics
	totalDuration := time.Since(start)
	successCount := 0
	failureCount := 0

	for _, result := range results {
		if result.Error == "" {
			successCount++
		} else {
			failureCount++
		}
	}

	logrus.WithFields(logrus.Fields{
		"operation":       operation,
		"total_containers": len(containerIDs),
		"successful":      successCount,
		"failed":          failureCount,
		"total_duration":  totalDuration,
		"avg_duration":    totalDuration / time.Duration(len(containerIDs)),
	}).Info("Parallel container operation completed")

	return results
}

// GetOperationSummary returns a summary of parallel operation results
func GetOperationSummary(results []ParallelOperationResult) map[string]interface{} {
	total := len(results)
	successful := 0
	failed := 0
	var totalDuration time.Duration
	var maxDuration time.Duration
	minDuration := time.Hour // Initialize with a large value

	for _, result := range results {
		if result.Error == "" {
			successful++
		} else {
			failed++
		}

		totalDuration += result.Duration

		if result.Duration > maxDuration {
			maxDuration = result.Duration
		}

		if result.Duration < minDuration {
			minDuration = result.Duration
		}
	}

	// Handle edge case where no operations were performed
	if total == 0 {
		minDuration = 0
	}

	return map[string]interface{}{
		"total_operations": total,
		"successful":      successful,
		"failed":          failed,
		"success_rate":    float64(successful) / float64(total) * 100,
		"total_duration":  totalDuration,
		"average_duration": totalDuration / time.Duration(total),
		"max_duration":    maxDuration,
		"min_duration":    minDuration,
	}
}

// FilterSuccessfulOperations returns only successful operation results
func FilterSuccessfulOperations(results []ParallelOperationResult) []ParallelOperationResult {
	var successful []ParallelOperationResult
	for _, result := range results {
		if result.Error == "" {
			successful = append(successful, result)
		}
	}
	return successful
}

// FilterFailedOperations returns only failed operation results
func FilterFailedOperations(results []ParallelOperationResult) []ParallelOperationResult {
	var failed []ParallelOperationResult
	for _, result := range results {
		if result.Error != "" {
			failed = append(failed, result)
		}
	}
	return failed
}