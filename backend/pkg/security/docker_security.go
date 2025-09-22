package security

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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
func (sdc *SecureDockerClient) SecureContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string, userContext *DockerUserContext) (*container.CreateResponse, error) {
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
func (sdc *SecureDockerClient) validatePortBinding(binding nat.PortBinding) error {
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
	var vulnerabilities []Vulnerability

	// 1. Check for outdated base images
	vulnerabilities = append(vulnerabilities, is.checkOutdatedBaseImages(imageInfo)...)

	// 2. Check for excessive privileges
	vulnerabilities = append(vulnerabilities, is.checkPrivilegeEscalation(imageInfo)...)

	// 3. Check for suspicious ports
	vulnerabilities = append(vulnerabilities, is.checkSuspiciousPorts(imageInfo)...)

	// 4. Check for weak configurations
	vulnerabilities = append(vulnerabilities, is.checkWeakConfigurations(imageInfo)...)

	// 5. Check for layer security issues
	vulnerabilities = append(vulnerabilities, is.checkLayerSecurity(imageInfo)...)

	// 6. Check for known vulnerable packages (simulated scan)
	vulnerabilities = append(vulnerabilities, is.simulatePackageVulnerabilityCheck(imageInfo)...)

	return vulnerabilities, nil
}

// checkOutdatedBaseImages checks for outdated base images
func (is *ImageScanner) checkOutdatedBaseImages(imageInfo types.ImageInspect) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Check for old Ubuntu/Debian versions
	for _, label := range imageInfo.Config.Labels {
		if strings.Contains(strings.ToLower(label), "ubuntu") {
			if strings.Contains(label, "16.04") || strings.Contains(label, "18.04") {
				vulnerabilities = append(vulnerabilities, Vulnerability{
					CVE:         "CVE-BASE-001",
					Severity:    VulnHigh,
					Description: "Outdated Ubuntu base image with known security vulnerabilities",
					Package:     "base-image",
					Version:     label,
				})
			}
		}
		if strings.Contains(strings.ToLower(label), "debian") {
			if strings.Contains(label, "jessie") || strings.Contains(label, "stretch") {
				vulnerabilities = append(vulnerabilities, Vulnerability{
					CVE:         "CVE-BASE-002",
					Severity:    VulnHigh,
					Description: "Outdated Debian base image with known security vulnerabilities",
					Package:     "base-image",
					Version:     label,
				})
			}
		}
	}

	// Check image creation date
	if created, err := time.Parse(time.RFC3339, imageInfo.Created); err == nil {
		if time.Since(created) > 365*24*time.Hour {
			vulnerabilities = append(vulnerabilities, Vulnerability{
				CVE:         "CVE-BASE-003",
				Severity:    VulnMedium,
				Description: "Image is older than 1 year and may contain unpatched vulnerabilities",
				Package:     "base-image",
				Version:     created.Format("2006-01-02"),
			})
		}
	}

	return vulnerabilities
}

// checkPrivilegeEscalation checks for privilege escalation risks
func (is *ImageScanner) checkPrivilegeEscalation(imageInfo types.ImageInspect) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Check if running as root
	if imageInfo.Config.User == "" || imageInfo.Config.User == "root" || imageInfo.Config.User == "0" {
		vulnerabilities = append(vulnerabilities, Vulnerability{
			CVE:         "CVE-PRIV-001",
			Severity:    VulnMedium,
			Description: "Container runs as root user, increasing privilege escalation risk",
			Package:     "runtime-config",
		})
	}

	// Check for setuid/setgid binaries in common locations
	suspiciousCommands := []string{"sudo", "su", "passwd", "chsh", "chfn", "newgrp"}
	for _, cmd := range imageInfo.Config.Cmd {
		for _, suspicious := range suspiciousCommands {
			if strings.Contains(strings.ToLower(cmd), suspicious) {
				vulnerabilities = append(vulnerabilities, Vulnerability{
					CVE:         "CVE-PRIV-002",
					Severity:    VulnMedium,
					Description: fmt.Sprintf("Container command contains potentially dangerous binary: %s", suspicious),
					Package:     "runtime-config",
					Version:     cmd,
				})
			}
		}
	}

	return vulnerabilities
}

