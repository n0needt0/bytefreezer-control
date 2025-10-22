package api

import (
	"context"

	"github.com/n0needt0/bytefreezer-control/middleware"
)

// AuditInfo contains extracted information for audit logging
type AuditInfo struct {
	UserID    string
	UserEmail string
	IPAddress string
}

// ExtractAuditInfo extracts user ID, email, and IP address from request context
// This ensures audit logs capture the information at the time of the event
// The middleware.AuditMiddleware must be applied for this to work
func ExtractAuditInfo(ctx context.Context) AuditInfo {
	// Try to extract audit info from context (set by middleware.AuditMiddleware)
	if auditInfo, ok := ctx.Value(middleware.AuditInfoContextKey).(*middleware.AuditInfo); ok && auditInfo != nil {
		return AuditInfo{
			UserID:    auditInfo.UserID,
			UserEmail: auditInfo.UserEmail,
			IPAddress: auditInfo.IPAddress,
		}
	}

	// Fallback if middleware didn't set audit info (shouldn't happen in production)
	return AuditInfo{
		UserID:    "system",
		UserEmail: "system@bytefreezer.local",
		IPAddress: "unknown",
	}
}
