package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
)

const TestInterval = 5 * time.Minute

// DatasetTestingService handles periodic testing of datasets
type DatasetTestingService struct {
	storage       storage.Storage
	healthService *HealthService
	stopChan      chan struct{}
	ticker        *time.Ticker
	started       bool
	mutex         sync.Mutex
}

// NewDatasetTestingService creates a new dataset testing service
func NewDatasetTestingService(storage storage.Storage, healthService *HealthService) *DatasetTestingService {
	return &DatasetTestingService{
		storage:       storage,
		healthService: healthService,
		stopChan:      make(chan struct{}),
	}
}

// Start begins periodic testing of datasets
func (s *DatasetTestingService) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Prevent multiple starts
	if s.started {
		log.Warn("Dataset testing service already started, ignoring duplicate start request")
		return
	}
	s.started = true

	s.ticker = time.NewTicker(TestInterval)

	log.Infof("Starting dataset testing service - will test all active datasets every %v", TestInterval)

	// Run initial test after 30 seconds (not immediately to let system stabilize)
	go func() {
		time.Sleep(30 * time.Second)
		s.testAllDatasets()
	}()

	// Start periodic testing
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.testAllDatasets()
			case <-s.stopChan:
				log.Info("Dataset testing service stopped")
				return
			}
		}
	}()
}

// Stop stops the periodic testing
func (s *DatasetTestingService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopChan)
}

// testAllDatasets tests all active datasets
func (s *DatasetTestingService) testAllDatasets() {
	ctx := context.Background()

	log.Debug("Running periodic dataset tests...")

	// Get all datasets
	result, err := s.storage.ListAllDatasets(ctx, storage.ListOptions{})
	if err != nil {
		log.Errorf("Failed to list datasets for testing: %v", err)
		return
	}

	testedCount := 0
	activeCount := 0

	for _, dataset := range result.Items {
		// Only test active datasets
		if !dataset.Active {
			continue
		}

		activeCount++

		// Test the dataset
		if err := s.testDataset(ctx, &dataset); err != nil {
			log.Warnf("Failed to test dataset %s/%s: %v", dataset.TenantID, dataset.ID, err)
		} else {
			testedCount++
		}
	}

	log.Infof("Periodic dataset testing completed: %d active datasets, %d tested successfully", activeCount, testedCount)
}

// testDataset tests a single dataset (input and output)
func (s *DatasetTestingService) testDataset(ctx context.Context, dataset *storage.Dataset) error {
	inputStatus := "untested"
	inputMessage := ""
	outputStatus := "untested"
	outputMessage := ""
	now := time.Now()

	// Test Input: Check if any proxy configuration has a plugin for this dataset
	// AND verify the proxy is actually running via health service
	proxyConfigs, err := s.storage.ListProxyConfigs(ctx, dataset.TenantID)
	if err != nil {
		log.Warnf("Could not check proxy configs for dataset %s: %v", dataset.ID, err)
		inputStatus = "untested"
		inputMessage = "Unable to check proxy configuration"
	} else {
		// Get all running proxies from health service
		runningProxies := make(map[string]bool)
		if s.healthService != nil {
			healthRecords, err := s.healthService.GetHealthRecordsByService("bytefreezer-proxy")
			if err == nil {
				for _, record := range healthRecords {
					if record.Status == "Healthy" || record.Status == "Active" {
						runningProxies[record.InstanceID] = true
					}
				}
			}
		}

		// Check if any plugin config references this dataset
		foundInProxy := false
		var proxyInstanceID string
		for _, proxyConfig := range proxyConfigs {
			for _, pluginConfig := range proxyConfig.PluginConfigs {
				// Check dataset_id in the nested config object
				var datasetID string
				if config, ok := pluginConfig["config"].(map[string]interface{}); ok {
					if id, ok := config["dataset_id"].(string); ok {
						datasetID = id
					}
				}

				if datasetID == dataset.ID {
					foundInProxy = true
					proxyInstanceID = proxyConfig.InstanceID

					// Check if config was applied AND proxy is running
					if !proxyConfig.ConfigApplied {
						inputStatus = "untested"
						inputMessage = "Waiting for proxy to apply configuration"
					} else if !runningProxies[proxyInstanceID] {
						inputStatus = "degraded"
						inputMessage = fmt.Sprintf("Proxy %s is configured but not running/healthy", proxyInstanceID)
					} else {
						inputStatus = "active"
						inputMessage = fmt.Sprintf("Plugin configured and proxy %s is running", proxyInstanceID)
					}
					break
				}
			}
			if foundInProxy {
				break
			}
		}

		if !foundInProxy {
			inputStatus = "degraded"
			inputMessage = "No proxy configuration found for this dataset"
		}
	}

	// Test Output: Check S3/Minio configuration and actually write test data
	destType := dataset.Config.Destination.Type
	if (destType == "s3" || destType == "minio") && dataset.Config.Destination.Connection.Bucket != "" {
		conn := dataset.Config.Destination.Connection

		// Extract S3 credentials
		accessKey := conn.Credentials.AccessKey
		secretKey := conn.Credentials.SecretKey
		region := conn.Region
		endpoint := conn.Endpoint
		useSSL := conn.SSL
		bucket := conn.Bucket

		// Try to create S3 client
		s3Cleaner, err := storage.NewS3Cleaner(
			ctx,
			accessKey,
			secretKey,
			region,
			endpoint,
			useSSL,
		)

		if err != nil {
			outputStatus = "degraded"
			outputMessage = fmt.Sprintf("Failed to create S3 client: %v", err)
			log.Debugf("S3 client creation failed for dataset %s: %v", dataset.ID, err)
		} else {
			// Actually test writing to the bucket
			err = s3Cleaner.TestWrite(ctx, bucket)
			if err != nil {
				outputStatus = "degraded"
				outputMessage = fmt.Sprintf("Failed to write test data to bucket: %v", err)
				log.Debugf("S3 write test failed for dataset %s bucket %s: %v", dataset.ID, bucket, err)
			} else {
				outputStatus = "active"
				outputMessage = "S3 connection successful and write test passed"
				log.Debugf("S3 write test passed for dataset %s bucket %s", dataset.ID, bucket)
			}
		}
	} else {
		outputStatus = "degraded"
		outputMessage = "S3 output not configured"
	}

	// Update dataset with test results
	dataset.InputTestStatus = inputStatus
	dataset.InputTestMessage = inputMessage
	dataset.OutputTestStatus = outputStatus
	dataset.OutputTestMessage = outputMessage
	dataset.LastTestedAt = &now

	// Update overall dataset status based on test results
	// If either input or output test is degraded, mark dataset as degraded
	if inputStatus == "degraded" || outputStatus == "degraded" {
		dataset.Status = "degraded"
	} else if inputStatus == "active" && outputStatus == "active" {
		dataset.Status = "active"
	}
	// If tests are "untested", keep current status

	if err := s.storage.UpdateDataset(ctx, dataset); err != nil {
		return fmt.Errorf("failed to update dataset test status: %w", err)
	}

	log.Debugf("Dataset %s/%s tested - Input: %s, Output: %s",
		dataset.TenantID, dataset.ID, inputStatus, outputStatus)

	return nil
}
