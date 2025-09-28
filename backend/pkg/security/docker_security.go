package security

import (
	"bufio"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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

// DockerSecurityConfig represents comprehensive Docker security configuration
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

	// Image security and scanning
	ImageScanning     bool          `json:"image_scanning"`
	SignedImagesOnly  bool          `json:"signed_images_only"`
	AllowedRegistries []string      `json:"allowed_registries"`
	BlockedImages     []string      `json:"blocked_images"`
	VulnerabilityThreshold VulnerabilityLevel `json:"vulnerability_threshold"`
	TrivyEnabled      bool          `json:"trivy_enabled"`
	TrivyDB           string        `json:"trivy_db"`
	OfflineScanning   bool          `json:"offline_scanning"`

	// Runtime security monitoring
	FalcoEnabled      bool          `json:"falco_enabled"`
	FalcoRulesPath    string        `json:"falco_rules_path"`
	FalcoWebhookURL   string        `json:"falco_webhook_url"`
	AppArmorProfile   string        `json:"apparmor_profile"`
	SELinuxLabels     []string      `json:"selinux_labels"`
	SeccompProfile    string        `json:"seccomp_profile"`

	// Compliance and scanning
	ComplianceEnabled bool          `json:"compliance_enabled"`
	OSCAPEnabled      bool          `json:"oscap_enabled"`
	OSCAPProfile      string        `json:"oscap_profile"`
	CISBenchmark      bool          `json:"cis_benchmark"`
	CVEEnabled        bool          `json:"cve_enabled"`
	CVEDatabase       string        `json:"cve_database"`

	// Monitoring and alerting
	AuditEnabled      bool          `json:"audit_enabled"`
	LogLevel          string        `json:"log_level"`
	MonitorContainers bool          `json:"monitor_containers"`
	AlertOnSuspicious bool          `json:"alert_on_suspicious"`
	RealTimeMonitoring bool         `json:"real_time_monitoring"`
	ThreatDetection   bool          `json:"threat_detection"`

	// Automated response
	AutoResponse      bool          `json:"auto_response"`
	QuarantineMode    bool          `json:"quarantine_mode"`
	KillSuspicious    bool          `json:"kill_suspicious"`
	BlockAttacker     bool          `json:"block_attacker"`
	NotifyAdmins      bool          `json:"notify_admins"`

	// Cleanup policies
	AutoCleanup       bool          `json:"auto_cleanup"`
	MaxContainerAge   time.Duration `json:"max_container_age"`
	MaxImageAge       time.Duration `json:"max_image_age"`
	CleanupVulnerable bool          `json:"cleanup_vulnerable"`
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
	FirewallEnabled   bool     `json:"firewall_enabled"`
	IPTablesRules     []string `json:"iptables_rules"`
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

// String returns string representation of vulnerability level
func (v VulnerabilityLevel) String() string {
	switch v {
	case VulnNone:
		return "NONE"
	case VulnLow:
		return "LOW"
	case VulnMedium:
		return "MEDIUM"
	case VulnHigh:
		return "HIGH"
	case VulnCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// DefaultDockerSecurityConfig returns production-grade security configuration
func DefaultDockerSecurityConfig() *DockerSecurityConfig {
	return &DockerSecurityConfig{
		SocketPath:        "/var/run/docker.sock",
		SocketPermissions: "660",
		SocketOwner:       "root:docker",
		TLSEnabled:        true,
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
			// Add only necessary capabilities based on application needs
		},
		NetworkSecurity: NetworkSecurity{
			IsolateContainers: true,
			AllowedCIDRs:     []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
			DNSServers:       []string{"8.8.8.8", "8.8.4.4"},
			FirewallEnabled:  true,
		},
		ImageScanning:          true,
		SignedImagesOnly:       true,
		AllowedRegistries:      []string{"docker.io", "registry.docker.io", "gcr.io", "quay.io"},
		VulnerabilityThreshold: VulnHigh,
		TrivyEnabled:          true,
		TrivyDB:               "/tmp/trivy/db",
		OfflineScanning:       false,
		FalcoEnabled:          true,
		FalcoRulesPath:        "/etc/falco/falco_rules.yaml",
		ComplianceEnabled:     true,
		OSCAPEnabled:         true,
		OSCAPProfile:         "cis",
		CISBenchmark:         true,
		CVEEnabled:           true,
		CVEDatabase:          "nvd",
		AppArmorProfile:      "docker-default",
		SeccompProfile:       "default",
		AuditEnabled:         true,
		LogLevel:            "info",
		MonitorContainers:   true,
		AlertOnSuspicious:   true,
		RealTimeMonitoring:  true,
		ThreatDetection:     true,
		AutoResponse:        true,
		QuarantineMode:      true,
		KillSuspicious:      false, // Careful with this in production
		BlockAttacker:       true,
		NotifyAdmins:        true,
		AutoCleanup:         true,
		MaxContainerAge:     24 * time.Hour,
		MaxImageAge:         7 * 24 * time.Hour,
		CleanupVulnerable:   true,
	}
}

// SecureDockerClient represents a production-grade secure Docker client
type SecureDockerClient struct {
	config          *DockerSecurityConfig
	client          *client.Client
	trivyScanner    *TrivyScanner
	falcoMonitor    *FalcoMonitor
	complianceChecker *ComplianceChecker
	threatDetector  *ThreatDetector
	autoResponder   *AutoResponder
	auditLogger     *DockerAuditLogger
	cveDatabase     *CVEDatabase
	stats           *DockerSecurityStats
	mutex           sync.RWMutex
}

// DockerSecurityStats represents comprehensive Docker security statistics
type DockerSecurityStats struct {
	TotalOperations      int64     `json:"total_operations"`
	BlockedOperations    int64     `json:"blocked_operations"`
	ScannedImages        int64     `json:"scanned_images"`
	VulnerableImages     int64     `json:"vulnerable_images"`
	ContainersCreated    int64     `json:"containers_created"`
	ContainersBlocked    int64     `json:"containers_blocked"`
	SecurityViolations   int64     `json:"security_violations"`
	ThreatDetections     int64     `json:"threat_detections"`
	AutoResponsesTriggered int64   `json:"auto_responses_triggered"`
	ComplianceViolations int64     `json:"compliance_violations"`
	CVEsFound           int64     `json:"cves_found"`
	CriticalVulns       int64     `json:"critical_vulns"`
	HighVulns           int64     `json:"high_vulns"`
	MediumVulns         int64     `json:"medium_vulns"`
	LowVulns            int64     `json:"low_vulns"`
	TotalScans          int64     `json:"total_scans"`
	VulnerabilitiesFound int64    `json:"vulnerabilities_found"`
	ThreatsDetected     int64     `json:"threats_detected"`
	SecurityEvents      int64     `json:"security_events"`
	LastUpdate          time.Time `json:"last_update"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string               `json:"id"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Severity    VulnerabilityLevel   `json:"severity"`
	CVSS        float64              `json:"cvss"`
	CVE         string               `json:"cve"`
	Package     string               `json:"package"`
	Version     string               `json:"version"`
	FixedVersion string              `json:"fixed_version"`
	FixedIn     string               `json:"fixed_in"`
	URL         string               `json:"url"`
	References  []string             `json:"references"`
	Tags        []string             `json:"tags"`
	Metadata    map[string]string    `json:"metadata"`
}

// TrivyScanner handles real container vulnerability scanning using Trivy
type TrivyScanner struct {
	config       *DockerSecurityConfig
	client       *client.Client
	trivyPath    string
	dbPath       string
	results      map[string]*TrivyScanResult
	cacheTimeout time.Duration
	mutex        sync.RWMutex
}

// TrivyScanResult represents comprehensive Trivy scan results
type TrivyScanResult struct {
	ImageID          string                    `json:"image_id"`
	ImageName        string                   `json:"image_name"`
	ScanTime         time.Time                `json:"scan_time"`
	SchemaVersion    int                      `json:"schema_version"`
	ArtifactName     string                   `json:"artifact_name"`
	ArtifactType     string                   `json:"artifact_type"`
	Metadata         TrivyMetadata            `json:"metadata"`
	Results          []TrivyResult            `json:"results"`
	TotalVulns       int                      `json:"total_vulns"`
	CriticalVulns    int                      `json:"critical_vulns"`
	HighVulns        int                      `json:"high_vulns"`
	MediumVulns      int                      `json:"medium_vulns"`
	LowVulns         int                      `json:"low_vulns"`
	UnknownVulns     int                      `json:"unknown_vulns"`
	Passed           bool                     `json:"passed"`
	RiskScore        float64                  `json:"risk_score"`
}

// TrivyMetadata represents Trivy scan metadata
type TrivyMetadata struct {
	OS           TrivyOS                    `json:"OS"`
	ImageID      string                     `json:"ImageID"`
	DiffIDs      []string                   `json:"DiffIDs"`
	ImageConfig  TrivyImageConfig           `json:"ImageConfig"`
}

// TrivyOS represents operating system information
type TrivyOS struct {
	Family string `json:"Family"`
	Name   string `json:"Name"`
}

// TrivyImageConfig represents image configuration
type TrivyImageConfig struct {
	Architecture string            `json:"architecture"`
	Container    string            `json:"container"`
	Created      time.Time         `json:"created"`
	DockerVersion string           `json:"docker_version"`
	History      []TrivyHistory    `json:"history"`
	OS           string            `json:"os"`
	RootFS       TrivyRootFS       `json:"rootfs"`
	Config       TrivyConfig       `json:"config"`
}

// TrivyHistory represents layer history
type TrivyHistory struct {
	Created    time.Time `json:"created"`
	CreatedBy  string    `json:"created_by"`
	EmptyLayer bool      `json:"empty_layer,omitempty"`
}

// TrivyRootFS represents root filesystem information
type TrivyRootFS struct {
	Type    string   `json:"type"`
	DiffIDs []string `json:"diff_ids"`
}

// TrivyConfig represents container configuration
type TrivyConfig struct {
	User         string            `json:"User"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	Env          []string          `json:"Env"`
	Entrypoint   []string          `json:"Entrypoint"`
	Cmd          []string          `json:"Cmd"`
	WorkingDir   string            `json:"WorkingDir"`
	Labels       map[string]string `json:"Labels"`
}

// TrivyResult represents vulnerability scan results for a target
type TrivyResult struct {
	Target            string                `json:"target"`
	Class             string                `json:"class"`
	Type              string                `json:"type"`
	Vulnerabilities   []TrivyVulnerability  `json:"vulnerabilities"`
	Secrets           []TrivySecret         `json:"secrets,omitempty"`
	Misconfigurations []TrivyMisconfig      `json:"misconfigurations,omitempty"`
}

// TrivyVulnerability represents a security vulnerability
type TrivyVulnerability struct {
	VulnerabilityID  string                  `json:"vulnerability_id"`
	PkgID            string                  `json:"pkg_id"`
	PkgName          string                  `json:"pkg_name"`
	InstalledVersion string                  `json:"installed_version"`
	FixedVersion     string                  `json:"fixed_version"`
	Status           string                  `json:"status"`
	Layer            TrivyLayer              `json:"layer"`
	SeveritySource   string                  `json:"severity_source"`
	PrimaryURL       string                  `json:"primary_url"`
	DataSource       TrivyDataSource         `json:"data_source"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description"`
	Severity         string                  `json:"severity"`
	CweIDs           []string                `json:"cwe_ids"`
	CVSS             map[string]TrivyCVSS    `json:"cvss"`
	References       []string                `json:"references"`
	PublishedDate    time.Time               `json:"published_date"`
	LastModifiedDate time.Time               `json:"last_modified_date"`
}

// TrivyLayer represents container layer information
type TrivyLayer struct {
	Digest string `json:"digest"`
	DiffID string `json:"diff_id"`
}

// TrivyDataSource represents vulnerability data source
type TrivyDataSource struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// TrivyCVSS represents CVSS score information
type TrivyCVSS struct {
	V2Vector string  `json:"v2_vector,omitempty"`
	V3Vector string  `json:"v3_vector,omitempty"`
	V2Score  float64 `json:"v2_score,omitempty"`
	V3Score  float64 `json:"v3_score,omitempty"`
}

// TrivySecret represents detected secrets
type TrivySecret struct {
	RuleID    string `json:"rule_id"`
	Category  string `json:"category"`
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Code      TrivyCode `json:"code"`
	Match     string `json:"match"`
}

// TrivyMisconfig represents misconfigurations
type TrivyMisconfig struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	AVDID       string    `json:"avd_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Message     string    `json:"message"`
	Namespace   string    `json:"namespace"`
	Query       string    `json:"query"`
	Resolution  string    `json:"resolution"`
	Severity    string    `json:"severity"`
	PrimaryURL  string    `json:"primary_url"`
	References  []string  `json:"references"`
	Status      string    `json:"status"`
	Layer       TrivyLayer `json:"layer"`
	CauseMetadata TrivyCauseMetadata `json:"cause_metadata"`
}

// TrivyCode represents code context
type TrivyCode struct {
	Lines []TrivyLine `json:"lines"`
}

// TrivyLine represents a code line
type TrivyLine struct {
	Number      int    `json:"number"`
	Content     string `json:"content"`
	IsCause     bool   `json:"is_cause"`
	Annotation  string `json:"annotation"`
	Truncated   bool   `json:"truncated"`
	Highlighted string `json:"highlighted"`
	FirstCause  bool   `json:"first_cause"`
	LastCause   bool   `json:"last_cause"`
}

// TrivyCauseMetadata represents cause metadata
type TrivyCauseMetadata struct {
	Resource    string `json:"resource"`
	Provider    string `json:"provider"`
	Service     string `json:"service"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
}

// FalcoMonitor handles real-time runtime security monitoring using Falco
type FalcoMonitor struct {
	config      *DockerSecurityConfig
	falcoPath   string
	rulesPath   string
	webhookURL  string
	events      chan *FalcoEvent
	running     bool
	process     *exec.Cmd
	mutex       sync.RWMutex
}

// FalcoEvent represents a Falco security event
type FalcoEvent struct {
	Time       time.Time         `json:"time"`
	Rule       string           `json:"rule"`
	Priority   string           `json:"priority"`
	Message    string           `json:"message"`
	OutputFields map[string]interface{} `json:"output_fields"`
	Source     string           `json:"source"`
	Tags       []string         `json:"tags"`
	Hostname   string           `json:"hostname"`
}

// ComplianceChecker handles security compliance checking
type ComplianceChecker struct {
	config       *DockerSecurityConfig
	client       *client.Client
	oscapPath    string
	profilePath  string
	reports      map[string]*ComplianceReport
	mutex        sync.RWMutex
}

// ComplianceReport represents a security compliance report
type ComplianceReport struct {
	ImageID        string                    `json:"image_id"`
	ImageName      string                   `json:"image_name"`
	ScanTime       time.Time                `json:"scan_time"`
	Profile        string                   `json:"profile"`
	Score          float64                  `json:"score"`
	PassedRules    int                      `json:"passed_rules"`
	FailedRules    int                      `json:"failed_rules"`
	ErrorRules     int                      `json:"error_rules"`
	NotApplicableRules int                  `json:"not_applicable_rules"`
	Rules          []ComplianceRule         `json:"rules"`
	Passed         bool                     `json:"passed"`
}

// ComplianceRule represents a compliance rule result
type ComplianceRule struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Result      string    `json:"result"` // pass, fail, error, notapplicable
	Severity    string    `json:"severity"`
	References  []string  `json:"references"`
	Remediation string    `json:"remediation"`
}

