package service

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"docker-auto/pkg/docker"

	"github.com/sirupsen/logrus"
)

// NewComplianceChecker creates a new compliance checker
func NewComplianceChecker(logger *logrus.Entry) *ComplianceChecker {
	checker := &ComplianceChecker{
		checks: make(map[ComplianceStandard][]*ComplianceRule),
		logger: logger.WithField("subcomponent", "compliance_checker"),
	}

	// Initialize compliance rules
	checker.initializeCISRules()
	checker.initializeNISTRules()
	checker.initializeSOC2Rules()

	return checker
}

// RunChecks runs compliance checks for a specific standard
func (cc *ComplianceChecker) RunChecks(ctx context.Context, standard ComplianceStandard, container *docker.ContainerInfo) ([]*ComplianceCheck, error) {
	rules, exists := cc.checks[standard]
	if !exists {
		return nil, fmt.Errorf("no rules found for standard: %s", standard)
	}

	var checks []*ComplianceCheck
	for _, rule := range rules {
		check := rule.CheckFunc(ctx, container)
		if check != nil {
			checks = append(checks, check)
		}
	}

	cc.logger.WithFields(logrus.Fields{
		"container_id": container.ID,
		"standard":     standard,
		"checks_run":   len(checks),
	}).Info("Compliance checks completed")

	return checks, nil
}

// Initialize CIS (Center for Internet Security) rules
func (cc *ComplianceChecker) initializeCISRules() {
	cisRules := []*ComplianceRule{
		{
			ID:          "CIS-4.1",
			Name:        "Ensure that a user for the container has been created",
			Description: "Container should not run as root user",
			Standard:    StandardCIS,
			Category:    "User Management",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkNonRootUser,
		},
		{
			ID:          "CIS-4.5",
			Name:        "Ensure Content trust for Docker is Enabled",
			Description: "Content trust provides the ability to use digital signatures",
			Standard:    StandardCIS,
			Category:    "Image Security",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkContentTrust,
		},
		{
			ID:          "CIS-5.1",
			Name:        "Ensure that, if applicable, AppArmor Profile is Enabled",
			Description: "AppArmor is a Linux kernel security module",
			Standard:    StandardCIS,
			Category:    "Security Modules",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkAppArmorProfile,
		},
		{
			ID:          "CIS-5.2",
			Name:        "Ensure that, if applicable, SELinux security options are set",
			Description: "SELinux provides additional layer of security",
			Standard:    StandardCIS,
			Category:    "Security Modules",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkSELinuxOptions,
		},
		{
			ID:          "CIS-5.3",
			Name:        "Ensure that Linux Kernel Capabilities are restricted within containers",
			Description: "Linux kernel capabilities should be restricted",
			Standard:    StandardCIS,
			Category:    "Capabilities",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkCapabilities,
		},
		{
			ID:          "CIS-5.4",
			Name:        "Ensure that privileged containers are not used",
			Description: "Privileged containers should not be used",
			Standard:    StandardCIS,
			Category:    "Privilege Escalation",
			Severity:    SeverityCritical,
			CheckFunc:   cc.checkPrivilegedContainer,
		},
		{
			ID:          "CIS-5.5",
			Name:        "Ensure sensitive host system directories are not mounted on containers",
			Description: "Sensitive host directories should not be mounted",
			Standard:    StandardCIS,
			Category:    "Mount Security",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkSensitiveMounts,
		},
		{
			ID:          "CIS-5.7",
			Name:        "Ensure the default ulimit is configured appropriately",
			Description: "Default ulimit should be configured",
			Standard:    StandardCIS,
			Category:    "Resource Limits",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkUlimits,
		},
		{
			ID:          "CIS-5.10",
			Name:        "Ensure that the memory usage for containers is limited",
			Description: "Memory usage should be limited",
			Standard:    StandardCIS,
			Category:    "Resource Limits",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkMemoryLimits,
		},
		{
			ID:          "CIS-5.11",
			Name:        "Ensure that CPU priority is set appropriately on containers",
			Description: "CPU priority should be set",
			Standard:    StandardCIS,
			Category:    "Resource Limits",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkCPULimits,
		},
		{
			ID:          "CIS-5.12",
			Name:        "Ensure that the container's root filesystem is mounted as read only",
			Description: "Root filesystem should be read-only",
			Standard:    StandardCIS,
			Category:    "Filesystem Security",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkReadOnlyRootFS,
		},
		{
			ID:          "CIS-5.15",
			Name:        "Ensure that the host's process namespace is not shared",
			Description: "Host process namespace should not be shared",
			Standard:    StandardCIS,
			Category:    "Namespace Isolation",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkProcessNamespace,
		},
		{
			ID:          "CIS-5.16",
			Name:        "Ensure that the host's network namespace is not shared",
			Description: "Host network namespace should not be shared",
			Standard:    StandardCIS,
			Category:    "Network Security",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkNetworkNamespace,
		},
	}

	cc.checks[StandardCIS] = cisRules
}

