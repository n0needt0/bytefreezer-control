package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

// PostgreSQLStorage implements the Storage interface using PostgreSQL
type PostgreSQLStorage struct {
	db       *sql.DB
	config   Config
	migrator *PostgreSQLMigrator
}

// NewPostgreSQLStorage creates a new PostgreSQL storage instance
func NewPostgreSQLStorage(config Config) (*PostgreSQLStorage, error) {
	// Set default SSL mode if not specified
	if config.SSLMode == "" {
		config.SSLMode = "disable"
	}

	db, err := sql.Open("postgres", config.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	storage := &PostgreSQLStorage{
		db:     db,
		config: config,
	}

	storage.migrator = NewPostgreSQLMigrator(db)
	return storage, nil
}

// Account operations
func (p *PostgreSQLStorage) CreateAccount(ctx context.Context, account *Account) error {
	if account.ID == "" {
		account.ID = GenerateShortID()
	}

	// Set default config if empty
	if isEmptyAccountConfig(account.Config) {
		account.Config = GetDefaultAccountConfig()
	}

	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	configJSON, err := json.Marshal(account.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal account config: %w", err)
	}

	query := `
		INSERT INTO control_accounts (id, name, email, active, created_at, updated_at, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err = p.db.ExecContext(ctx, query,
		account.ID, account.Name, account.Email, account.Active,
		account.CreatedAt, account.UpdatedAt, configJSON)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique violation
			return fmt.Errorf("account with email %s already exists", account.Email)
		}
		return fmt.Errorf("failed to create account: %w", err)
	}

	return nil
}

func (p *PostgreSQLStorage) GetAccount(ctx context.Context, id string) (*Account, error) {
	query := `
		SELECT id, name, email, active, created_at, updated_at, config
		FROM control_accounts WHERE id = $1`

	var account Account
	var configJSON []byte

	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&account.ID, &account.Name, &account.Email, &account.Active,
		&account.CreatedAt, &account.UpdatedAt, &configJSON)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	if err := json.Unmarshal(configJSON, &account.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account config: %w", err)
	}

	return &account, nil
}

func (p *PostgreSQLStorage) GetAccountByEmail(ctx context.Context, email string) (*Account, error) {
	query := `
		SELECT id, name, email, active, created_at, updated_at, config
		FROM control_accounts WHERE email = $1`

	var account Account
	var configJSON []byte

	err := p.db.QueryRowContext(ctx, query, email).Scan(
		&account.ID, &account.Name, &account.Email, &account.Active,
		&account.CreatedAt, &account.UpdatedAt, &configJSON)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get account by email: %w", err)
	}

	if err := json.Unmarshal(configJSON, &account.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account config: %w", err)
	}

	return &account, nil
}

func (p *PostgreSQLStorage) UpdateAccount(ctx context.Context, account *Account) error {
	account.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(account.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal account config: %w", err)
	}

	query := `
		UPDATE control_accounts
		SET name = $1, email = $2, active = $3, updated_at = $4, config = $5
		WHERE id = $6`

	result, err := p.db.ExecContext(ctx, query,
		account.Name, account.Email, account.Active,
		account.UpdatedAt, configJSON, account.ID)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return fmt.Errorf("account with email %s already exists", account.Email)
		}
		return fmt.Errorf("failed to update account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (p *PostgreSQLStorage) DeleteAccount(ctx context.Context, id string) error {
	query := `DELETE FROM control_accounts WHERE id = $1`
	result, err := p.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (p *PostgreSQLStorage) ListAccounts(ctx context.Context, opts ListOptions) (*ListResult[Account], error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	query := `
		SELECT id, name, email, active, created_at, updated_at, config
		FROM control_accounts
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := p.db.QueryContext(ctx, query, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var account Account
		var configJSON []byte

		err := rows.Scan(
			&account.ID, &account.Name, &account.Email, &account.Active,
			&account.CreatedAt, &account.UpdatedAt, &configJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}

		if err := json.Unmarshal(configJSON, &account.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal account config: %w", err)
		}

		accounts = append(accounts, account)
	}

	return &ListResult[Account]{
		Items: accounts,
		Total: len(accounts),
	}, nil
}

// Tenant operations
func (p *PostgreSQLStorage) CreateTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.ID == "" {
		tenant.ID = GenerateShortID()
	}

	// Set default config if empty
	if isEmptyTenantConfig(tenant.Config) {
		tenant.Config = getDefaultTenantConfig()
	}

	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	configJSON, err := json.Marshal(tenant.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant config: %w", err)
	}

	query := `
		INSERT INTO control_tenants (id, account_id, name, display_name, description, active, created_at, updated_at, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err = p.db.ExecContext(ctx, query,
		tenant.ID, tenant.AccountID, tenant.Name, tenant.DisplayName, tenant.Description, tenant.Active,
		tenant.CreatedAt, tenant.UpdatedAt, configJSON)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" { // unique violation
			return fmt.Errorf("tenant with name '%s' already exists for this account", tenant.Name)
		}
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

func (p *PostgreSQLStorage) GetTenant(ctx context.Context, accountID, tenantID string) (*Tenant, error) {
	query := `
		SELECT id, account_id, name, description, active, created_at, updated_at, config
		FROM control_tenants WHERE id = $1 AND account_id = $2`

	var tenant Tenant
	var configJSON []byte

	err := p.db.QueryRowContext(ctx, query, tenantID, accountID).Scan(
		&tenant.ID, &tenant.AccountID, &tenant.Name, &tenant.Description, &tenant.Active,
		&tenant.CreatedAt, &tenant.UpdatedAt, &configJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if err := json.Unmarshal(configJSON, &tenant.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tenant config: %w", err)
	}

	return &tenant, nil
}

// GetTenantByID gets a tenant by ID alone (without account ID) - used for proxy validation
func (p *PostgreSQLStorage) GetTenantByID(ctx context.Context, tenantID string) (*Tenant, error) {
	query := `
		SELECT id, account_id, name, display_name, description, active, created_at, updated_at, config
		FROM control_tenants WHERE id = $1`

	var tenant Tenant
	var configJSON []byte

	err := p.db.QueryRowContext(ctx, query, tenantID).Scan(
		&tenant.ID, &tenant.AccountID, &tenant.Name, &tenant.DisplayName, &tenant.Description, &tenant.Active,
		&tenant.CreatedAt, &tenant.UpdatedAt, &configJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by ID: %w", err)
	}

	if err := json.Unmarshal(configJSON, &tenant.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tenant config: %w", err)
	}

	return &tenant, nil
}

func (p *PostgreSQLStorage) UpdateTenant(ctx context.Context, tenant *Tenant) error {
	tenant.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(tenant.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal tenant config: %w", err)
	}

	query := `
		UPDATE control_tenants
		SET name = $1, description = $2, active = $3, updated_at = $4, config = $5
		WHERE id = $6 AND account_id = $7`

	result, err := p.db.ExecContext(ctx, query,
		tenant.Name, tenant.Description, tenant.Active,
		tenant.UpdatedAt, configJSON, tenant.ID, tenant.AccountID)

	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (p *PostgreSQLStorage) DeleteTenant(ctx context.Context, accountID, tenantID string) error {
	query := `DELETE FROM control_tenants WHERE id = $1 AND account_id = $2`

	result, err := p.db.ExecContext(ctx, query, tenantID, accountID)
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (p *PostgreSQLStorage) ListTenants(ctx context.Context, accountID string, opts ListOptions) (*ListResult[Tenant], error) {
	if opts.Limit <= 0 {
		opts.Limit = 10
	}

	query := `
		SELECT id, account_id, name, description, active, created_at, updated_at, config
		FROM control_tenants
		WHERE account_id = $1
		ORDER BY created_at DESC
		LIMIT $2`

	rows, err := p.db.QueryContext(ctx, query, accountID, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		var configJSON []byte

		err := rows.Scan(&tenant.ID, &tenant.AccountID, &tenant.Name, &tenant.Description, &tenant.Active,
			&tenant.CreatedAt, &tenant.UpdatedAt, &configJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		if err := json.Unmarshal(configJSON, &tenant.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tenant config: %w", err)
		}

		tenants = append(tenants, tenant)
	}

	return &ListResult[Tenant]{
		Items: tenants,
		Total: len(tenants),
	}, nil
}

// ListAllTenants lists all tenants across all accounts (flat list for UI)
func (p *PostgreSQLStorage) ListAllTenants(ctx context.Context, opts ListOptions) (*ListResult[Tenant], error) {
	if opts.Limit <= 0 {
		opts.Limit = 100 // Higher default for flat list
	}

	query := `
		SELECT id, account_id, name, description, active, created_at, updated_at, config
		FROM control_tenants
		ORDER BY created_at DESC
		LIMIT $1`

	rows, err := p.db.QueryContext(ctx, query, opts.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list all tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		var configJSON []byte

		err := rows.Scan(&tenant.ID, &tenant.AccountID, &tenant.Name, &tenant.Description, &tenant.Active,
			&tenant.CreatedAt, &tenant.UpdatedAt, &configJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		if err := json.Unmarshal(configJSON, &tenant.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tenant config: %w", err)
		}

		tenants = append(tenants, tenant)
	}

	return &ListResult[Tenant]{
		Items: tenants,
		Total: len(tenants),
	}, nil
}

// Advanced tenant queries using PostgreSQL JSONB operations
func (p *PostgreSQLStorage) FindTenantsBySubscriptionTier(ctx context.Context, tier string) ([]*Tenant, error) {
	query := `
		SELECT id, account_id, name, description, active, created_at, updated_at, config
		FROM control_tenants
		WHERE config->'subscription'->>'tier' = $1`

	return p.queryTenants(ctx, query, tier)
}

func (p *PostgreSQLStorage) FindTenantsByOrganizationSize(ctx context.Context, size string) ([]*Tenant, error) {
	query := `
		SELECT id, account_id, name, description, active, created_at, updated_at, config
		FROM control_tenants
		WHERE config->'organization'->>'size' = $1`

	return p.queryTenants(ctx, query, size)
}

func (p *PostgreSQLStorage) GetTenantsWithNotificationEnabled(ctx context.Context, notificationType string) ([]*Tenant, error) {
	query := `
		SELECT id, account_id, name, description, active, created_at, updated_at, config
		FROM control_tenants
		WHERE config->'notifications'->>$1 = 'true'`

	return p.queryTenants(ctx, query, notificationType)
}

func (p *PostgreSQLStorage) queryTenants(ctx context.Context, query string, args ...interface{}) ([]*Tenant, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*Tenant
	for rows.Next() {
		var tenant Tenant
		var configJSON []byte

		err := rows.Scan(&tenant.ID, &tenant.AccountID, &tenant.Name, &tenant.Description, &tenant.Active,
			&tenant.CreatedAt, &tenant.UpdatedAt, &configJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		if err := json.Unmarshal(configJSON, &tenant.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal tenant config: %w", err)
		}

		tenants = append(tenants, &tenant)
	}

	return tenants, nil
}

// Dataset operations
func (p *PostgreSQLStorage) CreateDataset(ctx context.Context, dataset *Dataset) error {
	if dataset.ID == "" {
		dataset.ID = GenerateShortID()
	}

	// Set default config if empty
	if isEmptyDatasetConfig(dataset.Config) {
		dataset.Config = getDefaultDatasetConfig()
	}

	now := time.Now()
	dataset.CreatedAt = now
	dataset.UpdatedAt = now

	configJSON, err := json.Marshal(dataset.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset config: %w", err)
	}

	metricsJSON, err := json.Marshal(dataset.ProcessingMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal processing metrics: %w", err)
	}

	query := `
		INSERT INTO control_datasets (id, tenant_id, name, display_name, description, active, status, created_at, updated_at,
			config, records_processed, last_processed_at, error_count, last_error, processing_metrics)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err = p.db.ExecContext(ctx, query,
		dataset.ID, dataset.TenantID, dataset.Name, dataset.DisplayName, dataset.Description, dataset.Active,
		dataset.Status, dataset.CreatedAt, dataset.UpdatedAt, configJSON,
		dataset.RecordsProcessed, dataset.LastProcessedAt, dataset.ErrorCount,
		dataset.LastError, metricsJSON)

	if err != nil {
		return fmt.Errorf("failed to create dataset: %w", err)
	}

	return nil
}

