package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/go-goodies/log"
)

// ecosystemMonitor implements the EcosystemMonitor interface
type ecosystemMonitor struct {
	config   *config.Config
	client   *http.Client
	statuses map[string]*ServiceStatus
	mutex    sync.RWMutex
}

// NewEcosystemMonitor creates a new ecosystem monitor
func NewEcosystemMonitor(config *config.Config) EcosystemMonitor {
	return &ecosystemMonitor{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		statuses: make(map[string]*ServiceStatus),
	}
}

// CheckAllServices checks the health of all configured services
func (e *ecosystemMonitor) CheckAllServices() error {
	services := map[string]config.ServiceEndpoint{
		"receiver": e.config.Services.Receiver,
		"proxy":    e.config.Services.Proxy,
		"soc":      e.config.Services.SOC,
		"packer":   e.config.Services.Packer,
	}

	var wg sync.WaitGroup
	for name, service := range services {
		wg.Add(1)
		go func(serviceName string, serviceConfig config.ServiceEndpoint) {
			defer wg.Done()
			status, err := e.checkSingleService(serviceName, serviceConfig)
			if err != nil {
				log.Warnf("Failed to check service %s: %v", serviceName, err)
			}

			e.mutex.Lock()
			e.statuses[serviceName] = status
			e.mutex.Unlock()
		}(name, service)
	}

	wg.Wait()
	log.Debug("Completed health check for all ecosystem services")
	return nil
}

// CheckService checks the health of a specific service
func (e *ecosystemMonitor) CheckService(serviceName string) (*ServiceStatus, error) {
	var serviceConfig config.ServiceEndpoint

	switch serviceName {
	case "receiver":
		serviceConfig = e.config.Services.Receiver
	case "proxy":
		serviceConfig = e.config.Services.Proxy
	case "soc":
		serviceConfig = e.config.Services.SOC
	case "packer":
		serviceConfig = e.config.Services.Packer
	default:
		return nil, fmt.Errorf("unknown service: %s", serviceName)
	}

	status, err := e.checkSingleService(serviceName, serviceConfig)
	if err != nil {
		return status, err
	}

	e.mutex.Lock()
	e.statuses[serviceName] = status
	e.mutex.Unlock()

	return status, nil
}

// GetServiceStatuses returns the current status of all services
func (e *ecosystemMonitor) GetServiceStatuses() map[string]*ServiceStatus {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	// Create a deep copy to avoid race conditions
	statuses := make(map[string]*ServiceStatus)
	for name, status := range e.statuses {
		statusCopy := *status
		statuses[name] = &statusCopy
	}

	return statuses
}

// checkSingleService performs a health check on a single service
func (e *ecosystemMonitor) checkSingleService(serviceName string, serviceConfig config.ServiceEndpoint) (*ServiceStatus, error) {
	status := &ServiceStatus{
		Name:      serviceName,
		URL:       serviceConfig.URL,
		LastCheck: time.Now(),
		Healthy:   false,
	}

	// Check health endpoint
	healthURL := serviceConfig.URL + serviceConfig.HealthEndpoint
	start := time.Now()

	resp, err := e.client.Get(healthURL)
	status.ResponseTime = time.Since(start)

	if err != nil {
		status.Error = fmt.Sprintf("Health check failed: %v", err)
		return status, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		status.Error = fmt.Sprintf("Health check returned status %d", resp.StatusCode)
		return status, fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	// Parse health response to get version info
	var healthResponse struct {
		Status  string `json:"status"`
		Version string `json:"version"`
		Service string `json:"service"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&healthResponse); err == nil {
		status.Version = healthResponse.Version
		if healthResponse.Status == "ok" {
			status.Healthy = true
		}
	} else {
		// If we can't decode the response, but we got 200, consider it healthy
		status.Healthy = true
	}

	// Optionally fetch config information
	if status.Healthy && serviceConfig.ConfigEndpoint != "" {
		configURL := serviceConfig.URL + serviceConfig.ConfigEndpoint
		if configResp, err := e.client.Get(configURL); err == nil && configResp.StatusCode == http.StatusOK {
			var configData map[string]interface{}
			if err := json.NewDecoder(configResp.Body).Decode(&configData); err == nil {
				status.Config = configData
			}
			configResp.Body.Close()
		}
	}

	log.Debugf("Health check for %s: healthy=%v, response_time=%v", serviceName, status.Healthy, status.ResponseTime)
	return status, nil
}