// Initialize NIST rules
func (cc *ComplianceChecker) initializeNISTRules() {
	nistRules := []*ComplianceRule{
		{
			ID:          "NIST-AC-2",
			Name:        "Account Management",
			Description: "Ensure proper account management practices",
			Standard:    StandardNIST,
			Category:    "Access Control",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkAccountManagement,
		},
		{
			ID:          "NIST-AC-3",
			Name:        "Access Enforcement",
			Description: "Ensure access enforcement mechanisms",
			Standard:    StandardNIST,
			Category:    "Access Control",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkAccessEnforcement,
		},
		{
			ID:          "NIST-CM-2",
			Name:        "Baseline Configuration",
			Description: "Ensure baseline configuration management",
			Standard:    StandardNIST,
			Category:    "Configuration Management",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkBaselineConfiguration,
		},
		{
			ID:          "NIST-SC-7",
			Name:        "Boundary Protection",
			Description: "Ensure proper boundary protection",
			Standard:    StandardNIST,
			Category:    "System Communications",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkBoundaryProtection,
		},
	}

	cc.checks[StandardNIST] = nistRules
}

// Initialize SOC2 rules
func (cc *ComplianceChecker) initializeSOC2Rules() {
	soc2Rules := []*ComplianceRule{
		{
			ID:          "SOC2-CC6.1",
			Name:        "Logical and Physical Access Controls",
			Description: "Ensure proper access controls are implemented",
			Standard:    StandardSOC2,
			Category:    "Security",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkAccessControls,
		},
		{
			ID:          "SOC2-CC6.3",
			Name:        "Network Access Controls",
			Description: "Ensure proper network access controls",
			Standard:    StandardSOC2,
			Category:    "Security",
			Severity:    SeverityHigh,
			CheckFunc:   cc.checkNetworkAccessControls,
		},
		{
			ID:          "SOC2-CC7.1",
			Name:        "System Monitoring",
			Description: "Ensure proper system monitoring",
			Standard:    StandardSOC2,
			Category:    "Monitoring",
			Severity:    SeverityMedium,
			CheckFunc:   cc.checkSystemMonitoring,
		},
	}

	cc.checks[StandardSOC2] = soc2Rules
}

// Compliance check implementations

// checkNonRootUser checks if container runs as non-root user
func (cc *ComplianceChecker) checkNonRootUser(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "non-root-user",
		Name:        "Non-root user check",
		Description: "Verify container runs as non-root user",
		Standard:    StandardCIS,
		Category:    "User Security",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check if container config specifies a user
	user := container.Config.User
	if user == "" || user == "root" || user == "0" {
		check.Status = ComplianceStatusFail
		check.Details = "Container is running as root user"
		check.Remediation = "Set a non-root user in Dockerfile using USER instruction or in docker run command using --user flag"
		check.Evidence["user"] = user
		check.Evidence["user_id"] = "0"
	} else {
		check.Status = ComplianceStatusPass
		check.Details = fmt.Sprintf("Container is running as user: %s", user)
		check.Evidence["user"] = user
	}

	return check
}

