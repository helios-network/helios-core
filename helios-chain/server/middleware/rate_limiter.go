package middleware

import (
	"sync"
	"time"
)

// RateLimiter implements a simple rate limiting mechanism
type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int           // maximum requests per window
	window   time.Duration // time window for rate limiting
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get or create the request history for this IP
	requests, exists := rl.requests[ip]
	if !exists {
		requests = make([]time.Time, 0)
	}

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, reqTime := range requests {
		if reqTime.After(windowStart) {
			validRequests = append(validRequests, reqTime)
		}
	}

	// Check if we're under the limit
	if len(validRequests) >= rl.limit {
		return false
	}

	// Add current request
	validRequests = append(validRequests, now)
	rl.requests[ip] = validRequests

	return true
}

// Cleanup removes old entries to prevent memory leaks
func (rl *RateLimiter) Cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	for ip, requests := range rl.requests {
		validRequests := make([]time.Time, 0)
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				validRequests = append(validRequests, reqTime)
			}
		}

		if len(validRequests) == 0 {
			delete(rl.requests, ip)
		} else {
			rl.requests[ip] = validRequests
		}
	}
}

// GetMetrics returns metrics for the rate limiter
func (rl *RateLimiter) GetMetrics() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	totalIPs := len(rl.requests)
	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Count active requests in current window
	activeRequests := 0
	for _, requests := range rl.requests {
		for _, reqTime := range requests {
			if reqTime.After(windowStart) {
				activeRequests++
			}
		}
	}

	return map[string]interface{}{
		"active_requests":   activeRequests,
		"total_ips_tracked": totalIPs,
		"limit":             rl.limit,
		"window_duration":   rl.window.String(),
	}
}

// Reset clears all rate limiting data
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Clear all requests
	for ip := range rl.requests {
		delete(rl.requests, ip)
	}
}
