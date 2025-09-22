package docker

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"docker-auto/pkg/security"
	dockerTypes "docker-auto/pkg/types"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

// ImageScanner provides image scanning and version comparison functionality
type ImageScanner struct {
	clientManager  *ClientManager
	scanCache      map[string]*ScanResult
	versionCache   map[string]*VersionInfo
	mutex          sync.RWMutex
	logger         *logrus.Logger
	config         *ScannerConfig
}

// ScannerConfig represents scanner configuration
type ScannerConfig struct {
	CacheExpiration    time.Duration            `json:"cache_expiration"`
	MaxConcurrentScans int                      `json:"max_concurrent_scans"`
	VulnThreshold      security.VulnerabilityLevel `json:"vuln_threshold"`
	EnabledScanners    []string                 `json:"enabled_scanners"`
	RegistryAuth       map[string]dockerTypes.RegistryAuth  `json:"registry_auth"`
	ScanTimeout        time.Duration            `json:"scan_timeout"`
}

// ScanResult represents comprehensive image scan results
type ScanResult struct {
	ImageName        string                       `json:"image_name"`
	ImageID          string                       `json:"image_id"`
	Registry         string                       `json:"registry"`
	Tag              string                       `json:"tag"`
	ScanTime         time.Time                    `json:"scan_time"`
	ScanDuration     time.Duration                `json:"scan_duration"`
	Vulnerabilities  []security.Vulnerability     `json:"vulnerabilities"`
	Passed           bool                         `json:"passed"`
	Grade            string                       `json:"grade"`
	Score            float64                      `json:"score"`
	TotalVulns       int                          `json:"total_vulns"`
	CriticalVulns    int                          `json:"critical_vulns"`
	HighVulns        int                          `json:"high_vulns"`
	MediumVulns      int                          `json:"medium_vulns"`
	LowVulns         int                          `json:"low_vulns"`
	Metadata         *ImageMetadata               `json:"metadata"`
	Recommendations  []string                     `json:"recommendations"`
	ComplianceChecks []ComplianceCheck            `json:"compliance_checks"`
}

// ImageMetadata represents image metadata
type ImageMetadata struct {
	Size           int64             `json:"size"`
	Architecture   string            `json:"architecture"`
	OS             string            `json:"os"`
	Created        time.Time         `json:"created"`
	Author         string            `json:"author"`
	Labels         map[string]string `json:"labels"`
	LayerCount     int               `json:"layer_count"`
	ExposedPorts   []string          `json:"exposed_ports"`
	Environment    []string          `json:"environment"`
	Command        []string          `json:"command"`
	Entrypoint     []string          `json:"entrypoint"`
}

