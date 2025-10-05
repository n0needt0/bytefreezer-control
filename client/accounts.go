package client

import (
	"context"
	"fmt"
)

// CreateAccount creates a new account
func (c *Client) CreateAccount(ctx context.Context, req CreateAccountRequest) (*Account, error) {
	resp, err := c.doRequest(ctx, "POST", "/api/v2/accounts", req)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := c.parseResponse(resp, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// GetAccount retrieves an account by ID
func (c *Client) GetAccount(ctx context.Context, accountID string) (*Account, error) {
	path := fmt.Sprintf("/api/v2/accounts/%s", accountID)
	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := c.parseResponse(resp, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// ListAccounts lists all accounts with optional limit
func (c *Client) ListAccounts(ctx context.Context, limit int) ([]Account, error) {
	path := "/api/v2/accounts"
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}

	resp, err := c.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var accounts []Account
	if err := c.parseResponse(resp, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// UpdateAccount updates an existing account
func (c *Client) UpdateAccount(ctx context.Context, accountID string, req UpdateAccountRequest) (*Account, error) {
	path := fmt.Sprintf("/api/v2/accounts/%s", accountID)
	resp, err := c.doRequest(ctx, "PUT", path, req)
	if err != nil {
		return nil, err
	}

	var account Account
	if err := c.parseResponse(resp, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// DeleteAccount deletes an account by ID
func (c *Client) DeleteAccount(ctx context.Context, accountID string) error {
	path := fmt.Sprintf("/api/v2/accounts/%s", accountID)
	resp, err := c.doRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to delete account: status %d", resp.StatusCode)
	}

	return nil
}
