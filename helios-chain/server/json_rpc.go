package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
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

	// Start monitoring goroutine for rate limiting stats
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rateMetrics := rateLimiter.GetMetrics()
			connMetrics := connLimiter.GetMetrics()

			ctx.Logger.Info("Rate limiting stats",
				"ips_tracked", rateMetrics["total_ips_tracked"],
				"current_connections", connMetrics["current_connections"],
				"available_slots", connMetrics["available_slots"],
			)
		}
	}()

	for _, api := range apis {
		//////////////////////////////
		// Swagger for rpc 8545 eth_
		//////////////////////////////
		if api.Namespace == "eth" {
			apiService, ok := api.Service.(*eth.CachedPublicAPI)
			if ok {
				generateSwagger(ctx, apiService, r, config)
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

		// Track timing
		start := time.Now()

		// Call the actual RPC server
		rpcServer.ServeHTTP(w, r)

		// Calculate duration and track
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
			"rate_limiter":       rateLimiter.GetMetrics(),
			"connection_limiter": connLimiter.GetMetrics(),
			"method_tracker":     methodTracker.GetAllMethodStats(), // Add method tracker metrics
			"rate_limit_info": map[string]interface{}{
				"requests_per_second": config.JSONRPC.RateLimitRequestsPerSecond,
				"window_duration":     config.JSONRPC.RateLimitWindow.String(),
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
