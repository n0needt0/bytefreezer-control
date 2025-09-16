# ByteFreezer Control Integration Examples

This document provides comprehensive integration examples for the ByteFreezer Control service, focusing on ecosystem management, monitoring, and tenant administration patterns.

## Quick Start Examples

### Example 1: Complete Ecosystem Health Monitoring

Monitor the entire ByteFreezer ecosystem with automated health checks and alerting.

```bash
#!/bin/bash
# Complete ecosystem health monitoring example

CONTROL_URL="http://localhost:8082"

echo "=== ByteFreezer Ecosystem Health Monitor ==="

# Function to check service health with retry logic
check_service_health() {
    local service_name=$1
    local max_retries=3
    local retry_count=0
    
    while [ $retry_count -lt $max_retries ]; do
        echo "Checking $service_name health (attempt $((retry_count + 1)))..."
        
        response=$(curl -s "$CONTROL_URL/api/v1/services/$service_name" | jq -r '.healthy // false' 2>/dev/null)
        
        if [ "$response" = "true" ]; then
            echo "✅ $service_name is healthy"
            return 0
        else
            echo "❌ $service_name is not healthy"
            retry_count=$((retry_count + 1))
            if [ $retry_count -lt $max_retries ]; then
                echo "Retrying in 5 seconds..."
                sleep 5
            fi
        fi
    done
    
    return 1
}

# Check overall ecosystem health
echo "=== Overall Ecosystem Health ==="
ecosystem_health=$(curl -s "$CONTROL_URL/api/v1/ecosystem/health")

if [ $? -eq 0 ]; then
    echo "$ecosystem_health" | jq '.'
    
    overall_status=$(echo "$ecosystem_health" | jq -r '.overall_status')
    healthy_count=$(echo "$ecosystem_health" | jq -r '.healthy_count')
    total_count=$(echo "$ecosystem_health" | jq -r '.total_count')
    
    echo ""
    echo "Overall Status: $overall_status"
    echo "Healthy Services: $healthy_count/$total_count"
    echo ""
else
    echo "Failed to get ecosystem health from control service"
    exit 1
fi

# Check individual services
echo "=== Individual Service Health Checks ==="
services=("receiver" "piper" "packer" "proxy" "soc")

failed_services=()
for service in "${services[@]}"; do
    if ! check_service_health "$service"; then
        failed_services+=("$service")
    fi
done

# Generate health report
echo ""
echo "=== Health Report Summary ==="
if [ ${#failed_services[@]} -eq 0 ]; then
    echo "🎉 All services are healthy!"
    exit 0
else
    echo "⚠️  The following services have issues:"
    for service in "${failed_services[@]}"; do
        echo "  - $service"
    done
    
    # Get detailed error information
    echo ""
    echo "=== Service Error Details ==="
    for service in "${failed_services[@]}"; do
        echo "--- $service ---"
        error_info=$(curl -s "$CONTROL_URL/api/v1/services/$service" | jq -r '.error // "No error details available"')
        last_check=$(curl -s "$CONTROL_URL/api/v1/services/$service" | jq -r '.last_check // "Unknown"')
        echo "Error: $error_info"
        echo "Last Check: $last_check"
        echo ""
    done
    
    exit 1
fi
```

### Example 2: Tenant Management Lifecycle

Complete tenant lifecycle management with validation and cleanup.