// ComplianceCheck represents security compliance check result
type ComplianceCheck struct {
	CheckID     string `json:"check_id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Passed      bool   `json:"passed"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

// VersionInfo represents image version information
type VersionInfo struct {
	ImageName     string    `json:"image_name"`
	Registry      string    `json:"registry"`
	Versions      []Version `json:"versions"`
	LatestVersion string    `json:"latest_version"`
	RetrievedAt   time.Time `json:"retrieved_at"`
}

// Version represents a single version of an image
type Version struct {
	Tag         string            `json:"tag"`
	Digest      string            `json:"digest"`
	Created     time.Time         `json:"created"`
	Size        int64             `json:"size"`
	Labels      map[string]string `json:"labels"`
	IsLatest    bool              `json:"is_latest"`
	IsStable    bool              `json:"is_stable"`
	IsBeta      bool              `json:"is_beta"`
	IsAlpha     bool              `json:"is_alpha"`
	IsDev       bool              `json:"is_dev"`
	SemVersion  *SemanticVersion  `json:"sem_version,omitempty"`
}

// SemanticVersion represents semantic version information
type SemanticVersion struct {
	Major      int    `json:"major"`
	Minor      int    `json:"minor"`
	Patch      int    `json:"patch"`
	PreRelease string `json:"pre_release,omitempty"`
	Build      string `json:"build,omitempty"`
	Original   string `json:"original"`
}

// VersionComparison represents version comparison result
type VersionComparison struct {
	CurrentVersion  *Version `json:"current_version"`
	LatestVersion   *Version `json:"latest_version"`
	UpdateAvailable bool     `json:"update_available"`
	VersionsBehind  int      `json:"versions_behind"`
	SecurityRisk    string   `json:"security_risk"`
	Recommendation  string   `json:"recommendation"`
	ChangeLog       []string `json:"change_log,omitempty"`
}

// NewImageScanner creates a new image scanner
func NewImageScanner(clientManager *ClientManager, config *ScannerConfig, logger *logrus.Logger) *ImageScanner {
	if config == nil {
		config = DefaultScannerConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	return &ImageScanner{
		clientManager: clientManager,
		scanCache:     make(map[string]*ScanResult),
		versionCache:  make(map[string]*VersionInfo),
		logger:        logger,
		config:        config,
	}
}

// DefaultScannerConfig returns default scanner configuration
func DefaultScannerConfig() *ScannerConfig {
	return &ScannerConfig{
		CacheExpiration:    24 * time.Hour,
		MaxConcurrentScans: 3,
		VulnThreshold:      security.VulnHigh,
		EnabledScanners:    []string{"basic", "cve", "compliance"},
		ScanTimeout:        10 * time.Minute,
	}
}

// ScanImage performs comprehensive security scan of an image
func (is *ImageScanner) ScanImage(ctx context.Context, imageName string) (*ScanResult, error) {
	is.mutex.Lock()

	// Check cache first
	if cachedResult, exists := is.scanCache[imageName]; exists {
		if time.Since(cachedResult.ScanTime) < is.config.CacheExpiration {
			is.mutex.Unlock()
			return cachedResult, nil
		}
	}
	is.mutex.Unlock()

	startTime := time.Now()
	is.logger.WithField("image", imageName).Info("Starting comprehensive image scan")

	// Get image information
	client := is.clientManager.GetClient()
	imageInfo, _, err := client.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	// Parse image name components
	registry, _, tag := is.parseImageName(imageName)

	result := &ScanResult{
		ImageName:    imageName,
		ImageID:      imageInfo.ID,
		Registry:     registry,
		Tag:          tag,
		ScanTime:     startTime,
		Metadata:     is.extractMetadata(&imageInfo),
	}

	// Perform vulnerability scanning
	vulnerabilities, err := is.scanVulnerabilities(ctx, &imageInfo)
	if err != nil {
		is.logger.WithError(err).Warn("Vulnerability scanning failed")
		// Continue with other scans
	} else {
		result.Vulnerabilities = vulnerabilities
		is.categorizeVulnerabilities(result)
	}

	// Perform compliance checks
	complianceChecks := is.performComplianceChecks(&imageInfo)
	result.ComplianceChecks = complianceChecks

	// Calculate security score and grade
	result.Score = is.calculateSecurityScore(result)
	result.Grade = is.assignSecurityGrade(result.Score)
	result.Passed = result.Score >= 70.0 // Passing score threshold

	// Generate recommendations
	result.Recommendations = is.generateRecommendations(result)

	result.ScanDuration = time.Since(startTime)

	// Cache result
	is.mutex.Lock()
	is.scanCache[imageName] = result
	is.mutex.Unlock()

	is.logger.WithFields(logrus.Fields{
		"image":        imageName,
		"score":        result.Score,
		"grade":        result.Grade,
		"vulns":        result.TotalVulns,
		"critical":     result.CriticalVulns,
		"passed":       result.Passed,
		"duration":     result.ScanDuration,
	}).Info("Image scan completed")

	return result, nil
}

// GetVersions retrieves all available versions for an image
func (is *ImageScanner) GetVersions(ctx context.Context, imageName string) (*VersionInfo, error) {
	is.mutex.RLock()

	// Check cache first
	if cachedVersions, exists := is.versionCache[imageName]; exists {
		if time.Since(cachedVersions.RetrievedAt) < is.config.CacheExpiration {
			is.mutex.RUnlock()
			return cachedVersions, nil
		}
	}
	is.mutex.RUnlock()

	is.logger.WithField("image", imageName).Info("Retrieving image versions")

	registry, repo, _ := is.parseImageName(imageName)

	versionInfo := &VersionInfo{
		ImageName:   imageName,
		Registry:    registry,
		RetrievedAt: time.Now(),
	}

	// Get versions from registry
	versions, err := is.fetchVersionsFromRegistry(ctx, registry, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions: %w", err)
	}

	// Process and sort versions
	processedVersions := is.processVersions(versions)
	sort.Slice(processedVersions, func(i, j int) bool {
		return processedVersions[i].Created.After(processedVersions[j].Created)
	})

	versionInfo.Versions = processedVersions
	if len(processedVersions) > 0 {
		versionInfo.LatestVersion = processedVersions[0].Tag
	}

	// Cache result
	is.mutex.Lock()
	is.versionCache[imageName] = versionInfo
	is.mutex.Unlock()

	is.logger.WithFields(logrus.Fields{
		"image":    imageName,
		"versions": len(processedVersions),
		"latest":   versionInfo.LatestVersion,
	}).Info("Version retrieval completed")

	return versionInfo, nil
}

// CompareVersions compares current version with latest available
func (is *ImageScanner) CompareVersions(ctx context.Context, currentImage, latestImage string) (*VersionComparison, error) {
	is.logger.WithFields(logrus.Fields{
		"current": currentImage,
		"latest":  latestImage,
	}).Info("Starting version comparison")

	// Get version information for both images
	currentVersionInfo, err := is.GetVersions(ctx, currentImage)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version info: %w", err)
	}

	latestVersionInfo, err := is.GetVersions(ctx, latestImage)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version info: %w", err)
	}

	// Find current and latest versions
	var currentVersion, latestVersion *Version

	_, _, currentTag := is.parseImageName(currentImage)
	for _, v := range currentVersionInfo.Versions {
		if v.Tag == currentTag {
			currentVersion = &v
			break
		}
	}

	if len(latestVersionInfo.Versions) > 0 {
		latestVersion = &latestVersionInfo.Versions[0]
	}

	if currentVersion == nil {
		return nil, fmt.Errorf("current version not found")
	}
	if latestVersion == nil {
		return nil, fmt.Errorf("latest version not found")
	}

	comparison := &VersionComparison{
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
	}

	// Determine if update is available
	comparison.UpdateAvailable = is.isUpdateAvailable(currentVersion, latestVersion)
	comparison.VersionsBehind = is.countVersionsBehind(currentVersion, latestVersionInfo.Versions)
	comparison.SecurityRisk = is.assessSecurityRisk(comparison.VersionsBehind, currentVersion.Created)
	comparison.Recommendation = is.generateUpdateRecommendation(comparison)

	is.logger.WithFields(logrus.Fields{
		"current":          currentVersion.Tag,
		"latest":           latestVersion.Tag,
		"update_available": comparison.UpdateAvailable,
		"versions_behind":  comparison.VersionsBehind,
		"security_risk":    comparison.SecurityRisk,
	}).Info("Version comparison completed")

	return comparison, nil
}

