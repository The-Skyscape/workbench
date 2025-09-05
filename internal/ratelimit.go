package internal

import (
	"sync"
	"time"
)

// Simple in-memory rate limiter for authentication attempts
type RateLimiter struct {
	attempts map[string][]time.Time
	mu       sync.Mutex
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	// Clean up old entries periodically
	go rl.cleanup()
	return rl
}

// Allow checks if an attempt is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove old attempts
	var recent []time.Time
	for _, t := range rl.attempts[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	// Check if under limit
	if len(recent) >= rl.limit {
		rl.attempts[key] = recent
		return false
	}

	// Add this attempt
	recent = append(recent, now)
	rl.attempts[key] = recent
	return true
}

// cleanup removes old entries to prevent memory growth
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for key, attempts := range rl.attempts {
			var recent []time.Time
			for _, t := range attempts {
				if t.After(cutoff) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				delete(rl.attempts, key)
			} else {
				rl.attempts[key] = recent
			}
		}
		rl.mu.Unlock()
	}
}

// AuthRateLimiter is the global rate limiter for authentication
// Allows 5 attempts per minute per IP
var AuthRateLimiter = NewRateLimiter(5, time.Minute)