package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Application settings
	Port        int    `mapstructure:"APP_PORT"`
	Environment string `mapstructure:"APP_ENV"`
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	LogFormat   string `mapstructure:"LOG_FORMAT"`

	// Database settings
	Database DatabaseConfig `mapstructure:",squash"`

	// Cache settings
	Cache CacheConfig `mapstructure:",squash"`

	// JWT settings
	JWT JWTConfig `mapstructure:",squash"`

	// Docker settings
	Docker DockerConfig `mapstructure:",squash"`

	// Image check settings
	ImageCheck ImageCheckConfig `mapstructure:",squash"`

	// Notification settings
	Notification NotificationConfig `mapstructure:",squash"`

	// Security settings
	Security SecurityConfig `mapstructure:",squash"`

	// System settings
	System SystemConfig `mapstructure:",squash"`

	// Monitoring settings
	Monitoring MonitoringConfig `mapstructure:",squash"`

	// Scheduler settings
	Scheduler SchedulerConfig `mapstructure:",squash"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"DB_HOST"`
	Port     int    `mapstructure:"DB_PORT"`
	Name     string `mapstructure:"DB_NAME"`
	User     string `mapstructure:"DB_USER"`
	Password string `mapstructure:"DB_PASSWORD"`
	SSLMode  string `mapstructure:"DB_SSL_MODE"`
	Debug    bool   `mapstructure:"DB_DEBUG"`

	// Connection pool settings
	MaxIdleConns        int `mapstructure:"DB_MAX_IDLE_CONNS"`
	MaxOpenConns        int `mapstructure:"DB_MAX_OPEN_CONNS"`
	ConnMaxLifetimeMin  int `mapstructure:"DB_CONN_MAX_LIFETIME_MINUTES"`
}

type CacheConfig struct {
	// Memory cache settings
	DefaultTTLMinutes     int  `mapstructure:"CACHE_DEFAULT_TTL_MINUTES"`
	ImageCacheTTLHours    int  `mapstructure:"CACHE_IMAGE_TTL_HOURS"`
	ConfigCacheTTLMinutes int  `mapstructure:"CACHE_CONFIG_TTL_MINUTES"`
	CleanupIntervalMinutes int `mapstructure:"CACHE_CLEANUP_INTERVAL_MINUTES"`
	Enabled               bool `mapstructure:"CACHE_ENABLED"`
}

type JWTConfig struct {
	Secret           string `mapstructure:"JWT_SECRET"`
	ExpireHours      int    `mapstructure:"JWT_EXPIRE_HOURS"`
	RefreshDays      int    `mapstructure:"JWT_REFRESH_DAYS"`
}

type DockerConfig struct {
	Host           string `mapstructure:"DOCKER_HOST"`
	APIVersion     string `mapstructure:"DOCKER_API_VERSION"`
	Timeout        int    `mapstructure:"DOCKER_TIMEOUT"`
	ValidateImages bool   `mapstructure:"DOCKER_VALIDATE_IMAGES"`
}

type ImageCheckConfig struct {
	DefaultInterval      int `mapstructure:"DEFAULT_CHECK_INTERVAL"`
	MaxConcurrentChecks  int `mapstructure:"MAX_CONCURRENT_CHECKS"`
	ImageCacheHours      int `mapstructure:"IMAGE_CACHE_HOURS"`
}

type NotificationConfig struct {
	Email   EmailConfig   `mapstructure:",squash"`
	Webhook WebhookConfig `mapstructure:",squash"`
	WeChat  WeChatConfig  `mapstructure:",squash"`
}

type EmailConfig struct {
	Enabled    bool   `mapstructure:"EMAIL_ENABLED"`
	SMTPHost   string `mapstructure:"SMTP_HOST"`
	SMTPPort   int    `mapstructure:"SMTP_PORT"`
	Username   string `mapstructure:"SMTP_USERNAME"`
	Password   string `mapstructure:"SMTP_PASSWORD"`
	From       string `mapstructure:"SMTP_FROM"`
}

