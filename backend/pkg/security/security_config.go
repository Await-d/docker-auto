package security

import (
	"crypto/tls"
	"fmt"
	"time"
)

// ValidationConfig represents input validation configuration
type ValidationConfig struct {
	MaxLength         int      `json:"max_length"`
	AllowedPatterns   []string `json:"allowed_patterns"`
	BlockedPatterns   []string `json:"blocked_patterns"`
	StrictMode        bool     `json:"strict_mode"`
	EnableXSSProtection bool   `json:"enable_xss_protection"`
	EnableSQLInjectionProtection bool `json:"enable_sql_injection_protection"`
	MaxRequestSize    int      `json:"max_request_size"`
	AllowedMimeTypes  []string `json:"allowed_mime_types"`
	SQLInjectionProtection bool `json:"sql_injection_protection"`
	XSSProtection     bool     `json:"xss_protection"`
	PathTraversalProtection bool `json:"path_traversal_protection"`
	CommandInjectionProtection bool `json:"command_injection_protection"`
	InputSanitization bool     `json:"input_sanitization"`
}

// SecurityConfig represents security middleware configuration
type SecurityConfig struct {
	EnableHTTPS       bool              `json:"enable_https"`
	HSTSMaxAge        int               `json:"hsts_max_age"`
	ContentSecurityPolicy string        `json:"content_security_policy"`
	XFrameOptions     string            `json:"x_frame_options"`
	XContentTypeOptions string          `json:"x_content_type_options"`
	ReferrerPolicy    string            `json:"referrer_policy"`
	TrustedProxies    []string          `json:"trusted_proxies"`
	CORSConfig        *CORSConfig       `json:"cors_config"`
	ForceHTTPS        bool              `json:"force_https"`
	HTTPSRedirect     bool              `json:"https_redirect"`
	EnableHSTS        bool              `json:"enable_hsts"`
	AllowInsecureLocal bool             `json:"allow_insecure_local"`
}

// CORSConfig represents CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string `json:"allowed_origins"`
	AllowedMethods   []string `json:"allowed_methods"`
	AllowedHeaders   []string `json:"allowed_headers"`
	ExposedHeaders   []string `json:"exposed_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"`
}

// SecurityMasterConfig represents the master security configuration
type SecurityMasterConfig struct {
	// Environment settings
	Environment    string `json:"environment"`    // development, staging, production
	SecurityLevel  int    `json:"security_level"` // 1=low, 2=medium, 3=high

	// JWT Configuration
	JWT            *JWTConfig              `json:"jwt"`

	// Rate Limiting Configuration
	RateLimit      *EnhancedRateLimitConfig `json:"rate_limit"`

	// Input Validation Configuration
	Validation     *ValidationConfig       `json:"validation"`

	// Security Middleware Configuration
	Security       *SecurityConfig         `json:"security"`

	// WebSocket Security Configuration
	WebSocket      *WebSocketSecurityConfig `json:"websocket"`

	// Database Security Configuration
	Database       *DatabaseSecurityConfig  `json:"database"`

	// Docker Security Configuration
	Docker         *DockerSecurityConfig    `json:"docker"`

	// TLS/SSL Configuration
	TLS            *TLSConfig              `json:"tls"`

	// Monitoring Configuration
	Monitoring     *MonitoringConfig       `json:"monitoring"`

	// Compliance Configuration
	Compliance     *ComplianceConfig       `json:"compliance"`
}

// TLSConfig represents TLS/SSL configuration
type TLSConfig struct {
	Enabled            bool              `json:"enabled"`
	MinVersion         uint16            `json:"min_version"`
	MaxVersion         uint16            `json:"max_version"`
	CipherSuites       []uint16          `json:"cipher_suites"`
	PreferServerCiphers bool             `json:"prefer_server_ciphers"`
	SessionTicketsDisabled bool          `json:"session_tickets_disabled"`
	InsecureSkipVerify bool              `json:"insecure_skip_verify"`

	// Certificate settings
	CertFile           string            `json:"cert_file"`
	KeyFile            string            `json:"key_file"`
	CAFile             string            `json:"ca_file"`
	ClientAuth         tls.ClientAuthType `json:"client_auth"`

	// HSTS settings
	HSTSEnabled        bool              `json:"hsts_enabled"`
	HSTSMaxAge         int               `json:"hsts_max_age"`
	HSTSIncludeSubdomains bool           `json:"hsts_include_subdomains"`
	HSTSPreload        bool              `json:"hsts_preload"`

	// Certificate rotation
	AutoRotation       bool              `json:"auto_rotation"`
	RotationDays       int               `json:"rotation_days"`

	// Additional security headers
	EnableCSP             bool              `json:"enable_csp"`
	CSPDirectives         string            `json:"csp_directives"`
	EnableXFrameOptions   bool              `json:"enable_x_frame_options"`
	XFrameOptions         string            `json:"x_frame_options"`
	EnableXSSProtection   bool              `json:"enable_xss_protection"`
	EnableContentTypeOptions bool           `json:"enable_content_type_options"`
	EnableReferrerPolicy  bool              `json:"enable_referrer_policy"`
	ReferrerPolicy        string            `json:"referrer_policy"`
}

