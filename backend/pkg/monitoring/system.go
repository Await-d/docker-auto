package monitoring

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// SystemMetricsCollector collects system-level metrics
type SystemMetricsCollector struct {
	dockerClient *client.Client
	startTime    time.Time
}

// NewSystemMetricsCollector creates a new system metrics collector
func NewSystemMetricsCollector() *SystemMetricsCollector {
	dockerClient, _ := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	return &SystemMetricsCollector{
		dockerClient: dockerClient,
		startTime:    time.Now(),
	}
}

// Collect gathers all system metrics
func (smc *SystemMetricsCollector) Collect() (SystemMetrics, error) {
	metrics := SystemMetrics{
		Timestamp: time.Now(),
	}

	// Collect CPU metrics
	cpuMetrics, err := smc.collectCPUMetrics()
	if err == nil {
		metrics.CPU = cpuMetrics
	}

	// Collect memory metrics
	memMetrics, err := smc.collectMemoryMetrics()
	if err == nil {
		metrics.Memory = memMetrics
	}

	// Collect disk metrics
	diskMetrics, err := smc.collectDiskMetrics()
	if err == nil {
		metrics.Disk = diskMetrics
	}

	// Collect network metrics
	netMetrics, err := smc.collectNetworkMetrics()
	if err == nil {
		metrics.Network = netMetrics
	}

	// Collect Docker metrics
	dockerMetrics, err := smc.collectDockerMetrics()
	if err == nil {
		metrics.Docker = dockerMetrics
	}

	// Collect application metrics
	appMetrics := smc.collectApplicationMetrics()
	metrics.Application = appMetrics

	return metrics, nil
}

// collectCPUMetrics collects CPU usage and load metrics
func (smc *SystemMetricsCollector) collectCPUMetrics() (CPUMetrics, error) {
	metrics := CPUMetrics{
		Cores: runtime.NumCPU(),
	}

	// Get CPU usage from /proc/stat
	usage, err := smc.getCPUUsage()
	if err == nil {
		metrics.Usage = usage
	}

	// Get load averages from /proc/loadavg
	load1, load5, load15, err := smc.getLoadAverage()
	if err == nil {
		metrics.LoadAvg1 = load1
		metrics.LoadAvg5 = load5
		metrics.LoadAvg15 = load15
	}

	return metrics, nil
}

// collectMemoryMetrics collects memory usage metrics
func (smc *SystemMetricsCollector) collectMemoryMetrics() (MemoryMetrics, error) {
	metrics := MemoryMetrics{}

	// Read /proc/meminfo
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return metrics, err
	}
	defer file.Close()

	memInfo := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := strings.TrimSuffix(parts[0], ":")
			value, err := strconv.ParseUint(parts[1], 10, 64)
			if err == nil {
				memInfo[key] = value * 1024 // Convert KB to bytes
			}
		}
	}

	metrics.Total = memInfo["MemTotal"]
	metrics.Available = memInfo["MemAvailable"]
	metrics.Used = metrics.Total - metrics.Available
	if metrics.Total > 0 {
		metrics.Usage = float64(metrics.Used) / float64(metrics.Total) * 100
	}

	// Swap metrics
	metrics.Swap.Total = memInfo["SwapTotal"]
	swapFree := memInfo["SwapFree"]
	metrics.Swap.Used = metrics.Swap.Total - swapFree
	if metrics.Swap.Total > 0 {
		metrics.Swap.Usage = float64(metrics.Swap.Used) / float64(metrics.Swap.Total) * 100
	}

	return metrics, nil
}

// collectDiskMetrics collects disk usage metrics
func (smc *SystemMetricsCollector) collectDiskMetrics() (DiskMetrics, error) {
	metrics := DiskMetrics{}

	// Get disk usage for root filesystem
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return metrics, err
	}

	// Calculate disk metrics
	blockSize := uint64(stat.Bsize)
	metrics.Total = stat.Blocks * blockSize
	metrics.Available = stat.Bavail * blockSize
	metrics.Used = metrics.Total - metrics.Available

	if metrics.Total > 0 {
		metrics.Usage = float64(metrics.Used) / float64(metrics.Total) * 100
	}

	// Get disk I/O statistics from /proc/diskstats
	ioStats, err := smc.getDiskIOStats()
	if err == nil {
		metrics.ReadOps = ioStats.ReadOps
		metrics.WriteOps = ioStats.WriteOps
		metrics.ReadBytes = ioStats.ReadBytes
		metrics.WriteBytes = ioStats.WriteBytes
	}

	return metrics, nil
}

// collectNetworkMetrics collects network usage metrics
func (smc *SystemMetricsCollector) collectNetworkMetrics() (NetworkMetrics, error) {
	metrics := NetworkMetrics{}

	// Read /proc/net/dev
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return metrics, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 17 {
			// Skip loopback interface
			if strings.HasPrefix(parts[0], "lo:") {
				continue
			}

			// Parse network statistics
			rxBytes, _ := strconv.ParseUint(parts[1], 10, 64)
			rxPackets, _ := strconv.ParseUint(parts[2], 10, 64)
			rxErrs, _ := strconv.ParseUint(parts[3], 10, 64)
			rxDrop, _ := strconv.ParseUint(parts[4], 10, 64)
			txBytes, _ := strconv.ParseUint(parts[9], 10, 64)
			txPackets, _ := strconv.ParseUint(parts[10], 10, 64)
			txErrs, _ := strconv.ParseUint(parts[11], 10, 64)
			txDrop, _ := strconv.ParseUint(parts[12], 10, 64)

			metrics.BytesReceived += rxBytes
			metrics.PacketsReceived += rxPackets
			metrics.ErrorsReceived += rxErrs
			metrics.DroppedReceived += rxDrop
			metrics.BytesSent += txBytes
			metrics.PacketsSent += txPackets
			metrics.ErrorsSent += txErrs
			metrics.DroppedSent += txDrop
		}
	}

	return metrics, nil
}