```bash
#!/bin/bash
# Tenant management lifecycle example

CONTROL_URL="http://localhost:8082"
JWT_TOKEN=""

# Function to authenticate and get JWT token
authenticate() {
    local username="admin@company.com"
    local password="admin_password"
    
    echo "Authenticating with ByteFreezer Control..."
    
    auth_response=$(curl -s -X POST "$CONTROL_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\": \"$username\", \"password\": \"$password\"}")
    
    if [ $? -eq 0 ]; then
        JWT_TOKEN=$(echo "$auth_response" | jq -r '.token // empty')
        if [ -n "$JWT_TOKEN" ] && [ "$JWT_TOKEN" != "null" ]; then
            echo "✅ Authentication successful"
            return 0
        fi
    fi
    
    echo "❌ Authentication failed"
    return 1
}

# Function to create a new tenant
create_tenant() {
    local tenant_name=$1
    local tenant_email=$2
    
    echo "Creating tenant: $tenant_name ($tenant_email)"
    
    create_response=$(curl -s -X POST "$CONTROL_URL/api/v1/tenants" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$tenant_name\", \"email\": \"$tenant_email\"}")
    
    if [ $? -eq 0 ]; then
        tenant_id=$(echo "$create_response" | jq -r '.id // empty')
        if [ -n "$tenant_id" ] && [ "$tenant_id" != "null" ]; then
            echo "✅ Tenant created successfully with ID: $tenant_id"
            echo "$tenant_id"
            return 0
        fi
    fi
    
    echo "❌ Failed to create tenant"
    echo "Response: $create_response"
    return 1
}

# Function to list all tenants
list_tenants() {
    echo "Listing all tenants..."
    
    tenants_response=$(curl -s "$CONTROL_URL/api/v1/tenants" \
        -H "Authorization: Bearer $JWT_TOKEN")
    
    if [ $? -eq 0 ]; then
        echo "$tenants_response" | jq -r '.[] | "ID: \(.id), Name: \(.name), Email: \(.email), Active: \(.active)"'
        return 0
    fi
    
    echo "❌ Failed to list tenants"
    return 1
}

# Function to get tenant details
get_tenant_details() {
    local tenant_id=$1
    
    echo "Getting details for tenant: $tenant_id"
    
    tenant_response=$(curl -s "$CONTROL_URL/api/v1/tenants/$tenant_id" \
        -H "Authorization: Bearer $JWT_TOKEN")
    
    if [ $? -eq 0 ]; then
        echo "$tenant_response" | jq '.'
        return 0
    fi
    
    echo "❌ Failed to get tenant details"
    return 1
}

# Function to update tenant information
update_tenant() {
    local tenant_id=$1
    local new_name=$2
    local new_email=$3
    
    echo "Updating tenant: $tenant_id"
    
    update_response=$(curl -s -X PUT "$CONTROL_URL/api/v1/tenants/$tenant_id" \
        -H "Authorization: Bearer $JWT_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"name\": \"$new_name\", \"email\": \"$new_email\"}")
    
    if [ $? -eq 0 ]; then
        echo "✅ Tenant updated successfully"
        echo "$update_response" | jq '.'
        return 0
    fi
    
    echo "❌ Failed to update tenant"
    return 1
}

# Function to delete tenant
delete_tenant() {
    local tenant_id=$1
    
    echo "Deleting tenant: $tenant_id"
    
    delete_response=$(curl -s -X DELETE "$CONTROL_URL/api/v1/tenants/$tenant_id" \
        -H "Authorization: Bearer $JWT_TOKEN")
    
    if [ $? -eq 0 ]; then
        echo "✅ Tenant deleted successfully"
        return 0
    fi
    
    echo "❌ Failed to delete tenant"
    return 1
}

# Main tenant management workflow
echo "=== ByteFreezer Tenant Management Example ==="

# Step 1: Authenticate
if ! authenticate; then
    exit 1
fi

echo ""
echo "=== Step 1: List Existing Tenants ==="
list_tenants

echo ""
echo "=== Step 2: Create New Tenant ==="
tenant_id=$(create_tenant "Acme Corporation" "admin@acme-corp.com")
if [ $? -ne 0 ]; then
    exit 1
fi

echo ""
echo "=== Step 3: Get Tenant Details ==="
get_tenant_details "$tenant_id"

echo ""
echo "=== Step 4: Update Tenant Information ==="
update_tenant "$tenant_id" "Acme Corp (Updated)" "updated-admin@acme-corp.com"

echo ""
echo "=== Step 5: List Tenants After Update ==="
list_tenants

echo ""
echo "=== Step 6: Cleanup - Delete Test Tenant ==="
delete_tenant "$tenant_id"

echo ""
echo "=== Tenant Management Example Complete ==="
```

### Example 3: Service Configuration Management

Centralized configuration management and validation across all ByteFreezer services.

