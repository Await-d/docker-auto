package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"docker-auto/pkg/docker"

	"github.com/sirupsen/logrus"
)

// NewRiskAnalyzer creates a new risk analyzer
func NewRiskAnalyzer(logger *logrus.Entry) *RiskAnalyzer {
	analyzer := &RiskAnalyzer{
		riskModels: make(map[string]*RiskModel),
		logger:     logger.WithField("subcomponent", "risk_analyzer"),
	}

	// Initialize default risk model
	analyzer.initializeDefaultRiskModel()

	return analyzer
}

// AssessRisk performs a comprehensive risk assessment of a container
func (ra *RiskAnalyzer) AssessRisk(ctx context.Context, container *docker.ContainerInfo) *RiskAssessment {
	model := ra.riskModels["default"]
	if model == nil {
		ra.logger.Error("Default risk model not found")
		return &RiskAssessment{
			ID:               fmt.Sprintf("risk_%s_%d", container.ID, time.Now().UnixNano()),
			Timestamp:        time.Now(),
			ContainerID:      container.ID,
			ContainerName:    container.Name,
			OverallRiskScore: 50.0, // Default medium risk
			RiskLevel:        RiskLevelMedium,
			Factors:          []RiskFactor{},
			Recommendations:  []SecurityRecommendation{},
			LastUpdated:      time.Now(),
		}
	}

	// Calculate risk factors
	var factors []RiskFactor
	totalWeightedScore := 0.0
	totalWeight := 0.0

	for _, factorDef := range model.Factors {
		score := factorDef.Evaluator(ctx, container)
		factor := RiskFactor{
			Name:        factorDef.Name,
			Description: factorDef.Description,
			Score:       score,
			Weight:      factorDef.Weight,
			Category:    factorDef.Category,
		}
		factors = append(factors, factor)
		totalWeightedScore += score * factorDef.Weight
		totalWeight += factorDef.Weight
	}

	// Calculate overall risk score
	overallScore := 0.0
	if totalWeight > 0 {
		overallScore = totalWeightedScore / totalWeight
	}

	// Determine risk level
	riskLevel := ra.determineRiskLevel(overallScore, model.Thresholds)

	// Generate recommendations
	recommendations := ra.generateRecommendations(factors, container)

	assessment := &RiskAssessment{
		ID:               fmt.Sprintf("risk_%s_%d", container.ID, time.Now().UnixNano()),
		Timestamp:        time.Now(),
		ContainerID:      container.ID,
		ContainerName:    container.Name,
		OverallRiskScore: overallScore,
		RiskLevel:        riskLevel,
		Factors:          factors,
		Recommendations:  recommendations,
		LastUpdated:      time.Now(),
	}

	ra.logger.WithFields(logrus.Fields{
		"container_id":  container.ID,
		"risk_score":    overallScore,
		"risk_level":    riskLevel,
		"factors_count": len(factors),
	}).Info("Risk assessment completed")

	return assessment
}