// MonitoringConfig represents security monitoring configuration
type MonitoringConfig struct {
	Enabled              bool          `json:"enabled"`
	LogLevel             string        `json:"log_level"`

	// Intrusion Detection
	IntrusionDetection   bool          `json:"intrusion_detection"`
	SuspiciousThreshold  int           `json:"suspicious_threshold"`
	BlockSuspicious      bool          `json:"block_suspicious"`

	// Alert settings
	AlertsEnabled        bool          `json:"alerts_enabled"`
	AlertWebhook         string        `json:"alert_webhook"`
	AlertEmail           string        `json:"alert_email"`

	// Metrics collection
	MetricsEnabled       bool          `json:"metrics_enabled"`
	MetricsInterval      time.Duration `json:"metrics_interval"`

	// Log retention
	LogRetentionDays     int           `json:"log_retention_days"`
	CompressLogs         bool          `json:"compress_logs"`
}

// ComplianceConfig represents compliance configuration
type ComplianceConfig struct {
	// Standards compliance
	SOC2Enabled      bool   `json:"soc2_enabled"`
	ISO27001Enabled  bool   `json:"iso27001_enabled"`
	PCIDSSEnabled    bool   `json:"pci_dss_enabled"`
	HIPAAEnabled     bool   `json:"hipaa_enabled"`
	GDPREnabled      bool   `json:"gdpr_enabled"`

	// Audit settings
	AuditLogging     bool   `json:"audit_logging"`
	AuditLevel       string `json:"audit_level"`
	AuditRetention   int    `json:"audit_retention_days"`

	// Data protection
	DataEncryption   bool   `json:"data_encryption"`
	PIIProtection    bool   `json:"pii_protection"`
	DataMasking      bool   `json:"data_masking"`
}

// GetSecurityConfig returns security configuration based on environment
func GetSecurityConfig(env string) *SecurityMasterConfig {
	switch env {
	case "production":
		return getProductionConfig()
	case "staging":
		return getStagingConfig()
	case "development":
		return getDevelopmentConfig()
	default:
		return getDevelopmentConfig()
	}
}

// getProductionConfig returns production security configuration
func getProductionConfig() *SecurityMasterConfig {
	return &SecurityMasterConfig{
		Environment:   "production",
		SecurityLevel: 3,

		JWT: &JWTConfig{
			AccessTokenTTL:    15 * time.Minute,
			RefreshTokenTTL:   7 * 24 * time.Hour,
			TokenRotation:     true,
			BlacklistEnabled:  true,
			SecureHeaders:     true,
			IssuerName:        "docker-auto-prod",
			MaxTokenAge:       24 * time.Hour,
			RotationThreshold: 5 * time.Minute,
			CleanupInterval:   time.Hour,
		},

		RateLimit: &EnhancedRateLimitConfig{
			GlobalLimit:         5000,
			GlobalWindow:        time.Hour,
			UserLimit:          500,
			UserWindow:         time.Hour,
			IPLimit:            50,
			IPWindow:           time.Minute,
			SubnetLimit:        500,
			SubnetWindow:       time.Hour,
			EnableBanning:      true,
			BanThreshold:       10,
			BanDuration:        time.Hour,
			MaxBanDuration:     24 * time.Hour,
			EnableDynamicLimits: true,
			LoadThreshold:      0.8,
			DynamicMultiplier:  0.5,
		},

		Validation: &ValidationConfig{
			MaxRequestSize:             5 * 1024 * 1024, // 5MB
			AllowedMimeTypes:          []string{"application/json", "application/x-www-form-urlencoded"},
			SQLInjectionProtection:    true,
			XSSProtection:             true,
			PathTraversalProtection:   true,
			CommandInjectionProtection: true,
			InputSanitization:         true,
		},

		Security: &SecurityConfig{
			ForceHTTPS:            true,
			HTTPSRedirect:         true,
			EnableHSTS:            true,
			HSTSMaxAge:           31536000, // 1 year
		},

		WebSocket: &WebSocketSecurityConfig{
			RequireAuth:           true,
			TokenValidation:       true,
			SessionValidation:     true,
			OriginValidation:      true,
			StrictOriginCheck:     true,
			MaxConnections:        500,
			MaxConnectionsPerIP:   5,
			MaxConnectionsPerUser: 3,
			MessageValidation:     true,
			MaxMessageSize:        32 * 1024, // 32KB
			EnableRateLimit:       true,
			MessagesPerMinute:     30,
			HeartbeatInterval:     30 * time.Second,
			ConnectionTimeout:     30 * time.Second,
			IdleTimeout:           300 * time.Second,
		},

		Database: &DatabaseSecurityConfig{
			TLSEnabled:         true,
			PreparedStatements: true,
			QueryTimeout:       30 * time.Second,
			MaxQueryLength:     5000,
			AuditEnabled:       true,
			AuditLevel:         AuditAll,
			LogSlowQueries:     true,
			SlowQueryThreshold: 1 * time.Second,
			EncryptionEnabled:  true,
			MaxOpenConns:       15,
			MaxIdleConns:       3,
			ConnMaxLifetime:    5 * time.Minute,
		},

		Docker: &DockerSecurityConfig{
			TLSEnabled:         true,
			UserNamespacing:    true,
			RequireAuth:        true,
			ReadOnlyRootFS:     true,
			NoNewPrivileges:    true,
			ImageScanning:      true,
			SignedImagesOnly:   true,
			AuditEnabled:       true,
			MonitorContainers:  true,
			AlertOnSuspicious:  true,
			VulnerabilityThreshold: VulnMedium,
			ResourceLimits: ResourceLimits{
				CPULimit:     500000000, // 0.5 CPU
				MemoryLimit:  268435456, // 256MB
				PIDsLimit:    50,
				ULimitNoFile: 512,
				ULimitNProc:  32,
			},
		},

		TLS: &TLSConfig{
			Enabled:                true,
			MinVersion:             tls.VersionTLS12,
			MaxVersion:             tls.VersionTLS13,
			PreferServerCiphers:    true,
			SessionTicketsDisabled: true,
			ClientAuth:             tls.RequireAndVerifyClientCert,
			HSTSEnabled:            true,
			HSTSMaxAge:            31536000,
			HSTSIncludeSubdomains: true,
			HSTSPreload:           true,
			AutoRotation:          true,
			RotationDays:          30,
			EnableCSP:             true,
			CSPDirectives:         "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'",
			EnableXFrameOptions:   true,
			XFrameOptions:         "DENY",
			EnableXSSProtection:   true,
			EnableContentTypeOptions: true,
			EnableReferrerPolicy: true,
			ReferrerPolicy:       "strict-origin-when-cross-origin",
		},

		Monitoring: &MonitoringConfig{
			Enabled:             true,
			LogLevel:            "info",
			IntrusionDetection:  true,
			SuspiciousThreshold: 5,
			BlockSuspicious:     true,
			AlertsEnabled:       true,
			MetricsEnabled:      true,
			MetricsInterval:     time.Minute,
			LogRetentionDays:    90,
			CompressLogs:        true,
		},

		Compliance: &ComplianceConfig{
			SOC2Enabled:      true,
			ISO27001Enabled:  true,
			AuditLogging:     true,
			AuditLevel:       "full",
			AuditRetention:   365,
			DataEncryption:   true,
			PIIProtection:    true,
			DataMasking:      true,
		},
	}
}