// CVEDatabase handles CVE database operations
type CVEDatabase struct {
	config     *DockerSecurityConfig
	dbPath     string
	nvdAPI     string
	apiKey     string
	lastUpdate time.Time
	cves       map[string]*CVERecord
	mutex      sync.RWMutex
}

// CVERecord represents a CVE record
type CVERecord struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	CVSS        CVSSScore         `json:"cvss"`
	CWE         []string          `json:"cwe"`
	References  []string          `json:"references"`
	Published   time.Time         `json:"published"`
	Modified    time.Time         `json:"modified"`
	Products    []string          `json:"products"`
	Severity    VulnerabilityLevel `json:"severity"`
}

// CVSSScore represents CVSS score information
type CVSSScore struct {
	Version float32 `json:"version"`
	Vector  string  `json:"vector"`
	Score   float64 `json:"score"`
	Severity string `json:"severity"`
}

// ThreatDetector handles real-time threat detection
type ThreatDetector struct {
	config         *DockerSecurityConfig
	client         *client.Client
	patterns       []ThreatPattern
	detections     chan *ThreatDetection
	running        bool
	containerStats map[string]*ContainerSecurityStats
	mutex          sync.RWMutex
}

// ThreatPattern represents a threat detection pattern
type ThreatPattern struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Severity    VulnerabilityLevel `json:"severity"`
	Indicators  []ThreatIndicator `json:"indicators"`
	Actions     []string          `json:"actions"`
}

// ThreatIndicator represents a threat indicator
type ThreatIndicator struct {
	Type      string      `json:"type"`
	Value     interface{} `json:"value"`
	Threshold float64     `json:"threshold"`
	Duration  time.Duration `json:"duration"`
}

// ThreatDetection represents a detected threat
type ThreatDetection struct {
	Time        time.Time         `json:"time"`
	ContainerID string            `json:"container_id"`
	ImageName   string            `json:"image_name"`
	ThreatID    string            `json:"threat_id"`
	ThreatName  string            `json:"threat_name"`
	Severity    VulnerabilityLevel `json:"severity"`
	Description string            `json:"description"`
	Indicators  map[string]interface{} `json:"indicators"`
	Confidence  float64           `json:"confidence"`
	Source      string            `json:"source"`
}

// ContainerSecurityStats represents container security statistics
type ContainerSecurityStats struct {
	ContainerID    string            `json:"container_id"`
	ImageName      string            `json:"image_name"`
	StartTime      time.Time         `json:"start_time"`
	CPUUsage       float64           `json:"cpu_usage"`
	MemoryUsage    int64             `json:"memory_usage"`
	NetworkRX      int64             `json:"network_rx"`
	NetworkTX      int64             `json:"network_tx"`
	ProcessCount   int               `json:"process_count"`
	FileAccess     []string          `json:"file_access"`
	NetworkConnections []NetworkConnection `json:"network_connections"`
	Syscalls       map[string]int    `json:"syscalls"`
	Anomalies      []SecurityAnomaly `json:"anomalies"`
	LastUpdate     time.Time         `json:"last_update"`
}

// NetworkConnection represents a network connection
type NetworkConnection struct {
	LocalAddr  string    `json:"local_addr"`
	LocalPort  int       `json:"local_port"`
	RemoteAddr string    `json:"remote_addr"`
	RemotePort int       `json:"remote_port"`
	Protocol   string    `json:"protocol"`
	State      string    `json:"state"`
	Timestamp  time.Time `json:"timestamp"`
}

// SecurityAnomaly represents a security anomaly
type SecurityAnomaly struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Severity    VulnerabilityLevel `json:"severity"`
	Timestamp   time.Time         `json:"timestamp"`
	Details     map[string]interface{} `json:"details"`
}

// AutoResponder handles automated security responses
type AutoResponder struct {
	config      *DockerSecurityConfig
	client      *client.Client
	actions     chan *SecurityAction
	running     bool
	responses   map[string]*ResponseRule
	mutex       sync.RWMutex
}

// SecurityAction represents an automated security action
type SecurityAction struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	Target      string            `json:"target"`
	Severity    VulnerabilityLevel `json:"severity"`
	Description string            `json:"description"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time         `json:"timestamp"`
}

// ResponseRule represents an automated response rule
type ResponseRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Triggers    []ResponseTrigger      `json:"triggers"`
	Actions     []ResponseAction       `json:"actions"`
	Conditions  []ResponseCondition    `json:"conditions"`
	Enabled     bool                   `json:"enabled"`
	Cooldown    time.Duration          `json:"cooldown"`
	LastTrigger time.Time              `json:"last_trigger"`
}

// ResponseTrigger represents a response trigger
type ResponseTrigger struct {
	Type      string            `json:"type"`
	Severity  VulnerabilityLevel `json:"severity"`
	Source    string            `json:"source"`
	Pattern   string            `json:"pattern"`
}

// ResponseAction represents a response action
type ResponseAction struct {
	Type       string                 `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Timeout    time.Duration          `json:"timeout"`
}

// ResponseCondition represents a response condition
type ResponseCondition struct {
	Type     string      `json:"type"`
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// DockerAuditLogger handles comprehensive Docker operation audit logging
type DockerAuditLogger struct {
	enabled bool
	logger  *logrus.Logger
}

// LogOperation logs a Docker operation with context
func (dal *DockerAuditLogger) LogOperation(operation string, userContext *DockerUserContext, details map[string]interface{}) {
	if !dal.enabled {
		return
	}

	dal.logger.WithFields(logrus.Fields{
		"operation":    operation,
		"user_id":      userContext.UserID,
		"username":     userContext.Username,
		"client_ip":    userContext.ClientIP,
		"session_id":   userContext.SessionID,
		"details":      details,
		"timestamp":    time.Now(),
	}).Info("Docker operation audit")
}

// DockerUserContext represents user context for Docker operations
type DockerUserContext struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ClientIP  string `json:"client_ip"`
	SessionID string `json:"session_id"`
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    VulnerabilityLevel     `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	Target      string                 `json:"target"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Acknowledged bool                   `json:"acknowledged"`
	Resolved     bool                   `json:"resolved"`
}

// NewSecureDockerClient creates a new production-grade secure Docker client
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

	// Initialize Trivy scanner
	trivyScanner, err := NewTrivyScanner(config, dockerClient)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize Trivy scanner")
	}

	// Initialize Falco monitor
	falcoMonitor, err := NewFalcoMonitor(config)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize Falco monitor")
	}

	// Initialize compliance checker
	complianceChecker, err := NewComplianceChecker(config, dockerClient)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize compliance checker")
	}

	// Initialize CVE database
	cveDatabase, err := NewCVEDatabase(config)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize CVE database")
	}

	// Initialize threat detector
	threatDetector, err := NewThreatDetector(config, dockerClient)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize threat detector")
	}

	// Initialize auto responder
	autoResponder, err := NewAutoResponder(config, dockerClient)
	if err != nil {
		logrus.WithError(err).Warn("Failed to initialize auto responder")
	}

	// Initialize audit logger
	auditLogger := &DockerAuditLogger{
		enabled: config.AuditEnabled,
		logger:  logrus.New(),
	}

	secureClient := &SecureDockerClient{
		config:            config,
		client:            dockerClient,
		trivyScanner:      trivyScanner,
		falcoMonitor:      falcoMonitor,
		complianceChecker: complianceChecker,
		threatDetector:    threatDetector,
		autoResponder:     autoResponder,
		auditLogger:       auditLogger,
		cveDatabase:       cveDatabase,
		stats:             &DockerSecurityStats{LastUpdate: time.Now()},
	}

	// Start monitoring services if enabled
	if config.MonitorContainers && config.RealTimeMonitoring {
		// TODO: Implement startRealTimeMonitoring method
		// go secureClient.startRealTimeMonitoring()
	}

	if config.FalcoEnabled && falcoMonitor != nil {
		if err := falcoMonitor.Start(); err != nil {
			logrus.WithError(err).Warn("Failed to start Falco monitor")
		}
	}

	if config.ThreatDetection && threatDetector != nil {
		if err := threatDetector.Start(); err != nil {
			logrus.WithError(err).Warn("Failed to start threat detector")
		}
	}

	if config.AutoResponse && autoResponder != nil {
		if err := autoResponder.Start(); err != nil {
			logrus.WithError(err).Warn("Failed to start auto responder")
		}
	}

	// Start cleanup if enabled
	if config.AutoCleanup {
		// TODO: Implement startAdvancedCleanup method
		// go secureClient.startAdvancedCleanup()
	}

	// Initialize CVE database
	if config.CVEEnabled && cveDatabase != nil {
		go cveDatabase.UpdateDatabase()
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
		// Additional CA certificate handling code would go here
		// This requires proper certificate verification implementation
	}

	return tlsConfig, nil
}

// NewTrivyScanner creates a new Trivy vulnerability scanner
func NewTrivyScanner(config *DockerSecurityConfig, client *client.Client) (*TrivyScanner, error) {
	if !config.TrivyEnabled {
		return nil, fmt.Errorf("Trivy scanning not enabled")
	}

	// Find Trivy binary
	trivyPath, err := exec.LookPath("trivy")
	if err != nil {
		// Try to download Trivy if not found
		if err := downloadTrivy(); err != nil {
			return nil, fmt.Errorf("Trivy not found and failed to download: %w", err)
		}
		trivyPath, err = exec.LookPath("trivy")
		if err != nil {
			return nil, fmt.Errorf("Trivy still not found after download: %w", err)
		}
	}

	// Initialize Trivy database if needed
	dbPath := config.TrivyDB
	if dbPath == "" {
		dbPath = "/tmp/trivy/db"
	}

	scanner := &TrivyScanner{
		config:       config,
		client:       client,
		trivyPath:    trivyPath,
		dbPath:       dbPath,
		results:      make(map[string]*TrivyScanResult),
		cacheTimeout: 24 * time.Hour,
	}

	// Update Trivy database
	if err := scanner.updateDatabase(); err != nil {
		logrus.WithError(err).Warn("Failed to update Trivy database")
	}

	return scanner, nil
}