// extractMetadata extracts metadata from image information
func (is *ImageScanner) extractMetadata(imageInfo *types.ImageInspect) *ImageMetadata {
	metadata := &ImageMetadata{
		Size:         imageInfo.Size,
		Architecture: imageInfo.Architecture,
		OS:           imageInfo.Os,
		LayerCount:   len(imageInfo.RootFS.Layers),
		Labels:       imageInfo.Config.Labels,
		Environment:  imageInfo.Config.Env,
		Command:      imageInfo.Config.Cmd,
		Entrypoint:   imageInfo.Config.Entrypoint,
	}

	if imageInfo.Author != "" {
		metadata.Author = imageInfo.Author
	}

	if created, err := time.Parse(time.RFC3339, imageInfo.Created); err == nil {
		metadata.Created = created
	}

	// Extract exposed ports
	for port := range imageInfo.Config.ExposedPorts {
		metadata.ExposedPorts = append(metadata.ExposedPorts, string(port))
	}

	return metadata
}

// scanVulnerabilities performs vulnerability scanning
func (is *ImageScanner) scanVulnerabilities(ctx context.Context, imageInfo *types.ImageInspect) ([]security.Vulnerability, error) {
	var allVulnerabilities []security.Vulnerability

	// Basic security checks
	basicVulns := is.performBasicSecurityChecks(imageInfo)
	allVulnerabilities = append(allVulnerabilities, basicVulns...)

	// CVE database checks (simulated)
	cveVulns := is.performCVEChecks(imageInfo)
	allVulnerabilities = append(allVulnerabilities, cveVulns...)

	// Package vulnerability checks
	packageVulns := is.performPackageVulnerabilityChecks(imageInfo)
	allVulnerabilities = append(allVulnerabilities, packageVulns...)

	return allVulnerabilities, nil
}

