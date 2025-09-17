package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/go-connections/nat"
	"github.com/docker/go-units"
)

// Container represents a Docker container
type Container struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	ImageID string            `json:"image_id"`
	Status  string            `json:"status"`
	State   string            `json:"state"`
	Created time.Time         `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
	Ports   []PortBinding     `json:"ports,omitempty"`
	Mounts  []MountPoint      `json:"mounts,omitempty"`
}

// Image represents a Docker image
type Image struct {
	ID          string            `json:"id"`
	Repository  string            `json:"repository"`
	Tag         string            `json:"tag"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Created     time.Time         `json:"created"`
	Labels      map[string]string `json:"labels,omitempty"`
	RepoTags    []string          `json:"repo_tags,omitempty"`
	RepoDigests []string          `json:"repo_digests,omitempty"`
}

// PortBinding represents port binding information
type PortBinding struct {
	PrivatePort int    `json:"private_port"`
	PublicPort  int    `json:"public_port,omitempty"`
	Type        string `json:"type"`
	IP          string `json:"ip,omitempty"`
}

// MountPoint represents mount point information
type MountPoint struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
	Propagation string `json:"propagation"`
}

// ImageListOptions represents options for listing images
type ImageListOptions struct {
	All     bool              `json:"all"`
	Filters map[string]string `json:"filters,omitempty"`
}

// ImageRemoveOptions represents options for removing images
type ImageRemoveOptions struct {
	Force         bool `json:"force"`
	PruneChildren bool `json:"prune_children"`
}

// ContainerListOptions represents options for listing containers
type ContainerListOptions struct {
	All     bool              `json:"all"`
	Latest  bool              `json:"latest"`
	Size    bool              `json:"size"`
	Filters map[string]string `json:"filters,omitempty"`
}

// ContainerCreateConfig represents configuration for creating a container
type ContainerCreateConfig struct {
	Name          string                 `json:"name"`
	Image         string                 `json:"image"`
	Tag           string                 `json:"tag"`
	Env           []string               `json:"env"`
	Ports         map[string]string      `json:"ports"`
	Volumes       []VolumeMount          `json:"volumes"`
	Networks      []string               `json:"networks"`
	RestartPolicy string                 `json:"restart_policy"`
	Labels        map[string]string      `json:"labels"`
	NetworkMode   string                 `json:"network_mode"`
	Resources     *ResourceConfig        `json:"resources,omitempty"`
	HealthCheck   *HealthCheckConfig     `json:"health_check,omitempty"`
	Command       []string               `json:"command,omitempty"`
	Entrypoint    []string               `json:"entrypoint,omitempty"`
	WorkingDir    string                 `json:"working_dir,omitempty"`
	User          string                 `json:"user,omitempty"`
	Hostname      string                 `json:"hostname,omitempty"`
	Privileged    bool                   `json:"privileged"`
	AutoRemove    bool                   `json:"auto_remove"`
	ReadOnly      bool                   `json:"read_only"`
	ExtraHosts    []string               `json:"extra_hosts,omitempty"`
	DNS           []string               `json:"dns,omitempty"`
	DNSSearch     []string               `json:"dns_search,omitempty"`
	DNSOptions    []string               `json:"dns_options,omitempty"`
	Capabilities  *CapabilityConfig      `json:"capabilities,omitempty"`
	Devices       []DeviceMapping        `json:"devices,omitempty"`
	Tmpfs         map[string]string      `json:"tmpfs,omitempty"`
	Ulimits       []UlimitConfig         `json:"ulimits,omitempty"`
	LogConfig     *LogConfig             `json:"log_config,omitempty"`
}

// VolumeMount represents a volume mount configuration
type VolumeMount struct {
	Type        string            `json:"type"`          // bind, volume, tmpfs
	Source      string            `json:"source"`        // host path or volume name
	Target      string            `json:"target"`        // container path
	ReadOnly    bool              `json:"read_only"`
	Consistency string            `json:"consistency"`   // default, consistent, cached, delegated
	BindOptions *BindOptions      `json:"bind_options,omitempty"`
	VolumeOptions *VolumeOptions  `json:"volume_options,omitempty"`
	TmpfsOptions *TmpfsOptions    `json:"tmpfs_options,omitempty"`
}

// BindOptions represents bind mount options
type BindOptions struct {
	Propagation string `json:"propagation"` // private, rprivate, shared, rshared, slave, rslave
}

