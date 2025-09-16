package security

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

// DockerSecurityConfig represents Docker security configuration
type DockerSecurityConfig struct {
	// Socket security
	SocketPath        string        `json:"socket_path"`
	SocketPermissions string        `json:"socket_permissions"`
	SocketOwner       string        `json:"socket_owner"`
	TLSEnabled        bool          `json:"tls_enabled"`
	TLSConfig         *tls.Config   `json:"-"`
	TLSCertPath       string        `json:"tls_cert_path"`
	TLSKeyPath        string        `json:"tls_key_path"`
	TLSCAPath         string        `json:"tls_ca_path"`

	// Access control
	UserNamespacing   bool          `json:"user_namespacing"`
	RestrictedUsers   []string      `json:"restricted_users"`
	AllowedUsers      []string      `json:"allowed_users"`
	RequireAuth       bool          `json:"require_auth"`

	// Container security
	ResourceLimits    ResourceLimits `json:"resource_limits"`
	SecurityOpts      []string       `json:"security_opts"`
	ReadOnlyRootFS    bool          `json:"read_only_root_fs"`
	NoNewPrivileges   bool          `json:"no_new_privileges"`
	DropCapabilities  []string      `json:"drop_capabilities"`
	AddCapabilities   []string      `json:"add_capabilities"`

	// Network security
	NetworkSecurity   NetworkSecurity `json:"network_security"`
	DisableNetworking bool           `json:"disable_networking"`
	AllowedNetworks   []string       `json:"allowed_networks"`
	RestrictedPorts   []int          `json:"restricted_ports"`

	// Image security
	ImageScanning     bool          `json:"image_scanning"`
	SignedImagesOnly  bool          `json:"signed_images_only"`
	AllowedRegistries []string      `json:"allowed_registries"`
	BlockedImages     []string      `json:"blocked_images"`
	VulnerabilityThreshold VulnerabilityLevel `json:"vulnerability_threshold"`

	// Runtime security
	AppArmorProfile   string        `json:"apparmor_profile"`
	SELinuxLabels     []string      `json:"selinux_labels"`
	SeccompProfile    string        `json:"seccomp_profile"`

	// Monitoring
	AuditEnabled      bool          `json:"audit_enabled"`
	LogLevel          string        `json:"log_level"`
	MonitorContainers bool          `json:"monitor_containers"`
	AlertOnSuspicious bool          `json:"alert_on_suspicious"`

	// Cleanup policies
	AutoCleanup       bool          `json:"auto_cleanup"`
	MaxContainerAge   time.Duration `json:"max_container_age"`
	MaxImageAge       time.Duration `json:"max_image_age"`
}

// ResourceLimits represents container resource limits
type ResourceLimits struct {
	CPULimit      int64 `json:"cpu_limit"`       // CPU limit in nano CPUs
	MemoryLimit   int64 `json:"memory_limit"`    // Memory limit in bytes
	DiskLimit     int64 `json:"disk_limit"`      // Disk limit in bytes
	PIDsLimit     int64 `json:"pids_limit"`      // Max number of PIDs
	ULimitNoFile  int64 `json:"ulimit_nofile"`   // File descriptor limit
	ULimitNProc   int64 `json:"ulimit_nproc"`    // Process limit
}

// NetworkSecurity represents network security configuration
type NetworkSecurity struct {
	IsolateContainers bool     `json:"isolate_containers"`
	AllowedCIDRs      []string `json:"allowed_cidrs"`
	BlockedCIDRs      []string `json:"blocked_cidrs"`
	DNSServers        []string `json:"dns_servers"`
	SearchDomains     []string `json:"search_domains"`
}

// VulnerabilityLevel represents vulnerability severity levels
type VulnerabilityLevel int

const (
	VulnNone VulnerabilityLevel = iota
	VulnLow
	VulnMedium
	VulnHigh
	VulnCritical
)

