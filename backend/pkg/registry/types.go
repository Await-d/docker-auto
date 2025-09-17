package registry

import (
	"fmt"
	"strings"
	"time"
)

// AuthConfig represents registry authentication configuration
type AuthConfig struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	Email    string `json:"email,omitempty"`
	AuthType string `json:"auth_type"` // basic, token, oauth
}

// ImageTag represents an image tag with metadata
type ImageTag struct {
	Name        string            `json:"name"`
	Digest      string            `json:"digest"`
	Created     time.Time         `json:"created"`
	Size        int64             `json:"size"`
	Architecture string           `json:"architecture,omitempty"`
	OS          string            `json:"os,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ImageInfo represents complete image information including latest tag and metadata
type ImageInfo struct {
	Repository   string                 `json:"repository"`
	LatestTag    string                 `json:"latest_tag"`
	LatestDigest string                 `json:"latest_digest"`
	Tags         []ImageTag             `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	LastChecked  time.Time              `json:"last_checked"`
}

// ImageManifest represents image manifest information
type ImageManifest struct {
	Digest       string            `json:"digest"`
	MediaType    string            `json:"media_type"`
	Size         int64             `json:"size"`
	Architecture string            `json:"architecture,omitempty"`
	OS           string            `json:"os,omitempty"`
	Created      time.Time         `json:"created"`
	Labels       map[string]string `json:"labels,omitempty"`
	Config       *ConfigInfo       `json:"config,omitempty"`
	Layers       []LayerInfo       `json:"layers,omitempty"`
}

// ConfigInfo represents image configuration information
type ConfigInfo struct {
	Digest    string            `json:"digest"`
	Size      int64             `json:"size"`
	MediaType string            `json:"media_type"`
	Env       []string          `json:"env,omitempty"`
	Cmd       []string          `json:"cmd,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// LayerInfo represents image layer information
type LayerInfo struct {
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	MediaType string `json:"media_type"`
}

// RepositoryInfo represents repository information
type RepositoryInfo struct {
	Name         string            `json:"name"`
	Description  string            `json:"description,omitempty"`
	Stars        int               `json:"stars,omitempty"`
	Pulls        int64             `json:"pulls,omitempty"`
	IsOfficial   bool              `json:"is_official,omitempty"`
	IsAutomated  bool              `json:"is_automated,omitempty"`
	LastUpdated  time.Time         `json:"last_updated,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	Architecture []string          `json:"architecture,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// RepositorySearchResult represents search result for repositories
type RepositorySearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Stars       int    `json:"stars,omitempty"`
	IsOfficial  bool   `json:"is_official,omitempty"`
	IsAutomated bool   `json:"is_automated,omitempty"`
}

// UpdateCheckResult represents the result of checking for image updates
type UpdateCheckResult struct {
	Repository      string                    `json:"repository"`
	CurrentTag      string                    `json:"current_tag"`
	CurrentDigest   string                    `json:"current_digest"`
	LatestTag       string                    `json:"latest_tag,omitempty"`
	LatestDigest    string                    `json:"latest_digest,omitempty"`
	UpdateAvailable bool                      `json:"update_available"`
	UpdateType      string                    `json:"update_type,omitempty"` // major, minor, patch, unknown
	ComparedTags    []TagComparison           `json:"compared_tags,omitempty"`
	SecurityIssues  []SecurityVulnerability   `json:"security_issues,omitempty"`
	LastChecked     time.Time                 `json:"last_checked"`
}

// TagComparison represents comparison between two tags
type TagComparison struct {
	Tag         string    `json:"tag"`
	Digest      string    `json:"digest"`
	Created     time.Time `json:"created"`
	Size        int64     `json:"size"`
	IsNewer     bool      `json:"is_newer"`
	VersionDiff string    `json:"version_diff,omitempty"` // major, minor, patch, unknown
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID          string    `json:"id"`
	Package     string    `json:"package,omitempty"`
	Version     string    `json:"version,omitempty"`
	Severity    string    `json:"severity"` // critical, high, medium, low
	CVSS        float64   `json:"cvss,omitempty"`
	Description string    `json:"description,omitempty"`
	FixedIn     string    `json:"fixed_in,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	Links       []string  `json:"links,omitempty"`
}

// RegistryInfo represents information about a registry
type RegistryInfo struct {
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	Type        string            `json:"type"` // dockerhub, harbor, gcr, ecr, etc.
	Version     string            `json:"version,omitempty"`
	Available   bool              `json:"available"`
	Features    []string          `json:"features,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	LastChecked time.Time         `json:"last_checked"`
}

