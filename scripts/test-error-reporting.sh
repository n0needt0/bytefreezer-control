#!/bin/bash

# ByteFreezer Error Reporting Test Script
# This script tests the error reporting system by sending test errors to the control service

set -e

# Configuration
CONTROL_URL="${CONTROL_URL:-http://192.168.86.103:8082}"
SYSTEM_API_KEY="${SYSTEM_API_KEY:-bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_header() {
    echo -e "${BLUE}===================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}===================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# Function to send error report
send_error() {
    local component=$1
    local error_type=$2
    local severity=$3
    local message=$4
    local tenant_id=${5:-""}
    local dataset_id=${6:-""}

    echo -e "\nSending ${severity} error from ${component}..."

    # Build JSON payload
    local payload=$(cat <<EOF
{
  "error_type": "${error_type}",
  "component": "${component}",
  "tenant_id": "${tenant_id}",
  "dataset_id": "${dataset_id}",
  "error_message": "${message}",
  "severity": "${severity}",
  "error_sample": {
    "test": true,
    "timestamp": "$(date -Iseconds)",
    "pid": $$
  },
  "metadata": {
    "test_script": "test-error-reporting.sh",
    "hostname": "$(hostname)"
  }
}
EOF
)

    # Send request
    response=$(curl -s -w "\n%{http_code}" -X POST \
        "${CONTROL_URL}/api/v1/errors" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${SYSTEM_API_KEY}" \
        -d "${payload}")

    http_code=$(echo "${response}" | tail -n1)
    body=$(echo "${response}" | head -n-1)

    if [ "${http_code}" = "200" ]; then
        print_success "Error reported successfully"
        echo "Response: ${body}"
        return 0
    else
        print_error "Failed to report error (HTTP ${http_code})"
        echo "Response: ${body}"
        return 1
    fi
}

# Function to get error stats
get_error_stats() {
    echo -e "\nFetching error statistics..."

    response=$(curl -s -w "\n%{http_code}" -X GET \
        "${CONTROL_URL}/api/v1/errors/stats" \
        -H "Authorization: Bearer ${SYSTEM_API_KEY}")

    http_code=$(echo "${response}" | tail -n1)
    body=$(echo "${response}" | head -n-1)

    if [ "${http_code}" = "200" ]; then
        print_success "Error statistics retrieved"
        echo "${body}" | python3 -m json.tool 2>/dev/null || echo "${body}"
        return 0
    else
        print_error "Failed to get error statistics (HTTP ${http_code})"
        echo "Response: ${body}"
        return 1
    fi
}

# Function to list errors
list_errors() {
    local limit=${1:-10}
    local severity=${2:-""}
    local component=${3:-""}

    echo -e "\nListing errors (limit=${limit})..."

    local params="limit=${limit}"
    [ -n "${severity}" ] && params="${params}&severity=${severity}"
    [ -n "${component}" ] && params="${params}&component=${component}"

    response=$(curl -s -w "\n%{http_code}" -X GET \
        "${CONTROL_URL}/api/v1/errors?${params}" \
        -H "Authorization: Bearer ${SYSTEM_API_KEY}")

    http_code=$(echo "${response}" | tail -n1)
    body=$(echo "${response}" | head -n-1)

    if [ "${http_code}" = "200" ]; then
        print_success "Errors listed successfully"
        echo "${body}" | python3 -m json.tool 2>/dev/null || echo "${body}"
        return 0
    else
        print_error "Failed to list errors (HTTP ${http_code})"
        echo "Response: ${body}"
        return 1
    fi
}

# Main test execution
main() {
    print_header "ByteFreezer Error Reporting Test"

    echo "Control URL: ${CONTROL_URL}"
    echo "Testing with system API key"
    echo ""

    # Test 1: Critical error from packer
    print_header "Test 1: Critical Error - Packer S3 Upload Failure"
    send_error "packer" "s3_upload_failure" "critical" \
        "Failed to upload Parquet file to S3" \
        "tenant-001" "dataset-001" || true
    sleep 1

    # Test 2: Error from receiver
    print_header "Test 2: Error - Receiver Storage Failure"
    send_error "receiver" "storage_failure" "error" \
        "Failed to write data to local storage" \
        "tenant-002" "dataset-002" || true
    sleep 1

    # Test 3: Warning from piper
    print_header "Test 3: Warning - Piper Processing Delay"
    send_error "piper" "processing_delay" "warning" \
        "Processing queue is growing beyond threshold" \
        "tenant-001" "dataset-003" || true
    sleep 1

    # Test 4: Error from proxy
    print_header "Test 4: Error - Proxy UDP Parse Failure"
    send_error "proxy" "udp_parse_failure" "error" \
        "Failed to parse UDP packet" \
        "tenant-003" "dataset-001" || true
    sleep 1

    # Test 5: Critical error from control
    print_header "Test 5: Critical Error - Control Database Connection Lost"
    send_error "control" "database_connection_lost" "critical" \
        "PostgreSQL connection failed after 3 retries" \
        "" "" || true
    sleep 1

    # Test 6: Multiple occurrences of same error (deduplication test)
    print_header "Test 6: Deduplication Test - Same Error Multiple Times"
    for i in {1..5}; do
        echo "Attempt ${i}/5..."
        send_error "packer" "json_parse_error" "warning" \
            "Failed to parse JSON line in source file" \
            "tenant-001" "dataset-001" || true
        sleep 0.5
    done

    # Test 7: Component-specific errors
    print_header "Test 7: Various Component Errors"
    send_error "receiver" "disk_space_low" "warning" \
        "Disk space below 10% threshold" "" "" || true
    sleep 0.5

    send_error "piper" "transformation_failure" "error" \
        "Failed to apply data transformation" \
        "tenant-002" "dataset-001" || true
    sleep 0.5

    send_error "packer" "parquet_conversion_failed" "error" \
        "Failed to convert NDJSON to Parquet format" \
        "tenant-001" "dataset-002" || true
    sleep 0.5

    # Get error statistics
    print_header "Error Statistics"
    get_error_stats || true

    # List recent errors
    print_header "Recent Errors (Last 5)"
    list_errors 5 || true

    # List only critical errors
    print_header "Critical Errors Only"
    list_errors 10 "critical" || true

    # List packer errors
    print_header "Packer Errors Only"
    list_errors 10 "" "packer" || true

    print_header "Test Complete"
    print_success "All test scenarios executed"
    echo ""
    echo "Next steps:"
    echo "1. Check the UI at ${CONTROL_URL}/dashboard/errors"
    echo "2. Verify deduplication is working (same errors should have occurrence_count > 1)"
    echo "3. Check adaptive sampling for high-frequency errors"
    echo ""
    echo "To verify in database:"
    echo "  psql -h 192.168.86.137 -U bytefreezer -d bytefreezer \\"
    echo "    -c 'SELECT component, error_type, severity, occurrence_count, sample_rate FROM system_errors ORDER BY last_seen DESC LIMIT 10;'"
}

# Run main function
main "$@"