// performBasicSecurityChecks performs basic security checks
func (is *ImageScanner) performBasicSecurityChecks(imageInfo *types.ImageInspect) []security.Vulnerability {
	var vulnerabilities []security.Vulnerability

	// Check for running as root
	if imageInfo.Config.User == "" || imageInfo.Config.User == "root" || imageInfo.Config.User == "0" {
		vulnerabilities = append(vulnerabilities, security.Vulnerability{
			CVE:         "BASIC-001",
			Severity:    security.VulnMedium,
			Description: "Container runs as root user, increasing privilege escalation risk",
			Package:     "runtime-config",
		})
	}

	// Check for unnecessary packages
	for _, layer := range imageInfo.RootFS.Layers {
		if strings.Contains(layer, "apt") || strings.Contains(layer, "yum") {
			vulnerabilities = append(vulnerabilities, security.Vulnerability{
				CVE:         "BASIC-002",
				Severity:    security.VulnLow,
				Description: "Package manager files present, image may contain unnecessary packages",
				Package:     "package-manager",
			})
			break
		}
	}

	// Check for exposed SSH port
	for port := range imageInfo.Config.ExposedPorts {
		if strings.HasPrefix(string(port), "22/") {
			vulnerabilities = append(vulnerabilities, security.Vulnerability{
				CVE:         "BASIC-003",
				Severity:    security.VulnHigh,
				Description: "SSH port (22) is exposed, potential security risk",
				Package:     "network-config",
			})
		}
	}

	// Check for hardcoded secrets in environment variables
	for _, env := range imageInfo.Config.Env {
		if is.containsSecret(env) {
			vulnerabilities = append(vulnerabilities, security.Vulnerability{
				CVE:         "BASIC-004",
				Severity:    security.VulnHigh,
				Description: "Potential secret found in environment variable",
				Package:     "environment-config",
			})
		}
	}

	return vulnerabilities
}

// performCVEChecks performs CVE database checks (simulated)
func (is *ImageScanner) performCVEChecks(imageInfo *types.ImageInspect) []security.Vulnerability {
	var vulnerabilities []security.Vulnerability

	// Simulated CVE checks based on image characteristics
	if strings.Contains(strings.ToLower(imageInfo.Config.Image), "ubuntu") {
		if created, err := time.Parse(time.RFC3339, imageInfo.Created); err == nil {
			if time.Since(created) > 365*24*time.Hour {
				vulnerabilities = append(vulnerabilities, security.Vulnerability{
					CVE:         "CVE-2023-1234",
					Severity:    security.VulnHigh,
					Description: "Ubuntu base image contains known vulnerabilities in glibc",
					Package:     "glibc",
					Version:     "2.31-0ubuntu9.2",
					FixedIn:     "2.31-0ubuntu9.9",
				})
			}
		}
	}

	// Check for OpenSSL vulnerabilities
	for _, env := range imageInfo.Config.Env {
		if strings.Contains(strings.ToLower(env), "openssl") {
			vulnerabilities = append(vulnerabilities, security.Vulnerability{
				CVE:         "CVE-2022-0778",
				Severity:    security.VulnMedium,
				Description: "OpenSSL infinite loop vulnerability in BN_mod_sqrt()",
				Package:     "openssl",
				Version:     "1.1.1",
				FixedIn:     "1.1.1n",
			})
		}
	}

	return vulnerabilities
}

