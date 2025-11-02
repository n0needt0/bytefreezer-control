package config

import (
	"fmt"
	"os"

	"github.com/n0needt0/go-goodies/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App            AppConfig            `yaml:"app"`
	Logging        LoggingConfig        `yaml:"logging"`
	Server         ServerConfig         `yaml:"server"`
	Database       DatabaseConfig       `yaml:"database"`
	S3             S3Config             `yaml:"s3"`
	Otel           OtelConfig           `yaml:"otel"`
	Housekeeping   HousekeepingConfig   `yaml:"housekeeping"`
	Auth           AuthConfig           `yaml:"auth"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit"`
	HealthReporting HealthReportingConfig `yaml:"health_reporting"`
	Services       ServicesConfig       `yaml:"services"`
	Dev            bool                 `yaml:"dev"`

	// Initialized components (set at runtime)
	DatabaseService interface{} `yaml:"-"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type LoggingConfig struct {
	Level    string `yaml:"level"`
	Encoding string `yaml:"encoding"`
}

type ServerConfig struct {
	ApiPort int `yaml:"api_port"`
}

type DatabaseConfig struct {
	Enabled            bool   `yaml:"enabled"`
	Type               string `yaml:"type"`
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	Database           string `yaml:"database"`
	Username           string `yaml:"username"`
	Password           string `yaml:"password"`
	SSLMode            string `yaml:"ssl_mode"`
	MaxConnections     int    `yaml:"max_connections"`
	MaxIdleConnections int    `yaml:"max_idle_connections"`
}

type S3Config struct {
	Enabled      bool   `yaml:"enabled"`
	IntakeBucket string `yaml:"intake_bucket"`
	PiperBucket  string `yaml:"piper_bucket"`
	Region       string `yaml:"region"`
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"access_key"`
	SecretKey    string `yaml:"secret_key"`
	UseSSL       bool   `yaml:"use_ssl"`
}

type OtelConfig struct {
	Enabled               bool   `yaml:"enabled"`
	ServiceName           string `yaml:"service_name"`
	ScrapeIntervalSeconds int    `yaml:"scrape_interval_seconds"`
	MetricsHost           string `yaml:"metrics_host"`
	MetricsPort           int    `yaml:"metrics_port"`
}

type HousekeepingConfig struct {
	Enabled         bool `yaml:"enabled"`
	IntervalSeconds int  `yaml:"interval_seconds"`
}

type AuthConfig struct {
	JWTSecret        string `yaml:"jwt_secret"`
	TokenExpiryHours int    `yaml:"token_expiry_hours"`
	ServiceAPIKey    string `yaml:"service_api_key"` // API key for service-to-service auth (piper, packer)
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
}

type HealthReportingConfig struct {
	Enabled           bool   `yaml:"enabled"`
	ControlURL        string `yaml:"control_url"`
	ReportInterval    int    `yaml:"report_interval"`    // Interval in seconds
	TimeoutSeconds    int    `yaml:"timeout_seconds"`
	RegisterOnStartup bool   `yaml:"register_on_startup"`
}

type ServicesConfig struct {
	PiperURL    string `yaml:"piper_url"`    // Piper service URL for transformation proxying
	ReceiverURL string `yaml:"receiver_url"` // Receiver service URL for health checks
	PackerURL   string `yaml:"packer_url"`   // Packer service URL for health checks (optional)
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configFile, envPrefix string, config *Config) error {
	// Load from YAML file
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// TODO: Override with environment variables if needed
	// This can be implemented similar to bytefreezer-receiver

	log.Debugf("Loaded configuration from %s", configFile)
	return nil
}

// InitializeComponents initializes all configuration-dependent components
func (c *Config) InitializeComponents() error {
	// Initialize database connection if enabled
	if c.Database.Enabled {
		// TODO: Initialize database connection
		log.Info("Database connection would be initialized here")
	}

	// Initialize ecosystem monitor
	// TODO: Initialize ecosystem monitoring service
	log.Info("Ecosystem monitor would be initialized here")

	return nil
}
