package server

import (
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
			logger.Info(r.Msg, r.Ctx...)
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

	for _, api := range apis {
		//////////////////////////////
		// Swagger for rpc 8545 eth_
		//////////////////////////////
		if api.Namespace == "eth" {
			apiService, ok := api.Service.(*eth.PublicAPI)
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

	r.HandleFunc("/", rpcServer.ServeHTTP).Methods("POST")

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