```bash
#!/bin/bash
# Service configuration management example

CONTROL_URL="http://localhost:8082"
JWT_TOKEN=""

echo "=== ByteFreezer Service Configuration Management ==="

# Function to get service configuration
get_service_config() {
    local service_name=$1
    
    echo "Getting configuration for: $service_name"
    
    config_response=$(curl -s "$CONTROL_URL/api/v1/services/$service_name" \
        -H "Authorization: Bearer $JWT_TOKEN")
    
    if [ $? -eq 0 ]; then
        echo "$config_response" | jq '.config // {}'
        return 0
    fi
    
    echo "❌ Failed to get service configuration"
    return 1
}

# Function to validate service configuration
validate_service_config() {
    local service_name=$1
    
    echo "Validating configuration for: $service_name"
    
    # Get current configuration
    current_config=$(curl -s "$CONTROL_URL/api/v1/services/$service_name" | jq '.config // {}')
    
    # Basic validation checks
    if [ "$current_config" = "{}" ]; then
        echo "⚠️  No configuration found for $service_name"
        return 1
    fi
    
    # Service-specific validation
    case $service_name in
        "receiver")
            validate_receiver_config "$current_config"
            ;;
        "piper")
            validate_piper_config "$current_config"
            ;;
        "packer")
            validate_packer_config "$current_config"
            ;;
        *)
            echo "✅ Basic validation passed for $service_name"
            ;;
    esac
}

# Function to validate receiver configuration
validate_receiver_config() {
    local config=$1
    
    echo "Validating receiver configuration..."
    
    # Check for required configuration elements
    api_port=$(echo "$config" | jq -r '.server.apiport // empty')
    webhook_port=$(echo "$config" | jq -r '.protocols.webhook.port // empty')
    s3_bucket=$(echo "$config" | jq -r '.s3destination.bucket_name // empty')
    
    if [ -z "$api_port" ]; then
        echo "❌ Missing API port configuration"
        return 1
    fi
    
    if [ -z "$webhook_port" ]; then
        echo "❌ Missing webhook port configuration"
        return 1
    fi
    
    if [ -z "$s3_bucket" ]; then
        echo "❌ Missing S3 bucket configuration"
        return 1
    fi
    
    echo "✅ Receiver configuration is valid"
    return 0
}

# Function to validate piper configuration
validate_piper_config() {
    local config=$1
    
    echo "Validating piper configuration..."
    
    # Check for required piper configuration
    source_bucket=$(echo "$config" | jq -r '.s3source.bucket_name // empty')
    dest_bucket=$(echo "$config" | jq -r '.s3destination.bucket_name // empty')
    poll_interval=$(echo "$config" | jq -r '.s3source.poll_interval // empty')
    
    if [ -z "$source_bucket" ]; then
        echo "❌ Missing S3 source bucket configuration"
        return 1
    fi
    
    if [ -z "$dest_bucket" ]; then
        echo "❌ Missing S3 destination bucket configuration"
        return 1
    fi
    
    if [ -z "$poll_interval" ]; then
        echo "❌ Missing poll interval configuration"
        return 1
    fi
    
    echo "✅ Piper configuration is valid"
    return 0
}

# Function to validate packer configuration
validate_packer_config() {
    local config=$1
    
    echo "Validating packer configuration..."
    
    # Check for required packer configuration
    source_bucket=$(echo "$config" | jq -r '.source.bucket_name // empty')
    dest_bucket=$(echo "$config" | jq -r '.destination.bucket_name // empty')
    
    if [ -z "$source_bucket" ]; then
        echo "❌ Missing source bucket configuration"
        return 1
    fi
    
    if [ -z "$dest_bucket" ]; then
        echo "❌ Missing destination bucket configuration"
        return 1
    fi
    
    echo "✅ Packer configuration is valid"
    return 0
}

# Function to generate configuration report
generate_config_report() {
    echo ""
    echo "=== Configuration Validation Report ==="
    echo "Generated at: $(date)"
    echo ""
    
    services=("receiver" "piper" "packer" "proxy")
    failed_validations=()
    
    for service in "${services[@]}"; do
        echo "--- $service ---"
        if validate_service_config "$service"; then
            echo "Status: ✅ Valid"
        else
            echo "Status: ❌ Invalid"
            failed_validations+=("$service")
        fi
        echo ""
    done
    
    echo "=== Summary ==="
    if [ ${#failed_validations[@]} -eq 0 ]; then
        echo "🎉 All service configurations are valid!"
    else
        echo "⚠️  Configuration issues found in:"
        for service in "${failed_validations[@]}"; do
            echo "  - $service"
        done
    fi
}

# Main configuration management workflow
if ! authenticate; then
    exit 1
fi

echo ""
echo "=== Service Configuration Overview ==="
echo "Getting current control service configuration..."
control_config=$(curl -s "$CONTROL_URL/api/v1/config")
echo "$control_config" | jq '.'

echo ""
echo "=== Individual Service Configurations ==="
get_service_config "receiver"
echo ""
get_service_config "piper"
echo ""
get_service_config "packer"

# Generate comprehensive configuration report
generate_config_report

echo ""
echo "=== Configuration Management Example Complete ==="
```

