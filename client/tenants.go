package client

import (
	"context"
	"fmt"
)

// CreateTenant creates a new tenant within an account
func (c *Client) CreateTenant(ctx context.Context, accountID string, req CreateTenantRequest) (*Tenant, error) {
	path := fmt.Sprintf("/api/v2/accounts/%s/tenants", accountID)
	resp, err := c.doRequest(ctx, "POST", path, req)
	if err != nil {
		return nil, err
	}

	var tenant Tenant
	if err := c.parseResponse(resp, &tenant); err != nil {
		return nil, err
	}

	return &tenant, nil
}

// GetTenant retrieves a tenant by ID
func (c *Client) GetTenant(ctx context.Context, accountID, tenantID string) (*Tenant, error) {
	path := fmt.Sprintf("/api/v2/accounts/%s/tenants/%s", accountID, tenantID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tenant Tenant
	if err := c.parseResponse(resp, &tenant); err != nil {
		return nil, err
	}

	return &tenant, nil
}

// ListTenants lists all tenants for an account with optional limit
func (c *Client) ListTenants(ctx context.Context, accountID string, limit int) ([]Tenant, error) {
	path := fmt.Sprintf("/api/v2/accounts/%s/tenants", accountID)
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var tenants []Tenant
	if err := c.parseResponse(resp, &tenants); err != nil {
		return nil, err
	}

	return tenants, nil
}

// UpdateTenant updates an existing tenant
func (c *Client) UpdateTenant(ctx context.Context, accountID, tenantID string, req UpdateTenantRequest) (*Tenant, error) {
	path := fmt.Sprintf("/api/v2/accounts/%s/tenants/%s", accountID, tenantID)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return nil, err
	}

	var tenant Tenant
	if err := c.parseResponse(resp, &tenant); err != nil {
		return nil, err
	}

	return &tenant, nil
}

// DeleteTenant deletes a tenant by ID
func (c *Client) DeleteTenant(ctx context.Context, accountID, tenantID string) error {
	path := fmt.Sprintf("/api/v2/accounts/%s/tenants/%s", accountID, tenantID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete tenant: status %d", resp.StatusCode)
	}

	return nil
}
