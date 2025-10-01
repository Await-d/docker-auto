package service

import (
	"context"
	"time"
)

// SecurityScanner defines the interface for container security scanning
type SecurityScanner interface {
	// Core scanning operations
	ScanContainer(ctx context.Context, containerID string) (*ScanResult, error)
	ScanImage(ctx context.Context, imageID string) (*ImageScanResult, error)

	// Vulnerability scanning
	ScanVulnerabilities(ctx context.Context, containerID string) (*VulnerabilityScanResult, error)
	GetVulnerabilityDetails(ctx context.Context, vulnerabilityID string) (*VulnerabilityDetail, error)

	// Security configuration scanning
	ScanConfiguration(ctx context.Context, containerID string) (*ConfigurationScanResult, error)
	ScanSecrets(ctx context.Context, containerID string) (*SecretsScanResult, error)

	// Compliance scanning
	ScanCompliance(ctx context.Context, containerID string, standards []string) (*ComplianceScanResult, error)

	// Batch operations
	ScanMultipleContainers(ctx context.Context, containerIDs []string) (map[string]*ScanResult, error)

	// Scanner management
	GetScannerInfo() *ScannerInfo
	UpdateScannerConfig(config *ScannerConfig) error
	GetScanHistory(ctx context.Context, containerID string, limit int) ([]*ScanResult, error)
}

// ScanResult represents the complete scan result for a container
type ScanResult struct {
	ScanID              string                     `json:"scan_id"`
	ContainerID         string                     `json:"container_id"`
	ImageID             string                     `json:"image_id"`
	ImageName           string                     `json:"image_name"`
	ScanStartTime       time.Time                  `json:"scan_start_time"`
	ScanEndTime         time.Time                  `json:"scan_end_time"`
	ScanDuration        time.Duration              `json:"scan_duration"`
	Scanner             string                     `json:"scanner"`
	ScannerVersion      string                     `json:"scanner_version"`

	// Vulnerability results
	Vulnerabilities     []*Vulnerability           `json:"vulnerabilities"`
	VulnerabilityStats  *VulnerabilityStats        `json:"vulnerability_stats"`

	// Configuration results
	ConfigurationIssues []*ConfigurationIssue      `json:"configuration_issues"`
	SecretIssues        []*SecretIssue             `json:"secret_issues"`

	// Overall assessment
	RiskScore           float64                    `json:"risk_score"`
	SecurityGrade       string                     `json:"security_grade"`

	// Metadata
	Metadata            map[string]interface{}     `json:"metadata"`
	Success             bool                       `json:"success"`
	ErrorMessage        string                     `json:"error_message,omitempty"`
}

// ImageScanResult represents scan results for a container image
type ImageScanResult struct {
	ImageID             string                     `json:"image_id"`
	ImageName           string                     `json:"image_name"`
	ImageTag            string                     `json:"image_tag"`
	Registry            string                     `json:"registry"`
	Digest              string                     `json:"digest"`
	Size                int64                      `json:"size"`
	Architecture        string                     `json:"architecture"`
	OS                  string                     `json:"os"`

	// Layer information
	Layers              []*LayerInfo               `json:"layers"`

	// Vulnerability results
	Vulnerabilities     []*Vulnerability           `json:"vulnerabilities"`
	VulnerabilityStats  *VulnerabilityStats        `json:"vulnerability_stats"`

	// Package information
	Packages            []*PackageInfo             `json:"packages"`

	// Scan metadata
	ScanTime            time.Time                  `json:"scan_time"`
	Scanner             string                     `json:"scanner"`
	Success             bool                       `json:"success"`
	ErrorMessage        string                     `json:"error_message,omitempty"`
}