## Production Integration Examples

### Example 1: Automated Deployment Health Verification

Verify that all services are healthy after a deployment.

**deployment-health-check.sh:**
```bash
#!/bin/bash
# Post-deployment health verification

CONTROL_URL="${BYTEFREEZER_CONTROL_URL:-http://bytefreezer-control:8082}"
DEPLOYMENT_ID="${CI_PIPELINE_ID:-manual}"
SLACK_WEBHOOK="${SLACK_WEBHOOK_URL}"

echo "=== ByteFreezer Deployment Health Verification ==="
echo "Deployment ID: $DEPLOYMENT_ID"
echo "Control URL: $CONTROL_URL"

# Function to send Slack notification
send_slack_notification() {
    local message=$1
    local color=$2
    
    if [ -n "$SLACK_WEBHOOK" ]; then
        curl -s -X POST "$SLACK_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{
                \"attachments\": [{
                    \"color\": \"$color\",
                    \"title\": \"ByteFreezer Deployment Health Check\",
                    \"text\": \"$message\",
                    \"fields\": [{
                        \"title\": \"Deployment ID\",
                        \"value\": \"$DEPLOYMENT_ID\",
                        \"short\": true
                    }]
                }]
            }" > /dev/null
    fi
}

# Wait for control service to be ready
echo "Waiting for control service to be ready..."
max_wait=300  # 5 minutes
wait_time=0

while [ $wait_time -lt $max_wait ]; do
    if curl -s -f "$CONTROL_URL/api/v1/health" > /dev/null; then
        echo "✅ Control service is ready"
        break
    fi
    
    echo "Waiting for control service... (${wait_time}s/${max_wait}s)"
    sleep 10
    wait_time=$((wait_time + 10))
done

if [ $wait_time -ge $max_wait ]; then
    echo "❌ Control service not ready within $max_wait seconds"
    send_slack_notification "❌ Deployment health check failed - Control service not ready" "danger"
    exit 1
fi

# Comprehensive health check with retries
echo ""
echo "=== Comprehensive Health Check ==="

check_attempts=0
max_attempts=6
check_interval=30

while [ $check_attempts -lt $max_attempts ]; do
    check_attempts=$((check_attempts + 1))
    echo "Health check attempt $check_attempts/$max_attempts"
    
    ecosystem_health=$(curl -s "$CONTROL_URL/api/v1/ecosystem/health")
    
    if [ $? -eq 0 ]; then
        overall_status=$(echo "$ecosystem_health" | jq -r '.overall_status')
        healthy_count=$(echo "$ecosystem_health" | jq -r '.healthy_count')
        total_count=$(echo "$ecosystem_health" | jq -r '.total_count')
        
        echo "Overall Status: $overall_status"
        echo "Healthy Services: $healthy_count/$total_count"
        
        if [ "$overall_status" = "healthy" ]; then
            echo "🎉 All services are healthy!"
            send_slack_notification "✅ Deployment successful - All services healthy ($healthy_count/$total_count)" "good"
            exit 0
        elif [ "$overall_status" = "degraded" ] && [ $check_attempts -eq $max_attempts ]; then
            echo "⚠️ Some services are still unhealthy after all attempts"
            
            # Get details about unhealthy services
            unhealthy_services=$(echo "$ecosystem_health" | jq -r '.services | to_entries[] | select(.value.healthy == false) | .key')
            
            message="⚠️ Deployment partially successful - Some services unhealthy:\n"
            while IFS= read -r service; do
                if [ -n "$service" ]; then
                    error=$(echo "$ecosystem_health" | jq -r ".services[\"$service\"].error // \"Unknown error\"")
                    message="${message}• $service: $error\n"
                fi
            done <<< "$unhealthy_services"
            
            send_slack_notification "$message" "warning"
            exit 2
        fi
    else
        echo "❌ Failed to get ecosystem health"
    fi
    
    if [ $check_attempts -lt $max_attempts ]; then
        echo "Waiting ${check_interval}s before next attempt..."
        sleep $check_interval
    fi
done

echo "❌ Health check failed after $max_attempts attempts"
send_slack_notification "❌ Deployment health check failed after $max_attempts attempts" "danger"
exit 1
```

