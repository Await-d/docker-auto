package config

import (
	"time"

	"docker-auto/pkg/alerting"
	"docker-auto/pkg/health"
	"docker-auto/pkg/logging"
)

// MetricsConfig represents metrics collection configuration
type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled" json:"enabled"`
	CollectionInterval time.Duration `yaml:"collection_interval" json:"collection_interval"`
	RetentionPeriod    time.Duration `yaml:"retention_period" json:"retention_period"`
	MaxMetrics         int           `yaml:"max_metrics" json:"max_metrics"`
	BufferSize         int           `yaml:"buffer_size" json:"buffer_size"`
	Storage            StorageConfig `yaml:"storage" json:"storage"`
	Export             ExportConfig  `yaml:"export" json:"export"`
}

// StorageConfig represents storage configuration for metrics
type StorageConfig struct {
	Type string `yaml:"type" json:"type"`
}

// ExportConfig represents export configuration for metrics
type ExportConfig struct {
	Enabled   bool          `yaml:"enabled" json:"enabled"`
	Format    string        `yaml:"format" json:"format"`
	Interval  time.Duration `yaml:"interval" json:"interval"`
	BatchSize int           `yaml:"batch_size" json:"batch_size"`
}

// MonitoringConfig represents the complete monitoring configuration
type MonitoringConfig struct {
	Logging           logging.LogConfig                        `yaml:"logging" json:"logging"`
	Monitoring        MetricsConfig                            `yaml:"monitoring" json:"monitoring"`
	HealthChecks      health.HealthConfig                      `yaml:"health_checks" json:"health_checks"`
	HealthCheckConfigs HealthCheckConfigs                      `yaml:"health_check_configs" json:"health_check_configs"`
	Alerting          alerting.AlertingConfig                  `yaml:"alerting" json:"alerting"`
	AlertRoutes       []alerting.AlertRoute                    `yaml:"alert_routes" json:"alert_routes"`
	AlertReceivers    []alerting.AlertReceiver                 `yaml:"alert_receivers" json:"alert_receivers"`
	AlertRules        []alerting.AlertRule                     `yaml:"alert_rules" json:"alert_rules"`
	Development       DevelopmentConfig                        `yaml:"development" json:"development"`
	Production        ProductionConfig                         `yaml:"production" json:"production"`
}

// HealthCheckConfigs contains configurations for specific health checks
type HealthCheckConfigs struct {
	Database   health.DatabaseHealthConfig   `yaml:"database" json:"database"`
	Docker     health.DockerHealthConfig     `yaml:"docker" json:"docker"`
	API        health.HTTPHealthConfig       `yaml:"api" json:"api"`
	FileSystem health.FileSystemHealthConfig `yaml:"filesystem" json:"filesystem"`
	Memory     health.MemoryHealthConfig     `yaml:"memory" json:"memory"`
	CPU        health.CPUHealthConfig        `yaml:"cpu" json:"cpu"`
}

// DevelopmentConfig contains development-specific configuration
type DevelopmentConfig struct {
	DebugLogging  bool `yaml:"debug_logging" json:"debug_logging"`
	DebugMetrics  bool `yaml:"debug_metrics" json:"debug_metrics"`
	MockServices  bool `yaml:"mock_services" json:"mock_services"`
	TestAlerts    bool `yaml:"test_alerts" json:"test_alerts"`
}