func (p *PostgreSQLStorage) GetDataset(ctx context.Context, tenantID, datasetID string) (*Dataset, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description, active, status, created_at, updated_at,
			config, records_processed, last_processed_at, error_count, last_error, processing_metrics
		FROM control_datasets WHERE tenant_id = $1 AND id = $2`

	var dataset Dataset
	var configJSON, metricsJSON []byte
	var lastError sql.NullString

	err := p.db.QueryRowContext(ctx, query, tenantID, datasetID).Scan(
		&dataset.ID, &dataset.TenantID, &dataset.Name, &dataset.DisplayName, &dataset.Description,
		&dataset.Active, &dataset.Status, &dataset.CreatedAt, &dataset.UpdatedAt,
		&configJSON, &dataset.RecordsProcessed, &dataset.LastProcessedAt,
		&dataset.ErrorCount, &lastError, &metricsJSON)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dataset not found: %s/%s", tenantID, datasetID)
		}
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}

	// Handle NULL last_error
	if lastError.Valid {
		dataset.LastError = lastError.String
	} else {
		dataset.LastError = ""
	}

	if err := json.Unmarshal(configJSON, &dataset.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dataset config: %w", err)
	}

	// ProcessingMetrics is stored as JSON string, not object
	if len(metricsJSON) > 0 {
		dataset.ProcessingMetrics = string(metricsJSON)
	}

	return &dataset, nil
}

func (p *PostgreSQLStorage) UpdateDataset(ctx context.Context, dataset *Dataset) error {
	dataset.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(dataset.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset config: %w", err)
	}

	metricsJSON, err := json.Marshal(dataset.ProcessingMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal processing metrics: %w", err)
	}

	query := `
		UPDATE control_datasets 
		SET name = $1, description = $2, active = $3, status = $4, updated_at = $5,
			config = $6, records_processed = $7, last_processed_at = $8, 
			error_count = $9, last_error = $10, processing_metrics = $11
		WHERE tenant_id = $12 AND id = $13`

	result, err := p.db.ExecContext(ctx, query,
		dataset.Name, dataset.Description, dataset.Active, dataset.Status,
		dataset.UpdatedAt, configJSON, dataset.RecordsProcessed,
		dataset.LastProcessedAt, dataset.ErrorCount, dataset.LastError,
		metricsJSON, dataset.TenantID, dataset.ID)

	if err != nil {
		return fmt.Errorf("failed to update dataset: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dataset not found: %s/%s", dataset.TenantID, dataset.ID)
	}

	return nil
}

func (p *PostgreSQLStorage) DeleteDataset(ctx context.Context, tenantID, datasetID string, skipS3Cleanup bool) error {
	// Step 1: Get dataset to retrieve S3 configuration
	dataset, err := p.GetDataset(ctx, tenantID, datasetID)
	if err != nil {
		return fmt.Errorf("failed to get dataset before deletion: %w", err)
	}

	// Step 2: Update status to "deleting" to prevent modifications during cleanup
	dataset.Status = "deleting"
	if err := p.UpdateDataset(ctx, dataset); err != nil {
		return fmt.Errorf("failed to update dataset status to deleting: %w", err)
	}

	// Step 3: Clean up S3 storage across all three layers (intake, piper, packer)
	bucket, region, endpoint, accessKey, secretKey, useSSL, err := GetS3ConfigFromDataset(dataset)
	if err != nil {
		// Log warning but continue with database deletion
		// This handles cases where S3 config is missing or dataset never had data
		fmt.Printf("Warning: Could not extract S3 config for dataset %s/%s: %v\n", tenantID, datasetID, err)
	} else {
		// Create S3 cleaner with proper endpoint URL
		s3Cleaner, err := NewS3Cleaner(ctx, accessKey, secretKey, region, endpoint, useSSL)
		if err != nil {
			return fmt.Errorf("failed to create S3 cleaner: %w", err)
		}

		// Perform S3 cleanup
		if err := s3Cleaner.CleanupDatasetStorage(ctx, bucket, tenantID, datasetID); err != nil {
			// S3 cleanup failed - this is critical, don't delete the database record
			// Update status to error so user can retry
			dataset.Status = "error"
			dataset.LastError = fmt.Sprintf("S3 cleanup failed: %v", err)
			if updateErr := p.UpdateDataset(ctx, dataset); updateErr != nil {
				return fmt.Errorf("S3 cleanup failed and could not update dataset status: %w (original error: %v)", updateErr, err)
			}
			return fmt.Errorf("failed to cleanup S3 storage: %w", err)
		}
	}

	// Step 4: Delete database record only after successful S3 cleanup
	query := `DELETE FROM control_datasets WHERE tenant_id = $1 AND id = $2`
	result, err := p.db.ExecContext(ctx, query, tenantID, datasetID)
	if err != nil {
		return fmt.Errorf("failed to delete dataset from database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dataset not found in database: %s/%s", tenantID, datasetID)
	}

	return nil
}

func (p *PostgreSQLStorage) ListDatasets(ctx context.Context, tenantID string, opts ListOptions) (*ListResult[Dataset], error) {
	query := `
		SELECT id, tenant_id, name, display_name, description, active, status, created_at, updated_at,
			config, records_processed, last_processed_at, error_count, last_error, processing_metrics
		FROM control_datasets WHERE tenant_id = $1`
	
	args := []interface{}{tenantID}
	argIndex := 2

	// Add filtering if specified
	if opts.Filter != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1)
		filterPattern := "%" + opts.Filter + "%"
		args = append(args, filterPattern, filterPattern)
		argIndex += 2
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, opts.Limit)
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list datasets: %w", err)
	}
	defer rows.Close()

	var datasets []Dataset
	for rows.Next() {
		var dataset Dataset
		var configJSON, metricsJSON []byte
		var lastError sql.NullString

		err := rows.Scan(
			&dataset.ID, &dataset.TenantID, &dataset.Name, &dataset.DisplayName, &dataset.Description,
			&dataset.Active, &dataset.Status, &dataset.CreatedAt, &dataset.UpdatedAt,
			&configJSON, &dataset.RecordsProcessed, &dataset.LastProcessedAt,
			&dataset.ErrorCount, &lastError, &metricsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dataset: %w", err)
		}

		if lastError.Valid {
			dataset.LastError = lastError.String
		}

		if err := json.Unmarshal(configJSON, &dataset.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dataset config: %w", err)
		}

		if len(metricsJSON) > 0 {
			dataset.ProcessingMetrics = string(metricsJSON)
		}

		datasets = append(datasets, dataset)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM control_datasets WHERE tenant_id = $1`
	countArgs := []interface{}{tenantID}
	if opts.Filter != "" {
		countQuery += " AND (name ILIKE $2 OR description ILIKE $3)"
		filterPattern := "%" + opts.Filter + "%"
		countArgs = append(countArgs, filterPattern, filterPattern)
	}

	var total int
	err = p.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset count: %w", err)
	}

	return &ListResult[Dataset]{
		Items: datasets,
		Total: total,
	}, nil
}

