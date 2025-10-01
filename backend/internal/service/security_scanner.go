package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SecurityScannerService manages Docker image security scanning and vulnerability detection
type SecurityScannerService struct {
	db                  *gorm.DB
	dockerClient        DockerServiceInterface
	vulnerabilityDB     VulnerabilityDBInterface
	scanner             SecurityScannerInterface
	logger              *logrus.Logger
	scanners            map[string]Scanner
	mutex               sync.RWMutex
	scanHistory         map[string]*ScanResult
	alertThresholds     *SecurityAlertThresholds
}

// Scanner interface defines security scanner implementations
type Scanner interface {
	ScanImage(ctx context.Context, imageID, imageName string) (*ScanResult, error)
	GetScannerName() string
	GetScannerVersion() string
}

// SecurityAlertThresholds defines thresholds for security alerts
type SecurityAlertThresholds struct {
	CriticalVulns int `json:"criticalVulns"`
	HighVulns     int `json:"highVulns"`
	MediumVulns   int `json:"mediumVulns"`
	LowVulns      int `json:"lowVulns"`
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	CVE              string    `json:"cve" gorm:"index"`
	Severity         string    `json:"severity"`
	Score            float64   `json:"score"`
	Vector           string    `json:"vector"`
	Description      string    `json:"description"`
	Package          string    `json:"package"`
	InstalledVersion string    `json:"installedVersion"`
	FixedVersion     string    `json:"fixedVersion"`
	ContainerID      string    `json:"containerId" gorm:"index"`
	ContainerName    string    `json:"containerName"`
	ImageID          string    `json:"imageId" gorm:"index"`
	ImageName        string    `json:"imageName"`
	Layer            string    `json:"layer"`
	ScanID           string    `json:"scanId" gorm:"index"`
	Status           string    `json:"status"` // new, acknowledged, fixed, ignored
	FirstDetected    time.Time `json:"firstDetected"`
	LastSeen         time.Time `json:"lastSeen"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// SecurityScan represents a security scan record
type SecurityScan struct {
	ID               uint                    `json:"id" gorm:"primaryKey"`
	ScanID           string                  `json:"scanId" gorm:"uniqueIndex"`
	ContainerID      string                  `json:"containerId" gorm:"index"`
	ContainerName    string                  `json:"containerName"`
	ImageID          string                  `json:"imageId" gorm:"index"`
	ImageName        string                  `json:"imageName"`
	Scanner          string                  `json:"scanner"`
	ScannerVersion   string                  `json:"scannerVersion"`
	Status           string                  `json:"status"` // pending, running, completed, failed
	StartTime        time.Time               `json:"startTime"`
	EndTime          *time.Time              `json:"endTime,omitempty"`
	TotalVulns       int                     `json:"totalVulns"`
	CriticalVulns    int                     `json:"criticalVulns"`
	HighVulns        int                     `json:"highVulns"`
	MediumVulns      int                     `json:"mediumVulns"`
	LowVulns         int                     `json:"lowVulns"`
	SecurityScore    float64                 `json:"securityScore"`
	Vulnerabilities  []SecurityVulnerability `json:"vulnerabilities,omitempty"`
	Metadata         json.RawMessage         `json:"metadata"`
	ErrorMessage     string                  `json:"errorMessage,omitempty"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
}

// SecurityFinding represents individual security findings
type SecurityFinding struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Package     string    `json:"package,omitempty"`
	Version     string    `json:"version,omitempty"`
	FixVersion  string    `json:"fix_version,omitempty"`
	CVSS        float64   `json:"cvss,omitempty"`
	References  []string  `json:"references,omitempty"`
	FoundAt     time.Time `json:"found_at"`
}

// SecurityOverview provides security overview statistics
type SecurityOverview struct {
	TotalContainers      int                     `json:"totalContainers"`
	ScannedContainers    int                     `json:"scannedContainers"`
	VulnerableContainers int                     `json:"vulnerableContainers"`
	TotalVulnerabilities int                     `json:"totalVulnerabilities"`
	CriticalVulns        int                     `json:"criticalVulns"`
	HighVulns            int                     `json:"highVulns"`
	MediumVulns          int                     `json:"mediumVulns"`
	LowVulns             int                     `json:"lowVulns"`
	SecurityScore        float64                 `json:"securityScore"`
	LastScanTime         time.Time               `json:"lastScanTime"`
	TopVulnerabilities   []SecurityVulnerability `json:"topVulnerabilities"`
	RiskDistribution     map[string]int          `json:"riskDistribution"`
	ComplianceReport     ComplianceReport        `json:"complianceReport"`
}

