package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// DatabaseHealthCheck implements health check for database
type DatabaseHealthCheck struct {
	config DatabaseHealthConfig
	db     *sql.DB
}

// NewDatabaseHealthCheck creates a new database health check
func NewDatabaseHealthCheck(config DatabaseHealthConfig, db *sql.DB) *DatabaseHealthCheck {
	return &DatabaseHealthCheck{
		config: config,
		db:     db,
	}
}

func (dhc *DatabaseHealthCheck) Name() string {
	return dhc.config.Name
}

func (dhc *DatabaseHealthCheck) Dependencies() []string {
	return []string{}
}

func (dhc *DatabaseHealthCheck) Config() HealthCheckConfig {
	return dhc.config.HealthCheckConfig
}

func (dhc *DatabaseHealthCheck) Check(ctx context.Context) HealthResult {
	start := time.Now()
	result := HealthResult{
		Status:    HealthStatusUnhealthy,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Test database connection
	ctx, cancel := context.WithTimeout(ctx, dhc.config.QueryTimeout)
	defer cancel()

	// Execute test query
	testQuery := dhc.config.TestQuery
	if testQuery == "" {
		testQuery = "SELECT 1"
	}

	row := dhc.db.QueryRowContext(ctx, testQuery)
	var dummy int
	err := row.Scan(&dummy)

	result.Duration = time.Since(start)

	if err != nil {
		result.Message = fmt.Sprintf("Database health check failed: %v", err)
		result.Error = err.Error()
		return result
	}

	// Check database statistics
	stats := dhc.db.Stats()
	result.Details["open_connections"] = stats.OpenConnections
	result.Details["in_use"] = stats.InUse
	result.Details["idle"] = stats.Idle
	result.Details["wait_count"] = stats.WaitCount
	result.Details["wait_duration"] = stats.WaitDuration.String()
	result.Details["max_idle_closed"] = stats.MaxIdleClosed
	result.Details["max_lifetime_closed"] = stats.MaxLifetimeClosed

	// Check connection limits
	if dhc.config.MaxConnections > 0 && stats.OpenConnections > dhc.config.MaxConnections {
		result.Status = HealthStatusDegraded
		result.Message = fmt.Sprintf("Database connection count is high: %d/%d", stats.OpenConnections, dhc.config.MaxConnections)
		return result
	}

	result.Status = HealthStatusHealthy
	result.Message = "Database is healthy"
	return result
}

// DockerHealthCheck implements health check for Docker daemon
type DockerHealthCheck struct {
	config       DockerHealthConfig
	dockerClient *client.Client
}

// NewDockerHealthCheck creates a new Docker health check
func NewDockerHealthCheck(config DockerHealthConfig) *DockerHealthCheck {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	return &DockerHealthCheck{
		config:       config,
		dockerClient: dockerClient,
	}
}

func (dhc *DockerHealthCheck) Name() string {
	return dhc.config.Name
}

func (dhc *DockerHealthCheck) Dependencies() []string {
	return []string{}
}

func (dhc *DockerHealthCheck) Config() HealthCheckConfig {
	return dhc.config.HealthCheckConfig
}

func (dhc *DockerHealthCheck) Check(ctx context.Context) HealthResult {
	start := time.Now()
	result := HealthResult{
		Status:    HealthStatusUnhealthy,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if dhc.dockerClient == nil {
		result.Message = "Docker client not available"
		result.Duration = time.Since(start)
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, dhc.config.RequestTimeout)
	defer cancel()

	// Test Docker daemon connectivity
	_, err := dhc.dockerClient.Ping(ctx)
	if err != nil {
		result.Message = fmt.Sprintf("Docker daemon ping failed: %v", err)
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Get Docker info
	info, err := dhc.dockerClient.Info(ctx)
	if err == nil {
		result.Details["docker_version"] = info.ServerVersion
		result.Details["containers"] = info.Containers
		result.Details["containers_running"] = info.ContainersRunning
		result.Details["containers_paused"] = info.ContainersPaused
		result.Details["containers_stopped"] = info.ContainersStopped
		result.Details["images"] = info.Images
	}

	// Check containers if configured
	if dhc.config.CheckContainers {
		containers, err := dhc.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
		if err != nil {
			result.Status = HealthStatusDegraded
			result.Message = fmt.Sprintf("Failed to list containers: %v", err)
			result.Duration = time.Since(start)
			return result
		}

		runningContainers := 0
		for _, container := range containers {
			if container.State == "running" {
				runningContainers++
			}
		}
		result.Details["total_containers"] = len(containers)
		result.Details["running_containers"] = runningContainers
	}

	result.Status = HealthStatusHealthy
	result.Message = "Docker daemon is healthy"
	result.Duration = time.Since(start)
	return result
}

// HTTPHealthCheck implements health check for HTTP endpoints
type HTTPHealthCheck struct {
	config     HTTPHealthConfig
	httpClient *http.Client
}

// NewHTTPHealthCheck creates a new HTTP health check
func NewHTTPHealthCheck(config HTTPHealthConfig) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if !config.FollowRedirects && len(via) > 0 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

func (hhc *HTTPHealthCheck) Name() string {
	return hhc.config.Name
}

func (hhc *HTTPHealthCheck) Dependencies() []string {
	return []string{}
}

func (hhc *HTTPHealthCheck) Config() HealthCheckConfig {
	return hhc.config.HealthCheckConfig
}

func (hhc *HTTPHealthCheck) Check(ctx context.Context) HealthResult {
	start := time.Now()
	result := HealthResult{
		Status:    HealthStatusUnhealthy,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	method := hhc.config.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, hhc.config.URL, nil)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to create HTTP request: %v", err)
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Add headers
	for key, value := range hhc.config.Headers {
		req.Header.Set(key, value)
	}

	// Make request
	resp, err := hhc.httpClient.Do(req)
	if err != nil {
		result.Message = fmt.Sprintf("HTTP request failed: %v", err)
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}
	defer resp.Body.Close()

	result.Details["status_code"] = resp.StatusCode
	result.Details["response_time"] = time.Since(start).String()

	// Check expected status codes
	expectedStatuses := hhc.config.ExpectedStatus
	if len(expectedStatuses) == 0 {
		expectedStatuses = []int{200}
	}

	statusOK := false
	for _, expectedStatus := range expectedStatuses {
		if resp.StatusCode == expectedStatus {
			statusOK = true
			break
		}
	}

	if !statusOK {
		result.Message = fmt.Sprintf("HTTP check failed: unexpected status code %d", resp.StatusCode)
		result.Duration = time.Since(start)
		return result
	}

	result.Status = HealthStatusHealthy
	result.Message = "HTTP endpoint is healthy"
	result.Duration = time.Since(start)
	return result
}

// FileSystemHealthCheck implements health check for filesystem
type FileSystemHealthCheck struct {
	config FileSystemHealthConfig
}

// NewFileSystemHealthCheck creates a new filesystem health check
func NewFileSystemHealthCheck(config FileSystemHealthConfig) *FileSystemHealthCheck {
	return &FileSystemHealthCheck{
		config: config,
	}
}

func (fshc *FileSystemHealthCheck) Name() string {
	return fshc.config.Name
}

func (fshc *FileSystemHealthCheck) Dependencies() []string {
	return []string{}
}

func (fshc *FileSystemHealthCheck) Config() HealthCheckConfig {
	return fshc.config.HealthCheckConfig
}

func (fshc *FileSystemHealthCheck) Check(ctx context.Context) HealthResult {
	start := time.Now()
	result := HealthResult{
		Status:    HealthStatusUnhealthy,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check if path exists
	stat, err := os.Stat(fshc.config.Path)
	if err != nil {
		result.Message = fmt.Sprintf("Filesystem check failed: %v", err)
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	result.Details["path"] = fshc.config.Path
	result.Details["is_dir"] = stat.IsDir()
	result.Details["mod_time"] = stat.ModTime()
	result.Details["size"] = stat.Size()

	// Get disk usage
	var diskStat syscall.Statfs_t
	err = syscall.Statfs(fshc.config.Path, &diskStat)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to get disk stats: %v", err)
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Calculate disk usage
	blockSize := uint64(diskStat.Bsize)
	totalSpace := diskStat.Blocks * blockSize
	freeSpace := diskStat.Bavail * blockSize
	usedSpace := totalSpace - freeSpace
	usagePercent := float64(usedSpace) / float64(totalSpace) * 100

	result.Details["total_space"] = totalSpace
	result.Details["free_space"] = freeSpace
	result.Details["used_space"] = usedSpace
	result.Details["usage_percent"] = usagePercent

	// Check minimum free space
	if fshc.config.MinFreeSpace > 0 && freeSpace < fshc.config.MinFreeSpace {
		result.Status = HealthStatusDegraded
		result.Message = fmt.Sprintf("Low disk space: %d bytes free (minimum: %d bytes)", freeSpace, fshc.config.MinFreeSpace)
		result.Duration = time.Since(start)
		return result
	}

	// Check minimum free percentage
	if fshc.config.MinFreePercent > 0 && (100-usagePercent) < fshc.config.MinFreePercent {
		result.Status = HealthStatusDegraded
		result.Message = fmt.Sprintf("Low disk space: %.2f%% free (minimum: %.2f%%)", 100-usagePercent, fshc.config.MinFreePercent)
		result.Duration = time.Since(start)
		return result
	}

	// Test read access
	if fshc.config.CheckReadable {
		if stat.IsDir() {
			_, err = os.ReadDir(fshc.config.Path)
		} else {
			_, err = os.ReadFile(fshc.config.Path)
		}
		if err != nil {
			result.Message = fmt.Sprintf("Read test failed: %v", err)
			result.Error = err.Error()
			result.Duration = time.Since(start)
			return result
		}
	}

	// Test write access
	if fshc.config.CheckWritable {
		testFile := fshc.config.TestFilePath
		if testFile == "" {
			if stat.IsDir() {
				testFile = fshc.config.Path + "/.health_check_test"
			} else {
				testFile = fshc.config.Path + ".health_check_test"
			}
		}

		err = os.WriteFile(testFile, []byte("health check test"), 0644)
		if err != nil {
			result.Message = fmt.Sprintf("Write test failed: %v", err)
			result.Error = err.Error()
			result.Duration = time.Since(start)
			return result
		}

		// Clean up test file
		os.Remove(testFile)
	}

	result.Status = HealthStatusHealthy
	result.Message = "Filesystem is healthy"
	result.Duration = time.Since(start)
	return result
}

// MemoryHealthCheck implements health check for memory usage
type MemoryHealthCheck struct {
	config MemoryHealthConfig
}

// NewMemoryHealthCheck creates a new memory health check
func NewMemoryHealthCheck(config MemoryHealthConfig) *MemoryHealthCheck {
	return &MemoryHealthCheck{
		config: config,
	}
}

func (mhc *MemoryHealthCheck) Name() string {
	return mhc.config.Name
}

func (mhc *MemoryHealthCheck) Dependencies() []string {
	return []string{}
}

func (mhc *MemoryHealthCheck) Config() HealthCheckConfig {
	return mhc.config.HealthCheckConfig
}

func (mhc *MemoryHealthCheck) Check(ctx context.Context) HealthResult {
	start := time.Now()
	result := HealthResult{
		Status:    HealthStatusUnhealthy,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Get memory statistics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	result.Details["alloc"] = memStats.Alloc
	result.Details["total_alloc"] = memStats.TotalAlloc
	result.Details["sys"] = memStats.Sys
	result.Details["num_gc"] = memStats.NumGC
	result.Details["heap_alloc"] = memStats.HeapAlloc
	result.Details["heap_sys"] = memStats.HeapSys
	result.Details["heap_idle"] = memStats.HeapIdle
	result.Details["heap_inuse"] = memStats.HeapInuse

	// Check memory usage against limits
	if mhc.config.MaxMemoryBytes > 0 && memStats.Alloc > mhc.config.MaxMemoryBytes {
		result.Status = HealthStatusDegraded
		result.Message = fmt.Sprintf("High memory usage: %d bytes (limit: %d bytes)", memStats.Alloc, mhc.config.MaxMemoryBytes)
		result.Duration = time.Since(start)
		return result
	}

	// Calculate memory usage percentage (this is a simplified calculation)
	if mhc.config.MaxMemoryPercent > 0 {
		usagePercent := float64(memStats.Alloc) / float64(memStats.Sys) * 100
		result.Details["usage_percent"] = usagePercent

		if usagePercent > mhc.config.MaxMemoryPercent {
			result.Status = HealthStatusDegraded
			result.Message = fmt.Sprintf("High memory usage: %.2f%% (limit: %.2f%%)", usagePercent, mhc.config.MaxMemoryPercent)
			result.Duration = time.Since(start)
			return result
		}
	}

	result.Status = HealthStatusHealthy
	result.Message = "Memory usage is healthy"
	result.Duration = time.Since(start)
	return result
}