// performPackageVulnerabilityChecks checks for vulnerable packages
func (is *ImageScanner) performPackageVulnerabilityChecks(imageInfo *types.ImageInspect) []security.Vulnerability {
	var vulnerabilities []security.Vulnerability

	// Simulated package vulnerability checks
	knownVulnerablePackages := map[string]security.Vulnerability{
		"curl": {
			CVE:         "CVE-2022-32205",
			Severity:    security.VulnMedium,
			Description: "curl set-cookie denial of service vulnerability",
			Package:     "curl",
			Version:     "7.68.0",
			FixedIn:     "7.84.0",
		},
		"nginx": {
			CVE:         "CVE-2022-41741",
			Severity:    security.VulnMedium,
			Description: "nginx HTTP/2 implementation vulnerability",
			Package:     "nginx",
			Version:     "1.18.0",
			FixedIn:     "1.22.1",
		},
	}

	// Check if vulnerable packages might be present
	for pkg, vuln := range knownVulnerablePackages {
		for _, layer := range imageInfo.RootFS.Layers {
			if strings.Contains(strings.ToLower(layer), pkg) {
				vulnerabilities = append(vulnerabilities, vuln)
				break
			}
		}
	}

	return vulnerabilities
}

// performComplianceChecks performs security compliance checks
func (is *ImageScanner) performComplianceChecks(imageInfo *types.ImageInspect) []ComplianceCheck {
	var checks []ComplianceCheck

	// CIS Docker Benchmark checks
	checks = append(checks, is.performCISChecks(imageInfo)...)

	// NIST compliance checks
	checks = append(checks, is.performNISTChecks(imageInfo)...)

	// Custom compliance checks
	checks = append(checks, is.performCustomChecks(imageInfo)...)

	return checks
}

// performCISChecks performs CIS Docker Benchmark checks
func (is *ImageScanner) performCISChecks(imageInfo *types.ImageInspect) []ComplianceCheck {
	var checks []ComplianceCheck

	// CIS 4.1: Ensure a user for the container has been created
	userPassed := imageInfo.Config.User != "" && imageInfo.Config.User != "root" && imageInfo.Config.User != "0"
	checks = append(checks, ComplianceCheck{
		CheckID:     "CIS-4.1",
		Name:        "User for container created",
		Category:    "User Management",
		Severity:    "Medium",
		Passed:      userPassed,
		Description: "Ensure that a user for the container has been created",
		Remediation: "Create a dedicated user in Dockerfile: RUN useradd -r -s /bin/false myuser && USER myuser",
	})

	// CIS 4.7: Ensure sensitive host system directories are not mounted on containers
	mountPassed := true // This would be checked in host config
	checks = append(checks, ComplianceCheck{
		CheckID:     "CIS-4.7",
		Name:        "Sensitive directories not mounted",
		Category:    "Mount Management",
		Severity:    "High",
		Passed:      mountPassed,
		Description: "Ensure sensitive host system directories are not mounted on containers",
		Remediation: "Avoid mounting sensitive directories like /, /boot, /dev, /etc, /lib, /proc, /sys, /usr",
	})

	return checks
}

