package config

import (
	"fmt"
	"os"

	"github.com/n0needt0/go-goodies/log"
	"gopkg.in/yaml.v3"
)

type Config struct {
	App          AppConfig          `yaml:"app"`
	Logging      LoggingConfig      `yaml:"logging"`
	Server       ServerConfig       `yaml:"server"`
	Services     ServicesConfig     `yaml:"services"`
	Database     DatabaseConfig     `yaml:"database"`
	Otel         OtelConfig         `yaml:"otel"`
	Housekeeping HousekeepingConfig `yaml:"housekeeping"`
	Auth         AuthConfig         `yaml:"auth"`
	RateLimit    RateLimitConfig    `yaml:"rate_limit"`
	Dev          bool               `yaml:"dev"`

	// Initialized components (set at runtime)
	DatabaseService  interface{} `yaml:"-"`
	EcosystemMonitor interface{} `yaml:"-"`
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

type ServicesConfig struct {
	Receiver ServiceEndpoint `yaml:"receiver"`
	Proxy    ServiceEndpoint `yaml:"proxy"`
	SOC      ServiceEndpoint `yaml:"soc"`
	Packer   ServiceEndpoint `yaml:"packer"`
}

type ServiceEndpoint struct {
	URL            string `yaml:"url"`
	HealthEndpoint string `yaml:"health_endpoint"`
	ConfigEndpoint string `yaml:"config_endpoint"`
	TimeoutSeconds int    `yaml:"timeout_seconds"`
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

type OtelConfig struct {
	Enabled               bool   `yaml:"enabled"`
	Endpoint              string `yaml:"endpoint"`
	ServiceName           string `yaml:"service_name"`
	ScrapeIntervalSeconds int    `yaml:"scrape_interval_seconds"`
}

type HousekeepingConfig struct {
	Enabled         bool `yaml:"enabled"`
	IntervalSeconds int  `yaml:"interval_seconds"`
}

type AuthConfig struct {
	Enabled          bool     `yaml:"enabled"`
	JWTSecret        string   `yaml:"jwt_secret"`
	TokenExpiryHours int      `yaml:"token_expiry_hours"`
	AdminUsers       []string `yaml:"admin_users"`
}

type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
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