// ListAllDatasets lists all datasets across all tenants (flat list for UI)
func (p *PostgreSQLStorage) ListAllDatasets(ctx context.Context, opts ListOptions) (*ListResult[Dataset], error) {
	query := `
		SELECT id, tenant_id, name, display_name, description, active, status, created_at, updated_at,
			config, records_processed, last_processed_at, error_count, last_error, processing_metrics
		FROM control_datasets WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	// Add filtering if specified
	if opts.Filter != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1)
		filterPattern := "%" + opts.Filter + "%"
		args = append(args, filterPattern, filterPattern)
		argIndex += 2
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, opts.Limit)
	} else {
		query += " LIMIT 100" // Default limit for safety
	}

	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list all datasets: %w", err)
	}
	defer rows.Close()

	var datasets []Dataset
	for rows.Next() {
		var dataset Dataset
		var configJSON, metricsJSON []byte
		var lastError sql.NullString

		err := rows.Scan(
			&dataset.ID, &dataset.TenantID, &dataset.Name, &dataset.DisplayName, &dataset.Description,
			&dataset.Active, &dataset.Status, &dataset.CreatedAt, &dataset.UpdatedAt,
			&configJSON, &dataset.RecordsProcessed, &dataset.LastProcessedAt,
			&dataset.ErrorCount, &lastError, &metricsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dataset: %w", err)
		}

		if lastError.Valid {
			dataset.LastError = lastError.String
		}

		if err := json.Unmarshal(configJSON, &dataset.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dataset config: %w", err)
		}

		if len(metricsJSON) > 0 {
			dataset.ProcessingMetrics = string(metricsJSON)
		}

		datasets = append(datasets, dataset)
	}

	// Get total count
	countQuery := `SELECT COUNT(*) FROM control_datasets WHERE 1=1`
	countArgs := []interface{}{}
	if opts.Filter != "" {
		countQuery += " AND (name ILIKE $1 OR description ILIKE $2)"
		filterPattern := "%" + opts.Filter + "%"
		countArgs = append(countArgs, filterPattern, filterPattern)
	}

	var total int
	err = p.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get dataset count: %w", err)
	}

	return &ListResult[Dataset]{
		Items: datasets,
		Total: total,
	}, nil
}

// Advanced dataset queries
func (p *PostgreSQLStorage) FindDatasetsByStatus(ctx context.Context, tenantID, status string) ([]*Dataset, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description, active, status, created_at, updated_at,
			config, records_processed, last_processed_at, error_count, last_error, processing_metrics
		FROM control_datasets WHERE tenant_id = $1 AND status = $2`

	return p.queryDatasets(ctx, query, tenantID, status)
}