type WebhookConfig struct {
	Enabled bool   `mapstructure:"WEBHOOK_ENABLED"`
	URL     string `mapstructure:"WEBHOOK_URL"`
}

type WeChatConfig struct {
	Enabled    bool   `mapstructure:"WECHAT_ENABLED"`
	WebhookURL string `mapstructure:"WECHAT_WEBHOOK_URL"`
}

type SecurityConfig struct {
	EncryptionKey       string `mapstructure:"ENCRYPTION_KEY"`
	HTTPSEnabled        bool   `mapstructure:"HTTPS_ENABLED"`
	SSLCertPath         string `mapstructure:"SSL_CERT_PATH"`
	SSLKeyPath          string `mapstructure:"SSL_KEY_PATH"`
	CORSAllowedOrigins  string `mapstructure:"CORS_ALLOWED_ORIGINS"`
	CORSAllowedMethods  string `mapstructure:"CORS_ALLOWED_METHODS"`
	CORSAllowedHeaders  string `mapstructure:"CORS_ALLOWED_HEADERS"`
}

type SystemConfig struct {
	MaxLogRetentionDays    int `mapstructure:"MAX_LOG_RETENTION_DAYS"`
	MaxUpdateHistoryCount  int `mapstructure:"MAX_UPDATE_HISTORY_COUNT"`
	MaxUploadSizeMB        int `mapstructure:"MAX_UPLOAD_SIZE_MB"`
	MaxMemoryUsagePercent  int `mapstructure:"MAX_MEMORY_USAGE_PERCENT"`
	MaxDiskUsagePercent    int `mapstructure:"MAX_DISK_USAGE_PERCENT"`
	MaxCPUUsagePercent     int `mapstructure:"MAX_CPU_USAGE_PERCENT"`
}

type MonitoringConfig struct {
	PrometheusEnabled       bool   `mapstructure:"PROMETHEUS_ENABLED"`
	PrometheusPath          string `mapstructure:"PROMETHEUS_PATH"`
	HealthCheckInterval     int    `mapstructure:"HEALTH_CHECK_INTERVAL"`
	HealthCheckTimeout      int    `mapstructure:"HEALTH_CHECK_TIMEOUT"`
}

