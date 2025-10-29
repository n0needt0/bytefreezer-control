# Error Access Control Testing Guide

This guide provides commands to test the error access control functionality that filters errors based on user role and account.

## Expected Behavior

- **System Admin (role="system_admin")**: Should see ALL errors from all accounts
- **Account Admin (role="account_admin" or other roles)**: Should ONLY see errors where `account_id` matches their JWT token's `account_id`

## Prerequisites

You need valid user credentials for:
1. A system_admin user
2. An account_admin user (or any non-system-admin user)

## Test Commands

### Method 1: Individual Commands

#### Test 1: System Admin Access

```bash
# Login as system_admin
curl -X POST http://192.168.86.103:8082/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"your_password"}' \
  | jq -r '.token'

# Save the token from above, then query errors
# Replace YOUR_SYSTEM_ADMIN_TOKEN with the actual token
curl -X GET "http://192.168.86.103:8082/api/v1/errors?status=active&limit=10" \
  -H "Authorization: Bearer YOUR_SYSTEM_ADMIN_TOKEN" \
  | jq '{count: .count, total: .total, sample_errors: .errors[0:3] | map({id, account_id, component, error_type})}'
```

**Expected Result**: Should see errors from ALL accounts

#### Test 2: Account Admin Access

```bash
# Login as account_admin
curl -X POST http://192.168.86.103:8082/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"email":"account_admin@example.com","password":"your_password"}' \
  | jq -r '.token'

# Query errors with account_admin token
# Replace YOUR_ACCOUNT_ADMIN_TOKEN with the actual token
curl -X GET "http://192.168.86.103:8082/api/v1/errors?status=active&limit=10" \
  -H "Authorization: Bearer YOUR_ACCOUNT_ADMIN_TOKEN" \
  | jq '{count: .count, total: .total, sample_errors: .errors[0:3] | map({id, account_id, component, error_type})}'
```

**Expected Result**: Should ONLY see errors where `account_id` matches the user's account

### Method 2: Automated Test Script

Save this as `test_error_access_control.sh`:

```bash
#!/bin/bash

# Configuration - UPDATE THESE WITH YOUR ACTUAL CREDENTIALS
SYSTEM_ADMIN_EMAIL="admin@example.com"
SYSTEM_ADMIN_PASSWORD="your_password"
ACCOUNT_ADMIN_EMAIL="account_admin@example.com"
ACCOUNT_ADMIN_PASSWORD="your_password"
API_BASE="http://192.168.86.103:8082"

echo "=================================================="
echo "Error Access Control Testing"
echo "=================================================="
echo ""

# Test with system_admin
echo "=== Test 1: System Admin Access ==="
echo "Logging in as system_admin..."
SYSTEM_TOKEN=$(curl -s -X POST "${API_BASE}/api/v1/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${SYSTEM_ADMIN_EMAIL}\",\"password\":\"${SYSTEM_ADMIN_PASSWORD}\"}" \
  | jq -r '.token')

if [ "$SYSTEM_TOKEN" = "null" ] || [ -z "$SYSTEM_TOKEN" ]; then
  echo "ERROR: Failed to login as system_admin"
  exit 1
fi

echo "System admin token: ${SYSTEM_TOKEN:0:20}..."
echo ""
echo "Fetching errors..."

curl -s -X GET "${API_BASE}/api/v1/errors?status=active&limit=5" \
  -H "Authorization: Bearer $SYSTEM_TOKEN" \
  | jq '{
      count: .count,
      total: .total,
      sample_errors: .errors[0:3] | map({
        id,
        account_id,
        component,
        error_type
      })
    }'

echo ""
echo "=================================================="
echo ""

# Test with account_admin
echo "=== Test 2: Account Admin Access ==="
echo "Logging in as account_admin..."
ACCOUNT_TOKEN=$(curl -s -X POST "${API_BASE}/api/v1/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${ACCOUNT_ADMIN_EMAIL}\",\"password\":\"${ACCOUNT_ADMIN_PASSWORD}\"}" \
  | jq -r '.token')

if [ "$ACCOUNT_TOKEN" = "null" ] || [ -z "$ACCOUNT_TOKEN" ]; then
  echo "ERROR: Failed to login as account_admin"
  exit 1
fi

echo "Account admin token: ${ACCOUNT_TOKEN:0:20}..."
echo ""
echo "Fetching errors..."

curl -s -X GET "${API_BASE}/api/v1/errors?status=active&limit=5" \
  -H "Authorization: Bearer $ACCOUNT_TOKEN" \
  | jq '{
      count: .count,
      total: .total,
      sample_errors: .errors[0:3] | map({
        id,
        account_id,
        component,
        error_type
      })
    }'

echo ""
echo "=================================================="
echo ""

# Check middleware logs
echo "=== Middleware Debug Logs ==="
echo "Checking logs on .103..."
ssh andrew@192.168.86.103 "tail -100 /tmp/control-debug.log | grep -E 'JWT MIDDLEWARE|ACCESS CONTROL' | tail -20"

echo ""
echo "=================================================="
echo "Testing Complete"
echo "=================================================="
```

