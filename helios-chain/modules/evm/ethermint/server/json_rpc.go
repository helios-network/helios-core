// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. 
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"golang.org/x/sync/errgroup"

	rpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	ethlog "github.com/ethereum/go-ethereum/log"
	ethrpc "github.com/ethereum/go-ethereum/rpc"
	"helios-core/helios-chain/app/ante"
	"helios-core/helios-chain/modules/evm/ethermint/rpc"
	"helios-core/helios-chain/modules/evm/ethermint/rpc/stream"
	"helios-core/helios-chain/modules/evm/ethermint/server/config"
	ethermint "helios-core/helios-chain/modules/evm/ethermint/types"

	sdklog "cosmossdk.io/log"
)

const (
	ServerStartTime = 5 * time.Second
	MaxRetry        = 6
)

type AppWithPendingTxStream interface {
	RegisterPendingTxListener(listener ante.PendingTxListener)
}

// StartJSONRPC starts the JSON-RPC server
func StartJSONRPC(srvCtx *server.Context,
	clientCtx client.Context,
	g *errgroup.Group,
	config *config.Config,
	indexer ethermint.EVMTxIndexer,
	app AppWithPendingTxStream,
) (*http.Server, chan struct{}, error) {
	logger := srvCtx.Logger.With("module", "geth")

	evtClient, ok := clientCtx.Client.(rpcclient.EventsClient)
	if !ok {
		return nil, nil, fmt.Errorf("client %T does not implement EventsClient", clientCtx.Client)
	}

	var rpcStream *stream.RPCStream
	var err error
	for i := 0; i < MaxRetry; i++ {
		rpcStream, err = stream.NewRPCStreams(evtClient, logger, clientCtx.TxConfig.TxDecoder())
		if err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create rpc streams after %d attempts: %w", MaxRetry, err)
	}

	app.RegisterPendingTxListener(rpcStream.ListenPendingTx)

	handler := NewWrappedSdkLogger(logger)
	ethlog.SetDefault(ethlog.NewLogger(handler))

	rpcServer := ethrpc.NewServer()

	allowUnprotectedTxs := config.JSONRPC.AllowUnprotectedTxs
	rpcAPIArr := config.JSONRPC.API

	apis := rpc.GetRPCAPIs(srvCtx, clientCtx, rpcStream, allowUnprotectedTxs, indexer, rpcAPIArr)

	for _, api := range apis {
		if err := rpcServer.RegisterName(api.Namespace, api.Service); err != nil {
			srvCtx.Logger.Error(
				"failed to register service in JSON RPC namespace",
				"namespace", api.Namespace,
				"service", api.Service,
			)
			return nil, nil, err
		}
	}

	r := mux.NewRouter()
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

	g.Go(func() error {
		srvCtx.Logger.Info("Starting JSON-RPC server", "address", config.JSONRPC.Address)
		if err := httpSrv.Serve(ln); err != nil {
			if err == http.ErrServerClosed {
				close(httpSrvDone)
			}

			srvCtx.Logger.Error("failed to start JSON-RPC server", "error", err.Error())
			return err
		}
		return nil
	})

	srvCtx.Logger.Info("Starting JSON WebSocket server", "address", config.JSONRPC.WsAddress)

	wsSrv := rpc.NewWebsocketsServer(clientCtx, srvCtx.Logger, rpcStream, config)
	wsSrv.Start()
	return httpSrv, httpSrvDone, nil
}

type WrappedSdkLogger struct {
	logger sdklog.Logger
}

func NewWrappedSdkLogger(logger sdklog.Logger) *WrappedSdkLogger {
	return &WrappedSdkLogger{
		logger: logger,
	}
}

func (c *WrappedSdkLogger) Handle(ctx context.Context, r slog.Record) error {
	switch r.Level {
	case ethlog.LvlTrace, ethlog.LvlDebug:
		c.logger.Debug(r.Message, ctx)
	case ethlog.LvlInfo, ethlog.LevelWarn:
		c.logger.Info(r.Message, ctx)
	case ethlog.LevelError, ethlog.LevelCrit:
		c.logger.Error(r.Message, ctx)
	}
	return nil
}

func (h *WrappedSdkLogger) Enabled(_ context.Context, level slog.Level) bool {
	return true
}

func (h *WrappedSdkLogger) WithGroup(_ string) slog.Handler {
	return h
}

func (h *WrappedSdkLogger) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}