// Load reads configuration from environment variables and config files
func Load() (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Read from environment variables
	v.AutomaticEnv()

	// Try to read from config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("/etc/docker-auto")

	// Reading config file is optional
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

func setDefaults(v *viper.Viper) {
	// Application defaults
	v.SetDefault("APP_PORT", 8080)
	v.SetDefault("APP_ENV", "development")
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_FORMAT", "json")

	// Database defaults
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", 5432)
	v.SetDefault("DB_NAME", "dockerauto")
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_PASSWORD", "password")
	v.SetDefault("DB_SSL_MODE", "disable")
	v.SetDefault("DB_DEBUG", false)
	v.SetDefault("DB_MAX_IDLE_CONNS", 10)
	v.SetDefault("DB_MAX_OPEN_CONNS", 100)
	v.SetDefault("DB_CONN_MAX_LIFETIME_MINUTES", 60)

	// Cache defaults
	v.SetDefault("CACHE_ENABLED", true)
	v.SetDefault("CACHE_DEFAULT_TTL_MINUTES", 30)
	v.SetDefault("CACHE_IMAGE_TTL_HOURS", 6)
	v.SetDefault("CACHE_CONFIG_TTL_MINUTES", 5)
	v.SetDefault("CACHE_CLEANUP_INTERVAL_MINUTES", 5)

	// JWT defaults (will be validated later)
	v.SetDefault("JWT_SECRET", "")
	v.SetDefault("JWT_EXPIRE_HOURS", 24)
	v.SetDefault("JWT_REFRESH_DAYS", 7)

	// Docker defaults
	v.SetDefault("DOCKER_HOST", "unix:///var/run/docker.sock")
	v.SetDefault("DOCKER_API_VERSION", "1.41")
	v.SetDefault("DOCKER_TIMEOUT", 30)
	v.SetDefault("DOCKER_VALIDATE_IMAGES", false)

	// Image check defaults
	v.SetDefault("DEFAULT_CHECK_INTERVAL", 60)
	v.SetDefault("MAX_CONCURRENT_CHECKS", 10)
	v.SetDefault("IMAGE_CACHE_HOURS", 6)

	// Notification defaults
	v.SetDefault("EMAIL_ENABLED", false)
	v.SetDefault("SMTP_PORT", 587)
	v.SetDefault("WEBHOOK_ENABLED", false)
	v.SetDefault("WECHAT_ENABLED", false)

	// Security defaults
	v.SetDefault("HTTPS_ENABLED", false)
	v.SetDefault("CORS_ALLOWED_ORIGINS", "*")
	v.SetDefault("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	v.SetDefault("CORS_ALLOWED_HEADERS", "Content-Type,Authorization,X-Requested-With")

	// System defaults
	v.SetDefault("MAX_LOG_RETENTION_DAYS", 30)
	v.SetDefault("MAX_UPDATE_HISTORY_COUNT", 1000)
	v.SetDefault("MAX_UPLOAD_SIZE_MB", 10)
	v.SetDefault("MAX_MEMORY_USAGE_PERCENT", 80)
	v.SetDefault("MAX_DISK_USAGE_PERCENT", 85)
	v.SetDefault("MAX_CPU_USAGE_PERCENT", 90)

	// Monitoring defaults
	v.SetDefault("PROMETHEUS_ENABLED", true)
	v.SetDefault("PROMETHEUS_PATH", "/metrics")
	v.SetDefault("HEALTH_CHECK_INTERVAL", 30)
	v.SetDefault("HEALTH_CHECK_TIMEOUT", 10)
}

func validate(config *Config) error {
	// Validate required fields
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	if config.JWT.Secret == "" {
		return fmt.Errorf("JWT secret is required (set JWT_SECRET environment variable)")
	}

	if len(config.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	if config.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}

	// Cache validation (optional since it's in-memory)
	if config.Cache.DefaultTTLMinutes <= 0 {
		config.Cache.DefaultTTLMinutes = 30
	}

	// Validate environment
	validEnvs := []string{"development", "production", "test"}
	if !contains(validEnvs, config.Environment) {
		return fmt.Errorf("invalid environment: %s, must be one of %v", config.Environment, validEnvs)
	}

	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}

// GetDSN returns database connection string for PostgreSQL
func (c *Config) GetDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// IsCacheEnabled returns true if caching is enabled
func (c *Config) IsCacheEnabled() bool {
	return c.Cache.Enabled
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return strings.EqualFold(c.Environment, "development")
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

// SchedulerConfig holds scheduler configuration
type SchedulerConfig struct {
	MaxConcurrentTasks int           `mapstructure:"SCHEDULER_MAX_CONCURRENT_TASKS"`
	TaskTimeout        time.Duration `mapstructure:"SCHEDULER_TASK_TIMEOUT"`
	RetryDelay         time.Duration `mapstructure:"SCHEDULER_RETRY_DELAY"`
	MaxRetries         int           `mapstructure:"SCHEDULER_MAX_RETRIES"`
	CleanupInterval    time.Duration `mapstructure:"SCHEDULER_CLEANUP_INTERVAL"`
	HistoryRetention   time.Duration `mapstructure:"SCHEDULER_HISTORY_RETENTION"`
	LogLevel           string        `mapstructure:"SCHEDULER_LOG_LEVEL"`
	EnableMetrics      bool          `mapstructure:"SCHEDULER_ENABLE_METRICS"`
	TimeZone           string        `mapstructure:"SCHEDULER_TIMEZONE"`
}