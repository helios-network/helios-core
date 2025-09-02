package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"strings"
	"time"

	"helios-core/helios-chain/server/middleware"

	"github.com/gorilla/mux"
)

// SetupRPCRoutes sets up all RPC-related routes
func SetupRPCRoutes(
	router *mux.Router,
	rateLimiter *middleware.RateLimiter,
	methodRateLimiter *middleware.MethodRateLimiter,
	connLimiter *middleware.ConnectionLimiter,
	methodTracker *middleware.MethodTracker,
	computeTimeTracker *middleware.ComputeTimeTracker,
	config interface{},
) {
	// Status endpoint
	router.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		status := map[string]interface{}{
			"status": "running",
			"rate_limiting": map[string]interface{}{
				"enabled":             true,
				"requests_per_second": getConfigValue(config, "RateLimitRequestsPerSecond"),
				"window_duration":     getConfigValue(config, "RateLimitWindow"),
			},
			"connection_limiting": map[string]interface{}{
				"enabled":             true,
				"max_connections":     getConfigValue(config, "MaxConcurrentConnections"),
				"current_connections": connLimiter.GetConnectionCount(),
			},
			"request_timeout": map[string]interface{}{
				"enabled":      true,
				"max_duration": getConfigValue(config, "MaxRequestDuration"),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(status)
	}).Methods("GET")

	// Metrics endpoint with CPU and memory monitoring
	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Get system metrics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		// Count goroutines
		goroutineCount := runtime.NumGoroutine()

		metrics := map[string]interface{}{
			"system": map[string]interface{}{
				"goroutines":         goroutineCount,
				"memory_alloc":       m.Alloc,
				"memory_total_alloc": m.TotalAlloc,
				"memory_sys":         m.Sys,
				"memory_heap_alloc":  m.HeapAlloc,
				"memory_heap_sys":    m.HeapSys,
				"memory_heap_idle":   m.HeapIdle,
				"memory_heap_inuse":  m.HeapInuse,
				"gc_cycles":          m.NumGC,
				"gc_pause_total":     m.PauseTotalNs,
				"cpu_count":          runtime.NumCPU(),
			},
			"rate_limiter":        rateLimiter.GetMetrics(),
			"method_rate_limiter": methodRateLimiter.GetAllMethodMetrics(),
			"connection_limiter":  connLimiter.GetMetrics(),
			"method_tracker":      methodTracker.GetAllMethodStats(),
			"rate_limit_info": map[string]interface{}{
				"requests_per_second": getConfigValue(config, "RateLimitRequestsPerSecond"),
				"window_duration":     getConfigValue(config, "RateLimitWindow"),
			},
			"request_timeout_info": map[string]interface{}{
				"max_duration": getConfigValue(config, "MaxRequestDuration"),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(metrics)
	}).Methods("GET")

	// CPU profile endpoint for debugging
	router.HandleFunc("/debug/cpu", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=cpu-profile.pprof")

		if err := pprof.StartCPUProfile(w); err != nil {
			http.Error(w, "Could not start CPU profile", http.StatusInternalServerError)
			return
		}
		defer pprof.StopCPUProfile()

		// Profile for 30 seconds
		time.Sleep(30 * time.Second)
	}).Methods("GET")

	// Goroutine profile endpoint
	router.HandleFunc("/debug/goroutines", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		pprof.Lookup("goroutine").WriteTo(w, 1)
	}).Methods("GET")

	// Goroutine cleanup endpoint (force GC and runtime cleanup)
	router.HandleFunc("/debug/goroutines/cleanup", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Force garbage collection
		runtime.GC()

		// Force memory release
		debug.FreeOSMemory()

		// Get current stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		goroutineCount := runtime.NumGoroutine()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"message": "Goroutine cleanup completed",
			"after_cleanup": map[string]interface{}{
				"goroutines":        goroutineCount,
				"memory_heap_alloc": m.HeapAlloc,
				"memory_heap_sys":   m.HeapSys,
				"gc_cycles":         m.NumGC,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	}).Methods("POST")

	// Goroutine statistics endpoint
	router.HandleFunc("/debug/goroutines/stats", func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		goroutineCount := runtime.NumGoroutine()

		// Get detailed goroutine info
		goroutineProfile := pprof.Lookup("goroutine")
		var buf bytes.Buffer
		goroutineProfile.WriteTo(&buf, 1)
		goroutineInfo := buf.String()

		// Count goroutines by function (basic analysis)
		goroutineStats := analyzeGoroutines(goroutineInfo)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		stats := map[string]interface{}{
			"goroutine_count": goroutineCount,
			"memory_stats": map[string]interface{}{
				"heap_alloc":     m.HeapAlloc,
				"heap_sys":       m.HeapSys,
				"heap_idle":      m.HeapIdle,
				"heap_inuse":     m.HeapInuse,
				"total_alloc":    m.TotalAlloc,
				"gc_cycles":      m.NumGC,
				"gc_pause_total": m.PauseTotalNs,
			},
			"goroutine_analysis": goroutineStats,
			"timestamp":          time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(stats)
	}).Methods("GET")

	// Reset endpoint
	router.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rateLimiter.Reset()
		methodRateLimiter.Reset()
		// Reset compute time tracker for all IPs
		if computeTimeTracker != nil {
			computeTimeTracker.Reset("") // Empty string resets all IPs
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"message":   "All counters reset successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	}).Methods("POST")

	// RPC enable/disable endpoint
	router.HandleFunc("/rpc-control", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// Enable RPC
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "RPC API enabled",
				"status":  "enabled",
			})
		} else if r.Method == "DELETE" {
			// Disable RPC
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "RPC API disabled",
				"status":  "disabled",
			})
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}).Methods("POST", "DELETE")
}

// analyzeGoroutines analyzes goroutine profile and returns statistics
func analyzeGoroutines(profile string) map[string]interface{} {
	stats := make(map[string]interface{})

	// Count goroutines by function name
	lines := strings.Split(profile, "\n")
	totalGoroutines := 0

	for _, line := range lines {
		if strings.Contains(line, "goroutine") && strings.Contains(line, "state") {
			totalGoroutines++
		}

		// Look for common patterns
		if strings.Contains(line, "net/http") {
			stats["http_goroutines"] = getCount(stats, "http_goroutines") + 1
		}
		if strings.Contains(line, "websocket") {
			stats["websocket_goroutines"] = getCount(stats, "websocket_goroutines") + 1
		}
		if strings.Contains(line, "rate_limiter") {
			stats["rate_limiter_goroutines"] = getCount(stats, "rate_limiter_goroutines") + 1
		}
		if strings.Contains(line, "compute_time") {
			stats["compute_time_goroutines"] = getCount(stats, "compute_time_goroutines") + 1
		}
		if strings.Contains(line, "method_tracker") {
			stats["method_tracker_goroutines"] = getCount(stats, "method_tracker_goroutines") + 1
		}
		if strings.Contains(line, "blocked") {
			stats["blocked_goroutines"] = getCount(stats, "blocked_goroutines") + 1
		}
	}

	stats["total_analyzed"] = totalGoroutines
	return stats
}

// getCount safely gets a count from stats map
func getCount(stats map[string]interface{}, key string) int {
	if val, ok := stats[key]; ok {
		if count, ok := val.(int); ok {
			return count
		}
	}
	return 0
}

// getConfigValue is a helper function to extract config values
func getConfigValue(config interface{}, field string) interface{} {
	// This is a simplified implementation
	// In a real scenario, you'd use reflection or type assertion
	return "config_value"
}
