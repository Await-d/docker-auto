package docker

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"

	"docker-auto/internal/config"
)

// DockerClient wraps the Docker client with performance optimizations
type DockerClient struct {
	client     *client.Client
	config     *config.Config
	timeout    time.Duration
	connPool   *ConnectionPool
	operationQueue chan Operation
	workerDone chan struct{}
	metrics    *ClientMetrics
}

// ConnectionPool manages Docker client connections for performance
type ConnectionPool struct {
	mu      sync.RWMutex
	clients []*client.Client
	maxSize int
	current int
}

// Operation represents a Docker operation to be executed
type Operation struct {
	Name     string
	Func     func(*client.Client) error
	Callback func(error)
	Timeout  time.Duration
}

// ClientMetrics tracks Docker client performance
type ClientMetrics struct {
	mu                sync.RWMutex
	OperationCount    int64
	TotalDuration     time.Duration
	FailureCount      int64
	ActiveOperations  int64
	ConnectionsInUse  int64
	LastOperationTime time.Time
}

// ClientConfig holds configuration for Docker client creation
type ClientConfig struct {
	Host       string
	APIVersion string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// NewDockerClient creates a high-performance Docker client with connection pooling
func NewDockerClient(cfg *config.Config) (*DockerClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	timeout := time.Duration(cfg.Docker.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	// Configure optimized HTTP client with connection pooling
	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			ForceAttemptHTTP2:   true,
		},
	}

	// Create Docker client options
	opts := []client.Opt{
		client.WithHost(cfg.Docker.Host),
		client.WithAPIVersionNegotiation(),
		client.WithHTTPClient(httpClient),
	}

	// Set specific API version if provided
	if cfg.Docker.APIVersion != "" {
		opts = append(opts, client.WithVersion(cfg.Docker.APIVersion))
	}

	// Create primary Docker client
	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Initialize connection pool
	connPool, err := NewConnectionPool(5, opts) // Pool of 5 connections
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	dockerClientWrapper := &DockerClient{
		client:         dockerClient,
		config:         cfg,
		timeout:        timeout,
		connPool:       connPool,
		operationQueue: make(chan Operation, 1000),
		workerDone:     make(chan struct{}),
		metrics: &ClientMetrics{
			LastOperationTime: time.Now(),
		},
	}

	// Start worker goroutines for parallel operations
	for i := 0; i < 10; i++ {
		go dockerClientWrapper.operationWorker()
	}

	logrus.WithFields(logrus.Fields{
		"host":            cfg.Docker.Host,
		"api_version":     cfg.Docker.APIVersion,
		"timeout":         timeout,
		"pool_size":       5,
		"worker_count":    10,
	}).Info("High-performance Docker client initialized")

	return dockerClientWrapper, nil
}

// NewDockerClientWithConfig creates a Docker client with custom configuration
func NewDockerClientWithConfig(clientConfig ClientConfig) (*DockerClient, error) {
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
	}

	if clientConfig.Host != "" {
		opts = append(opts, client.WithHost(clientConfig.Host))
	}

	if clientConfig.APIVersion != "" {
		opts = append(opts, client.WithVersion(clientConfig.APIVersion))
	}

	if clientConfig.HTTPClient != nil {
		opts = append(opts, client.WithHTTPClient(clientConfig.HTTPClient))
	}

	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	timeout := clientConfig.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &DockerClient{
		client:  dockerClient,
		timeout: timeout,
	}, nil
}

// Close closes the Docker client and all pooled connections
func (d *DockerClient) Close() error {
	// Signal workers to stop
	close(d.workerDone)

	// Close connection pool
	if d.connPool != nil {
		if err := d.connPool.Close(); err != nil {
			logrus.WithError(err).Warn("Failed to close connection pool")
		}
	}

	// Close primary client
	if d.client != nil {
		return d.client.Close()
	}
	return nil
}

// GetClient returns the underlying Docker client
func (d *DockerClient) GetClient() *client.Client {
	return d.client
}

// GetTimeout returns the client timeout
func (d *DockerClient) GetTimeout() time.Duration {
	return d.timeout
}

// Ping checks if the Docker daemon is accessible
func (d *DockerClient) Ping(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), d.timeout)
		defer cancel()
	}

	_, err := d.client.Ping(ctx)
	if err != nil {
		return fmt.Errorf("failed to ping Docker daemon: %w", err)
	}
	return nil
}