// ComplianceReport represents security compliance status
type ComplianceReport struct {
	OverallScore     float64            `json:"overallScore"`
	Policies         map[string]bool    `json:"policies"`
	Recommendations  []string           `json:"recommendations"`
	LastAssessment   time.Time          `json:"lastAssessment"`
}

// ScanSecurityAlert represents a security alert from scanning
type ScanSecurityAlert struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	AlertType     string    `json:"alertType"` // vulnerability, policy, compliance
	Severity      string    `json:"severity"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	ContainerID   string    `json:"containerId,omitempty"`
	ContainerName string    `json:"containerName,omitempty"`
	VulnID        string    `json:"vulnId,omitempty"`
	Status        string    `json:"status"` // new, acknowledged, resolved
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

// NewSecurityScannerService creates a new security scanner service
func NewSecurityScannerService(
	db *gorm.DB,
	dockerClient DockerServiceInterface,
	vulnDB VulnerabilityDBInterface,
	scanner SecurityScannerInterface,
	logger *logrus.Logger,
) *SecurityScannerService {
	return &SecurityScannerService{
		db:              db,
		dockerClient:    dockerClient,
		vulnerabilityDB: vulnDB,
		scanner:         scanner,
		logger:          logger,
		scanners:        make(map[string]Scanner),
		scanHistory:     make(map[string]*ScanResult),
		alertThresholds: &SecurityAlertThresholds{
			CriticalVulns: 1,
			HighVulns:     5,
			MediumVulns:   10,
			LowVulns:      50,
		},
	}
}

// Initialize sets up the security scanner service
func (sss *SecurityScannerService) Initialize(ctx context.Context) error {
	// Migrate database tables
	if err := sss.db.AutoMigrate(&SecurityVulnerability{}, &SecurityScan{}, &SecurityAlert{}); err != nil {
		return fmt.Errorf("failed to migrate security tables: %w", err)
	}

	// Initialize default scanners
	if err := sss.initializeDefaultScanners(); err != nil {
		sss.logger.WithError(err).Warn("Failed to initialize default scanners")
	}

	return nil
}

// RegisterScanner registers a new security scanner
func (sss *SecurityScannerService) RegisterScanner(scanner Scanner) {
	sss.mutex.Lock()
	defer sss.mutex.Unlock()
	sss.scanners[scanner.GetScannerName()] = scanner
	sss.logger.WithFields(logrus.Fields{
		"scanner": scanner.GetScannerName(),
		"version": scanner.GetScannerVersion(),
	}).Info("Security scanner registered")
}

// ScanAllContainers performs security scans on all running containers
func (sss *SecurityScannerService) ScanAllContainers(ctx context.Context) error {
	sss.logger.Info("Starting security scan for all containers")

	containers, err := sss.dockerClient.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3) // Limit concurrent scans

	for _, container := range containers {
		wg.Add(1)
		go func(containerID, imageID, imageName, containerName string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := sss.ScanContainer(ctx, containerID, imageID, imageName, containerName); err != nil {
				sss.logger.WithError(err).WithFields(logrus.Fields{
					"containerId":   containerID,
					"containerName": containerName,
				}).Error("Failed to scan container")
			}
		}(container.ID, container.ImageID, container.Image, container.Name)
	}

	wg.Wait()
	sss.logger.Info("Completed security scan for all containers")

	return nil
}

// ScanContainer performs a security scan on a specific container
func (sss *SecurityScannerService) ScanContainer(ctx context.Context, containerID, imageID, imageName, containerName string) error {
	scanID := fmt.Sprintf("scan_%s_%d", containerID[:12], time.Now().Unix())

	sss.logger.WithFields(logrus.Fields{
		"scanId":        scanID,
		"containerId":   containerID,
		"containerName": containerName,
	}).Info("Starting container security scan")

	// Create scan record
	scan := &SecurityScan{
		ScanID:        scanID,
		ContainerID:   containerID,
		ContainerName: containerName,
		ImageID:       imageID,
		ImageName:     imageName,
		Status:        "running",
		StartTime:     time.Now(),
	}

	if err := sss.db.Create(scan).Error; err != nil {
		return fmt.Errorf("failed to create scan record: %w", err)
	}

	// Perform scan using available scanners
	results := make([]*ScanResult, 0)
	sss.mutex.RLock()
	scanners := make([]Scanner, 0, len(sss.scanners))
	for _, scanner := range sss.scanners {
		scanners = append(scanners, scanner)
	}
	sss.mutex.RUnlock()

	if len(scanners) == 0 {
		// Use built-in basic scanner
		result, err := sss.performBasicScan(ctx, scanID, containerID, imageID, imageName, containerName)
		if err != nil {
			sss.markScanFailed(scan, err)
			return err
		}
		results = append(results, result)
	} else {
		// Use registered scanners
		for _, scanner := range scanners {
			result, err := scanner.ScanImage(ctx, imageID, imageName)
			if err != nil {
				sss.logger.WithError(err).WithField("scanner", scanner.GetScannerName()).Warn("Scanner failed")
				continue
			}
			result.ScanID = scanID
			result.ContainerID = containerID
			results = append(results, result)
		}
	}

	if len(results) == 0 {
		err := fmt.Errorf("all scanners failed")
		sss.markScanFailed(scan, err)
		return err
	}

	// Aggregate results
	finalResult := sss.aggregateScanResults(results)

	// Store results
	if err := sss.storeScanResults(scan, finalResult); err != nil {
		sss.logger.WithError(err).Error("Failed to store scan results")
	}

	// Generate alerts if needed
	if err := sss.generateSecurityAlerts(ctx, finalResult); err != nil {
		sss.logger.WithError(err).Error("Failed to generate security alerts")
	}

	// Update scan status
	endTime := time.Now()
	scan.EndTime = &endTime
	scan.Status = "completed"
	if finalResult.VulnerabilityStats != nil {
		scan.TotalVulns = finalResult.VulnerabilityStats.Total
		scan.CriticalVulns = finalResult.VulnerabilityStats.Critical
		scan.HighVulns = finalResult.VulnerabilityStats.High
		scan.MediumVulns = finalResult.VulnerabilityStats.Medium
		scan.LowVulns = finalResult.VulnerabilityStats.Low
	}
	scan.SecurityScore = finalResult.RiskScore

	if err := sss.db.Save(scan).Error; err != nil {
		sss.logger.WithError(err).Error("Failed to update scan status")
	}

	sss.logger.WithFields(logrus.Fields{
		"scanId":        scanID,
		"containerId":   containerID,
		"totalVulns":    func() int { if finalResult.VulnerabilityStats != nil { return finalResult.VulnerabilityStats.Total }; return 0 }(),
		"securityScore": finalResult.RiskScore,
	}).Info("Container security scan completed")

	return nil
}

// performBasicScan performs a basic security scan using built-in capabilities
func (sss *SecurityScannerService) performBasicScan(ctx context.Context, scanID, containerID, imageID, imageName, containerName string) (*ScanResult, error) {
	result := &ScanResult{
		ScanID:         scanID,
		ContainerID:    containerID,
		ImageID:        imageID,
		ImageName:      imageName,
		Scanner:        "basic",
		ScannerVersion: "1.0.0",
		ScanStartTime:  time.Now(),
		ScanEndTime:    time.Now(),
		ScanDuration:   0,
		Vulnerabilities: []*Vulnerability{},
		VulnerabilityStats: &VulnerabilityStats{
			Total:    0,
			Critical: 0,
			High:     0,
			Medium:   0,
			Low:      0,
			Unknown:  0,
			Fixable:  0,
			Unfixable: 0,
		},
		ConfigurationIssues: []*ConfigurationIssue{},
		SecretIssues:       []*SecretIssue{},
		RiskScore:          0.0,
		SecurityGrade:      "unknown",
		Metadata:           make(map[string]interface{}),
		Success:            true,
	}

	// Get image history and layers
	layers, err := sss.dockerClient.GetImageLayers(ctx, imageID)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to get image layers: %v", err)
		result.ScanEndTime = time.Now()
		result.ScanDuration = time.Since(result.ScanStartTime)
		result.Success = false
		return result, nil
	}

	result.Metadata["layers"] = len(layers)

	// Check for known vulnerable packages
	vulns, err := sss.scanForKnownVulnerabilities(ctx, imageID, imageName, layers)
	if err != nil {
		sss.logger.WithError(err).Warn("Failed to scan for known vulnerabilities")
	}

	// Convert SecurityVulnerability to Vulnerability
	var vulnerabilities []*Vulnerability
	for _, vuln := range vulns {
		vulnerabilities = append(vulnerabilities, &Vulnerability{
			ID:               vuln.CVE,
			CVE:              vuln.CVE,
			Title:            fmt.Sprintf("Vulnerability in %s", vuln.Package),
			Description:      vuln.Description,
			Severity:         vuln.Severity,
			Score:            vuln.Score,
			ScoringSystem:    "CVSS",
			Package:          vuln.Package,
			InstalledVersion: vuln.InstalledVersion,
			FixedVersion:     vuln.FixedVersion,
			Layer:            vuln.Layer,
			PublishedDate:    vuln.FirstDetected,
			ModifiedDate:     vuln.LastSeen,
			DiscoveredDate:   vuln.FirstDetected,
		})
	}

	result.Vulnerabilities = vulnerabilities

	// Count vulnerabilities by severity
	var critical, high, medium, low int
	for _, vuln := range vulnerabilities {
		switch strings.ToLower(vuln.Severity) {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}

	// Update vulnerability stats
	result.VulnerabilityStats.Total = len(vulnerabilities)
	result.VulnerabilityStats.Critical = critical
	result.VulnerabilityStats.High = high
	result.VulnerabilityStats.Medium = medium
	result.VulnerabilityStats.Low = low

	// Calculate security score
	result.RiskScore = sss.calculateSecurityScore(critical, high, medium, low)
	result.ScanEndTime = time.Now()
	result.ScanDuration = time.Since(result.ScanStartTime)

	return result, nil
}

// scanForKnownVulnerabilities scans for known vulnerabilities in image layers
func (sss *SecurityScannerService) scanForKnownVulnerabilities(ctx context.Context, imageID, imageName string, layers []string) ([]SecurityVulnerability, error) {
	var vulnerabilities []SecurityVulnerability

	// Get package information from image
	packages, err := sss.dockerClient.GetImagePackages(ctx, imageID)
	if err != nil {
		return vulnerabilities, fmt.Errorf("failed to get image packages: %w", err)
	}

	// Query vulnerability database for each package
	for _, pkg := range packages {
		vulns, err := sss.vulnerabilityDB.QueryVulnerabilities(ctx, pkg.Name, pkg.Version)
		if err != nil {
			sss.logger.WithError(err).WithField("package", pkg.Name).Debug("Failed to query vulnerabilities")
			continue
		}

		for _, vuln := range vulns {
			vulnerability := SecurityVulnerability{
				CVE:              vuln.CVE,
				Severity:         vuln.Severity,
				Score:            vuln.Score,
				Vector:           vuln.Vector,
				Description:      vuln.Description,
				Package:          pkg.Name,
				InstalledVersion: pkg.Version,
				FixedVersion:     vuln.FixedVersion,
				ImageID:          imageID,
				ImageName:        imageName,
				Status:           "new",
				FirstDetected:    time.Now(),
				LastSeen:         time.Now(),
			}
			vulnerabilities = append(vulnerabilities, vulnerability)
		}
	}

	return vulnerabilities, nil
}

// aggregateScanResults aggregates results from multiple scanners
func (sss *SecurityScannerService) aggregateScanResults(results []*ScanResult) *ScanResult {
	if len(results) == 1 {
		return results[0]
	}

	// Use the first result as base
	aggregated := results[0]
	vulnMap := make(map[string]*Vulnerability)

	// Collect all unique vulnerabilities
	for _, result := range results {
		for _, vuln := range result.Vulnerabilities {
			key := fmt.Sprintf("%s_%s_%s", vuln.CVE, vuln.Package, vuln.InstalledVersion)
			if existing, exists := vulnMap[key]; exists {
				// Keep the highest severity
				if sss.severityWeight(vuln.Severity) > sss.severityWeight(existing.Severity) {
					vulnMap[key] = vuln
				}
			} else {
				vulnMap[key] = vuln
			}
		}
	}

	// Convert map back to slice
	aggregated.Vulnerabilities = make([]*Vulnerability, 0, len(vulnMap))
	for _, vuln := range vulnMap {
		aggregated.Vulnerabilities = append(aggregated.Vulnerabilities, vuln)
	}

	// Recalculate counts and update vulnerability stats
	var critical, high, medium, low int
	for _, vuln := range aggregated.Vulnerabilities {
		switch strings.ToLower(vuln.Severity) {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}

	// Update vulnerability stats
	if aggregated.VulnerabilityStats == nil {
		aggregated.VulnerabilityStats = &VulnerabilityStats{}
	}
	aggregated.VulnerabilityStats.Total = len(aggregated.Vulnerabilities)
	aggregated.VulnerabilityStats.Critical = critical
	aggregated.VulnerabilityStats.High = high
	aggregated.VulnerabilityStats.Medium = medium
	aggregated.VulnerabilityStats.Low = low

	// Recalculate security score
	aggregated.RiskScore = sss.calculateSecurityScore(critical, high, medium, low)

	return aggregated
}

// storeScanResults stores scan results in the database
func (sss *SecurityScannerService) storeScanResults(scan *SecurityScan, result *ScanResult) error {
	tx := sss.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Store vulnerabilities - convert to SecurityVulnerability for database storage
	for _, vuln := range result.Vulnerabilities {
		dbVuln := SecurityVulnerability{
			CVE:              vuln.CVE,
			Severity:         vuln.Severity,
			Score:            vuln.Score,
			Vector:           vuln.ScoringSystem,
			Description:      vuln.Description,
			Package:          vuln.Package,
			InstalledVersion: vuln.InstalledVersion,
			FixedVersion:     vuln.FixedVersion,
			ContainerID:      result.ContainerID,
			ContainerName:    scan.ContainerName,
			ImageID:          result.ImageID,
			ImageName:        result.ImageName,
			Layer:            vuln.Layer,
			ScanID:           result.ScanID,
			Status:           "new",
			FirstDetected:    vuln.PublishedDate,
			LastSeen:         vuln.ModifiedDate,
		}

		if err := tx.Create(&dbVuln).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to store vulnerability: %w", err)
		}
	}

	// Store metadata
	if metadata, err := json.Marshal(result.Metadata); err == nil {
		scan.Metadata = metadata
	}

	if err := tx.Save(scan).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to save scan: %w", err)
	}

	return tx.Commit().Error
}

// generateSecurityAlerts generates security alerts based on scan results
func (sss *SecurityScannerService) generateSecurityAlerts(ctx context.Context, result *ScanResult) error {
	// Check if critical vulnerabilities exceed threshold
	criticalCount := 0
	if result.VulnerabilityStats != nil {
		criticalCount = result.VulnerabilityStats.Critical
	}

	if criticalCount >= sss.alertThresholds.CriticalVulns {
		alert := &SecurityAlert{
			AlertType:     "vulnerability",
			Severity:      "critical",
			Title:         fmt.Sprintf("Critical vulnerabilities detected in %s", result.ImageName),
			Description:   fmt.Sprintf("Found %d critical vulnerabilities", criticalCount),
			ContainerID:   &result.ContainerID,
			Status:        "new",
		}

		if err := sss.db.Create(alert).Error; err != nil {
			return fmt.Errorf("failed to create alert: %w", err)
		}
	}

	// Additional alert logic can be added here

	return nil
}

// GetSecurityOverview returns security overview statistics
func (sss *SecurityScannerService) GetSecurityOverview(ctx context.Context) (*SecurityOverview, error) {
	overview := &SecurityOverview{
		RiskDistribution: make(map[string]int),
	}

	// Get total containers
	containers, err := sss.dockerClient.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}
	overview.TotalContainers = len(containers)

	// Get scanned containers count
	var scannedCount int64
	if err := sss.db.Model(&SecurityScan{}).Where("status = ?", "completed").
		Distinct("container_id").Count(&scannedCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count scanned containers: %w", err)
	}
	overview.ScannedContainers = int(scannedCount)

	// Get vulnerability statistics
	var vulnStats struct {
		Total    int64
		Critical int64
		High     int64
		Medium   int64
		Low      int64
	}

	if err := sss.db.Model(&SecurityVulnerability{}).Count(&vulnStats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to count total vulnerabilities: %w", err)
	}
	overview.TotalVulnerabilities = int(vulnStats.Total)

	// Count by severity
	severities := []string{"critical", "high", "medium", "low"}
	for _, severity := range severities {
		var count int64
		sss.db.Model(&SecurityVulnerability{}).Where("severity = ?", severity).Count(&count)

		switch severity {
		case "critical":
			overview.CriticalVulns = int(count)
		case "high":
			overview.HighVulns = int(count)
		case "medium":
			overview.MediumVulns = int(count)
		case "low":
			overview.LowVulns = int(count)
		}

		overview.RiskDistribution[severity] = int(count)
	}

	// Calculate overall security score
	overview.SecurityScore = sss.calculateSecurityScore(
		overview.CriticalVulns,
		overview.HighVulns,
		overview.MediumVulns,
		overview.LowVulns,
	)

	// Get vulnerable containers count
	var vulnerableCount int64
	sss.db.Model(&SecurityVulnerability{}).Distinct("container_id").Count(&vulnerableCount)
	overview.VulnerableContainers = int(vulnerableCount)

	// Get last scan time
	var lastScan SecurityScan
	if err := sss.db.Order("created_at DESC").First(&lastScan).Error; err == nil {
		overview.LastScanTime = lastScan.CreatedAt
	}

	// Get top vulnerabilities
	var topVulns []SecurityVulnerability
	if err := sss.db.Order("score DESC").Limit(5).Find(&topVulns).Error; err == nil {
		overview.TopVulnerabilities = topVulns
	}

	// Get compliance report
	overview.ComplianceReport = sss.assessComplianceStatus(overview)

	return overview, nil
}

// assessComplianceStatus assesses security compliance status
func (sss *SecurityScannerService) assessComplianceStatus(overview *SecurityOverview) ComplianceReport {
	status := ComplianceReport{
		Policies:        make(map[string]bool),
		Recommendations: make([]string, 0),
		LastAssessment:  time.Now(),
	}

	// Basic compliance checks
	status.Policies["no_critical_vulns"] = overview.CriticalVulns == 0
	status.Policies["limited_high_vulns"] = overview.HighVulns <= 5
	status.Policies["regular_scanning"] = time.Since(overview.LastScanTime) <= 7*24*time.Hour

	// Calculate overall compliance score
	totalPolicies := len(status.Policies)
	passedPolicies := 0
	for _, passed := range status.Policies {
		if passed {
			passedPolicies++
		}
	}
	status.OverallScore = (float64(passedPolicies) / float64(totalPolicies)) * 100

	// Generate recommendations
	if overview.CriticalVulns > 0 {
		status.Recommendations = append(status.Recommendations, "Immediately address critical vulnerabilities")
	}
	if overview.HighVulns > 5 {
		status.Recommendations = append(status.Recommendations, "Reduce high-severity vulnerabilities")
	}
	if time.Since(overview.LastScanTime) > 7*24*time.Hour {
		status.Recommendations = append(status.Recommendations, "Perform regular security scans")
	}

	return status
}

// Helper functions
func (sss *SecurityScannerService) initializeDefaultScanners() error {
	// Initialize built-in scanners here
	// This could include integration with tools like Trivy, Clair, etc.
	return nil
}

func (sss *SecurityScannerService) markScanFailed(scan *SecurityScan, err error) {
	scan.Status = "failed"
	scan.ErrorMessage = err.Error()
	endTime := time.Now()
	scan.EndTime = &endTime

	if dbErr := sss.db.Save(scan).Error; dbErr != nil {
		sss.logger.WithError(dbErr).Error("Failed to mark scan as failed")
	}
}

func (sss *SecurityScannerService) calculateSecurityScore(critical, high, medium, low int) float64 {
	// Weighted scoring: critical=10, high=5, medium=2, low=1
	totalWeightedVulns := float64(critical*10 + high*5 + medium*2 + low*1)

	if totalWeightedVulns == 0 {
		return 100.0 // Perfect score
	}

	// Normalize to 0-100 scale (higher is better)
	// Base score of 100, subtract weighted vulnerabilities
	score := 100.0 - totalWeightedVulns
	if score < 0 {
		score = 0
	}

	return score
}

func (sss *SecurityScannerService) severityWeight(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}