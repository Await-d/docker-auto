package main

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"docker-auto/internal/service"
	"docker-auto/pkg/dashboard"
	"docker-auto/pkg/docker"

	"github.com/docker/docker/api/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"gorm.io/gorm"
)

// DockerClientAdapter adapts docker.DockerClient to dashboard.DockerClientInterface
type DockerClientAdapter struct {
	client *docker.DockerClient
}

func (dca *DockerClientAdapter) ListContainers(ctx context.Context, all bool) ([]dashboard.ContainerInfo, error) {
	if dca.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	dockerContainers, err := dca.client.ListContainers(ctx, types.ContainerListOptions{All: all})
	if err != nil {
		return nil, err
	}

	containers := make([]dashboard.ContainerInfo, 0, len(dockerContainers))
	for _, container := range dockerContainers {
		// Extract health status (health info requires container inspection)
		var health *dashboard.HealthInfo
		// Health info is not available in the list containers response,
		// it would need to be retrieved via container inspection if needed

		// Convert ports
		ports := make([]dashboard.Port, 0, len(container.Ports))
		for _, port := range container.Ports {
			ports = append(ports, dashboard.Port{
				PrivatePort: int(port.PrivatePort),
				PublicPort:  int(port.PublicPort),
				Type:        port.Type,
				IP:          port.IP,
			})
		}

		dashboardContainer := dashboard.ContainerInfo{
			ID:      container.ID,
			Names:   container.Names,
			Image:   container.Image,
			State:   container.State,
			Status:  container.Status,
			Created: time.Unix(container.Created, 0),
			Ports:   ports,
			Labels:  container.Labels,
			Health:  health,
		}
		containers = append(containers, dashboardContainer)
	}

	return containers, nil
}

func (dca *DockerClientAdapter) GetContainerStats(ctx context.Context, containerID string) (*dashboard.ContainerStats, error) {
	if dca.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	// Get container info first
	containerInfo, err := dca.client.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Get container stats
	dockerStats, err := dca.client.GetContainerStats(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Extract container name
	name := containerID
	if containerInfo.Name != "" {
		name = strings.TrimPrefix(containerInfo.Name, "/")
	}

	// Calculate stats from Docker stats structure
	var cpuPercent float64
	var memoryUsage, memoryLimit uint64
	var networkRx, networkTx uint64
	var blockRead, blockWrite uint64

	if dockerStats != nil {
		// Calculate CPU percentage (simplified calculation)
		if dockerStats.CPUStats.SystemUsage > 0 && dockerStats.PreCPUStats.SystemUsage > 0 {
			cpuDelta := float64(dockerStats.CPUStats.CPUUsage.TotalUsage - dockerStats.PreCPUStats.CPUUsage.TotalUsage)
			systemDelta := float64(dockerStats.CPUStats.SystemUsage - dockerStats.PreCPUStats.SystemUsage)
			if systemDelta > 0 {
				cpuPercent = (cpuDelta / systemDelta) * float64(len(dockerStats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
			}
		}

		// Memory stats
		memoryUsage = dockerStats.MemoryStats.Usage
		memoryLimit = dockerStats.MemoryStats.Limit

		// Network stats (sum all interfaces)
		for _, network := range dockerStats.Networks {
			networkRx += network.RxBytes
			networkTx += network.TxBytes
		}

		// Block I/O stats
		for _, blkio := range dockerStats.BlkioStats.IoServiceBytesRecursive {
			if blkio.Op == "Read" {
				blockRead += blkio.Value
			} else if blkio.Op == "Write" {
				blockWrite += blkio.Value
			}
		}
	}

	stats := &dashboard.ContainerStats{
		ContainerID: containerID,
		Name:        name,
		CPUPercent:  cpuPercent,
		MemoryUsage: memoryUsage,
		MemoryLimit: memoryLimit,
		NetworkRx:   networkRx,
		NetworkTx:   networkTx,
		BlockRead:   blockRead,
		BlockWrite:  blockWrite,
		Timestamp:   time.Now(),
	}

	return stats, nil
}

func (dca *DockerClientAdapter) Ping(ctx context.Context) error {
	if dca.client == nil {
		return fmt.Errorf("docker client not initialized")
	}

	err := dca.client.Ping(ctx)
	return err
}

// SystemMonitorAdapter implements dashboard.SystemMonitorInterface using gopsutil
type SystemMonitorAdapter struct{}

func (sma *SystemMonitorAdapter) GetCPUUsage() (float64, error) {
	percentages, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}

	if len(percentages) == 0 {
		return 0, fmt.Errorf("no CPU usage data available")
	}

	return percentages[0], nil
}

func (sma *SystemMonitorAdapter) GetMemoryUsage() (*dashboard.MemoryUsage, error) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}

	return &dashboard.MemoryUsage{
		Total:       memInfo.Total,
		Available:   memInfo.Available,
		Used:        memInfo.Used,
		UsedPercent: memInfo.UsedPercent,
	}, nil
}

