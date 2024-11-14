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
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	ethermint "helios-core/helios-chain/modules/evm/ethermint/types"
	"helios-core/helios-chain/modules/evm/statedb"
	"helios-core/helios-chain/modules/evm/types"
)

// CustomContractFn defines a custom precompiled contract generator with ctx, rules and returns a precompiled contract.
type CustomContractFn func(sdk.Context, params.Rules) vm.PrecompiledContract

// Keeper grants access to the EVM module state and implements the go-ethereum StateDB interface.
type Keeper struct {
	// Protobuf codec
	cdc codec.Codec
	// Store key required for the EVM Prefix KVStore. It is required by:
	// - storing account's Storage State
	// - storing account's Code
	// - storing module parameters
	storeKey storetypes.StoreKey

	// key to access the object store, which is reset on every block during Commit
	objectKey storetypes.StoreKey

	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	authority sdk.AccAddress
	// access to account state
	accountKeeper types.AccountKeeper
	// update balance and accounting operations with coins
	bankKeeper types.BankKeeper
	// access historical headers for EVM state transition execution
	stakingKeeper types.StakingKeeper
	// fetch EIP1559 base fee and parameters
	// feeMarketKeeper types.FeeMarketKeeper

	// chain ID number obtained from the context's chain id
	eip155ChainID *big.Int

	// EVM Tracer
	evmTracer *cosmostracing.Hooks

	// EVM Hooks for tx post-processing
	hooks types.EvmHooks

	// Legacy subspace
	ss                paramstypes.Subspace
	customContractFns []CustomContractFn
}

// NewKeeper generates new evm module keeper
func NewKeeper(
	cdc codec.Codec,
	storeKey, objectKey storetypes.StoreKey,
	authority sdk.AccAddress,
	ak types.AccountKeeper,
	bankKeeper types.BankKeeper,
	sk types.StakingKeeper,
	// fmk types.FeeMarketKeeper,
	ss paramstypes.Subspace,
	customContractFns []CustomContractFn,
) *Keeper {
	// ensure evm module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	// ensure the authority account is correct
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	// NOTE: we pass in the parameter space to the CommitStateDB in order to use custom denominations for the EVM operations
	return &Keeper{
		cdc:               cdc,
		authority:         authority,
		accountKeeper:     ak,
		bankKeeper:        bankKeeper,
		stakingKeeper:     sk,
		// feeMarketKeeper:   fmk,
		storeKey:          storeKey,
		objectKey:         objectKey,
		ss:                ss,
		customContractFns: customContractFns,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// WithChainID sets the chain id to the local variable in the keeper
func (k *Keeper) WithChainID(ctx sdk.Context) {
	chainID, err := ethermint.ParseChainID(ctx.ChainID())
	if err != nil {
		panic(err)
	}

	if k.eip155ChainID != nil && k.eip155ChainID.Cmp(chainID) != 0 {
		panic("chain id already set")
	}

	k.eip155ChainID = chainID
}

// ChainID returns the EIP155 chain ID for the EVM context
func (k Keeper) ChainID() *big.Int {
	return k.eip155ChainID
}

func (k *Keeper) InitChainer(ctx sdk.Context) {
	if tracer := cosmostracing.GetTracingHooks(ctx); tracer != nil && tracer.OnBlockchainInit != nil {
		tracer.OnBlockchainInit(types.DefaultChainConfig().EthereumConfig(k.ChainID()))
	}
}

// ----------------------------------------------------------------------------
// Block Bloom
// Required by Web3 API.
// ----------------------------------------------------------------------------

// EmitBlockBloomEvent emit block bloom events
func (k Keeper) EmitBlockBloomEvent(ctx sdk.Context, bloom []byte) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBlockBloom,
			sdk.NewAttribute(types.AttributeKeyEthereumBloom, string(bloom)),
		),
	)
}

// GetAuthority returns the x/evm module authority address
func (k Keeper) GetAuthority() sdk.AccAddress {
	return k.authority
}

// ----------------------------------------------------------------------------
// Storage
// ----------------------------------------------------------------------------

// GetAccountStorage return state storage associated with an account
func (k Keeper) GetAccountStorage(ctx sdk.Context, address common.Address) types.Storage {
	storage := types.Storage{}

	k.ForEachStorage(ctx, address, func(key, value common.Hash) bool {
		storage = append(storage, types.NewState(key, value))
		return true
	})

	return storage
}

// ----------------------------------------------------------------------------
// Account
// ----------------------------------------------------------------------------

