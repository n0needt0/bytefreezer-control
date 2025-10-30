# ByteFreezer Error Reporting System - Deployment Checklist

## Pre-Deployment

### 1. Database Migrations

**Status**: ✅ Already created (migrations 11-14)

Verify migrations are applied on tp3:

```bash
ssh tp3
psql -h 192.168.86.137 -U bytefreezer -d bytefreezer -c "SELECT version, name FROM control_migrations WHERE version >= 11 ORDER BY version;"
```

Expected output:
```
 version |           name
---------+---------------------------
      11 | error_tracking
      12 | error_tracking_trigger
      13 | error_tracking_upsert
      14 | error_tracking_upsert_func
```

If migrations are not applied, run on control service:
```bash
cd /home/andrew/workspace/bytefreezer/bytefreezer-control
./bytefreezer-control --migrate
```

### 2. Build All Services

```bash
# Control (tp3)
cd /home/andrew/workspace/bytefreezer/bytefreezer-control
go build -o bytefreezer-control

# Packer (tp2)
cd /home/andrew/workspace/bytefreezer/bytefreezer-packer
go build -o bytefreezer-packer

# Receiver (tp2)
cd /home/andrew/workspace/bytefreezer/bytefreezer-receiver
go build -o bytefreezer-receiver

# Piper (tp1)
cd /home/andrew/workspace/bytefreezer/bytefreezer-piper
go build -o bytefreezer-piper

# Proxy (tp1)
cd /home/andrew/workspace/bytefreezer/bytefreezer-proxy
go build -o bytefreezer-proxy
```

### 3. Configuration Updates

**Status**: ✅ All config files updated with `error_tracking.enabled: true`

Verify configurations:

```bash
# Packer (tp2)
grep -A2 "error_tracking" /home/andrew/workspace/bytefreezer/bytefreezer-packer/config.yaml

# Receiver (tp2)
grep -A2 "error_tracking" /home/andrew/workspace/bytefreezer/bytefreezer-receiver/config.yaml

# Piper (tp1)
grep -A2 "error_tracking" /home/andrew/workspace/bytefreezer/bytefreezer-piper/config.yaml

# Proxy (tp1)
grep -A2 "error_tracking" /home/andrew/workspace/bytefreezer/bytefreezer-proxy/config.yaml
```

All should show:
```yaml
error_tracking:
  enabled: true
```

## Deployment Steps

### Step 1: Deploy Control Service (tp3)

```bash
# Stop service
ssh tp3 "sudo systemctl stop bytefreezer-control"

# Deploy new binary
scp /home/andrew/workspace/bytefreezer/bytefreezer-control/bytefreezer-control tp3:/tmp/
ssh tp3 "sudo mv /tmp/bytefreezer-control /usr/local/bin/ && sudo chmod +x /usr/local/bin/bytefreezer-control"

# Deploy new config (if not using Ansible)
scp /home/andrew/workspace/bytefreezer/bytefreezer-control/config.yaml tp3:/tmp/
ssh tp3 "sudo cp /tmp/config.yaml /etc/bytefreezer/control/config.yaml"

# Start service
ssh tp3 "sudo systemctl start bytefreezer-control"

# Verify service is running
ssh tp3 "sudo systemctl status bytefreezer-control"
ssh tp3 "curl -s http://localhost:8082/api/v1/health | jq"
```

**Verify error reporting endpoint is available:**
```bash
curl -s -H "Authorization: Bearer bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c" \
  http://192.168.86.103:8082/api/v1/errors/stats | jq
```

Expected: Should return error statistics (even if empty)

### Step 2: Deploy Packer (tp2)

```bash
# Stop service
ssh tp2 "sudo systemctl stop bytefreezer-packer"

# Deploy new binary
scp /home/andrew/workspace/bytefreezer/bytefreezer-packer/bytefreezer-packer tp2:/tmp/
ssh tp2 "sudo mv /tmp/bytefreezer-packer /usr/local/bin/ && sudo chmod +x /usr/local/bin/bytefreezer-packer"

# Deploy new config
scp /home/andrew/workspace/bytefreezer/bytefreezer-packer/config.yaml tp2:/tmp/
ssh tp2 "sudo cp /tmp/config.yaml /etc/bytefreezer/packer/config.yaml"

# Start service
ssh tp2 "sudo systemctl start bytefreezer-packer"

# Verify service is running
ssh tp2 "sudo systemctl status bytefreezer-packer"
ssh tp2 "journalctl -u bytefreezer-packer -n 50 --no-pager" | grep -i error
```

### Step 3: Deploy Receiver (tp2)

