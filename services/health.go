package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/n0needt0/go-goodies/log"
)

type HealthService struct {
	db     *sql.DB
	mutex  sync.RWMutex
	client *http.Client
}

type ServiceRegistration struct {
	ServiceType   string                 `json:"service_type"`
	InstanceID    string                 `json:"instance_id"`
	InstanceAPI   string                 `json:"instance_api"`
	Status        string                 `json:"status"`
	Configuration map[string]interface{} `json:"configuration"`
	Timestamp     time.Time              `json:"timestamp"`
}

type HealthRecord struct {
	ID             int                    `json:"id"`
	ServiceType    string                 `json:"service_type"`
	InstanceID     string                 `json:"instance_id"`
	InstanceAPI    string                 `json:"instance_api"`
	Status         string                 `json:"status"`
	Configuration  map[string]interface{} `json:"configuration"`
	Metrics        map[string]interface{} `json:"metrics"`
	ResponseTimeMs *int                   `json:"response_time_ms"`
	LastSeen       time.Time              `json:"last_seen"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

func NewHealthService(db *sql.DB) *HealthService {
	return &HealthService{
		db: db,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// RegisterService registers a new service instance
func (h *HealthService) RegisterService(registration ServiceRegistration) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Get hostname for instance_id if not provided
	if registration.InstanceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("failed to get hostname: %w", err)
		}
		registration.InstanceID = hostname
	}

	// Set initial status to Starting
	if registration.Status == "" {
		registration.Status = "Starting"
	}

	configJson, err := json.Marshal(registration.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	log.Infof("Registering service %s instance %s - config JSON: %s",
		registration.ServiceType, registration.InstanceID, string(configJson))

	_, err = h.db.Exec(`
		SELECT upsert_health_current($1, $2, $3, $4, $5, NULL, NULL)`,
		registration.ServiceType,
		registration.InstanceID,
		registration.InstanceAPI,
		registration.Status,
		configJson,
	)

	if err != nil {
		log.Errorf("Failed to execute upsert_health_current for %s:%s: %v",
			registration.ServiceType, registration.InstanceID, err)
		return fmt.Errorf("failed to register service: %w", err)
	}

	log.Infof("Successfully registered service %s instance %s at %s",
		registration.ServiceType, registration.InstanceID, registration.InstanceAPI)

	return nil
}

// UpdateServiceHealth updates health status for a service instance
func (h *HealthService) UpdateServiceHealth(serviceType, instanceID, status string, metrics map[string]interface{}, responseTimeMs *int) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	var metricsValue interface{}
	if metrics != nil {
		metricsJson, err := json.Marshal(metrics)
		if err != nil {
			log.Errorf("Failed to marshal metrics for %s:%s: %v", serviceType, instanceID, err)
			return fmt.Errorf("failed to marshal metrics: %w", err)
		}
		metricsValue = metricsJson
		log.Infof("Updating health for %s:%s - metrics JSON: %s", serviceType, instanceID, string(metricsJson))
	} else {
		// Use NULL for nil metrics instead of nil byte slice
		metricsValue = nil
		log.Infof("Updating health for %s:%s - no metrics (nil)", serviceType, instanceID)
	}

	result, err := h.db.Exec(`
		UPDATE health_current
		SET status = $1, metrics = $2, response_time_ms = $3, last_seen = NOW(), updated_at = NOW()
		WHERE service_type = $4 AND instance_id = $5`,
		status, metricsValue, responseTimeMs, serviceType, instanceID)

	if err != nil {
		log.Errorf("Database UPDATE failed for %s:%s: %v", serviceType, instanceID, err)
		return fmt.Errorf("failed to update service health: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Warnf("Could not get rows affected for %s:%s: %v", serviceType, instanceID, err)
	} else {
		log.Debugf("Updated %d rows for %s:%s (status: %s)", rowsAffected, serviceType, instanceID, status)
		if rowsAffected == 0 {
			log.Warnf("No rows updated for %s:%s - record may not exist in health_current table", serviceType, instanceID)
		}
	}

	return nil
}

// GetAllHealthRecords returns all current health records
func (h *HealthService) GetAllHealthRecords() ([]HealthRecord, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	rows, err := h.db.Query(`
		SELECT id, service_type, instance_id, instance_api, status,
		       configuration, metrics, response_time_ms, last_seen, created_at, updated_at
		FROM health_current
		ORDER BY service_type, instance_id`)

	if err != nil {
		return nil, fmt.Errorf("failed to query health records: %w", err)
	}
	defer rows.Close()

	var records []HealthRecord
	for rows.Next() {
		var record HealthRecord
		var configJson, metricsJson sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.ServiceType,
			&record.InstanceID,
			&record.InstanceAPI,
			&record.Status,
			&configJson,
			&metricsJson,
			&record.ResponseTimeMs,
			&record.LastSeen,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health record: %w", err)
		}

		// Parse JSON fields
		if configJson.Valid {
			if err := json.Unmarshal([]byte(configJson.String), &record.Configuration); err != nil {
				log.Warnf("Failed to unmarshal configuration for %s:%s: %v",
					record.ServiceType, record.InstanceID, err)
			}
		}

		if metricsJson.Valid {
			if err := json.Unmarshal([]byte(metricsJson.String), &record.Metrics); err != nil {
				log.Warnf("Failed to unmarshal metrics for %s:%s: %v",
					record.ServiceType, record.InstanceID, err)
			}
		}

		records = append(records, record)
	}

	return records, nil
}

// GetHealthRecordsByService returns health records for a specific service type
func (h *HealthService) GetHealthRecordsByService(serviceType string) ([]HealthRecord, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	rows, err := h.db.Query(`
		SELECT id, service_type, instance_id, instance_api, status,
		       configuration, metrics, response_time_ms, last_seen, created_at, updated_at
		FROM health_current
		WHERE service_type = $1
		ORDER BY instance_id`, serviceType)

	if err != nil {
		return nil, fmt.Errorf("failed to query health records for service %s: %w", serviceType, err)
	}
	defer rows.Close()

	var records []HealthRecord
	for rows.Next() {
		var record HealthRecord
		var configJson, metricsJson sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.ServiceType,
			&record.InstanceID,
			&record.InstanceAPI,
			&record.Status,
			&configJson,
			&metricsJson,
			&record.ResponseTimeMs,
			&record.LastSeen,
			&record.CreatedAt,
			&record.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan health record: %w", err)
		}

		// Parse JSON fields
		if configJson.Valid {
			if err := json.Unmarshal([]byte(configJson.String), &record.Configuration); err != nil {
				log.Warnf("Failed to unmarshal configuration for %s:%s: %v",
					record.ServiceType, record.InstanceID, err)
			}
		}

		if metricsJson.Valid {
			if err := json.Unmarshal([]byte(metricsJson.String), &record.Metrics); err != nil {
				log.Warnf("Failed to unmarshal metrics for %s:%s: %v",
					record.ServiceType, record.InstanceID, err)
			}
		}

		records = append(records, record)
	}

	return records, nil
}

// PollServicesHealth polls all registered services for their health status
func (h *HealthService) PollServicesHealth() error {
	records, err := h.GetAllHealthRecords()
	if err != nil {
		return fmt.Errorf("failed to get health records: %w", err)
	}

	for _, record := range records {
		go h.pollSingleService(record)
	}

	return nil
}

// pollSingleService polls a single service's health endpoint
func (h *HealthService) pollSingleService(record HealthRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	healthURL := fmt.Sprintf("http://%s/api/v1/health", record.InstanceAPI)

	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		log.Warnf("Failed to create health check request for %s:%s: %v",
			record.ServiceType, record.InstanceID, err)
		h.UpdateServiceHealth(record.ServiceType, record.InstanceID, "Unhealthy", nil, nil)
		return
	}

	resp, err := h.client.Do(req)
	responseTime := int(time.Since(start).Milliseconds())

	if err != nil {
		log.Debugf("Health check failed for %s:%s: %v",
			record.ServiceType, record.InstanceID, err)
		h.UpdateServiceHealth(record.ServiceType, record.InstanceID, "Unhealthy", nil, &responseTime)
		return
	}
	defer resp.Body.Close()

	status := "Unhealthy"
	if resp.StatusCode == 200 {
		status = "Healthy"
	}

	// Try to parse response for metrics
	var metrics map[string]interface{}
	if resp.StatusCode == 200 {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(&metrics); err != nil {
			log.Debugf("Failed to decode health response for %s:%s: %v",
				record.ServiceType, record.InstanceID, err)
		}
	}

	err = h.UpdateServiceHealth(record.ServiceType, record.InstanceID, status, metrics, &responseTime)
	if err != nil {
		log.Warnf("Failed to update health status for %s:%s: %v",
			record.ServiceType, record.InstanceID, err)
	} else {
		log.Debugf("Health check completed for %s:%s - status: %s, response time: %dms",
			record.ServiceType, record.InstanceID, status, responseTime)
	}
}

// CleanupStaleRecords moves stale records to history and cleans up old history
func (h *HealthService) CleanupStaleRecords() error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// Move stale records to history
	var movedCount int
	err := h.db.QueryRow("SELECT move_stale_health_to_history()").Scan(&movedCount)
	if err != nil {
		return fmt.Errorf("failed to move stale records to history: %w", err)
	}

	if movedCount > 0 {
		log.Infof("Moved %d stale health records to history", movedCount)
	}

	// Cleanup old history records
	var deletedCount int
	err = h.db.QueryRow("SELECT cleanup_old_health_history()").Scan(&deletedCount)
	if err != nil {
		return fmt.Errorf("failed to cleanup old history records: %w", err)
	}

	if deletedCount > 0 {
		log.Infof("Cleaned up %d old health history records", deletedCount)
	}

	return nil
}

// GetHealthSummary returns summary statistics for the health dashboard
func (h *HealthService) GetHealthSummary() (map[string]interface{}, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	summary := make(map[string]interface{})

	// Count total services
	var totalServices, healthyServices, unhealthyServices int
	err := h.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'Healthy' THEN 1 END) as healthy,
			COUNT(CASE WHEN status = 'Unhealthy' THEN 1 END) as unhealthy
		FROM health_current`).Scan(&totalServices, &healthyServices, &unhealthyServices)

	if err != nil {
		return nil, fmt.Errorf("failed to get health summary: %w", err)
	}

	summary["total_services"] = totalServices
	summary["healthy_services"] = healthyServices
	summary["unhealthy_services"] = unhealthyServices

	// Get service type breakdown
	rows, err := h.db.Query(`
		SELECT service_type, COUNT(*) as count,
		       COUNT(CASE WHEN status = 'Healthy' THEN 1 END) as healthy_count
		FROM health_current
		GROUP BY service_type
		ORDER BY service_type`)

	if err != nil {
		return nil, fmt.Errorf("failed to get service breakdown: %w", err)
	}
	defer rows.Close()

	serviceBreakdown := make(map[string]map[string]int)
	for rows.Next() {
		var serviceType string
		var total, healthy int
		if err := rows.Scan(&serviceType, &total, &healthy); err != nil {
			return nil, fmt.Errorf("failed to scan service breakdown: %w", err)
		}
		serviceBreakdown[serviceType] = map[string]int{
			"total":   total,
			"healthy": healthy,
		}
	}

	summary["service_breakdown"] = serviceBreakdown
	summary["timestamp"] = time.Now()

	return summary, nil
}

// RegisterSelf registers the control service itself in the health system
func (h *HealthService) RegisterSelf(serviceType, instanceAPI string, config map[string]interface{}) error {
	registration := ServiceRegistration{
		ServiceType:   serviceType,
		InstanceAPI:   instanceAPI,
		Status:        "Starting", // Control service starts with Starting status like all other services
		Configuration: config,
		Timestamp:     time.Now(),
	}

	return h.RegisterService(registration)
}

// UpdateSelfHealth updates the control service's own health status
func (h *HealthService) UpdateSelfHealth(serviceType string, metrics map[string]interface{}) error {
	// Get hostname for instance ID
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	return h.UpdateServiceHealth(serviceType, hostname, "Healthy", metrics, nil)
}