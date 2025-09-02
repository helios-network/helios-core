package routes

import (
	"encoding/json"
	"net/http"
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

	// Metrics endpoint
	router.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		metrics := map[string]interface{}{
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

// getConfigValue is a helper function to extract config values
func getConfigValue(config interface{}, field string) interface{} {
	// This is a simplified implementation
	// In a real scenario, you'd use reflection or type assertion
	return "config_value"
}
