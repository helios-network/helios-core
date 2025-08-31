package server

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"cosmossdk.io/log"
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

// ConnectionLimiter implements a connection limiting mechanism
type ConnectionLimiter struct {
	connections map[string]bool
	mutex       sync.RWMutex
	maxConn     int
}

// NewConnectionLimiter creates a new connection limiter
func NewConnectionLimiter(maxConnections int) *ConnectionLimiter {
	return &ConnectionLimiter{
		connections: make(map[string]bool),
		maxConn:     maxConnections,
	}
}

// AllowConnection checks if a new connection from the given IP is allowed
func (cl *ConnectionLimiter) AllowConnection(ip string) bool {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	if len(cl.connections) >= cl.maxConn {
		return false
	}

	cl.connections[ip] = true
	return true
}

// RemoveConnection removes a connection for the given IP
func (cl *ConnectionLimiter) RemoveConnection(ip string) {
	cl.mutex.Lock()
	defer cl.mutex.Unlock()

	delete(cl.connections, ip)
}

// GetConnectionCount returns the current number of connections
func (cl *ConnectionLimiter) GetConnectionCount() int {
	cl.mutex.RLock()
	defer cl.mutex.RUnlock()

	return len(cl.connections)
}

// GetMetrics returns metrics about the rate limiter
func (rl *RateLimiter) GetMetrics() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	return map[string]interface{}{
		"total_ips_tracked": len(rl.requests),
		"limit":             rl.limit,
		"window_duration":   rl.window.String(),
	}
}

// Reset clears all rate limiting data
func (rl *RateLimiter) Reset() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.requests = make(map[string][]time.Time)
}

// GetMetrics returns metrics about the connection limiter
func (cl *ConnectionLimiter) GetMetrics() map[string]interface{} {
	cl.mutex.RLock()
	defer cl.mutex.RUnlock()

	return map[string]interface{}{
		"current_connections": len(cl.connections),
		"max_connections":     cl.maxConn,
		"available_slots":     cl.maxConn - len(cl.connections),
	}
}

// RateLimitMiddleware creates a middleware that limits requests per IP
func RateLimitMiddleware(limiter *RateLimiter, logger log.Logger) func(http.Handler) http.Handler {
	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			limiter.Cleanup()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)

			// Check rate limit
			if !limiter.Allow(ip) {
				logger.Warn("rate limit exceeded", "ip", ip)
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ConnectionLimitMiddleware creates a middleware that limits concurrent connections
func ConnectionLimitMiddleware(limiter *ConnectionLimiter, logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)

			// Check connection limit
			if !limiter.AllowConnection(ip) {
				logger.Warn("connection limit exceeded", "ip", ip, "current_connections", limiter.GetConnectionCount())
				http.Error(w, "Connection limit exceeded. Please try again later.", http.StatusServiceUnavailable)
				return
			}

			// Remove connection when request is done
			defer limiter.RemoveConnection(ip)

			next.ServeHTTP(w, r)
		})
	}
}

// MethodTracker tracks method calls and their response times
type MethodTracker struct {
	methods map[string]*MethodStats
	mutex   sync.RWMutex
}

// MethodStats contains statistics for a specific method
type MethodStats struct {
	TotalCalls   int64         `json:"total_calls"`
	TotalTime    time.Duration `json:"total_time"`
	AverageTime  time.Duration `json:"average_time"`
	MinTime      time.Duration `json:"min_time"`
	MaxTime      time.Duration `json:"max_time"`
	LastCallTime time.Time     `json:"last_call_time"`
	ErrorCount   int64         `json:"error_count"`
}

// NewMethodTracker creates a new method tracker
func NewMethodTracker() *MethodTracker {
	return &MethodTracker{
		methods: make(map[string]*MethodStats),
	}
}

// TrackMethod tracks a method call and its response time
func (mt *MethodTracker) TrackMethod(method string, duration time.Duration, isError bool) {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	stats, exists := mt.methods[method]
	if !exists {
		stats = &MethodStats{
			MinTime: duration,
			MaxTime: duration,
		}
		mt.methods[method] = stats
	}

	// Update statistics
	stats.TotalCalls++
	stats.TotalTime += duration
	stats.AverageTime = time.Duration(stats.TotalTime.Nanoseconds() / stats.TotalCalls)
	stats.LastCallTime = time.Now()

	if duration < stats.MinTime {
		stats.MinTime = duration
	}
	if duration > stats.MaxTime {
		stats.MaxTime = duration
	}

	if isError {
		stats.ErrorCount++
	}
}