```bash
# Stop service
ssh tp2 "sudo systemctl stop bytefreezer-receiver"

# Deploy new binary
scp /home/andrew/workspace/bytefreezer/bytefreezer-receiver/bytefreezer-receiver tp2:/tmp/
ssh tp2 "sudo mv /tmp/bytefreezer-receiver /usr/local/bin/ && sudo chmod +x /usr/local/bin/bytefreezer-receiver"

# Deploy new config
scp /home/andrew/workspace/bytefreezer/bytefreezer-receiver/config.yaml tp2:/tmp/
ssh tp2 "sudo cp /tmp/config.yaml /etc/bytefreezer/receiver/config.yaml"

# Start service
ssh tp2 "sudo systemctl start bytefreezer-receiver"

# Verify service is running
ssh tp2 "sudo systemctl status bytefreezer-receiver"
```

### Step 4: Deploy Piper (tp1)

```bash
# Stop service
ssh tp1 "sudo systemctl stop bytefreezer-piper"

# Deploy new binary
scp /home/andrew/workspace/bytefreezer/bytefreezer-piper/bytefreezer-piper tp1:/tmp/
ssh tp1 "sudo mv /tmp/bytefreezer-piper /usr/local/bin/ && sudo chmod +x /usr/local/bin/bytefreezer-piper"

# Deploy new config
scp /home/andrew/workspace/bytefreezer/bytefreezer-piper/config.yaml tp1:/tmp/
ssh tp1 "sudo cp /tmp/config.yaml /etc/bytefreezer/piper/config.yaml"

# Start service
ssh tp1 "sudo systemctl start bytefreezer-piper"

# Verify service is running
ssh tp1 "sudo systemctl status bytefreezer-piper"
```

### Step 5: Deploy Proxy (tp1)

```bash
# Stop service
ssh tp1 "sudo systemctl stop bytefreezer-proxy"

# Deploy new binary
scp /home/andrew/workspace/bytefreezer/bytefreezer-proxy/bytefreezer-proxy tp1:/tmp/
ssh tp1 "sudo mv /tmp/bytefreezer-proxy /usr/local/bin/ && sudo chmod +x /usr/local/bin/bytefreezer-proxy"

# Deploy new config
scp /home/andrew/workspace/bytefreezer/bytefreezer-proxy/config.yaml tp1:/tmp/
ssh tp1 "sudo cp /tmp/config.yaml /etc/bytefreezer/proxy/config.yaml"

# Start service
ssh tp1 "sudo systemctl start bytefreezer-proxy"

# Verify service is running
ssh tp1 "sudo systemctl status bytefreezer-proxy"
```

## Post-Deployment Testing

### 1. Run Test Script

```bash
cd /home/andrew/workspace/bytefreezer/bytefreezer-control/scripts

# Run Python test script (recommended)
python3 test-error-reporting.py

# Or run Bash test script
./test-error-reporting.sh
```

### 2. Verify in Database

```bash
psql -h 192.168.86.137 -U bytefreezer -d bytefreezer <<EOF
-- Check that errors were recorded
SELECT component, error_type, severity, occurrence_count, sample_rate, last_seen
FROM system_errors
ORDER BY last_seen DESC
LIMIT 10;

-- Check error statistics
SELECT
  component,
  COUNT(*) as error_types,
  SUM(occurrence_count) as total_occurrences
FROM system_errors
GROUP BY component
ORDER BY total_occurrences DESC;

-- Check sampling is working
SELECT
  error_type,
  occurrence_count,
  sample_rate,
  samples_collected,
  samples_dropped
FROM system_errors
WHERE samples_dropped > 0
ORDER BY occurrence_count DESC;
EOF
```

### 3. Verify UI

1. Navigate to: http://192.168.86.103:8082/dashboard/errors
2. Login with admin account
3. Verify:
   - Statistics cards show correct counts
   - Errors list displays test errors
   - Filters work (Severity, Component, Status)
   - Expandable error details show samples and metadata
   - Account filtering works (if logged in as non-admin)

### 4. Verify API Endpoints

```bash
# Get error statistics
curl -s -H "Authorization: Bearer bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c" \
  http://192.168.86.103:8082/api/v1/errors/stats | jq

# List errors
curl -s -H "Authorization: Bearer bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c" \
  "http://192.168.86.103:8082/api/v1/errors?limit=5" | jq

# List only critical errors
curl -s -H "Authorization: Bearer bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c" \
  "http://192.168.86.103:8082/api/v1/errors?severity=critical&limit=10" | jq
```

## Monitoring

### Key Metrics to Monitor

1. **Total Active Errors**
   ```sql
   SELECT COUNT(*) FROM system_errors WHERE status = 'active';
   ```

2. **Critical Errors**
   ```sql
   SELECT component, error_type, occurrence_count, last_seen
   FROM system_errors
   WHERE severity = 'critical' AND status = 'active'
   ORDER BY last_seen DESC;
   ```