// initializeDefaultRiskModel initializes the default risk assessment model
func (ra *RiskAnalyzer) initializeDefaultRiskModel() {
	model := &RiskModel{
		Name:    "Default Container Risk Model",
		Version: "1.0",
		Factors: []RiskFactorDefinition{
			{
				Name:        "Privileged Execution",
				Description: "Risk associated with privileged container execution",
				Category:    "Privilege Escalation",
				Weight:      0.25,
				Evaluator:   ra.evaluatePrivilegedRisk,
			},
			{
				Name:        "Root User Execution",
				Description: "Risk of running as root user",
				Category:    "User Security",
				Weight:      0.20,
				Evaluator:   ra.evaluateRootUserRisk,
			},
			{
				Name:        "Capability Exposure",
				Description: "Risk from excessive Linux capabilities",
				Category:    "Capabilities",
				Weight:      0.15,
				Evaluator:   ra.evaluateCapabilityRisk,
			},
			{
				Name:        "Network Exposure",
				Description: "Risk from network configuration",
				Category:    "Network Security",
				Weight:      0.15,
				Evaluator:   ra.evaluateNetworkRisk,
			},
			{
				Name:        "Mount Security",
				Description: "Risk from host filesystem mounts",
				Category:    "Mount Security",
				Weight:      0.10,
				Evaluator:   ra.evaluateMountRisk,
			},
			{
				Name:        "Resource Limits",
				Description: "Risk from lack of resource constraints",
				Category:    "Resource Management",
				Weight:      0.10,
				Evaluator:   ra.evaluateResourceRisk,
			},
			{
				Name:        "Image Security",
				Description: "Risk associated with container image",
				Category:    "Image Security",
				Weight:      0.05,
				Evaluator:   ra.evaluateImageRisk,
			},
		},
		Weights: map[string]float64{
			"privilege":    0.25,
			"user":         0.20,
			"capabilities": 0.15,
			"network":      0.15,
			"mounts":       0.10,
			"resources":    0.10,
			"image":        0.05,
		},
		Thresholds: map[RiskLevel]float64{
			RiskLevelLow:      25.0,
			RiskLevelMedium:   50.0,
			RiskLevelHigh:     75.0,
			RiskLevelCritical: 100.0,
		},
	}

	ra.riskModels["default"] = model
}

// Risk factor evaluators

// evaluatePrivilegedRisk evaluates risk from privileged execution
func (ra *RiskAnalyzer) evaluatePrivilegedRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	if privileged, ok := container.HostConfig["Privileged"].(bool); ok && privileged {
		return 100.0 // Maximum risk for privileged containers
	}
	return 0.0
}

// evaluateRootUserRisk evaluates risk from root user execution
func (ra *RiskAnalyzer) evaluateRootUserRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	user := container.Config.User
	if user == "" || user == "root" || user == "0" {
		return 80.0 // High risk for root execution
	}

	// Check if user is numeric and low UID (potentially privileged)
	if uid, err := strconv.Atoi(user); err == nil {
		if uid < 1000 {
			return 60.0 // Medium-high risk for system users
		}
	}

	return 20.0 // Low risk for proper non-root users
}

// evaluateCapabilityRisk evaluates risk from Linux capabilities
func (ra *RiskAnalyzer) evaluateCapabilityRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	risk := 0.0

	// High-risk capabilities
	dangerousCaps := map[string]float64{
		"SYS_ADMIN":  40.0,
		"NET_ADMIN":  30.0,
		"SYS_PTRACE": 35.0,
		"SYS_MODULE": 45.0,
		"DAC_OVERRIDE": 25.0,
		"SETUID":     20.0,
		"SETGID":     20.0,
		"NET_RAW":    15.0,
	}

	// Check added capabilities
	if capAdd, ok := container.HostConfig["CapAdd"].([]interface{}); ok && capAdd != nil {
		for _, cap := range capAdd {
			if capStr, ok := cap.(string); ok {
				capUpper := strings.ToUpper(capStr)
				if capUpper == "ALL" {
					return 100.0 // Maximum risk for all capabilities
				}
				if riskValue, exists := dangerousCaps[capUpper]; exists {
					risk += riskValue
				} else {
					risk += 10.0 // Default risk for unknown capabilities
				}
			}
		}
	}

	// Reduce risk if capabilities are explicitly dropped
	if capDrop, ok := container.HostConfig["CapDrop"].([]interface{}); ok && capDrop != nil {
		droppedCount := len(capDrop)
		if droppedCount > 0 {
			risk *= (1.0 - float64(droppedCount)*0.05) // Reduce by 5% per dropped capability
		}
	}

	// Cap the risk at 100
	if risk > 100.0 {
		risk = 100.0
	}

	return risk
}

