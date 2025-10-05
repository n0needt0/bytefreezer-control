package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
)

// Services contains all service instances
type Services struct {
	Config           *config.Config
	Storage          storage.Storage // New storage layer
	Database         DatabaseService // Legacy - to be deprecated
	EcosystemMonitor EcosystemMonitor
	Stats            *ControlStats
	mutex            sync.RWMutex
}

// ControlStats tracks control service statistics
type ControlStats struct {
	StartTime             time.Time
	UptimeSeconds         int64
	APIRequests           int64
	ServicesMonitored     int64
	HealthChecksPerformed int64
	ConfigUpdates         int64
	DatabaseQueries       int64
	LastActivity          time.Time
	mutex                 sync.RWMutex
}

// DatabaseService interface for database operations
type DatabaseService interface {
	Connect() error
	Close() error
	Ping() error
	// Add tenant management methods
	GetTenants() ([]Tenant, error)
	GetTenant(id string) (*Tenant, error)
	CreateTenant(tenant *Tenant) error
	UpdateTenant(tenant *Tenant) error
	DeleteTenant(id string) error
}

// EcosystemMonitor interface for monitoring ByteFreezer services
type EcosystemMonitor interface {
	CheckAllServices() error
	CheckService(serviceName string) (*ServiceStatus, error)
	GetServiceStatuses() map[string]*ServiceStatus
}

// ServiceStatus represents the status of a ByteFreezer service
type ServiceStatus struct {
	Name         string                 `json:"name"`
	URL          string                 `json:"url"`
	Healthy      bool                   `json:"healthy"`
	LastCheck    time.Time              `json:"last_check"`
	ResponseTime time.Duration          `json:"response_time"`
	Version      string                 `json:"version"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// Tenant represents a tenant in the system
type Tenant struct {
	ID                   string                 `json:"id" db:"id"`
	Name                 string                 `json:"name" db:"name"`
	Email                string                 `json:"email" db:"email"`
	Active               bool                   `json:"active" db:"active"`
	CreatedAt            time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at" db:"updated_at"`
	Datasets             []Dataset              `json:"datasets"`
	NotificationSettings map[string]interface{} `json:"notification_settings" db:"notification_settings"`
}

// Dataset represents a dataset within a tenant
type Dataset struct {
	ID          string                 `json:"id" db:"id"`
	TenantID    string                 `json:"tenant_id" db:"tenant_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Active      bool                   `json:"active" db:"active"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	Pipeline    map[string]interface{} `json:"pipeline" db:"pipeline"`
}

// NewServices creates a new services instance
func NewServices(config *config.Config) *Services {
	services := &Services{
		Config: config,
		Stats: &ControlStats{
			StartTime:    time.Now(),
			LastActivity: time.Now(),
		},
	}

	// Initialize storage layer if database is enabled
	if config.Database.Enabled {
		// Build PostgreSQL connection URI
		uri := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Database.Host,
			config.Database.Port,
			config.Database.Username,
			config.Database.Password,
			config.Database.Database,
			config.Database.SSLMode)

		storageConfig := storage.Config{
			Type:           config.Database.Type,
			URI:            uri,
			Database:       config.Database.Database,
			TimeoutSeconds: 10, // Default timeout
			SSLMode:        config.Database.SSLMode,
		}

		storageInstance, err := storage.NewStorage(storageConfig)
		if err != nil {
			log.Warnf("Failed to initialize storage: %v", err)
		} else {
			services.Storage = storageInstance
			log.Info("Storage layer initialized successfully")

			// Run migrations
			ctx := context.Background()
			if err := storageInstance.Migrate(ctx); err != nil {
				log.Warnf("Failed to run migrations: %v", err)
			} else {
				log.Info("Database migrations completed successfully")
			}
		}

		// Legacy database service for backward compatibility
		dbService := NewDatabaseService(&config.Database)
		if err := dbService.Connect(); err != nil {
			log.Warnf("Failed to connect to database: %v", err)
		}
		services.Database = dbService
	}

	// Initialize ecosystem monitor
	monitor := NewEcosystemMonitor(config)
	services.EcosystemMonitor = monitor

	return services
}

// GetStats returns current statistics
func (s *Services) GetStats() *ControlStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	s.Stats.mutex.RLock()
	defer s.Stats.mutex.RUnlock()

	// Create a copy to avoid race conditions
	return &ControlStats{
		StartTime:             s.Stats.StartTime,
		UptimeSeconds:         s.Stats.UptimeSeconds,
		APIRequests:           s.Stats.APIRequests,
		ServicesMonitored:     s.Stats.ServicesMonitored,
		HealthChecksPerformed: s.Stats.HealthChecksPerformed,
		ConfigUpdates:         s.Stats.ConfigUpdates,
		DatabaseQueries:       s.Stats.DatabaseQueries,
		LastActivity:          s.Stats.LastActivity,
	}
}

// IncrementAPIRequests increments the API requests counter
func (s *Services) IncrementAPIRequests() {
	s.Stats.mutex.Lock()
	defer s.Stats.mutex.Unlock()

	s.Stats.APIRequests++
	s.Stats.LastActivity = time.Now()
}

// IncrementHealthChecks increments the health checks counter
func (s *Services) IncrementHealthChecks() {
	s.Stats.mutex.Lock()
	defer s.Stats.mutex.Unlock()

	s.Stats.HealthChecksPerformed++
}

// IncrementConfigUpdates increments the config updates counter
func (s *Services) IncrementConfigUpdates() {
	s.Stats.mutex.Lock()
	defer s.Stats.mutex.Unlock()

	s.Stats.ConfigUpdates++
	s.Stats.LastActivity = time.Now()
}

// IncrementDatabaseQueries increments the database queries counter
func (s *Services) IncrementDatabaseQueries() {
	s.Stats.mutex.Lock()
	defer s.Stats.mutex.Unlock()

	s.Stats.DatabaseQueries++
}