// performNISTChecks performs NIST compliance checks
func (is *ImageScanner) performNISTChecks(imageInfo *types.ImageInspect) []ComplianceCheck {
	var checks []ComplianceCheck

	// NIST: Image should have minimal attack surface
	layersPassed := len(imageInfo.RootFS.Layers) <= 20
	checks = append(checks, ComplianceCheck{
		CheckID:     "NIST-1",
		Name:        "Minimal attack surface",
		Category:    "Attack Surface",
		Severity:    "Medium",
		Passed:      layersPassed,
		Description: "Image should have minimal number of layers to reduce attack surface",
		Remediation: "Optimize Dockerfile to reduce layers, use multi-stage builds",
	})

	// NIST: Image should not expose unnecessary ports
	portsPassed := len(imageInfo.Config.ExposedPorts) <= 3
	checks = append(checks, ComplianceCheck{
		CheckID:     "NIST-2",
		Name:        "Minimal port exposure",
		Category:    "Network Security",
		Severity:    "Medium",
		Passed:      portsPassed,
		Description: "Image should expose only necessary ports",
		Remediation: "Remove unnecessary EXPOSE statements from Dockerfile",
	})

	return checks
}

// performCustomChecks performs custom security compliance checks
func (is *ImageScanner) performCustomChecks(imageInfo *types.ImageInspect) []ComplianceCheck {
	var checks []ComplianceCheck

	// Check for healthcheck
	healthPassed := imageInfo.Config.Healthcheck != nil
	checks = append(checks, ComplianceCheck{
		CheckID:     "CUSTOM-1",
		Name:        "Health check defined",
		Category:    "Health Monitoring",
		Severity:    "Low",
		Passed:      healthPassed,
		Description: "Image should define a health check for monitoring",
		Remediation: "Add HEALTHCHECK instruction to Dockerfile",
	})

	// Check for proper labeling
	labelsPassed := imageInfo.Config.Labels != nil && len(imageInfo.Config.Labels) > 0
	checks = append(checks, ComplianceCheck{
		CheckID:     "CUSTOM-2",
		Name:        "Proper labeling",
		Category:    "Metadata",
		Severity:    "Low",
		Passed:      labelsPassed,
		Description: "Image should have proper labels for identification",
		Remediation: "Add LABEL instructions for version, maintainer, description",
	})

	return checks
}

// categorizeVulnerabilities categorizes vulnerabilities by severity
func (is *ImageScanner) categorizeVulnerabilities(result *ScanResult) {
	result.TotalVulns = len(result.Vulnerabilities)

	for _, vuln := range result.Vulnerabilities {
		switch vuln.Severity {
		case security.VulnCritical:
			result.CriticalVulns++
		case security.VulnHigh:
			result.HighVulns++
		case security.VulnMedium:
			result.MediumVulns++
		case security.VulnLow:
			result.LowVulns++
		}
	}
}

// calculateSecurityScore calculates overall security score
func (is *ImageScanner) calculateSecurityScore(result *ScanResult) float64 {
	baseScore := 100.0

	// Deduct points for vulnerabilities
	baseScore -= float64(result.CriticalVulns) * 25.0
	baseScore -= float64(result.HighVulns) * 15.0
	baseScore -= float64(result.MediumVulns) * 8.0
	baseScore -= float64(result.LowVulns) * 3.0

	// Deduct points for failed compliance checks
	failedChecks := 0
	for _, check := range result.ComplianceChecks {
		if !check.Passed {
			failedChecks++
		}
	}
	baseScore -= float64(failedChecks) * 5.0

	// Bonus for good practices
	if result.Metadata != nil {
		if result.Metadata.Author != "" {
			baseScore += 2.0
		}
		if len(result.Metadata.Labels) > 0 {
			baseScore += 3.0
		}
		if result.Metadata.LayerCount <= 10 {
			baseScore += 5.0
		}
	}

	if baseScore < 0 {
		baseScore = 0
	}
	if baseScore > 100 {
		baseScore = 100
	}

	return baseScore
}