// evaluateNetworkRisk evaluates risk from network configuration
func (ra *RiskAnalyzer) evaluateNetworkRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	risk := 0.0

	// Host network mode is high risk
	if networkMode, ok := container.HostConfig["NetworkMode"].(string); ok && networkMode == "host" {
		risk += 70.0
	}

	// Publishing all ports is risky
	if publishAllPorts, ok := container.HostConfig["PublishAllPorts"].(bool); ok && publishAllPorts {
		risk += 30.0
	}

	// Check for published ports
	if portBindings, ok := container.HostConfig["PortBindings"].(map[string]interface{}); ok && portBindings != nil {
		portCount := len(portBindings)
		if portCount > 5 {
			risk += 20.0 // Many published ports increase risk
		} else if portCount > 0 {
			risk += 10.0 // Some published ports have moderate risk
		}
	}

	// DNS settings that could be risky
	if dnsServers, ok := container.HostConfig["DNS"].([]interface{}); ok && dnsServers != nil && len(dnsServers) > 0 {
		for _, dnsInterface := range dnsServers {
			if dns, ok := dnsInterface.(string); ok {
				// Check for potentially risky DNS servers
				if dns == "8.8.8.8" || dns == "1.1.1.1" {
					// Public DNS is generally safe
					continue
				} else if strings.HasPrefix(dns, "127.") || strings.HasPrefix(dns, "169.254.") {
					risk += 15.0 // Local/link-local DNS could be risky
				}
			}
		}
	}

	// Cap the risk at 100
	if risk > 100.0 {
		risk = 100.0
	}

	return risk
}

// evaluateMountRisk evaluates risk from filesystem mounts
func (ra *RiskAnalyzer) evaluateMountRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	risk := 0.0

	// Critical system paths that should never be mounted
	criticalPaths := map[string]float64{
		"/":              100.0,
		"/boot":          80.0,
		"/dev":           70.0,
		"/etc":           60.0,
		"/lib":           50.0,
		"/proc":          90.0,
		"/sys":           85.0,
		"/usr":           45.0,
		"/var/lib/docker": 95.0,
		"/etc/passwd":    75.0,
		"/etc/shadow":    90.0,
		"/etc/group":     60.0,
		"/run":           40.0,
		"/tmp":           20.0,
	}

	if binds, ok := container.HostConfig["Binds"].([]interface{}); ok && binds != nil {
		for _, bindInterface := range binds {
			if bind, ok := bindInterface.(string); ok {
				parts := strings.Split(bind, ":")
				if len(parts) >= 2 {
					hostPath := parts[0]

					// Check for exact matches
					if riskValue, exists := criticalPaths[hostPath]; exists {
						risk += riskValue
						continue
					}

					// Check for path prefixes
					for criticalPath, riskValue := range criticalPaths {
						if strings.HasPrefix(hostPath, criticalPath+"/") {
							risk += riskValue * 0.8 // Slightly less risk for subdirectories
							break
						}
					}

					// Check if mount is read-write (more risky)
					if len(parts) < 3 || !strings.Contains(parts[2], "ro") {
						risk += 5.0 // Additional risk for read-write mounts
					}
				}
			}
		}
	}

	// Check for volume mounts from host
	if volumeDriver, ok := container.HostConfig["VolumeDriver"].(string); ok && volumeDriver != "" {
		risk += 10.0 // External volume drivers add some risk
	}

	// Cap the risk at 100
	if risk > 100.0 {
		risk = 100.0
	}

	return risk
}

