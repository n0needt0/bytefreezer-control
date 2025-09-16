package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/go-goodies/log"
)

type contextKey string

const UserContextKey = contextKey("user")

type UserClaims struct {
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	jwt.RegisteredClaims
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
				if isAdminEndpoint(r.URL.Path) && !claims.IsAdmin {
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

// GenerateToken generates a JWT token for a user
func GenerateToken(username string, isAdmin bool, authConfig config.AuthConfig) (string, error) {
	claims := UserClaims{
		Username: username,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(authConfig.TokenExpiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(authConfig.JWTSecret))
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
