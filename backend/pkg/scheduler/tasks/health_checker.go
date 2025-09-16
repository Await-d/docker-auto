package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"docker-auto/internal/model"
	"docker-auto/internal/repository"
	"docker-auto/internal/service"
	"docker-auto/pkg/docker"
	"docker-auto/pkg/scheduler"

	"github.com/sirupsen/logrus"
)

// HealthCheckerTask implements the Task interface for container health checking
type HealthCheckerTask struct {
	containerRepo       repository.ContainerRepository
	containerService    *service.ContainerService
	notificationService *service.NotificationService
	dockerClient        *docker.DockerClient
	httpClient          *http.Client
}

// NewHealthCheckerTask creates a new health checker task
func NewHealthCheckerTask(
	containerRepo repository.ContainerRepository,
	containerService *service.ContainerService,
	notificationService *service.NotificationService,
	dockerClient *docker.DockerClient,
) *HealthCheckerTask {
	return &HealthCheckerTask{
		containerRepo:       containerRepo,
		containerService:    containerService,
		notificationService: notificationService,
		dockerClient:        dockerClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute runs the health checking task
func (t *HealthCheckerTask) Execute(ctx context.Context, params scheduler.TaskParameters) error {
	logger := logrus.WithFields(logrus.Fields{
		"task_type": t.GetType(),
		"task_name": t.GetName(),
	})

	logger.Info("Starting container health check task")

	// Parse task-specific parameters
	healthParams, err := t.parseParameters(params)
	if err != nil {
		return fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Get containers to check
	containers, err := t.getContainersToCheck(ctx, params.TargetContainers)
	if err != nil {
		return fmt.Errorf("failed to get containers to check: %w", err)
	}

	logger.WithField("container_count", len(containers)).Info("Found containers to health check")

	if len(containers) == 0 {
		logger.Info("No containers to health check")
		return nil
	}

	// Perform health checks
	results, err := t.performHealthChecks(ctx, containers, healthParams)
	if err != nil {
		return fmt.Errorf("failed to perform health checks: %w", err)
	}

	// Process results
	if err := t.processResults(ctx, results, healthParams); err != nil {
		return fmt.Errorf("failed to process results: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"containers_checked": len(containers),
		"healthy_containers": results.HealthyContainers,
		"unhealthy_containers": results.UnhealthyContainers,
		"failed_checks": results.FailedChecks,
		"restarted_containers": results.RestartedContainers,
	}).Info("Container health check task completed")

	return nil
}

// GetName returns the task name
func (t *HealthCheckerTask) GetName() string {
	return "Container Health Checker"
}

// GetType returns the task type
func (t *HealthCheckerTask) GetType() model.TaskType {
	return model.TaskTypeHealthCheck
}

// Validate validates task parameters
func (t *HealthCheckerTask) Validate(params scheduler.TaskParameters) error {
	if params.TaskType != model.TaskTypeHealthCheck {
		return fmt.Errorf("invalid task type: expected %s, got %s", model.TaskTypeHealthCheck, params.TaskType)
	}

	// Validate parameters structure
	if _, err := t.parseParameters(params); err != nil {
		return fmt.Errorf("invalid parameters: %w", err)
	}

	return nil
}

// GetDefaultTimeout returns the default timeout for this task
func (t *HealthCheckerTask) GetDefaultTimeout() time.Duration {
	return 15 * time.Minute
}

// CanRunConcurrently returns true if this task can run concurrently
func (t *HealthCheckerTask) CanRunConcurrently() bool {
	return true
}

// HealthCheckParameters represents parameters for health checking
type HealthCheckParameters struct {
	CheckTimeout        time.Duration `json:"check_timeout"`
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	MaxConcurrent       int           `json:"max_concurrent"`
	NotifyOnFailure     bool          `json:"notify_on_failure"`
	NotifyOnRecovery    bool          `json:"notify_on_recovery"`
	RestartOnFailure    bool          `json:"restart_on_failure"`
	RestartMaxAttempts  int           `json:"restart_max_attempts"`
	RestartDelay        time.Duration `json:"restart_delay"`
	CheckServices       []string      `json:"check_services"`
	HTTPChecks          []HTTPHealthCheck `json:"http_checks"`
	TCPChecks           []TCPHealthCheck  `json:"tcp_checks"`
	CommandChecks       []CommandHealthCheck `json:"command_checks"`
	EnableDockerHealth  bool          `json:"enable_docker_health"`
	FailureThreshold    int           `json:"failure_threshold"`
	SuccessThreshold    int           `json:"success_threshold"`
}

// HTTPHealthCheck represents an HTTP health check configuration
type HTTPHealthCheck struct {
	ContainerName string            `json:"container_name"`
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Headers       map[string]string `json:"headers"`
	ExpectedStatus int              `json:"expected_status"`
	ExpectedBody   string            `json:"expected_body"`
	Timeout       time.Duration     `json:"timeout"`
}

// TCPHealthCheck represents a TCP health check configuration
type TCPHealthCheck struct {
	ContainerName string        `json:"container_name"`
	Host          string        `json:"host"`
	Port          int           `json:"port"`
	Timeout       time.Duration `json:"timeout"`
}

// CommandHealthCheck represents a command-based health check
type CommandHealthCheck struct {
	ContainerName string   `json:"container_name"`
	Command       []string `json:"command"`
	ExpectedExit  int      `json:"expected_exit"`
	Timeout       time.Duration `json:"timeout"`
}

// HealthCheckResult represents the result of health checking all containers
type HealthCheckResult struct {
	ContainerResults    []*ContainerHealthResult `json:"container_results"`
	HealthyContainers   int                      `json:"healthy_containers"`
	UnhealthyContainers int                      `json:"unhealthy_containers"`
	FailedChecks        int                      `json:"failed_checks"`
	RestartedContainers int                      `json:"restarted_containers"`
	Duration            time.Duration            `json:"duration"`
	CheckedAt           time.Time                `json:"checked_at"`
	Errors              []HealthCheckError       `json:"errors"`
}

// ContainerHealthResult represents the health check result for a single container
type ContainerHealthResult struct {
	Container            *model.Container      `json:"container"`
	OverallHealth        HealthStatus          `json:"overall_health"`
	DockerHealth         *DockerHealthInfo     `json:"docker_health,omitempty"`
	CustomChecks         []*CustomCheckResult  `json:"custom_checks"`
	ResourceMetrics      *ResourceMetrics      `json:"resource_metrics,omitempty"`
	CheckedAt            time.Time             `json:"checked_at"`
	Duration             time.Duration         `json:"duration"`
	ConsecutiveFailures  int                   `json:"consecutive_failures"`
	LastHealthyAt        *time.Time            `json:"last_healthy_at,omitempty"`
	ActionsTaken         []HealthAction        `json:"actions_taken"`
	Error                string                `json:"error,omitempty"`
}

// HealthStatus represents the health status of a container
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusWarning   HealthStatus = "warning"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// DockerHealthInfo represents Docker's built-in health check information
type DockerHealthInfo struct {
	Status      string            `json:"status"`
	FailingStreak int             `json:"failing_streak"`
	Log         []DockerHealthLog `json:"log"`
}

// DockerHealthLog represents a Docker health check log entry
type DockerHealthLog struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	ExitCode int       `json:"exit_code"`
	Output   string    `json:"output"`
}

// CustomCheckResult represents the result of a custom health check
type CustomCheckResult struct {
	CheckType   string        `json:"check_type"` // http, tcp, command
	CheckName   string        `json:"check_name"`
	Success     bool          `json:"success"`
	Duration    time.Duration `json:"duration"`
	Message     string        `json:"message"`
	Error       string        `json:"error,omitempty"`
	Details     interface{}   `json:"details,omitempty"`
}

// ResourceMetrics represents container resource usage metrics
type ResourceMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   int64   `json:"memory_usage"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkIO     NetworkIO `json:"network_io"`
	DiskIO        DiskIO    `json:"disk_io"`
}

// NetworkIO represents network I/O metrics
type NetworkIO struct {
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
}

// DiskIO represents disk I/O metrics
type DiskIO struct {
	ReadBytes  int64 `json:"read_bytes"`
	WriteBytes int64 `json:"write_bytes"`
}

// HealthAction represents an action taken based on health check results
type HealthAction struct {
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
}

// HealthCheckError represents an error during health checking
type HealthCheckError struct {
	ContainerID   int64  `json:"container_id"`
	ContainerName string `json:"container_name"`
	Error         string `json:"error"`
	CheckType     string `json:"check_type"`
	Recoverable   bool   `json:"recoverable"`
}

// parseParameters parses and validates task parameters
func (t *HealthCheckerTask) parseParameters(params scheduler.TaskParameters) (*HealthCheckParameters, error) {
	// Set defaults
	healthParams := &HealthCheckParameters{
		CheckTimeout:       30 * time.Second,
		MaxRetries:         3,
		RetryDelay:         5 * time.Second,
		MaxConcurrent:      10,
		NotifyOnFailure:    true,
		NotifyOnRecovery:   true,
		RestartOnFailure:   false,
		RestartMaxAttempts: 3,
		RestartDelay:       10 * time.Second,
		CheckServices:      []string{},
		HTTPChecks:         []HTTPHealthCheck{},
		TCPChecks:          []TCPHealthCheck{},
		CommandChecks:      []CommandHealthCheck{},
		EnableDockerHealth: true,
		FailureThreshold:   3,
		SuccessThreshold:   2,
	}

	// Parse from parameters map
	if params.Parameters != nil {
		jsonData, err := json.Marshal(params.Parameters)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal parameters: %w", err)
		}

		if err := json.Unmarshal(jsonData, healthParams); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}
	}

	// Validate parameters
	if healthParams.MaxConcurrent <= 0 {
		healthParams.MaxConcurrent = 10
	}
	if healthParams.MaxConcurrent > 50 {
		healthParams.MaxConcurrent = 50
	}

	if healthParams.CheckTimeout <= 0 {
		healthParams.CheckTimeout = 30 * time.Second
	}

	if healthParams.MaxRetries < 0 {
		healthParams.MaxRetries = 0
	}

	return healthParams, nil
}

// getContainersToCheck retrieves containers that should be health checked
func (t *HealthCheckerTask) getContainersToCheck(ctx context.Context, targetContainers []int64) ([]*model.Container, error) {
	var containers []*model.Container

	if len(targetContainers) > 0 {
		// Check specific containers
		for _, containerID := range targetContainers {
			container, err := t.containerRepo.GetByID(ctx, containerID)
			if err != nil {
				logrus.WithError(err).WithField("container_id", containerID).Warn("Failed to get container")
				continue
			}
			containers = append(containers, container)
		}
	} else {
		// Check all running containers
		filter := &model.ContainerFilter{
			Status: &model.ContainerStatusRunning,
			Limit:  1000,
		}

		allContainers, _, err := t.containerRepo.List(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("failed to list containers: %w", err)
		}

		containers = allContainers
	}

	return containers, nil
}

// performHealthChecks performs health checks on all containers
func (t *HealthCheckerTask) performHealthChecks(ctx context.Context, containers []*model.Container, params *HealthCheckParameters) (*HealthCheckResult, error) {
	startTime := time.Now()
	result := &HealthCheckResult{
		ContainerResults: make([]*ContainerHealthResult, 0, len(containers)),
		CheckedAt:        startTime,
	}

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, params.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, container := range containers {
		wg.Add(1)
		go func(c *model.Container) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				return
			}

			// Check this container's health
			containerResult := t.checkContainerHealth(ctx, c, params)

			// Add to results
			mu.Lock()
			result.ContainerResults = append(result.ContainerResults, containerResult)
			switch containerResult.OverallHealth {
			case HealthStatusHealthy:
				result.HealthyContainers++
			case HealthStatusUnhealthy:
				result.UnhealthyContainers++
			default:
				result.FailedChecks++
			}
			if containerResult.Error != "" {
				result.Errors = append(result.Errors, HealthCheckError{
					ContainerID:   int64(c.ID),
					ContainerName: c.Name,
					Error:         containerResult.Error,
					Recoverable:   true,
				})
			}
			// Count restarts from actions taken
			for _, action := range containerResult.ActionsTaken {
				if action.Action == "restart" && action.Success {
					result.RestartedContainers++
				}
			}
			mu.Unlock()
		}(container)
	}

	wg.Wait()
	result.Duration = time.Since(startTime)

	return result, nil
}

// checkContainerHealth checks the health of a single container
func (t *HealthCheckerTask) checkContainerHealth(ctx context.Context, container *model.Container, params *HealthCheckParameters) *ContainerHealthResult {
	startTime := time.Now()
	result := &ContainerHealthResult{
		Container:     container,
		CheckedAt:     startTime,
		CustomChecks:  []*CustomCheckResult{},
		ActionsTaken:  []HealthAction{},
	}

	logger := logrus.WithFields(logrus.Fields{
		"container_id":   container.ID,
		"container_name": container.Name,
	})

	// Check Docker health status
	if params.EnableDockerHealth {
		result.DockerHealth = t.checkDockerHealth(ctx, container, params)
	}

	// Get resource metrics
	result.ResourceMetrics = t.getResourceMetrics(ctx, container)

	// Perform custom health checks
	result.CustomChecks = t.performCustomChecks(ctx, container, params)

	// Determine overall health status
	result.OverallHealth = t.determineOverallHealth(result)

	// Take actions based on health status
	if result.OverallHealth == HealthStatusUnhealthy {
		actions := t.takeHealthActions(ctx, container, result, params)
		result.ActionsTaken = append(result.ActionsTaken, actions...)
	}

	result.Duration = time.Since(startTime)

	logger.WithFields(logrus.Fields{
		"overall_health": result.OverallHealth,
		"duration":       result.Duration,
		"actions_taken":  len(result.ActionsTaken),
	}).Debug("Container health check completed")

	return result
}

// checkDockerHealth checks Docker's built-in health status
func (t *HealthCheckerTask) checkDockerHealth(ctx context.Context, container *model.Container, params *HealthCheckParameters) *DockerHealthInfo {
	if t.dockerClient == nil || container.ContainerID == "" {
		return nil
	}

	healthInfo := &DockerHealthInfo{}

	// Get container status including health
	status, err := t.dockerClient.GetContainerStatus(ctx, container.ContainerID)
	if err != nil {
		logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to get container status")
		return nil
	}

	healthInfo.Status = status.Health

	// If Docker health check is not configured, return basic status
	if healthInfo.Status == "" {
		if status.Running {
			healthInfo.Status = "healthy"
		} else {
			healthInfo.Status = "unhealthy"
		}
	}

	return healthInfo
}

// getResourceMetrics retrieves resource usage metrics for the container
func (t *HealthCheckerTask) getResourceMetrics(ctx context.Context, container *model.Container) *ResourceMetrics {
	if t.dockerClient == nil || container.ContainerID == "" {
		return nil
	}

	// Get container stats
	stats, err := t.dockerClient.GetContainerStats(ctx, container.ContainerID, false)
	if err != nil {
		logrus.WithError(err).WithField("container_id", container.ContainerID).Warn("Failed to get container stats")
		return nil
	}

	// Convert stats to our metrics format
	metrics := &ResourceMetrics{
		CPUPercent:    stats.CPUPercent,
		MemoryUsage:   int64(stats.MemoryUsage),
		MemoryPercent: stats.MemoryPercent,
		NetworkIO: NetworkIO{
			RxBytes:   int64(stats.NetworkRx),
			TxBytes:   int64(stats.NetworkTx),
			RxPackets: 0, // Not available in simplified stats
			TxPackets: 0, // Not available in simplified stats
		},
		DiskIO: DiskIO{
			ReadBytes:  int64(stats.DiskRead),
			WriteBytes: int64(stats.DiskWrite),
		},
	}

	return metrics
}

// performCustomChecks performs custom health checks
func (t *HealthCheckerTask) performCustomChecks(ctx context.Context, container *model.Container, params *HealthCheckParameters) []*CustomCheckResult {
	var results []*CustomCheckResult

	// Perform HTTP checks
	for _, httpCheck := range params.HTTPChecks {
		if httpCheck.ContainerName == container.Name {
			result := t.performHTTPCheck(ctx, httpCheck)
			results = append(results, result)
		}
	}

	// Perform TCP checks
	for _, tcpCheck := range params.TCPChecks {
		if tcpCheck.ContainerName == container.Name {
			result := t.performTCPCheck(ctx, tcpCheck)
			results = append(results, result)
		}
	}

	// Perform command checks
	for _, cmdCheck := range params.CommandChecks {
		if cmdCheck.ContainerName == container.Name {
			result := t.performCommandCheck(ctx, container, cmdCheck)
			results = append(results, result)
		}
	}

	return results
}

// performHTTPCheck performs an HTTP health check
func (t *HealthCheckerTask) performHTTPCheck(ctx context.Context, check HTTPHealthCheck) *CustomCheckResult {
	startTime := time.Now()
	result := &CustomCheckResult{
		CheckType: "http",
		CheckName: fmt.Sprintf("HTTP %s %s", check.Method, check.URL),
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, check.Method, check.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to create request: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Add headers
	for key, value := range check.Headers {
		req.Header.Set(key, value)
	}

	// Perform request with timeout
	client := &http.Client{Timeout: check.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("Request failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}
	defer resp.Body.Close()

	result.Duration = time.Since(startTime)

	// Check status code
	expectedStatus := check.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}

	if resp.StatusCode != expectedStatus {
		result.Error = fmt.Sprintf("Expected status %d, got %d", expectedStatus, resp.StatusCode)
		return result
	}

	// Check response body if specified
	if check.ExpectedBody != "" {
		body := make([]byte, 1024) // Read first 1KB
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])

		if !strings.Contains(bodyStr, check.ExpectedBody) {
			result.Error = fmt.Sprintf("Expected body to contain '%s'", check.ExpectedBody)
			return result
		}
	}

	result.Success = true
	result.Message = fmt.Sprintf("HTTP check successful, status: %d", resp.StatusCode)

	return result
}

// performTCPCheck performs a TCP connectivity check
func (t *HealthCheckerTask) performTCPCheck(ctx context.Context, check TCPHealthCheck) *CustomCheckResult {
	startTime := time.Now()
	result := &CustomCheckResult{
		CheckType: "tcp",
		CheckName: fmt.Sprintf("TCP %s:%d", check.Host, check.Port),
	}

	// Create dialer with timeout
	dialer := &net.Dialer{Timeout: check.Timeout}

	// Attempt connection
	conn, err := dialer.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", check.Host, check.Port))
	if err != nil {
		result.Error = fmt.Sprintf("Connection failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	conn.Close()
	result.Duration = time.Since(startTime)
	result.Success = true
	result.Message = "TCP connection successful"

	return result
}

// performCommandCheck performs a command-based health check
func (t *HealthCheckerTask) performCommandCheck(ctx context.Context, container *model.Container, check CommandHealthCheck) *CustomCheckResult {
	startTime := time.Now()
	result := &CustomCheckResult{
		CheckType: "command",
		CheckName: fmt.Sprintf("Command: %s", strings.Join(check.Command, " ")),
	}

	if t.dockerClient == nil || container.ContainerID == "" {
		result.Error = "Docker client not available or container not running"
		result.Duration = time.Since(startTime)
		return result
	}

	// Execute command in container
	execResult, err := t.dockerClient.ExecCommand(ctx, container.ContainerID, check.Command)
	if err != nil {
		result.Error = fmt.Sprintf("Command execution failed: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.Duration = time.Since(startTime)

	// Check exit code
	expectedExit := check.ExpectedExit
	if execResult.ExitCode != expectedExit {
		result.Error = fmt.Sprintf("Expected exit code %d, got %d", expectedExit, execResult.ExitCode)
		result.Details = map[string]interface{}{
			"stdout":    execResult.Stdout,
			"stderr":    execResult.Stderr,
			"exit_code": execResult.ExitCode,
		}
		return result
	}

	result.Success = true
	result.Message = fmt.Sprintf("Command executed successfully, exit code: %d", execResult.ExitCode)
	result.Details = map[string]interface{}{
		"stdout": execResult.Stdout,
		"stderr": execResult.Stderr,
	}

	return result
}

// determineOverallHealth determines the overall health status based on all checks
func (t *HealthCheckerTask) determineOverallHealth(result *ContainerHealthResult) HealthStatus {
	// Check Docker health first
	if result.DockerHealth != nil {
		switch result.DockerHealth.Status {
		case "healthy":
			// Continue to check custom checks
		case "unhealthy":
			return HealthStatusUnhealthy
		case "starting":
			return HealthStatusWarning
		default:
			return HealthStatusUnknown
		}
	}

	// Check custom health checks
	hasFailedChecks := false
	for _, check := range result.CustomChecks {
		if !check.Success {
			hasFailedChecks = true
			break
		}
	}

	if hasFailedChecks {
		return HealthStatusUnhealthy
	}

	// Check resource metrics for warnings
	if result.ResourceMetrics != nil {
		if result.ResourceMetrics.CPUPercent > 90 || result.ResourceMetrics.MemoryPercent > 90 {
			return HealthStatusWarning
		}
	}

	return HealthStatusHealthy
}

// takeHealthActions takes appropriate actions based on health status
func (t *HealthCheckerTask) takeHealthActions(ctx context.Context, container *model.Container, result *ContainerHealthResult, params *HealthCheckParameters) []HealthAction {
	var actions []HealthAction

	// Restart container if configured and appropriate
	if params.RestartOnFailure && result.OverallHealth == HealthStatusUnhealthy {
		action := t.restartContainer(ctx, container, params)
		actions = append(actions, action)
	}

	return actions
}

// restartContainer attempts to restart an unhealthy container
func (t *HealthCheckerTask) restartContainer(ctx context.Context, container *model.Container, params *HealthCheckParameters) HealthAction {
	action := HealthAction{
		Action:    "restart",
		Timestamp: time.Now(),
	}

	if t.containerService == nil {
		action.Error = "Container service not available"
		action.Success = false
		return action
	}

	// Note: In a real implementation, you would check restart limits and cooldowns
	// For now, we'll attempt the restart
	err := t.containerService.RestartContainer(ctx, 1, int64(container.ID)) // userID would come from context
	if err != nil {
		action.Error = fmt.Sprintf("Restart failed: %v", err)
		action.Success = false
	} else {
		action.Success = true
		action.Message = "Container restarted successfully"
	}

	return action
}

// processResults processes the health check results
func (t *HealthCheckerTask) processResults(ctx context.Context, results *HealthCheckResult, params *HealthCheckParameters) error {
	// Send notifications for unhealthy containers
	if params.NotifyOnFailure && results.UnhealthyContainers > 0 {
		if err := t.sendUnhealthyNotification(ctx, results); err != nil {
			logrus.WithError(err).Warn("Failed to send unhealthy container notification")
		}
	}

	// Send recovery notifications if containers were restarted
	if params.NotifyOnRecovery && results.RestartedContainers > 0 {
		if err := t.sendRecoveryNotification(ctx, results); err != nil {
			logrus.WithError(err).Warn("Failed to send recovery notification")
		}
	}

	return nil
}

// sendUnhealthyNotification sends notification about unhealthy containers
func (t *HealthCheckerTask) sendUnhealthyNotification(ctx context.Context, results *HealthCheckResult) error {
	if t.notificationService == nil {
		return nil
	}

	var unhealthyContainers []string
	for _, result := range results.ContainerResults {
		if result.OverallHealth == HealthStatusUnhealthy {
			unhealthyContainers = append(unhealthyContainers, result.Container.Name)
		}
	}

	notification := &model.Notification{
		Type:     model.NotificationTypeHealthCheck,
		Title:    "Unhealthy Containers Detected",
		Message:  fmt.Sprintf("Health check found %d unhealthy container(s): %s",
			results.UnhealthyContainers, strings.Join(unhealthyContainers, ", ")),
		Priority: model.NotificationPriorityHigh,
		Data: map[string]interface{}{
			"unhealthy_containers": unhealthyContainers,
			"total_checked":        len(results.ContainerResults),
			"healthy_containers":   results.HealthyContainers,
		},
	}

	return t.notificationService.SendNotification(ctx, notification)
}

// sendRecoveryNotification sends notification about recovery actions
func (t *HealthCheckerTask) sendRecoveryNotification(ctx context.Context, results *HealthCheckResult) error {
	if t.notificationService == nil {
		return nil
	}

	notification := &model.Notification{
		Type:     model.NotificationTypeHealthCheck,
		Title:    "Container Recovery Actions Taken",
		Message:  fmt.Sprintf("Attempted to restart %d unhealthy container(s)",
			results.RestartedContainers),
		Priority: model.NotificationPriorityNormal,
		Data: map[string]interface{}{
			"restarted_containers": results.RestartedContainers,
			"unhealthy_containers": results.UnhealthyContainers,
			"total_checked":        len(results.ContainerResults),
		},
	}

	return t.notificationService.SendNotification(ctx, notification)
}