// evaluateResourceRisk evaluates risk from resource configuration
func (ra *RiskAnalyzer) evaluateResourceRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	risk := 0.0

	// Memory limits
	if memory, ok := container.HostConfig["Memory"].(float64); ok {
		if memory == 0 {
			risk += 25.0 // No memory limit is risky
		} else if memory > 8*1024*1024*1024 { // > 8GB
			risk += 10.0 // Very high memory limits have some risk
		}
	} else {
		risk += 25.0 // No memory limit configured
	}

	// CPU limits
	hasCPULimit := false
	if cpuShares, ok := container.HostConfig["CPUShares"].(float64); ok && cpuShares > 0 {
		hasCPULimit = true
	}
	if cpuPeriod, ok := container.HostConfig["CPUPeriod"].(float64); ok && cpuPeriod > 0 {
		hasCPULimit = true
	}
	if cpuQuota, ok := container.HostConfig["CPUQuota"].(float64); ok && cpuQuota > 0 {
		hasCPULimit = true
	}
	if cpusetCpus, ok := container.HostConfig["CpusetCpus"].(string); ok && cpusetCpus != "" {
		hasCPULimit = true
	}

	if !hasCPULimit {
		risk += 20.0 // No CPU limits are risky
	}

	// PID limits
	if pidsLimit, ok := container.HostConfig["PidsLimit"].(float64); !ok || pidsLimit == 0 {
		risk += 15.0 // No PID limits can lead to fork bombs
	}

	// Ulimits
	if ulimits, ok := container.HostConfig["Ulimits"].([]interface{}); !ok || ulimits == nil || len(ulimits) == 0 {
		risk += 10.0 // No ulimits set
	}

	// OOM Kill Disable
	if oomKillDisable, ok := container.HostConfig["OomKillDisable"].(bool); ok && oomKillDisable {
		risk += 20.0 // Disabling OOM killer is risky
	}

	return risk
}

// evaluateImageRisk evaluates risk from container image
func (ra *RiskAnalyzer) evaluateImageRisk(ctx context.Context, container *docker.ContainerInfo) float64 {
	risk := 0.0

	image := container.Config.Image

	// Check for latest tag usage
	if strings.HasSuffix(image, ":latest") || !strings.Contains(image, ":") {
		risk += 30.0 // Using latest tag or no tag is risky
	}

	// Check for official/verified images (lower risk)
	if strings.HasPrefix(image, "docker.io/library/") ||
	   !strings.Contains(image, "/") {
		risk -= 10.0 // Official images are safer
	}

	// Check for suspicious image names
	suspiciousPatterns := []string{
		"test", "debug", "temp", "experimental", "dev", "alpha", "beta",
	}

	imageLower := strings.ToLower(image)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(imageLower, pattern) {
			risk += 15.0
			break
		}
	}

	// Check for digest usage (good practice)
	if strings.Contains(image, "@sha256:") {
		risk -= 15.0 // Using content digest is good practice
	}

	// Ensure risk is not negative
	if risk < 0 {
		risk = 0
	}

	return risk
}

// determineRiskLevel determines the risk level based on score and thresholds
func (ra *RiskAnalyzer) determineRiskLevel(score float64, thresholds map[RiskLevel]float64) RiskLevel {
	if score >= thresholds[RiskLevelCritical] {
		return RiskLevelCritical
	} else if score >= thresholds[RiskLevelHigh] {
		return RiskLevelHigh
	} else if score >= thresholds[RiskLevelMedium] {
		return RiskLevelMedium
	}
	return RiskLevelLow
}