// ProductionConfig contains production-specific configuration overrides
type ProductionConfig struct {
	Logging      logging.LogConfig        `yaml:"logging" json:"logging"`
	Monitoring   MetricsConfig            `yaml:"monitoring" json:"monitoring"`
	HealthChecks health.HealthConfig      `yaml:"health_checks" json:"health_checks"`
	Alerting     alerting.AlertingConfig  `yaml:"alerting" json:"alerting"`
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() *MonitoringConfig {
	return &MonitoringConfig{
		Logging: logging.LogConfig{
			Level:      logging.INFO,
			Format:     "json",
			Output:     "stdout",
			Rotation:   false,
			BufferSize: 1000,
		},
		Monitoring: MetricsConfig{
			Enabled:            true,
			CollectionInterval: 30 * time.Second,
			RetentionPeriod:    7 * 24 * time.Hour,
			MaxMetrics:         10000,
			BufferSize:         1000,
			Storage: StorageConfig{
				Type: "memory",
			},
			Export: ExportConfig{
				Enabled:   true,
				Format:    "prometheus",
				Interval:  60 * time.Second,
				BatchSize: 100,
			},
		},
		HealthChecks: health.HealthConfig{
			Enabled:          true,
			CheckInterval:    30 * time.Second,
			Timeout:          10 * time.Second,
			FailureThreshold: 3,
			SuccessThreshold: 1,
			GracePeriod:      60 * time.Second,
			RetryAttempts:    2,
			RetryDelay:       5 * time.Second,
			EnableRecovery:   false,
		},
		HealthCheckConfigs: HealthCheckConfigs{
			Database: health.DatabaseHealthConfig{
				HealthCheckConfig: health.HealthCheckConfig{
					Name:             "database",
					Enabled:          true,
					Interval:         60 * time.Second,
					Timeout:          5 * time.Second,
					FailureThreshold: 3,
					SuccessThreshold: 1,
					Critical:         true,
				},
				QueryTimeout: 3 * time.Second,
				TestQuery:    "SELECT 1",
			},
			Docker: health.DockerHealthConfig{
				HealthCheckConfig: health.HealthCheckConfig{
					Name:             "docker",
					Enabled:          true,
					Interval:         60 * time.Second,
					Timeout:          10 * time.Second,
					FailureThreshold: 2,
					SuccessThreshold: 1,
					Critical:         true,
				},
				RequestTimeout:  10 * time.Second,
				CheckContainers: true,
				CheckImages:     true,
				CheckNetworks:   true,
			},
			API: health.HTTPHealthConfig{
				HealthCheckConfig: health.HealthCheckConfig{
					Name:             "api",
					Enabled:          true,
					Interval:         30 * time.Second,
					Timeout:          5 * time.Second,
					FailureThreshold: 3,
					SuccessThreshold: 1,
					Critical:         false,
				},
				URL:             "http://localhost:8080/health",
				Method:          "GET",
				ExpectedStatus:  []int{200},
				FollowRedirects: true,
			},
			FileSystem: health.FileSystemHealthConfig{
				HealthCheckConfig: health.HealthCheckConfig{
					Name:             "filesystem",
					Enabled:          true,
					Interval:         300 * time.Second,
					Timeout:          5 * time.Second,
					FailureThreshold: 2,
					SuccessThreshold: 1,
					Critical:         false,
				},
				Path:             "/var/lib/docker-auto",
				MinFreeSpace:     1024 * 1024 * 1024, // 1GB
				MinFreePercent:   10.0,
				CheckWritable:    true,
				CheckReadable:    true,
			},
			Memory: health.MemoryHealthConfig{
				HealthCheckConfig: health.HealthCheckConfig{
					Name:             "memory",
					Enabled:          true,
					Interval:         60 * time.Second,
					Timeout:          2 * time.Second,
					FailureThreshold: 3,
					SuccessThreshold: 1,
					Critical:         false,
				},
				MaxMemoryPercent: 90.0,
				CheckSwap:        true,
				MaxSwapPercent:   50.0,
			},
			CPU: health.CPUHealthConfig{
				HealthCheckConfig: health.HealthCheckConfig{
					Name:             "cpu",
					Enabled:          true,
					Interval:         60 * time.Second,
					Timeout:          5 * time.Second,
					FailureThreshold: 5,
					SuccessThreshold: 2,
					Critical:         false,
				},
				MaxCPUPercent:   95.0,
				MaxLoadAverage:  10.0,
				SampleDuration:  30 * time.Second,
			},
		},
		Alerting: alerting.AlertingConfig{
			Enabled:            true,
			EvaluationInterval: 30 * time.Second,
			AlertTimeout:       5 * time.Minute,
			ResolveTimeout:     5 * time.Minute,
			GroupWait:          30 * time.Second,
			GroupInterval:      5 * time.Minute,
			RepeatInterval:     12 * time.Hour,
			MaxAlerts:          1000,
			Storage: alerting.AlertStorageConfig{
				Type:      "memory",
				Retention: 30 * 24 * time.Hour,
			},
		},
		Development: DevelopmentConfig{
			DebugLogging: false,
			DebugMetrics: false,
			MockServices: false,
			TestAlerts:   false,
		},
	}
}

// ApplyEnvironmentOverrides applies environment-specific overrides
func (c *MonitoringConfig) ApplyEnvironmentOverrides(environment string) {
	switch environment {
	case "production":
		if c.Production.Logging.Level != 0 {
			c.Logging = c.Production.Logging
		}
		if c.Production.Monitoring.Enabled {
			c.Monitoring = c.Production.Monitoring
		}
		if c.Production.HealthChecks.Enabled {
			c.HealthChecks = c.Production.HealthChecks
		}
		if c.Production.Alerting.Enabled {
			c.Alerting = c.Production.Alerting
		}
	case "development":
		if c.Development.DebugLogging {
			c.Logging.Level = logging.DEBUG
		}
		if c.Development.MockServices {
			// Disable external service health checks
			c.HealthCheckConfigs.Docker.Enabled = false
		}
	}
}

// Validate validates the monitoring configuration
func (c *MonitoringConfig) Validate() error {
	// Validate logging configuration
	if c.Logging.Level < logging.DEBUG || c.Logging.Level > logging.FATAL {
		c.Logging.Level = logging.INFO
	}

	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		c.Logging.Format = "json"
	}

	// Validate monitoring configuration
	if c.Monitoring.CollectionInterval < time.Second {
		c.Monitoring.CollectionInterval = 30 * time.Second
	}

	if c.Monitoring.RetentionPeriod < time.Hour {
		c.Monitoring.RetentionPeriod = 24 * time.Hour
	}

	if c.Monitoring.MaxMetrics < 100 {
		c.Monitoring.MaxMetrics = 10000
	}

	// Validate health check configuration
	if c.HealthChecks.CheckInterval < 10*time.Second {
		c.HealthChecks.CheckInterval = 30 * time.Second
	}

	if c.HealthChecks.Timeout < time.Second {
		c.HealthChecks.Timeout = 10 * time.Second
	}

	if c.HealthChecks.FailureThreshold < 1 {
		c.HealthChecks.FailureThreshold = 3
	}

	// Validate alerting configuration
	if c.Alerting.EvaluationInterval < 10*time.Second {
		c.Alerting.EvaluationInterval = 30 * time.Second
	}

	if c.Alerting.MaxAlerts < 100 {
		c.Alerting.MaxAlerts = 1000
	}

	return nil
}

