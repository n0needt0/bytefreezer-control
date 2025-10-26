package api

import (
	"context"
	"fmt"
	"time"

	"github.com/n0needt0/bytefreezer-control/storage"
	"github.com/n0needt0/go-goodies/log"
	"github.com/swaggest/usecase"
)

// RecordDatasetMetric handles POST requests to record dataset metrics
func (api *API) RecordDatasetMetric() usecase.Interactor {
	type recordDatasetMetricInput struct {
		TenantID       string                 `path:"tenantId" minLength:"1"`
		DatasetID      string                 `path:"datasetId" minLength:"1"`
		Component      string                 `json:"component" minLength:"1" maxLength:"50" required:"true" description:"Component name (proxy, piper, packer, control)"`
		InputBytes     int64                  `json:"input_bytes" minimum:"0" description:"Input bytes processed"`
		OutputBytes    int64                  `json:"output_bytes" minimum:"0" description:"Output bytes processed"`
		LinesProcessed int64                  `json:"lines_processed" minimum:"0" description:"Number of lines processed"`
		ErrorCount     int64                  `json:"error_count" minimum:"0" description:"Number of errors encountered"`
		CustomMetrics  map[string]interface{} `json:"custom_metrics,omitempty" description:"Additional custom metrics"`
	}

	type recordDatasetMetricOutput struct {
		Success    bool      `json:"success"`
		MetricID   int64     `json:"metric_id"`
		RecordedAt time.Time `json:"recorded_at"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input recordDatasetMetricInput, output *recordDatasetMetricOutput) error {
		api.Services.IncrementAPIRequests()

		// Validate component name
		validComponents := map[string]bool{
			"proxy": true, "receiver": true, "piper": true, "packer": true, "control": true,
		}
		if !validComponents[input.Component] {
			return fmt.Errorf("invalid component: %s (must be one of: proxy, receiver, piper, packer, control)", input.Component)
		}

		// Create metric record
		metric := &storage.DatasetMetric{
			TenantID:       input.TenantID,
			DatasetID:      input.DatasetID,
			Component:      input.Component,
			InputBytes:     input.InputBytes,
			OutputBytes:    input.OutputBytes,
			LinesProcessed: input.LinesProcessed,
			ErrorCount:     input.ErrorCount,
			CustomMetrics:  input.CustomMetrics,
			RecordedAt:     time.Now(),
		}

		// Record to database
		if err := api.Services.Storage.RecordDatasetMetric(ctx, metric); err != nil {
			log.Errorf("Failed to record dataset metric for %s/%s: %v", input.TenantID, input.DatasetID, err)
			return fmt.Errorf("failed to record metric: %w", err)
		}

		log.Infof("Successfully recorded dataset metric for %s/%s (component: %s, metric_id: %d)",
			input.TenantID, input.DatasetID, input.Component, metric.ID)

		output.Success = true
		output.MetricID = metric.ID
		output.RecordedAt = metric.RecordedAt

		return nil
	})

	return u
}

// QueryDatasetMetrics handles GET requests to query dataset metrics
func (api *API) QueryDatasetMetrics() usecase.Interactor {
	type queryDatasetMetricsInput struct {
		TenantID  string `path:"tenantId" minLength:"1"`
		DatasetID string `path:"datasetId" minLength:"1"`
		Component string `query:"component" description:"Filter by component (optional)"`
		Range     string `query:"range" required:"true" description:"Time range (15m, 1h, 6h, 24h, 7d, 30d, 90d)"`
		Limit     int    `query:"limit" minimum:"1" maximum:"10000" default:"1000" description:"Maximum number of records to return"`
	}

	type queryDatasetMetricsOutput struct {
		Metrics []*storage.DatasetMetric `json:"metrics"`
		Count   int                      `json:"count"`
		Range   string                   `json:"range"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input queryDatasetMetricsInput, output *queryDatasetMetricsOutput) error {
		api.Services.IncrementAPIRequests()

		// Parse time range
		startTime, endTime, err := parseTimeRange(input.Range)
		if err != nil {
			return fmt.Errorf("invalid time range: %w", err)
		}

		// Build filter
		filter := storage.MetricsQueryFilter{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			Component: input.Component,
			StartTime: startTime,
			EndTime:   endTime,
			Limit:     input.Limit,
		}

		// Query metrics
		metrics, err := api.Services.Storage.QueryDatasetMetrics(ctx, filter)
		if err != nil {
			log.Errorf("Failed to query dataset metrics for %s/%s: %v", input.TenantID, input.DatasetID, err)
			return fmt.Errorf("failed to query metrics: %w", err)
		}

		log.Infof("Successfully queried dataset metrics for %s/%s: %d records (range: %s)",
			input.TenantID, input.DatasetID, len(metrics), input.Range)

		output.Metrics = metrics
		output.Count = len(metrics)
		output.Range = input.Range

		return nil
	})

	return u
}