// Harbor specific types

// Project represents a Harbor project
type Project struct {
	ID           int               `json:"project_id"`
	Name         string            `json:"name"`
	Public       bool              `json:"public"`
	RepoCount    int               `json:"repo_count"`
	ChartCount   int               `json:"chart_count,omitempty"`
	CreationTime time.Time         `json:"creation_time"`
	UpdateTime   time.Time         `json:"update_time"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Repository represents a Harbor repository
type Repository struct {
	ID           int       `json:"id"`
	Name         string    `json:"name"`
	ProjectID    int       `json:"project_id"`
	Description  string    `json:"description,omitempty"`
	PullCount    int64     `json:"pull_count"`
	StarCount    int       `json:"star_count"`
	TagsCount    int       `json:"tags_count"`
	CreationTime time.Time `json:"creation_time"`
	UpdateTime   time.Time `json:"update_time"`
}

// Artifact represents a Harbor artifact
type Artifact struct {
	ID              int                  `json:"id"`
	Type            string               `json:"type"`
	MediaType       string               `json:"media_type"`
	ManifestMediaType string             `json:"manifest_media_type"`
	ProjectID       int                  `json:"project_id"`
	RepositoryID    int                  `json:"repository_id"`
	Digest          string               `json:"digest"`
	Size            int64                `json:"size"`
	PushTime        time.Time            `json:"push_time"`
	PullTime        time.Time            `json:"pull_time"`
	ExtraAttrs      map[string]interface{} `json:"extra_attrs,omitempty"`
	Annotations     map[string]string    `json:"annotations,omitempty"`
	References      []Reference          `json:"references,omitempty"`
	Tags            []ArtifactTag        `json:"tags,omitempty"`
	ScanOverview    map[string]*ScanResult `json:"scan_overview,omitempty"`
}

// ArtifactTag represents an artifact tag
type ArtifactTag struct {
	ID           int       `json:"id"`
	RepositoryID int       `json:"repository_id"`
	ArtifactID   int       `json:"artifact_id"`
	Name         string    `json:"name"`
	PushTime     time.Time `json:"push_time"`
	PullTime     time.Time `json:"pull_time"`
}

// Reference represents an artifact reference
type Reference struct {
	ParentID int    `json:"parent_id"`
	ChildID  int    `json:"child_id"`
	Platform string `json:"platform,omitempty"`
}

// ScanResult represents vulnerability scan result
type ScanResult struct {
	ReportID        string                       `json:"report_id,omitempty"`
	ScanStatus      string                       `json:"scan_status,omitempty"`
	Severity        string                       `json:"severity,omitempty"`
	Duration        int64                        `json:"duration,omitempty"`
	Summary         *VulnerabilitySummary        `json:"summary,omitempty"`
	Vulnerabilities []SecurityVulnerability      `json:"vulnerabilities,omitempty"`
	Scanner         *Scanner                     `json:"scanner,omitempty"`
	StartTime       time.Time                    `json:"start_time,omitempty"`
	EndTime         time.Time                    `json:"end_time,omitempty"`
}

// VulnerabilitySummary represents summary of vulnerabilities
type VulnerabilitySummary struct {
	Total    int            `json:"total"`
	Critical int            `json:"critical"`
	High     int            `json:"high"`
	Medium   int            `json:"medium"`
	Low      int            `json:"low"`
	Unknown  int            `json:"unknown"`
	Summary  map[string]int `json:"summary,omitempty"`
}

// Scanner represents vulnerability scanner information
type Scanner struct {
	Name    string `json:"name"`
	Vendor  string `json:"vendor"`
	Version string `json:"version"`
}

// Registry client configuration

// ClientConfig represents configuration for registry clients
type ClientConfig struct {
	BaseURL     string            `json:"base_url"`
	Auth        *AuthConfig       `json:"auth,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
	RetryCount  int               `json:"retry_count,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	TLSConfig   *TLSConfig        `json:"tls_config,omitempty"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	InsecureSkipVerify bool   `json:"insecure_skip_verify,omitempty"`
	CAFile             string `json:"ca_file,omitempty"`
	CertFile           string `json:"cert_file,omitempty"`
	KeyFile            string `json:"key_file,omitempty"`
}