// checkContentTrust checks if content trust is enabled
func (cc *ComplianceChecker) checkContentTrust(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "content-trust",
		Name:        "Content trust check",
		Description: "Verify content trust is enabled for images",
		Standard:    StandardCIS,
		Category:    "Image Security",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check image digest for signed content
	if strings.Contains(container.Image, "@sha256:") {
		check.Status = ComplianceStatusPass
		check.Details = "Image uses content digest, indicating signed content"
		check.Evidence["has_digest"] = true
		check.Evidence["image"] = container.Image
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "Image does not use content digest"
		check.Remediation = "Use docker images with digest (sha256) to ensure content trust"
		check.Evidence["has_digest"] = false
		check.Evidence["image"] = container.Image
	}

	return check
}

// checkAppArmorProfile checks AppArmor profile
func (cc *ComplianceChecker) checkAppArmorProfile(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "apparmor-profile",
		Name:        "AppArmor profile check",
		Description: "Verify AppArmor profile is configured",
		Standard:    StandardCIS,
		Category:    "Security Modules",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check for AppArmor profile in security options
	if container.HostConfig.SecurityOpt != nil {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.HasPrefix(opt, "apparmor=") {
				profile := strings.TrimPrefix(opt, "apparmor=")
				if profile != "unconfined" {
					check.Status = ComplianceStatusPass
					check.Details = fmt.Sprintf("AppArmor profile configured: %s", profile)
					check.Evidence["apparmor_profile"] = profile
					return check
				}
			}
		}
	}

	check.Status = ComplianceStatusWarning
	check.Details = "No AppArmor profile configured"
	check.Remediation = "Configure AppArmor profile using --security-opt apparmor=PROFILE"
	check.Evidence["apparmor_profile"] = "none"

	return check
}

// checkSELinuxOptions checks SELinux security options
func (cc *ComplianceChecker) checkSELinuxOptions(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "selinux-options",
		Name:        "SELinux options check",
		Description: "Verify SELinux security options are configured",
		Standard:    StandardCIS,
		Category:    "Security Modules",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check for SELinux options in security options
	if container.HostConfig.SecurityOpt != nil {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.HasPrefix(opt, "label=") {
				check.Status = ComplianceStatusPass
				check.Details = fmt.Sprintf("SELinux label configured: %s", opt)
				check.Evidence["selinux_label"] = opt
				return check
			}
		}
	}

	check.Status = ComplianceStatusWarning
	check.Details = "No SELinux security options configured"
	check.Remediation = "Configure SELinux options using --security-opt label=LABEL"
	check.Evidence["selinux_label"] = "none"

	return check
}

// checkCapabilities checks Linux kernel capabilities
func (cc *ComplianceChecker) checkCapabilities(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "capabilities",
		Name:        "Capabilities check",
		Description: "Verify Linux kernel capabilities are restricted",
		Standard:    StandardCIS,
		Category:    "Capabilities",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	dangerousCaps := []string{"SYS_ADMIN", "NET_ADMIN", "SYS_PTRACE", "SYS_MODULE"}

	// Check added capabilities
	if container.HostConfig.CapAdd != nil {
		for _, cap := range container.HostConfig.CapAdd {
			for _, dangerous := range dangerousCaps {
				if strings.ToUpper(cap) == dangerous || cap == "ALL" {
					check.Status = ComplianceStatusFail
					check.Details = fmt.Sprintf("Dangerous capability added: %s", cap)
					check.Remediation = "Remove dangerous capabilities or use --cap-drop ALL and only add required capabilities"
					check.Evidence["dangerous_caps"] = container.HostConfig.CapAdd
					return check
				}
			}
		}
		check.Evidence["cap_add"] = container.HostConfig.CapAdd
	}

	// Check dropped capabilities
	if container.HostConfig.CapDrop != nil {
		check.Evidence["cap_drop"] = container.HostConfig.CapDrop
		if len(container.HostConfig.CapDrop) > 0 {
			check.Status = ComplianceStatusPass
			check.Details = "Capabilities have been dropped"
		}
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "No capabilities explicitly dropped"
		check.Remediation = "Use --cap-drop to remove unnecessary capabilities"
	}

	return check
}

