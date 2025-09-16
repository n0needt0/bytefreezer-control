package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/n0needt0/bytefreezer-control/config"
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

// RateLimiter manages rate limiting for different IPs
type RateLimiter struct {
	buckets map[string]*TokenBucket
	config  config.RateLimitConfig
	mutex   sync.RWMutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config config.RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		config:  config,
	}

	// Start cleanup goroutine to remove old buckets
	go rl.cleanupBuckets()

	return rl
}

// RateLimitMiddleware provides rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			clientIP := getClientIP(r)

			// Check if request is allowed
			if !rl.allowRequest(clientIP) {
				log.Warnf("Rate limit exceeded for IP: %s", clientIP)
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// allowRequest checks if the request should be allowed based on rate limiting
func (rl *RateLimiter) allowRequest(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.buckets[clientIP]
	if !exists {
		bucket = &TokenBucket{
			tokens:     rl.config.BurstSize,
			capacity:   rl.config.BurstSize,
			refillRate: rl.config.RequestsPerMinute,
			lastRefill: time.Now(),
		}
		rl.buckets[clientIP] = bucket
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
		for ip, bucket := range rl.buckets {
			bucket.mutex.Lock()
			if now.Sub(bucket.lastRefill) > 30*time.Minute {
				delete(rl.buckets, ip)
			}
			bucket.mutex.Unlock()
		}
		rl.mutex.Unlock()
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		return xff
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
