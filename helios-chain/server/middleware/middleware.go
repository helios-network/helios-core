package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"cosmossdk.io/log"
)

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
			// Use non-destructive method extraction
			method := extractMethodFromRequestNonDestructive(r)
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

// CombinedMiddleware combines rate limiting and connection limiting
func CombinedMiddleware(rateLimiter *RateLimiter, connLimiter *ConnectionLimiter, logger log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// Apply rate limiting first
		rateLimited := RateLimitMiddleware(rateLimiter, logger)(next)
		// Then apply connection limiting
		return ConnectionLimitMiddleware(connLimiter, logger)(rateLimited)
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

// extractMethodFromRequestNonDestructive extracts the method name without consuming the request body
func extractMethodFromRequestNonDestructive(r *http.Request) string {
	// Only process POST requests with JSON content
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
		return ""
	}

	// Check if body is nil
	if r.Body == nil {
		return ""
	}

	// Read a small portion of the body to extract method without consuming it
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		return ""
	}

	// Restore the body for actual processing
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Parse JSON to extract method
	var request struct {
		Method string `json:"method"`
	}

	if err := json.Unmarshal(bodyBytes, &request); err == nil {
		return request.Method
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
