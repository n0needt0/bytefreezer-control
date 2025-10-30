#!/usr/bin/env python3
"""
ByteFreezer Error Reporting Test Script
Tests the error reporting system with various scenarios
"""

import json
import sys
import time
import requests
from datetime import datetime
from typing import Optional, Dict, Any

# Configuration
CONTROL_URL = "http://192.168.86.103:8082"
SYSTEM_API_KEY = "bytefreezer-service-api-key-8f4a2d1b-3c5e-4f6a-9b8c-7d2e1f3a4b5c"

# ANSI color codes
class Colors:
    RED = '\033[0;31m'
    GREEN = '\033[0;32m'
    YELLOW = '\033[1;33m'
    BLUE = '\033[0;34m'
    NC = '\033[0m'  # No Color

def print_header(message: str):
    """Print a formatted header"""
    print(f"\n{Colors.BLUE}{'=' * 60}{Colors.NC}")
    print(f"{Colors.BLUE}{message}{Colors.NC}")
    print(f"{Colors.BLUE}{'=' * 60}{Colors.NC}")

def print_success(message: str):
    """Print success message"""
    print(f"{Colors.GREEN}✓ {message}{Colors.NC}")

def print_error(message: str):
    """Print error message"""
    print(f"{Colors.RED}✗ {message}{Colors.NC}")

def print_warning(message: str):
    """Print warning message"""
    print(f"{Colors.YELLOW}⚠ {message}{Colors.NC}")

def send_error(
    component: str,
    error_type: str,
    severity: str,
    message: str,
    tenant_id: str = "",
    dataset_id: str = "",
    api_key: Optional[str] = None,
    account_id: Optional[str] = None
) -> bool:
    """
    Send an error report to the control service

    Args:
        component: Component name (proxy, receiver, piper, packer, control, soc)
        error_type: Error type identifier
        severity: Error severity (debug, info, warning, error, critical)
        message: Error message
        tenant_id: Optional tenant ID
        dataset_id: Optional dataset ID
        api_key: API key or JWT token (defaults to SYSTEM_API_KEY)
        account_id: Optional account ID (for account-scoped endpoints)

    Returns:
        True if successful, False otherwise
    """
    print(f"\nSending {severity} error from {component}...")

    if api_key is None:
        api_key = SYSTEM_API_KEY

    # Build payload
    payload = {
        "error_type": error_type,
        "component": component,
        "error_message": message,
        "severity": severity,
        "error_sample": {
            "test": True,
            "timestamp": datetime.now().isoformat(),
            "script": "test-error-reporting.py"
        },
        "metadata": {
            "test_script": "test-error-reporting.py",
            "test_run": datetime.now().isoformat()
        }
    }

    if tenant_id:
        payload["tenant_id"] = tenant_id
    if dataset_id:
        payload["dataset_id"] = dataset_id

    # Determine endpoint
    if account_id:
        url = f"{CONTROL_URL}/api/v1/accounts/{account_id}/errors"
    else:
        url = f"{CONTROL_URL}/api/v1/errors"

    # Send request
    try:
        response = requests.post(
            url,
            json=payload,
            headers={
                "Content-Type": "application/json",
                "Authorization": f"Bearer {api_key}"
            },
            timeout=10
        )

        if response.status_code == 200:
            print_success("Error reported successfully")
            print(f"Response: {response.json()}")
            return True
        else:
            print_error(f"Failed to report error (HTTP {response.status_code})")
            print(f"Response: {response.text}")
            return False

    except Exception as e:
        print_error(f"Exception occurred: {str(e)}")
        return False

def get_error_stats(api_key: Optional[str] = None, account_id: Optional[str] = None) -> Optional[Dict[str, Any]]:
    """Get error statistics"""
    print("\nFetching error statistics...")

    if api_key is None:
        api_key = SYSTEM_API_KEY

    # Determine endpoint
    if account_id:
        url = f"{CONTROL_URL}/api/v1/accounts/{account_id}/errors/stats"
    else:
        url = f"{CONTROL_URL}/api/v1/errors/stats"

    try:
        response = requests.get(
            url,
            headers={"Authorization": f"Bearer {api_key}"},
            timeout=10
        )

        if response.status_code == 200:
            print_success("Error statistics retrieved")
            stats = response.json()
            print(json.dumps(stats, indent=2))
            return stats
        else:
            print_error(f"Failed to get error statistics (HTTP {response.status_code})")
            print(f"Response: {response.text}")
            return None

    except Exception as e:
        print_error(f"Exception occurred: {str(e)}")
        return None

def list_errors(
    limit: int = 10,
    severity: str = "",
    component: str = "",
    api_key: Optional[str] = None,
    account_id: Optional[str] = None
) -> Optional[Dict[str, Any]]:
    """List errors with optional filters"""
    print(f"\nListing errors (limit={limit})...")

    if api_key is None:
        api_key = SYSTEM_API_KEY

    # Build query parameters
    params = {"limit": limit}
    if severity:
        params["severity"] = severity
    if component:
        params["component"] = component

    # Determine endpoint
    if account_id:
        url = f"{CONTROL_URL}/api/v1/accounts/{account_id}/errors"
    else:
        url = f"{CONTROL_URL}/api/v1/errors"

    try:
        response = requests.get(
            url,
            params=params,
            headers={"Authorization": f"Bearer {api_key}"},
            timeout=10
        )

        if response.status_code == 200:
            print_success("Errors listed successfully")
            errors_data = response.json()
            print(f"Total errors: {errors_data.get('total', 0)}")

            for error in errors_data.get('errors', [])[:5]:  # Show first 5
                print(f"  - [{error['severity'].upper()}] {error['component']}: {error['error_type']}")
                print(f"    {error['error_message']}")
                print(f"    Occurrences: {error['occurrence_count']}, Last seen: {error['last_seen']}")

            return errors_data
        else:
            print_error(f"Failed to list errors (HTTP {response.status_code})")
            print(f"Response: {response.text}")
            return None

    except Exception as e:
        print_error(f"Exception occurred: {str(e)}")
        return None