func (p *PostgreSQLStorage) FindDatasetsBySourceType(ctx context.Context, tenantID, sourceType string) ([]*Dataset, error) {
	query := `
		SELECT id, tenant_id, name, display_name, description, active, status, created_at, updated_at,
			config, records_processed, last_processed_at, error_count, last_error, processing_metrics
		FROM control_datasets
		WHERE tenant_id = $1 AND config->>'source'->>'type' = $2`

	return p.queryDatasets(ctx, query, tenantID, sourceType)
}

func (p *PostgreSQLStorage) queryDatasets(ctx context.Context, query string, args ...interface{}) ([]*Dataset, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query datasets: %w", err)
	}
	defer rows.Close()

	var datasets []*Dataset
	for rows.Next() {
		var dataset Dataset
		var configJSON, metricsJSON []byte

		err := rows.Scan(
			&dataset.ID, &dataset.TenantID, &dataset.Name, &dataset.DisplayName, &dataset.Description,
			&dataset.Active, &dataset.Status, &dataset.CreatedAt, &dataset.UpdatedAt,
			&configJSON, &dataset.RecordsProcessed, &dataset.LastProcessedAt,
			&dataset.ErrorCount, &dataset.LastError, &metricsJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dataset: %w", err)
		}

		if err := json.Unmarshal(configJSON, &dataset.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dataset config: %w", err)
		}

		if len(metricsJSON) > 0 {
			if err := json.Unmarshal(metricsJSON, &dataset.ProcessingMetrics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal processing metrics: %w", err)
			}
		}

		datasets = append(datasets, &dataset)
	}

	return datasets, nil
}

