package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
	"github.com/n0needt0/bytefreezer-control/services"
	"github.com/n0needt0/go-goodies/log"
)

// TokenBucket represents a token bucket for rate limiting
type TokenBucket struct {
	tokens     int
	capacity   int
	refillRate int
	lastRefill time.Time
	mutex      sync.Mutex
}

// RateLimiter manages rate limiting for different users
type RateLimiter struct {
	buckets     map[string]*TokenBucket // keyed by user ID
	config      config.RateLimitConfig
	authService *services.AuthService
	mutex       sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config config.RateLimitConfig, authService *services.AuthService) *RateLimiter {
	rl := &RateLimiter{
		buckets:     make(map[string]*TokenBucket),
		config:      config,
		authService: authService,
	}

	// Start cleanup goroutine to remove old buckets
	go rl.cleanupBuckets()

	return rl
}

// RateLimitMiddleware provides token-based (per-user) rate limiting middleware
// This middleware must be applied AFTER JWT authentication middleware
func (rl *RateLimiter) RateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get user ID from context (set by JWT auth middleware)
			userID, ok := r.Context().Value("user_id").(string)
			if !ok || userID == "" {
				// No authenticated user - skip rate limiting for public endpoints
				// (or fall back to IP-based if needed)
				next.ServeHTTP(w, r)
				return
			}

			// Get user details to check rate limit settings
			user, err := rl.authService.GetUserByID(r.Context(), userID)
			if err != nil {
				log.Warnf("Failed to get user for rate limiting: %v", err)
				// Allow request if we can't get user details (fail open)
				next.ServeHTTP(w, r)
				return
			}

			// Check if rate limiting is enabled for this user
			if !user.RateLimitEnabled {
				next.ServeHTTP(w, r)
				return
			}

			// Check if request is allowed based on user's rate limits
			if !rl.allowRequest(userID, user.RateLimitRequestsPerMin, user.RateLimitBurstSize) {
				log.Warnf("Rate limit exceeded for user: %s (%s)", userID, user.Email)
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allowRequest checks if the request should be allowed based on user's rate limiting
func (rl *RateLimiter) allowRequest(userID string, requestsPerMin, burstSize int) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.buckets[userID]
	if !exists {
		bucket = &TokenBucket{
			tokens:     burstSize,
			capacity:   burstSize,
			refillRate: requestsPerMin,
			lastRefill: time.Now(),
		}
		rl.buckets[userID] = bucket
	} else {
		// Update bucket capacity if user's limits changed
		if bucket.capacity != burstSize || bucket.refillRate != requestsPerMin {
			bucket.capacity = burstSize
			bucket.refillRate = requestsPerMin
			if bucket.tokens > burstSize {
				bucket.tokens = burstSize
			}
		}
	}

	return bucket.consume()
}

// consume attempts to consume a token from the bucket
func (tb *TokenBucket) consume() bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Refill tokens based on time passed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tokensToAdd := int(elapsed.Minutes() * float64(tb.refillRate))

	if tokensToAdd > 0 {
		tb.tokens += tokensToAdd
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	// Check if we can consume a token
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// cleanupBuckets removes old unused buckets
func (rl *RateLimiter) cleanupBuckets() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for userID, bucket := range rl.buckets {
			bucket.mutex.Lock()
			if now.Sub(bucket.lastRefill) > 30*time.Minute {
				delete(rl.buckets, userID)
			}
			bucket.mutex.Unlock()
		}
		rl.mutex.Unlock()
	}
}
