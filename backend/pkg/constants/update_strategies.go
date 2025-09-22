package constants

// Container update strategy constants
const (
	// UpdateStrategyRecreate stops the old container and creates a new one
	UpdateStrategyRecreate = "recreate"

	// UpdateStrategyRolling performs a rolling update
	UpdateStrategyRolling = "rolling"

	// UpdateStrategyBlueGreen performs a blue-green deployment
	UpdateStrategyBlueGreen = "blue_green"

	// UpdateStrategyCanary performs a canary deployment
	UpdateStrategyCanary = "canary"

	// UpdateStrategyInPlace updates the container in place (if supported)
	UpdateStrategyInPlace = "in_place"

	// UpdateStrategyA/B performs A/B testing deployment
	UpdateStrategyAB = "ab_testing"
)

// Container restart policy constants
const (
	// RestartPolicyNo never restart the container
	RestartPolicyNo = "no"

	// RestartPolicyAlways always restart the container
	RestartPolicyAlways = "always"

	// RestartPolicyUnlessStopped restart unless explicitly stopped
	RestartPolicyUnlessStopped = "unless-stopped"

	// RestartPolicyOnFailure restart only on failure
	RestartPolicyOnFailure = "on-failure"
)

// Container health status constants
const (
	// HealthStatusHealthy indicates the service is healthy
	HealthStatusHealthy = "healthy"

	// HealthStatusUnhealthy indicates the service is unhealthy
	HealthStatusUnhealthy = "unhealthy"

	// HealthStatusStarting indicates the service is starting
	HealthStatusStarting = "starting"

	// HealthStatusNone indicates no health check is configured
	HealthStatusNone = "none"

	// HealthStatusUnknown indicates health status is unknown
	HealthStatusUnknown = "unknown"
)

// Container state constants
const (
	// ContainerStateCreated indicates container is created but not started
	ContainerStateCreated = "created"

	// ContainerStateRunning indicates container is running
	ContainerStateRunning = "running"

	// ContainerStatePaused indicates container is paused
	ContainerStatePaused = "paused"

	// ContainerStateRestarting indicates container is restarting
	ContainerStateRestarting = "restarting"

	// ContainerStateRemoving indicates container is being removed
	ContainerStateRemoving = "removing"

	// ContainerStateDead indicates container is dead
	ContainerStateDead = "dead"

	// ContainerStateExited indicates container has exited
	ContainerStateExited = "exited"
)

// Update configuration constants
const (
	// DefaultUpdateTimeout is the default timeout for update operations
	DefaultUpdateTimeout = "10m"

	// DefaultUpdateBatchSize is the default batch size for rolling updates
	DefaultUpdateBatchSize = 1

	// DefaultUpdateDelay is the default delay between update batches
	DefaultUpdateDelay = "30s"

	// DefaultRollbackTimeout is the default timeout for rollback operations
	DefaultRollbackTimeout = "5m"

	// DefaultHealthCheckTimeout is the default timeout for health checks during updates
	DefaultHealthCheckTimeout = "30s"

	// DefaultHealthCheckInterval is the default interval for health checks during updates
	DefaultHealthCheckInterval = "5s"

	// DefaultHealthCheckRetries is the default number of health check retries
	DefaultHealthCheckRetries = 3
)

// Volume mount type constants
const (
	// VolumeMountTypeBind represents bind mounts
	VolumeMountTypeBind = "bind"

	// VolumeMountTypeVolume represents volume mounts
	VolumeMountTypeVolume = "volume"

	// VolumeMountTypeTmpfs represents tmpfs mounts
	VolumeMountTypeTmpfs = "tmpfs"

	// VolumeMountTypeNpipe represents named pipe mounts (Windows)
	VolumeMountTypeNpipe = "npipe"
)

// Volume consistency constants
const (
	// VolumeConsistencyDefault represents default consistency
	VolumeConsistencyDefault = "default"

	// VolumeConsistencyConsistent represents consistent consistency
	VolumeConsistencyConsistent = "consistent"

	// VolumeConsistencyCached represents cached consistency
	VolumeConsistencyCached = "cached"

	// VolumeConsistencyDelegated represents delegated consistency
	VolumeConsistencyDelegated = "delegated"
)