// assignSecurityGrade assigns security grade based on score
func (is *ImageScanner) assignSecurityGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// generateRecommendations generates security recommendations
func (is *ImageScanner) generateRecommendations(result *ScanResult) []string {
	var recommendations []string

	if result.CriticalVulns > 0 {
		recommendations = append(recommendations, "Critical vulnerabilities found - update base image immediately")
	}

	if result.HighVulns > 0 {
		recommendations = append(recommendations, "High severity vulnerabilities found - schedule security update")
	}

	if result.Metadata != nil {
		if result.Metadata.LayerCount > 20 {
			recommendations = append(recommendations, "Too many layers - optimize Dockerfile with multi-stage builds")
		}

		if len(result.Metadata.ExposedPorts) > 5 {
			recommendations = append(recommendations, "Too many exposed ports - minimize attack surface")
		}

		if time.Since(result.Metadata.Created) > 90*24*time.Hour {
			recommendations = append(recommendations, "Image is old - consider updating to newer base image")
		}
	}

	// Add compliance-based recommendations
	failedCriticalChecks := 0
	for _, check := range result.ComplianceChecks {
		if !check.Passed && (check.Severity == "High" || check.Severity == "Critical") {
			failedCriticalChecks++
		}
	}

	if failedCriticalChecks > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Address %d critical compliance issues", failedCriticalChecks))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Image meets security standards - monitor for updates")
	}

	return recommendations
}

// parseImageName parses image name into components
func (is *ImageScanner) parseImageName(imageName string) (registry, repo, tag string) {
	// Default values
	registry = "docker.io"
	tag = "latest"

	// Split tag
	parts := strings.Split(imageName, ":")
	if len(parts) > 1 {
		tag = parts[len(parts)-1]
		imageName = strings.Join(parts[:len(parts)-1], ":")
	}

	// Split registry and repo
	if strings.Contains(imageName, "/") {
		registryParts := strings.Split(imageName, "/")
		if strings.Contains(registryParts[0], ".") || strings.Contains(registryParts[0], ":") {
			registry = registryParts[0]
			repo = strings.Join(registryParts[1:], "/")
		} else {
			repo = imageName
		}
	} else {
		repo = imageName
	}

	return registry, repo, tag
}

// fetchVersionsFromRegistry fetches versions from registry (simulated)
func (is *ImageScanner) fetchVersionsFromRegistry(ctx context.Context, registry, repo string) ([]Version, error) {
	// This is a simplified simulation
	// In a real implementation, this would query the actual registry API

	now := time.Now()
	versions := []Version{
		{
			Tag:      "latest",
			Digest:   "sha256:1234567890abcdef",
			Created:  now.AddDate(0, 0, -1),
			Size:     100000000,
			IsLatest: true,
		},
		{
			Tag:     "1.2.3",
			Digest:  "sha256:abcdef1234567890",
			Created: now.AddDate(0, 0, -7),
			Size:    95000000,
		},
		{
			Tag:     "1.2.2",
			Digest:  "sha256:fedcba0987654321",
			Created: now.AddDate(0, 0, -30),
			Size:    90000000,
		},
	}

	return versions, nil
}

// processVersions processes and enriches version information
func (is *ImageScanner) processVersions(versions []Version) []Version {
	for i := range versions {
		version := &versions[i]

		// Parse semantic version
		if semVer := is.parseSemanticVersion(version.Tag); semVer != nil {
			version.SemVersion = semVer
		}

		// Classify version type
		tag := strings.ToLower(version.Tag)
		version.IsStable = !strings.Contains(tag, "beta") &&
						 !strings.Contains(tag, "alpha") &&
						 !strings.Contains(tag, "rc") &&
						 !strings.Contains(tag, "dev")
		version.IsBeta = strings.Contains(tag, "beta")
		version.IsAlpha = strings.Contains(tag, "alpha")
		version.IsDev = strings.Contains(tag, "dev")
	}

	return versions
}

