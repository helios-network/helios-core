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

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	ethlog "github.com/ethereum/go-ethereum/log"
	ethrpc "github.com/ethereum/go-ethereum/rpc"

	"helios-core/helios-chain/rpc/namespaces/ethereum/eth"

	svrconfig "helios-core/helios-chain/server/config"
	evmostypes "helios-core/helios-chain/types"
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
	rateLimiter := NewRateLimiter(config.JSONRPC.RateLimitRequestsPerSecond, config.JSONRPC.RateLimitWindow)

	// Create method-based rate limiter with method-specific limits
	methodRateLimiter := NewMethodRateLimiter(config.JSONRPC.RateLimitRequestsPerSecond, config.JSONRPC.RateLimitWindow)

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
	connLimiter := NewConnectionLimiter(config.JSONRPC.MaxConcurrentConnections)

	// Create method tracker for performance monitoring
	methodTracker := NewMethodTracker()

	// Apply the combined middleware to all routes
	r.Use(CombinedMiddleware(rateLimiter, connLimiter, ctx.Logger))

	ctx.Logger.Info("Applied rate limiting middleware",
		"requests_per_second", config.JSONRPC.RateLimitRequestsPerSecond,
		"window_duration", config.JSONRPC.RateLimitWindow,
		"max_concurrent_connections", config.JSONRPC.MaxConcurrentConnections)

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
		clientIP := getClientIP(r)

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
				done <- true
			}()

			// Create a response writer that captures the response
			rw := &responseWriter{ResponseWriter: w}

			// Call the actual RPC server
			rpcServer.ServeHTTP(rw, r.WithContext(reqCtx))
		}()

		// Wait for completion or timeout
		select {
		case <-done:
			// Request completed successfully
			duration := time.Since(start)

			// Track the method call
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

			// Track the timeout as an error
			if method != "" {
				methodTracker.TrackMethod(method, duration, true)
			} else {
				methodTracker.TrackMethod("unknown", duration, true)
			}

			// Log the timeout (not critical anymore)
			ctx.Logger.Warn("JSON-RPC request exceeded max duration, cancelling request",
				"method", method,
				"duration", duration,
				"max_duration", config.JSONRPC.MaxRequestDuration,
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

			// The request is already cancelled by the context timeout
			// No need to force exit the process
		}
	}).Methods("POST")

	// Add monitoring endpoint for rate limiting and connection stats
	r.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		status := map[string]interface{}{
			"rate_limiting": map[string]interface{}{
				"enabled": true,
				"limit":   config.JSONRPC.RateLimitRequestsPerSecond,
				"window":  config.JSONRPC.RateLimitWindow.String(),
			},
			"connection_limiting": map[string]interface{}{
				"enabled":             true,
				"max_connections":     config.JSONRPC.MaxConcurrentConnections,
				"current_connections": connLimiter.GetConnectionCount(),
			},
			"request_timeout": map[string]interface{}{
				"enabled":      true,
				"max_duration": config.JSONRPC.MaxRequestDuration.String(),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		// Convert to JSON
		json.NewEncoder(w).Encode(status)
	}).Methods("GET")

	// Add detailed metrics endpoint for rate limiting
	r.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Get detailed metrics from both limiters
		metrics := map[string]interface{}{
			"rate_limiter":        rateLimiter.GetMetrics(),
			"method_rate_limiter": methodRateLimiter.GetAllMethodMetrics(),
			"connection_limiter":  connLimiter.GetMetrics(),
			"method_tracker":      methodTracker.GetAllMethodStats(), // Add method tracker metrics
			"rate_limit_info": map[string]interface{}{
				"requests_per_second": config.JSONRPC.RateLimitRequestsPerSecond,
				"window_duration":     config.JSONRPC.RateLimitWindow.String(),
			},
			"request_timeout_info": map[string]interface{}{
				"max_duration": config.JSONRPC.MaxRequestDuration.String(),
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(metrics)
	}).Methods("GET")

	// Add reset endpoint for rate limiting (useful for testing and maintenance)
	r.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		// Only allow POST method for security
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Reset rate limiting counters
		rateLimiter.Reset()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		response := map[string]interface{}{
			"message":   "Rate limiting counters reset successfully",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
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