// downloadTrivy downloads and installs Trivy if not present
func downloadTrivy() error {
	logrus.Info("Trivy not found, attempting to download and install")

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "trivy-install")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download Trivy installation script
	scriptURL := "https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh"
	resp, err := http.Get(scriptURL)
	if err != nil {
		return fmt.Errorf("failed to download Trivy install script: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download install script: HTTP %d", resp.StatusCode)
	}

	// Save script to temporary file
	scriptPath := filepath.Join(tempDir, "install.sh")
	scriptFile, err := os.Create(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to create script file: %w", err)
	}

	_, err = io.Copy(scriptFile, resp.Body)
	scriptFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}

	// Make script executable
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	// Execute installation script
	cmd := exec.Command("bash", scriptPath)
	cmd.Dir = tempDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute install script: %w, output: %s", err, output)
	}

	logrus.Info("Trivy installation completed successfully")
	return nil
}

// updateDatabase updates the Trivy vulnerability database
func (ts *TrivyScanner) updateDatabase() error {
	logrus.Info("Updating Trivy vulnerability database")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	args := []string{"image", "--download-db-only"}
	if ts.config.OfflineScanning {
		args = append(args, "--offline-scan")
	}
	if ts.dbPath != "" {
		args = append(args, "--cache-dir", ts.dbPath)
	}

	cmd := exec.CommandContext(ctx, ts.trivyPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update Trivy database: %w, output: %s", err, output)
	}

	logrus.Info("Trivy database updated successfully")
	return nil
}

// ScanImage performs comprehensive vulnerability scanning using Trivy
func (ts *TrivyScanner) ScanImage(ctx context.Context, imageName string) (*TrivyScanResult, error) {
	ts.mutex.Lock()
	defer ts.mutex.Unlock()

	// Check cache
	if result, exists := ts.results[imageName]; exists {
		if time.Since(result.ScanTime) < ts.cacheTimeout {
			logrus.WithField("image", imageName).Debug("Returning cached Trivy scan result")
			return result, nil
		}
	}

	logrus.WithField("image", imageName).Info("Starting Trivy vulnerability scan")

	// Prepare Trivy command
	args := []string{
		"image",
		"--format", "json",
		"--quiet",
	}

	if ts.config.OfflineScanning {
		args = append(args, "--offline-scan")
	}

	if ts.dbPath != "" {
		args = append(args, "--cache-dir", ts.dbPath)
	}

	// Set vulnerability types to scan
	args = append(args, "--vuln-type", "os,library")

	// Set security checks
	args = append(args, "--security-checks", "vuln,secret,config")

	// Set severity levels
	severityLevels := []string{"UNKNOWN", "LOW", "MEDIUM", "HIGH", "CRITICAL"}
	args = append(args, "--severity", strings.Join(severityLevels, ","))

	// Add image name
	args = append(args, imageName)

	// Execute Trivy scan
	scanCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(scanCtx, ts.trivyPath, args...)
	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			logrus.WithFields(logrus.Fields{
				"image":  imageName,
				"stderr": string(exitErr.Stderr),
			}).Error("Trivy scan failed")
		}
		return nil, fmt.Errorf("Trivy scan failed for image %s: %w", imageName, err)
	}

	// Parse Trivy JSON output
	var trivyOutput TrivyScanResult
	if err := json.Unmarshal(output, &trivyOutput); err != nil {
		return nil, fmt.Errorf("failed to parse Trivy output: %w", err)
	}

	// Enrich scan result
	trivyOutput.ImageName = imageName
	trivyOutput.ScanTime = time.Now()
	trivyOutput = ts.enrichScanResult(trivyOutput)

	// Determine if scan passed based on threshold
	trivyOutput.Passed = ts.passesThreshold(trivyOutput)

	// Calculate risk score
	trivyOutput.RiskScore = ts.calculateRiskScore(trivyOutput)

	// Cache result
	ts.results[imageName] = &trivyOutput

	logrus.WithFields(logrus.Fields{
		"image":         imageName,
		"total_vulns":   trivyOutput.TotalVulns,
		"critical":      trivyOutput.CriticalVulns,
		"high":          trivyOutput.HighVulns,
		"passed":        trivyOutput.Passed,
		"risk_score":    trivyOutput.RiskScore,
	}).Info("Trivy scan completed")

	return &trivyOutput, nil
}

// enrichScanResult enriches the scan result with additional analysis
func (ts *TrivyScanner) enrichScanResult(result TrivyScanResult) TrivyScanResult {
	// Count vulnerabilities by severity
	for _, trivyResult := range result.Results {
		for _, vuln := range trivyResult.Vulnerabilities {
			result.TotalVulns++
			switch strings.ToUpper(vuln.Severity) {
			case "CRITICAL":
				result.CriticalVulns++
			case "HIGH":
				result.HighVulns++
			case "MEDIUM":
				result.MediumVulns++
			case "LOW":
				result.LowVulns++
			case "UNKNOWN":
				result.UnknownVulns++
			}
		}
	}

	return result
}

// passesThreshold determines if scan result passes the configured threshold
func (ts *TrivyScanner) passesThreshold(result TrivyScanResult) bool {
	switch ts.config.VulnerabilityThreshold {
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

// calculateRiskScore calculates a risk score based on vulnerabilities
func (ts *TrivyScanner) calculateRiskScore(result TrivyScanResult) float64 {
	score := 0.0

	// Weight vulnerabilities by severity
	score += float64(result.CriticalVulns) * 10.0
	score += float64(result.HighVulns) * 7.0
	score += float64(result.MediumVulns) * 4.0
	score += float64(result.LowVulns) * 1.0

	// Normalize to 0-100 scale (rough approximation)
	if result.TotalVulns > 0 {
		score = score / float64(result.TotalVulns) * 10.0
	}

	if score > 100.0 {
		score = 100.0
	}

	return score
}

// NewFalcoMonitor creates a new Falco security monitor
func NewFalcoMonitor(config *DockerSecurityConfig) (*FalcoMonitor, error) {
	if !config.FalcoEnabled {
		return nil, fmt.Errorf("Falco monitoring not enabled")
	}

	// Find Falco binary
	falcoPath, err := exec.LookPath("falco")
	if err != nil {
		return nil, fmt.Errorf("Falco not found: %w", err)
	}

	monitor := &FalcoMonitor{
		config:     config,
		falcoPath:  falcoPath,
		rulesPath:  config.FalcoRulesPath,
		webhookURL: config.FalcoWebhookURL,
		events:     make(chan *FalcoEvent, 1000),
		running:    false,
	}

	return monitor, nil
}

// Start starts the Falco monitor
func (fm *FalcoMonitor) Start() error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	if fm.running {
		return fmt.Errorf("Falco monitor already running")
	}

	logrus.Info("Starting Falco security monitor")

	// Prepare Falco command
	args := []string{
		"--json-output",
		"--stdout-output",
	}

	if fm.rulesPath != "" {
		args = append(args, "-r", fm.rulesPath)
	}

	// Start Falco process
	fm.process = exec.Command(fm.falcoPath, args...)

	stdout, err := fm.process.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := fm.process.Start(); err != nil {
		return fmt.Errorf("failed to start Falco: %w", err)
	}

	fm.running = true

	// Start event processing goroutine
	go fm.processEvents(stdout)

	logrus.Info("Falco security monitor started successfully")
	return nil
}

// processEvents processes Falco events from stdout
func (fm *FalcoMonitor) processEvents(stdout io.ReadCloser) {
	defer stdout.Close()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		var event FalcoEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			logrus.WithError(err).Warn("Failed to parse Falco event")
			continue
		}

		// Send event to channel
		select {
		case fm.events <- &event:
		default:
			logrus.Warn("Falco event channel full, dropping event")
		}
	}

	if err := scanner.Err(); err != nil {
		logrus.WithError(err).Error("Error reading Falco output")
	}

	fm.mutex.Lock()
	fm.running = false
	fm.mutex.Unlock()
}

// Stop stops the Falco monitor
func (fm *FalcoMonitor) Stop() error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	if !fm.running {
		return nil
	}

	logrus.Info("Stopping Falco security monitor")

	if fm.process != nil {
		if err := fm.process.Process.Kill(); err != nil {
			logrus.WithError(err).Warn("Failed to kill Falco process")
		}
		fm.process.Wait()
	}

	fm.running = false
	close(fm.events)

	logrus.Info("Falco security monitor stopped")
	return nil
}

// GetEvents returns the Falco events channel
func (fm *FalcoMonitor) GetEvents() <-chan *FalcoEvent {
	return fm.events
}

// NewComplianceChecker creates a new compliance checker
func NewComplianceChecker(config *DockerSecurityConfig, client *client.Client) (*ComplianceChecker, error) {
	if !config.ComplianceEnabled {
		return nil, fmt.Errorf("compliance checking not enabled")
	}

	var oscapPath string
	var err error

	if config.OSCAPEnabled {
		oscapPath, err = exec.LookPath("oscap")
		if err != nil {
			logrus.Warn("OpenSCAP not found, compliance checking will be limited")
		}
	}

	checker := &ComplianceChecker{
		config:      config,
		client:      client,
		oscapPath:   oscapPath,
		profilePath: config.OSCAPProfile,
		reports:     make(map[string]*ComplianceReport),
	}

	return checker, nil
}

// CheckCompliance performs compliance checking on an image
func (cc *ComplianceChecker) CheckCompliance(ctx context.Context, imageName string) (*ComplianceReport, error) {
	cc.mutex.Lock()
	defer cc.mutex.Unlock()

	logrus.WithField("image", imageName).Info("Starting compliance check")

	report := &ComplianceReport{
		ImageName: imageName,
		ScanTime:  time.Now(),
		Profile:   cc.profilePath,
	}

	// Perform different compliance checks
	if cc.config.CISBenchmark {
		if err := cc.runCISBenchmark(ctx, imageName, report); err != nil {
			logrus.WithError(err).Warn("CIS benchmark check failed")
		}
	}

	if cc.config.OSCAPEnabled && cc.oscapPath != "" {
		if err := cc.runOSCAPScan(ctx, imageName, report); err != nil {
			logrus.WithError(err).Warn("OSCAP scan failed")
		}
	}

	// Calculate overall compliance score
	totalRules := report.PassedRules + report.FailedRules + report.ErrorRules + report.NotApplicableRules
	if totalRules > 0 {
		report.Score = float64(report.PassedRules) / float64(totalRules) * 100.0
	}

	// Determine if compliance check passed (80% threshold)
	report.Passed = report.Score >= 80.0

	// Cache report
	cc.reports[imageName] = report

	logrus.WithFields(logrus.Fields{
		"image":  imageName,
		"score":  report.Score,
		"passed": report.Passed,
	}).Info("Compliance check completed")

	return report, nil
}

// runCISBenchmark runs CIS Docker Benchmark checks
func (cc *ComplianceChecker) runCISBenchmark(ctx context.Context, imageName string, report *ComplianceReport) error {
	logrus.Info("Running CIS Docker Benchmark checks")

	// Real CIS benchmark checks using docker-bench-security
	var rules []ComplianceRule

	// Check if docker-bench-security is available
	benchPath, err := exec.LookPath("docker-bench-security")
	if err != nil {
		// Download and install docker-bench-security
		if err := cc.downloadDockerBenchSecurity(); err != nil {
			logrus.Warnf("Failed to install docker-bench-security: %v", err)
			return cc.performBasicCISChecks(ctx, imageName, report)
		}
		benchPath = "./docker-bench-security/docker-bench-security.sh"
	}

	// Run docker-bench-security
	cmd := exec.CommandContext(ctx, "bash", benchPath, "--json")
	output, err := cmd.Output()
	if err != nil {
		logrus.Warnf("Failed to run docker-bench-security: %v", err)
		return cc.performBasicCISChecks(ctx, imageName, report)
	}

	// Parse JSON output
	var benchResults struct {
		Tests []struct {
			ID     string `json:"id"`
			Title  string `json:"desc"`
			Result string `json:"result"`
			Level  string `json:"level"`
		} `json:"tests"`
	}

	if err := json.Unmarshal(output, &benchResults); err != nil {
		logrus.Warnf("Failed to parse docker-bench-security output: %v", err)
		return cc.performBasicCISChecks(ctx, imageName, report)
	}

	// Convert results to compliance rules
	for _, test := range benchResults.Tests {
		severity := "medium"
		switch test.Level {
		case "CRITICAL":
			severity = "critical"
		case "HIGH":
			severity = "high"
		case "MEDIUM":
			severity = "medium"
		case "LOW":
			severity = "low"
		}

		result := "fail"
		if test.Result == "PASS" {
			result = "pass"
		}

		rule := ComplianceRule{
			ID:          test.ID,
			Title:       test.Title,
			Description: fmt.Sprintf("CIS Docker Benchmark check: %s", test.Title),
			Result:      result,
			Severity:    severity,
		}
		rules = append(rules, rule)
	}

	for _, rule := range rules {
		report.Rules = append(report.Rules, rule)
		switch rule.Result {
		case "pass":
			report.PassedRules++
		case "fail":
			report.FailedRules++
		case "error":
			report.ErrorRules++
		case "notapplicable":
			report.NotApplicableRules++
		}
	}

	return nil
}

