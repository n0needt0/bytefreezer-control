# Error Tracking Security Guidelines

## Access Control

### Account-Based Filtering
- **Regular Users**: Can only see errors from their own account_id
- **System Administrators**: Can see all errors across all accounts
- Errors without an account_id are only visible to system administrators

### Implementation
When components report errors via `/api/v2/errors/track`, they should include:
- `account_id` - For account-scoped errors
- `tenant_id` - For tenant-scoped errors (optional)
- `dataset_id` - For dataset-scoped errors (optional)

The API automatically filters errors based on the authenticated user's permissions.

## Sensitive Data Protection

### DO NOT Include Sensitive Data in Error Messages

Components **MUST NOT** include the following types of sensitive data in error messages, samples, or metadata:

#### Authentication & Authorization
- Passwords or password hashes
- API keys, tokens, or secrets
- Session IDs or cookies
- OAuth tokens or refresh tokens
- JWT tokens (full tokens - token prefixes are OK)
- Private keys or certificates

#### Personal Identifiable Information (PII)
- Email addresses (use user IDs instead)
- Phone numbers
- Physical addresses
- Social security numbers
- Credit card numbers or financial data
- Health information
- IP addresses (unless necessary for debugging network issues)

#### Business Sensitive Data
- Database connection strings with credentials
- S3 bucket URLs with embedded credentials
- Internal system paths that reveal infrastructure
- Proprietary algorithms or business logic details

### Safe Error Message Examples

#### ❌ Bad (Contains Sensitive Data)
```json
{
  "error_message": "Failed to authenticate user@example.com with password 'secret123'",
  "error_sample": {
    "user_email": "john.doe@company.com",
    "api_key": "sk_live_1234567890abcdef",
    "connection_string": "postgresql://user:password@host:5432/db"
  }
}
```

#### ✅ Good (No Sensitive Data)
```json
{
  "error_message": "Failed to authenticate user (ID: usr_abc123)",
  "error_sample": {
    "user_id": "usr_abc123",
    "error_code": "AUTH_FAILED",
    "attempt_count": 3,
    "connection_type": "postgresql"
  }
}
```

### Best Practices

1. **Use IDs Instead of Values**
   - User IDs instead of emails
   - Tenant IDs instead of tenant names
   - Resource IDs instead of resource details

2. **Sanitize Stack Traces**
   - Remove file paths that reveal infrastructure
   - Redact environment variable values
   - Remove database query parameters

3. **Use Error Codes**
   - Define error codes for common failures
   - Include error codes in messages instead of detailed reasons
   - Document error codes separately in secure documentation

4. **Implement Redaction**
   - Use helper functions to automatically redact sensitive patterns
   - Redact query parameters from URLs
   - Mask partial values (e.g., "sk_live_***def" for API keys)

5. **Limit Sample Data**
   - Only include minimal context needed for debugging
   - Use hashes or checksums instead of actual data
   - Truncate large values

### Example Redaction Helper

```go
// Example Go function for redacting sensitive data
func RedactSensitiveData(message string) string {
    // Redact passwords
    message = regexp.MustCompile(`password[=:]\s*['"]?([^'"\s]+)['"]?`).
        ReplaceAllString(message, "password=***")

    // Redact API keys
    message = regexp.MustCompile(`(sk_|pk_|api_key[=:])[a-zA-Z0-9]+`).
        ReplaceAllString(message, "$1***")

    // Redact connection strings
    message = regexp.MustCompile(`://[^:]+:[^@]+@`).
        ReplaceAllString(message, "://***:***@")

    return message
}
```

## Compliance

Following these guidelines helps maintain compliance with:
- GDPR (General Data Protection Regulation)
- CCPA (California Consumer Privacy Act)
- SOC 2 Type II requirements
- HIPAA (if handling health data)

## Reporting Violations

If you discover sensitive data in error logs:
1. Immediately notify the security team
2. Document the error ID and type
3. Update the component to prevent future occurrences
4. Request data purging if necessary
