package keeper

import (
	"fmt"
	"math/big"
	"os"

	"helios-core/helios-chain/x/evm/core/vm"
	"helios-core/helios-chain/x/evm/wrappers"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"helios-core/helios-chain/x/evm/statedb"
	"helios-core/helios-chain/x/evm/types"
)

// Keeper grants access to the EVM module state and implements the go-ethereum StateDB interface.
type Keeper struct {
	// Protobuf codec
	cdc codec.BinaryCodec
	// Store key required for the EVM Prefix KVStore. It is required by:
	// - storing account's Storage State
	// - storing account's Code
	// - storing transaction Logs
	// - storing Bloom filters by block height. Needed for the Web3 API.
	storeKey storetypes.StoreKey

	// key to access the transient store, which is reset on every block during Commit
	transientKey storetypes.StoreKey

	// the address capable of executing a MsgUpdateParams message. Typically, this should be the x/gov module account.
	authority sdk.AccAddress

	// access to account state
	accountKeeper types.AccountKeeper

	// bankWrapper is used to convert the Cosmos SDK coin used in the EVM to the
	// proper decimal representation.
	bankWrapper types.BankWrapper

	// access historical headers for EVM state transition execution
	stakingKeeper types.StakingKeeper
	// fetch EIP1559 base fee and parameters
	feeMarketWrapper *wrappers.FeeMarketWrapper
	// erc20Keeper interface needed to instantiate erc20 precompiles
	erc20Keeper types.Erc20Keeper

	// Tracer used to collect execution traces from the EVM transaction execution
	tracer string

	// Legacy subspace
	ss paramstypes.Subspace

	// precompiles defines the map of all available precompiled smart contracts.
	// Some of these precompiled contracts might not be active depending on the EVM
	// parameters.
	precompiles map[common.Address]vm.PrecompiledContract

	// EVM Hooks for tx post-processing
	hooks types.EvmHooks
}

// NewKeeper generates new evm module keeper
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey, transientKey storetypes.StoreKey,
	authority sdk.AccAddress,
	ak types.AccountKeeper,
	bankKeeper types.BankKeeper,
	sk types.StakingKeeper,
	fmk types.FeeMarketKeeper,
	erc20Keeper types.Erc20Keeper,
	tracer string,
	ss paramstypes.Subspace,
) *Keeper {
	// ensure evm module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic("the EVM module account has not been set")
	}

	// ensure the authority account is correct
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	bankWrapper := wrappers.NewBankWrapper(bankKeeper)
	feeMarketWrapper := wrappers.NewFeeMarketWrapper(fmk)

	// NOTE: we pass in the parameter space to the CommitStateDB in order to use custom denominations for the EVM operations
	return &Keeper{
		cdc:              cdc,
		authority:        authority,
		accountKeeper:    ak,
		bankWrapper:      bankWrapper,
		stakingKeeper:    sk,
		feeMarketWrapper: feeMarketWrapper,
		storeKey:         storeKey,
		transientKey:     transientKey,
		tracer:           tracer,
		erc20Keeper:      erc20Keeper,
		ss:               ss,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// ----------------------------------------------------------------------------
// Block Bloom
// Required by Web3 API.
// ----------------------------------------------------------------------------

// EmitBlockBloomEvent emit block bloom events
func (k Keeper) EmitBlockBloomEvent(ctx sdk.Context, bloom ethtypes.Bloom) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeBlockBloom,
			sdk.NewAttribute(types.AttributeKeyEthereumBloom, string(bloom.Bytes())),
		),
	)
}

// GetAuthority returns the x/evm module authority address
func (k Keeper) GetAuthority() sdk.AccAddress {
	return k.authority
}

// GetBlockBloomTransient returns bloom bytes for the current block height
func (k Keeper) GetBlockBloomTransient(ctx sdk.Context) *big.Int {
	store := prefix.NewStore(ctx.TransientStore(k.transientKey), types.KeyPrefixTransientBloom)
	heightBz := sdk.Uint64ToBigEndian(uint64(ctx.BlockHeight())) //nolint:gosec // G115
	bz := store.Get(heightBz)
	if len(bz) == 0 {
		return big.NewInt(0)
	}

	return new(big.Int).SetBytes(bz)
}