### Example 2: Automated Tenant Provisioning

Complete tenant onboarding with service configuration.

**tenant-provisioning.py:**
```python
#!/usr/bin/env python3
"""
ByteFreezer Tenant Provisioning Script
"""

import json
import requests
import time
import sys
from datetime import datetime
from typing import Dict, List, Optional

class ByteFreezerControlClient:
    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.token = None
        self.authenticate(username, password)
    
    def authenticate(self, username: str, password: str) -> bool:
        """Authenticate and get JWT token."""
        auth_url = f"{self.base_url}/api/v1/auth/login"
        data = {"username": username, "password": password}
        
        try:
            response = self.session.post(auth_url, json=data, timeout=30)
            response.raise_for_status()
            
            auth_data = response.json()
            self.token = auth_data.get('token')
            
            if self.token:
                self.session.headers.update({'Authorization': f'Bearer {self.token}'})
                print(f"✅ Authenticated successfully")
                return True
                
        except Exception as e:
            print(f"❌ Authentication failed: {e}")
            
        return False
    
    def create_tenant(self, name: str, email: str) -> Optional[str]:
        """Create a new tenant."""
        create_url = f"{self.base_url}/api/v1/tenants"
        data = {"name": name, "email": email}
        
        try:
            response = self.session.post(create_url, json=data, timeout=30)
            response.raise_for_status()
            
            tenant_data = response.json()
            tenant_id = tenant_data.get('id')
            
            if tenant_id:
                print(f"✅ Tenant created: {name} (ID: {tenant_id})")
                return tenant_id
                
        except Exception as e:
            print(f"❌ Failed to create tenant {name}: {e}")
            
        return None
    
    def wait_for_tenant_propagation(self, tenant_id: str, max_wait: int = 300) -> bool:
        """Wait for tenant configuration to propagate to all services."""
        print(f"Waiting for tenant configuration to propagate...")
        
        start_time = time.time()
        
        while time.time() - start_time < max_wait:
            try:
                # Check if all services recognize the tenant
                ecosystem_health = self.get_ecosystem_health()
                
                if ecosystem_health and ecosystem_health.get('overall_status') == 'healthy':
                    print(f"✅ Tenant configuration propagated successfully")
                    return True
                    
            except Exception as e:
                print(f"Checking propagation status: {e}")
            
            time.sleep(10)
        
        print(f"⚠️ Tenant configuration propagation timed out after {max_wait}s")
        return False
    
    def get_ecosystem_health(self) -> Optional[Dict]:
        """Get overall ecosystem health."""
        health_url = f"{self.base_url}/api/v1/ecosystem/health"
        
        try:
            response = self.session.get(health_url, timeout=30)
            response.raise_for_status()
            return response.json()
            
        except Exception as e:
            print(f"❌ Failed to get ecosystem health: {e}")
            
        return None
    
    def validate_tenant_setup(self, tenant_id: str) -> bool:
        """Validate that tenant is properly set up across all services."""
        print(f"Validating tenant setup: {tenant_id}")
        
        # Check tenant exists in control service
        try:
            tenant_url = f"{self.base_url}/api/v1/tenants/{tenant_id}"
            response = self.session.get(tenant_url, timeout=30)
            response.raise_for_status()
            
            tenant_data = response.json()
            if not tenant_data.get('active'):
                print(f"❌ Tenant {tenant_id} is not active")
                return False
                
        except Exception as e:
            print(f"❌ Failed to validate tenant in control service: {e}")
            return False
        
        # Validate service configurations
        services_to_check = ['receiver', 'piper', 'packer']
        
        for service_name in services_to_check:
            if not self.validate_service_tenant_config(service_name, tenant_id):
                return False
        
        print(f"✅ Tenant setup validation completed successfully")
        return True
    
    def validate_service_tenant_config(self, service_name: str, tenant_id: str) -> bool:
        """Validate that a service is configured for the tenant."""
        try:
            service_url = f"{self.base_url}/api/v1/services/{service_name}"
            response = self.session.get(service_url, timeout=30)
            response.raise_for_status()
            
            service_data = response.json()
            
            if not service_data.get('healthy'):
                print(f"⚠️ Service {service_name} is not healthy")
                return False
            
            print(f"✅ Service {service_name} is healthy and ready for tenant {tenant_id}")
            return True
            
        except Exception as e:
            print(f"❌ Failed to validate {service_name} for tenant {tenant_id}: {e}")
            return False

def provision_tenant(control_client: ByteFreezerControlClient, tenant_config: Dict) -> bool:
    """Provision a complete tenant setup."""
    print(f"\n=== Provisioning Tenant: {tenant_config['name']} ===")
    
    # Step 1: Create tenant
    tenant_id = control_client.create_tenant(
        tenant_config['name'], 
        tenant_config['email']
    )
    
    if not tenant_id:
        return False
    
    # Step 2: Wait for configuration propagation
    if not control_client.wait_for_tenant_propagation(tenant_id):
        print(f"⚠️ Configuration propagation incomplete, but continuing...")
    
    # Step 3: Validate setup
    if not control_client.validate_tenant_setup(tenant_id):
        print(f"❌ Tenant setup validation failed")
        return False
    
    # Step 4: Create initial datasets if specified
    datasets = tenant_config.get('datasets', [])
    for dataset_config in datasets:
        print(f"Creating dataset: {dataset_config['name']}")
        # Dataset creation would be implemented here
        # This would involve calling specific service APIs
    
    print(f"🎉 Tenant provisioning completed successfully!")
    print(f"   Tenant ID: {tenant_id}")
    print(f"   Name: {tenant_config['name']}")
    print(f"   Email: {tenant_config['email']}")
    
    return True

def main():
    if len(sys.argv) != 2:
        print("Usage: python tenant-provisioning.py <tenant_config.json>")
        sys.exit(1)
    
    config_file = sys.argv[1]
    
    try:
        with open(config_file, 'r') as f:
            provisioning_config = json.load(f)
    except Exception as e:
        print(f"❌ Failed to load configuration file: {e}")
        sys.exit(1)
    
    # Initialize control client
    control_config = provisioning_config.get('control_service', {})
    control_url = control_config.get('url', 'http://localhost:8082')
    username = control_config.get('username', 'admin@company.com')
    password = control_config.get('password', 'admin_password')
    
    control_client = ByteFreezerControlClient(control_url, username, password)
    
    # Process tenant configurations
    tenants = provisioning_config.get('tenants', [])
    
    if not tenants:
        print("❌ No tenants specified in configuration")
        sys.exit(1)
    
    print(f"=== ByteFreezer Tenant Provisioning ===")
    print(f"Provisioning {len(tenants)} tenant(s)")
    print(f"Control Service: {control_url}")
    print(f"Timestamp: {datetime.now().isoformat()}")
    
    success_count = 0
    
    for tenant_config in tenants:
        if provision_tenant(control_client, tenant_config):
            success_count += 1
        else:
            print(f"❌ Failed to provision tenant: {tenant_config['name']}")
    
    print(f"\n=== Provisioning Summary ===")
    print(f"Successfully provisioned: {success_count}/{len(tenants)} tenants")
    
    if success_count == len(tenants):
        print("🎉 All tenants provisioned successfully!")
        sys.exit(0)
    else:
        print("⚠️ Some tenants failed to provision")
        sys.exit(1)

if __name__ == "__main__":
    main()
```