// generateRecommendations generates security recommendations based on risk factors
func (ra *RiskAnalyzer) generateRecommendations(factors []RiskFactor, container *docker.ContainerInfo) []SecurityRecommendation {
	var recommendations []SecurityRecommendation

	for _, factor := range factors {
		if factor.Score > 50.0 { // High-risk factors
			switch factor.Category {
			case "Privilege Escalation":
				if factor.Score >= 100.0 { // Privileged container
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "priv-001",
						Title:       "Remove Privileged Mode",
						Description: "Container is running in privileged mode, which grants excessive permissions",
						Priority:    SeverityCritical,
						Actions: []string{
							"Remove --privileged flag from container execution",
							"Use specific capabilities instead of privileged mode",
							"Review if privileged access is truly necessary",
						},
						Resources: []string{
							"https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities",
						},
					})
				}

			case "User Security":
				if factor.Score >= 60.0 { // Root user
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "user-001",
						Title:       "Use Non-Root User",
						Description: "Container is running as root user, increasing security risk",
						Priority:    SeverityHigh,
						Actions: []string{
							"Create a dedicated non-root user in the Dockerfile",
							"Use USER instruction to switch to non-root user",
							"Ensure application works with non-root permissions",
						},
						Resources: []string{
							"https://docs.docker.com/develop/dev-best-practices/#user",
						},
					})
				}

			case "Capabilities":
				if factor.Score >= 30.0 { // Dangerous capabilities
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "cap-001",
						Title:       "Restrict Linux Capabilities",
						Description: "Container has dangerous Linux capabilities that increase attack surface",
						Priority:    SeverityHigh,
						Actions: []string{
							"Drop all capabilities with --cap-drop ALL",
							"Add only required capabilities with --cap-add",
							"Review if all added capabilities are necessary",
						},
						Resources: []string{
							"https://docs.docker.com/engine/reference/run/#runtime-privilege-and-linux-capabilities",
						},
					})
				}

			case "Network Security":
				if factor.Score >= 50.0 { // Network risks
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "net-001",
						Title:       "Improve Network Security",
						Description: "Container network configuration poses security risks",
						Priority:    SeverityMedium,
						Actions: []string{
							"Avoid using --network=host unless necessary",
							"Limit published ports to only required ones",
							"Use custom networks for container isolation",
							"Configure appropriate firewall rules",
						},
						Resources: []string{
							"https://docs.docker.com/network/",
						},
					})
				}

			case "Mount Security":
				if factor.Score >= 40.0 { // Dangerous mounts
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "mount-001",
						Title:       "Secure Filesystem Mounts",
						Description: "Container has potentially dangerous filesystem mounts",
						Priority:    SeverityHigh,
						Actions: []string{
							"Avoid mounting sensitive host directories",
							"Use read-only mounts where possible",
							"Consider using named volumes instead of bind mounts",
							"Review all mounted paths for necessity",
						},
						Resources: []string{
							"https://docs.docker.com/storage/",
						},
					})
				}

			case "Resource Management":
				if factor.Score >= 30.0 { // Resource limits
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "res-001",
						Title:       "Configure Resource Limits",
						Description: "Container lacks proper resource constraints",
						Priority:    SeverityMedium,
						Actions: []string{
							"Set memory limits with --memory flag",
							"Configure CPU limits with --cpus or --cpu-shares",
							"Set PID limits to prevent fork bombs",
							"Configure appropriate ulimits",
						},
						Resources: []string{
							"https://docs.docker.com/config/containers/resource_constraints/",
						},
					})
				}

			case "Image Security":
				if factor.Score >= 25.0 { // Image issues
					recommendations = append(recommendations, SecurityRecommendation{
						ID:          "img-001",
						Title:       "Improve Image Security",
						Description: "Container image has security concerns",
						Priority:    SeverityLow,
						Actions: []string{
							"Use specific image tags instead of 'latest'",
							"Use official or verified images when possible",
							"Implement image scanning in CI/CD pipeline",
							"Use content trust with image digests",
						},
						Resources: []string{
							"https://docs.docker.com/develop/dev-best-practices/#image-security",
						},
					})
				}
			}
		}
	}

	// Add general recommendations based on overall risk
	overallScore := 0.0
	for _, factor := range factors {
		overallScore += factor.Score * factor.Weight
	}

	if overallScore > 70.0 {
		recommendations = append(recommendations, SecurityRecommendation{
			ID:          "gen-001",
			Title:       "Comprehensive Security Review",
			Description: "Container has multiple high-risk security issues requiring immediate attention",
			Priority:    SeverityCritical,
			Actions: []string{
				"Conduct immediate security review of container configuration",
				"Implement security scanning in deployment pipeline",
				"Consider using security-focused base images",
				"Implement runtime security monitoring",
			},
			Resources: []string{
				"https://cheatsheetseries.owasp.org/cheatsheets/Docker_Security_Cheat_Sheet.html",
			},
		})
	}

	return recommendations
}