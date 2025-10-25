package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"helios-core/helios-chain/rpc"
	"helios-core/helios-chain/rpc/backend"
	"helios-core/helios-chain/server/middleware"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	ethlog "github.com/ethereum/go-ethereum/log"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	"helios-core/helios-chain/rpc/namespaces/ethereum/eth"

	svrconfig "helios-core/helios-chain/server/config"
	"helios-core/helios-chain/server/routes"
	evmostypes "helios-core/helios-chain/types"
	"os"
	"runtime"

	cosmossdklog "cosmossdk.io/log"
)

// extractMethodFromRequestBody extracts the method name from a JSON-RPC request body
func extractMethodFromRequestBody(r *http.Request) string {
	// Only process POST requests with JSON content
	if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
		return ""
	}

	// Check if body is nil
	if r.Body == nil {
		return ""
	}

	// Read a small portion of the body to extract method without consuming it
	bodyBytes, err := io.ReadAll(r.Body) //io.LimitReader(r.Body, 1024))
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

// parseMethodRateLimits parses the method rate limits configuration string
// Format: "method1:limit1,method2:limit2" (e.g., "eth_call:1,eth_estimateGas:1")
func parseMethodRateLimits(configString string) map[string]int {
	result := make(map[string]int)

	if configString == "" {
		return result
	}

	// Split by comma
	methods := strings.Split(configString, ",")
	for _, method := range methods {
		// Split by colon
		parts := strings.Split(strings.TrimSpace(method), ":")
		if len(parts) == 2 {
			methodName := strings.TrimSpace(parts[0])
			limitStr := strings.TrimSpace(parts[1])

			// Parse limit
			if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
				result[methodName] = limit
			}
		}
	}

	return result
}

// startMonitoring starts automatic monitoring and logging every minute
func startMonitoring(
	logger cosmossdklog.Logger,
	rateLimiter *middleware.RateLimiter,
	methodRateLimiter *middleware.MethodRateLimiter,
	connLimiter *middleware.ConnectionLimiter,
	methodTracker *middleware.MethodTracker,
	computeTimeTracker *middleware.ComputeTimeTracker,
) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Get system metrics
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		goroutineCount := runtime.NumGoroutine()

		// Create monitoring data
		monitoringData := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
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
		}

		if computeTimeTracker != nil {
			monitoringData["compute_time_tracker"] = computeTimeTracker.GetMetrics()
		}

		// Convert to JSON
		jsonData, err := json.MarshalIndent(monitoringData, "", "  ")
		if err != nil {
			logger.Error("Failed to marshal monitoring data", "error", err)
			continue
		}

		// Log to file
		filename := fmt.Sprintf("monitoring_%s.log", time.Now().Format("2006-01-02"))
		if err := os.WriteFile(filename, jsonData, 0644); err != nil {
			logger.Error("Failed to write monitoring file", "error", err, "filename", filename)
			continue
		}

		// Log critical metrics to console
		logger.Info("Monitoring snapshot",
			"goroutines", goroutineCount,
			"memory_alloc_mb", m.Alloc/1024/1024,
			"memory_heap_mb", m.HeapAlloc/1024/1024,
			"gc_cycles", m.NumGC,
			"filename", filename)

		// Alert if critical thresholds are exceeded
		if goroutineCount > 10000 {
			logger.Error("CRITICAL: Too many goroutines", "count", goroutineCount)
		}
		if m.HeapAlloc > 1<<30 { // 1GB
			logger.Error("CRITICAL: High memory usage", "heap_alloc_mb", m.HeapAlloc/1024/1024)
		}
	}
}