// checkPrivilegedContainer checks if container is privileged
func (cc *ComplianceChecker) checkPrivilegedContainer(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "privileged-container",
		Name:        "Privileged container check",
		Description: "Verify container is not running in privileged mode",
		Standard:    StandardCIS,
		Category:    "Privilege Escalation",
		Severity:    SeverityCritical,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	if container.HostConfig.Privileged {
		check.Status = ComplianceStatusFail
		check.Details = "Container is running in privileged mode"
		check.Remediation = "Do not use --privileged flag unless absolutely necessary"
		check.Evidence["privileged"] = true
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container is not running in privileged mode"
		check.Evidence["privileged"] = false
	}

	return check
}

// checkSensitiveMounts checks for sensitive host directory mounts
func (cc *ComplianceChecker) checkSensitiveMounts(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "sensitive-mounts",
		Name:        "Sensitive mounts check",
		Description: "Verify sensitive host directories are not mounted",
		Standard:    StandardCIS,
		Category:    "Mount Security",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	sensitivePaths := []string{
		"/", "/boot", "/dev", "/etc", "/lib", "/proc", "/sys", "/usr",
		"/var/lib/docker", "/etc/passwd", "/etc/shadow", "/etc/group",
	}

	var violations []string
	if container.HostConfig.Binds != nil {
		for _, bind := range container.HostConfig.Binds {
			parts := strings.Split(bind, ":")
			if len(parts) >= 2 {
				hostPath := parts[0]
				for _, sensitive := range sensitivePaths {
					if hostPath == sensitive || strings.HasPrefix(hostPath, sensitive+"/") {
						violations = append(violations, bind)
						break
					}
				}
			}
		}
	}

	if len(violations) > 0 {
		check.Status = ComplianceStatusFail
		check.Details = fmt.Sprintf("Sensitive host paths mounted: %v", violations)
		check.Remediation = "Avoid mounting sensitive host directories into containers"
		check.Evidence["violations"] = violations
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "No sensitive host directories mounted"
		check.Evidence["violations"] = []string{}
	}

	return check
}

// checkUlimits checks ulimit configuration
func (cc *ComplianceChecker) checkUlimits(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "ulimits",
		Name:        "Ulimits check",
		Description: "Verify ulimits are configured appropriately",
		Standard:    StandardCIS,
		Category:    "Resource Limits",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	if container.HostConfig.Ulimits != nil && len(container.HostConfig.Ulimits) > 0 {
		check.Status = ComplianceStatusPass
		check.Details = "Ulimits are configured"
		check.Evidence["ulimits_configured"] = true
		check.Evidence["ulimits"] = container.HostConfig.Ulimits
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "No ulimits configured"
		check.Remediation = "Configure appropriate ulimits using --ulimit flag"
		check.Evidence["ulimits_configured"] = false
	}

	return check
}

// checkMemoryLimits checks memory limits
func (cc *ComplianceChecker) checkMemoryLimits(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "memory-limits",
		Name:        "Memory limits check",
		Description: "Verify memory limits are configured",
		Standard:    StandardCIS,
		Category:    "Resource Limits",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	if container.HostConfig.Memory > 0 {
		check.Status = ComplianceStatusPass
		check.Details = fmt.Sprintf("Memory limit configured: %d bytes", container.HostConfig.Memory)
		check.Evidence["memory_limit"] = container.HostConfig.Memory
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "No memory limit configured"
		check.Remediation = "Configure memory limit using --memory flag"
		check.Evidence["memory_limit"] = 0
	}

	return check
}