func (sma *SystemMonitorAdapter) GetDiskUsage() (*dashboard.DiskUsage, error) {
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return nil, err
	}

	return &dashboard.DiskUsage{
		Path:        diskInfo.Path,
		Total:       diskInfo.Total,
		Free:        diskInfo.Free,
		Used:        diskInfo.Used,
		UsedPercent: diskInfo.UsedPercent,
	}, nil
}

func (sma *SystemMonitorAdapter) GetNetworkUsage() (*dashboard.NetworkUsage, error) {
	netStats, err := net.IOCounters(false)
	if err != nil {
		return nil, err
	}

	if len(netStats) == 0 {
		return &dashboard.NetworkUsage{}, nil
	}

	// Sum up all network interfaces
	var totalBytesIn, totalBytesOut, totalPacketsIn, totalPacketsOut uint64
	for _, stat := range netStats {
		totalBytesIn += stat.BytesRecv
		totalBytesOut += stat.BytesSent
		totalPacketsIn += stat.PacketsRecv
		totalPacketsOut += stat.PacketsSent
	}

	return &dashboard.NetworkUsage{
		BytesIn:    totalBytesIn,
		BytesOut:   totalBytesOut,
		PacketsIn:  totalPacketsIn,
		PacketsOut: totalPacketsOut,
	}, nil
}

// DatabaseAdapter implements dashboard.DatabaseInterface
type DatabaseAdapter struct {
	db *gorm.DB
}

func (da *DatabaseAdapter) GetSecurityVulnerabilities(ctx context.Context) ([]dashboard.Vulnerability, error) {
	var vulnerabilities []service.SecurityVulnerability

	err := da.db.WithContext(ctx).Find(&vulnerabilities).Error
	if err != nil {
		return nil, err
	}

	// Convert to dashboard format
	dashboardVulns := make([]dashboard.Vulnerability, len(vulnerabilities))
	for i, vuln := range vulnerabilities {
		dashboardVulns[i] = dashboard.Vulnerability{
			ID:          fmt.Sprintf("%d", vuln.ID),
			CVE:         vuln.CVE,
			Severity:    vuln.Severity,
			Description: vuln.Description,
			Package:     vuln.Package,
			Version:     vuln.InstalledVersion,
			FixedIn:     vuln.FixedVersion,
			Container:   vuln.ContainerName,
			CreatedAt:   vuln.CreatedAt,
		}
	}

	return dashboardVulns, nil
}