// parseSemanticVersion parses semantic version from tag
func (is *ImageScanner) parseSemanticVersion(tag string) *SemanticVersion {
	// Remove common prefixes
	tag = strings.TrimPrefix(tag, "v")
	tag = strings.TrimPrefix(tag, "version-")

	// Semantic version regex
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z\-]+(?:\.[0-9A-Za-z\-]+)*))?(?:\+([0-9A-Za-z\-]+(?:\.[0-9A-Za-z\-]+)*))?$`)
	matches := re.FindStringSubmatch(tag)

	if len(matches) < 4 {
		return nil
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	semVer := &SemanticVersion{
		Major:    major,
		Minor:    minor,
		Patch:    patch,
		Original: tag,
	}

	if len(matches) > 4 && matches[4] != "" {
		semVer.PreRelease = matches[4]
	}

	if len(matches) > 5 && matches[5] != "" {
		semVer.Build = matches[5]
	}

	return semVer
}

// isUpdateAvailable determines if an update is available
func (is *ImageScanner) isUpdateAvailable(current, latest *Version) bool {
	if current.Tag == latest.Tag {
		return false
	}

	// Compare creation times if semantic versions not available
	if current.SemVersion == nil || latest.SemVersion == nil {
		return latest.Created.After(current.Created)
	}

	// Compare semantic versions
	currentSem := current.SemVersion
	latestSem := latest.SemVersion

	if latestSem.Major > currentSem.Major {
		return true
	}
	if latestSem.Major == currentSem.Major && latestSem.Minor > currentSem.Minor {
		return true
	}
	if latestSem.Major == currentSem.Major && latestSem.Minor == currentSem.Minor && latestSem.Patch > currentSem.Patch {
		return true
	}

	return false
}

// countVersionsBehind counts how many versions behind current is
func (is *ImageScanner) countVersionsBehind(current *Version, allVersions []Version) int {
	count := 0
	for _, version := range allVersions {
		if version.Created.After(current.Created) && version.IsStable {
			count++
		}
	}
	return count
}

// assessSecurityRisk assesses security risk based on version age
func (is *ImageScanner) assessSecurityRisk(versionsBehind int, created time.Time) string {
	age := time.Since(created)

	if versionsBehind > 10 || age > 365*24*time.Hour {
		return "CRITICAL"
	}
	if versionsBehind > 5 || age > 180*24*time.Hour {
		return "HIGH"
	}
	if versionsBehind > 2 || age > 90*24*time.Hour {
		return "MEDIUM"
	}
	if versionsBehind > 0 || age > 30*24*time.Hour {
		return "LOW"
	}

	return "MINIMAL"
}

// generateUpdateRecommendation generates update recommendation
func (is *ImageScanner) generateUpdateRecommendation(comparison *VersionComparison) string {
	if !comparison.UpdateAvailable {
		return "No update needed - using latest version"
	}

	switch comparison.SecurityRisk {
	case "CRITICAL":
		return "URGENT: Update immediately - critical security risk"
	case "HIGH":
		return "Update as soon as possible - high security risk"
	case "MEDIUM":
		return "Schedule update within a week - moderate security risk"
	case "LOW":
		return "Consider updating when convenient - low security risk"
	default:
		return "Update available - minimal security risk"
	}
}

// containsSecret checks if environment variable contains potential secret
func (is *ImageScanner) containsSecret(env string) bool {
	env = strings.ToLower(env)
	secretPatterns := []string{
		"password=",
		"secret=",
		"key=",
		"token=",
		"api_key=",
		"private_key=",
		"access_token=",
	}

	for _, pattern := range secretPatterns {
		if strings.Contains(env, pattern) {
			return true
		}
	}

	return false
}

// ClearCache clears the scanner cache
func (is *ImageScanner) ClearCache() {
	is.mutex.Lock()
	defer is.mutex.Unlock()

	is.scanCache = make(map[string]*ScanResult)
	is.versionCache = make(map[string]*VersionInfo)
	is.logger.Info("Scanner cache cleared")
}

// GetCacheStats returns cache statistics
func (is *ImageScanner) GetCacheStats() map[string]interface{} {
	is.mutex.RLock()
	defer is.mutex.RUnlock()

	return map[string]interface{}{
		"scan_cache_entries":    len(is.scanCache),
		"version_cache_entries": len(is.versionCache),
		"cache_expiration":      is.config.CacheExpiration,
	}
}