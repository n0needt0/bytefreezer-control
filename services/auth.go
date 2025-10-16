package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/n0needt0/go-goodies/log"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	db        *sql.DB
	jwtSecret []byte
}

type User struct {
	ID             string                 `json:"id"`
	AccountID      string                 `json:"account_id"`
	Email          string                 `json:"email"`
	Role           string                 `json:"role"` // system_admin, account_admin, account_readonly
	FirstName      string                 `json:"first_name"`
	LastName       string                 `json:"last_name"`
	Active         bool                   `json:"active"`
	EmailVerified  bool                   `json:"email_verified"`
	OAuthProvider  *string                `json:"oauth_provider,omitempty"`
	LastLoginAt    *time.Time             `json:"last_login_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

type Session struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	TokenHash         string    `json:"-"`
	RefreshTokenHash  string    `json:"-"`
	ExpiresAt         time.Time `json:"expires_at"`
	RefreshExpiresAt  time.Time `json:"refresh_expires_at"`
	IPAddress         string    `json:"ip_address"`
	UserAgent         string    `json:"user_agent"`
	CreatedAt         time.Time `json:"created_at"`
	LastAccessedAt    time.Time `json:"last_accessed_at"`
}

type JWTClaims struct {
	UserID    string `json:"user_id"`
	AccountID string `json:"account_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	User         User      `json:"user"`
}

func NewAuthService(db *sql.DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:        db,
		jwtSecret: []byte(jwtSecret),
	}
}

// AuthenticateUser verifies email and password, returns user if valid
func (a *AuthService) AuthenticateUser(ctx context.Context, email, password string) (*User, error) {
	var user User
	var passwordHash string
	var firstName, lastName sql.NullString
	var lastLoginAt sql.NullTime
	var oauthProvider sql.NullString

	query := `
		SELECT id, account_id, email, password_hash, role, first_name, last_name,
		       active, email_verified, oauth_provider, last_login_at, created_at, updated_at
		FROM control_users
		WHERE email = $1 AND active = true
	`

	err := a.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.AccountID, &user.Email, &passwordHash, &user.Role,
		&firstName, &lastName, &user.Active, &user.EmailVerified,
		&oauthProvider, &lastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid email or password")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Check if user has password_hash (not OAuth-only user)
	if passwordHash == "" {
		return nil, fmt.Errorf("password authentication not available for this user")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Populate optional fields
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if oauthProvider.Valid {
		user.OAuthProvider = &oauthProvider.String
	}

	// Update last login timestamp
	_, err = a.db.ExecContext(ctx,
		"UPDATE control_users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1",
		user.ID)
	if err != nil {
		log.Warnf("Failed to update last_login_at for user %s: %v", user.ID, err)
	}

	return &user, nil
}

// GenerateTokenPair creates access and refresh tokens for a user
func (a *AuthService) GenerateTokenPair(user *User) (string, string, time.Time, error) {
	// Access token: 1 hour expiry
	expiresAt := time.Now().Add(1 * time.Hour)
	claims := JWTClaims{
		UserID:    user.ID,
		AccountID: user.AccountID,
		Email:     user.Email,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bytefreezer-control",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Refresh token: 7 days expiry
	refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	refreshClaims := JWTClaims{
		UserID:    user.ID,
		AccountID: user.AccountID,
		Email:     user.Email,
		Role:      user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(refreshExpiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "bytefreezer-control-refresh",
		},
	}

	refreshTokenJWT := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshToken, err := refreshTokenJWT.SignedString(a.jwtSecret)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return accessToken, refreshToken, expiresAt, nil
}

// CreateSession stores session information in database
func (a *AuthService) CreateSession(ctx context.Context, userID, accessToken, refreshToken, ipAddress, userAgent string, expiresAt, refreshExpiresAt time.Time) error {
	// Hash tokens before storing
	accessTokenHash, err := hashToken(accessToken)
	if err != nil {
		return fmt.Errorf("failed to hash access token: %w", err)
	}

	refreshTokenHash, err := hashToken(refreshToken)
	if err != nil {
		return fmt.Errorf("failed to hash refresh token: %w", err)
	}

	query := `
		INSERT INTO control_sessions (user_id, token_hash, refresh_token_hash, expires_at, refresh_expires_at, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = a.db.ExecContext(ctx, query, userID, accessTokenHash, refreshTokenHash, expiresAt, refreshExpiresAt, ipAddress, userAgent)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// ValidateToken validates JWT token and returns claims
func (a *AuthService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// GetUserByID retrieves user by ID
func (a *AuthService) GetUserByID(ctx context.Context, userID string) (*User, error) {
	var user User
	var firstName, lastName sql.NullString
	var lastLoginAt sql.NullTime
	var oauthProvider sql.NullString

	query := `
		SELECT id, account_id, email, role, first_name, last_name,
		       active, email_verified, oauth_provider, last_login_at, created_at, updated_at
		FROM control_users
		WHERE id = $1
	`

	err := a.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.AccountID, &user.Email, &user.Role,
		&firstName, &lastName, &user.Active, &user.EmailVerified,
		&oauthProvider, &lastLoginAt, &user.CreatedAt, &user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Populate optional fields
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if oauthProvider.Valid {
		user.OAuthProvider = &oauthProvider.String
	}

	return &user, nil
}

// RefreshToken validates a refresh token and returns new access and refresh tokens
func (a *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*User, string, string, time.Time, error) {
	// Validate refresh token
	claims, err := a.ValidateToken(refreshTokenString)
	if err != nil {
		return nil, "", "", time.Time{}, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Verify this is a refresh token (not an access token)
	if claims.Issuer != "bytefreezer-control-refresh" {
		return nil, "", "", time.Time{}, fmt.Errorf("not a refresh token")
	}

	// Get updated user information
	user, err := a.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return nil, "", "", time.Time{}, fmt.Errorf("user not found: %w", err)
	}

	// Check if user is still active
	if !user.Active {
		return nil, "", "", time.Time{}, fmt.Errorf("user account is inactive")
	}

	// Generate new token pair
	newAccessToken, newRefreshToken, expiresAt, err := a.GenerateTokenPair(user)
	if err != nil {
		return nil, "", "", time.Time{}, fmt.Errorf("failed to generate new tokens: %w", err)
	}

	return user, newAccessToken, newRefreshToken, expiresAt, nil
}

// DeleteSession removes session by token hash
func (a *AuthService) DeleteSession(ctx context.Context, tokenString string) error {
	tokenHash, err := hashToken(tokenString)
	if err != nil {
		return fmt.Errorf("failed to hash token: %w", err)
	}

	_, err = a.db.ExecContext(ctx, "DELETE FROM control_sessions WHERE token_hash = $1", tokenHash)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions
func (a *AuthService) CleanupExpiredSessions(ctx context.Context) error {
	result, err := a.db.ExecContext(ctx, "DELETE FROM control_sessions WHERE expires_at < NOW()")
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Infof("Cleaned up %d expired sessions", rowsAffected)
	}

	return nil
}

// Helper function to hash tokens
func hashToken(token string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

// GenerateRandomToken generates a cryptographically secure random token
func GenerateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