// GetAlertRuleByID returns an alert rule by its ID
func (c *MonitoringConfig) GetAlertRuleByID(id string) *alerting.AlertRule {
	for i, rule := range c.AlertRules {
		if rule.ID == id {
			return &c.AlertRules[i]
		}
	}
	return nil
}

// GetAlertReceiverByName returns an alert receiver by its name
func (c *MonitoringConfig) GetAlertReceiverByName(name string) *alerting.AlertReceiver {
	for i, receiver := range c.AlertReceivers {
		if receiver.Name == name {
			return &c.AlertReceivers[i]
		}
	}
	return nil
}

// IsHealthCheckEnabled checks if a specific health check is enabled
func (c *MonitoringConfig) IsHealthCheckEnabled(name string) bool {
	if !c.HealthChecks.Enabled {
		return false
	}

	switch name {
	case "database":
		return c.HealthCheckConfigs.Database.Enabled
	case "docker":
		return c.HealthCheckConfigs.Docker.Enabled
	case "api":
		return c.HealthCheckConfigs.API.Enabled
	case "filesystem":
		return c.HealthCheckConfigs.FileSystem.Enabled
	case "memory":
		return c.HealthCheckConfigs.Memory.Enabled
	case "cpu":
		return c.HealthCheckConfigs.CPU.Enabled
	default:
		return false
	}
}