// GetVersion returns Docker version information
func (d *DockerClient) GetVersion(ctx context.Context) (*types.Version, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), d.timeout)
		defer cancel()
	}

	version, err := d.client.ServerVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker version: %w", err)
	}
	return &version, nil
}

// GetInfo returns Docker system information
func (d *DockerClient) GetInfo(ctx context.Context) (*types.Info, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), d.timeout)
		defer cancel()
	}

	info, err := d.client.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Docker info: %w", err)
	}
	return &info, nil
}

// IsAvailable checks if Docker daemon is available and responding
func (d *DockerClient) IsAvailable(ctx context.Context) bool {
	return d.Ping(ctx) == nil
}

// GetClientVersion returns the Docker client version
func (d *DockerClient) GetClientVersion() string {
	return d.client.ClientVersion()
}

// NegotiateAPIVersion negotiates the API version with the Docker daemon
func (d *DockerClient) NegotiateAPIVersion(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), d.timeout)
		defer cancel()
	}

	d.client.NegotiateAPIVersion(ctx)
	return nil
}

// GetDaemonHost returns the Docker daemon host
func (d *DockerClient) GetDaemonHost() string {
	return d.client.DaemonHost()
}

// WithTimeout creates a new context with the client's default timeout
func (d *DockerClient) WithTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, d.timeout)
}

// WithCustomTimeout creates a new context with a custom timeout
func (d *DockerClient) WithCustomTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}

// HealthCheck performs a comprehensive health check of the Docker daemon
func (d *DockerClient) HealthCheck(ctx context.Context) (*HealthStatus, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), d.timeout)
		defer cancel()
	}

	health := &HealthStatus{
		Timestamp: time.Now(),
	}

	// Check ping
	start := time.Now()
	err := d.Ping(ctx)
	health.PingDuration = time.Since(start)
	health.Available = err == nil

	if !health.Available {
		health.Error = err.Error()
		return health, nil
	}

	// Get version info
	version, err := d.GetVersion(ctx)
	if err != nil {
		health.Error = fmt.Sprintf("failed to get version: %v", err)
		return health, nil
	}
	health.Version = version.Version
	health.APIVersion = version.APIVersion

	// Get system info
	info, err := d.GetInfo(ctx)
	if err != nil {
		health.Error = fmt.Sprintf("failed to get info: %v", err)
		return health, nil
	}

	health.ContainersRunning = info.ContainersRunning
	health.ContainersPaused = info.ContainersPaused
	health.ContainersStopped = info.ContainersStopped
	health.Images = info.Images
	health.MemTotal = info.MemTotal
	health.NCPU = info.NCPU

	return health, nil
}

// HealthStatus represents the health status of Docker daemon
type HealthStatus struct {
	Timestamp         time.Time     `json:"timestamp"`
	Available         bool          `json:"available"`
	PingDuration      time.Duration `json:"ping_duration"`
	Version           string        `json:"version,omitempty"`
	APIVersion        string        `json:"api_version,omitempty"`
	ContainersRunning int           `json:"containers_running"`
	ContainersPaused  int           `json:"containers_paused"`
	ContainersStopped int           `json:"containers_stopped"`
	Images            int           `json:"images"`
	MemTotal          int64         `json:"mem_total"`
	NCPU              int           `json:"ncpu"`
	Error             string        `json:"error,omitempty"`
}

// IsHealthy returns true if the Docker daemon is healthy
func (h *HealthStatus) IsHealthy() bool {
	return h.Available && h.Error == ""
}

// GetTotalContainers returns the total number of containers
func (h *HealthStatus) GetTotalContainers() int {
	return h.ContainersRunning + h.ContainersPaused + h.ContainersStopped
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(size int, opts []client.Opt) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		maxSize: size,
		clients: make([]*client.Client, 0, size),
	}

	// Create initial connections
	for i := 0; i < size; i++ {
		cli, err := client.NewClientWithOpts(opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create pooled client %d: %w", i, err)
		}
		pool.clients = append(pool.clients, cli)
	}

	return pool, nil
}

// GetClient retrieves a client from the pool
func (p *ConnectionPool) GetClient() *client.Client {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.clients) > 0 {
		client := p.clients[0]
		p.clients = p.clients[1:]
		p.current++
		return client
	}

	// Pool exhausted, return nil (caller should handle)
	return nil
}

// ReturnClient returns a client to the pool
func (p *ConnectionPool) ReturnClient(cli *client.Client) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.clients) < p.maxSize {
		p.clients = append(p.clients, cli)
		p.current--
	}
}

