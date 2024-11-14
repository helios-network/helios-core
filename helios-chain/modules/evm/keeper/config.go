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
package keeper

import (
	"math/big"

	cosmostracing "helios-core/helios-chain/modules/evm/tracing"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	rpctypes "helios-core/helios-chain/modules/evm/ethermint/rpc/types"
	"helios-core/helios-chain/modules/evm/statedb"
	"helios-core/helios-chain/modules/evm/types"
)

// EVMBlockConfig encapsulates the common parameters needed to execute an EVM message,
// it's cached in object store during the block execution.
type EVMBlockConfig struct {
	Params          types.Params
	// FeeMarketParams feemarkettypes.Params
	ChainConfig     *params.ChainConfig
	CoinBase        common.Address
	BaseFee         *big.Int
	// not supported, always zero
	Random *common.Hash
	// unused, always zero
	Difficulty *big.Int
	// cache the big.Int version of block number, avoid repeated allocation
	BlockNumber *big.Int
	BlockTime   uint64
	Rules       params.Rules
}

// EVMConfig encapsulates common parameters needed to create an EVM to execute a message
// It's mainly to reduce the number of method parameters
type EVMConfig struct {
	*EVMBlockConfig
	TxConfig       statedb.TxConfig
	Tracer         *cosmostracing.Hooks
	DebugTrace     bool
	Overrides      *rpctypes.StateOverride
	BlockOverrides *rpctypes.BlockOverrides
}

// EVMBlockConfig creates the EVMBlockConfig based on current state
func (k *Keeper) EVMBlockConfig(ctx sdk.Context, chainID *big.Int) (*EVMBlockConfig, error) {
	objStore := ctx.ObjectStore(k.objectKey)
	v := objStore.Get(types.KeyPrefixObjectParams)
	if v != nil {
		return v.(*EVMBlockConfig), nil
	}

	params := k.GetParams(ctx)
	ethCfg := params.ChainConfig.EthereumConfig(chainID)

	// feemarketParams := k.feeMarketKeeper.GetParams(ctx)

	// get the coinbase address from the block proposer
	coinbase, err := k.GetCoinbaseAddress(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to obtain coinbase address")
	}

	var baseFee *big.Int
	if types.IsLondon(ethCfg, ctx.BlockHeight()) {
		// baseFee = feemarketParams.GetBaseFee()
		// should not be nil if london hardfork enabled
		if baseFee == nil {
			baseFee = new(big.Int)
		}
	}

	blockTime := uint64(ctx.BlockHeader().Time.Unix())
	blockNumber := big.NewInt(ctx.BlockHeight())
	rules := ethCfg.Rules(blockNumber, ethCfg.MergeNetsplitBlock != nil, blockTime)

	var zero common.Hash
	cfg := &EVMBlockConfig{
		Params:          params,
		// FeeMarketParams: feemarketParams,
		ChainConfig:     ethCfg,
		CoinBase:        coinbase,
		BaseFee:         baseFee,
		Difficulty:      big.NewInt(0),
		Random:          &zero,
		BlockNumber:     blockNumber,
		BlockTime:       blockTime,
		Rules:           rules,
	}
	objStore.Set(types.KeyPrefixObjectParams, cfg)
	return cfg, nil
}

func (k *Keeper) RemoveParamsCache(ctx sdk.Context) {
	ctx.ObjectStore(k.objectKey).Delete(types.KeyPrefixObjectParams)
}

// EVMConfig creates the EVMConfig based on current state
func (k *Keeper) EVMConfig(ctx sdk.Context, chainID *big.Int, txHash common.Hash) (*EVMConfig, error) {
	blockCfg, err := k.EVMBlockConfig(ctx, chainID)
	if err != nil {
		return nil, err
	}

	var txConfig statedb.TxConfig
	if txHash == (common.Hash{}) {
		txConfig = statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))
	} else {
		txConfig = k.TxConfig(ctx, txHash)
	}

	cfg := &EVMConfig{
		EVMBlockConfig: blockCfg,
		TxConfig:       txConfig,
		Tracer:         k.evmTracer,
	}

	return cfg, nil
}

// TxConfig loads `TxConfig` from current transient storage
func (k *Keeper) TxConfig(ctx sdk.Context, txHash common.Hash) statedb.TxConfig {
	return statedb.NewTxConfig(
		common.BytesToHash(ctx.HeaderHash()), // BlockHash
		txHash,                               // TxHash
		0, 0,
	)
}

// VMConfig creates an EVM configuration from the debug setting and the extra EIPs enabled on the
// module parameters. The config generated uses the default JumpTable from the EVM.
func (k Keeper) VMConfig(ctx sdk.Context, cfg *EVMConfig) vm.Config {
	noBaseFee := true
	// if types.IsLondon(cfg.ChainConfig, ctx.BlockHeight()) {
	// 	noBaseFee = cfg.FeeMarketParams.NoBaseFee
	// }

	vmCfg := vm.Config{
		NoBaseFee: noBaseFee,
		ExtraEips: cfg.Params.EIPs(),
	}

	if vmCfg.Tracer == nil && cfg.Tracer != nil && cfg.Tracer.Hooks != nil {
		vmCfg.Tracer = cfg.Tracer.Hooks
	}

	return vmCfg
}