// SetBlockBloomTransient sets the given bloom bytes to the transient store. This value is reset on
// every block.
func (k Keeper) SetBlockBloomTransient(ctx sdk.Context, bloom *big.Int) {
	store := prefix.NewStore(ctx.TransientStore(k.transientKey), types.KeyPrefixTransientBloom)
	heightBz := sdk.Uint64ToBigEndian(uint64(ctx.BlockHeight())) //nolint:gosec // G115
	store.Set(heightBz, bloom.Bytes())
}

// ----------------------------------------------------------------------------
// Tx
// ----------------------------------------------------------------------------

// SetTxIndexTransient set the index of processing transaction
func (k Keeper) SetTxIndexTransient(ctx sdk.Context, index uint64) {
	store := ctx.TransientStore(k.transientKey)
	store.Set(types.KeyPrefixTransientTxIndex, sdk.Uint64ToBigEndian(index))
}

// GetTxIndexTransient returns EVM transaction index on the current block.
func (k Keeper) GetTxIndexTransient(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	return sdk.BigEndianToUint64(store.Get(types.KeyPrefixTransientTxIndex))
}

// ----------------------------------------------------------------------------
// Log
// ----------------------------------------------------------------------------

// GetLogSizeTransient returns EVM log index on the current block.
func (k Keeper) GetLogSizeTransient(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	return sdk.BigEndianToUint64(store.Get(types.KeyPrefixTransientLogSize))
}