// VolumeOptions represents volume mount options
type VolumeOptions struct {
	NoCopy       bool              `json:"no_copy"`
	Labels       map[string]string `json:"labels"`
	DriverConfig *VolumeDriverConfig `json:"driver_config,omitempty"`
}

// VolumeDriverConfig represents volume driver configuration
type VolumeDriverConfig struct {
	Name    string            `json:"name"`
	Options map[string]string `json:"options"`
}

// TmpfsOptions represents tmpfs mount options
type TmpfsOptions struct {
	SizeBytes int64 `json:"size_bytes"`
	Mode      int32 `json:"mode"`
}

// ResourceConfig represents container resource constraints
type ResourceConfig struct {
	CPUShares          int64             `json:"cpu_shares,omitempty"`
	Memory             int64             `json:"memory,omitempty"`
	MemorySwap         int64             `json:"memory_swap,omitempty"`
	MemoryReservation  int64             `json:"memory_reservation,omitempty"`
	KernelMemory       int64             `json:"kernel_memory,omitempty"`
	CPUQuota           int64             `json:"cpu_quota,omitempty"`
	CPUPeriod          int64             `json:"cpu_period,omitempty"`
	CPUSetCPUs         string            `json:"cpuset_cpus,omitempty"`
	CPUSetMems         string            `json:"cpuset_mems,omitempty"`
	BlkioWeight        uint16            `json:"blkio_weight,omitempty"`
	BlkioWeightDevice  []WeightDevice    `json:"blkio_weight_device,omitempty"`
	BlkioDeviceReadBps []ThrottleDevice  `json:"blkio_device_read_bps,omitempty"`
	BlkioDeviceWriteBps []ThrottleDevice `json:"blkio_device_write_bps,omitempty"`
	BlkioDeviceReadIOps []ThrottleDevice `json:"blkio_device_read_iops,omitempty"`
	BlkioDeviceWriteIOps []ThrottleDevice `json:"blkio_device_write_iops,omitempty"`
	DeviceCgroupRules  []string          `json:"device_cgroup_rules,omitempty"`
	DiskQuota          int64             `json:"disk_quota,omitempty"`
	OomKillDisable     bool              `json:"oom_kill_disable"`
	OomScoreAdj        int               `json:"oom_score_adj,omitempty"`
	PidsLimit          int64             `json:"pids_limit,omitempty"`
}

// WeightDevice represents a device weight configuration
type WeightDevice struct {
	Path   string `json:"path"`
	Weight uint16 `json:"weight"`
}

// ThrottleDevice represents a device throttle configuration
type ThrottleDevice struct {
	Path string `json:"path"`
	Rate uint64 `json:"rate"`
}