// VulnerabilityScanResult represents vulnerability-specific scan results
type VulnerabilityScanResult struct {
	ContainerID         string                     `json:"container_id"`
	Vulnerabilities     []*Vulnerability           `json:"vulnerabilities"`
	Stats               *VulnerabilityStats        `json:"stats"`
	ScanTime            time.Time                  `json:"scan_time"`
	DatabaseVersion     string                     `json:"database_version"`
	DatabaseUpdatedAt   time.Time                  `json:"database_updated_at"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID                  string                     `json:"id"`
	CVE                 string                     `json:"cve"`
	Title               string                     `json:"title"`
	Description         string                     `json:"description"`
	Severity            string                     `json:"severity"`
	Score               float64                    `json:"score"`
	ScoringSystem       string                     `json:"scoring_system"`

	// Affected components
	Package             string                     `json:"package"`
	InstalledVersion    string                     `json:"installed_version"`
	FixedVersion        string                     `json:"fixed_version"`

	// References and links
	References          []string                   `json:"references"`
	Links               []string                   `json:"links"`

	// Classification
	Category            string                     `json:"category"`
	CWE                 []string                   `json:"cwe"`

	// Impact assessment
	Exploitability      string                     `json:"exploitability"`
	Impact              string                     `json:"impact"`

	// Detection details
	Layer               string                     `json:"layer"`
	FilePath            string                     `json:"file_path"`

	// Remediation
	Solution            string                     `json:"solution"`
	Workaround          string                     `json:"workaround"`

	// Metadata
	PublishedDate       time.Time                  `json:"published_date"`
	ModifiedDate        time.Time                  `json:"modified_date"`
	DiscoveredDate      time.Time                  `json:"discovered_date"`
}

// VulnerabilityStats represents vulnerability statistics
type VulnerabilityStats struct {
	Total               int                        `json:"total"`
	Critical            int                        `json:"critical"`
	High                int                        `json:"high"`
	Medium              int                        `json:"medium"`
	Low                 int                        `json:"low"`
	Unknown             int                        `json:"unknown"`
	Fixable             int                        `json:"fixable"`
	Unfixable           int                        `json:"unfixable"`
}

// ConfigurationScanResult represents configuration security scan results
type ConfigurationScanResult struct {
	ContainerID         string                     `json:"container_id"`
	Issues              []*ConfigurationIssue      `json:"issues"`
	Recommendations     []*SecurityRecommendation  `json:"recommendations"`
	ComplianceScore     float64                    `json:"compliance_score"`
	ScanTime            time.Time                  `json:"scan_time"`
}

// ConfigurationIssue represents a configuration security issue
type ConfigurationIssue struct {
	ID                  string                     `json:"id"`
	Title               string                     `json:"title"`
	Description         string                     `json:"description"`
	Severity            string                     `json:"severity"`
	Category            string                     `json:"category"`
	Rule                string                     `json:"rule"`

	// Configuration details
	ConfigKey           string                     `json:"config_key"`
	CurrentValue        interface{}                `json:"current_value"`
	RecommendedValue    interface{}                `json:"recommended_value"`

	// Remediation
	Solution            string                     `json:"solution"`
	Impact              string                     `json:"impact"`

	// References
	References          []string                   `json:"references"`
}

// SecretsScanResult represents secrets detection scan results
type SecretsScanResult struct {
	ContainerID         string                     `json:"container_id"`
	Secrets             []*SecretIssue             `json:"secrets"`
	ScanTime            time.Time                  `json:"scan_time"`
}

// SecretIssue represents a detected secret or sensitive information
type SecretIssue struct {
	ID                  string                     `json:"id"`
	Type                string                     `json:"type"`
	Description         string                     `json:"description"`
	Severity            string                     `json:"severity"`

	// Location details
	FilePath            string                     `json:"file_path"`
	LineNumber          int                        `json:"line_number"`
	Context             string                     `json:"context"`

	// Secret details
	Pattern             string                     `json:"pattern"`
	Entropy             float64                    `json:"entropy"`

	// Remediation
	Recommendation      string                     `json:"recommendation"`
}

// ComplianceScanResult represents compliance scan results
type ComplianceScanResult struct {
	ContainerID         string                     `json:"container_id"`
	Standard            string                     `json:"standard"`
	Checks              []*ComplianceCheck         `json:"checks"`
	OverallScore        float64                    `json:"overall_score"`
	Grade               string                     `json:"grade"`
	ScanTime            time.Time                  `json:"scan_time"`
}

// VulnerabilityDetail represents detailed information about a specific vulnerability
type VulnerabilityDetail struct {
	*Vulnerability

	// Extended information
	CVSS                *CVSSInfo                  `json:"cvss"`
	ExploitInfo         *ExploitInfo               `json:"exploit_info"`
	PatchInfo           *PatchInfo                 `json:"patch_info"`
	RelatedCVEs         []string                   `json:"related_cves"`

	// Industry data
	EPSSScore           float64                    `json:"epss_score"`
	KEVCatalog          bool                       `json:"kev_catalog"`

	// Organizational context
	AssetImpact         string                     `json:"asset_impact"`
	BusinessCriticality string                     `json:"business_criticality"`
}

// CVSSInfo represents CVSS scoring information
type CVSSInfo struct {
	Version             string                     `json:"version"`
	Vector              string                     `json:"vector"`
	BaseScore           float64                    `json:"base_score"`
	TemporalScore       float64                    `json:"temporal_score"`
	EnvironmentalScore  float64                    `json:"environmental_score"`

	// Base metrics
	AttackVector        string                     `json:"attack_vector"`
	AttackComplexity    string                     `json:"attack_complexity"`
	PrivilegesRequired  string                     `json:"privileges_required"`
	UserInteraction     string                     `json:"user_interaction"`
	Scope               string                     `json:"scope"`
	ConfidentialityImpact string                   `json:"confidentiality_impact"`
	IntegrityImpact     string                     `json:"integrity_impact"`
	AvailabilityImpact  string                     `json:"availability_impact"`
}

// ExploitInfo represents information about known exploits
type ExploitInfo struct {
	ExploitAvailable    bool                       `json:"exploit_available"`
	ExploitType         string                     `json:"exploit_type"`
	ExploitMaturity     string                     `json:"exploit_maturity"`
	ExploitSources      []string                   `json:"exploit_sources"`
	LastExploitDate     time.Time                  `json:"last_exploit_date"`
}

// PatchInfo represents patch availability information
type PatchInfo struct {
	PatchAvailable      bool                       `json:"patch_available"`
	PatchDate           time.Time                  `json:"patch_date"`
	PatchVersion        string                     `json:"patch_version"`
	PatchNotes          string                     `json:"patch_notes"`
	PatchURL            string                     `json:"patch_url"`
}

// LayerInfo represents information about an image layer
type LayerInfo struct {
	Digest              string                     `json:"digest"`
	Size                int64                      `json:"size"`
	Command             string                     `json:"command"`
	CreatedBy           string                     `json:"created_by"`
	CreatedAt           time.Time                  `json:"created_at"`
}

// PackageInfo represents information about installed packages
type PackageInfo struct {
	Name                string                     `json:"name"`
	Version             string                     `json:"version"`
	Type                string                     `json:"type"`
	Architecture        string                     `json:"architecture"`
	Description         string                     `json:"description"`
	License             string                     `json:"license"`
	Maintainer          string                     `json:"maintainer"`
	Layer               string                     `json:"layer"`
	FilePath            string                     `json:"file_path"`
}

// ScannerInfo represents information about the security scanner
type ScannerInfo struct {
	Name                string                     `json:"name"`
	Version             string                     `json:"version"`
	DatabaseVersion     string                     `json:"database_version"`
	DatabaseUpdatedAt   time.Time                  `json:"database_updated_at"`
	SupportedFormats    []string                   `json:"supported_formats"`
	Capabilities        []string                   `json:"capabilities"`
}

// ScannerConfig represents configuration for the security scanner
type ScannerConfig struct {
	// Scanner settings
	Timeout             time.Duration              `json:"timeout"`
	MaxConcurrentScans  int                        `json:"max_concurrent_scans"`
	EnabledScanners     []string                   `json:"enabled_scanners"`

	// Vulnerability database settings
	DatabaseURL         string                     `json:"database_url"`
	UpdateInterval      time.Duration              `json:"update_interval"`

	// Scan options
	ScanSecrets         bool                       `json:"scan_secrets"`
	ScanConfiguration   bool                       `json:"scan_configuration"`
	ScanLicenses        bool                       `json:"scan_licenses"`

	// Filtering options
	IgnoreSeverities    []string                   `json:"ignore_severities"`
	IgnoreCVEs          []string                   `json:"ignore_cves"`
	IgnorePackages      []string                   `json:"ignore_packages"`

	// Output options
	OutputFormat        string                     `json:"output_format"`
	IncludePackages     bool                       `json:"include_packages"`
	IncludeLayers       bool                       `json:"include_layers"`

	// Integration settings
	RegistryCredentials map[string]RegistryAuth    `json:"registry_credentials"`
}

// RegistryAuth represents authentication for container registries
type RegistryAuth struct {
	Username            string                     `json:"username"`
	Password            string                     `json:"password"`
	Token               string                     `json:"token"`
	IdentityToken       string                     `json:"identity_token"`
	RegistryToken       string                     `json:"registry_token"`
}