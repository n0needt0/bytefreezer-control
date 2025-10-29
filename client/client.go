package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Config represents the configuration for the control service client
type Config struct {
	BaseURL        string
	APIKey         string
	TimeoutSeconds int
}

// Client provides methods to interact with the ByteFreezer Control Service API
type Client struct {
	config     Config
	httpClient *http.Client
	baseURL    string
}

// Account represents an account from the control service API
type Account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Tenant represents a tenant from the control service API
type Tenant struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Dataset represents a dataset from the control service API
type Dataset struct {
	ID        string         `json:"id"`
	TenantID  string         `json:"tenant_id"`
	Name      string         `json:"name"`
	Active    bool           `json:"active"`
	Status    string         `json:"status"`
	Config    DatasetConfig  `json:"config"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// DatasetConfig represents the dataset configuration
type DatasetConfig struct {
	Source      interface{} `json:"source"`
	Processing  interface{} `json:"processing"`
	Destination struct {
		Type       string `json:"type"`
		Connection struct {
			URL         string                 `json:"url"`
			Bucket      string                 `json:"bucket"`
			Region      string                 `json:"region"`
			Endpoint    string                 `json:"endpoint"`
			SSL         bool                   `json:"ssl"`
			Prefix      string                 `json:"prefix"`
			Credentials struct {
				Type      string `json:"type"`
				AccessKey string `json:"access_key"`
				SecretKey string `json:"secret_key"`
			} `json:"credentials"`
			TimeoutSeconds int `json:"timeout_seconds"`
			Retries        int `json:"retries"`
		} `json:"connection"`
		Format       string                 `json:"format"`
		Partitioning interface{}            `json:"partitioning"`
		Compression  string                 `json:"compression"`
		Custom       map[string]interface{} `json:"custom"`
	} `json:"destination"`
	Schedule   interface{} `json:"schedule"`
	Monitoring interface{} `json:"monitoring"`
	Transform  interface{} `json:"transform"`
	Parquet    map[string]interface{} `json:"parquet,omitempty"`
	Custom     map[string]interface{} `json:"custom,omitempty"`
}

// NewClient creates a new control service client
func NewClient(config Config) *Client {
	timeout := time.Duration(config.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: config.BaseURL,
	}
}

// ListTenants retrieves all tenants for a given account
func (c *Client) ListTenants(ctx context.Context, accountID string, limit int) ([]Tenant, error) {
	if accountID == "" {
		return nil, fmt.Errorf("accountID cannot be empty")
	}

	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/accounts/%s/tenants", c.baseURL, accountID))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []Tenant `json:"items"`
		Total int      `json:"total"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}

// ListAccounts retrieves all accounts
func (c *Client) ListAccounts(ctx context.Context, limit int) ([]Account, error) {
	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/accounts", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []Account `json:"items"`
		Total int       `json:"total"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}

// ListDatasets retrieves all datasets for a given tenant
func (c *Client) ListDatasets(ctx context.Context, tenantID string, limit int) ([]Dataset, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenantID cannot be empty")
	}

	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/api/v1/tenants/%s/datasets", c.baseURL, tenantID))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if limit > 0 {
		q := u.Query()
		q.Set("limit", strconv.Itoa(limit))
		u.RawQuery = q.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	if c.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		Items []Dataset `json:"items"`
		Total int       `json:"total"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response.Items, nil
}