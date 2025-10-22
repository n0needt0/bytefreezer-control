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
				ctx := context.WithValue(r.Context(), JWTClaimsContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}
		})
	}
}

// ConditionalJWTAuthMiddleware provides JWT authentication middleware that skips public endpoints
func ConditionalJWTAuthMiddleware(authConfig config.AuthConfig, authService *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for public endpoints
			if isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Apply authentication for all other endpoints
			authMiddleware := JWTAuthMiddleware(authConfig, authService)
			authMiddleware(next).ServeHTTP(w, r)
		})
	}
}