// StartJSONRPC starts the JSON-RPC server
func StartJSONRPC(ctx *server.Context,
	clientCtx client.Context,
	tmRPCAddr,
	tmEndpoint string,
	config *svrconfig.Config,
	indexer evmostypes.EVMTxIndexer,
) (*http.Server, chan struct{}, error) {

	tmWsClient := ConnectTmWS(tmRPCAddr, tmEndpoint, ctx.Logger)

	logger := ctx.Logger.With("module", "geth")
	ethlog.Root().SetHandler(ethlog.FuncHandler(func(r *ethlog.Record) error {
		switch r.Lvl {
		case ethlog.LvlTrace, ethlog.LvlDebug:
			logger.Debug(r.Msg, r.Ctx...)
		case ethlog.LvlInfo, ethlog.LvlWarn:
			logger.Debug(r.Msg, r.Ctx...)
		case ethlog.LvlError, ethlog.LvlCrit:
			logger.Error(r.Msg, r.Ctx...)
		}
		return nil
	}))

	rpcServer := ethrpc.NewServer()

	allowUnprotectedTxs := config.JSONRPC.AllowUnprotectedTxs
	rpcAPIArr := config.JSONRPC.API

	apis := rpc.GetRPCAPIs(ctx, clientCtx, tmWsClient, allowUnprotectedTxs, indexer, rpcAPIArr)

	r := mux.NewRouter()

	// Apply rate limiting and connection limiting middleware
	// Use configuration values for rate limiting
	rateLimiter := middleware.NewRateLimiter(config.JSONRPC.RateLimitRequestsPerSecond, config.JSONRPC.RateLimitWindow)

	// Create method-based rate limiter with method-specific limits
	methodRateLimiter := middleware.NewMethodRateLimiter(config.JSONRPC.RateLimitRequestsPerSecond, config.JSONRPC.RateLimitWindow)

	// Parse and apply method-specific rate limits from configuration
	methodLimits := parseMethodRateLimits(config.JSONRPC.MethodRateLimits)
	for method, limit := range methodLimits {
		methodRateLimiter.SetMethodLimit(method, limit)
		ctx.Logger.Info("Applied method-specific rate limit",
			"method", method,
			"limit", limit,
			"window", config.JSONRPC.RateLimitWindow)
	}

	// Use configuration values for connection limiting
	connLimiter := middleware.NewConnectionLimiter(config.JSONRPC.MaxConcurrentConnections)

	// Create method tracker for performance monitoring
	methodTracker := middleware.NewMethodTracker()

	// Create compute time tracker for intelligent prediction
	computeTimeTracker := middleware.NewComputeTimeTracker(
		config.JSONRPC.ComputeTimeLimitPerWindowPerIP,
		config.JSONRPC.ComputeTimeWindow,
	)

	// Link method tracker with compute time tracker
	methodTracker.SetComputeTimeTracker(computeTimeTracker)

	// Start automatic monitoring and logging
	go startMonitoring(ctx.Logger, rateLimiter, methodRateLimiter, connLimiter, methodTracker, computeTimeTracker)

	// Apply the combined middleware to all routes
	r.Use(middleware.CombinedMiddleware(rateLimiter, connLimiter, ctx.Logger))

	ctx.Logger.Info("Applied rate limiting middleware",
		"requests_per_second", config.JSONRPC.RateLimitRequestsPerSecond,
		"window_duration", config.JSONRPC.RateLimitWindow,
		"max_concurrent_connections", config.JSONRPC.MaxConcurrentConnections)

	// Setup organized RPC routes
	routes.SetupRPCRoutes(r, rateLimiter, methodRateLimiter, connLimiter, methodTracker, computeTimeTracker, &config.JSONRPC)

	for _, api := range apis {
		//////////////////////////////
		// Swagger for rpc 8545 eth_
		//////////////////////////////
		if api.Namespace == "eth" {
			apiService, ok := api.Service.(*eth.CachedPublicAPI)
			if ok {
				generateSwagger(ctx, apiService, r, config)

				apiService.StartCleanupCacheRoutine()
			}

		}
		//////////////////////////////
		if err := rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
			ctx.Logger.Error(
				"failed to register service in JSON RPC namespace",
				"namespace", api.Namespace,
				"service", api.Service,
			)
			return nil, nil, err
		}
	}

	// Create a wrapper around the RPC server to track method calls on the main endpoint
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Extract method from request body before processing
		method := extractMethodFromRequestBody(r)

		// Get client IP for rate limiting
		clientIP := middleware.GetClientIP(r)

		// Check method-specific rate limiting
		if !methodRateLimiter.Allow(method, clientIP) {
			ctx.Logger.Warn("Method rate limit exceeded",
				"method", method,
				"client_ip", clientIP)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": map[string]interface{}{
					"code":    -32029,
					"message": "Method rate limit exceeded",
					"data":    fmt.Sprintf("Method %s exceeded rate limit", method),
				},
			})
			return
		}

		// PREDICT compute time before execution to prevent timeouts
		if !computeTimeTracker.PredictComputeTime(clientIP, method) {
			ctx.Logger.Warn("Predicted compute time limit exceeded for IP",
				"ip", clientIP,
				"method", method,
				"limit", config.JSONRPC.ComputeTimeLimitPerWindowPerIP)

			// Send error response before execution
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": map[string]interface{}{
					"code":    -32030,
					"message": "Predicted compute time limit exceeded",
					"data":    fmt.Sprintf("IP %s would exceed compute time limit for method %s", clientIP, method),
				},
			})
			return
		}

		// Create context with timeout for max request duration
		reqCtx, cancel := context.WithTimeout(r.Context(), config.JSONRPC.MaxRequestDuration)
		defer cancel()

		// Track timing
		start := time.Now()

		// Create a channel to signal completion
		done := make(chan bool, 1)

		// Run the RPC server in a goroutine
		go func() {
			defer func() {
				// Recover from any panics
				if panicErr := recover(); panicErr != nil {
					ctx.Logger.Error("Panic in RPC server, recovering",
						"panic", panicErr,
						"method", method,
						"client_ip", clientIP)

					// Send error response to client
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"jsonrpc": "2.0",
						"id":      nil,
						"error": map[string]interface{}{
							"code":    -32603,
							"message": "Internal error",
							"data":    "Server panic recovered",
						},
					})
				}

				// Always signal completion (with timeout protection)
				select {
				case done <- true:
				default:
					// Channel full, request already handled
				}
			}()

			// Create a response writer that captures the response
			rw := &middleware.ResponseWriter{ResponseWriter: w}

			// Call the actual RPC server
			rpcServer.ServeHTTP(rw, r.WithContext(reqCtx))
		}()

		// Wait for completion or timeout
		select {
		case <-done:
			// Request completed successfully
			duration := time.Since(start)

			// Update compute time limit per IP
			computeTimeTracker.AddComputeTime(clientIP, duration)

			// Track the method call manually
			if method != "" {
				methodTracker.TrackMethod(method, duration, false)
			} else {
				methodTracker.TrackMethod("unknown", duration, false)
			}

			// Log slow requests
			if duration > 1*time.Second {
				ctx.Logger.Warn("slow JSON-RPC request",
					"method", method,
					"duration", duration)
			}

		case <-reqCtx.Done():
			// Request timed out - cancel the request and return error
			duration := time.Since(start)

			// Track the timeout as an error manually
			if method != "" {
				methodTracker.TrackMethod(method, duration, true)
			} else {
				methodTracker.TrackMethod("unknown", duration, true)
			}

			// Log the timeout (not critical anymore)
			logMsg := "JSON-RPC request exceeded max duration, cancelling request"
			if reqCtx.Err() != nil && strings.Contains(reqCtx.Err().Error(), "context canceled") && methodTracker.IsComputeTimeExceeded(method, clientIP) {
				logMsg = "JSON-RPC request cancelled due to predicted compute time limit exceeding"
			}
			ctx.Logger.Warn(logMsg,
				"method", method,
				"duration", duration.Milliseconds(),
				"max_duration", config.JSONRPC.MaxRequestDuration.Milliseconds(),
				"timeout_error", reqCtx.Err().Error())

			// Send error response to client
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestTimeout)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": map[string]interface{}{
					"code":    -32000,
					"message": "Request timeout exceeded",
					"data":    fmt.Sprintf("Request exceeded maximum duration of %v", config.JSONRPC.MaxRequestDuration),
				},
			})

		case <-time.After(config.JSONRPC.MaxRequestDuration + 5*time.Second):
			// Emergency timeout - something went wrong with the goroutine
			ctx.Logger.Error("Emergency timeout - RPC goroutine did not complete",
				"method", method,
				"client_ip", clientIP,
				"duration", time.Since(start))

			// Send error response to client
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      nil,
				"error": map[string]interface{}{
					"code":    -32603,
					"message": "Internal error",
					"data":    "Request processing timeout",
				},
			})
		}
	}).Methods("POST")

	handlerWithCors := cors.Default()
	if config.API.EnableUnsafeCORS {
		handlerWithCors = cors.AllowAll()
	}

	httpSrv := &http.Server{
		Addr:              config.JSONRPC.Address,
		Handler:           handlerWithCors.Handler(r),
		ReadHeaderTimeout: config.JSONRPC.HTTPTimeout,
		ReadTimeout:       config.JSONRPC.HTTPTimeout,
		WriteTimeout:      config.JSONRPC.HTTPTimeout,
		IdleTimeout:       config.JSONRPC.HTTPIdleTimeout,
	}
	httpSrvDone := make(chan struct{}, 1)

	ln, err := Listen(httpSrv.Addr, config)
	if err != nil {
		return nil, nil, err
	}

	errCh := make(chan error)
	go func() {
		ctx.Logger.Info("Starting JSON-RPC server", "address", config.JSONRPC.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				close(httpSrvDone)
				return
			}

			ctx.Logger.Error("failed to start JSON-RPC server", "error", err.Error())
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		ctx.Logger.Error("failed to boot JSON-RPC server", "error", err.Error())
		return nil, nil, err
	case <-time.After(svrconfig.ServerStartTime): // assume JSON RPC server started successfully
	}

	ctx.Logger.Info("Starting JSON WebSocket server", "address", config.JSONRPC.WsAddress)

	// allocate separate WS connection to Tendermint
	tmWsClient = ConnectTmWS(tmRPCAddr, tmEndpoint, ctx.Logger)
	backend := backend.NewBackend(ctx, ctx.Logger, clientCtx, allowUnprotectedTxs, indexer)
	wsSrv := rpc.NewWebsocketsServer(clientCtx, ctx.Logger, tmWsClient, config, backend)
	wsSrv.Start()
	return httpSrv, httpSrvDone, nil
}
