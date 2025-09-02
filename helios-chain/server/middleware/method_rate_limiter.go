package middleware

import (
	"sync"
	"time"
)

// MethodRateLimiter provides rate limiting per method with different limits
type MethodRateLimiter struct {
	methodLimits map[string]*RateLimiter
	mutex        sync.RWMutex
	defaultLimit int
	window       time.Duration
}

// NewMethodRateLimiter creates a new method-based rate limiter
func NewMethodRateLimiter(defaultLimit int, window time.Duration) *MethodRateLimiter {
	return &MethodRateLimiter{
		methodLimits: make(map[string]*RateLimiter),
		defaultLimit: defaultLimit,
		window:       window,
	}
}

// SetMethodLimit sets a specific limit for a method
func (mrl *MethodRateLimiter) SetMethodLimit(method string, limit int) {
	mrl.mutex.Lock()
	defer mrl.mutex.Unlock()

	mrl.methodLimits[method] = NewRateLimiter(limit, mrl.window)
}

// Allow checks if a request is allowed for a specific method
func (mrl *MethodRateLimiter) Allow(method, ip string) bool {
	mrl.mutex.RLock()
	defer mrl.mutex.RUnlock()

	// Get method-specific limiter or use default
	limiter, exists := mrl.methodLimits[method]
	if !exists {
		// Create default limiter for this method if not exists
		limiter = NewRateLimiter(mrl.defaultLimit, mrl.window)
		mrl.methodLimits[method] = limiter
	}

	return limiter.Allow(ip)
}

// GetMethodMetrics returns metrics for a specific method
func (mrl *MethodRateLimiter) GetMethodMetrics(method string) map[string]interface{} {
	mrl.mutex.RLock()
	defer mrl.mutex.RUnlock()

	limiter, exists := mrl.methodLimits[method]
	if !exists {
		return map[string]interface{}{
			"limit":           mrl.defaultLimit,
			"window_duration": mrl.window.String(),
			"total_ips":       0,
		}
	}

	return limiter.GetMetrics()
}

// GetAllMethodMetrics returns metrics for all methods
func (mrl *MethodRateLimiter) GetAllMethodMetrics() map[string]interface{} {
	mrl.mutex.RLock()
	defer mrl.mutex.RUnlock()

	result := make(map[string]interface{})
	for method, limiter := range mrl.methodLimits {
		result[method] = limiter.GetMetrics()
	}

	return result
}

// Reset resets all method limiters
func (mrl *MethodRateLimiter) Reset() {
	mrl.mutex.Lock()
	defer mrl.mutex.Unlock()

	for _, limiter := range mrl.methodLimits {
		limiter.Reset()
	}
}
