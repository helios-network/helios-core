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
package flags

import (
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// CometBFT/cosmos-sdk full-node start flags
const (
	WithCometBFT = "with-cometbft"
	Address      = "address"
	Transport    = "transport"
	TraceStore   = "trace-store"
	CPUProfile   = "cpu-profile"
	// The type of database for application and snapshots databases
	AppDBBackend = "app-db-backend"
)

// GRPC-related flags.
const (
	GRPCOnly       = "grpc-only"
	GRPCEnable     = "grpc.enable"
	GRPCAddress    = "grpc.address"
	GRPCWebEnable  = "grpc-web.enable"
	GRPCWebAddress = "grpc-web.address"
)

// Cosmos API flags
const (
	RPCEnable         = "api.enable"
	EnabledUnsafeCors = "api.enabled-unsafe-cors"
)

// JSON-RPC flags
const (
	JSONRPCEnable              = "json-rpc.enable"
	JSONRPCAPI                 = "json-rpc.api"
	JSONRPCAddress             = "json-rpc.address"
	JSONWsAddress              = "json-rpc.ws-address"
	JSONRPCGasCap              = "json-rpc.gas-cap"
	JSONRPCEVMTimeout          = "json-rpc.evm-timeout"
	JSONRPCTxFeeCap            = "json-rpc.txfee-cap"
	JSONRPCFilterCap           = "json-rpc.filter-cap"
	JSONRPCFeeHistoryCap       = "json-rpc.feehistory-cap"
	JSONRPCLogsCap             = "json-rpc.logs-cap"
	JSONRPCBlockRangeCap       = "json-rpc.block-range-cap"
	JSONRPCHTTPTimeout         = "json-rpc.http-timeout"
	JSONRPCHTTPIdleTimeout     = "json-rpc.http-idle-timeout"
	JSONRPCAllowUnprotectedTxs = "json-rpc.allow-unprotected-txs"
	JSONRPCMaxOpenConnections  = "json-rpc.max-open-connections"
	JSONRPCEnableIndexer       = "json-rpc.enable-indexer"
	JSONRPCAllowIndexerGap     = "json-rpc.allow-indexer-gap"
	// JSONRPCEnableMetrics enables EVM RPC metrics server.
	// Set to `metrics` which is hardcoded flag from go-ethereum.
	JSONRPCEnableMetrics            = "metrics"
	JSONRPCFixRevertGasRefundHeight = "json-rpc.fix-revert-gas-refund-height"
	JSONRPCReturnDataLimit          = "json-rpc.return-data-limit"
)

// EVM flags
const (
	EVMTracer          = "evm.tracer"
	EVMMaxTxGasWanted  = "evm.max-tx-gas-wanted"
	EVMBlockExecutor   = "evm.block-executor"
	EVMBlockSTMWorkers = "evm.block-stm-workers"
)

// TLS flags
const (
	TLSCertPath = "tls.certificate-path"
	TLSKeyPath  = "tls.key-path"
)

// AddGlobalFlags adds global flags to the provided Cobra command.
func AddGlobalFlags(cmd *cobra.Command) (*cobra.Command, error) {
	cmd.PersistentFlags().String(flags.FlagChainID, "ethermint-9000", "Specify Chain ID for sending Tx")
	cmd.PersistentFlags().String(flags.FlagNode, "tcp://localhost:26657", "<host>:<port> to CometBFT RPC interface for this chain")
	cmd.PersistentFlags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Select keyring's backend")

	if err := viper.BindPFlag(flags.FlagNode, cmd.PersistentFlags().Lookup(flags.FlagNode)); err != nil {
		return nil, err
	}
	if err := viper.BindPFlag(flags.FlagKeyringBackend, cmd.PersistentFlags().Lookup(flags.FlagKeyringBackend)); err != nil {
		return nil, err
	}
	return cmd, nil
}