// Close closes all clients in the pool
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, cli := range p.clients {
		if err := cli.Close(); err != nil {
			logrus.WithError(err).Warn("Failed to close pooled Docker client")
		}
	}
	p.clients = nil
	return nil
}

// operationWorker processes Docker operations concurrently
func (d *DockerClient) operationWorker() {
	for {
		select {
		case op := <-d.operationQueue:
			d.executeOperation(op)
		case <-d.workerDone:
			return
		}
	}
}

// executeOperation executes a Docker operation with metrics tracking
func (d *DockerClient) executeOperation(op Operation) {
	start := time.Now()

	// Update metrics
	d.metrics.mu.Lock()
	d.metrics.ActiveOperations++
	d.metrics.mu.Unlock()

	defer func() {
		duration := time.Since(start)
		d.metrics.mu.Lock()
		d.metrics.ActiveOperations--
		d.metrics.OperationCount++
		d.metrics.TotalDuration += duration
		d.metrics.LastOperationTime = time.Now()
		d.metrics.mu.Unlock()

		logrus.WithFields(logrus.Fields{
			"operation": op.Name,
			"duration":  duration,
		}).Debug("Docker operation completed")
	}()

	// Get client from pool or use primary client
	cli := d.connPool.GetClient()
	if cli == nil {
		cli = d.client
	} else {
		defer d.connPool.ReturnClient(cli)
		d.metrics.mu.Lock()
		d.metrics.ConnectionsInUse++
		d.metrics.mu.Unlock()
		defer func() {
			d.metrics.mu.Lock()
			d.metrics.ConnectionsInUse--
			d.metrics.mu.Unlock()
		}()
	}

	// Execute with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), op.Timeout)
	defer cancel()

	// Execute operation (ctx could be used by op.Func if needed)
	_ = ctx // Suppress unused variable warning
	err := op.Func(cli)
	if err != nil {
		d.metrics.mu.Lock()
		d.metrics.FailureCount++
		d.metrics.mu.Unlock()
	}

	// Call callback if provided
	if op.Callback != nil {
		op.Callback(err)
	}
}

// ExecuteAsync executes a Docker operation asynchronously
func (d *DockerClient) ExecuteAsync(name string, timeout time.Duration, operation func(*client.Client) error, callback func(error)) error {
	if timeout <= 0 {
		timeout = d.timeout
	}

	op := Operation{
		Name:     name,
		Func:     operation,
		Callback: callback,
		Timeout:  timeout,
	}

	select {
	case d.operationQueue <- op:
		return nil
	default:
		return fmt.Errorf("operation queue is full")
	}
}

// ExecuteSync executes a Docker operation synchronously with performance tracking
func (d *DockerClient) ExecuteSync(name string, timeout time.Duration, operation func(*client.Client) error) error {
	start := time.Now()

	defer func() {
		duration := time.Since(start)
		d.metrics.mu.Lock()
		d.metrics.OperationCount++
		d.metrics.TotalDuration += duration
		d.metrics.LastOperationTime = time.Now()
		d.metrics.mu.Unlock()

		logrus.WithFields(logrus.Fields{
			"operation": name,
			"duration":  duration,
			"sync":      true,
		}).Debug("Docker sync operation completed")
	}()

	if timeout <= 0 {
		timeout = d.timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_ = ctx // Suppress unused variable warning
	err := operation(d.client)
	if err != nil {
		d.metrics.mu.Lock()
		d.metrics.FailureCount++
		d.metrics.mu.Unlock()
	}

	return err
}

// GetMetrics returns current client metrics
func (d *DockerClient) GetMetrics() *ClientMetrics {
	d.metrics.mu.RLock()
	defer d.metrics.mu.RUnlock()

	return &ClientMetrics{
		OperationCount:    d.metrics.OperationCount,
		TotalDuration:     d.metrics.TotalDuration,
		FailureCount:      d.metrics.FailureCount,
		ActiveOperations:  d.metrics.ActiveOperations,
		ConnectionsInUse:  d.metrics.ConnectionsInUse,
		LastOperationTime: d.metrics.LastOperationTime,
	}
}

// GetAverageOperationTime calculates average operation time
func (m *ClientMetrics) GetAverageOperationTime() time.Duration {
	if m.OperationCount == 0 {
		return 0
	}
	return m.TotalDuration / time.Duration(m.OperationCount)
}

// GetFailureRate calculates operation failure rate
func (m *ClientMetrics) GetFailureRate() float64 {
	if m.OperationCount == 0 {
		return 0
	}
	return float64(m.FailureCount) / float64(m.OperationCount)
}