package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
)

// contextKey is used for storing values in context
type jwtContextKey string

const (
	// JWTClaimsContextKey is the key for storing JWT claims in context
	JWTClaimsContextKey = jwtContextKey("jwt_claims")
)

// GetJWTClaims extracts JWT claims from context
// Returns nil if no claims found (e.g., auth disabled or public endpoint)
func GetJWTClaims(ctx context.Context) *services.JWTClaims {
	claims, ok := ctx.Value(JWTClaimsContextKey).(*services.JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// GetAccountIDFromContext extracts account_id from JWT claims in context
// Returns empty string if no claims or user is system admin (account_id is NULL)
func GetAccountIDFromContext(ctx context.Context) string {
	claims := GetJWTClaims(ctx)
	if claims == nil {
		return ""
	}
	return claims.AccountID
}

// IsSystemAdmin checks if the user is a system admin based on JWT claims
func IsSystemAdmin(ctx context.Context) bool {
	claims := GetJWTClaims(ctx)
	if claims == nil {
		return false
	}
	return claims.Role == "system_admin"
}

// JWTAuthMiddleware provides JWT authentication middleware that stores claims in context
func JWTAuthMiddleware(authConfig config.AuthConfig, authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check Bearer token format
			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := bearerToken[1]

			// Parse and validate token using auth service JWT secret
			token, err := jwt.ParseWithClaims(tokenString, &services.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return authService.GetJWTSecret(), nil
			})

			if err != nil {
				log.Warnf("Invalid token: %v", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(*services.JWTClaims); ok && token.Valid {
				// Add JWT claims to context for audit logging
				log.Debugf("[JWT MIDDLEWARE] Successfully parsed JWT claims for user: AccountID=%s, Role=%s, UserID=%s",
					claims.AccountID, claims.Role, claims.UserID)
				ctx := context.WithValue(r.Context(), JWTClaimsContextKey, claims)
				log.Debugf("[JWT MIDDLEWARE] Added claims to context with key=%v", JWTClaimsContextKey)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				log.Warnf("[JWT MIDDLEWARE] Invalid token claims or invalid token")
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}

// ConditionalJWTAuthMiddleware provides authentication middleware that:
// 1. Skips public endpoints
// 2. Validates system-wide API token (for piper/packer/receiver)
// 3. Validates account-specific API keys (for proxy)
// 4. Validates JWT tokens (for users)
func ConditionalJWTAuthMiddleware(authConfig config.AuthConfig, authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for public endpoints
			if isPublicEndpoint(r.URL.Path) {
				log.Debugf("[JWT MIDDLEWARE] Skipping auth for public endpoint: %s", r.URL.Path)
				next.ServeHTTP(w, r)
				return
			}

			// Extract Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// Check Bearer token format
			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := bearerToken[1]

			// Try system-wide API token first (for internal services: piper/packer/receiver)
			if authConfig.ServiceAPIKey != "" && tokenString == authConfig.ServiceAPIKey {
				log.Debugf("[JWT MIDDLEWARE] Valid system API token for endpoint: %s", r.URL.Path)
				claims := &services.JWTClaims{
					UserID:    "system",
					AccountID: "", // System account sees all data
					Email:     "system@bytefreezer.internal",
					Role:      "system_admin",
				}
				ctx := context.WithValue(r.Context(), JWTClaimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Try account-specific API key (for proxy)
			accountID, err := authService.ValidateAPIKey(r.Context(), tokenString)
			if err == nil {
				log.Debugf("[JWT MIDDLEWARE] Valid account API key for account %s, endpoint: %s", accountID, r.URL.Path)
				// Create claims with account context (filtered by account_id)
				claims := &services.JWTClaims{
					UserID:    "api-key",
					AccountID: accountID,
					Email:     fmt.Sprintf("api-key@%s.bytefreezer.internal", accountID),
					Role:      "account_admin", // API keys have account_admin privileges
				}
				ctx := context.WithValue(r.Context(), JWTClaimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			log.Debugf("[JWT MIDDLEWARE] Applying JWT auth for endpoint: %s", r.URL.Path)
			// Otherwise, apply JWT authentication
			authMiddleware := JWTAuthMiddleware(authConfig, authService)
			authMiddleware(next).ServeHTTP(w, r)
		})
	}
}
