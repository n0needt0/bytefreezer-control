package services

import (
	"context"
	"database/sql"
	"encoding/json"

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

// LogAction logs a user action to the audit log
func (a *AuditLogService) LogAction(ctx context.Context, userID, accountID, action, resourceType, resourceID string, details map[string]interface{}) {
	// Convert details to JSON
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Warnf("Failed to marshal audit log details: %v", err)
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO control_audit_log (user_id, account_id, action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	ipAddress := ""
	userAgent := ""
	if details != nil {
		if ip, ok := details["ip_address"].(string); ok {
			ipAddress = ip
		}
		if ua, ok := details["user_agent"].(string); ok {
			userAgent = ua
		}
	}

	_, err = a.db.ExecContext(ctx, query, userID, accountID, action, resourceType, resourceID, detailsJSON, ipAddress, userAgent)
	if err != nil {
		log.Errorf("Failed to write audit log entry: %v", err)
	}
}
