package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"docker-auto/pkg/docker"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

// TrivySecurityScanner implements SecurityScanner using Trivy
type TrivySecurityScanner struct {
	dockerClient *docker.DockerClient
	config       *ScannerConfig
	logger       *logrus.Entry
}

// NewTrivySecurityScanner creates a new Trivy-based security scanner
func NewTrivySecurityScanner(dockerClient *docker.DockerClient, config *ScannerConfig, logger *logrus.Entry) SecurityScanner {
	if config == nil {
		config = &ScannerConfig{
			Timeout:             5 * time.Minute,
			MaxConcurrentScans:  3,
			EnabledScanners:     []string{"trivy"},
			ScanSecrets:         true,
			ScanConfiguration:   true,
			ScanLicenses:        true,
			OutputFormat:        "json",
			IncludePackages:     true,
			IncludeLayers:       true,
		}
	}

	return &TrivySecurityScanner{
		dockerClient: dockerClient,
		config:       config,
		logger:       logger.WithField("component", "trivy_scanner"),
	}
}

// ScanContainer scans a running container for security vulnerabilities
func (t *TrivySecurityScanner) ScanContainer(ctx context.Context, containerID string) (*ScanResult, error) {
	t.logger.WithField("container_id", containerID).Info("Starting container security scan")

	startTime := time.Now()
	scanID := fmt.Sprintf("scan_%s_%d", containerID, startTime.UnixNano())

	// Get container information
	containerJSON, err := t.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	imageName := containerJSON.Config.Image
	imageID := containerJSON.Image

	// Create timeout context
	scanCtx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	// Run Trivy scan
	vulnerabilities, err := t.scanImageWithTrivy(scanCtx, imageName)
	if err != nil {
		return &ScanResult{
			ScanID:        scanID,
			ContainerID:   containerID,
			ImageID:       imageID,
			ImageName:     imageName,
			ScanStartTime: startTime,
			ScanEndTime:   time.Now(),
			ScanDuration:  time.Since(startTime),
			Scanner:       "trivy",
			Success:       false,
			ErrorMessage:  err.Error(),
		}, nil // Return partial result with error
	}

	// Scan configuration if enabled
	var configIssues []*ConfigurationIssue
	if t.config.ScanConfiguration {
		configIssues, err = t.scanContainerConfiguration(scanCtx, containerJSON)
		if err != nil {
			t.logger.WithError(err).Warn("Failed to scan container configuration")
		}
	}

	// Scan secrets if enabled
	var secretIssues []*SecretIssue
	if t.config.ScanSecrets {
		secretIssues, err = t.scanContainerSecrets(scanCtx, containerID)
		if err != nil {
			t.logger.WithError(err).Warn("Failed to scan container secrets")
		}
	}

	// Calculate vulnerability stats
	stats := t.calculateVulnerabilityStats(vulnerabilities)

	// Calculate risk score
	riskScore := t.calculateRiskScore(stats, configIssues, secretIssues)

	// Determine security grade
	securityGrade := t.determineSecurityGrade(riskScore)

	endTime := time.Now()

	result := &ScanResult{
		ScanID:              scanID,
		ContainerID:         containerID,
		ImageID:             imageID,
		ImageName:           imageName,
		ScanStartTime:       startTime,
		ScanEndTime:         endTime,
		ScanDuration:        endTime.Sub(startTime),
		Scanner:             "trivy",
		ScannerVersion:      t.getTrivyVersion(),
		Vulnerabilities:     vulnerabilities,
		VulnerabilityStats:  stats,
		ConfigurationIssues: configIssues,
		SecretIssues:        secretIssues,
		RiskScore:           riskScore,
		SecurityGrade:       securityGrade,
		Metadata: map[string]interface{}{
			"scan_options": t.config,
		},
		Success: true,
	}

	t.logger.WithFields(logrus.Fields{
		"container_id":    containerID,
		"scan_duration":   result.ScanDuration,
		"vulnerabilities": stats.Total,
		"risk_score":      riskScore,
		"security_grade":  securityGrade,
	}).Info("Container security scan completed")

	return result, nil
}

