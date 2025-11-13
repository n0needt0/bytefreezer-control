package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/bytedance/sonic"
)

// ====================================================================================
// PIPER FILE LOCK OPERATIONS
// ====================================================================================

// AcquireFileLock attempts to acquire a lock on a file for processing
func (c *Client) AcquireFileLock(ctx context.Context, tenantID, datasetID, fileKey, lockedBy string, lockDurationSeconds int) (*PiperFileLock, error) {
	payload := map[string]interface{}{
		"tenant_id":             tenantID,
		"dataset_id":            datasetID,
		"file_key":              fileKey,
		"locked_by":             lockedBy,
		"lock_duration_seconds": lockDurationSeconds,
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/piper/locks/files", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var lock PiperFileLock
	if err := sonic.Unmarshal(respBody, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &lock, nil
}

// ReleaseFileLock releases a lock on a file
func (c *Client) ReleaseFileLock(ctx context.Context, tenantID, datasetID, fileKey, lockedBy string) error {
	payload := map[string]interface{}{
		"tenant_id":  tenantID,
		"dataset_id": datasetID,
		"file_key":   fileKey,
		"locked_by":  lockedBy,
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/api/v1/piper/locks/files", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CheckFileLock checks if a file is currently locked
func (c *Client) CheckFileLock(ctx context.Context, tenantID, datasetID, fileKey string) (*PiperFileLock, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/piper/locks/files/%s/%s/%s", c.baseURL, tenantID, datasetID, url.PathEscape(fileKey)), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No lock exists
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var lock PiperFileLock
	if err := sonic.Unmarshal(respBody, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &lock, nil
}

// CleanupExpiredFileLocks removes all expired file locks
func (c *Client) CleanupExpiredFileLocks(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/api/v1/piper/locks/files/cleanup/expired", c.baseURL), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var result CountResponse
	if err := sonic.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Count, nil
}

// CleanupStaleFileLocks removes file locks with stale heartbeats
func (c *Client) CleanupStaleFileLocks(ctx context.Context, staleThresholdSeconds int) (int, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/locks/files/cleanup/stale", c.baseURL))
	if err != nil {
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("stale_threshold_seconds", strconv.Itoa(staleThresholdSeconds))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var result CountResponse
	if err := sonic.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Count, nil
}

// ====================================================================================
// PIPER JOB RECORD OPERATIONS
// ====================================================================================

// CreatePiperJob creates a new job record
func (c *Client) CreatePiperJob(ctx context.Context, job *PiperJobRecord) (*PiperJobRecord, error) {
	body, err := sonic.Marshal(job)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/piper/jobs", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var createdJob PiperJobRecord
	if err := sonic.Unmarshal(respBody, &createdJob); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &createdJob, nil
}

// UpdatePiperJobStatus updates the status of a job
func (c *Client) UpdatePiperJobStatus(ctx context.Context, jobID, status, errorMessage, outputFile string, recordsProcessed int64) error {
	payload := map[string]interface{}{
		"status":            status,
		"error_message":     errorMessage,
		"output_file":       outputFile,
		"records_processed": recordsProcessed,
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", fmt.Sprintf("%s/api/v1/piper/jobs/%s/status", c.baseURL, jobID), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetPiperJob retrieves a job by ID
func (c *Client) GetPiperJob(ctx context.Context, jobID string) (*PiperJobRecord, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/piper/jobs/%s", c.baseURL, jobID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var job PiperJobRecord
	if err := sonic.Unmarshal(respBody, &job); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &job, nil
}

// GetPiperJobsByStatus retrieves jobs by status
func (c *Client) GetPiperJobsByStatus(ctx context.Context, status string, limit int) ([]PiperJobRecord, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/jobs", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	if status != "" {
		q.Set("status", status)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []PiperJobRecord `json:"items"`
		Total int              `json:"total"`
	}
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}

// GetPiperJobsForTenant retrieves all jobs for a tenant
func (c *Client) GetPiperJobsForTenant(ctx context.Context, tenantID string, limit int) ([]PiperJobRecord, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/jobs/tenant/%s", c.baseURL, tenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []PiperJobRecord `json:"items"`
		Total int              `json:"total"`
	}
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}

// CleanupOldPiperJobs deletes jobs older than specified days
func (c *Client) CleanupOldPiperJobs(ctx context.Context, olderThanDays int) (int, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/jobs/cleanup/old", c.baseURL))
	if err != nil {
		return 0, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	q.Set("older_than_days", strconv.Itoa(olderThanDays))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var result CountResponse
	if err := sonic.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Count, nil
}

// ====================================================================================
// PIPER PIPELINE CONFIGURATION CACHE OPERATIONS
// ====================================================================================

// CachePipelineConfiguration caches a pipeline configuration
func (c *Client) CachePipelineConfiguration(ctx context.Context, tenantID, datasetID string, configuration map[string]interface{}, ttlSeconds int) error {
	payload := map[string]interface{}{
		"tenant_id":     tenantID,
		"dataset_id":    datasetID,
		"configuration": configuration,
		"ttl_seconds":   ttlSeconds,
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/piper/cache/pipelines", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetCachedPipelineConfiguration retrieves a cached pipeline configuration
func (c *Client) GetCachedPipelineConfiguration(ctx context.Context, tenantID, datasetID string) (*PiperPipelineConfiguration, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/api/v1/piper/cache/pipelines/%s/%s", c.baseURL, tenantID, datasetID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var config PiperPipelineConfiguration
	if err := sonic.Unmarshal(respBody, &config); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &config, nil
}

// InvalidatePipelineConfiguration removes a cached pipeline configuration
func (c *Client) InvalidatePipelineConfiguration(ctx context.Context, tenantID, datasetID string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/api/v1/piper/cache/pipelines/%s/%s", c.baseURL, tenantID, datasetID), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ListCachedPipelines lists all cached pipeline configurations
func (c *Client) ListCachedPipelines(ctx context.Context, limit int) ([]PiperPipelineConfiguration, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/cache/pipelines", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []PiperPipelineConfiguration `json:"items"`
		Total int                          `json:"total"`
	}
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}

// CleanupExpiredPipelineCache removes expired pipeline cache entries
func (c *Client) CleanupExpiredPipelineCache(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/api/v1/piper/cache/pipelines/cleanup/expired", c.baseURL), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var result CountResponse
	if err := sonic.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Count, nil
}

// ====================================================================================
// PIPER TENANT CACHE OPERATIONS
// ====================================================================================

// CacheTenant caches tenant information
func (c *Client) CacheTenant(ctx context.Context, tenantID string, tenantData map[string]interface{}, ttlSeconds int) error {
	payload := map[string]interface{}{
		"tenant_id":   tenantID,
		"tenant_data": tenantData,
		"ttl_seconds": ttlSeconds,
	}

	body, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("%s/api/v1/piper/cache/tenants", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// GetCachedTenants retrieves all cached tenants
func (c *Client) GetCachedTenants(ctx context.Context, limit int) ([]PiperTenantCache, error) {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/cache/tenants", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []PiperTenantCache `json:"items"`
		Total int                `json:"total"`
	}
	if err := sonic.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}

// InvalidateTenantCache removes cached tenant(s)
func (c *Client) InvalidateTenantCache(ctx context.Context, tenantID string) error {
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/piper/cache/tenants", c.baseURL))
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	if tenantID != "" {
		q := u.Query()
		q.Set("tenant_id", tenantID)
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// CleanupExpiredTenantCache removes expired tenant cache entries
func (c *Client) CleanupExpiredTenantCache(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", fmt.Sprintf("%s/api/v1/piper/cache/tenants/cleanup/expired", c.baseURL), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	var result CountResponse
	if err := sonic.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Count, nil
}
