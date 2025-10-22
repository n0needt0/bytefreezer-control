package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/n0needt0/bytefreezer-control/services"
)

// auditContextKey is used for storing audit info in context
type auditContextKey string

const (
	// AuditInfoContextKey is the key for storing audit information in context
	AuditInfoContextKey = auditContextKey("audit_info")
)

// AuditInfo contains extracted information for audit logging
type AuditInfo struct {
	UserID    string
	UserEmail string
	IPAddress string
}

// AuditMiddleware extracts user ID, email from JWT claims and IP address from request
// and stores them in context for audit logging
func AuditMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auditInfo := AuditInfo{
				UserID:    "system", // Default fallback
				UserEmail: "system@bytefreezer.local",
				IPAddress: extractIPAddress(r),
			}

			// Try to extract JWT claims from context (set by auth middleware)
			if claims, ok := r.Context().Value(JWTClaimsContextKey).(*services.JWTClaims); ok && claims != nil {
				if claims.UserID != "" {
					auditInfo.UserID = claims.UserID
				}
				if claims.Email != "" {
					auditInfo.UserEmail = claims.Email
				}
			}

			// Store audit info in context
			ctx := context.WithValue(r.Context(), AuditInfoContextKey, &auditInfo)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractIPAddress extracts the client IP address from the request
// Checks X-Forwarded-For and X-Real-IP headers first, then falls back to RemoteAddr
func extractIPAddress(r *http.Request) string {
	// Check X-Forwarded-For header (comma-separated list, first is original client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr (may include port)
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Return as-is if not in host:port format
	}
	return ip
}