// DefaultDockerSecurityConfig returns secure default configuration
func DefaultDockerSecurityConfig() *DockerSecurityConfig {
	return &DockerSecurityConfig{
		SocketPath:        "/var/run/docker.sock",
		SocketPermissions: "660",
		SocketOwner:       "root:docker",
		TLSEnabled:        false, // Enable in production
		UserNamespacing:   true,
		RequireAuth:       true,
		ResourceLimits: ResourceLimits{
			CPULimit:     1000000000, // 1 CPU
			MemoryLimit:  536870912,  // 512MB
			DiskLimit:    1073741824, // 1GB
			PIDsLimit:    100,
			ULimitNoFile: 1024,
			ULimitNProc:  64,
		},
		SecurityOpts: []string{
			"no-new-privileges:true",
			"apparmor:docker-default",
		},
		ReadOnlyRootFS:    true,
		NoNewPrivileges:   true,
		DropCapabilities: []string{
			"ALL",
		},
		AddCapabilities: []string{
			// Add only necessary capabilities
		},
		NetworkSecurity: NetworkSecurity{
			IsolateContainers: true,
			AllowedCIDRs:     []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
		},
		ImageScanning:          true,
		SignedImagesOnly:       false, // Enable in production
		AllowedRegistries:      []string{"docker.io", "registry.docker.io"},
		VulnerabilityThreshold: VulnHigh,
		AppArmorProfile:        "docker-default",
		SeccompProfile:         "default",
		AuditEnabled:           true,
		LogLevel:              "info",
		MonitorContainers:     true,
		AlertOnSuspicious:     true,
		AutoCleanup:           true,
		MaxContainerAge:       24 * time.Hour,
		MaxImageAge:           7 * 24 * time.Hour,
	}
}

// SecureDockerClient represents a secure Docker client wrapper
type SecureDockerClient struct {
	config       *DockerSecurityConfig
	client       *client.Client
	auditLogger  *DockerAuditLogger
	scanner      *ImageScanner
	stats        *DockerSecurityStats
	mutex        sync.RWMutex
}

// DockerSecurityStats represents Docker security statistics
type DockerSecurityStats struct {
	TotalOperations     int64     `json:"total_operations"`
	BlockedOperations   int64     `json:"blocked_operations"`
	ScannedImages       int64     `json:"scanned_images"`
	VulnerableImages    int64     `json:"vulnerable_images"`
	ContainersCreated   int64     `json:"containers_created"`
	ContainersBlocked   int64     `json:"containers_blocked"`
	SecurityViolations  int64     `json:"security_violations"`
	LastUpdate          time.Time `json:"last_update"`
}

// DockerAuditLogger handles Docker operation audit logging
type DockerAuditLogger struct {
	enabled bool
	logger  *logrus.Logger
}

// ImageScanner handles container image security scanning
type ImageScanner struct {
	config  *DockerSecurityConfig
	client  *client.Client
	results map[string]*ScanResult
	mutex   sync.RWMutex
}