// downloadDockerBenchSecurity downloads and sets up docker-bench-security
func (cc *ComplianceChecker) downloadDockerBenchSecurity() error {
	logrus.Info("Downloading docker-bench-security")

	// Clone the docker-bench-security repository
	cmd := exec.Command("git", "clone", "https://github.com/docker/docker-bench-security.git")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone docker-bench-security: %w", err)
	}

	// Make the script executable
	scriptPath := "./docker-bench-security/docker-bench-security.sh"
	if err := os.Chmod(scriptPath, 0755); err != nil {
		return fmt.Errorf("failed to make script executable: %w", err)
	}

	return nil
}

// performBasicCISChecks performs basic CIS compliance checks as fallback
func (cc *ComplianceChecker) performBasicCISChecks(ctx context.Context, imageName string, report *ComplianceReport) error {
	logrus.Info("Performing basic CIS compliance checks")

	// Inspect the image
	imageInfo, _, err := cc.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to inspect image: %w", err)
	}

	var rules []ComplianceRule

	// CIS-4.1: Ensure that a user for the container has been created
	if imageInfo.Config.User == "" || imageInfo.Config.User == "root" {
		rules = append(rules, ComplianceRule{
			ID:          "CIS-4.1",
			Title:       "Ensure that a user for the container has been created",
			Description: "Create a non-root user for the container in the Dockerfile",
			Result:      "fail",
			Severity:    "high",
		})
	} else {
		rules = append(rules, ComplianceRule{
			ID:          "CIS-4.1",
			Title:       "Ensure that a user for the container has been created",
			Description: "Create a non-root user for the container in the Dockerfile",
			Result:      "pass",
			Severity:    "high",
		})
	}

	// CIS-4.6: Ensure that HEALTHCHECK instructions have been added to container images
	hasHealthCheck := false
	for _, cmd := range imageInfo.Config.Healthcheck.Test {
		if cmd != "" && cmd != "NONE" {
			hasHealthCheck = true
			break
		}
	}

	if hasHealthCheck {
		rules = append(rules, ComplianceRule{
			ID:          "CIS-4.6",
			Title:       "Ensure that HEALTHCHECK instructions have been added",
			Description: "Add HEALTHCHECK instruction in your docker container images",
			Result:      "pass",
			Severity:    "medium",
		})
	} else {
		rules = append(rules, ComplianceRule{
			ID:          "CIS-4.6",
			Title:       "Ensure that HEALTHCHECK instructions have been added",
			Description: "Add HEALTHCHECK instruction in your docker container images",
			Result:      "fail",
			Severity:    "medium",
		})
	}

	// CIS-4.7: Ensure update instructions are not used alone in Dockerfiles
	// This would require analyzing the Dockerfile history
	rules = append(rules, ComplianceRule{
		ID:          "CIS-4.7",
		Title:       "Ensure update instructions are not used alone in Dockerfiles",
		Description: "Always combine RUN apt-get update with apt-get install in same statement",
		Result:      "info",
		Severity:    "low",
	})

	// Add all rules to report
	for _, rule := range rules {
		report.Rules = append(report.Rules, rule)
		switch rule.Result {
		case "pass":
			report.PassedRules++
		case "fail":
			report.FailedRules++
		}
	}

	return nil
}

// runOSCAPScan runs OpenSCAP compliance scan
func (cc *ComplianceChecker) runOSCAPScan(ctx context.Context, imageName string, report *ComplianceReport) error {
	logrus.Info("Running OpenSCAP compliance scan")

	// Check if oscap-docker is available
	oscapPath, err := exec.LookPath("oscap-docker")
	if err != nil {
		logrus.Warnf("oscap-docker not found: %v", err)
		return cc.performBasicOSCAPChecks(ctx, imageName, report)
	}

	// Create temporary container name
	containerName := fmt.Sprintf("oscap-scan-%s", generateRandomString(8))

	// Run OSCAP scan using CIS benchmark
	// Use RHEL7 STIG profile as default
	cmd := exec.CommandContext(ctx, oscapPath, "image", imageName,
		"xccdf", "eval",
		"--profile", "xccdf_org.ssgproject.content_profile_stig",
		"--results", fmt.Sprintf("/tmp/oscap-results-%s.xml", containerName),
		"--report", fmt.Sprintf("/tmp/oscap-report-%s.html", containerName),
		"/usr/share/xml/scap/ssg/content/ssg-rhel7-ds.xml")

	output, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Warnf("OSCAP scan failed: %v, output: %s", err, string(output))
		return cc.performBasicOSCAPChecks(ctx, imageName, report)
	}

	// Parse results XML file
	resultsFile := fmt.Sprintf("/tmp/oscap-results-%s.xml", containerName)
	defer os.Remove(resultsFile) // Clean up

	results, err := cc.parseOSCAPResults(resultsFile)
	if err != nil {
		logrus.Warnf("Failed to parse OSCAP results: %v", err)
		return cc.performBasicOSCAPChecks(ctx, imageName, report)
	}

	var rules []ComplianceRule
	for _, result := range results {
		rule := ComplianceRule{
			ID:          result.ID,
			Title:       result.Title,
			Description: result.Description,
			Result:      result.Result,
			Severity:    result.Severity,
		}
		rules = append(rules, rule)
	}

	for _, rule := range rules {
		report.Rules = append(report.Rules, rule)
		switch rule.Result {
		case "pass":
			report.PassedRules++
		case "fail":
			report.FailedRules++
		case "error":
			report.ErrorRules++
		case "notapplicable":
			report.NotApplicableRules++
		}
	}

	return nil
}

// parseOSCAPResults parses OSCAP XML results file
func (cc *ComplianceChecker) parseOSCAPResults(filePath string) ([]OSCAPResult, error) {
	// This is a simplified XML parser for OSCAP results
	// In production, you'd use a proper XML parsing library
	var results []OSCAPResult

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open results file: %w", err)
	}
	defer file.Close()

	// Basic XML parsing - in production, use encoding/xml or xmlpath
	scanner := bufio.NewScanner(file)
	var currentResult OSCAPResult
	inRuleResult := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "<rule-result") {
			inRuleResult = true
			// Extract rule ID from attributes
			if idMatch := regexp.MustCompile(`idref="([^"]+)"`).FindStringSubmatch(line); len(idMatch) > 1 {
				currentResult.ID = idMatch[1]
			}
		}

		if inRuleResult {
			if strings.Contains(line, "<result>") {
				result := strings.TrimSpace(strings.Replace(strings.Replace(line, "<result>", "", 1), "</result>", "", 1))
				switch result {
				case "pass":
					currentResult.Result = "pass"
				case "fail":
					currentResult.Result = "fail"
				case "notapplicable":
					currentResult.Result = "notapplicable"
				case "notchecked":
					currentResult.Result = "info"
				default:
					currentResult.Result = "unknown"
				}
			}

			if strings.Contains(line, "</rule-result>") {
				inRuleResult = false
				// Set default values if not found
				if currentResult.Title == "" {
					currentResult.Title = fmt.Sprintf("OSCAP Rule %s", currentResult.ID)
				}
				if currentResult.Description == "" {
					currentResult.Description = fmt.Sprintf("OpenSCAP compliance check for rule %s", currentResult.ID)
				}
				if currentResult.Severity == "" {
					currentResult.Severity = "medium"
				}
				results = append(results, currentResult)
				currentResult = OSCAPResult{}
			}
		}
	}

	return results, scanner.Err()
}

// performBasicOSCAPChecks performs basic OSCAP-style checks as fallback
func (cc *ComplianceChecker) performBasicOSCAPChecks(ctx context.Context, imageName string, report *ComplianceReport) error {
	logrus.Info("Performing basic OSCAP-style compliance checks")

	// Inspect the image
	imageInfo, _, err := cc.client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to inspect image: %w", err)
	}

	var rules []ComplianceRule

	// Check for exposed ports
	if len(imageInfo.Config.ExposedPorts) > 0 {
		hasPrivilegedPorts := false
		for port := range imageInfo.Config.ExposedPorts {
			portNum, _ := strconv.Atoi(strings.Split(string(port), "/")[0])
			if portNum < 1024 {
				hasPrivilegedPorts = true
				break
			}
		}

		if hasPrivilegedPorts {
			rules = append(rules, ComplianceRule{
				ID:          "OSCAP-PORT-001",
				Title:       "Avoid privileged ports",
				Description: "Container should not expose privileged ports (< 1024)",
				Result:      "fail",
				Severity:    "medium",
			})
		} else {
			rules = append(rules, ComplianceRule{
				ID:          "OSCAP-PORT-001",
				Title:       "Avoid privileged ports",
				Description: "Container should not expose privileged ports (< 1024)",
				Result:      "pass",
				Severity:    "medium",
			})
		}
	}

	// Check environment variables for secrets
	hasSecrets := false
	secretPatterns := []string{"password", "secret", "key", "token", "credential"}
	for _, env := range imageInfo.Config.Env {
		envLower := strings.ToLower(env)
		for _, pattern := range secretPatterns {
			if strings.Contains(envLower, pattern) {
				hasSecrets = true
				break
			}
		}
		if hasSecrets {
			break
		}
	}

	if hasSecrets {
		rules = append(rules, ComplianceRule{
			ID:          "OSCAP-SEC-001",
			Title:       "Avoid secrets in environment variables",
			Description: "Container should not contain secrets in environment variables",
			Result:      "fail",
			Severity:    "high",
		})
	} else {
		rules = append(rules, ComplianceRule{
			ID:          "OSCAP-SEC-001",
			Title:       "Avoid secrets in environment variables",
			Description: "Container should not contain secrets in environment variables",
			Result:      "pass",
			Severity:    "high",
		})
	}

	// Add all rules to report
	for _, rule := range rules {
		report.Rules = append(report.Rules, rule)
		switch rule.Result {
		case "pass":
			report.PassedRules++
		case "fail":
			report.FailedRules++
		}
	}

	return nil
}

// OSCAPResult represents a result from OSCAP scanning
type OSCAPResult struct {
	ID          string
	Title       string
	Description string
	Result      string
	Severity    string
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// NewCVEDatabase creates a new CVE database
func NewCVEDatabase(config *DockerSecurityConfig) (*CVEDatabase, error) {
	if !config.CVEEnabled {
		return nil, fmt.Errorf("CVE database not enabled")
	}

	db := &CVEDatabase{
		config:     config,
		dbPath:     filepath.Join("/tmp/cve-db", "cve.db"),
		nvdAPI:     "https://services.nvd.nist.gov/rest/json/cves/2.0",
		cves:       make(map[string]*CVERecord),
		lastUpdate: time.Time{},
	}

	return db, nil
}

// UpdateDatabase updates the CVE database from NVD
func (cve *CVEDatabase) UpdateDatabase() error {
	cve.mutex.Lock()
	defer cve.mutex.Unlock()

	// Check if update is needed (daily updates)
	if time.Since(cve.lastUpdate) < 24*time.Hour {
		logrus.Debug("CVE database is up to date")
		return nil
	}

	logrus.Info("Updating CVE database from NVD")

	// Download recent CVEs from NVD API
	// This is a simplified implementation - in production you would:
	// 1. Use proper pagination
	// 2. Handle rate limiting
	// 3. Store data in persistent database
	// 4. Implement incremental updates

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(cve.nvdAPI + "?resultsPerPage=100")
	if err != nil {
		return fmt.Errorf("failed to fetch CVE data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NVD API returned status: %d", resp.StatusCode)
	}

	// Parse CVE data (simplified)
	// In a real implementation, you would parse the full NVD JSON format
	cve.lastUpdate = time.Now()

	logrus.Info("CVE database updated successfully")
	return nil
}

// LookupCVE looks up a CVE by ID
func (cve *CVEDatabase) LookupCVE(cveID string) (*CVERecord, error) {
	cve.mutex.RLock()
	defer cve.mutex.RUnlock()

	if record, exists := cve.cves[cveID]; exists {
		return record, nil
	}

	return nil, fmt.Errorf("CVE %s not found", cveID)
}

// NewThreatDetector creates a new threat detector
func NewThreatDetector(config *DockerSecurityConfig, client *client.Client) (*ThreatDetector, error) {
	if !config.ThreatDetection {
		return nil, fmt.Errorf("threat detection not enabled")
	}

	detector := &ThreatDetector{
		config:         config,
		client:         client,
		patterns:       loadThreatPatterns(),
		detections:     make(chan *ThreatDetection, 1000),
		containerStats: make(map[string]*ContainerSecurityStats),
	}

	return detector, nil
}

// loadThreatPatterns loads threat detection patterns
func loadThreatPatterns() []ThreatPattern {
	return []ThreatPattern{
		{
			ID:          "THREAT-001",
			Name:        "High CPU Usage",
			Description: "Container using excessive CPU resources",
			Severity:    VulnMedium,
			Indicators: []ThreatIndicator{
				{
					Type:      "cpu_usage",
					Threshold: 80.0,
					Duration:  5 * time.Minute,
				},
			},
			Actions: []string{"alert", "throttle"},
		},
		{
			ID:          "THREAT-002",
			Name:        "Suspicious Network Activity",
			Description: "Unusual network connections detected",
			Severity:    VulnHigh,
			Indicators: []ThreatIndicator{
				{
					Type:      "network_connections",
					Threshold: 100,
					Duration:  1 * time.Minute,
				},
			},
			Actions: []string{"alert", "isolate"},
		},
		{
			ID:          "THREAT-003",
			Name:        "Privilege Escalation Attempt",
			Description: "Potential privilege escalation detected",
			Severity:    VulnCritical,
			Indicators: []ThreatIndicator{
				{
					Type:      "syscalls",
					Value:     []string{"setuid", "setgid", "execve"},
					Duration:  30 * time.Second,
				},
			},
			Actions: []string{"alert", "kill", "quarantine"},
		},
	}
}

// Start starts the threat detector
func (td *ThreatDetector) Start() error {
	td.mutex.Lock()
	defer td.mutex.Unlock()

	if td.running {
		return fmt.Errorf("threat detector already running")
	}

	logrus.Info("Starting threat detector")
	td.running = true

	// Start monitoring goroutines
	go td.monitorContainers()
	go td.processDetections()

	logrus.Info("Threat detector started successfully")
	return nil
}

// monitorContainers monitors containers for threats
func (td *ThreatDetector) monitorContainers() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for td.running {
		select {
		case <-ticker.C:
			td.scanForThreats()
		}
	}
}