// GetMethodStats returns statistics for a specific method
func (mt *MethodTracker) GetMethodStats(method string) *MethodStats {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	if stats, exists := mt.methods[method]; exists {
		// Create a copy to avoid race conditions
		return &MethodStats{
			TotalCalls:   stats.TotalCalls,
			TotalTime:    stats.TotalTime,
			AverageTime:  stats.AverageTime,
			MinTime:      stats.MinTime,
			MaxTime:      stats.MaxTime,
			LastCallTime: stats.LastCallTime,
			ErrorCount:   stats.ErrorCount,
		}
	}
	return nil
}

// GetAllMethodStats returns statistics for all methods
func (mt *MethodTracker) GetAllMethodStats() map[string]*MethodStats {
	mt.mutex.RLock()
	defer mt.mutex.RUnlock()

	// Create a copy to avoid race conditions
	result := make(map[string]*MethodStats)
	for method, stats := range mt.methods {
		result[method] = &MethodStats{
			TotalCalls:   stats.TotalCalls,
			TotalTime:    stats.TotalTime,
			AverageTime:  stats.AverageTime,
			MinTime:      stats.MinTime,
			MaxTime:      stats.MaxTime,
			LastCallTime: stats.LastCallTime,
			ErrorCount:   stats.ErrorCount,
		}
	}
	return result
}

// Reset resets all method statistics
func (mt *MethodTracker) Reset() {
	mt.mutex.Lock()
	defer mt.mutex.Unlock()

	mt.methods = make(map[string]*MethodStats)
}

// CalculateTotalCalls calculates the total number of calls across all methods
func CalculateTotalCalls(methodStats map[string]*MethodStats) int64 {
	var total int64
	for _, stats := range methodStats {
		total += stats.TotalCalls
	}
	return total
}

// CalculateTotalErrors calculates the total number of errors across all methods
func CalculateTotalErrors(methodStats map[string]*MethodStats) int64 {
	var total int64
	for _, stats := range methodStats {
		total += stats.ErrorCount
	}
	return total
}

// CalculateAverageResponseTime calculates the average response time across all methods
func CalculateAverageResponseTime(methodStats map[string]*MethodStats) time.Duration {
	var totalCalls int64
	var totalTime time.Duration

	for _, stats := range methodStats {
		totalCalls += stats.TotalCalls
		totalTime += stats.TotalTime
	}

	if totalCalls == 0 {
		return 0
	}

	return time.Duration(totalTime.Nanoseconds() / totalCalls)
}

// MethodTrackingMiddleware creates a middleware that tracks method calls and response times
func MethodTrackingMiddleware(tracker *MethodTracker, logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a custom response writer to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Extract method from request body if it's a JSON-RPC request
			method := extractMethodFromRequest(r)
			if method == "" {
				method = "unknown"
			}

			// Track the method call
			isError := rw.statusCode >= 400
			tracker.TrackMethod(method, duration, isError)

			// Log slow requests
			if duration > 1*time.Second {
				logger.Warn("slow JSON-RPC request",
					"method", method,
					"duration", duration,
					"status", rw.statusCode)
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}

// extractMethodFromRequest extracts the method name from a JSON-RPC request
func extractMethodFromRequest(r *http.Request) string {
	// Only process POST requests with JSON content
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
		return ""
	}

	// Try to read the request body to extract method
	// Note: This is a simplified approach. In production, you might want to use a more sophisticated method
	// that doesn't consume the body for actual processing
	if r.Body != nil {
		// Create a limited reader to avoid reading large bodies
		limitedBody := http.MaxBytesReader(nil, r.Body, 1024) // Limit to 1KB

		// Parse JSON to extract method
		var request struct {
			Method string `json:"method"`
		}

		if err := json.NewDecoder(limitedBody).Decode(&request); err == nil {
			return request.Method
		}
	}

	return ""
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check for forwarded headers first
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	// Fall back to remote address
	ip := r.RemoteAddr
	if ip == "" {
		return "unknown"
	}

	// Remove port if present
	if colonIndex := len(ip) - 1; colonIndex >= 0 && ip[colonIndex] == ':' {
		ip = ip[:colonIndex]
	}

	return ip
}

// CombinedMiddleware combines rate limiting and connection limiting
func CombinedMiddleware(rateLimiter *RateLimiter, connLimiter *ConnectionLimiter, logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Apply rate limiting first
		rateLimited := RateLimitMiddleware(rateLimiter, logger)(next)
		// Then apply connection limiting
		return ConnectionLimitMiddleware(connLimiter, logger)(rateLimited)
	}
}