#### Run the script:

```bash
chmod +x test_error_access_control.sh
./test_error_access_control.sh
```

## Checking Debug Logs

The service logs detailed information about JWT authentication and access control:

```bash
# View recent logs with JWT and access control information
ssh andrew@192.168.86.103 "tail -100 /tmp/control-debug.log | grep -E 'JWT MIDDLEWARE|ACCESS CONTROL'"

# Follow logs in real-time
ssh andrew@192.168.86.103 "tail -f /tmp/control-debug.log | grep -E 'JWT MIDDLEWARE|ACCESS CONTROL'"
```

### Expected Log Output

For a successful authenticated request, you should see:

```
[JWT MIDDLEWARE] Applying auth for endpoint: /api/v1/errors
[JWT MIDDLEWARE] Successfully parsed JWT claims for user: AccountID=acc_123, Role=account_admin, UserID=user_456
[JWT MIDDLEWARE] Added claims to context with key=jwt_claims
[ACCESS CONTROL] Attempting to extract JWT claims from context
[ACCESS CONTROL] Successfully extracted claims: AccountID=acc_123, Role=account_admin, IsSystemAdmin=false
[ACCESS CONTROL] Applying account filter: account_id = acc_123
```

For a system admin:

```
[JWT MIDDLEWARE] Applying auth for endpoint: /api/v1/errors
[JWT MIDDLEWARE] Successfully parsed JWT claims for user: AccountID=system, Role=system_admin, UserID=admin_1
[JWT MIDDLEWARE] Added claims to context with key=jwt_claims
[ACCESS CONTROL] Attempting to extract JWT claims from context
[ACCESS CONTROL] Successfully extracted claims: AccountID=system, Role=system_admin, IsSystemAdmin=true
[ACCESS CONTROL] No account filter applied (isSystemAdmin=true, userAccountID=system)
```

## Verifying Database Data

To verify the actual errors in the database and their account associations:

```bash
# Check errors grouped by account_id
ssh andrew@192.168.86.137 "PGPASSWORD=bytefreezer123 psql -h localhost -U bytefreezer -d bytefreezer -c \"
  SELECT
    account_id,
    component,
    COUNT(*) as error_count,
    COUNT(DISTINCT error_type) as unique_types
  FROM system_errors
  WHERE status = 'active'
  GROUP BY account_id, component
  ORDER BY account_id, component;
\""

# View sample errors
ssh andrew@192.168.86.137 "PGPASSWORD=bytefreezer123 psql -h localhost -U bytefreezer -d bytefreezer -c \"
  SELECT
    id,
    account_id,
    component,
    error_type,
    severity,
    occurrence_count
  FROM system_errors
  WHERE status = 'active'
  ORDER BY id
  LIMIT 20;
\""
```

## Troubleshooting

### Issue: Getting all errors even as account_admin

**Symptoms**: Account admin sees errors from all accounts instead of just their own

**Debug steps**:
1. Check the logs to see if JWT claims are being extracted correctly
2. Verify the user's JWT token contains the correct `account_id` and `role`
3. Check if the database errors actually have `account_id` set

```bash
# Decode JWT token to inspect claims (requires jq and base64)
echo "YOUR_TOKEN" | cut -d. -f2 | base64 -d 2>/dev/null | jq
```

### Issue: "unauthorized: invalid or missing authentication token"

**Symptoms**: Request is rejected with 401 error

**Possible causes**:
- JWT token is expired
- JWT token is malformed
- Middleware cannot extract claims from context

**Debug**: Check logs for authentication errors

### Issue: No logs appearing

**Possible causes**:
- Service not running on .103
- Logging to different file
- Log level set too high (not showing DEBUG logs)

**Check service status**:
```bash
ssh andrew@192.168.86.103 "ps aux | grep bytefreezer-control | grep -v grep"
```

## Success Criteria

The access control is working correctly when:

1. ✅ System admin can see ALL errors (count matches total errors in database)
2. ✅ Account admin can ONLY see errors with matching `account_id` (filtered count)
3. ✅ Logs show JWT claims being extracted with correct AccountID and Role
4. ✅ Logs show access control filter being applied for non-system-admins
5. ✅ No "Failed to extract JWT claims" errors in logs for authenticated requests