// SetHooks sets the hooks for the EVM module
// It should be called only once during initialization, it panic if called more than once.
func (k *Keeper) SetHooks(eh types.EvmHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set evm hooks twice")
	}

	k.hooks = eh
	return k
}

// PostTxProcessing delegate the call to the hooks. If no hook has been registered, this function returns with a `nil` error
func (k *Keeper) PostTxProcessing(ctx sdk.Context, msg *core.Message, receipt *ethtypes.Receipt) error {
	if k.hooks == nil {
		return nil
	}
	return k.hooks.PostTxProcessing(ctx, msg, receipt)
}

// SetTracer should only be called during initialization
func (k *Keeper) SetTracer(tracer *cosmostracing.Hooks) {
	k.evmTracer = tracer
}

// GetAccount load nonce and codehash without balance,
// more efficient in cases where balance is not needed.
func (k *Keeper) GetAccount(ctx sdk.Context, addr common.Address) *statedb.Account {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	codeHash := types.EmptyCodeHash
	ethAcct, ok := acct.(ethermint.EthAccountI)
	if ok {
		codeHash = ethAcct.GetCodeHash().Bytes()
	}

	return &statedb.Account{
		Nonce:    acct.GetSequence(),
		CodeHash: codeHash,
	}
}

// GetAccountOrEmpty returns empty account if not exist, returns error if it's not `EthAccount`
func (k *Keeper) GetAccountOrEmpty(ctx sdk.Context, addr common.Address) statedb.Account {
	acct := k.GetAccount(ctx, addr)
	if acct != nil {
		return *acct
	}

	// empty account
	return *statedb.NewEmptyAccount()
}

// GetNonce returns the sequence number of an account, returns 0 if not exists.
func (k *Keeper) GetNonce(ctx sdk.Context, addr common.Address) uint64 {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return 0
	}

	return acct.GetSequence()
}

// GetEVMDenomBalance returns the balance of evm denom
func (k *Keeper) GetEVMDenomBalance(ctx sdk.Context, addr common.Address) *big.Int {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	evmParams := k.GetParams(ctx)
	evmDenom := evmParams.GetEvmDenom()
	// if node is pruned, params is empty. Return invalid value
	if evmDenom == "" {
		return big.NewInt(-1)
	}
	return k.GetBalance(ctx, cosmosAddr, evmDenom)
}

// GetBalance load account's balance of specified denom
func (k *Keeper) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) *big.Int {
	return k.bankKeeper.GetBalance(ctx, addr, denom).Amount.BigInt()
}

// GetBaseFee returns current base fee, return values:
// - `nil`: london hardfork not enabled.
// - `0`: london hardfork enabled but feemarket is not enabled.
// - `n`: both london hardfork and feemarket are enabled.
func (k Keeper) GetBaseFee(ctx sdk.Context, ethCfg *params.ChainConfig) *big.Int {
	return k.getBaseFee(ctx, types.IsLondon(ethCfg, ctx.BlockHeight()))
}

func (k Keeper) getBaseFee(ctx sdk.Context, london bool) *big.Int {
	if !london {
		return nil
	}
	// baseFee := k.feeMarketKeeper.GetBaseFee(ctx)
	// if baseFee == nil {
	// 	// return 0 if feemarket not enabled.
	// 	baseFee = big.NewInt(0)
	// }
	//baseFee = big.NewInt(0)
	//return baseFee
	return big.NewInt(0)
}

// GetTransientGasUsed returns the gas used by current cosmos tx.
func (k Keeper) GetTransientGasUsed(ctx sdk.Context) uint64 {
	store := ctx.ObjectStore(k.objectKey)
	v := store.Get(types.ObjectGasUsedKey(ctx.TxIndex()))
	if v == nil {
		return 0
	}
	return v.(uint64)
}

// SetTransientGasUsed sets the gas used by current cosmos tx.
func (k Keeper) SetTransientGasUsed(ctx sdk.Context, gasUsed uint64) {
	store := ctx.ObjectStore(k.objectKey)
	store.Set(types.ObjectGasUsedKey(ctx.TxIndex()), gasUsed)
}

// AddTransientGasUsed accumulate gas used by each eth msgs included in current cosmos tx.
func (k Keeper) AddTransientGasUsed(ctx sdk.Context, gasUsed uint64) (uint64, error) {
	result := k.GetTransientGasUsed(ctx) + gasUsed
	if result < gasUsed {
		return 0, errorsmod.Wrap(types.ErrGasOverflow, "transient gas used")
	}
	k.SetTransientGasUsed(ctx, result)
	return result, nil
}
