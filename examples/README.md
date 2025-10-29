# ByteFreezer Control - Examples

This directory contains example scripts and utilities for working with ByteFreezer Control Service.

## Database Population

### populate_fake_data_simple.go (Recommended)

Populates the Control Service database with fake data that matches the development mode data used in receiver, piper, and packer services. This allows you to test Control Service integration without relying on hardcoded fake data.

**What it creates:**

1. **Account**: `dev-account` (Development Account)
   - ID: `dev-account`
   - Email: `dev@bytefreezer.local`
   - Tier: free
   - Max tenants: 10, Max datasets: 50

2. **Tenants** (4 tenants matching fake data):
   - `customer-1` - Production eBPF Data
     - Datasets: `ebpf-data`, `sflow-data`
   - `tenant-001` - Development Tenant Alpha
     - Datasets: `dataset-001` (Web Analytics), `dataset-004` (User Events), `dataset-007` (Sales Data)
   - `tenant-002` - Development Tenant Beta
     - Datasets: `dataset-002` (User Events), `dataset-005` (Web Analytics), `dataset-008` (Sales Data)
   - `tenant-003` - Development Tenant Gamma
     - Datasets: `dataset-003` (Sales Data), `dataset-006` (Web Analytics), `dataset-009` (User Events)

3. **Dataset Configurations**: Each dataset includes:
   - Source configuration (webhook, format, data_hint)
   - Processing configuration (raw storage, partitioning, compression)
   - Destination configuration (S3 bucket, prefix, region)

**Usage:**

```bash
# Navigate to examples directory
cd /home/andrew/workspace/bytefreezer/bytefreezer-control/examples

# Run the script with your PostgreSQL connection string
go run populate_fake_data_simple.go "postgres://user:password@localhost:5432/bytefreezer?sslmode=disable"
```

**Example with Docker PostgreSQL:**

```bash
# Start PostgreSQL with Docker
docker run -d \
  --name bytefreezer-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=bytefreezer \
  -p 5432:5432 \
  postgres:15

# Populate the database
go run populate_fake_data_simple.go "postgres://postgres:postgres@localhost:5432/bytefreezer?sslmode=disable"
```

**What it does:**

1. ✅ Runs database migrations (creates all tables)
2. ✅ Creates development account with configuration
3. ✅ Creates 4 tenants with bearer tokens
4. ✅ Creates 11 datasets across all tenants
5. ✅ Exports summary to `fake_data_summary.json`
6. ✅ Prints API endpoints and service configuration

**Output:**

```
✅ Migrations completed
✅ Created account: Development Account (dev-account)
  ✅ Created tenant: Customer 1 - Production eBPF Data (customer-1)
    ✅ Created dataset: eBPF Observability Data (ebpf-data)
    ✅ Created dataset: sFlow Network Monitoring Data (sflow-data)
  ✅ Created tenant: Development Tenant Alpha (tenant-001)
    ✅ Created dataset: Web Analytics (dataset-001)
    ✅ Created dataset: User Events (dataset-004)
    ✅ Created dataset: Sales Data (dataset-007)
  ... (etc)

📊 Database Population Summary:
  Account: Development Account (dev-account)
  Tenants: 4
  Datasets: 11

🔗 Control Service API Usage:
  List tenants:   GET http://localhost:8080/api/v1/accounts/dev-account/tenants
  Get tenant:     GET http://localhost:8080/api/v1/accounts/dev-account/tenants/customer-1
  List datasets:  GET http://localhost:8080/api/v1/accounts/dev-account/tenants/customer-1/datasets

⚙️  Service Configuration:
control_service:
  enabled: true
  base_url: http://localhost:8080
  account_id: dev-account
  api_key: your-api-key

💾 Summary exported to: fake_data_summary.json
✅ Database population complete!
```

## Using with ByteFreezer Services

After populating the database, update your service configurations:

### Receiver Configuration

```yaml
control_service:
  enabled: true
  base_url: "http://localhost:8080"
  account_id: "dev-account"
  api_key: "your-api-key"
  timeout_seconds: 30

# Remove or set to false
dev: false
bytefreezer:
  controller: ""  # No longer needed
```

### Packer Configuration

```yaml
control_service:
  enabled: true
  base_url: "http://localhost:8080"
  account_id: "dev-account"
  api_key: "your-api-key"
  timeout_seconds: 30

# Remove or set to false
dev: false
bytefreezer:
  controller: ""  # No longer needed
```

### Piper Configuration

```yaml
control_service:
  enabled: true
  base_url: "http://localhost:8080"
  account_id: "dev-account"
  api_key: "your-api-key"
  timeout_seconds: 30

# Remove or set to false
dev: false
pipeline:
  controller_endpoint: ""  # No longer needed
```

## Testing the Integration

1. **Start Control Service:**
   ```bash
   cd /home/andrew/workspace/bytefreezer/bytefreezer-control
   ./bytefreezer-control --config config.yaml
   ```

2. **Verify tenant data:**
   ```bash
   # List tenants
   curl http://localhost:8080/api/v1/accounts/dev-account/tenants

   # Get specific tenant
   curl http://localhost:8080/api/v1/accounts/dev-account/tenants/customer-1

   # List datasets
   curl http://localhost:8080/api/v1/accounts/dev-account/tenants/customer-1/datasets
   ```

3. **Start receiver/packer/piper with Control Service enabled:**
   ```bash
   cd /home/andrew/workspace/bytefreezer/bytefreezer-receiver
   ./bytefreezer-receiver --config config.yaml
   # Check logs for: "Successfully fetched X tenants from Control Service"
   ```

## Bearer Token Mapping

The fake data includes bearer tokens that match the dev mode:

- `customer-1`: `eb4ba9e3236eaefae736495bd79f0d6de753bd7b6547b184ac0c439ff76982cf`
- `tenant-001`: `bearer-token-tenant-001-dev`
- `tenant-002`: `bearer-token-tenant-002-dev`
- `tenant-003`: `bearer-token-tenant-003-dev`

These tokens are stored in the tenant configuration JSONB field under `bearer_token`.

## Dataset IDs

All dataset IDs match the fake data used across services:

- eBPF data: `ebpf-data`
- sFlow data: `sflow-data`
- Web Analytics: `dataset-001`, `dataset-005`, `dataset-006`
- User Events: `dataset-002`, `dataset-004`, `dataset-009`
- Sales Data: `dataset-003`, `dataset-007`, `dataset-008`

## Notes

- The script creates data with the same structure as fake data in dev mode
- All tenants are created as `active: true`
- All datasets include S3 destination configuration
- Bearer tokens are stored in tenant config JSONB for backward compatibility
- The script is idempotent - it will fail if data already exists (use this to avoid duplicates)

## Cleanup

To reset the database:

```bash
# Drop and recreate database
docker exec -it bytefreezer-postgres psql -U postgres -c "DROP DATABASE bytefreezer;"
docker exec -it bytefreezer-postgres psql -U postgres -c "CREATE DATABASE bytefreezer;"

# Re-run population script
go run populate_fake_data_simple.go "postgres://postgres:postgres@localhost:5432/bytefreezer?sslmode=disable"
```
