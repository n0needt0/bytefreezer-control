package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/n0needt0/go-goodies/log"
)

// ServiceReport represents a service health report
type ServiceReport struct {
	ServiceName   string                 `json:"service_name"`
	ServiceID     string                 `json:"service_id"`
	Version       string                 `json:"version"`
	Timestamp     time.Time              `json:"timestamp"`
	Healthy       bool                   `json:"healthy"`
	Configuration map[string]interface{} `json:"configuration"`
	Metrics       map[string]interface{} `json:"metrics"`
}

// Config represents health reporting configuration
type Config struct {
	Enabled         bool          `yaml:"enabled" json:"enabled"`
	ControlURL      string        `yaml:"control_url" json:"control_url"`
	ReportInterval  time.Duration `yaml:"report_interval" json:"report_interval"`
	ServiceName     string        `yaml:"service_name" json:"service_name"`
	ServiceID       string        `yaml:"service_id" json:"service_id"`
	Version         string        `yaml:"version" json:"version"`
	TimeoutSeconds  int           `yaml:"timeout_seconds" json:"timeout_seconds"`
}

// Reporter handles health reporting to the control service
type Reporter struct {
	config           Config
	httpClient       *http.Client
	configProvider   ConfigProvider
	metricsProvider  MetricsProvider
	healthChecker    HealthChecker
	stopChan         chan struct{}
	wg               sync.WaitGroup
	mutex            sync.RWMutex
}

// ConfigProvider provides current service configuration
type ConfigProvider interface {
	GetSanitizedConfig() map[string]interface{}
}

// MetricsProvider provides current service metrics
type MetricsProvider interface {
	GetMetrics() map[string]interface{}
}

// HealthChecker provides current service health status
type HealthChecker interface {
	IsHealthy() bool
}

// NewReporter creates a new health reporter
func NewReporter(config Config, configProvider ConfigProvider, metricsProvider MetricsProvider, healthChecker HealthChecker) *Reporter {
	return &Reporter{
		config:          config,
		configProvider:  configProvider,
		metricsProvider: metricsProvider,
		healthChecker:   healthChecker,
		httpClient: &http.Client{
			Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
		},
		stopChan: make(chan struct{}),
	}
}

// Start begins the health reporting process
func (r *Reporter) Start(ctx context.Context) error {
	if !r.config.Enabled {
		log.Info("Health reporting is disabled")
		return nil
	}

	if r.config.ControlURL == "" {
		return fmt.Errorf("control_url is required when health reporting is enabled")
	}

	log.Infof("Starting health reporter for service %s, reporting to %s every %v",
		r.config.ServiceName, r.config.ControlURL, r.config.ReportInterval)

	r.wg.Add(1)
	go r.reportLoop(ctx)

	return nil
}

// Stop stops the health reporting process
func (r *Reporter) Stop() {
	close(r.stopChan)
	r.wg.Wait()
	log.Info("Health reporter stopped")
}

// reportLoop runs the periodic health reporting
func (r *Reporter) reportLoop(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(r.config.ReportInterval)
	defer ticker.Stop()

	// Send initial report immediately
	r.sendReport(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Info("Health reporter stopping due to context cancellation")
			return
		case <-r.stopChan:
			log.Info("Health reporter stopping due to stop signal")
			return
		case <-ticker.C:
			r.sendReport(ctx)
		}
	}
}

// sendReport sends a health report to the control service
func (r *Reporter) sendReport(ctx context.Context) {
	report := r.buildReport()

	jsonData, err := json.Marshal(report)
	if err != nil {
		log.Errorf("Failed to marshal health report: %v", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/v1/services/report", r.config.ControlURL),
		bytes.NewBuffer(jsonData))
	if err != nil {
		log.Errorf("Failed to create health report request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		log.Warnf("Failed to send health report: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Debugf("Health report sent successfully to control service")
	} else {
		log.Warnf("Health report failed with status %d", resp.StatusCode)
	}
}

// buildReport builds a health report from current service state
func (r *Reporter) buildReport() ServiceReport {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var config map[string]interface{}
	var metrics map[string]interface{}
	var healthy bool

	if r.configProvider != nil {
		config = r.configProvider.GetSanitizedConfig()
	} else {
		config = make(map[string]interface{})
	}

	if r.metricsProvider != nil {
		metrics = r.metricsProvider.GetMetrics()
	} else {
		metrics = make(map[string]interface{})
	}

	if r.healthChecker != nil {
		healthy = r.healthChecker.IsHealthy()
	} else {
		healthy = true // Default to healthy if no checker provided
	}

	return ServiceReport{
		ServiceName:   r.config.ServiceName,
		ServiceID:     r.config.ServiceID,
		Version:       r.config.Version,
		Timestamp:     time.Now(),
		Healthy:       healthy,
		Configuration: config,
		Metrics:       metrics,
	}
}

// SendImmediateReport sends a health report immediately (for testing or on-demand)
func (r *Reporter) SendImmediateReport(ctx context.Context) error {
	if !r.config.Enabled {
		return fmt.Errorf("health reporting is disabled")
	}

	r.sendReport(ctx)
	return nil
}