func (p *PostgreSQLStorage) GetDatasetMetrics(ctx context.Context, tenantID, datasetID string) (*DatasetMetrics, error) {
	query := `SELECT processing_metrics FROM control_datasets WHERE tenant_id = $1 AND id = $2`

	var metricsJSON []byte
	err := p.db.QueryRowContext(ctx, query, tenantID, datasetID).Scan(&metricsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("dataset not found: %s/%s", tenantID, datasetID)
		}
		return nil, fmt.Errorf("failed to get dataset metrics: %w", err)
	}

	var metrics DatasetMetrics
	if len(metricsJSON) > 0 {
		if err := json.Unmarshal(metricsJSON, &metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal dataset metrics: %w", err)
		}
	}

	return &metrics, nil
}

func (p *PostgreSQLStorage) UpdateDatasetMetrics(ctx context.Context, tenantID, datasetID string, metrics *DatasetMetrics) error {
	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal dataset metrics: %w", err)
	}

	query := `UPDATE control_datasets SET processing_metrics = $1, updated_at = $2 WHERE tenant_id = $3 AND id = $4`

	result, err := p.db.ExecContext(ctx, query, metricsJSON, time.Now(), tenantID, datasetID)
	if err != nil {
		return fmt.Errorf("failed to update dataset metrics: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("dataset not found: %s/%s", tenantID, datasetID)
	}

	return nil
}