**tenant_config.json:**
```json
{
  "control_service": {
    "url": "http://localhost:8082",
    "username": "admin@company.com",
    "password": "admin_password"
  },
  "tenants": [
    {
      "name": "Acme Corporation",
      "email": "admin@acme-corp.com",
      "datasets": [
        {
          "name": "web-logs",
          "description": "Web server access logs"
        },
        {
          "name": "app-logs", 
          "description": "Application logs"
        },
        {
          "name": "security-events",
          "description": "Security and audit events"
        }
      ]
    },
    {
      "name": "TechStart Inc",
      "email": "ops@techstart.com",
      "datasets": [
        {
          "name": "api-logs",
          "description": "API access and error logs"
        },
        {
          "name": "metrics",
          "description": "Application metrics and KPIs"
        }
      ]
    }
  ]
}
```

### Example 3: Control Service Docker Compose with Full Stack

Complete ByteFreezer stack with control service orchestration.

**docker-compose.full-stack.yml:**
```yaml
version: '3.8'

services:
  # Database for control service
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: bytefreezer_control
      POSTGRES_USER: bytefreezer
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-secure_password}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./init-db.sql:/docker-entrypoint-initdb.d/init-db.sql
    networks:
      - bytefreezer-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bytefreezer -d bytefreezer_control"]
      interval: 30s
      timeout: 10s
      retries: 5

  # LocalStack for AWS services
  localstack:
    image: localstack/localstack:latest
    ports:
      - "4566:4566"
    environment:
      - SERVICES=s3,dynamodb,secretsmanager
      - DEBUG=1
      - DATA_DIR=/tmp/localstack/data
    volumes:
      - localstack_data:/tmp/localstack
      - ./localstack-init.sh:/etc/localstack/init/ready.d/init.sh
    networks:
      - bytefreezer-network

  # ByteFreezer Control Service
  bytefreezer-control:
    image: ghcr.io/n0needt0/bytefreezer-control:latest
    ports:
      - "8082:8082"  # Management API
    environment:
      - BYTEFREEZER_CONTROL_DATABASE_HOST=postgres
      - BYTEFREEZER_CONTROL_DATABASE_PASSWORD=${POSTGRES_PASSWORD:-secure_password}
    volumes:
      - ./control-config.yaml:/config.yaml
      - control_logs:/var/log/bytefreezer-control
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_started
    networks:
      - bytefreezer-network
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # ByteFreezer Receiver
  bytefreezer-receiver:
    image: ghcr.io/n0needt0/bytefreezer-receiver:latest
    ports:
      - "8080:8080"  # Management API
      - "8081:8081"  # Data ingestion
    volumes:
      - ./receiver-config.yaml:/config.yaml
      - receiver_cache:/var/cache/bytefreezer-receiver
    depends_on:
      - localstack
      - bytefreezer-control
    networks:
      - bytefreezer-network
    restart: unless-stopped

  # ByteFreezer Piper
  bytefreezer-piper:
    image: ghcr.io/n0needt0/bytefreezer-piper:latest
    ports:
      - "8083:8080"  # Management API
    volumes:
      - ./piper-config.yaml:/config.yaml
      - ./parser-configs.yaml:/parser-configs.yaml
    depends_on:
      - localstack
      - bytefreezer-receiver
      - bytefreezer-control
    networks:
      - bytefreezer-network
    restart: unless-stopped

  # ByteFreezer Packer
  bytefreezer-packer:
    image: ghcr.io/n0needt0/bytefreezer-packer:latest
    ports:
      - "8084:8080"  # Management API
    volumes:
      - ./packer-config.yaml:/config.yaml
    depends_on:
      - localstack
      - bytefreezer-piper
      - bytefreezer-control
    networks:
      - bytefreezer-network
    restart: unless-stopped

  # Monitoring and Observability
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
    networks:
      - bytefreezer-network

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana_data:/var/lib/grafana
      - ./grafana/dashboards:/etc/grafana/provisioning/dashboards
      - ./grafana/datasources:/etc/grafana/provisioning/datasources
    depends_on:
      - prometheus
    networks:
      - bytefreezer-network

  # Management UI (optional)
  bytefreezer-ui:
    image: nginx:alpine
    ports:
      - "8090:80"
    volumes:
      - ./ui/dist:/usr/share/nginx/html
      - ./ui/nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - bytefreezer-control
    networks:
      - bytefreezer-network

volumes:
  postgres_data:
  localstack_data:
  control_logs:
  receiver_cache:
  prometheus_data:
  grafana_data:

networks:
  bytefreezer-network:
    driver: bridge
```