3. **High Frequency Errors** (may need investigation)
   ```sql
   SELECT component, error_type, occurrence_count, sample_rate
   FROM system_errors
   WHERE occurrence_count > 100
   ORDER BY occurrence_count DESC
   LIMIT 10;
   ```

4. **Sampling Efficiency**
   ```sql
   SELECT
     component,
     AVG(sample_rate) as avg_sample_rate,
     SUM(samples_collected) as total_collected,
     SUM(samples_dropped) as total_dropped
   FROM system_errors
   GROUP BY component;
   ```

### Service Logs

Monitor service logs for error reporting issues:

```bash
# Control service
ssh tp3 "journalctl -u bytefreezer-control -f" | grep -i "error"

# Packer service
ssh tp2 "journalctl -u bytefreezer-packer -f" | grep -i "error"

# Receiver service
ssh tp2 "journalctl -u bytefreezer-receiver -f" | grep -i "error"

# Piper service
ssh tp1 "journalctl -u bytefreezer-piper -f" | grep -i "error"

# Proxy service
ssh tp1 "journalctl -u bytefreezer-proxy -f" | grep -i "error"
```

## Rollback Plan

If issues occur, rollback to previous version:

### 1. Stop Services
```bash
ssh tp3 "sudo systemctl stop bytefreezer-control"
ssh tp2 "sudo systemctl stop bytefreezer-packer bytefreezer-receiver"
ssh tp1 "sudo systemctl stop bytefreezer-piper bytefreezer-proxy"
```

### 2. Restore Previous Binaries
```bash
# Restore from backup (assuming backups were made)
ssh tp3 "sudo cp /usr/local/bin/bytefreezer-control.backup /usr/local/bin/bytefreezer-control"
ssh tp2 "sudo cp /usr/local/bin/bytefreezer-packer.backup /usr/local/bin/bytefreezer-packer"
ssh tp2 "sudo cp /usr/local/bin/bytefreezer-receiver.backup /usr/local/bin/bytefreezer-receiver"
ssh tp1 "sudo cp /usr/local/bin/bytefreezer-piper.backup /usr/local/bin/bytefreezer-piper"
ssh tp1 "sudo cp /usr/local/bin/bytefreezer-proxy.backup /usr/local/bin/bytefreezer-proxy"
```

### 3. Restore Previous Configs
```bash
ssh tp3 "sudo cp /etc/bytefreezer/control/config.yaml.backup /etc/bytefreezer/control/config.yaml"
# ... repeat for other services
```

### 4. Start Services
```bash
ssh tp3 "sudo systemctl start bytefreezer-control"
ssh tp2 "sudo systemctl start bytefreezer-packer bytefreezer-receiver"
ssh tp1 "sudo systemctl start bytefreezer-piper bytefreezer-proxy"
```

**Note**: Database migrations cannot be rolled back easily. The system_errors table can be left in place without harm.

## Troubleshooting

### Error: "error reporting service not available"

**Cause**: Control service not initialized properly

**Fix**:
```bash
ssh tp3 "sudo systemctl restart bytefreezer-control"
ssh tp3 "journalctl -u bytefreezer-control -n 100 --no-pager" | grep -i "error reporting"
```

Should see: `Error reporting service initialized`

### Error: "failed to report error: connection refused"

**Cause**: Service cannot reach control service

**Fix**:
1. Verify control service is running: `curl http://192.168.86.103:8082/api/v1/health`
2. Check network connectivity: `ping 192.168.86.103`
3. Verify firewall rules allow traffic to port 8082
4. Check config: `control_service.base_url` is correct

### Error: "invalid token" or "authentication required"

**Cause**: Wrong API key or JWT token

**Fix**:
1. System services: Verify `control_service.api_key` matches control service configuration
2. Proxy: Ensure account JWT token is valid and not expired

### High Sample Drop Rate

**Expected Behavior**: For high-frequency errors (1000+ occurrences), sampling reduces automatically

**Action Required**: Only if sample_rate is low but occurrence_count is also low
- Check logs for error reporting failures
- Verify network connectivity to control service

## Success Criteria

✅ All services start without errors
✅ Test script reports all errors successfully
✅ Database contains test errors with correct deduplication
✅ UI displays errors correctly with filtering
✅ API endpoints return expected data
✅ Service logs show "Error reporting service initialized"
✅ No authentication failures in logs
✅ Adaptive sampling is working (sample_rate decreases for high-frequency errors)

## Support

For issues:
1. Check service logs: `journalctl -u bytefreezer-{service} -n 100`
2. Check database: Query `system_errors` table
3. Run test script: `python3 test-error-reporting.py`
4. Review ERROR_REPORTING_GUIDE.md for detailed documentation