def test_deduplication():
    """Test error deduplication by sending the same error multiple times"""
    print_header("Deduplication Test - Same Error Multiple Times")

    for i in range(1, 6):
        print(f"Attempt {i}/5...")
        send_error(
            "packer",
            "json_parse_error",
            "warning",
            "Failed to parse JSON line in source file",
            "tenant-001",
            "dataset-001"
        )
        time.sleep(0.5)

    print_success("Deduplication test completed - check occurrence_count in database")

def test_adaptive_sampling():
    """Test adaptive sampling by generating many errors"""
    print_header("Adaptive Sampling Test - High Frequency Errors")
    print_warning("Generating 50 identical errors to test adaptive sampling...")

    for i in range(1, 51):
        if i % 10 == 0:
            print(f"Progress: {i}/50 errors sent...")

        send_error(
            "receiver",
            "high_frequency_error",
            "warning",
            "This error occurs very frequently for testing sampling",
            "tenant-999",
            "dataset-999"
        )
        time.sleep(0.1)

    print_success("Adaptive sampling test completed")
    print("Expected behavior:")
    print("  - 0-99 occurrences: 100% sampling (sample_rate = 1.0)")
    print("  - 100-999 occurrences: 10% sampling (sample_rate = 0.1)")
    print("  - Check samples_dropped to see sampling in action")

def run_basic_tests():
    """Run basic error reporting tests"""
    print_header("Basic Error Reporting Tests")

    # Test 1: Critical error from packer
    print_header("Test 1: Critical Error - Packer S3 Upload Failure")
    send_error(
        "packer",
        "s3_upload_failure",
        "critical",
        "Failed to upload Parquet file to S3 destination bucket",
        "tenant-001",
        "dataset-001"
    )
    time.sleep(1)

    # Test 2: Error from receiver
    print_header("Test 2: Error - Receiver Storage Failure")
    send_error(
        "receiver",
        "storage_failure",
        "error",
        "Failed to write data to local storage - disk full",
        "tenant-002",
        "dataset-002"
    )
    time.sleep(1)

    # Test 3: Warning from piper
    print_header("Test 3: Warning - Piper Processing Delay")
    send_error(
        "piper",
        "processing_delay",
        "warning",
        "Processing queue is growing beyond threshold (1000+ items)",
        "tenant-001",
        "dataset-003"
    )
    time.sleep(1)

    # Test 4: Error from proxy
    print_header("Test 4: Error - Proxy UDP Parse Failure")
    send_error(
        "proxy",
        "udp_parse_failure",
        "error",
        "Failed to parse UDP packet - invalid format",
        "tenant-003",
        "dataset-001"
    )
    time.sleep(1)

    # Test 5: Critical error from control
    print_header("Test 5: Critical Error - Control Database Connection Lost")
    send_error(
        "control",
        "database_connection_lost",
        "critical",
        "PostgreSQL connection failed after 3 retries",
        "",
        ""
    )
    time.sleep(1)

    # Test 6: Various component errors
    print_header("Test 6: Various Component Errors")

    send_error("receiver", "disk_space_low", "warning", "Disk space below 10% threshold", "", "")
    time.sleep(0.5)

    send_error("piper", "transformation_failure", "error", "Failed to apply data transformation", "tenant-002", "dataset-001")
    time.sleep(0.5)

    send_error("packer", "parquet_conversion_failed", "error", "Failed to convert NDJSON to Parquet format", "tenant-001", "dataset-002")
    time.sleep(0.5)

def main():
    """Main test execution"""
    print_header("ByteFreezer Error Reporting Test")
    print(f"Control URL: {CONTROL_URL}")
    print("Testing with system API key")

    try:
        # Run basic tests
        run_basic_tests()

        # Test deduplication
        test_deduplication()

        # Get error statistics
        print_header("Error Statistics")
        get_error_stats()

        # List recent errors
        print_header("Recent Errors (Last 10)")
        list_errors(10)

        # List only critical errors
        print_header("Critical Errors Only")
        list_errors(10, severity="critical")

        # List packer errors
        print_header("Packer Errors Only")
        list_errors(10, component="packer")

        # Ask if user wants to test adaptive sampling
        print_header("Additional Tests")
        print("Would you like to test adaptive sampling? (generates 50+ errors)")
        response = input("Type 'yes' to continue: ")
        if response.lower() in ['yes', 'y']:
            test_adaptive_sampling()

        print_header("Test Complete")
        print_success("All test scenarios executed successfully")
        print("\nNext steps:")
        print(f"1. Check the UI at {CONTROL_URL}/dashboard/errors")
        print("2. Verify deduplication is working (same errors should have occurrence_count > 1)")
        print("3. Check adaptive sampling for high-frequency errors")
        print("\nTo verify in database:")
        print("  psql -h 192.168.86.137 -U bytefreezer -d bytefreezer \\")
        print("    -c 'SELECT component, error_type, severity, occurrence_count, sample_rate FROM system_errors ORDER BY last_seen DESC LIMIT 10;'")

    except KeyboardInterrupt:
        print("\n\nTest interrupted by user")
        sys.exit(1)
    except Exception as e:
        print_error(f"Test failed with exception: {str(e)}")
        sys.exit(1)

if __name__ == "__main__":
    main()