// GetAggregatedMetrics handles GET requests for aggregated metrics by component
func (api *API) GetAggregatedMetrics() usecase.Interactor {
	type getAggregatedMetricsInput struct {
		TenantID  string `path:"tenantId" minLength:"1"`
		DatasetID string `path:"datasetId" minLength:"1"`
		Range     string `query:"range" required:"true" description:"Time range (15m, 1h, 6h, 24h, 7d, 30d, 90d)"`
	}

	type getAggregatedMetricsOutput struct {
		Components          []*storage.ComponentMetrics `json:"components"`
		Range               string                      `json:"range"`
		TotalInputBytes     int64                       `json:"total_input_bytes"`
		TotalOutputBytes    int64                       `json:"total_output_bytes"`
		TotalLinesProcessed int64                       `json:"total_lines_processed"`
		TotalErrors         int64                       `json:"total_errors"`
	}

	u := usecase.NewInteractor(func(ctx context.Context, input getAggregatedMetricsInput, output *getAggregatedMetricsOutput) error {
		api.Services.IncrementAPIRequests()

		// Parse time range
		startTime, endTime, err := parseTimeRange(input.Range)
		if err != nil {
			return fmt.Errorf("invalid time range: %w", err)
		}

		// Build filter
		filter := storage.MetricsQueryFilter{
			TenantID:  input.TenantID,
			DatasetID: input.DatasetID,
			StartTime: startTime,
			EndTime:   endTime,
		}

		// Get aggregated metrics
		components, err := api.Services.Storage.GetAggregatedMetrics(ctx, filter)
		if err != nil {
			log.Errorf("Failed to get aggregated metrics for %s/%s: %v", input.TenantID, input.DatasetID, err)
			return fmt.Errorf("failed to get aggregated metrics: %w", err)
		}

		// Calculate totals
		var totalInputBytes, totalOutputBytes, totalLinesProcessed, totalErrors int64
		for _, comp := range components {
			totalInputBytes += comp.InputBytes
			totalOutputBytes += comp.OutputBytes
			totalLinesProcessed += comp.LinesProcessed
			totalErrors += comp.ErrorCount
		}

		log.Infof("Successfully retrieved aggregated metrics for %s/%s: %d components (range: %s)",
			input.TenantID, input.DatasetID, len(components), input.Range)

		output.Components = components
		output.Range = input.Range
		output.TotalInputBytes = totalInputBytes
		output.TotalOutputBytes = totalOutputBytes
		output.TotalLinesProcessed = totalLinesProcessed
		output.TotalErrors = totalErrors

		return nil
	})

	return u
}

// parseTimeRange converts a time range string to start and end times
func parseTimeRange(timeRange string) (time.Time, time.Time, error) {
	now := time.Now()
	var duration time.Duration

	switch timeRange {
	case "15m":
		duration = 15 * time.Minute
	case "1h":
		duration = 1 * time.Hour
	case "6h":
		duration = 6 * time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "90d":
		duration = 90 * 24 * time.Hour
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time range: %s (must be one of: 15m, 1h, 6h, 24h, 7d, 30d, 90d)", timeRange)
	}

	startTime := now.Add(-duration)
	return startTime, now, nil
}