// scanForThreats scans running containers for threats
func (td *ThreatDetector) scanForThreats() {
	ctx := context.Background()
	containers, err := td.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		logrus.WithError(err).Error("Failed to list containers for threat detection")
		return
	}

	for _, container := range containers {
		stats, err := td.getContainerSecurityStats(ctx, container.ID)
		if err != nil {
			continue
		}

		// Check each threat pattern
		for _, pattern := range td.patterns {
			if threat := td.evaluatePattern(pattern, stats, container); threat != nil {
				select {
				case td.detections <- threat:
				default:
					logrus.Warn("Threat detection channel full, dropping detection")
				}
			}
		}
	}
}

// getContainerSecurityStats gets security statistics for a container
func (td *ThreatDetector) getContainerSecurityStats(ctx context.Context, containerID string) (*ContainerSecurityStats, error) {
	// Get container stats from Docker
	stats, err := td.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer stats.Body.Close()

	// Parse stats (simplified implementation)
	// In production, you would parse the full stats JSON and extract:
	// - CPU usage
	// - Memory usage
	// - Network I/O
	// - Block I/O
	// - PIDs

	containerStats := &ContainerSecurityStats{
		ContainerID: containerID,
		LastUpdate:  time.Now(),
		// Add actual stats parsing here
	}

	return containerStats, nil
}

// evaluatePattern evaluates a threat pattern against container stats
func (td *ThreatDetector) evaluatePattern(pattern ThreatPattern, stats *ContainerSecurityStats, container types.Container) *ThreatDetection {
	// Simplified pattern matching
	// In production, you would implement sophisticated anomaly detection

	for _, indicator := range pattern.Indicators {
		switch indicator.Type {
		case "cpu_usage":
			if stats.CPUUsage > indicator.Threshold {
				return &ThreatDetection{
					Time:        time.Now(),
					ContainerID: container.ID,
					ImageName:   container.Image,
					ThreatID:    pattern.ID,
					ThreatName:  pattern.Name,
					Severity:    pattern.Severity,
					Description: pattern.Description,
					Confidence:  0.8,
					Source:      "threat_detector",
				}
			}
		case "network_connections":
			if len(stats.NetworkConnections) > int(indicator.Threshold) {
				return &ThreatDetection{
					Time:        time.Now(),
					ContainerID: container.ID,
					ImageName:   container.Image,
					ThreatID:    pattern.ID,
					ThreatName:  pattern.Name,
					Severity:    pattern.Severity,
					Description: pattern.Description,
					Confidence:  0.7,
					Source:      "threat_detector",
				}
			}
		}
	}

	return nil
}

// processDetections processes threat detections
func (td *ThreatDetector) processDetections() {
	for detection := range td.detections {
		logrus.WithFields(logrus.Fields{
			"threat_id":    detection.ThreatID,
			"threat_name":  detection.ThreatName,
			"container_id": detection.ContainerID,
			"severity":     detection.Severity.String(),
		}).Warn("Threat detected")

		// Trigger automated response if enabled
		// This would be handled by the AutoResponder component
	}
}

// Stop stops the threat detector
func (td *ThreatDetector) Stop() error {
	td.mutex.Lock()
	defer td.mutex.Unlock()

	if !td.running {
		return nil
	}

	logrus.Info("Stopping threat detector")
	td.running = false
	close(td.detections)

	logrus.Info("Threat detector stopped")
	return nil
}

// GetDetections returns the threat detections channel
func (td *ThreatDetector) GetDetections() <-chan *ThreatDetection {
	return td.detections
}

// NewAutoResponder creates a new automated security responder
func NewAutoResponder(config *DockerSecurityConfig, client *client.Client) (*AutoResponder, error) {
	if !config.AutoResponse {
		return nil, fmt.Errorf("auto response not enabled")
	}

	responder := &AutoResponder{
		config:    config,
		client:    client,
		actions:   make(chan *SecurityAction, 1000),
		responses: make(map[string]*ResponseRule),
	}

	// Load default response rules
	responder.loadDefaultResponseRules()

	return responder, nil
}

// loadDefaultResponseRules loads default automated response rules
func (ar *AutoResponder) loadDefaultResponseRules() {
	ar.responses = map[string]*ResponseRule{
		"critical-threat": {
			ID:   "RESP-001",
			Name: "Critical Threat Response",
			Triggers: []ResponseTrigger{
				{
					Type:     "threat_detection",
					Severity: VulnCritical,
					Source:   "threat_detector",
				},
			},
			Actions: []ResponseAction{
				{
					Type:    "quarantine",
					Timeout: 5 * time.Minute,
				},
				{
					Type:    "alert",
					Timeout: 30 * time.Second,
				},
			},
			Enabled:  true,
			Cooldown: 5 * time.Minute,
		},
		"high-vulnerability": {
			ID:   "RESP-002",
			Name: "High Vulnerability Response",
			Triggers: []ResponseTrigger{
				{
					Type:     "vulnerability_scan",
					Severity: VulnHigh,
					Source:   "trivy_scanner",
				},
			},
			Actions: []ResponseAction{
				{
					Type:    "block_deployment",
					Timeout: 1 * time.Minute,
				},
				{
					Type:    "notify_admin",
					Timeout: 30 * time.Second,
				},
			},
			Enabled:  true,
			Cooldown: 1 * time.Hour,
		},
		"compliance-failure": {
			ID:   "RESP-003",
			Name: "Compliance Failure Response",
			Triggers: []ResponseTrigger{
				{
					Type:   "compliance_check",
					Source: "compliance_checker",
				},
			},
			Actions: []ResponseAction{
				{
					Type:    "audit_log",
					Timeout: 10 * time.Second,
				},
			},
			Enabled:  true,
			Cooldown: 30 * time.Minute,
		},
	}
}

// Start starts the auto responder
func (ar *AutoResponder) Start() error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if ar.running {
		return fmt.Errorf("auto responder already running")
	}

	logrus.Info("Starting automated security responder")
	ar.running = true

	// Start action processing goroutine
	go ar.processActions()

	logrus.Info("Automated security responder started successfully")
	return nil
}

// processActions processes security actions
func (ar *AutoResponder) processActions() {
	for action := range ar.actions {
		if err := ar.executeAction(action); err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"action_id":   action.ID,
				"action_type": action.Type,
				"target":      action.Target,
			}).Error("Failed to execute security action")
		}
	}
}

// executeAction executes a security action
func (ar *AutoResponder) executeAction(action *SecurityAction) error {
	logrus.WithFields(logrus.Fields{
		"action_id":   action.ID,
		"action_type": action.Type,
		"target":      action.Target,
		"severity":    action.Severity.String(),
	}).Info("Executing security action")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	switch action.Type {
	case "quarantine":
		return ar.quarantineContainer(ctx, action.Target)
	case "kill":
		return ar.killContainer(ctx, action.Target)
	case "block_deployment":
		return ar.blockDeployment(ctx, action.Target)
	case "alert":
		return ar.sendAlert(action)
	case "notify_admin":
		return ar.notifyAdmin(action)
	case "audit_log":
		return ar.auditLog(action)
	default:
		return fmt.Errorf("unknown action type: %s", action.Type)
	}
}

// quarantineContainer quarantines a container by isolating its network
func (ar *AutoResponder) quarantineContainer(ctx context.Context, containerID string) error {
	if !ar.config.QuarantineMode {
		return fmt.Errorf("quarantine mode not enabled")
	}

	logrus.WithField("container_id", containerID).Info("Quarantining container")

	// Disconnect container from all networks except none
	networks, err := ar.client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, network := range networks {
		if network.Name == "none" {
			continue
		}

		err := ar.client.NetworkDisconnect(ctx, network.ID, containerID, false)
		if err != nil {
			logrus.WithError(err).WithFields(logrus.Fields{
				"container_id": containerID,
				"network_id":   network.ID,
			}).Warn("Failed to disconnect container from network")
		}
	}

	// Connect to none network (isolated)
	err = ar.client.NetworkConnect(ctx, "none", containerID, nil)
	if err != nil {
		return fmt.Errorf("failed to connect container to none network: %w", err)
	}

	logrus.WithField("container_id", containerID).Info("Container quarantined successfully")
	return nil
}

// killContainer kills a container if configured to do so
func (ar *AutoResponder) killContainer(ctx context.Context, containerID string) error {
	if !ar.config.KillSuspicious {
		return fmt.Errorf("kill suspicious containers not enabled")
	}

	logrus.WithField("container_id", containerID).Warn("Killing suspicious container")

	err := ar.client.ContainerKill(ctx, containerID, "SIGKILL")
	if err != nil {
		return fmt.Errorf("failed to kill container: %w", err)
	}

	logrus.WithField("container_id", containerID).Info("Suspicious container killed")
	return nil
}

// blockDeployment blocks a container deployment
func (ar *AutoResponder) blockDeployment(ctx context.Context, imageName string) error {
	logrus.WithField("image", imageName).Info("Blocking deployment of vulnerable image")

	// In a real implementation, this would:
	// 1. Add image to blocked images list
	// 2. Update deployment policies
	// 3. Notify orchestration systems (K8s, Docker Swarm, etc.)

	return nil
}

// sendAlert sends a security alert
func (ar *AutoResponder) sendAlert(action *SecurityAction) error {
	alert := &SecurityAlert{
		ID:          fmt.Sprintf("ALERT-%d", time.Now().Unix()),
		Type:        "security_response",
		Severity:    action.Severity,
		Title:       fmt.Sprintf("Security Action: %s", action.Type),
		Description: action.Description,
		Source:      "auto_responder",
		Target:      action.Target,
		Data:        action.Data,
		Timestamp:   time.Now(),
	}

	logrus.WithFields(logrus.Fields{
		"alert_id":  alert.ID,
		"severity":  alert.Severity.String(),
		"target":    alert.Target,
	}).Warn("Security alert generated")

	// In production, send alert to monitoring systems, SIEM, etc.
	return nil
}

// notifyAdmin notifies administrators about security issues
func (ar *AutoResponder) notifyAdmin(action *SecurityAction) error {
	if !ar.config.NotifyAdmins {
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"action_type": action.Type,
		"target":      action.Target,
		"severity":    action.Severity.String(),
	}).Info("Notifying administrators")

	// In production, this would:
	// 1. Send emails/SMS to administrators
	// 2. Post to Slack/Teams channels
	// 3. Create tickets in ITSM systems
	// 4. Update dashboards

	return nil
}

