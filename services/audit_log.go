package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/n0needt0/go-goodies/log"
)

type AuditLogService struct {
	db *sql.DB
}

func NewAuditLogService(db *sql.DB) *AuditLogService {
	return &AuditLogService{
		db: db,
	}
}

// sensitiveKeys contains field names that should be masked in audit logs
var sensitiveKeys = []string{
	"password",
	"secret",
	"token",
	"key",
	"credential",
	"auth",
	"api_key",
	"access_key",
	"secret_key",
	"private_key",
	"jwt",
	"bearer",
	"authorization",
}

// isSensitiveKey checks if a key name contains sensitive information
func isSensitiveKey(key string) bool {
	keyLower := strings.ToLower(key)
	for _, sensitive := range sensitiveKeys {
		if strings.Contains(keyLower, sensitive) {
			return true
		}
	}
	return false
}

// maskSensitiveData recursively masks sensitive information in the details map
func maskSensitiveData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	masked := make(map[string]interface{})
	for key, value := range data {
		if isSensitiveKey(key) {
			// Mask the entire value
			masked[key] = "***REDACTED***"
		} else if nestedMap, ok := value.(map[string]interface{}); ok {
			// Recursively mask nested maps
			masked[key] = maskSensitiveData(nestedMap)
		} else if nestedSlice, ok := value.([]interface{}); ok {
			// Handle slices
			maskedSlice := make([]interface{}, len(nestedSlice))
			for i, item := range nestedSlice {
				if itemMap, ok := item.(map[string]interface{}); ok {
					maskedSlice[i] = maskSensitiveData(itemMap)
				} else {
					maskedSlice[i] = item
				}
			}
			masked[key] = maskedSlice
		} else {
			// Keep non-sensitive values as-is
			masked[key] = value
		}
	}
	return masked
}

// LogAction logs a user action to the audit log
func (a *AuditLogService) LogAction(ctx context.Context, userID, userEmail, accountID, action, resourceType, resourceID, ipAddress string, details map[string]interface{}) {
	// Mask sensitive data before processing
	maskedDetails := maskSensitiveData(details)

	// Extract resource name from masked details
	resourceName := ""
	if maskedDetails != nil {
		if name, ok := maskedDetails["dataset_name"].(string); ok {
			resourceName = name
		} else if name, ok := maskedDetails["tenant_name"].(string); ok {
			resourceName = name
		} else if name, ok := maskedDetails["account_name"].(string); ok {
			resourceName = name
		} else if name, ok := maskedDetails["resource_name"].(string); ok {
			resourceName = name
		}
	}

	// Convert masked details to JSON
	detailsJSON, err := json.Marshal(maskedDetails)
	if err != nil {
		log.Warnf("Failed to marshal audit log details: %v", err)
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO control_audit_log (user_id, user_email, account_id, action, resource_type, resource_id, resource_name, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	userAgent := ""
	if maskedDetails != nil {
		if ua, ok := maskedDetails["user_agent"].(string); ok {
			userAgent = ua
		}
	}

	_, err = a.db.ExecContext(ctx, query, userID, userEmail, accountID, action, resourceType, resourceID, resourceName, detailsJSON, ipAddress, userAgent)
	if err != nil {
		log.Errorf("Failed to write audit log entry (user_id=%s, email=%s, ip=%s, action=%s): %v", userID, userEmail, ipAddress, action, err)
	}
}