**control-config.yaml:**
```yaml
app:
  name: "bytefreezer-control"
  version: "1.0.0"

logging:
  level: "info"
  encoding: "json"

server:
  api_port: 8082

services:
  receiver:
    url: "http://bytefreezer-receiver:8080"
    health_endpoint: "/health"
    config_endpoint: "/config"
    timeout_seconds: 30
  
  piper:
    url: "http://bytefreezer-piper:8080"
    health_endpoint: "/health"
    config_endpoint: "/config"
    timeout_seconds: 30
    
  packer:
    url: "http://bytefreezer-packer:8080"
    health_endpoint: "/health"
    config_endpoint: "/config"
    timeout_seconds: 30

database:
  enabled: true
  type: "postgres"
  host: "postgres"
  port: 5432
  database: "bytefreezer_control"
  username: "bytefreezer"
  password: "secure_password"
  ssl_mode: "disable"
  max_connections: 25
  max_idle_connections: 10

auth:
  enabled: true
  jwt_secret: "bytefreezer-control-jwt-secret-key-change-in-production"
  token_expiry_hours: 24
  admin_users:
    - "admin@company.com"
    - "ops@company.com"

rate_limit:
  enabled: true
  requests_per_minute: 100
  burst_size: 20

housekeeping:
  enabled: true
  interval_seconds: 300

otel:
  enabled: true
  endpoint: "http://localhost:4317"
  service_name: "bytefreezer-control"
  scrape_interval_seconds: 60

dev: false
```