// Volume propagation constants
const (
	// VolumePropagationPrivate represents private propagation
	VolumePropagationPrivate = "private"

	// VolumePropagationRPrivate represents rprivate propagation
	VolumePropagationRPrivate = "rprivate"

	// VolumePropagationShared represents shared propagation
	VolumePropagationShared = "shared"

	// VolumePropagationRShared represents rshared propagation
	VolumePropagationRShared = "rshared"

	// VolumePropagationSlave represents slave propagation
	VolumePropagationSlave = "slave"

	// VolumePropagationRSlave represents rslave propagation
	VolumePropagationRSlave = "rslave"
)

// Network mode constants
const (
	// NetworkModeNone represents no network
	NetworkModeNone = "none"

	// NetworkModeBridge represents bridge network
	NetworkModeBridge = "bridge"

	// NetworkModeHost represents host network
	NetworkModeHost = "host"

	// NetworkModeContainer represents container network
	NetworkModeContainer = "container"

	// NetworkModeDefault represents default network mode
	NetworkModeDefault = "default"
)

// Log driver constants
const (
	// LogDriverJSONFile represents json-file log driver
	LogDriverJSONFile = "json-file"

	// LogDriverSyslog represents syslog log driver
	LogDriverSyslog = "syslog"

	// LogDriverJournald represents journald log driver
	LogDriverJournald = "journald"

	// LogDriverGelf represents gelf log driver
	LogDriverGelf = "gelf"

	// LogDriverFluentd represents fluentd log driver
	LogDriverFluentd = "fluentd"

	// LogDriverAWSLogs represents awslogs log driver
	LogDriverAWSLogs = "awslogs"

	// LogDriverSplunk represents splunk log driver
	LogDriverSplunk = "splunk"

	// LogDriverEtwlogs represents etwlogs log driver (Windows)
	LogDriverEtwlogs = "etwlogs"

	// LogDriverNone represents no logging
	LogDriverNone = "none"
)

// Image pull policy constants
const (
	// PullPolicyAlways always pull the image
	PullPolicyAlways = "always"

	// PullPolicyIfNotPresent pull image if not present locally
	PullPolicyIfNotPresent = "if_not_present"

	// PullPolicyNever never pull the image
	PullPolicyNever = "never"
)

// Container operation constants
const (
	// OperationStart represents container start operation
	OperationStart = "start"

	// OperationStop represents container stop operation
	OperationStop = "stop"

	// OperationRestart represents container restart operation
	OperationRestart = "restart"

	// OperationRemove represents container remove operation
	OperationRemove = "remove"

	// OperationPause represents container pause operation
	OperationPause = "pause"

	// OperationUnpause represents container unpause operation
	OperationUnpause = "unpause"

	// OperationKill represents container kill operation
	OperationKill = "kill"

	// OperationUpdate represents container update operation
	OperationUpdate = "update"

	// OperationCreate represents container create operation
	OperationCreate = "create"
)

// Exit code constants
const (
	// ExitCodeSuccess indicates successful execution
	ExitCodeSuccess = 0

	// ExitCodeGeneralError indicates general error
	ExitCodeGeneralError = 1

	// ExitCodeMisuse indicates misuse of shell command
	ExitCodeMisuse = 2

	// ExitCodeCannotExecute indicates cannot execute
	ExitCodeCannotExecute = 126

	// ExitCodeCommandNotFound indicates command not found
	ExitCodeCommandNotFound = 127

	// ExitCodeInvalidExit indicates invalid exit argument
	ExitCodeInvalidExit = 128

	// ExitCodeKilledBySIGINT indicates killed by SIGINT (Ctrl+C)
	ExitCodeKilledBySIGINT = 130

	// ExitCodeKilledBySIGTERM indicates killed by SIGTERM
	ExitCodeKilledBySIGTERM = 143

	// ExitCodeOOMKilled indicates killed by OOM killer
	ExitCodeOOMKilled = 137
)