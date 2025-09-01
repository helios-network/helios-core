package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(5, time.Second)

	// Test basic rate limiting
	require.True(t, rl.Allow("192.168.1.1"))
	require.True(t, rl.Allow("192.168.1.1"))
	require.True(t, rl.Allow("192.168.1.1"))
	require.True(t, rl.Allow("192.168.1.1"))
	require.True(t, rl.Allow("192.168.1.1"))
	require.False(t, rl.Allow("192.168.1.1")) // Should be blocked

	// Test different IPs
	require.True(t, rl.Allow("192.168.1.2"))
	require.True(t, rl.Allow("192.168.1.3"))

	// Test cleanup
	time.Sleep(1100 * time.Millisecond)      // Wait for window to expire
	require.True(t, rl.Allow("192.168.1.1")) // Should work again
}

func TestConnectionLimiter(t *testing.T) {
	cl := NewConnectionLimiter(3)

	// Test connection limiting
	require.True(t, cl.AllowConnection("192.168.1.1"))
	require.True(t, cl.AllowConnection("192.168.1.2"))
	require.True(t, cl.AllowConnection("192.168.1.3"))
	require.False(t, cl.AllowConnection("192.168.1.4")) // Should be blocked

	// Test connection removal
	cl.RemoveConnection("192.168.1.1")
	require.True(t, cl.AllowConnection("192.168.1.4")) // Should work now

	// Test metrics
	metrics := cl.GetMetrics()
	require.Equal(t, 3, metrics["current_connections"])
	require.Equal(t, 3, metrics["max_connections"])
	require.Equal(t, 0, metrics["available_slots"])
}

func TestRateLimitMiddleware(t *testing.T) {
	rl := NewRateLimiter(2, time.Second)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create mock logger
	mockLogger := &mockLogger{}

	// Apply middleware
	middleware := RateLimitMiddleware(rl, mockLogger)
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	// Test successful requests
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	// Test blocked request
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestConnectionLimitMiddleware(t *testing.T) {
	cl := NewConnectionLimiter(2)

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create mock logger
	mockLogger := &mockLogger{}

	// Apply middleware
	middleware := ConnectionLimitMiddleware(cl, mockLogger)
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"

	// Test successful connections
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}

	// Test blocked connection
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)
	require.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestGetClientIP(t *testing.T) {
	// Test X-Forwarded-For header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	require.Equal(t, "203.0.113.1", getClientIP(req))

	// Test X-Real-IP header
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "203.0.113.2")
	require.Equal(t, "203.0.113.2", getClientIP(req))

	// Test RemoteAddr fallback
	req = httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	require.Equal(t, "192.168.1.1", getClientIP(req))
}

func TestRateLimiterCleanup(t *testing.T) {
	rl := NewRateLimiter(5, 100*time.Millisecond)

	// Make some requests
	rl.Allow("192.168.1.1")
	rl.Allow("192.168.1.2")

	// Wait for cleanup
	time.Sleep(150 * time.Millisecond)

	// Check if cleanup worked
	metrics := rl.GetMetrics()
	require.Equal(t, 0, metrics["total_ips_tracked"])
}

func TestMethodTracker(t *testing.T) {
	mt := NewMethodTracker()

	// Track some methods
	mt.TrackMethod("eth_call", 100*time.Millisecond, false)
	mt.TrackMethod("eth_call", 200*time.Millisecond, false)
	mt.TrackMethod("eth_getBalance", 50*time.Millisecond, false)

	// Get stats
	stats := mt.GetMethodStats("eth_call")
	require.Equal(t, 2, stats.TotalCalls)
	require.Equal(t, 150*time.Millisecond, stats.AverageTime)
	require.Equal(t, 100*time.Millisecond, stats.MinTime)
	require.Equal(t, 200*time.Millisecond, stats.MaxTime)

	// Test error tracking
	mt.TrackMethod("eth_call", 300*time.Millisecond, true)
	stats = mt.GetMethodStats("eth_call")
	require.Equal(t, 1, stats.ErrorCount)
}

