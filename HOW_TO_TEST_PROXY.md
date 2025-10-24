> ok tell me how can i test configuration poll from control? 

● Testing Configuration Polling - Complete Guide

  Prerequisites Setup

  1. Run Database Migration
  cd /home/andrew/workspace/bytefreezer/bytefreezer-control
  PGPASSWORD=bytefreezer123 psql -h 192.168.86.137 -U bytefreezer -d bytefreezer \
    -f migrations/004_proxy_configuration.sql

  2. Verify Tables Created
  PGPASSWORD=bytefreezer123 psql -h 192.168.86.137 -U bytefreezer -d bytefreezer \
    -c "\d proxy_instances"

  Test Scenario 1: Basic Configuration Polling

  Step 1: Start Control Service
  cd /home/andrew/workspace/bytefreezer/bytefreezer-control
  ./bytefreezer-control

  Step 2: Get Your Proxy Instance Details
  # Get hostname (this becomes instance_id)
  hostname
  # Example output: prod-proxy-01

  # Get tenant_id from your proxy config
  grep tenant_id /path/to/proxy/config.yaml
  # Example: tenant_id: "my-tenant"

  Step 3: Create Initial Proxy Configuration in Control
  # Replace these values:
  # - prod-proxy-01 with your hostname
  # - my-tenant with your tenant_id
  # - YOUR_BEARER_TOKEN with your bearer token

  curl -X PUT http://192.168.86.137:8080/api/v2/proxies/prod-proxy-01/config \
    -H "Content-Type: application/json" \
    -d '{
    "tenant_id": "my-tenant",
    "instance_api": "prod-proxy-01:8080",
    "config_mode": "hybrid",
    "plugin_configs": [
      {
        "type": "sflow",
        "name": "sflow-listener",
        "config": {
          "host": "0.0.0.0",
          "port": 2055,
          "dataset_id": "sflow-data",
          "protocol": "sflow",
          "data_hint": "ndjson",
          "read_buffer_size": 65536,
          "worker_count": 4
        }
      }
    ],
    "proxy_settings": {
      "receiver": {
        "base_url": "http://receiver:8081",
        "timeout_seconds": 30,
        "upload_worker_count": 5
      },
      "batching": {
        "enabled": true,
        "max_lines": 10000,
        "max_bytes": 1048576,
        "timeout_seconds": 30,
        "compression_enabled": true,
        "compression_level": 6
      }
    }
  }'

  Expected Response:
  {
    "instance_id": "prod-proxy-01",
    "tenant_id": "my-tenant",
    "config_mode": "hybrid",
    "config_version": 1,
    "config_hash": "a1b2c3d4...",
    "config_applied": false,
    ...
  }

  Step 4: Configure Proxy for Polling

  Edit your proxy config.yaml:
  # Configuration mode
  config_mode: "hybrid"  # or "control-only" for testing

  # Control service URL
  control_url: "http://192.168.86.137:8080"

  # Config polling settings
  config_polling:
    enabled: true
    interval_seconds: 60  # Poll every minute for testing
    timeout_seconds: 30
    cache_directory: "/var/cache/bytefreezer-proxy"
    retry_on_error: true

  # Your tenant and auth
  tenant_id: "my-tenant"
  bearer_token: "your-bearer-token"

  # Start with minimal local config
  inputs: []  # Empty - will be loaded from control

  Step 5: Start Proxy and Watch Logs
  cd /home/andrew/workspace/bytefreezer/bytefreezer-proxy
  ./bytefreezer-proxy 2>&1 | grep -E "(Config polling|Configuration|Polling|Applied)"

  Expected Log Output:
  Config polling service starting (mode: hybrid, interval: 1m0s)
  Config polling enabled - polling every 1m0s
  Polling configuration from control: http://192.168.86.137:8080
  Configuration changed (version 0 -> 1)
  Applying new configuration (version 1)
  Reloading plugin service with new configuration
  Plugin service reloaded successfully with 1 plugins
  Configuration cached to /var/cache/bytefreezer-proxy/my-tenant-prod-proxy-01.json
  Configuration applied successfully (version 1)
  Reported config version 1 as applied to control

  Step 6: Verify Configuration Was Applied
  # Check control database
  PGPASSWORD=bytefreezer123 psql -h 192.168.86.137 -U bytefreezer -d bytefreezer \
    -c "SELECT instance_id, config_version, config_applied, config_applied_at 
        FROM proxy_instances 
        WHERE instance_id = 'prod-proxy-01';"

  Expected:
   instance_id   | config_version | config_applied |    config_applied_at     
  ---------------+----------------+----------------+--------------------------
   prod-proxy-01 |              1 | t              | 2025-10-21 10:05:23.123

  Test Scenario 2: Configuration Update & Dynamic Reload

  Step 1: Update Configuration in Control
  # Add a second plugin (ipfix on different port)
  curl -X PUT http://192.168.86.137:8080/api/v2/proxies/prod-proxy-01/config \
    -H "Content-Type: application/json" \
    -d '{
    "tenant_id": "my-tenant",
    "instance_api": "prod-proxy-01:8080",
    "config_mode": "hybrid",
    "plugin_configs": [
      {
        "type": "sflow",
        "name": "sflow-listener",
        "config": {
          "host": "0.0.0.0",
          "port": 2055,
          "dataset_id": "sflow-data"
        }
      },
      {
        "type": "ipfix",
        "name": "ipfix-listener",
        "config": {
          "host": "0.0.0.0",
          "port": 2056,
          "dataset_id": "ipfix-data"
        }
      }
    ],
    "proxy_settings": {
      "receiver": {
        "base_url": "http://receiver:8081"
      }
    }
  }'

  Step 2: Watch Proxy Logs (next poll cycle)

  Within 60 seconds, you should see:
  Polling configuration from control: http://192.168.86.137:8080
  Configuration changed (version 1 -> 2)
  Applying new configuration (version 2)
  Stopping plugin manager
  Reloading plugin service with new configuration
  Starting plugin: sflow-listener (sflow)
  Starting plugin: ipfix-listener (ipfix)
  Plugin service reloaded successfully with 2 plugins
  Configuration applied successfully (version 2)
  Reported config version 2 as applied to control

  Step 3: Verify Plugins Are Running
  # Check proxy API
  curl http://localhost:8080/api/v1/health | jq '.plugins'

  Expected:
  {
    "total_plugins": 2,
    "active_plugins": 2,
    "plugin_details": [
      {
        "name": "sflow-listener",
        "type": "sflow",
        "status": "running",
        "port": 2055
      },
      {
        "name": "ipfix-listener",
        "type": "ipfix",
        "status": "running",
        "port": 2056
      }
    ]
  }

  Test Scenario 3: Port Conflict Resolution

  Step 1: Create Port Conflict
  # Update config to use same port (2055) for both
  curl -X PUT http://192.168.86.137:8080/api/v2/proxies/prod-proxy-01/config \
    -H "Content-Type: application/json" \
    -d '{
    "tenant_id": "my-tenant",
    "instance_api": "prod-proxy-01:8080",
    "config_mode": "hybrid",
    "plugin_configs": [
      {
        "type": "ipfix",
        "name": "ipfix-now-on-2055",
        "config": {
          "host": "0.0.0.0",
          "port": 2055,
          "dataset_id": "ipfix-data"
        }
      }
    ]
  }'

  Step 2: Watch Proxy Logs
  Polling configuration from control
  Configuration changed (version 2 -> 3)
  Applying new configuration (version 3)
  Port conflict: Port 2055: local 'sflow[sflow-listener]' vs remote 'ipfix-now-on-2055' - REMOTE TAKES PRECEDENCE
  Stopping plugin manager
  Reloading plugin service with new configuration
  Plugin service reloaded successfully with 1 plugins
  Configuration applied successfully (version 3)

  Test Scenario 4: Control Unreachable (Hybrid Mode Fallback)

  Step 1: Stop Control Service
  # Stop control (Ctrl+C in control terminal)

  Step 2: Watch Proxy Logs
  Polling configuration from control
  Config poll failed: failed to poll control: dial tcp 192.168.86.137:8080: connection refused
  Control unreachable, using cached config
  Loaded cached configuration (version 3) from /var/cache/bytefreezer-proxy/my-tenant-prod-proxy-01.json

  Step 3: Verify Proxy Still Running
  curl http://localhost:8080/api/v1/health
  # Should still return healthy status

  Step 4: Restart Control
  cd /home/andrew/workspace/bytefreezer/bytefreezer-control
  ./bytefreezer-control

  Step 5: Proxy Reconnects on Next Poll
  Polling configuration from control: http://192.168.86.137:8080
  Configuration unchanged (version 3, hash abc123...)

  Test Scenario 5: View Configuration History

  Get Configuration History:
  curl "http://192.168.86.137:8080/api/v2/proxies/prod-proxy-01/config/history?tenant_id=my-tenant&limit=5" | jq

  Expected Response:
  {
    "history": [
      {
        "id": 3,
        "instance_id": "prod-proxy-01",
        "config_version": 2,
        "plugin_configs": [...],
        "timestamp": "2025-10-21T10:10:00Z"
      },
      {
        "id": 2,
        "instance_id": "prod-proxy-01",
        "config_version": 1,
        "plugin_configs": [...],
        "timestamp": "2025-10-21T10:05:00Z"
      }
    ],
    "count": 2
  }

  Debugging Commands

  Check Cache File:
  cat /var/cache/bytefreezer-proxy/my-tenant-$(hostname).json | jq

  Check Database State:
  PGPASSWORD=bytefreezer123 psql -h 192.168.86.137 -U bytefreezer -d bytefreezer -c "
  SELECT 
    instance_id,
    config_mode,
    config_version,
    config_applied,
    config_applied_at,
    updated_at,
    jsonb_array_length(plugin_configs) as plugin_count
  FROM proxy_instances;"

  Check Proxy Stats:
  curl http://localhost:8080/api/v1/config | jq '.plugins'

  Monitor Continuous Polling:
  # Terminal 1: Watch proxy logs
  tail -f /path/to/proxy.log | grep -E "Config|Polling"

  # Terminal 2: Update config every 30 seconds
  watch -n 30 'curl -X PUT http://192.168.86.137:8080/api/v2/proxies/prod-proxy-01/config -H "Content-Type: application/json" -d "{...}"'

  Testing Checklist

  - Database migration applied successfully
  - Control service started and accessible
  - Initial proxy configuration created in control
  - Proxy polls and applies configuration (version 1)
  - Configuration marked as applied in database
  - Configuration update triggers reload (version 2)
  - Plugins dynamically reload without restart
  - Port conflicts detected and resolved
  - Hybrid mode falls back to cache when control down
  - Configuration history tracking works
  - Cache file created with correct permissions (0600)

  This gives you a complete end-to-end testing workflow for the configuration polling system!