// checkCPULimits checks CPU limits
func (cc *ComplianceChecker) checkCPULimits(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "cpu-limits",
		Name:        "CPU limits check",
		Description: "Verify CPU limits are configured",
		Standard:    StandardCIS,
		Category:    "Resource Limits",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	hasCPULimit := false
	evidence := make(map[string]interface{})

	if container.HostConfig.CPUShares > 0 {
		hasCPULimit = true
		evidence["cpu_shares"] = container.HostConfig.CPUShares
	}

	if container.HostConfig.CPUPeriod > 0 {
		hasCPULimit = true
		evidence["cpu_period"] = container.HostConfig.CPUPeriod
	}

	if container.HostConfig.CPUQuota > 0 {
		hasCPULimit = true
		evidence["cpu_quota"] = container.HostConfig.CPUQuota
	}

	if container.HostConfig.CpusetCpus != "" {
		hasCPULimit = true
		evidence["cpuset_cpus"] = container.HostConfig.CpusetCpus
	}

	if hasCPULimit {
		check.Status = ComplianceStatusPass
		check.Details = "CPU limits are configured"
		check.Evidence = evidence
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "No CPU limits configured"
		check.Remediation = "Configure CPU limits using --cpu-shares, --cpus, or --cpuset-cpus flags"
		check.Evidence = evidence
	}

	return check
}

// checkReadOnlyRootFS checks if root filesystem is read-only
func (cc *ComplianceChecker) checkReadOnlyRootFS(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "readonly-rootfs",
		Name:        "Read-only root filesystem check",
		Description: "Verify root filesystem is mounted as read-only",
		Standard:    StandardCIS,
		Category:    "Filesystem Security",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	if container.HostConfig.ReadonlyRootfs {
		check.Status = ComplianceStatusPass
		check.Details = "Root filesystem is read-only"
		check.Evidence["readonly_rootfs"] = true
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "Root filesystem is not read-only"
		check.Remediation = "Use --read-only flag to make root filesystem read-only"
		check.Evidence["readonly_rootfs"] = false
	}

	return check
}

// checkProcessNamespace checks process namespace sharing
func (cc *ComplianceChecker) checkProcessNamespace(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "process-namespace",
		Name:        "Process namespace check",
		Description: "Verify host process namespace is not shared",
		Standard:    StandardCIS,
		Category:    "Namespace Isolation",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	pidMode := container.HostConfig.PidMode
	if pidMode == "host" {
		check.Status = ComplianceStatusFail
		check.Details = "Container shares host process namespace"
		check.Remediation = "Do not use --pid=host flag"
		check.Evidence["pid_mode"] = pidMode
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container has isolated process namespace"
		check.Evidence["pid_mode"] = pidMode
	}

	return check
}

// checkNetworkNamespace checks network namespace sharing
func (cc *ComplianceChecker) checkNetworkNamespace(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "network-namespace",
		Name:        "Network namespace check",
		Description: "Verify host network namespace is not shared",
		Standard:    StandardCIS,
		Category:    "Network Security",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	networkMode := container.HostConfig.NetworkMode
	if networkMode == "host" {
		check.Status = ComplianceStatusFail
		check.Details = "Container shares host network namespace"
		check.Remediation = "Do not use --network=host flag unless necessary"
		check.Evidence["network_mode"] = networkMode
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container has isolated network namespace"
		check.Evidence["network_mode"] = networkMode
	}

	return check
}

// NIST compliance checks

// checkAccountManagement checks account management practices
func (cc *ComplianceChecker) checkAccountManagement(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "account-management",
		Name:        "Account management check",
		Description: "Verify proper account management practices",
		Standard:    StandardNIST,
		Category:    "Access Control",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check for non-root user
	user := container.Config.User
	if user == "" || user == "root" || user == "0" {
		check.Status = ComplianceStatusFail
		check.Details = "Container running as root violates account management principles"
		check.Remediation = "Create and use dedicated non-root user accounts"
		check.Evidence["user"] = user
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container uses non-root user account"
		check.Evidence["user"] = user
	}

	return check
}