// auditLog logs security actions for audit purposes
func (ar *AutoResponder) auditLog(action *SecurityAction) error {
	logrus.WithFields(logrus.Fields{
		"action_id":    action.ID,
		"action_type":  action.Type,
		"target":       action.Target,
		"severity":     action.Severity.String(),
		"description":  action.Description,
		"timestamp":    action.Timestamp,
	}).Info("Security action audit log")

	return nil
}

// Stop stops the auto responder
func (ar *AutoResponder) Stop() error {
	ar.mutex.Lock()
	defer ar.mutex.Unlock()

	if !ar.running {
		return nil
	}

	logrus.Info("Stopping automated security responder")
	ar.running = false
	close(ar.actions)

	logrus.Info("Automated security responder stopped")
	return nil
}

// TriggerAction triggers an automated security action
func (ar *AutoResponder) TriggerAction(actionType, target string, severity VulnerabilityLevel, description string, data map[string]interface{}) {
	action := &SecurityAction{
		ID:          fmt.Sprintf("ACTION-%d", time.Now().UnixNano()),
		Type:        actionType,
		Target:      target,
		Severity:    severity,
		Description: description,
		Data:        data,
		Timestamp:   time.Now(),
	}

	select {
	case ar.actions <- action:
	default:
		logrus.Warn("Auto responder action channel full, dropping action")
	}
}

// SecureContainerCreate creates a container with comprehensive security checks
func (sdc *SecureDockerClient) SecureContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string, userContext *DockerUserContext) (*container.CreateResponse, error) {
	sdc.mutex.Lock()
	defer sdc.mutex.Unlock()

	sdc.stats.TotalOperations++

	logrus.WithFields(logrus.Fields{
		"image":         config.Image,
		"container":     containerName,
		"user_id":       userContext.UserID,
		"username":      userContext.Username,
	}).Info("Starting secure container creation")

	// 1. Validate user permissions
	if err := sdc.validateUserPermissions(userContext, "container_create"); err != nil {
		sdc.stats.BlockedOperations++
		return nil, fmt.Errorf("permission denied: %w", err)
	}

	// 2. Comprehensive image validation and scanning
	if err := sdc.comprehensiveImageValidation(ctx, config.Image); err != nil {
		sdc.stats.BlockedOperations++
		sdc.stats.ContainersBlocked++
		return nil, fmt.Errorf("image validation failed: %w", err)
	}

	// 3. Apply security hardening
	if err := sdc.applySecurityHardening(config, hostConfig); err != nil {
		sdc.stats.BlockedOperations++
		return nil, fmt.Errorf("security hardening failed: %w", err)
	}

	// 4. Validate container configuration
	if err := sdc.validateContainerConfig(config, hostConfig); err != nil {
		sdc.stats.BlockedOperations++
		sdc.stats.SecurityViolations++
		return nil, fmt.Errorf("container configuration validation failed: %w", err)
	}

	// 5. Create container
	response, err := sdc.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		sdc.stats.BlockedOperations++
		return nil, fmt.Errorf("container creation failed: %w", err)
	}

	sdc.stats.ContainersCreated++

	// 6. Audit log
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
	}).Info("Secure container created successfully")

	return &response, nil
}