func TestMethodTrackerCalculations(t *testing.T) {
	mt := NewMethodTracker()

	// Track methods with errors
	mt.TrackMethod("eth_call", 100*time.Millisecond, false)
	mt.TrackMethod("eth_call", 200*time.Millisecond, true)
	mt.TrackMethod("eth_getBalance", 50*time.Millisecond, false)

	allStats := mt.GetAllMethodStats()

	// Test calculations
	totalCalls := CalculateTotalCalls(allStats)
	require.Equal(t, 3, totalCalls)

	totalErrors := CalculateTotalErrors(allStats)
	require.Equal(t, 1, totalErrors)

	avgTime := CalculateAverageResponseTime(allStats)
	require.Equal(t, 116*time.Millisecond, avgTime) // (100+200+50)/3 ≈ 116
}

func TestMethodTrackerReset(t *testing.T) {
	mt := NewMethodTracker()

	// Track some methods
	mt.TrackMethod("eth_call", 100*time.Millisecond, false)
	mt.TrackMethod("eth_getBalance", 50*time.Millisecond, false)

	// Verify stats exist
	allStats := mt.GetAllMethodStats()
	require.Equal(t, 2, len(allStats))

	// Reset
	mt.Reset()

	// Verify stats are cleared
	allStats = mt.GetAllMethodStats()
	require.Equal(t, 0, len(allStats))
}

func TestMethodRateLimiter(t *testing.T) {
	// Create method rate limiter with default limit 10 per second
	mrl := NewMethodRateLimiter(10, time.Second)

	// Set specific limits for certain methods
	mrl.SetMethodLimit("eth_call", 1)
	mrl.SetMethodLimit("eth_estimateGas", 1)

	// Test default limit for unknown method
	require.True(t, mrl.Allow("eth_getBalance", "192.168.1.1"))
	require.True(t, mrl.Allow("eth_getBalance", "192.168.1.1"))
	require.True(t, mrl.Allow("eth_getBalance", "192.168.1.1"))
	// Should still allow after 3 calls (default limit is 10)

	// Test eth_call limit (1 per second)
	require.True(t, mrl.Allow("eth_call", "192.168.1.1"))
	require.False(t, mrl.Allow("eth_call", "192.168.1.1")) // Should be blocked

	// Test eth_estimateGas limit (1 per second)
	require.True(t, mrl.Allow("eth_estimateGas", "192.168.1.1"))
	require.False(t, mrl.Allow("eth_estimateGas", "192.168.1.1")) // Should be blocked

	// Test different IPs
	require.True(t, mrl.Allow("eth_call", "192.168.1.2"))        // Different IP should work
	require.True(t, mrl.Allow("eth_estimateGas", "192.168.1.2")) // Different IP should work
}

func TestMethodRateLimiterMetrics(t *testing.T) {
	mrl := NewMethodRateLimiter(5, time.Second)

	// Set method limits
	mrl.SetMethodLimit("eth_call", 1)
	mrl.SetMethodLimit("eth_estimateGas", 2)

	// Make some requests
	mrl.Allow("eth_call", "192.168.1.1")
	mrl.Allow("eth_estimateGas", "192.168.1.1")
	mrl.Allow("eth_getBalance", "192.168.1.1")

	// Get metrics
	allMetrics := mrl.GetAllMethodMetrics()

	// Verify metrics exist
	require.Contains(t, allMetrics, "eth_call")
	require.Contains(t, allMetrics, "eth_estimateGas")
	require.Contains(t, allMetrics, "eth_getBalance")

	// Verify specific method metrics
	ethCallMetrics := mrl.GetMethodMetrics("eth_call")
	require.Equal(t, 1, ethCallMetrics["limit"])
	require.Equal(t, "1s", ethCallMetrics["window_duration"])

	// Test reset
	mrl.Reset()

	// Metrics should be reset
	ethCallMetricsAfterReset := mrl.GetMethodMetrics("eth_call")
	require.Equal(t, 0, ethCallMetricsAfterReset["total_ips"])
}

// mockLogger implements log.Logger for testing
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, keyvals ...interface{}) {}
func (m *mockLogger) Info(msg string, keyvals ...interface{})  {}
func (m *mockLogger) Warn(msg string, keyvals ...interface{})  {}
func (m *mockLogger) Error(msg string, keyvals ...interface{}) {}
func (m *mockLogger) Impl() interface{}                        { return nil }
func (m *mockLogger) With(keyvals ...any) log.Logger           { return m }
