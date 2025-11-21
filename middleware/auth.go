package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/go-goodies/log"
)

type contextKey string

const UserContextKey = contextKey("user")

type UserClaims struct {
	UserID    string `json:"user_id"`
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// IsSystemAdmin returns true if the user has system_admin role
func (u *UserClaims) IsSystemAdmin() bool {
	return u.Role == "system_admin"
}

// AuthMiddleware provides JWT authentication middleware
func AuthMiddleware(authConfig config.AuthConfig) func(http.Handler) http.Handler {
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

			// Parse and validate token
			token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(authConfig.JWTSecret), nil
			})

			if err != nil {
				log.Warnf("Invalid token: %v", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
				// Check if user is admin for admin-only endpoints
				if isAdminEndpoint(r.URL.Path) && !claims.IsSystemAdmin() {
					http.Error(w, "Admin access required", http.StatusForbidden)
					return
				}

				// Add user to context
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}


// ConditionalAuthMiddleware provides authentication middleware that:
// 1. Skips public endpoints
// 2. Accepts API tokens for service-to-service communication
// 3. Accepts JWT tokens for user requests
func ConditionalAuthMiddleware(authConfig config.AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for public endpoints
			if isPublicEndpoint(r.URL.Path) {
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

			// Try API token first (for service-to-service communication)
			if authConfig.ServiceAPIKey != "" && tokenString == authConfig.ServiceAPIKey {
				// Valid service API token - create system admin context
				claims := &UserClaims{
					UserID:    "system",
					AccountID: "", // System account sees all data
					Email:     "system@bytefreezer.internal",
					Role:      "system_admin",
				}
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Otherwise, try JWT token (for user requests)
			token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(authConfig.JWTSecret), nil
			})

			if err != nil {
				log.Warnf("Invalid token: %v", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
				// Check if user is admin for admin-only endpoints
				if isAdminEndpoint(r.URL.Path) && !claims.IsSystemAdmin() {
					http.Error(w, "Admin access required", http.StatusForbidden)
					return
				}

				// Add user to context
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}

// isPublicEndpoint checks if the endpoint should skip authentication
func isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/api/v1",
		"/api/v1/health",
		"/api/v1/login",
		"/api/v1/password-reset",
		"/api/v1/docs",
		"/api/v1/docs/ai-agent",
		"/api/v1/docs/ai-agent/quick-reference",
	}

	// Check exact matches
	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}

	return false
}

// isAdminEndpoint checks if the endpoint requires admin privileges
func isAdminEndpoint(path string) bool {
	adminEndpoints := []string{
		"/api/v1/ecosystem/services/",
		"/api/v1/tenants",
	}

	for _, endpoint := range adminEndpoints {
		if strings.HasPrefix(path, endpoint) {
			return true
		}
	}

	return false
}

// CORSMiddleware provides CORS support for the API
func CORSMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow requests from UI development server and production
			allowedOrigins := []string{
				"http://localhost:3000",
				"http://localhost:3001",
				"http://localhost:3002",
				"http://localhost:3003",
				"https://dashboard.bytefreezer.org",
			}

			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