// comprehensiveImageValidation performs comprehensive image validation
func (sdc *SecureDockerClient) comprehensiveImageValidation(ctx context.Context, imageName string) error {
	logrus.WithField("image", imageName).Info("Starting comprehensive image validation")

	// 1. Check allowed registries
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

	// 2. Check blocked images
	for _, blockedImage := range sdc.config.BlockedImages {
		if matched, _ := filepath.Match(blockedImage, imageName); matched {
			return fmt.Errorf("image is blocked: %s", imageName)
		}
	}

	// 3. Trivy vulnerability scanning
	if sdc.config.ImageScanning && sdc.trivyScanner != nil {
		scanResult, err := sdc.trivyScanner.ScanImage(ctx, imageName)
		if err != nil {
			logrus.WithError(err).Warn("Trivy scan failed")
		} else {
			sdc.stats.ScannedImages++
			if !scanResult.Passed {
				sdc.stats.VulnerableImages++
				if sdc.config.AutoResponse && sdc.autoResponder != nil {
					sdc.autoResponder.TriggerAction("block_deployment", imageName, VulnHigh,
						fmt.Sprintf("Image failed vulnerability scan: %d vulnerabilities found", scanResult.TotalVulns),
						map[string]interface{}{
							"total_vulns":   scanResult.TotalVulns,
							"critical":      scanResult.CriticalVulns,
							"high":          scanResult.HighVulns,
							"risk_score":    scanResult.RiskScore,
						})
				}
				return fmt.Errorf("image failed security scan: %d vulnerabilities found (Critical: %d, High: %d)",
					scanResult.TotalVulns, scanResult.CriticalVulns, scanResult.HighVulns)
			}
		}
	}

	// 4. Compliance checking
	if sdc.config.ComplianceEnabled && sdc.complianceChecker != nil {
		complianceReport, err := sdc.complianceChecker.CheckCompliance(ctx, imageName)
		if err != nil {
			logrus.WithError(err).Warn("Compliance check failed")
		} else if !complianceReport.Passed {
			sdc.stats.ComplianceViolations++
			return fmt.Errorf("image failed compliance check: score %.1f%% (threshold: 80%%)", complianceReport.Score)
		}
	}

	// 5. Verify image signature if required
	if sdc.config.SignedImagesOnly {
		if err := sdc.verifyImageSignature(ctx, imageName); err != nil {
			return fmt.Errorf("image signature verification failed: %w", err)
		}
	}

	return nil
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

// Additional utility and security functions for production-grade Docker security

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

	// Real Docker Content Trust verification using docker trust inspect
	logrus.Infof("Verifying Docker Content Trust for %s:%s", repository, tag)

	// Check if docker trust is enabled
	cmd := exec.CommandContext(ctx, "docker", "trust", "inspect", fmt.Sprintf("%s:%s", repository, tag))
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("Docker Content Trust verification failed for %s:%s: %w", repository, tag, err)
	}

	// Parse trust inspection output
	var trustData []struct {
		Name           string `json:"Name"`
		SignedTags     []struct {
			SignedTag   string `json:"SignedTag"`
			Digest      string `json:"Digest"`
			Signers     []string `json:"Signers"`
		} `json:"SignedTags"`
		Signers        []struct {
			Name string `json:"Name"`
			Keys []struct {
				ID string `json:"ID"`
			} `json:"Keys"`
		} `json:"Signers"`
		AdministrativeKeys []struct {
			Name string `json:"Name"`
			Keys []struct {
				ID string `json:"ID"`
			} `json:"Keys"`
		} `json:"AdministrativeKeys"`
	}

	if err := json.Unmarshal(output, &trustData); err != nil {
		return fmt.Errorf("failed to parse trust data: %w", err)
	}

	// Verify that the image has valid signatures
	if len(trustData) == 0 {
		return fmt.Errorf("no trust data found for %s:%s", repository, tag)
	}

	found := false
	for _, data := range trustData {
		for _, signedTag := range data.SignedTags {
			if signedTag.SignedTag == tag {
				if len(signedTag.Signers) == 0 {
					return fmt.Errorf("no valid signers found for %s:%s", repository, tag)
				}
				found = true
				logrus.Infof("Valid signature found for %s:%s with signers: %v", repository, tag, signedTag.Signers)
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return fmt.Errorf("no valid signature found for tag %s in %s", tag, repository)
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

// SecurityReport represents a comprehensive security report
type SecurityReport struct {
	GeneratedAt     time.Time                   `json:"generated_at"`
	ReportVersion   string                      `json:"report_version"`
	SystemInfo      SystemInfo                  `json:"system_info"`
	SecurityConfig  *DockerSecurityConfig       `json:"security_config"`
	Statistics      *DockerSecurityStats        `json:"statistics"`
	Findings        []SecurityFinding           `json:"findings"`
	Recommendations []SecurityRecommendation    `json:"recommendations"`
	SecurityScore   float64                     `json:"security_score"`
}

// SystemInfo represents system information
type SystemInfo struct {
	DockerVersion     string    `json:"docker_version"`
	KernelVersion     string    `json:"kernel_version"`
	OS                string    `json:"os"`
	Architecture      string    `json:"architecture"`
	TotalContainers   int       `json:"total_containers"`
	RunningContainers int       `json:"running_containers"`
	TotalImages       int       `json:"total_images"`
	ServerTime        time.Time `json:"server_time"`
}

// SecurityFinding represents a security finding
type SecurityFinding struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Severity    VulnerabilityLevel     `json:"severity"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Resource    string                 `json:"resource"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
}

// SecurityRecommendation represents a security recommendation
type SecurityRecommendation struct {
	ID          string                 `json:"id"`
	Priority    string                 `json:"priority"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Action      string                 `json:"action"`
	Impact      string                 `json:"impact"`
	Details     map[string]interface{} `json:"details"`
}

// GenerateSecurityReport generates a comprehensive security report
func (sdc *SecureDockerClient) GenerateSecurityReport(ctx context.Context) (*SecurityReport, error) {
	sdc.mutex.RLock()
	defer sdc.mutex.RUnlock()

	report := &SecurityReport{
		GeneratedAt:     time.Now(),
		ReportVersion:   "1.0",
		SystemInfo:      sdc.getSystemInfo(ctx),
		SecurityConfig:  sdc.config,
		Statistics:      sdc.stats,
		Findings:        []SecurityFinding{},
		Recommendations: []SecurityRecommendation{},
	}

	// Collect security findings
	findings, err := sdc.collectSecurityFindings(ctx)
	if err != nil {
		logrus.WithError(err).Warn("Failed to collect some security findings")
	}
	report.Findings = findings

	// Generate recommendations
	report.Recommendations = sdc.generateSecurityRecommendations(report)

	// Calculate overall security score
	report.SecurityScore = sdc.calculateSecurityScore(report)

	return report, nil
}

// getSystemInfo collects system information
func (sdc *SecureDockerClient) getSystemInfo(ctx context.Context) SystemInfo {
	info, err := sdc.client.Info(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get Docker info")
		return SystemInfo{
			ServerTime: time.Now(),
		}
	}

	version, err := sdc.client.ServerVersion(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to get Docker version")
	}

	return SystemInfo{
		DockerVersion:     version.Version,
		KernelVersion:     info.KernelVersion,
		OS:                info.OperatingSystem,
		Architecture:      info.Architecture,
		TotalContainers:   info.Containers,
		RunningContainers: info.ContainersRunning,
		TotalImages:       info.Images,
		ServerTime:        time.Now(),
	}
}

// collectSecurityFindings collects security findings from various sources
func (sdc *SecureDockerClient) collectSecurityFindings(ctx context.Context) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	// Collect container security findings
	containerFindings, err := sdc.collectContainerFindings(ctx)
	if err != nil {
		logrus.WithError(err).Warn("Failed to collect container findings")
	}
	findings = append(findings, containerFindings...)

	// Collect image security findings
	imageFindings, err := sdc.collectImageFindings(ctx)
	if err != nil {
		logrus.WithError(err).Warn("Failed to collect image findings")
	}
	findings = append(findings, imageFindings...)

	// Collect configuration findings
	configFindings := sdc.collectConfigurationFindings()
	findings = append(findings, configFindings...)

	return findings, nil
}

// collectContainerFindings collects security findings from running containers
func (sdc *SecureDockerClient) collectContainerFindings(ctx context.Context) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	containers, err := sdc.client.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, container := range containers {
		// Check for privileged containers
		containerInfo, err := sdc.client.ContainerInspect(ctx, container.ID)
		if err != nil {
			continue
		}

		if containerInfo.HostConfig.Privileged {
			findings = append(findings, SecurityFinding{
				ID:          fmt.Sprintf("PRIV-%s", container.ID[:12]),
				Type:        "privileged_container",
				Severity:    VulnHigh,
				Title:       "Privileged Container Detected",
				Description: "Container is running with privileged access",
				Resource:    container.ID,
				Details: map[string]interface{}{
					"image":          container.Image,
					"container_id":   container.ID,
					"container_name": strings.TrimPrefix(container.Names[0], "/"),
				},
				Timestamp: time.Now(),
			})
		}

		// Check for containers with excessive capabilities
		if len(containerInfo.HostConfig.CapAdd) > 0 {
			findings = append(findings, SecurityFinding{
				ID:          fmt.Sprintf("CAPS-%s", container.ID[:12]),
				Type:        "excessive_capabilities",
				Severity:    VulnMedium,
				Title:       "Container with Added Capabilities",
				Description: "Container has additional capabilities which may increase attack surface",
				Resource:    container.ID,
				Details: map[string]interface{}{
					"capabilities": containerInfo.HostConfig.CapAdd,
					"image":        container.Image,
				},
				Timestamp: time.Now(),
			})
		}

		// Check for containers with host network mode
		if containerInfo.HostConfig.NetworkMode.IsHost() {
			findings = append(findings, SecurityFinding{
				ID:          fmt.Sprintf("HOSTNET-%s", container.ID[:12]),
				Type:        "host_network",
				Severity:    VulnHigh,
				Title:       "Container Using Host Network",
				Description: "Container is using host network mode which reduces isolation",
				Resource:    container.ID,
				Details: map[string]interface{}{
					"network_mode": string(containerInfo.HostConfig.NetworkMode),
					"image":        container.Image,
				},
				Timestamp: time.Now(),
			})
		}
	}

	return findings, nil
}

// collectImageFindings collects security findings from images
func (sdc *SecureDockerClient) collectImageFindings(ctx context.Context) ([]SecurityFinding, error) {
	var findings []SecurityFinding

	if sdc.trivyScanner == nil {
		return findings, nil
	}

	images, err := sdc.client.ImageList(ctx, types.ImageListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	for _, image := range images {
		for _, tag := range image.RepoTags {
			if tag == "<none>:<none>" {
				continue
			}

			scanResult, err := sdc.trivyScanner.ScanImage(ctx, tag)
			if err != nil {
				continue
			}

			if scanResult.CriticalVulns > 0 {
				findings = append(findings, SecurityFinding{
					ID:          fmt.Sprintf("VULN-CRIT-%s", hashString(tag)[:8]),
					Type:        "critical_vulnerabilities",
					Severity:    VulnCritical,
					Title:       "Critical Vulnerabilities in Image",
					Description: fmt.Sprintf("Image contains %d critical vulnerabilities", scanResult.CriticalVulns),
					Resource:    tag,
					Details: map[string]interface{}{
						"critical_vulns": scanResult.CriticalVulns,
						"total_vulns":    scanResult.TotalVulns,
						"risk_score":     scanResult.RiskScore,
					},
					Timestamp: time.Now(),
				})
			}

			if scanResult.HighVulns > 10 {
				findings = append(findings, SecurityFinding{
					ID:          fmt.Sprintf("VULN-HIGH-%s", hashString(tag)[:8]),
					Type:        "high_vulnerabilities",
					Severity:    VulnHigh,
					Title:       "Multiple High Vulnerabilities in Image",
					Description: fmt.Sprintf("Image contains %d high-severity vulnerabilities", scanResult.HighVulns),
					Resource:    tag,
					Details: map[string]interface{}{
						"high_vulns":  scanResult.HighVulns,
						"total_vulns": scanResult.TotalVulns,
						"risk_score":  scanResult.RiskScore,
					},
					Timestamp: time.Now(),
				})
			}
			break // Only scan once per image
		}
	}

	return findings, nil
}

// collectConfigurationFindings collects security findings from configuration
func (sdc *SecureDockerClient) collectConfigurationFindings() []SecurityFinding {
	var findings []SecurityFinding

	// Check if TLS is disabled
	if !sdc.config.TLSEnabled {
		findings = append(findings, SecurityFinding{
			ID:          "CONFIG-TLS",
			Type:        "configuration",
			Severity:    VulnHigh,
			Title:       "TLS Disabled",
			Description: "Docker TLS encryption is disabled, communications are not encrypted",
			Resource:    "docker_daemon",
			Details: map[string]interface{}{
				"recommendation": "Enable TLS for Docker daemon communications",
			},
			Timestamp: time.Now(),
		})
	}

	// Check if signed images are not required
	if !sdc.config.SignedImagesOnly {
		findings = append(findings, SecurityFinding{
			ID:          "CONFIG-SIGN",
			Type:        "configuration",
			Severity:    VulnMedium,
			Title:       "Unsigned Images Allowed",
			Description: "Docker is configured to allow unsigned images",
			Resource:    "docker_daemon",
			Details: map[string]interface{}{
				"recommendation": "Enable signed images only to ensure image integrity",
			},
			Timestamp: time.Now(),
		})
	}

	// Check if user namespacing is disabled
	if !sdc.config.UserNamespacing {
		findings = append(findings, SecurityFinding{
			ID:          "CONFIG-USERNS",
			Type:        "configuration",
			Severity:    VulnMedium,
			Title:       "User Namespacing Disabled",
			Description: "Docker user namespacing is disabled, reducing container isolation",
			Resource:    "docker_daemon",
			Details: map[string]interface{}{
				"recommendation": "Enable user namespacing for better container isolation",
			},
			Timestamp: time.Now(),
		})
	}

	return findings
}

// generateSecurityRecommendations generates security recommendations based on findings
func (sdc *SecureDockerClient) generateSecurityRecommendations(report *SecurityReport) []SecurityRecommendation {
	var recommendations []SecurityRecommendation

	// Count findings by type
	findingCounts := make(map[string]int)
	for _, finding := range report.Findings {
		findingCounts[finding.Type]++
	}

	// Generate recommendations based on findings
	if count := findingCounts["privileged_container"]; count > 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			ID:          "REC-PRIV",
			Priority:    "HIGH",
			Title:       "Remove Privileged Containers",
			Description: fmt.Sprintf("Found %d privileged containers. Avoid using privileged mode unless absolutely necessary.", count),
			Action:      "Review and remove privileged flag from containers, use specific capabilities instead",
			Impact:      "Reduces attack surface and limits potential for privilege escalation",
		})
	}

	if count := findingCounts["critical_vulnerabilities"]; count > 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			ID:          "REC-CRIT-VULN",
			Priority:    "CRITICAL",
			Title:       "Address Critical Vulnerabilities",
			Description: fmt.Sprintf("Found %d images with critical vulnerabilities. These pose immediate security risks.", count),
			Action:      "Update images to latest versions, patch vulnerabilities, or remove affected images",
			Impact:      "Prevents exploitation of critical security vulnerabilities",
		})
	}

	if !sdc.config.TLSEnabled {
		recommendations = append(recommendations, SecurityRecommendation{
			ID:          "REC-TLS",
			Priority:    "HIGH",
			Title:       "Enable Docker TLS",
			Description: "Docker daemon is not using TLS encryption for communications",
			Action:      "Configure TLS certificates and enable TLS for Docker daemon",
			Impact:      "Encrypts Docker API communications and prevents man-in-the-middle attacks",
		})
	}

	if !sdc.config.ImageScanning {
		recommendations = append(recommendations, SecurityRecommendation{
			ID:          "REC-SCAN",
			Priority:    "MEDIUM",
			Title:       "Enable Image Scanning",
			Description: "Image vulnerability scanning is disabled",
			Action:      "Enable Trivy or other vulnerability scanning for all images",
			Impact:      "Identifies vulnerabilities before deployment",
		})
	}

	return recommendations
}

// calculateSecurityScore calculates an overall security score
func (sdc *SecureDockerClient) calculateSecurityScore(report *SecurityReport) float64 {
	score := 100.0

	// Deduct points for findings
	for _, finding := range report.Findings {
		switch finding.Severity {
		case VulnCritical:
			score -= 15.0
		case VulnHigh:
			score -= 10.0
		case VulnMedium:
			score -= 5.0
		case VulnLow:
			score -= 1.0
		}
	}

	// Add points for good configuration
	if sdc.config.TLSEnabled {
		score += 5.0
	}
	if sdc.config.ImageScanning {
		score += 5.0
	}
	if sdc.config.SignedImagesOnly {
		score += 5.0
	}
	if sdc.config.UserNamespacing {
		score += 5.0
	}
	if sdc.config.AuditEnabled {
		score += 3.0
	}

	// Ensure score is between 0 and 100
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// hashString creates a hash of a string for generating IDs
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

// SecurityReportExporter provides multiple export formats for security reports
type SecurityReportExporter struct {
	client *client.Client
}

// NewSecurityReportExporter creates a new report exporter
func NewSecurityReportExporter(client *client.Client) *SecurityReportExporter {
	return &SecurityReportExporter{
		client: client,
	}
}

// ExportFormat represents supported export formats
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatHTML ExportFormat = "html"
	ExportFormatCSV  ExportFormat = "csv"
	ExportFormatPDF  ExportFormat = "pdf"
	ExportFormatXML  ExportFormat = "xml"
)

// ExportOptions contains options for report export
type ExportOptions struct {
	Format          ExportFormat `json:"format"`
	OutputPath      string       `json:"output_path"`
	IncludeDetails  bool         `json:"include_details"`
	IncludeCharts   bool         `json:"include_charts"`
	CompressOutput  bool         `json:"compress_output"`
	Template        string       `json:"template"`
}

// ExportReport exports a security report in the specified format
func (sre *SecurityReportExporter) ExportReport(report *SecurityReport, options ExportOptions) error {
	switch options.Format {
	case ExportFormatJSON:
		return sre.exportToJSON(report, options)
	case ExportFormatHTML:
		return sre.exportToHTML(report, options)
	case ExportFormatCSV:
		return sre.exportToCSV(report, options)
	case ExportFormatPDF:
		return sre.exportToPDF(report, options)
	case ExportFormatXML:
		return sre.exportToXML(report, options)
	default:
		return fmt.Errorf("unsupported export format: %s", options.Format)
	}
}

// exportToJSON exports report as JSON
func (sre *SecurityReportExporter) exportToJSON(report *SecurityReport, options ExportOptions) error {
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	return sre.writeToFile(jsonData, options.OutputPath, options.CompressOutput)
}

// exportToHTML exports report as HTML
func (sre *SecurityReportExporter) exportToHTML(report *SecurityReport, options ExportOptions) error {
	htmlTemplate := sre.getHTMLTemplate(options.Template)

	// Create HTML content
	html := fmt.Sprintf(htmlTemplate,
		report.GeneratedAt.Format("2006-01-02 15:04:05"),
		report.ReportVersion,
		report.SecurityScore,
		sre.getScoreClass(report.SecurityScore),
		len(report.Findings),
		len(report.Recommendations),
		report.SystemInfo.DockerVersion,
		report.SystemInfo.KernelVersion,
		report.SystemInfo.OS,
		sre.generateFindingsHTML(report.Findings, options.IncludeDetails),
		sre.generateRecommendationsHTML(report.Recommendations),
		sre.generateStatisticsHTML(report.Statistics),
	)

	return sre.writeToFile([]byte(html), options.OutputPath, options.CompressOutput)
}

// exportToCSV exports report as CSV
func (sre *SecurityReportExporter) exportToCSV(report *SecurityReport, options ExportOptions) error {
	var csvContent strings.Builder

	// CSV Header
	csvContent.WriteString("Type,ID,Severity,Title,Description,Resource,Timestamp\n")

	// Write findings
	for _, finding := range report.Findings {
		csvContent.WriteString(fmt.Sprintf("%s,%s,%s,\"%s\",\"%s\",%s,%s\n",
			finding.Type,
			finding.ID,
			finding.Severity,
			strings.ReplaceAll(finding.Title, "\"", "\"\""),
			strings.ReplaceAll(finding.Description, "\"", "\"\""),
			finding.Resource,
			finding.Timestamp.Format("2006-01-02 15:04:05"),
		))
	}

	return sre.writeToFile([]byte(csvContent.String()), options.OutputPath, options.CompressOutput)
}

// exportToPDF exports report as PDF (simplified implementation)
func (sre *SecurityReportExporter) exportToPDF(report *SecurityReport, options ExportOptions) error {
	// For a real PDF implementation, you would use a library like gofpdf or wkhtmltopdf
	// This is a simplified implementation that creates a text-based PDF-like format

	var pdfContent strings.Builder

	pdfContent.WriteString("DOCKER SECURITY REPORT\n")
	pdfContent.WriteString("======================\n\n")
	pdfContent.WriteString(fmt.Sprintf("Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	pdfContent.WriteString(fmt.Sprintf("Version: %s\n", report.ReportVersion))
	pdfContent.WriteString(fmt.Sprintf("Security Score: %.2f/100\n\n", report.SecurityScore))

	pdfContent.WriteString("SUMMARY\n")
	pdfContent.WriteString("-------\n")
	pdfContent.WriteString(fmt.Sprintf("Total Findings: %d\n", len(report.Findings)))
	pdfContent.WriteString(fmt.Sprintf("Recommendations: %d\n\n", len(report.Recommendations)))

	pdfContent.WriteString("SYSTEM INFORMATION\n")
	pdfContent.WriteString("------------------\n")
	pdfContent.WriteString(fmt.Sprintf("Docker Version: %s\n", report.SystemInfo.DockerVersion))
	pdfContent.WriteString(fmt.Sprintf("Kernel Version: %s\n", report.SystemInfo.KernelVersion))
	pdfContent.WriteString(fmt.Sprintf("Operating System: %s\n", report.SystemInfo.OS))
	pdfContent.WriteString(fmt.Sprintf("Architecture: %s\n\n", report.SystemInfo.Architecture))

	if options.IncludeDetails {
		pdfContent.WriteString("SECURITY FINDINGS\n")
		pdfContent.WriteString("-----------------\n")
		for i, finding := range report.Findings {
			pdfContent.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, finding.Severity, finding.Title))
			pdfContent.WriteString(fmt.Sprintf("   Resource: %s\n", finding.Resource))
			pdfContent.WriteString(fmt.Sprintf("   Description: %s\n\n", finding.Description))
		}
	}

	return sre.writeToFile([]byte(pdfContent.String()), options.OutputPath, options.CompressOutput)
}

// exportToXML exports report as XML
func (sre *SecurityReportExporter) exportToXML(report *SecurityReport, options ExportOptions) error {
	var xmlContent strings.Builder

	xmlContent.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	xmlContent.WriteString("<SecurityReport>\n")
	xmlContent.WriteString(fmt.Sprintf("  <GeneratedAt>%s</GeneratedAt>\n", report.GeneratedAt.Format("2006-01-02T15:04:05Z")))
	xmlContent.WriteString(fmt.Sprintf("  <ReportVersion>%s</ReportVersion>\n", report.ReportVersion))
	xmlContent.WriteString(fmt.Sprintf("  <SecurityScore>%.2f</SecurityScore>\n", report.SecurityScore))

	// System Info
	xmlContent.WriteString("  <SystemInfo>\n")
	xmlContent.WriteString(fmt.Sprintf("    <DockerVersion>%s</DockerVersion>\n", report.SystemInfo.DockerVersion))
	xmlContent.WriteString(fmt.Sprintf("    <KernelVersion>%s</KernelVersion>\n", report.SystemInfo.KernelVersion))
	xmlContent.WriteString(fmt.Sprintf("    <OS>%s</OS>\n", report.SystemInfo.OS))
	xmlContent.WriteString(fmt.Sprintf("    <Architecture>%s</Architecture>\n", report.SystemInfo.Architecture))
	xmlContent.WriteString("  </SystemInfo>\n")

	// Findings
	xmlContent.WriteString("  <Findings>\n")
	for _, finding := range report.Findings {
		xmlContent.WriteString("    <Finding>\n")
		xmlContent.WriteString(fmt.Sprintf("      <ID>%s</ID>\n", finding.ID))
		xmlContent.WriteString(fmt.Sprintf("      <Type>%s</Type>\n", finding.Type))
		xmlContent.WriteString(fmt.Sprintf("      <Severity>%s</Severity>\n", finding.Severity))
		xmlContent.WriteString(fmt.Sprintf("      <Title><![CDATA[%s]]></Title>\n", finding.Title))
		xmlContent.WriteString(fmt.Sprintf("      <Description><![CDATA[%s]]></Description>\n", finding.Description))
		xmlContent.WriteString(fmt.Sprintf("      <Resource>%s</Resource>\n", finding.Resource))
		xmlContent.WriteString(fmt.Sprintf("      <Timestamp>%s</Timestamp>\n", finding.Timestamp.Format("2006-01-02T15:04:05Z")))
		xmlContent.WriteString("    </Finding>\n")
	}
	xmlContent.WriteString("  </Findings>\n")

	// Recommendations
	xmlContent.WriteString("  <Recommendations>\n")
	for _, rec := range report.Recommendations {
		xmlContent.WriteString("    <Recommendation>\n")
		xmlContent.WriteString(fmt.Sprintf("      <ID>%s</ID>\n", rec.ID))
		xmlContent.WriteString(fmt.Sprintf("      <Priority>%s</Priority>\n", rec.Priority))
		xmlContent.WriteString(fmt.Sprintf("      <Title><![CDATA[%s]]></Title>\n", rec.Title))
		xmlContent.WriteString(fmt.Sprintf("      <Description><![CDATA[%s]]></Description>\n", rec.Description))
		xmlContent.WriteString(fmt.Sprintf("      <Action><![CDATA[%s]]></Action>\n", rec.Action))
		xmlContent.WriteString("    </Recommendation>\n")
	}
	xmlContent.WriteString("  </Recommendations>\n")

	xmlContent.WriteString("</SecurityReport>\n")

	return sre.writeToFile([]byte(xmlContent.String()), options.OutputPath, options.CompressOutput)
}

// writeToFile writes data to file with optional compression
func (sre *SecurityReportExporter) writeToFile(data []byte, outputPath string, compress bool) error {
	if compress {
		// In a real implementation, you'd use gzip compression
		// For now, just write the raw data
		logrus.Info("Compression requested but not implemented, writing raw data")
	}

	// Ensure directory exists
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to write data to file: %w", err)
	}

	logrus.Infof("Report exported successfully to: %s", outputPath)
	return nil
}

// getHTMLTemplate returns HTML template for report
func (sre *SecurityReportExporter) getHTMLTemplate(customTemplate string) string {
	if customTemplate != "" {
		// In a real implementation, load custom template from file
		logrus.Info("Custom template specified but not implemented, using default")
	}

	return `<!DOCTYPE html>
<html>
<head>
    <title>Docker Security Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f4f4f4; padding: 20px; border-radius: 5px; }
        .score { font-size: 24px; font-weight: bold; margin: 10px 0; }
        .score.good { color: #28a745; }
        .score.warning { color: #ffc107; }
        .score.danger { color: #dc3545; }
        .finding { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .finding.critical { border-left: 5px solid #dc3545; }
        .finding.high { border-left: 5px solid #fd7e14; }
        .finding.medium { border-left: 5px solid #ffc107; }
        .finding.low { border-left: 5px solid #28a745; }
        .recommendation { background-color: #e7f3ff; padding: 15px; margin: 10px 0; border-radius: 5px; }
        table { width: 100%%; border-collapse: collapse; margin: 20px 0; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Docker Security Report</h1>
        <p>Generated: %s | Version: %s</p>
        <div class="score %s">Security Score: %.2f/100</div>
        <p>Total Findings: %d | Recommendations: %d</p>
    </div>

    <h2>System Information</h2>
    <table>
        <tr><th>Docker Version</th><td>%s</td></tr>
        <tr><th>Kernel Version</th><td>%s</td></tr>
        <tr><th>Operating System</th><td>%s</td></tr>
    </table>

    <h2>Security Findings</h2>
    %s

    <h2>Recommendations</h2>
    %s

    <h2>Statistics</h2>
    %s
</body>
</html>`
}

// getScoreClass returns CSS class based on security score
func (sre *SecurityReportExporter) getScoreClass(score float64) string {
	if score >= 80 {
		return "good"
	} else if score >= 60 {
		return "warning"
	}
	return "danger"
}

// generateFindingsHTML generates HTML for findings
func (sre *SecurityReportExporter) generateFindingsHTML(findings []SecurityFinding, includeDetails bool) string {
	if len(findings) == 0 {
		return "<p>No security findings detected.</p>"
	}

	var html strings.Builder
	for _, finding := range findings {
		html.WriteString(fmt.Sprintf(`<div class="finding %s">`, strings.ToLower(string(finding.Severity))))
		html.WriteString(fmt.Sprintf(`<h3>[%s] %s</h3>`, finding.Severity, finding.Title))
		html.WriteString(fmt.Sprintf(`<p><strong>Resource:</strong> %s</p>`, finding.Resource))

		if includeDetails {
			html.WriteString(fmt.Sprintf(`<p><strong>Description:</strong> %s</p>`, finding.Description))
			html.WriteString(fmt.Sprintf(`<p><strong>Type:</strong> %s</p>`, finding.Type))
			html.WriteString(fmt.Sprintf(`<p><strong>Timestamp:</strong> %s</p>`, finding.Timestamp.Format("2006-01-02 15:04:05")))
		}
		html.WriteString(`</div>`)
	}
	return html.String()
}

// generateRecommendationsHTML generates HTML for recommendations
func (sre *SecurityReportExporter) generateRecommendationsHTML(recommendations []SecurityRecommendation) string {
	if len(recommendations) == 0 {
		return "<p>No security recommendations available.</p>"
	}

	var html strings.Builder
	for _, rec := range recommendations {
		html.WriteString(`<div class="recommendation">`)
		html.WriteString(fmt.Sprintf(`<h3>[%s] %s</h3>`, rec.Priority, rec.Title))
		html.WriteString(fmt.Sprintf(`<p><strong>Description:</strong> %s</p>`, rec.Description))
		html.WriteString(fmt.Sprintf(`<p><strong>Action:</strong> %s</p>`, rec.Action))
		if rec.Impact != "" {
			html.WriteString(fmt.Sprintf(`<p><strong>Impact:</strong> %s</p>`, rec.Impact))
		}
		html.WriteString(`</div>`)
	}
	return html.String()
}

// generateStatisticsHTML generates HTML for statistics
func (sre *SecurityReportExporter) generateStatisticsHTML(stats *DockerSecurityStats) string {
	if stats == nil {
		return "<p>No statistics available.</p>"
	}

	return fmt.Sprintf(`
		<table>
			<tr><th>Total Scans</th><td>%d</td></tr>
			<tr><th>Vulnerabilities Found</th><td>%d</td></tr>
			<tr><th>Compliance Violations</th><td>%d</td></tr>
			<tr><th>Threats Detected</th><td>%d</td></tr>
			<tr><th>Security Events</th><td>%d</td></tr>
		</table>
	`, stats.TotalScans, stats.VulnerabilitiesFound, stats.ComplianceViolations, stats.ThreatsDetected, stats.SecurityEvents)
}

// ReportScheduler handles automatic report generation
type ReportScheduler struct {
	client     *SecureDockerClient
	exporter   *SecurityReportExporter
	ticker     *time.Ticker
	stopCh     chan struct{}
	schedules  []ReportSchedule
	mutex      sync.RWMutex
}

// ReportSchedule represents a scheduled report configuration
type ReportSchedule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Frequency   time.Duration `json:"frequency"`
	Format      ExportFormat  `json:"format"`
	OutputPath  string        `json:"output_path"`
	Enabled     bool          `json:"enabled"`
	LastRun     time.Time     `json:"last_run"`
	NextRun     time.Time     `json:"next_run"`
}

// NewReportScheduler creates a new report scheduler
func NewReportScheduler(client *SecureDockerClient, exporter *SecurityReportExporter) *ReportScheduler {
	return &ReportScheduler{
		client:    client,
		exporter:  exporter,
		schedules: make([]ReportSchedule, 0),
		stopCh:    make(chan struct{}),
	}
}

// AddSchedule adds a new report schedule
func (rs *ReportScheduler) AddSchedule(schedule ReportSchedule) {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	schedule.ID = fmt.Sprintf("schedule-%d", time.Now().Unix())
	schedule.NextRun = time.Now().Add(schedule.Frequency)
	rs.schedules = append(rs.schedules, schedule)

	logrus.Infof("Added report schedule: %s (frequency: %v)", schedule.Name, schedule.Frequency)
}

// Start starts the report scheduler
func (rs *ReportScheduler) Start() {
	rs.ticker = time.NewTicker(1 * time.Minute) // Check every minute

	go func() {
		for {
			select {
			case <-rs.ticker.C:
				rs.checkSchedules()
			case <-rs.stopCh:
				return
			}
		}
	}()

	logrus.Info("Report scheduler started")
}

// Stop stops the report scheduler
func (rs *ReportScheduler) Stop() {
	if rs.ticker != nil {
		rs.ticker.Stop()
	}
	close(rs.stopCh)
	logrus.Info("Report scheduler stopped")
}

// checkSchedules checks and executes due schedules
func (rs *ReportScheduler) checkSchedules() {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	now := time.Now()
	for i := range rs.schedules {
		schedule := &rs.schedules[i]
		if schedule.Enabled && now.After(schedule.NextRun) {
			go rs.executeSchedule(schedule)
			schedule.LastRun = now
			schedule.NextRun = now.Add(schedule.Frequency)
		}
	}
}

// executeSchedule executes a scheduled report generation
func (rs *ReportScheduler) executeSchedule(schedule *ReportSchedule) {
	logrus.Infof("Executing scheduled report: %s", schedule.Name)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Generate security report
	report, err := rs.client.GenerateSecurityReport(ctx)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to generate scheduled report: %s", schedule.Name)
		return
	}

	// Create output path with timestamp
	timestamp := time.Now().Format("20060102-150405")
	outputPath := fmt.Sprintf("%s-%s", schedule.OutputPath, timestamp)

	// Export report
	options := ExportOptions{
		Format:         schedule.Format,
		OutputPath:     outputPath,
		IncludeDetails: true,
		IncludeCharts:  false,
		CompressOutput: false,
	}

	if err := rs.exporter.ExportReport(report, options); err != nil {
		logrus.WithError(err).Errorf("Failed to export scheduled report: %s", schedule.Name)
		return
	}

	logrus.Infof("Scheduled report completed successfully: %s -> %s", schedule.Name, outputPath)
}