// ScanResult represents image scan results
type ScanResult struct {
	ImageID         string            `json:"image_id"`
	ImageName       string            `json:"image_name"`
	ScanTime        time.Time         `json:"scan_time"`
	Vulnerabilities []Vulnerability   `json:"vulnerabilities"`
	Passed          bool              `json:"passed"`
	TotalVulns      int               `json:"total_vulns"`
	CriticalVulns   int               `json:"critical_vulns"`
	HighVulns       int               `json:"high_vulns"`
	MediumVulns     int               `json:"medium_vulns"`
	LowVulns        int               `json:"low_vulns"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	CVE         string             `json:"cve"`
	Severity    VulnerabilityLevel `json:"severity"`
	Description string             `json:"description"`
	Package     string             `json:"package"`
	Version     string             `json:"version"`
	FixedIn     string             `json:"fixed_in,omitempty"`
}

// NewSecureDockerClient creates a new secure Docker client
func NewSecureDockerClient(config *DockerSecurityConfig) (*SecureDockerClient, error) {
	if config == nil {
		config = DefaultDockerSecurityConfig()
	}

	// Configure Docker client options
	clientOpts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}

	// Configure TLS if enabled
	if config.TLSEnabled {
		tlsConfig, err := configureTLS(config)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
		clientOpts = append(clientOpts, client.WithHTTPClient(httpClient))
	}

	// Create Docker client
	dockerClient, err := client.NewClientWithOpts(clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Initialize audit logger
	auditLogger := &DockerAuditLogger{
		enabled: config.AuditEnabled,
		logger:  logrus.New(),
	}

	// Initialize image scanner
	scanner := &ImageScanner{
		config:  config,
		client:  dockerClient,
		results: make(map[string]*ScanResult),
	}

	secureClient := &SecureDockerClient{
		config:      config,
		client:      dockerClient,
		auditLogger: auditLogger,
		scanner:     scanner,
		stats:       &DockerSecurityStats{LastUpdate: time.Now()},
	}

	// Start monitoring if enabled
	if config.MonitorContainers {
		go secureClient.startMonitoring()
	}

	// Start cleanup if enabled
	if config.AutoCleanup {
		go secureClient.startCleanup()
	}

	return secureClient, nil
}

// configureTLS configures TLS settings for Docker client
func configureTLS(config *DockerSecurityConfig) (*tls.Config, error) {
	if config.TLSConfig != nil {
		return config.TLSConfig, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	// Load certificates if provided
	if config.TLSCertPath != "" && config.TLSKeyPath != "" {
		cert, err := tls.LoadX509KeyPair(config.TLSCertPath, config.TLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate if provided
	if config.TLSCAPath != "" {
		// Implementation for loading CA certificate
		// This requires additional certificate handling code
	}

	return tlsConfig, nil
}

// SecureContainerCreate creates a container with security checks
func (sdc *SecureDockerClient) SecureContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string, userContext *DockerUserContext) (*container.ContainerCreateCreatedBody, error) {
	sdc.mutex.Lock()
	defer sdc.mutex.Unlock()

	sdc.stats.TotalOperations++

	// Validate user permissions
	if err := sdc.validateUserPermissions(userContext, "container_create"); err != nil {
		sdc.stats.BlockedOperations++
		return nil, fmt.Errorf("permission denied: %w", err)
	}

	// Validate image
	if err := sdc.validateImage(ctx, config.Image); err != nil {
		sdc.stats.BlockedOperations++
		sdc.stats.ContainersBlocked++
		return nil, fmt.Errorf("image validation failed: %w", err)
	}

	// Apply security hardening
	if err := sdc.applySecurityHardening(config, hostConfig); err != nil {
		sdc.stats.BlockedOperations++
		return nil, fmt.Errorf("security hardening failed: %w", err)
	}

	// Validate container configuration
	if err := sdc.validateContainerConfig(config, hostConfig); err != nil {
		sdc.stats.BlockedOperations++
		sdc.stats.SecurityViolations++
		return nil, fmt.Errorf("container configuration validation failed: %w", err)
	}

	// Create container
	response, err := sdc.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		sdc.stats.BlockedOperations++
		return nil, fmt.Errorf("container creation failed: %w", err)
	}

	sdc.stats.ContainersCreated++

	// Audit log
	if sdc.config.AuditEnabled {
		sdc.auditLogger.LogOperation("container_create", userContext, map[string]interface{}{
			"container_id":   response.ID,
			"container_name": containerName,
			"image":          config.Image,
			"success":        true,
		})
	}

	logrus.WithFields(logrus.Fields{
		"container_id":   response.ID,
		"container_name": containerName,
		"image":          config.Image,
		"user_id":        userContext.UserID,
	}).Info("Secure container created")

	return &response, nil
}

// DockerUserContext represents user context for Docker operations
type DockerUserContext struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ClientIP  string `json:"client_ip"`
	SessionID string `json:"session_id"`
}

// validateUserPermissions validates user permissions for Docker operations
func (sdc *SecureDockerClient) validateUserPermissions(userContext *DockerUserContext, operation string) error {
	if !sdc.config.RequireAuth {
		return nil
	}

	// Check allowed users
	if len(sdc.config.AllowedUsers) > 0 {
		allowed := false
		for _, allowedUser := range sdc.config.AllowedUsers {
			if userContext.Username == allowedUser {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("user %s not in allowed users list", userContext.Username)
		}
	}

	// Check restricted users
	for _, restrictedUser := range sdc.config.RestrictedUsers {
		if userContext.Username == restrictedUser {
			return fmt.Errorf("user %s is restricted", userContext.Username)
		}
	}

	// Role-based access control
	switch strings.ToLower(userContext.Role) {
	case "admin":
		return nil // Admin can do anything
	case "developer":
		// Developers can create/manage containers but not system operations
		allowedOps := []string{"container_create", "container_start", "container_stop", "container_remove", "image_pull"}
		for _, op := range allowedOps {
			if operation == op {
				return nil
			}
		}
		return fmt.Errorf("operation %s not allowed for role %s", operation, userContext.Role)
	case "viewer":
		// Viewers can only inspect/list
		allowedOps := []string{"container_list", "container_inspect", "image_list", "image_inspect"}
		for _, op := range allowedOps {
			if operation == op {
				return nil
			}
		}
		return fmt.Errorf("operation %s not allowed for role %s", operation, userContext.Role)
	default:
		return fmt.Errorf("unknown role: %s", userContext.Role)
	}
}

// validateImage validates container image security
func (sdc *SecureDockerClient) validateImage(ctx context.Context, imageName string) error {
	// Check allowed registries
	if len(sdc.config.AllowedRegistries) > 0 {
		allowed := false
		for _, registry := range sdc.config.AllowedRegistries {
			if strings.HasPrefix(imageName, registry) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("image from registry not allowed: %s", imageName)
		}
	}

	// Check blocked images
	for _, blockedImage := range sdc.config.BlockedImages {
		if matched, _ := filepath.Match(blockedImage, imageName); matched {
			return fmt.Errorf("image is blocked: %s", imageName)
		}
	}

	// Scan image for vulnerabilities if enabled
	if sdc.config.ImageScanning {
		scanResult, err := sdc.scanner.ScanImage(ctx, imageName)
		if err != nil {
			logrus.WithError(err).Warn("Image scan failed")
			// Continue but log the failure
		} else if !scanResult.Passed {
			return fmt.Errorf("image failed security scan: %d vulnerabilities found", scanResult.TotalVulns)
		}
	}

	// Verify image signature if required
	if sdc.config.SignedImagesOnly {
		if err := sdc.verifyImageSignature(ctx, imageName); err != nil {
			return fmt.Errorf("image signature verification failed: %w", err)
		}
	}

	return nil
}

// applySecurityHardening applies security hardening to container configuration
func (sdc *SecureDockerClient) applySecurityHardening(config *container.Config, hostConfig *container.HostConfig) error {
	// Apply resource limits
	if hostConfig.Resources.Memory == 0 && sdc.config.ResourceLimits.MemoryLimit > 0 {
		hostConfig.Resources.Memory = sdc.config.ResourceLimits.MemoryLimit
	}

	if hostConfig.Resources.NanoCPUs == 0 && sdc.config.ResourceLimits.CPULimit > 0 {
		hostConfig.Resources.NanoCPUs = sdc.config.ResourceLimits.CPULimit
	}

	if hostConfig.Resources.PidsLimit == nil && sdc.config.ResourceLimits.PIDsLimit > 0 {
		pidsLimit := sdc.config.ResourceLimits.PIDsLimit
		hostConfig.Resources.PidsLimit = &pidsLimit
	}

	// Apply security options
	if len(sdc.config.SecurityOpts) > 0 {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, sdc.config.SecurityOpts...)
	}

	// Set read-only root filesystem
	if sdc.config.ReadOnlyRootFS {
		hostConfig.ReadonlyRootfs = true
	}

	// Drop capabilities
	if len(sdc.config.DropCapabilities) > 0 {
		if hostConfig.CapDrop == nil {
			hostConfig.CapDrop = make([]string, 0)
		}
		hostConfig.CapDrop = append(hostConfig.CapDrop, sdc.config.DropCapabilities...)
	}

	// Add only necessary capabilities
	if len(sdc.config.AddCapabilities) > 0 {
		if hostConfig.CapAdd == nil {
			hostConfig.CapAdd = make([]string, 0)
		}
		hostConfig.CapAdd = append(hostConfig.CapAdd, sdc.config.AddCapabilities...)
	}

	// Set AppArmor profile
	if sdc.config.AppArmorProfile != "" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "apparmor:"+sdc.config.AppArmorProfile)
	}

	// Set Seccomp profile
	if sdc.config.SeccompProfile != "" {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "seccomp:"+sdc.config.SeccompProfile)
	}

	// Disable networking if required
	if sdc.config.DisableNetworking {
		hostConfig.NetworkMode = "none"
	}

	return nil
}

// validateContainerConfig validates container configuration for security compliance
func (sdc *SecureDockerClient) validateContainerConfig(config *container.Config, hostConfig *container.HostConfig) error {
	// Validate privileged mode
	if hostConfig.Privileged {
		return fmt.Errorf("privileged containers are not allowed")
	}

	// Validate host network mode
	if hostConfig.NetworkMode.IsHost() {
		return fmt.Errorf("host network mode is not allowed")
	}

	// Validate bind mounts
	for _, bind := range hostConfig.Binds {
		if err := sdc.validateBindMount(bind); err != nil {
			return fmt.Errorf("invalid bind mount: %w", err)
		}
	}

	// Validate port mappings
	for _, port := range hostConfig.PortBindings {
		for _, binding := range port {
			if err := sdc.validatePortBinding(binding); err != nil {
				return fmt.Errorf("invalid port binding: %w", err)
			}
		}
	}

	// Validate environment variables
	for _, env := range config.Env {
		if err := sdc.validateEnvironmentVariable(env); err != nil {
			return fmt.Errorf("invalid environment variable: %w", err)
		}
	}

	return nil
}

// validateBindMount validates bind mount security
func (sdc *SecureDockerClient) validateBindMount(bind string) error {
	parts := strings.Split(bind, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid bind mount format")
	}

	hostPath := parts[0]

	// Restricted host paths
	restrictedPaths := []string{
		"/",
		"/bin",
		"/sbin",
		"/usr",
		"/lib",
		"/lib64",
		"/boot",
		"/dev",
		"/sys",
		"/proc",
		"/run",
		"/var/run/docker.sock",
	}

	for _, restricted := range restrictedPaths {
		if strings.HasPrefix(hostPath, restricted) {
			return fmt.Errorf("bind mount to restricted path: %s", hostPath)
		}
	}

	return nil
}

// validatePortBinding validates port binding security
func (sdc *SecureDockerClient) validatePortBinding(binding types.PortBinding) error {
	// Check restricted ports
	if binding.HostPort != "" {
		for _, restricted := range sdc.config.RestrictedPorts {
			if binding.HostPort == fmt.Sprintf("%d", restricted) {
				return fmt.Errorf("port %s is restricted", binding.HostPort)
			}
		}
	}

	// Validate IP binding
	if binding.HostIP == "" {
		// Default to localhost for security
		binding.HostIP = "127.0.0.1"
	} else {
		ip := net.ParseIP(binding.HostIP)
		if ip == nil {
			return fmt.Errorf("invalid host IP: %s", binding.HostIP)
		}
	}

	return nil
}

// validateEnvironmentVariable validates environment variable security
func (sdc *SecureDockerClient) validateEnvironmentVariable(env string) error {
	// Check for sensitive patterns
	sensitivePatterns := []string{
		"(?i)password=",
		"(?i)secret=",
		"(?i)key=.*[a-f0-9]{20,}",
		"(?i)token=",
		"(?i)api_key=",
	}

	for _, pattern := range sensitivePatterns {
		matched, _ := regexp.MatchString(pattern, env)
		if matched {
			return fmt.Errorf("potentially sensitive data in environment variable")
		}
	}

	return nil
}

// ScanImage scans a container image for vulnerabilities
func (is *ImageScanner) ScanImage(ctx context.Context, imageName string) (*ScanResult, error) {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	// Check if we already have results for this image
	if result, exists := is.results[imageName]; exists {
		// Return cached result if it's recent
		if time.Since(result.ScanTime) < 24*time.Hour {
			return result, nil
		}
	}

	// Inspect image
	imageInfo, _, err := is.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	// Create scan result
	result := &ScanResult{
		ImageID:   imageInfo.ID,
		ImageName: imageName,
		ScanTime:  time.Now(),
	}

	// Perform vulnerability scanning
	// This is a placeholder - integrate with actual vulnerability scanning tools
	// like Clair, Trivy, or commercial solutions
	vulnerabilities, err := is.performVulnerabilityScanning(imageInfo)
	if err != nil {
		return nil, fmt.Errorf("vulnerability scanning failed: %w", err)
	}

	result.Vulnerabilities = vulnerabilities
	result.TotalVulns = len(vulnerabilities)

	// Count vulnerabilities by severity
	for _, vuln := range vulnerabilities {
		switch vuln.Severity {
		case VulnCritical:
			result.CriticalVulns++
		case VulnHigh:
			result.HighVulns++
		case VulnMedium:
			result.MediumVulns++
		case VulnLow:
			result.LowVulns++
		}
	}

	// Determine if image passes security threshold
	result.Passed = is.passesThreshold(result)

	// Cache result
	is.results[imageName] = result

	return result, nil
}

// performVulnerabilityScanning performs the actual vulnerability scanning
func (is *ImageScanner) performVulnerabilityScanning(imageInfo types.ImageInspect) ([]Vulnerability, error) {
	// Placeholder implementation
	// In a real implementation, this would integrate with vulnerability databases
	// and scanning tools like:
	// - Clair
	// - Trivy
	// - Snyk
	// - Commercial solutions

	var vulnerabilities []Vulnerability

	// Example vulnerability detection based on image properties
	if len(imageInfo.RootFS.Layers) > 50 {
		vulnerabilities = append(vulnerabilities, Vulnerability{
			CVE:         "CUSTOM-001",
			Severity:    VulnMedium,
			Description: "Image has many layers which may indicate security risks",
			Package:     "base-image",
		})
	}

	return vulnerabilities, nil
}

// passesThreshold determines if scan result passes security threshold
func (is *ImageScanner) passesThreshold(result *ScanResult) bool {
	switch is.config.VulnerabilityThreshold {
	case VulnCritical:
		return result.CriticalVulns == 0
	case VulnHigh:
		return result.CriticalVulns == 0 && result.HighVulns == 0
	case VulnMedium:
		return result.CriticalVulns == 0 && result.HighVulns == 0 && result.MediumVulns == 0
	case VulnLow:
		return result.TotalVulns == 0
	default:
		return true
	}
}

// verifyImageSignature verifies the digital signature of a container image
func (sdc *SecureDockerClient) verifyImageSignature(ctx context.Context, imageName string) error {
	// Placeholder implementation
	// In a real implementation, this would integrate with:
	// - Docker Content Trust (Notary)
	// - Sigstore/Cosign
	// - Other signature verification systems

	logrus.WithField("image", imageName).Debug("Verifying image signature")
	return nil
}

// startMonitoring starts container monitoring
func (sdc *SecureDockerClient) startMonitoring() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		sdc.monitorContainers()
	}
}

// monitorContainers monitors running containers for security issues
func (sdc *SecureDockerClient) monitorContainers() {
	ctx := context.Background()
	containers, err := sdc.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list containers for monitoring")
		return
	}

	for _, container := range containers {
		if err := sdc.checkContainerSecurity(ctx, container); err != nil {
			logrus.WithFields(logrus.Fields{
				"container_id": container.ID,
				"image":        container.Image,
				"error":        err,
			}).Warn("Container security issue detected")

			if sdc.config.AlertOnSuspicious {
				sdc.handleSecurityAlert(container, err)
			}
		}
	}
}

// checkContainerSecurity checks a container for security issues
func (sdc *SecureDockerClient) checkContainerSecurity(ctx context.Context, container types.Container) error {
	// Check container age
	if sdc.config.MaxContainerAge > 0 {
		created := time.Unix(container.Created, 0)
		if time.Since(created) > sdc.config.MaxContainerAge {
			return fmt.Errorf("container exceeds maximum age: %v", time.Since(created))
		}
	}

	// Check resource usage
	stats, err := sdc.client.ContainerStats(ctx, container.ID, false)
	if err != nil {
		return fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	// Additional security checks can be added here
	// - Network activity monitoring
	// - Process monitoring
	// - File system changes
	// - Resource usage anomalies

	return nil
}

// handleSecurityAlert handles security alerts
func (sdc *SecureDockerClient) handleSecurityAlert(container types.Container, issue error) {
	alert := map[string]interface{}{
		"alert_type":   "container_security",
		"container_id": container.ID,
		"image":        container.Image,
		"issue":        issue.Error(),
		"timestamp":    time.Now(),
	}

	if sdc.config.AuditEnabled {
		sdc.auditLogger.LogOperation("security_alert", nil, alert)
	}

	// Additional alerting mechanisms can be added here:
	// - Send to monitoring systems
	// - Email/SMS notifications
	// - Integration with incident response systems
}

// startCleanup starts automatic cleanup of old containers and images
func (sdc *SecureDockerClient) startCleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sdc.performCleanup()
	}
}

// performCleanup performs cleanup of old containers and images
func (sdc *SecureDockerClient) performCleanup() {
	ctx := context.Background()

	// Clean up old containers
	if sdc.config.MaxContainerAge > 0 {
		sdc.cleanupOldContainers(ctx)
	}

	// Clean up old images
	if sdc.config.MaxImageAge > 0 {
		sdc.cleanupOldImages(ctx)
	}
}

// cleanupOldContainers removes containers older than the maximum age
func (sdc *SecureDockerClient) cleanupOldContainers(ctx context.Context) {
	containers, err := sdc.client.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		logrus.WithError(err).Error("Failed to list containers for cleanup")
		return
	}

	for _, container := range containers {
		created := time.Unix(container.Created, 0)
		if time.Since(created) > sdc.config.MaxContainerAge {
			logrus.WithFields(logrus.Fields{
				"container_id": container.ID,
				"age":          time.Since(created),
			}).Info("Removing old container")

			err := sdc.client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
				Force: true,
			})
			if err != nil {
				logrus.WithError(err).WithField("container_id", container.ID).Error("Failed to remove old container")
			}
		}
	}
}

// cleanupOldImages removes images older than the maximum age
func (sdc *SecureDockerClient) cleanupOldImages(ctx context.Context) {
	images, err := sdc.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list images for cleanup")
		return
	}

	for _, image := range images {
		created := time.Unix(image.Created, 0)
		if time.Since(created) > sdc.config.MaxImageAge {
			logrus.WithFields(logrus.Fields{
				"image_id": image.ID,
				"age":      time.Since(created),
			}).Info("Removing old image")

			_, err := sdc.client.ImageRemove(ctx, image.ID, types.ImageRemoveOptions{
				Force: true,
			})
			if err != nil {
				logrus.WithError(err).WithField("image_id", image.ID).Error("Failed to remove old image")
			}
		}
	}
}

// LogOperation logs a Docker operation for auditing
func (dal *DockerAuditLogger) LogOperation(operation string, userContext *DockerUserContext, details map[string]interface{}) {
	if !dal.enabled {
		return
	}

	logFields := logrus.Fields{
		"operation": operation,
		"timestamp": time.Now(),
	}

	if userContext != nil {
		logFields["user_id"] = userContext.UserID
		logFields["username"] = userContext.Username
		logFields["role"] = userContext.Role
		logFields["client_ip"] = userContext.ClientIP
		logFields["session_id"] = userContext.SessionID
	}

	for key, value := range details {
		logFields[key] = value
	}

	dal.logger.WithFields(logFields).Info("Docker operation")
}

// GetStats returns Docker security statistics
func (sdc *SecureDockerClient) GetStats() map[string]interface{} {
	sdc.mutex.RLock()
	defer sdc.mutex.RUnlock()

	sdc.stats.LastUpdate = time.Now()

	return map[string]interface{}{
		"operations": map[string]interface{}{
			"total":   sdc.stats.TotalOperations,
			"blocked": sdc.stats.BlockedOperations,
		},
		"containers": map[string]interface{}{
			"created": sdc.stats.ContainersCreated,
			"blocked": sdc.stats.ContainersBlocked,
		},
		"images": map[string]interface{}{
			"scanned":    sdc.stats.ScannedImages,
			"vulnerable": sdc.stats.VulnerableImages,
		},
		"security": map[string]interface{}{
			"violations": sdc.stats.SecurityViolations,
		},
		"config": map[string]interface{}{
			"tls_enabled":        sdc.config.TLSEnabled,
			"user_namespacing":   sdc.config.UserNamespacing,
			"image_scanning":     sdc.config.ImageScanning,
			"signed_images_only": sdc.config.SignedImagesOnly,
			"audit_enabled":      sdc.config.AuditEnabled,
		},
		"last_update": sdc.stats.LastUpdate,
	}
}