// checkAccessEnforcement checks access enforcement mechanisms
func (cc *ComplianceChecker) checkAccessEnforcement(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "access-enforcement",
		Name:        "Access enforcement check",
		Description: "Verify access enforcement mechanisms are in place",
		Standard:    StandardNIST,
		Category:    "Access Control",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check for security options (AppArmor, SELinux)
	hasSecurityEnforcement := false
	if container.HostConfig.SecurityOpt != nil {
		for _, opt := range container.HostConfig.SecurityOpt {
			if strings.HasPrefix(opt, "apparmor=") || strings.HasPrefix(opt, "label=") {
				hasSecurityEnforcement = true
				break
			}
		}
	}

	if hasSecurityEnforcement {
		check.Status = ComplianceStatusPass
		check.Details = "Security enforcement mechanisms configured"
		check.Evidence["security_opt"] = container.HostConfig.SecurityOpt
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "No explicit security enforcement mechanisms detected"
		check.Remediation = "Configure AppArmor or SELinux security options"
		check.Evidence["security_opt"] = "none"
	}

	return check
}

// checkBaselineConfiguration checks baseline configuration management
func (cc *ComplianceChecker) checkBaselineConfiguration(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "baseline-configuration",
		Name:        "Baseline configuration check",
		Description: "Verify baseline configuration management",
		Standard:    StandardNIST,
		Category:    "Configuration Management",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check for configuration consistency indicators
	configScore := 0
	evidence := make(map[string]interface{})

	// Check if image uses specific tag (not latest)
	if !strings.HasSuffix(container.Config.Image, ":latest") && strings.Contains(container.Config.Image, ":") {
		configScore++
		evidence["specific_tag"] = true
	} else {
		evidence["specific_tag"] = false
	}

	// Check if health check is configured
	if container.Config.Healthcheck != nil {
		configScore++
		evidence["health_check"] = true
	} else {
		evidence["health_check"] = false
	}

	// Check if labels are used for metadata
	if len(container.Config.Labels) > 0 {
		configScore++
		evidence["has_labels"] = true
		evidence["label_count"] = len(container.Config.Labels)
	} else {
		evidence["has_labels"] = false
	}

	if configScore >= 2 {
		check.Status = ComplianceStatusPass
		check.Details = "Good baseline configuration practices detected"
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "Baseline configuration could be improved"
		check.Remediation = "Use specific image tags, configure health checks, and use labels for metadata"
	}

	check.Evidence = evidence
	return check
}

// checkBoundaryProtection checks boundary protection
func (cc *ComplianceChecker) checkBoundaryProtection(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "boundary-protection",
		Name:        "Boundary protection check",
		Description: "Verify proper boundary protection is implemented",
		Standard:    StandardNIST,
		Category:    "System Communications",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check network mode
	networkMode := container.HostConfig.NetworkMode
	if networkMode == "host" {
		check.Status = ComplianceStatusFail
		check.Details = "Host network mode breaks boundary protection"
		check.Remediation = "Use bridge or custom networks instead of host network"
		check.Evidence["network_mode"] = networkMode
	} else if networkMode == "none" {
		check.Status = ComplianceStatusPass
		check.Details = "Network isolation provides strong boundary protection"
		check.Evidence["network_mode"] = networkMode
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container network is isolated from host"
		check.Evidence["network_mode"] = networkMode
	}

	return check
}

// SOC2 compliance checks