// getStagingConfig returns staging security configuration
func getStagingConfig() *SecurityMasterConfig {
	config := getProductionConfig()
	config.Environment = "staging"
	config.SecurityLevel = 2

	// Relax some settings for staging
	config.JWT.AccessTokenTTL = 30 * time.Minute
	config.JWT.TokenRotation = false
	config.Security.ForceHTTPS = false
	config.Docker.SignedImagesOnly = false
	config.TLS.ClientAuth = tls.VerifyClientCertIfGiven
	config.Monitoring.IntrusionDetection = false

	return config
}

// getDevelopmentConfig returns development security configuration
func getDevelopmentConfig() *SecurityMasterConfig {
	config := getStagingConfig()
	config.Environment = "development"
	config.SecurityLevel = 1

	// Further relax settings for development
	config.JWT.AccessTokenTTL = 2 * time.Hour
	config.JWT.BlacklistEnabled = false
	config.RateLimit.EnableBanning = false
	config.Security.AllowInsecureLocal = true
	config.WebSocket.RequireAuth = false
	config.Database.TLSEnabled = false
	config.Docker.TLSEnabled = false
	config.Docker.ImageScanning = false
	config.TLS.Enabled = false
	config.Monitoring.LogLevel = "debug"
	config.Compliance.AuditLogging = false

	return config
}

// ValidateConfig validates the security configuration
func ValidateConfig(config *SecurityMasterConfig) error {
	if config.Environment == "" {
		return fmt.Errorf("environment must be specified")
	}

	if config.SecurityLevel < 1 || config.SecurityLevel > 3 {
		return fmt.Errorf("security level must be between 1 and 3")
	}

	// Validate JWT configuration
	if config.JWT != nil {
		if config.JWT.SecretKey == "" {
			return fmt.Errorf("JWT secret key is required")
		}
		if config.JWT.AccessTokenTTL <= 0 {
			return fmt.Errorf("JWT access token TTL must be positive")
		}
	}

	// Validate TLS configuration
	if config.TLS != nil && config.TLS.Enabled {
		if config.TLS.CertFile == "" || config.TLS.KeyFile == "" {
			return fmt.Errorf("TLS certificate and key files are required when TLS is enabled")
		}
	}

	// Production-specific validations
	if config.Environment == "production" {
		if config.SecurityLevel < 2 {
			return fmt.Errorf("production environment requires security level 2 or higher")
		}

		if config.TLS != nil && !config.TLS.Enabled {
			return fmt.Errorf("TLS must be enabled in production")
		}

		if config.JWT != nil && !config.JWT.BlacklistEnabled {
			return fmt.Errorf("JWT blacklisting must be enabled in production")
		}
	}

	return nil
}