// Utility operations
func (p *PostgreSQLStorage) HealthCheck(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgreSQLStorage) Migrate(ctx context.Context) error {
	return p.migrator.Migrate(ctx, MigrationOptions{})
}

func (p *PostgreSQLStorage) Close() error {
	return p.db.Close()
}

func (p *PostgreSQLStorage) GetMigrator() Migrator {
	return p.migrator
}

// GetDB returns the underlying database connection for health service
func (p *PostgreSQLStorage) GetDB() *sql.DB {
	return p.db
}

// Helper function for checking empty dataset config
func isEmptyDatasetConfig(config DatasetConfig) bool {
	return config.Source.Type == ""
}

// isEmptyTenantConfig checks if tenant config is essentially empty (only has default empty maps)
func isEmptyTenantConfig(config TenantConfig) bool {
	return config.MaxDatasets == nil &&
		config.StorageQuotaGB == nil &&
		len(config.Metadata) == 0 &&
		len(config.CustomSettings) == 0
}

// API Key operations (stub implementations - to be fully implemented)
func (p *PostgreSQLStorage) CreateAPIKey(ctx context.Context, apiKey *APIKey) error {
	return fmt.Errorf("API key creation not implemented yet")
}

func (p *PostgreSQLStorage) GetAPIKey(ctx context.Context, id string) (*APIKey, error) {
	return nil, fmt.Errorf("API key retrieval not implemented yet")
}

func (p *PostgreSQLStorage) GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error) {
	return nil, fmt.Errorf("API key lookup by hash not implemented yet")
}

func (p *PostgreSQLStorage) UpdateAPIKey(ctx context.Context, apiKey *APIKey) error {
	return fmt.Errorf("API key update not implemented yet")
}

func (p *PostgreSQLStorage) DeleteAPIKey(ctx context.Context, id string) error {
	return fmt.Errorf("API key deletion not implemented yet")
}

func (p *PostgreSQLStorage) ListAPIKeys(ctx context.Context, accountID string, opts ListOptions) (*ListResult[APIKey], error) {
	return nil, fmt.Errorf("API key listing not implemented yet")
}

func (p *PostgreSQLStorage) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	return fmt.Errorf("API key last used update not implemented yet")
}