// checkAccessControls checks logical and physical access controls
func (cc *ComplianceChecker) checkAccessControls(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "access-controls",
		Name:        "Access controls check",
		Description: "Verify proper access controls are implemented",
		Standard:    StandardSOC2,
		Category:    "Security",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	controlsScore := 0
	evidence := make(map[string]interface{})

	// Check user configuration
	user := container.Config.User
	if user != "" && user != "root" && user != "0" {
		controlsScore++
		evidence["non_root_user"] = true
	} else {
		evidence["non_root_user"] = false
	}

	// Check for capabilities restrictions
	if container.HostConfig.CapDrop != nil && len(container.HostConfig.CapDrop) > 0 {
		controlsScore++
		evidence["caps_dropped"] = true
	} else {
		evidence["caps_dropped"] = false
	}

	// Check for read-only root filesystem
	if container.HostConfig.ReadonlyRootfs {
		controlsScore++
		evidence["readonly_rootfs"] = true
	} else {
		evidence["readonly_rootfs"] = false
	}

	if controlsScore >= 2 {
		check.Status = ComplianceStatusPass
		check.Details = "Adequate access controls implemented"
	} else {
		check.Status = ComplianceStatusFail
		check.Details = "Insufficient access controls"
		check.Remediation = "Implement non-root user, capability restrictions, and read-only filesystem"
	}

	check.Evidence = evidence
	return check
}

// checkNetworkAccessControls checks network access controls
func (cc *ComplianceChecker) checkNetworkAccessControls(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "network-access-controls",
		Name:        "Network access controls check",
		Description: "Verify proper network access controls",
		Standard:    StandardSOC2,
		Category:    "Security",
		Severity:    SeverityHigh,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	// Check network mode
	networkMode := container.HostConfig.NetworkMode
	if networkMode == "host" {
		check.Status = ComplianceStatusFail
		check.Details = "Host network mode provides insufficient network access controls"
		check.Remediation = "Use isolated network configurations"
		check.Evidence["network_mode"] = networkMode
	} else {
		check.Status = ComplianceStatusPass
		check.Details = "Container uses isolated network configuration"
		check.Evidence["network_mode"] = networkMode
	}

	// Check published ports
	if container.HostConfig.PublishAllPorts {
		check.Status = ComplianceStatusWarning
		check.Details = "All ports are published, which may be excessive"
		check.Remediation = "Only publish necessary ports explicitly"
		check.Evidence["publish_all_ports"] = true
	} else {
		check.Evidence["publish_all_ports"] = false
	}

	return check
}

// checkSystemMonitoring checks system monitoring capabilities
func (cc *ComplianceChecker) checkSystemMonitoring(ctx context.Context, container *docker.ContainerInfo) *ComplianceCheck {
	check := &ComplianceCheck{
		ID:          "system-monitoring",
		Name:        "System monitoring check",
		Description: "Verify proper system monitoring is implemented",
		Standard:    StandardSOC2,
		Category:    "Monitoring",
		Severity:    SeverityMedium,
		LastChecked: time.Now(),
		ContainerID: &container.ID,
		Evidence:    make(map[string]interface{}),
	}

	monitoringScore := 0
	evidence := make(map[string]interface{})

	// Check for health check configuration
	if container.Config.Healthcheck != nil {
		monitoringScore++
		evidence["health_check"] = true
	} else {
		evidence["health_check"] = false
	}

	// Check for logging configuration
	if container.HostConfig.LogConfig.Type != "" && container.HostConfig.LogConfig.Type != "none" {
		monitoringScore++
		evidence["logging_configured"] = true
		evidence["log_driver"] = container.HostConfig.LogConfig.Type
	} else {
		evidence["logging_configured"] = false
	}

	// Check for restart policy (indicates monitoring/recovery capability)
	if container.HostConfig.RestartPolicy.Name != "" && container.HostConfig.RestartPolicy.Name != "no" {
		monitoringScore++
		evidence["restart_policy"] = container.HostConfig.RestartPolicy.Name
	} else {
		evidence["restart_policy"] = "no"
	}

	if monitoringScore >= 2 {
		check.Status = ComplianceStatusPass
		check.Details = "Adequate system monitoring capabilities configured"
	} else {
		check.Status = ComplianceStatusWarning
		check.Details = "System monitoring capabilities could be improved"
		check.Remediation = "Configure health checks, logging, and restart policies"
	}

	check.Evidence = evidence
	return check
}