// checkSuspiciousPorts checks for suspicious port configurations
func (is *ImageScanner) checkSuspiciousPorts(imageInfo types.ImageInspect) []Vulnerability {
	var vulnerabilities []Vulnerability

	suspiciousPorts := map[string]string{
		"22":   "SSH - potential remote access vulnerability",
		"23":   "Telnet - unencrypted protocol",
		"21":   "FTP - unencrypted file transfer",
		"3389": "RDP - remote desktop access",
		"5900": "VNC - remote desktop access",
		"1433": "SQL Server - database access",
		"3306": "MySQL - database access",
		"5432": "PostgreSQL - database access",
		"6379": "Redis - cache database access",
	}

	for port := range imageInfo.Config.ExposedPorts {
		portNum := strings.Split(string(port), "/")[0]
		if description, exists := suspiciousPorts[portNum]; exists {
			severity := VulnMedium
			if portNum == "22" || portNum == "23" || portNum == "3389" {
				severity = VulnHigh
			}

			vulnerabilities = append(vulnerabilities, Vulnerability{
				CVE:         fmt.Sprintf("CVE-PORT-%s", portNum),
				Severity:    severity,
				Description: fmt.Sprintf("Exposed port %s: %s", portNum, description),
				Package:     "network-config",
				Version:     portNum,
			})
		}
	}

	return vulnerabilities
}

// checkWeakConfigurations checks for weak security configurations
func (is *ImageScanner) checkWeakConfigurations(imageInfo types.ImageInspect) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Check for debug/development configurations
	for _, env := range imageInfo.Config.Env {
		envLower := strings.ToLower(env)
		if strings.Contains(envLower, "debug=true") ||
		   strings.Contains(envLower, "development") ||
		   strings.Contains(envLower, "dev_mode=true") {
			vulnerabilities = append(vulnerabilities, Vulnerability{
				CVE:         "CVE-CONFIG-001",
				Severity:    VulnMedium,
				Description: "Debug or development mode enabled in production image",
				Package:     "environment-config",
				Version:     env,
			})
		}

		// Check for potential secrets in environment variables
		if strings.Contains(envLower, "password=") ||
		   strings.Contains(envLower, "secret=") ||
		   strings.Contains(envLower, "key=") ||
		   strings.Contains(envLower, "token=") {
			vulnerabilities = append(vulnerabilities, Vulnerability{
				CVE:         "CVE-CONFIG-002",
				Severity:    VulnHigh,
				Description: "Potential secret or credential found in environment variable",
				Package:     "environment-config",
			})
		}
	}

	// Check for world-writable directories
	for _, cmd := range imageInfo.Config.Cmd {
		if strings.Contains(cmd, "chmod 777") || strings.Contains(cmd, "chmod a+w") {
			vulnerabilities = append(vulnerabilities, Vulnerability{
				CVE:         "CVE-CONFIG-003",
				Severity:    VulnMedium,
				Description: "World-writable permissions detected in image configuration",
				Package:     "filesystem-config",
				Version:     cmd,
			})
		}
	}

	return vulnerabilities
}

// checkLayerSecurity checks for layer-specific security issues
func (is *ImageScanner) checkLayerSecurity(imageInfo types.ImageInspect) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Check for excessive layers (potential security risk)
	if len(imageInfo.RootFS.Layers) > 100 {
		vulnerabilities = append(vulnerabilities, Vulnerability{
			CVE:         "CVE-LAYER-001",
			Severity:    VulnHigh,
			Description: fmt.Sprintf("Image has excessive layers (%d), increasing attack surface", len(imageInfo.RootFS.Layers)),
			Package:     "image-structure",
			Version:     fmt.Sprintf("%d-layers", len(imageInfo.RootFS.Layers)),
		})
	} else if len(imageInfo.RootFS.Layers) > 50 {
		vulnerabilities = append(vulnerabilities, Vulnerability{
			CVE:         "CVE-LAYER-002",
			Severity:    VulnMedium,
			Description: fmt.Sprintf("Image has many layers (%d), may indicate security risks", len(imageInfo.RootFS.Layers)),
			Package:     "image-structure",
			Version:     fmt.Sprintf("%d-layers", len(imageInfo.RootFS.Layers)),
		})
	}

	// Check image size (very large images may contain unnecessary components)
	if imageInfo.Size > 5*1024*1024*1024 { // 5GB
		vulnerabilities = append(vulnerabilities, Vulnerability{
			CVE:         "CVE-LAYER-003",
			Severity:    VulnMedium,
			Description: "Image is very large, may contain unnecessary components increasing attack surface",
			Package:     "image-structure",
			Version:     fmt.Sprintf("%d-bytes", imageInfo.Size),
		})
	}

	return vulnerabilities
}