// HealthCheckConfig represents container health check configuration
type HealthCheckConfig struct {
	Test        []string      `json:"test"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Retries     int           `json:"retries"`
	StartPeriod time.Duration `json:"start_period"`
	Disabled    bool          `json:"disabled"`
}

// CapabilityConfig represents container capability configuration
type CapabilityConfig struct {
	Add  []string `json:"add"`
	Drop []string `json:"drop"`
}

// DeviceMapping represents a device mapping configuration
type DeviceMapping struct {
	PathOnHost        string `json:"path_on_host"`
	PathInContainer   string `json:"path_in_container"`
	CgroupPermissions string `json:"cgroup_permissions"`
}

// UlimitConfig represents a ulimit configuration
type UlimitConfig struct {
	Name string `json:"name"`
	Soft int64  `json:"soft"`
	Hard int64  `json:"hard"`
}

// LogConfig represents container logging configuration
type LogConfig struct {
	Type   string            `json:"type"`
	Config map[string]string `json:"config"`
}

// ContainerUpdateConfig represents configuration for updating a container
type ContainerUpdateConfig struct {
	Image         string             `json:"image"`
	Tag           string             `json:"tag"`
	Strategy      UpdateStrategy     `json:"strategy"`
	HealthCheck   *HealthCheckConfig `json:"health_check,omitempty"`
	Rollback      bool               `json:"rollback"`
	Timeout       time.Duration      `json:"timeout"`
	Resources     *ResourceConfig    `json:"resources,omitempty"`
	Environment   []string           `json:"environment,omitempty"`
	Labels        map[string]string  `json:"labels,omitempty"`
	RestartPolicy string             `json:"restart_policy,omitempty"`
}

// UpdateStrategy defines container update strategies
type UpdateStrategy string

const (
	UpdateStrategyRecreate    UpdateStrategy = "recreate"
	UpdateStrategyRolling     UpdateStrategy = "rolling"
	UpdateStrategyBlueGreen   UpdateStrategy = "blue_green"
	UpdateStrategyCanary      UpdateStrategy = "canary"
)

// RestartPolicy defines container restart policies
type RestartPolicy string

const (
	RestartPolicyNo             RestartPolicy = "no"
	RestartPolicyAlways         RestartPolicy = "always"
	RestartPolicyUnlessStopped  RestartPolicy = "unless-stopped"
	RestartPolicyOnFailure      RestartPolicy = "on-failure"
)

// ValidateConfig validates the container create configuration
func (c *ContainerCreateConfig) ValidateConfig() error {
	if c.Name == "" {
		return fmt.Errorf("container name is required")
	}

	if c.Image == "" {
		return fmt.Errorf("container image is required")
	}

	// Validate ports
	for hostPort, containerPort := range c.Ports {
		if _, err := strconv.Atoi(hostPort); err != nil {
			return fmt.Errorf("invalid host port: %s", hostPort)
		}
		if _, err := strconv.Atoi(containerPort); err != nil {
			return fmt.Errorf("invalid container port: %s", containerPort)
		}
	}

	// Validate volumes
	for _, volume := range c.Volumes {
		if volume.Target == "" {
			return fmt.Errorf("volume target path is required")
		}
		if volume.Type == "bind" && volume.Source == "" {
			return fmt.Errorf("bind mount source path is required")
		}
	}

	// Validate restart policy
	validPolicies := []string{"no", "always", "unless-stopped", "on-failure"}
	if c.RestartPolicy != "" {
		found := false
		for _, policy := range validPolicies {
			if c.RestartPolicy == policy {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid restart policy: %s", c.RestartPolicy)
		}
	}

	// Validate health check
	if c.HealthCheck != nil {
		if len(c.HealthCheck.Test) == 0 {
			return fmt.Errorf("health check test command is required")
		}
	}

	return nil
}

// ToDockerConfig converts ContainerCreateConfig to Docker API types
func (c *ContainerCreateConfig) ToDockerConfig() (*container.Config, *container.HostConfig, *network.NetworkingConfig, error) {
	// Set default tag if not specified
	tag := c.Tag
	if tag == "" {
		tag = "latest"
	}

	// Build full image name
	image := c.Image + ":" + tag

	// Container config
	config := &container.Config{
		Image:        image,
		Env:          c.Env,
		Labels:       c.Labels,
		WorkingDir:   c.WorkingDir,
		User:         c.User,
		Hostname:     c.Hostname,
		AttachStdout: false,
		AttachStderr: false,
		AttachStdin:  false,
		Tty:          false,
		OpenStdin:    false,
		StdinOnce:    false,
	}

	// Set command and entrypoint
	if len(c.Command) > 0 {
		config.Cmd = strslice.StrSlice(c.Command)
	}
	if len(c.Entrypoint) > 0 {
		config.Entrypoint = strslice.StrSlice(c.Entrypoint)
	}

	// Set exposed ports
	if len(c.Ports) > 0 {
		exposedPorts := make(nat.PortSet)
		for _, containerPort := range c.Ports {
			port, err := nat.NewPort("tcp", containerPort)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("invalid port: %s", containerPort)
			}
			exposedPorts[port] = struct{}{}
		}
		config.ExposedPorts = exposedPorts
	}

	// Set health check
	if c.HealthCheck != nil {
		config.Healthcheck = &container.HealthConfig{
			Test:        c.HealthCheck.Test,
			Interval:    c.HealthCheck.Interval,
			Timeout:     c.HealthCheck.Timeout,
			Retries:     c.HealthCheck.Retries,
			StartPeriod: c.HealthCheck.StartPeriod,
		}
		if c.HealthCheck.Disabled {
			config.Healthcheck.Test = []string{"NONE"}
		}
	}

	// Host config
	hostConfig := &container.HostConfig{
		AutoRemove:   c.AutoRemove,
		Privileged:   c.Privileged,
		ReadonlyRootfs: c.ReadOnly,
		NetworkMode:  container.NetworkMode(c.NetworkMode),
		ExtraHosts:   c.ExtraHosts,
		DNS:          c.DNS,
		DNSSearch:    c.DNSSearch,
		DNSOptions:   c.DNSOptions,
		Tmpfs:        c.Tmpfs,
	}

	// Set restart policy
	if c.RestartPolicy != "" {
		policy := container.RestartPolicy{Name: container.RestartPolicyMode(c.RestartPolicy)}
		if c.RestartPolicy == "on-failure" {
			policy.MaximumRetryCount = 3 // Default retry count
		}
		hostConfig.RestartPolicy = policy
	}

	// Set port bindings
	if len(c.Ports) > 0 {
		portBindings := make(nat.PortMap)
		for hostPort, containerPort := range c.Ports {
			containerPortNat, err := nat.NewPort("tcp", containerPort)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("invalid container port: %s", containerPort)
			}
			portBindings[containerPortNat] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			}
		}
		hostConfig.PortBindings = portBindings
	}

	// Set mounts
	if len(c.Volumes) > 0 {
		var mounts []mount.Mount
		for _, vol := range c.Volumes {
			m := mount.Mount{
				Type:     mount.Type(vol.Type),
				Source:   vol.Source,
				Target:   vol.Target,
				ReadOnly: vol.ReadOnly,
			}

			// Set consistency
			if vol.Consistency != "" {
				m.Consistency = mount.Consistency(vol.Consistency)
			}

			// Set bind options
			if vol.BindOptions != nil {
				m.BindOptions = &mount.BindOptions{
					Propagation: mount.Propagation(vol.BindOptions.Propagation),
				}
			}

			// Set volume options
			if vol.VolumeOptions != nil {
				m.VolumeOptions = &mount.VolumeOptions{
					NoCopy: vol.VolumeOptions.NoCopy,
					Labels: vol.VolumeOptions.Labels,
				}
				if vol.VolumeOptions.DriverConfig != nil {
					m.VolumeOptions.DriverConfig = &mount.Driver{
						Name:    vol.VolumeOptions.DriverConfig.Name,
						Options: vol.VolumeOptions.DriverConfig.Options,
					}
				}
			}

			// Set tmpfs options
			if vol.TmpfsOptions != nil {
				m.TmpfsOptions = &mount.TmpfsOptions{
					SizeBytes: vol.TmpfsOptions.SizeBytes,
					Mode:      os.FileMode(vol.TmpfsOptions.Mode),
				}
			}

			mounts = append(mounts, m)
		}
		hostConfig.Mounts = mounts
	}

	// Set resources
	if c.Resources != nil {
		resources := &container.Resources{
			CPUShares:          c.Resources.CPUShares,
			Memory:             c.Resources.Memory,
			MemorySwap:         c.Resources.MemorySwap,
			MemoryReservation:  c.Resources.MemoryReservation,
			KernelMemory:       c.Resources.KernelMemory,
			CPUQuota:           c.Resources.CPUQuota,
			CPUPeriod:          c.Resources.CPUPeriod,
			CpusetCpus:         c.Resources.CPUSetCPUs,
			CpusetMems:         c.Resources.CPUSetMems,
			BlkioWeight:        c.Resources.BlkioWeight,
			OomKillDisable:     &c.Resources.OomKillDisable,
			PidsLimit:          &c.Resources.PidsLimit,
		}

		// OOM score adjustment is not available in this API version

		// Set device cgroup rules
		if len(c.Resources.DeviceCgroupRules) > 0 {
			resources.DeviceCgroupRules = c.Resources.DeviceCgroupRules
		}

		hostConfig.Resources = *resources
	}

	// Set capabilities
	if c.Capabilities != nil {
		hostConfig.CapAdd = strslice.StrSlice(c.Capabilities.Add)
		hostConfig.CapDrop = strslice.StrSlice(c.Capabilities.Drop)
	}

	// Set devices
	if len(c.Devices) > 0 {
		var devices []container.DeviceMapping
		for _, device := range c.Devices {
			devices = append(devices, container.DeviceMapping{
				PathOnHost:        device.PathOnHost,
				PathInContainer:   device.PathInContainer,
				CgroupPermissions: device.CgroupPermissions,
			})
		}
		hostConfig.Devices = devices
	}

	// Set ulimits
	if len(c.Ulimits) > 0 {
		var ulimits []*units.Ulimit
		for _, ulimit := range c.Ulimits {
			ulimits = append(ulimits, &units.Ulimit{
				Name: ulimit.Name,
				Soft: ulimit.Soft,
				Hard: ulimit.Hard,
			})
		}
		hostConfig.Ulimits = ulimits
	}

	// Set log config
	if c.LogConfig != nil {
		hostConfig.LogConfig = container.LogConfig{
			Type:   c.LogConfig.Type,
			Config: c.LogConfig.Config,
		}
	}

	// Network config
	networkingConfig := &network.NetworkingConfig{}

	return config, hostConfig, networkingConfig, nil
}

// GetFullImageName returns the full image name with tag
func (c *ContainerCreateConfig) GetFullImageName() string {
	tag := c.Tag
	if tag == "" {
		tag = "latest"
	}
	return c.Image + ":" + tag
}

// GetValidUpdateStrategies returns all valid update strategies
func GetValidUpdateStrategies() []UpdateStrategy {
	return []UpdateStrategy{
		UpdateStrategyRecreate,
		UpdateStrategyRolling,
		UpdateStrategyBlueGreen,
		UpdateStrategyCanary,
	}
}

// GetValidRestartPolicies returns all valid restart policies
func GetValidRestartPolicies() []RestartPolicy {
	return []RestartPolicy{
		RestartPolicyNo,
		RestartPolicyAlways,
		RestartPolicyUnlessStopped,
		RestartPolicyOnFailure,
	}
}

// ParseImageName parses a full image name into image and tag components
func ParseImageName(fullImageName string) (image, tag string) {
	parts := strings.Split(fullImageName, ":")
	if len(parts) == 1 {
		return parts[0], "latest"
	}
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	// Handle registry with port (e.g., localhost:5000/image:tag)
	colonIndex := strings.LastIndex(fullImageName, ":")
	if colonIndex == -1 {
		return fullImageName, "latest"
	}
	return fullImageName[:colonIndex], fullImageName[colonIndex+1:]
}

// BuildImageName builds a full image name from image and tag
func BuildImageName(image, tag string) string {
	if tag == "" {
		tag = "latest"
	}
	return image + ":" + tag
}

// ValidateImageName validates an image name format
func ValidateImageName(imageName string) error {
	if imageName == "" {
		return fmt.Errorf("image name cannot be empty")
	}

	// Basic validation - can be extended
	if strings.Contains(imageName, " ") {
		return fmt.Errorf("image name cannot contain spaces")
	}

	return nil
}

// ContainerStatus represents detailed container runtime status
type ContainerStatus struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	State        string            `json:"state"`        // running, stopped, paused, etc.
	Status       string            `json:"status"`       // detailed status message
	Running      bool              `json:"running"`      // true if container is running
	Paused       bool              `json:"paused"`       // true if container is paused
	Restarting   bool              `json:"restarting"`   // true if container is restarting
	OOMKilled    bool              `json:"oom_killed"`   // true if killed by OOM
	Dead         bool              `json:"dead"`         // true if container is dead
	Pid          int               `json:"pid"`          // process ID of the container
	ExitCode     int               `json:"exit_code"`    // exit code of the container
	Error        string            `json:"error"`        // error message if any
	StartedAt    time.Time         `json:"started_at"`   // when the container was started
	FinishedAt   time.Time         `json:"finished_at"`  // when the container finished
	Health       *ContainerHealthStatus     `json:"health,omitempty"` // health check status
	RestartCount int               `json:"restart_count"` // number of restarts
	Platform     string            `json:"platform"`      // container platform
}

// ContainerHealthStatus represents container health status
type ContainerHealthStatus struct {
	Status        string              `json:"status"`          // healthy, unhealthy, starting, none
	FailingStreak int                 `json:"failing_streak"`  // number of consecutive failures
	Log           []HealthLogEntry    `json:"log,omitempty"`   // health check logs
}

// HealthLogEntry represents a single health check log entry
type HealthLogEntry struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	ExitCode int       `json:"exit_code"`
	Output   string    `json:"output"`
}

// ToJSON converts any struct to JSON string
func ToJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON converts JSON string to struct
func FromJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

// BulkOperationConfig represents configuration for bulk operations
type BulkOperationConfig struct {
	ContainerIDs      []string                              `json:"container_ids"`
	MaxConcurrency    int                                   `json:"max_concurrency"`
	Timeout           int                                   `json:"timeout"` // seconds
	ContinueOnError   bool                                  `json:"continue_on_error"`
	Force             bool                                  `json:"force"`
	FailFast          bool                                  `json:"fail_fast"`
	ProgressCallback  func(completed int, total int) error `json:"-"`
}

// ParallelOperationResult represents result of parallel operations
type ParallelOperationResult struct {
	ContainerID string        `json:"container_id"`
	Operation   string        `json:"operation"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	Data        interface{}   `json:"data,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
}

// DefaultBulkConfig returns default configuration for bulk operations
func DefaultBulkConfig() BulkOperationConfig {
	return BulkOperationConfig{
		MaxConcurrency:  5,
		Timeout:         30,
		ContinueOnError: true,
		Force:           false,
	}
}