// ScanImage scans a container image for security vulnerabilities
func (t *TrivySecurityScanner) ScanImage(ctx context.Context, imageID string) (*ImageScanResult, error) {
	t.logger.WithField("image_id", imageID).Info("Starting image security scan")

	// Create timeout context
	scanCtx, cancel := context.WithTimeout(ctx, t.config.Timeout)
	defer cancel()

	// Get image information
	imageInfo, err := t.dockerClient.InspectImage(scanCtx, imageID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	// Run Trivy scan
	vulnerabilities, err := t.scanImageWithTrivy(scanCtx, imageID)
	if err != nil {
		return &ImageScanResult{
			ImageID:      imageID,
			Success:      false,
			ErrorMessage: err.Error(),
			ScanTime:     time.Now(),
			Scanner:      "trivy",
		}, nil
	}

	// Get packages if enabled
	var packages []*PackageInfo
	if t.config.IncludePackages {
		packages, err = t.getImagePackages(scanCtx, imageID)
		if err != nil {
			t.logger.WithError(err).Warn("Failed to get image packages")
		}
	}

	// Calculate stats
	stats := t.calculateVulnerabilityStats(vulnerabilities)

	// Extract image details
	var imageName, imageTag, registry string
	if len(imageInfo.RepoTags) > 0 {
		parts := strings.Split(imageInfo.RepoTags[0], ":")
		if len(parts) >= 2 {
			imageName = parts[0]
			imageTag = parts[1]
		} else {
			imageName = imageInfo.RepoTags[0]
			imageTag = "latest"
		}

		// Extract registry
		if strings.Contains(imageName, "/") {
			registryParts := strings.Split(imageName, "/")
			if len(registryParts) > 1 && strings.Contains(registryParts[0], ".") {
				registry = registryParts[0]
			}
		}
	}

	result := &ImageScanResult{
		ImageID:            imageID,
		ImageName:          imageName,
		ImageTag:           imageTag,
		Registry:           registry,
		Size:               imageInfo.Size,
		Architecture:       imageInfo.Architecture,
		OS:                 imageInfo.Os,
		Vulnerabilities:    vulnerabilities,
		VulnerabilityStats: stats,
		Packages:           packages,
		ScanTime:           time.Now(),
		Scanner:            "trivy",
		Success:            true,
	}

	// Add digest if available
	if len(imageInfo.RepoDigests) > 0 {
		result.Digest = imageInfo.RepoDigests[0]
	}

	return result, nil
}

// ScanVulnerabilities performs a focused vulnerability scan
func (t *TrivySecurityScanner) ScanVulnerabilities(ctx context.Context, containerID string) (*VulnerabilityScanResult, error) {
	// Get container information
	containerJSON, err := t.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	// Run vulnerability scan
	vulnerabilities, err := t.scanImageWithTrivy(ctx, containerJSON.Config.Image)
	if err != nil {
		return nil, fmt.Errorf("vulnerability scan failed: %w", err)
	}

	// Calculate stats
	stats := t.calculateVulnerabilityStats(vulnerabilities)

	return &VulnerabilityScanResult{
		ContainerID:       containerID,
		Vulnerabilities:   vulnerabilities,
		Stats:             stats,
		ScanTime:          time.Now(),
		DatabaseVersion:   t.getDatabaseVersion(),
		DatabaseUpdatedAt: t.getDatabaseUpdatedAt(),
	}, nil
}

// GetVulnerabilityDetails gets detailed information about a specific vulnerability
func (t *TrivySecurityScanner) GetVulnerabilityDetails(ctx context.Context, vulnerabilityID string) (*VulnerabilityDetail, error) {
	// This would typically query a vulnerability database
	// For now, return a basic implementation
	return &VulnerabilityDetail{
		Vulnerability: &Vulnerability{
			ID:          vulnerabilityID,
			CVE:         vulnerabilityID,
			Title:       "Vulnerability details",
			Description: "Detailed vulnerability information",
		},
	}, nil
}

// ScanConfiguration scans container configuration for security issues
func (t *TrivySecurityScanner) ScanConfiguration(ctx context.Context, containerID string) (*ConfigurationScanResult, error) {
	containerJSON, err := t.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	issues, err := t.scanContainerConfiguration(ctx, containerJSON)
	if err != nil {
		return nil, fmt.Errorf("configuration scan failed: %w", err)
	}

	// Generate recommendations based on issues
	recommendations := t.generateConfigurationRecommendations(issues)

	// Calculate compliance score
	complianceScore := t.calculateComplianceScore(issues)

	return &ConfigurationScanResult{
		ContainerID:     containerID,
		Issues:          issues,
		Recommendations: recommendations,
		ComplianceScore: complianceScore,
		ScanTime:        time.Now(),
	}, nil
}

// ScanSecrets scans container for exposed secrets and sensitive information
func (t *TrivySecurityScanner) ScanSecrets(ctx context.Context, containerID string) (*SecretsScanResult, error) {
	secrets, err := t.scanContainerSecrets(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("secrets scan failed: %w", err)
	}

	return &SecretsScanResult{
		ContainerID: containerID,
		Secrets:     secrets,
		ScanTime:    time.Now(),
	}, nil
}

// ScanCompliance performs compliance scanning against specified standards
func (t *TrivySecurityScanner) ScanCompliance(ctx context.Context, containerID string, standards []string) (*ComplianceScanResult, error) {
	// This is a placeholder implementation
	// In a real implementation, this would check against actual compliance frameworks

	containerJSON, err := t.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	var allChecks []*ComplianceCheck

	for _, standard := range standards {
		checks := t.performComplianceChecks(containerJSON, standard)
		allChecks = append(allChecks, checks...)
	}

	// Calculate overall score
	overallScore := t.calculateOverallComplianceScore(allChecks)

	// Determine grade
	grade := t.determineComplianceGrade(overallScore)

	return &ComplianceScanResult{
		ContainerID:  containerID,
		Standard:     strings.Join(standards, ","),
		Checks:       allChecks,
		OverallScore: overallScore,
		Grade:        grade,
		ScanTime:     time.Now(),
	}, nil
}

// ScanMultipleContainers scans multiple containers concurrently
func (t *TrivySecurityScanner) ScanMultipleContainers(ctx context.Context, containerIDs []string) (map[string]*ScanResult, error) {
	results := make(map[string]*ScanResult)
	errors := make(map[string]error)

	// Use a semaphore to limit concurrent scans
	semaphore := make(chan struct{}, t.config.MaxConcurrentScans)
	resultChan := make(chan struct {
		containerID string
		result      *ScanResult
		err         error
	}, len(containerIDs))

	// Start concurrent scans
	for _, containerID := range containerIDs {
		go func(id string) {
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result, err := t.ScanContainer(ctx, id)
			resultChan <- struct {
				containerID string
				result      *ScanResult
				err         error
			}{id, result, err}
		}(containerID)
	}

	// Collect results
	for i := 0; i < len(containerIDs); i++ {
		select {
		case result := <-resultChan:
			if result.err != nil {
				errors[result.containerID] = result.err
				t.logger.WithError(result.err).WithField("container_id", result.containerID).Error("Failed to scan container")
			} else {
				results[result.containerID] = result.result
			}
		case <-ctx.Done():
			return results, ctx.Err()
		}
	}

	if len(errors) > 0 {
		t.logger.WithField("errors", errors).Warn("Some container scans failed")
	}

	return results, nil
}

// GetScannerInfo returns information about the scanner
func (t *TrivySecurityScanner) GetScannerInfo() *ScannerInfo {
	return &ScannerInfo{
		Name:              "Trivy",
		Version:           t.getTrivyVersion(),
		DatabaseVersion:   t.getDatabaseVersion(),
		DatabaseUpdatedAt: t.getDatabaseUpdatedAt(),
		SupportedFormats:  []string{"json", "table", "sarif"},
		Capabilities:      []string{"vulnerabilities", "secrets", "configuration", "licenses"},
	}
}

// UpdateScannerConfig updates the scanner configuration
func (t *TrivySecurityScanner) UpdateScannerConfig(config *ScannerConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	t.config = config
	t.logger.Info("Scanner configuration updated")
	return nil
}

// GetScanHistory returns scan history for a container
func (t *TrivySecurityScanner) GetScanHistory(ctx context.Context, containerID string, limit int) ([]*ScanResult, error) {
	// This would typically query a database of scan results
	// For now, return empty slice as this is a stateless implementation
	return []*ScanResult{}, nil
}

// Private helper methods

// scanImageWithTrivy runs Trivy scan on an image
func (t *TrivySecurityScanner) scanImageWithTrivy(ctx context.Context, imageName string) ([]*Vulnerability, error) {
	// Build Trivy command
	args := []string{
		"image",
		"--format", "json",
		"--quiet",
	}

	// Add severity filters if configured
	if len(t.config.IgnoreSeverities) > 0 {
		for _, severity := range t.config.IgnoreSeverities {
			args = append(args, "--ignore-severity", severity)
		}
	}

	// Add CVE ignores if configured
	if len(t.config.IgnoreCVEs) > 0 {
		// Create temporary ignore file
		// This is simplified - in production, you'd create a proper ignore file
		args = append(args, "--ignorefile", "/tmp/trivyignore")
	}

	args = append(args, imageName)

	// Execute Trivy command
	cmd := exec.CommandContext(ctx, "trivy", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("trivy scan failed: %w", err)
	}

	// Parse Trivy output
	var trivyResult struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
				PkgName         string `json:"PkgName"`
				InstalledVersion string `json:"InstalledVersion"`
				FixedVersion    string `json:"FixedVersion"`
				Title           string `json:"Title"`
				Description     string `json:"Description"`
				Severity        string `json:"Severity"`
				CVSS            map[string]struct {
					V3Score float64 `json:"V3Score"`
				} `json:"CVSS"`
				References      []string `json:"References"`
				PublishedDate   string   `json:"PublishedDate"`
				LastModifiedDate string  `json:"LastModifiedDate"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(output, &trivyResult); err != nil {
		return nil, fmt.Errorf("failed to parse trivy output: %w", err)
	}

	// Convert to our format
	var vulnerabilities []*Vulnerability
	for _, result := range trivyResult.Results {
		for _, vuln := range result.Vulnerabilities {
			// Parse published date
			var publishedDate time.Time
			if vuln.PublishedDate != "" {
				if parsed, err := time.Parse(time.RFC3339, vuln.PublishedDate); err == nil {
					publishedDate = parsed
				}
			}

			// Parse modified date
			var modifiedDate time.Time
			if vuln.LastModifiedDate != "" {
				if parsed, err := time.Parse(time.RFC3339, vuln.LastModifiedDate); err == nil {
					modifiedDate = parsed
				}
			}

			// Get CVSS score
			var score float64
			for _, cvss := range vuln.CVSS {
				if cvss.V3Score > 0 {
					score = cvss.V3Score
					break
				}
			}

			vulnerability := &Vulnerability{
				ID:               vuln.VulnerabilityID,
				CVE:              vuln.VulnerabilityID,
				Title:            vuln.Title,
				Description:      vuln.Description,
				Severity:         vuln.Severity,
				Score:            score,
				ScoringSystem:    "CVSS",
				Package:          vuln.PkgName,
				InstalledVersion: vuln.InstalledVersion,
				FixedVersion:     vuln.FixedVersion,
				References:       vuln.References,
				PublishedDate:    publishedDate,
				ModifiedDate:     modifiedDate,
				DiscoveredDate:   time.Now(),
			}

			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}

// scanContainerConfiguration scans container configuration for security issues
func (t *TrivySecurityScanner) scanContainerConfiguration(ctx context.Context, containerJSON *types.ContainerJSON) ([]*ConfigurationIssue, error) {
	var issues []*ConfigurationIssue

	// Check if running as root
	if containerJSON.Config.User == "" || containerJSON.Config.User == "root" || containerJSON.Config.User == "0" {
		issues = append(issues, &ConfigurationIssue{
			ID:               "config-root-user",
			Title:            "Container running as root",
			Description:      "Container is configured to run as root user, which increases security risk",
			Severity:         "high",
			Category:         "user_security",
			ConfigKey:        "User",
			CurrentValue:     containerJSON.Config.User,
			RecommendedValue: "non-root user",
			Solution:         "Configure container to run with a non-root user",
			Impact:           "Reduces privilege escalation risks",
		})
	}

	// Check for privileged mode
	if containerJSON.HostConfig.Privileged {
		issues = append(issues, &ConfigurationIssue{
			ID:               "config-privileged-mode",
			Title:            "Container running in privileged mode",
			Description:      "Container is running in privileged mode, granting extensive system access",
			Severity:         "critical",
			Category:         "privilege_escalation",
			ConfigKey:        "Privileged",
			CurrentValue:     true,
			RecommendedValue: false,
			Solution:         "Remove privileged mode and use specific capabilities instead",
			Impact:           "Reduces container escape risks",
		})
	}

	// Check for host network mode
	if string(containerJSON.HostConfig.NetworkMode) == "host" {
		issues = append(issues, &ConfigurationIssue{
			ID:               "config-host-network",
			Title:            "Container using host network",
			Description:      "Container is using host network mode, bypassing network isolation",
			Severity:         "high",
			Category:         "network_security",
			ConfigKey:        "NetworkMode",
			CurrentValue:     string(containerJSON.HostConfig.NetworkMode),
			RecommendedValue: "bridge or custom network",
			Solution:         "Use bridge or custom networks for better isolation",
			Impact:           "Improves network isolation and security",
		})
	}

	// Check for excessive capabilities
	if len(containerJSON.HostConfig.CapAdd) > 0 {
		for _, cap := range containerJSON.HostConfig.CapAdd {
			if cap == "ALL" || cap == "SYS_ADMIN" || cap == "NET_ADMIN" {
				issues = append(issues, &ConfigurationIssue{
					ID:               fmt.Sprintf("config-dangerous-cap-%s", cap),
					Title:            fmt.Sprintf("Dangerous capability: %s", cap),
					Description:      fmt.Sprintf("Container has dangerous capability %s", cap),
					Severity:         "high",
					Category:         "capabilities",
					ConfigKey:        "CapAdd",
					CurrentValue:     cap,
					RecommendedValue: "minimal required capabilities",
					Solution:         "Remove dangerous capabilities and use only what's necessary",
					Impact:           "Reduces privilege escalation risks",
				})
			}
		}
	}

	return issues, nil
}

// scanContainerSecrets scans container for exposed secrets
func (t *TrivySecurityScanner) scanContainerSecrets(ctx context.Context, containerID string) ([]*SecretIssue, error) {
	// This is a simplified implementation
	// In production, you would use Trivy's secret scanning capabilities or other secret detection tools

	var secrets []*SecretIssue

	// Check environment variables for potential secrets
	containerJSON, err := t.dockerClient.GetContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	for _, env := range containerJSON.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToUpper(parts[0])
		value := parts[1]

		// Check for common secret patterns
		if t.containsSecretPattern(key, value) {
			secrets = append(secrets, &SecretIssue{
				ID:             fmt.Sprintf("secret-env-%s", key),
				Type:           "environment_variable",
				Description:    fmt.Sprintf("Potential secret in environment variable: %s", key),
				Severity:       "medium",
				Context:        env,
				Pattern:        key,
				Recommendation: "Use secret management systems instead of environment variables",
			})
		}
	}

	return secrets, nil
}

// containsSecretPattern checks if an environment variable might contain secrets
func (t *TrivySecurityScanner) containsSecretPattern(key, value string) bool {
	secretPatterns := []string{
		"PASSWORD", "PASSWD", "SECRET", "TOKEN", "KEY", "CREDENTIAL",
		"AUTH", "API_KEY", "PRIVATE", "CERT", "PEM",
	}

	for _, pattern := range secretPatterns {
		if strings.Contains(key, pattern) && len(value) > 8 {
			return true
		}
	}

	return false
}

// calculateVulnerabilityStats calculates statistics from vulnerabilities
func (t *TrivySecurityScanner) calculateVulnerabilityStats(vulnerabilities []*Vulnerability) *VulnerabilityStats {
	stats := &VulnerabilityStats{}

	for _, vuln := range vulnerabilities {
		stats.Total++

		switch strings.ToUpper(vuln.Severity) {
		case "CRITICAL":
			stats.Critical++
		case "HIGH":
			stats.High++
		case "MEDIUM":
			stats.Medium++
		case "LOW":
			stats.Low++
		default:
			stats.Unknown++
		}

		if vuln.FixedVersion != "" {
			stats.Fixable++
		} else {
			stats.Unfixable++
		}
	}

	return stats
}

// calculateRiskScore calculates overall risk score
func (t *TrivySecurityScanner) calculateRiskScore(stats *VulnerabilityStats, configIssues []*ConfigurationIssue, secretIssues []*SecretIssue) float64 {
	var score float64

	// Vulnerability score (0-60 points)
	score += float64(stats.Critical) * 10
	score += float64(stats.High) * 5
	score += float64(stats.Medium) * 2
	score += float64(stats.Low) * 0.5

	// Configuration issues score (0-30 points)
	for _, issue := range configIssues {
		switch issue.Severity {
		case "critical":
			score += 10
		case "high":
			score += 5
		case "medium":
			score += 2
		case "low":
			score += 1
		}
	}

	// Secret issues score (0-10 points)
	for _, secret := range secretIssues {
		switch secret.Severity {
		case "critical":
			score += 5
		case "high":
			score += 3
		case "medium":
			score += 2
		case "low":
			score += 1
		}
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// determineSecurityGrade determines security grade based on risk score
func (t *TrivySecurityScanner) determineSecurityGrade(riskScore float64) string {
	if riskScore >= 80 {
		return "F"
	} else if riskScore >= 60 {
		return "D"
	} else if riskScore >= 40 {
		return "C"
	} else if riskScore >= 20 {
		return "B"
	}
	return "A"
}

// getTrivyVersion gets the Trivy version
func (t *TrivySecurityScanner) getTrivyVersion() string {
	cmd := exec.Command("trivy", "version")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Version:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])
			}
		}
	}

	return "unknown"
}

// getDatabaseVersion gets the vulnerability database version
func (t *TrivySecurityScanner) getDatabaseVersion() string {
	// This would typically check the Trivy DB version
	return "latest"
}

// getDatabaseUpdatedAt gets when the database was last updated
func (t *TrivySecurityScanner) getDatabaseUpdatedAt() time.Time {
	// This would typically check the Trivy DB timestamp
	return time.Now().AddDate(0, 0, -1) // Assume updated yesterday
}

// getImagePackages gets packages installed in the image
func (t *TrivySecurityScanner) getImagePackages(ctx context.Context, imageID string) ([]*PackageInfo, error) {
	// This would use Trivy to get package information
	// For now, return empty slice
	return []*PackageInfo{}, nil
}

// generateConfigurationRecommendations generates recommendations based on configuration issues
func (t *TrivySecurityScanner) generateConfigurationRecommendations(issues []*ConfigurationIssue) []*SecurityRecommendation {
	var recommendations []*SecurityRecommendation

	for _, issue := range issues {
		var priority SecuritySeverity
		switch issue.Severity {
		case "critical":
			priority = SeverityCritical
		case "high":
			priority = SeverityHigh
		case "medium":
			priority = SeverityMedium
		default:
			priority = SeverityLow
		}

		recommendation := &SecurityRecommendation{
			ID:          fmt.Sprintf("rec-%s", issue.ID),
			Title:       fmt.Sprintf("Fix %s", issue.Title),
			Description: issue.Solution,
			Priority:    priority,
			Actions:     []string{issue.Solution},
		}

		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// calculateComplianceScore calculates compliance score based on issues
func (t *TrivySecurityScanner) calculateComplianceScore(issues []*ConfigurationIssue) float64 {
	if len(issues) == 0 {
		return 100.0
	}

	totalPenalty := 0.0
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			totalPenalty += 25
		case "high":
			totalPenalty += 15
		case "medium":
			totalPenalty += 10
		case "low":
			totalPenalty += 5
		}
	}

	score := 100.0 - totalPenalty
	if score < 0 {
		score = 0
	}

	return score
}

// performComplianceChecks performs compliance checks for a specific standard
func (t *TrivySecurityScanner) performComplianceChecks(containerJSON *types.ContainerJSON, standard string) []*ComplianceCheck {
	var checks []*ComplianceCheck

	// This is a simplified implementation
	// In production, you would implement specific compliance frameworks

	// Check for non-root user
	check := &ComplianceCheck{
		ID:          "compliance-non-root",
		Name:        "Container should not run as root",
		Description: "Containers should run with non-root user to minimize security risks",
		Standard:    ComplianceStandard(standard),
		Category:    "User Security",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		Evidence:    make(map[string]interface{}),
	}

	if containerJSON.Config.User == "" || containerJSON.Config.User == "root" || containerJSON.Config.User == "0" {
		check.Status = ComplianceStatusFail
		check.Details = "Container is configured to run as root"
		check.Evidence["user"] = containerJSON.Config.User
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container runs with non-root user"
		check.Evidence["user"] = containerJSON.Config.User
	}

	checks = append(checks, check)

	return checks
}

// calculateOverallComplianceScore calculates overall compliance score
func (t *TrivySecurityScanner) calculateOverallComplianceScore(checks []*ComplianceCheck) float64 {
	if len(checks) == 0 {
		return 100.0
	}

	passed := 0
	for _, check := range checks {
		if check.Status == ComplianceStatusPass {
			passed++
		}
	}

	return float64(passed) / float64(len(checks)) * 100.0
}

// determineComplianceGrade determines compliance grade
func (t *TrivySecurityScanner) determineComplianceGrade(score float64) string {
	if score >= 90 {
		return "A"
	} else if score >= 80 {
		return "B"
	} else if score >= 70 {
		return "C"
	} else if score >= 60 {
		return "D"
	}
	return "F"
}