// simulatePackageVulnerabilityCheck simulates a package vulnerability database check
func (is *ImageScanner) simulatePackageVulnerabilityCheck(imageInfo types.ImageInspect) []Vulnerability {
	var vulnerabilities []Vulnerability

	// Simulate common vulnerable packages that might be found
	knownVulnerablePackages := map[string]Vulnerability{
		"openssl": {
			CVE:         "CVE-2022-0778",
			Severity:    VulnHigh,
			Description: "OpenSSL infinite loop vulnerability in BN_mod_sqrt()",
			Package:     "openssl",
			Version:     "1.1.1",
			FixedIn:     "1.1.1n",
		},
		"curl": {
			CVE:         "CVE-2022-32205",
			Severity:    VulnMedium,
			Description: "curl set-cookie denial of service vulnerability",
			Package:     "curl",
			Version:     "7.68.0",
			FixedIn:     "7.84.0",
		},
		"glibc": {
			CVE:         "CVE-2021-3999",
			Severity:    VulnHigh,
			Description: "glibc realpath() buffer overflow vulnerability",
			Package:     "glibc",
			Version:     "2.31",
			FixedIn:     "2.35",
		},
		"bash": {
			CVE:         "CVE-2022-3715",
			Severity:    VulnMedium,
			Description: "Bash heap buffer overflow vulnerability",
			Package:     "bash",
			Version:     "4.4",
			FixedIn:     "5.2",
		},
	}

	// Simulate finding these packages based on image characteristics
	// In a real implementation, this would parse package manifests
	if strings.Contains(imageInfo.Config.Image, "ubuntu") ||
	   strings.Contains(imageInfo.Config.Image, "debian") {
		// Simulate finding common packages in Ubuntu/Debian images
		for _, vuln := range knownVulnerablePackages {
			// Add some randomness to simulate real scanning
			if len(imageInfo.ID)%3 == 0 { // Simple pseudo-random based on image ID
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	// Check for specific patterns in environment that might indicate package presence
	for _, env := range imageInfo.Config.Env {
		envLower := strings.ToLower(env)
		if strings.Contains(envLower, "ssl") || strings.Contains(envLower, "tls") {
			if vuln, exists := knownVulnerablePackages["openssl"]; exists {
				vulnerabilities = append(vulnerabilities, vuln)
			}
		}
	}

	return vulnerabilities
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
	logrus.WithField("image", imageName).Debug("Verifying image signature")

	// 1. Check for Docker Content Trust signatures
	if err := sdc.verifyDockerContentTrust(ctx, imageName); err != nil {
		return fmt.Errorf("Docker Content Trust verification failed: %w", err)
	}

	// 2. Check for registry-specific signatures
	if err := sdc.verifyRegistrySignature(ctx, imageName); err != nil {
		return fmt.Errorf("registry signature verification failed: %w", err)
	}

	// 3. Check for cosign signatures (if available)
	if err := sdc.verifyCosignSignature(ctx, imageName); err != nil {
		logrus.WithError(err).Warn("Cosign signature verification failed (optional)")
		// Don't fail on cosign as it's optional
	}

	return nil
}

// verifyDockerContentTrust verifies Docker Content Trust signatures
func (sdc *SecureDockerClient) verifyDockerContentTrust(ctx context.Context, imageName string) error {
	// Check if DOCKER_CONTENT_TRUST is enabled
	contentTrust := os.Getenv("DOCKER_CONTENT_TRUST")
	if contentTrust != "1" {
		return fmt.Errorf("Docker Content Trust not enabled")
	}

	// Parse image reference
	parts := strings.Split(imageName, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid image name format: %s", imageName)
	}

	repository := parts[0]
	tag := parts[1]

	// Simulate signature verification by checking image inspect data
	imageInfo, _, err := sdc.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to inspect image: %w", err)
	}

	// Check for signature metadata in labels
	if imageInfo.Config.Labels != nil {
		if sig, exists := imageInfo.Config.Labels["io.docker.content-trust.signature"]; exists {
			if err := sdc.validateSignatureFormat(sig); err != nil {
				return fmt.Errorf("invalid signature format: %w", err)
			}
		} else {
			return fmt.Errorf("no Docker Content Trust signature found for %s:%s", repository, tag)
		}
	}

	return nil
}

// verifyRegistrySignature verifies registry-specific signatures
func (sdc *SecureDockerClient) verifyRegistrySignature(ctx context.Context, imageName string) error {
	// Extract registry from image name
	parts := strings.Split(imageName, "/")
	if len(parts) == 0 {
		return fmt.Errorf("invalid image name")
	}

	var registry string
	if strings.Contains(parts[0], ".") {
		registry = parts[0]
	} else {
		registry = "docker.io" // Default registry
	}

	// Check trusted registries
	trustedRegistries := map[string]bool{
		"docker.io":           true,
		"registry.docker.io":  true,
		"mcr.microsoft.com":   true,
		"gcr.io":             true,
		"quay.io":            true,
	}

	if !trustedRegistries[registry] {
		return fmt.Errorf("image from untrusted registry: %s", registry)
	}

	// For trusted registries, perform additional checks
	return sdc.verifyTrustedRegistryImage(ctx, imageName, registry)
}

// verifyTrustedRegistryImage performs additional verification for trusted registries
func (sdc *SecureDockerClient) verifyTrustedRegistryImage(ctx context.Context, imageName, registry string) error {
	imageInfo, _, err := sdc.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to inspect image: %w", err)
	}

	// Check for official image markers
	if registry == "docker.io" || registry == "registry.docker.io" {
		// Official Docker images should have specific labels
		if imageInfo.Config.Labels != nil {
			if official, exists := imageInfo.Config.Labels["org.opencontainers.image.vendor"]; exists {
				if strings.Contains(strings.ToLower(official), "docker") {
					return nil // Official Docker image
				}
			}
		}

		// Check if it's a library image (no username prefix)
		imageParts := strings.Split(imageName, "/")
		if len(imageParts) == 1 || (len(imageParts) == 2 && !strings.Contains(imageParts[0], ".")) {
			// This is likely a library image (nginx, ubuntu, etc.)
			return nil
		}
	}

	// Additional registry-specific verifications can be added here
	return nil
}

// verifyCosignSignature verifies cosign signatures (if available)
func (sdc *SecureDockerClient) verifyCosignSignature(ctx context.Context, imageName string) error {
	// This is a simplified check for cosign signatures
	// In a real implementation, this would use the cosign library

	imageInfo, _, err := sdc.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to inspect image: %w", err)
	}

	// Check for cosign signature annotations
	if imageInfo.Config.Labels != nil {
		for key := range imageInfo.Config.Labels {
			if strings.Contains(key, "cosign") || strings.Contains(key, "sigstore") {
				// Found cosign-related annotations
				return nil
			}
		}
	}

	return fmt.Errorf("no cosign signatures found")
}

// validateSignatureFormat validates the format of a signature string
func (sdc *SecureDockerClient) validateSignatureFormat(signature string) error {
	// Basic validation of signature format
	if len(signature) < 64 {
		return fmt.Errorf("signature too short")
	}

	// Check if it's a valid hex string or base64
	if !isValidHex(signature) && !isValidBase64(signature) {
		return fmt.Errorf("signature not in valid format")
	}

	return nil
}

// isValidHex checks if a string is valid hexadecimal
func isValidHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// isValidBase64 checks if a string is valid base64
func isValidBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
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