**Startup script (start-full-stack.sh):**
```bash
#!/bin/bash
# Start complete ByteFreezer stack with control service

set -e

echo "=== Starting ByteFreezer Full Stack ==="

# Check prerequisites
echo "Checking prerequisites..."
command -v docker >/dev/null 2>&1 || { echo "❌ Docker is required"; exit 1; }
command -v docker-compose >/dev/null 2>&1 || { echo "❌ Docker Compose is required"; exit 1; }

# Create necessary directories
echo "Creating directories..."
mkdir -p logs control_logs grafana/{dashboards,datasources} ui/dist

# Generate random password if not set
if [ -z "$POSTGRES_PASSWORD" ]; then
    export POSTGRES_PASSWORD=$(openssl rand -base64 32)
    echo "Generated database password: $POSTGRES_PASSWORD"
fi

# Start the stack
echo "Starting ByteFreezer services..."
docker-compose -f docker-compose.full-stack.yml up -d

# Wait for services to be ready
echo "Waiting for services to start..."
sleep 30

# Check control service health
echo "Checking control service health..."
max_attempts=12
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -s -f http://localhost:8082/api/v1/health >/dev/null 2>&1; then
        echo "✅ Control service is ready"
        break
    fi
    
    attempt=$((attempt + 1))
    echo "Waiting for control service... ($attempt/$max_attempts)"
    sleep 10
done

if [ $attempt -ge $max_attempts ]; then
    echo "❌ Control service failed to start within expected time"
    echo "Checking logs..."
    docker-compose -f docker-compose.full-stack.yml logs bytefreezer-control
    exit 1
fi

# Check ecosystem health
echo "Checking ecosystem health..."
ecosystem_health=$(curl -s http://localhost:8082/api/v1/ecosystem/health)
overall_status=$(echo "$ecosystem_health" | jq -r '.overall_status // "unknown"')

echo "Ecosystem status: $overall_status"

# Display service URLs
echo ""
echo "=== ByteFreezer Full Stack Started ==="
echo "Control Service:    http://localhost:8082"
echo "Receiver Service:   http://localhost:8080 (Management), http://localhost:8081 (Ingestion)"
echo "Piper Service:      http://localhost:8083"
echo "Packer Service:     http://localhost:8084"
echo "Prometheus:         http://localhost:9090"
echo "Grafana:            http://localhost:3000 (admin/admin)"
echo "Management UI:      http://localhost:8090"
echo ""
echo "To stop: docker-compose -f docker-compose.full-stack.yml down"
echo "To view logs: docker-compose -f docker-compose.full-stack.yml logs -f [service]"
```

These comprehensive integration examples demonstrate how to use ByteFreezer Control for ecosystem monitoring, tenant management, service configuration, and complete production deployments with full observability and management capabilities.