// Search and filter types

// SearchOptions represents search options for images/repositories
type SearchOptions struct {
	Query       string            `json:"query"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
	Sort        string            `json:"sort,omitempty"`        // name, stars, pulls, updated
	Order       string            `json:"order,omitempty"`       // asc, desc
	Filters     map[string]string `json:"filters,omitempty"`
	IsOfficial  *bool             `json:"is_official,omitempty"`
	IsAutomated *bool             `json:"is_automated,omitempty"`
}

// TagListOptions represents options for listing tags
type TagListOptions struct {
	Repository string `json:"repository"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	Sort       string `json:"sort,omitempty"`  // name, created, size
	Order      string `json:"order,omitempty"` // asc, desc
}

// Image search result
type ImageSearchResult struct {
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Stars        int       `json:"stars,omitempty"`
	Pulls        int64     `json:"pulls,omitempty"`
	IsOfficial   bool      `json:"is_official,omitempty"`
	IsAutomated  bool      `json:"is_automated,omitempty"`
	LastUpdated  time.Time `json:"last_updated,omitempty"`
	RegistryType string    `json:"registry_type,omitempty"`
}

// Error types

// RegistryError represents a registry-specific error
type RegistryError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Detail     string `json:"detail,omitempty"`
	StatusCode int    `json:"status_code,omitempty"`
}

func (e *RegistryError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrorCodeUnauthorized     = "UNAUTHORIZED"
	ErrorCodeImageNotFound    = "IMAGE_NOT_FOUND"
	ErrorCodeTagNotFound      = "TAG_NOT_FOUND"
	ErrorCodeRegistryTimeout  = "REGISTRY_TIMEOUT"
	ErrorCodeRateLimit        = "RATE_LIMIT"
	ErrorCodeInvalidResponse  = "INVALID_RESPONSE"
	ErrorCodeConnectionFailed = "CONNECTION_FAILED"
)

// Helper functions

// NewRegistryError creates a new registry error
func NewRegistryError(code, message string) *RegistryError {
	return &RegistryError{
		Code:    code,
		Message: message,
	}
}

// IsVersionNewer compares two version strings and returns true if v1 is newer than v2
func IsVersionNewer(v1, v2 string) bool {
	// Simple version comparison - in a real implementation you might want
	// to use a proper semantic versioning library
	return v1 > v2
}

// ParseImageRef parses an image reference into registry, namespace, repository, and tag
func ParseImageRef(imageRef string) (registry, namespace, repository, tag string) {
	// Default values
	registry = "docker.io"
	namespace = "library"
	tag = "latest"

	// Split image reference
	// Format: [registry/][namespace/]repository[:tag]
	parts := strings.Split(imageRef, "/")

	switch len(parts) {
	case 1:
		// just repository name: nginx
		repository = parts[0]
	case 2:
		if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") {
			// registry/repository: gcr.io/nginx
			registry = parts[0]
			repository = parts[1]
			namespace = ""
		} else {
			// namespace/repository: library/nginx
			namespace = parts[0]
			repository = parts[1]
		}
	case 3:
		// registry/namespace/repository: gcr.io/project/nginx
		registry = parts[0]
		namespace = parts[1]
		repository = parts[2]
	}

	// Extract tag from repository if present
	if strings.Contains(repository, ":") {
		repoParts := strings.SplitN(repository, ":", 2)
		repository = repoParts[0]
		tag = repoParts[1]
	}

	return registry, namespace, repository, tag
}

// BuildImageRef builds an image reference from components
func BuildImageRef(registry, namespace, repository, tag string) string {
	var parts []string

	if registry != "" && registry != "docker.io" {
		parts = append(parts, registry)
	}

	if namespace != "" && namespace != "library" {
		parts = append(parts, namespace)
	}

	parts = append(parts, repository)

	imageRef := strings.Join(parts, "/")

	if tag != "" && tag != "latest" {
		imageRef += ":" + tag
	}

	return imageRef
}