// collectDockerMetrics collects Docker-related metrics
func (smc *SystemMetricsCollector) collectDockerMetrics() (DockerMetrics, error) {
	metrics := DockerMetrics{}

	if smc.dockerClient == nil {
		return metrics, fmt.Errorf("docker client not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get container statistics
	containers, err := smc.dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return metrics, err
	}

	for _, container := range containers {
		switch container.State {
		case "running":
			metrics.ContainersRunning++
		case "exited", "stopped":
			metrics.ContainersStopped++
		case "paused":
			metrics.ContainersPaused++
		}
	}

	// Get image count
	images, err := smc.dockerClient.ImageList(ctx, types.ImageListOptions{})
	if err == nil {
		metrics.Images = len(images)
	}

	// Get volume count
	volumes, err := smc.dockerClient.VolumeList(ctx, types.VolumeListOptions{})
	if err == nil {
		metrics.Volumes = len(volumes.Volumes)
	}

	// Get network count
	networks, err := smc.dockerClient.NetworkList(ctx, types.NetworkListOptions{})
	if err == nil {
		metrics.Networks = len(networks)
	}

	return metrics, nil
}

// collectApplicationMetrics collects application-specific metrics
func (smc *SystemMetricsCollector) collectApplicationMetrics() ApplicationMetrics {
	metrics := ApplicationMetrics{
		Uptime:  time.Since(smc.startTime),
		Version: "1.0.0", // This should be set from build information
	}

	// These would typically be collected from the application's internal state
	// For now, we'll set some default values
	metrics.RequestsTotal = 0
	metrics.RequestsPerSecond = 0
	metrics.ResponseTime = 0
	metrics.ErrorRate = 0
	metrics.ActiveConnections = 0

	return metrics
}

// Helper functions for system metric collection

// getCPUUsage calculates CPU usage percentage
func (smc *SystemMetricsCollector) getCPUUsage() (float64, error) {
	// Read /proc/stat twice to calculate CPU usage
	stat1, err := smc.readProcStat()
	if err != nil {
		return 0, err
	}

	time.Sleep(100 * time.Millisecond)

	stat2, err := smc.readProcStat()
	if err != nil {
		return 0, err
	}

	// Calculate CPU usage
	idle1 := stat1[3] + stat1[4]
	idle2 := stat2[3] + stat2[4]

	total1 := uint64(0)
	total2 := uint64(0)
	for _, val := range stat1 {
		total1 += val
	}
	for _, val := range stat2 {
		total2 += val
	}

	totalDelta := total2 - total1
	idleDelta := idle2 - idle1

	if totalDelta == 0 {
		return 0, nil
	}

	return (1.0 - float64(idleDelta)/float64(totalDelta)) * 100, nil
}

// readProcStat reads CPU statistics from /proc/stat
func (smc *SystemMetricsCollector) readProcStat() ([]uint64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read /proc/stat")
	}

	line := scanner.Text()
	parts := strings.Fields(line)
	if len(parts) < 8 || parts[0] != "cpu" {
		return nil, fmt.Errorf("invalid /proc/stat format")
	}

	values := make([]uint64, len(parts)-1)
	for i, part := range parts[1:] {
		val, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil, err
		}
		values[i] = val
	}

	return values, nil
}

// getLoadAverage reads load averages from /proc/loadavg
func (smc *SystemMetricsCollector) getLoadAverage() (float64, float64, float64, error) {
	file, err := os.Open("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0, 0, fmt.Errorf("failed to read /proc/loadavg")
	}

	parts := strings.Fields(scanner.Text())
	if len(parts) < 3 {
		return 0, 0, 0, fmt.Errorf("invalid /proc/loadavg format")
	}

	load1, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, 0, 0, err
	}

	load5, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, 0, 0, err
	}

	load15, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return 0, 0, 0, err
	}

	return load1, load5, load15, nil
}

// DiskIOStats represents disk I/O statistics
type DiskIOStats struct {
	ReadOps    uint64
	WriteOps   uint64
	ReadBytes  uint64
	WriteBytes uint64
}

// getDiskIOStats reads disk I/O statistics from /proc/diskstats
func (smc *SystemMetricsCollector) getDiskIOStats() (DiskIOStats, error) {
	stats := DiskIOStats{}

	file, err := os.Open("/proc/diskstats")
	if err != nil {
		return stats, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) >= 14 {
			// Skip loop devices and partitions (only consider whole disks like sda, sdb, etc.)
			deviceName := parts[2]
			if strings.HasPrefix(deviceName, "loop") ||
			   len(deviceName) > 3 && (deviceName[len(deviceName)-1] >= '0' && deviceName[len(deviceName)-1] <= '9') {
				continue
			}

			readOps, _ := strconv.ParseUint(parts[3], 10, 64)
			readSectors, _ := strconv.ParseUint(parts[5], 10, 64)
			writeOps, _ := strconv.ParseUint(parts[7], 10, 64)
			writeSectors, _ := strconv.ParseUint(parts[9], 10, 64)

			stats.ReadOps += readOps
			stats.WriteOps += writeOps
			stats.ReadBytes += readSectors * 512  // Sectors are typically 512 bytes
			stats.WriteBytes += writeSectors * 512
		}
	}

	return stats, nil
}