// SetLogSizeTransient fetches the current EVM log index from the transient store, increases its
// value by one and then sets the new index back to the transient store.
func (k Keeper) SetLogSizeTransient(ctx sdk.Context, logSize uint64) {
	store := ctx.TransientStore(k.transientKey)
	store.Set(types.KeyPrefixTransientLogSize, sdk.Uint64ToBigEndian(logSize))
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

// Tracer return a default vm.Tracer based on current keeper state
func (k Keeper) Tracer(ctx sdk.Context, msg core.Message, ethCfg *params.ChainConfig) vm.EVMLogger {
	return types.NewTracer(k.tracer, msg, ethCfg, ctx.BlockHeight())
}

// GetAccountWithoutBalance load nonce and codehash without balance,
// more efficient in cases where balance is not needed.
func (k *Keeper) GetAccountWithoutBalance(ctx sdk.Context, addr common.Address) *statedb.Account {
	cosmosAddr := sdk.AccAddress(addr.Bytes())
	acct := k.accountKeeper.GetAccount(ctx, cosmosAddr)
	if acct == nil {
		return nil
	}

	codeHashBz := k.GetCodeHash(ctx, addr).Bytes()

	return &statedb.Account{
		Nonce:    acct.GetSequence(),
		CodeHash: codeHashBz,
	}
}

// GetAccountOrEmpty returns empty account if not exist.
func (k *Keeper) GetAccountOrEmpty(ctx sdk.Context, addr common.Address) statedb.Account {
	acct := k.GetAccount(ctx, addr)
	if acct != nil {
		return *acct
	}

	// empty account
	return statedb.Account{
		Balance:  new(big.Int),
		CodeHash: types.EmptyCodeHash,
	}
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

// GetBalance load account's balance of gas token.
func (k *Keeper) GetBalance(ctx sdk.Context, addr common.Address) *big.Int {
	cosmosAddr := sdk.AccAddress(addr.Bytes())

	// Get the balance via bank wrapper to convert it to 18 decimals if needed.
	coin := k.bankWrapper.GetBalance(ctx, cosmosAddr, types.GetEVMCoinDenom())

	return coin.Amount.BigInt()
}

// GetBaseFee returns current base fee, return values:
// - `nil`: london hardfork not enabled.
// - `0`: london hardfork enabled but feemarket is not enabled.
// - `n`: both london hardfork and feemarket are enabled.
func (k Keeper) GetBaseFee(ctx sdk.Context) *big.Int {
	ethCfg := types.GetEthChainConfig()
	if !types.IsLondon(ethCfg, ctx.BlockHeight()) {
		return nil
	}
	baseFee := k.feeMarketWrapper.GetBaseFee(ctx)
	if baseFee == nil {
		// return 0 if feemarket not enabled.
		baseFee = big.NewInt(0)
	}
	return baseFee
}

// GetMinGasMultiplier returns the MinGasMultiplier param from the fee market module
func (k Keeper) GetMinGasMultiplier(ctx sdk.Context) math.LegacyDec {
	return k.feeMarketWrapper.GetParams(ctx).MinGasMultiplier
}

// GetMinGasPrice returns the MinGasPrice param from the fee market module
// adapted according to the evm denom decimals
func (k Keeper) GetMinGasPrice(ctx sdk.Context) math.LegacyDec {
	return k.feeMarketWrapper.GetParams(ctx).MinGasPrice
}

// ResetTransientGasUsed reset gas used to prepare for execution of current cosmos tx, called in ante handler.
func (k Keeper) ResetTransientGasUsed(ctx sdk.Context) {
	store := ctx.TransientStore(k.transientKey)
	store.Delete(types.KeyPrefixTransientGasUsed)
}

// GetTransientGasUsed returns the gas used by current cosmos tx.
func (k Keeper) GetTransientGasUsed(ctx sdk.Context) uint64 {
	store := ctx.TransientStore(k.transientKey)
	return sdk.BigEndianToUint64(store.Get(types.KeyPrefixTransientGasUsed))
}

// SetTransientGasUsed sets the gas used by current cosmos tx.
func (k Keeper) SetTransientGasUsed(ctx sdk.Context, gasUsed uint64) {
	store := ctx.TransientStore(k.transientKey)
	bz := sdk.Uint64ToBigEndian(gasUsed)
	store.Set(types.KeyPrefixTransientGasUsed, bz)
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

func (k *Keeper) SetErc20Keeper(erc20Keeper types.Erc20Keeper) {
	k.erc20Keeper = erc20Keeper
}

// SetHooks sets the hooks for the EVM module
// It should be called only once during initialization, it panic if called more than once.
func (k *Keeper) SetHooks(eh types.EvmHooks) *Keeper {
	if k.hooks != nil {
		panic("cannot set evm hooks twice")
	}

	k.hooks = eh
	return k
}

// CleanHooks resets the hooks for the EVM module
// NOTE: Should only be used for testing purposes
func (k *Keeper) CleanHooks() *Keeper {
	k.hooks = nil
	return k
}

// PostTxProcessing delegate the call to the hooks. If no hook has been registered, this function returns with a `nil` error
func (k *Keeper) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	logFile, _ := os.OpenFile("/tmp/helios-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	fmt.Fprintf(logFile, "======> EVM Keeper: PostTxProcessing called with msg.To: %v\n", msg.To())
	fmt.Fprintf(logFile, "======> EVM Keeper: Hooks type: %T\n", k.hooks)
	if k.hooks == nil {
		fmt.Fprintf(logFile, "======> EVM Keeper: No hooks registered\n")
		return nil
	}
	fmt.Fprintf(logFile, "======> EVM Keeper: Calling hooks.PostTxProcessing with gas used: %d\n", receipt.GasUsed)
	return k.hooks.PostTxProcessing(ctx, msg, receipt)
}

// GetTotalTransactionCount returns the total number of transactions in the blockchain
func (k Keeper) GetTotalTransactionCount(ctx sdk.Context) uint64 {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTotalTxCount)
	bz := store.Get([]byte("total_tx_count"))
	if bz == nil {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

// SetTotalTransactionCount sets the total number of transactions in the blockchain
func (k Keeper) SetTotalTransactionCount(ctx sdk.Context, count uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTotalTxCount)
	store.Set([]byte("total_tx_count"), sdk.Uint64ToBigEndian(count))
}

func (k Keeper) IncrementTotalTransactionCount(ctx sdk.Context, increment uint64) error {
	if increment == 0 {
		return nil
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTotalTxCount)
	key := []byte("total_tx_count")

	bz := store.Get(key)
	var currentCount uint64 = 0
	if bz != nil {
		currentCount = sdk.BigEndianToUint64(bz)
	}

	newCount := currentCount + increment
	store.Set(key, sdk.Uint64ToBigEndian(newCount))

	return nil
}

func (k Keeper) UpdateTransactionCountForBlock(ctx sdk.Context) {
	blockHeight := ctx.BlockHeight()

	if blockHeight <= 0 {
		return
	}

	txCount := k.GetTxIndexTransient(ctx)

	if txCount > 0 {
		if err := k.IncrementTotalTransactionCount(ctx, txCount); err != nil {
			k.Logger(ctx).Error("Failed to increment transaction count",
				"error", err,
				"block_height", blockHeight,
				"tx_count", txCount)
		}
	}
}