func (da *DatabaseAdapter) GetLastSecurityScanTime(ctx context.Context) (time.Time, error) {
	var lastScan service.SecurityScan

	err := da.db.WithContext(ctx).
		Order("created_at DESC").
		First(&lastScan).Error

	if err == gorm.ErrRecordNotFound {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	return lastScan.CreatedAt, nil
}

func (da *DatabaseAdapter) GetPendingUpdatesCount(ctx context.Context) (int, error) {
	var count int64

	err := da.db.WithContext(ctx).
		Model(&service.UpdateActivity{}).
		Where("status = ?", "pending").
		Count(&count).Error

	return int(count), err
}

func (da *DatabaseAdapter) GetRecentUpdatesCount(ctx context.Context, since time.Time) (int, error) {
	var count int64

	err := da.db.WithContext(ctx).
		Model(&service.UpdateActivity{}).
		Where("status = ? AND updated_at > ?", "completed", since).
		Count(&count).Error

	return int(count), err
}

func (da *DatabaseAdapter) GetFailedUpdatesCount(ctx context.Context, since time.Time) (int, error) {
	var count int64

	err := da.db.WithContext(ctx).
		Model(&service.UpdateActivity{}).
		Where("status IN ? AND updated_at > ?", []string{"failed", "rolled_back"}, since).
		Count(&count).Error

	return int(count), err
}

func (da *DatabaseAdapter) GetLastUpdateTime(ctx context.Context) (time.Time, error) {
	var lastUpdate service.UpdateActivity

	err := da.db.WithContext(ctx).
		Where("status = ?", "completed").
		Order("updated_at DESC").
		First(&lastUpdate).Error

	if err == gorm.ErrRecordNotFound {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, err
	}

	return lastUpdate.UpdatedAt, nil
}

func (da *DatabaseAdapter) IsAutoUpdateEnabled(ctx context.Context) (bool, error) {
	// This would check system settings or container configurations
	// For now, return a default value
	return true, nil
}

func (da *DatabaseAdapter) Ping(ctx context.Context) error {
	sqlDB, err := da.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.PingContext(ctx)
}

// DockerServiceAdapter adapts docker client to service interface
type DockerServiceAdapter struct {
	client *docker.DockerClient
}

func (dsa *DockerServiceAdapter) ListContainers(ctx context.Context) ([]service.ContainerInfo, error) {
	if dsa.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	dockerContainers, err := dsa.client.ListContainers(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	containers := make([]service.ContainerInfo, 0, len(dockerContainers))
	for _, container := range dockerContainers {
		// Convert ports
		ports := make([]service.PortInfo, 0, len(container.Ports))
		for _, port := range container.Ports {
			ports = append(ports, service.PortInfo{
				PrivatePort: int(port.PrivatePort),
				PublicPort:  int(port.PublicPort),
				Type:        port.Type,
				IP:          port.IP,
			})
		}

		// Extract name
		name := container.ID
		if len(container.Names) > 0 {
			name = strings.TrimPrefix(container.Names[0], "/")
		}

		serviceContainer := service.ContainerInfo{
			ID:      container.ID,
			Names:   container.Names,
			Name:    name,
			Image:   container.Image,
			ImageID: container.ImageID,
			Status:  container.Status,
			State:   container.State,
			Created: time.Unix(container.Created, 0),
			Labels:  container.Labels,
			Ports:   ports,
		}
		containers = append(containers, serviceContainer)
	}

	return containers, nil
}

func (dsa *DockerServiceAdapter) GetContainer(ctx context.Context, containerID string) (*service.ContainerInfo, error) {
	if dsa.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	container, err := dsa.client.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	// Convert ports from NetworkSettings
	var ports []service.PortInfo
	if container.NetworkSettings != nil && container.NetworkSettings.Ports != nil {
		for port, bindings := range container.NetworkSettings.Ports {
			for _, binding := range bindings {
				publicPort := 0
				if binding.HostPort != "" {
					// Convert string port to int (ignoring errors for now)
					if p, err := strconv.Atoi(binding.HostPort); err == nil {
						publicPort = p
					}
				}
				privatePort := int(port.Int())
				ports = append(ports, service.PortInfo{
					PrivatePort: privatePort,
					PublicPort:  publicPort,
					Type:        string(port.Proto()),
					IP:          binding.HostIP,
				})
			}
		}
	}

	// Extract name
	name := container.ID
	if container.Name != "" {
		name = strings.TrimPrefix(container.Name, "/")
	}

	return &service.ContainerInfo{
		ID:      container.ID,
		Names:   []string{name}, // ContainerJSON has Name field, not Names array
		Name:    name,
		Image:   container.Image,
		ImageID: container.Image, // Use container.Image as ImageID
		Status:  container.State.Status,
		State:   container.State.Status, // State should be the status string
		Created: func() time.Time {
			if t, err := time.Parse(time.RFC3339, container.Created); err == nil {
				return t
			}
			return time.Now() // fallback
		}(),
		Labels:  container.Config.Labels,
		Ports:   ports,
	}, nil
}

func (dsa *DockerServiceAdapter) GetLatestImageTag(ctx context.Context, imageName string) (string, error) {
	if dsa.client == nil {
		return "", fmt.Errorf("docker client not initialized")
	}

	// Extract repository and current tag
	parts := strings.Split(imageName, ":")
	if len(parts) < 2 {
		return "latest", nil
	}

	repository := parts[0]

	// For now, return "latest" as we'd need to implement Docker registry API calls
	// This is a placeholder - in production, you'd query the registry
	_ = repository

	return "latest", nil
}

func (dsa *DockerServiceAdapter) ExecCommand(ctx context.Context, containerID, command string) error {
	if dsa.client == nil {
		return fmt.Errorf("docker client not initialized")
	}

	// This would execute a command in the container
	// Implementation depends on your docker client's capabilities
	return fmt.Errorf("exec command not implemented")
}

func (dsa *DockerServiceAdapter) PullImage(ctx context.Context, imageName string) error {
	if dsa.client == nil {
		return fmt.Errorf("docker client not initialized")
	}

	reader, err := dsa.client.PullImage(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Read the response to completion (required for pull to finish)
	// In a real implementation, you might want to process this output
	_, err = io.ReadAll(reader)
	return err
}

func (dsa *DockerServiceAdapter) StopContainer(ctx context.Context, containerID string, timeout time.Duration) error {
	if dsa.client == nil {
		return fmt.Errorf("docker client not initialized")
	}

	timeoutInt := int(timeout.Seconds())
	return dsa.client.StopContainer(ctx, containerID, &timeoutInt)
}

func (dsa *DockerServiceAdapter) UpdateContainer(ctx context.Context, containerID, newImage string) error {
	if dsa.client == nil {
		return fmt.Errorf("docker client not initialized")
	}

	// This would update the container with a new image
	// Implementation depends on your requirements
	return fmt.Errorf("update container not implemented")
}

func (dsa *DockerServiceAdapter) GetImageLayers(ctx context.Context, imageID string) ([]string, error) {
	if dsa.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	// This would get image layer information
	// Placeholder implementation
	return []string{}, nil
}

func (dsa *DockerServiceAdapter) GetImagePackages(ctx context.Context, imageID string) ([]service.PackageInfo, error) {
	if dsa.client == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	// This would scan image for installed packages
	// Placeholder implementation
	return []service.PackageInfo{}, nil
}

// VulnerabilityDBAdapter implements service.VulnerabilityDBInterface
type VulnerabilityDBAdapter struct{}

func (vdb *VulnerabilityDBAdapter) QueryVulnerabilities(ctx context.Context, packageName, version string) ([]service.VulnerabilityInfo, error) {
	// This would query a vulnerability database like NVD
	// Placeholder implementation
	return []service.VulnerabilityInfo{}, nil
}

func (vdb *VulnerabilityDBAdapter) UpdateVulnerabilityDB(ctx context.Context) error {
	// This would update the vulnerability database
	// Placeholder implementation
	return nil
}

func (vdb *VulnerabilityDBAdapter) GetLastUpdateTime() time.Time {
	// Return current time as placeholder
	return time.Now()
}

// SecurityScannerAdapter implements service.SecurityScannerInterface
type SecurityScannerAdapter struct{}

func (ssa *SecurityScannerAdapter) ScanImage(ctx context.Context, imageID, imageName string) (*service.ScanResult, error) {
	// This would perform actual security scanning
	// Placeholder implementation that creates a basic scan result
	return &service.ScanResult{
		ScanID:         fmt.Sprintf("scan_%s_%d", imageID[:12], time.Now().Unix()),
		ImageID:        imageID,
		ImageName:      imageName,
		Scanner:        "placeholder",
		ScannerVersion: "1.0.0",
		Vulnerabilities: []*service.Vulnerability{}, // Use proper slice type
		// LowVulns field not available in service.ScanResult
		Metadata:        make(map[string]interface{}),
	}, nil
}

func (ssa *SecurityScannerAdapter) GetScannerInfo() service.ScannerInfo {
	return service.ScannerInfo{
		Name:    "PlaceholderScanner",
		Version: "1.0.0",
		// Vendor field not available in service.ScannerInfo
	}
}