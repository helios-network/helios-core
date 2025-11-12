package keeper

import (
	"fmt"
	gomath "math"
	"math/big"
	"sort"
	"time"

	cmn "helios-core/helios-chain/precompiles/common"
	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	erc20keeper "helios-core/helios-chain/x/erc20/keeper"
	logoskeeper "helios-core/helios-chain/x/logos/keeper"

	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"

	"cosmossdk.io/errors"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/Helios-Chain-Labs/metrics"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/hyperion/types"
)

// Keeper maintains the link to storage and exposes getter/setter methods for the various parts of the state machine
type Keeper struct {
	cdc      codec.Codec         // The wire codec for binary encoding/decoding.
	storeKey storetypes.StoreKey // Unexposed key to access store from sdk.Context
	memKey   storetypes.StoreKey // Unexposed key to access memstore from sdk.Context

	StakingKeeper  types.StakingKeeper
	bankKeeper     types.BankKeeper
	DistKeeper     distrkeeper.Keeper
	SlashingKeeper types.SlashingKeeper
	erc20Keeper    erc20keeper.Keeper
	logosKeeper    logoskeeper.Keeper
	chronosKeeper  chronoskeeper.Keeper

	AttestationHandler interface {
		Handle(sdk.Context, types.EthereumClaim, *types.Attestation) error
	}

	svcTags  metrics.Tags
	grpcTags metrics.Tags

	// address authorized to execute MsgUpdateParams. Default: gov module
	authority     string
	accountKeeper keeper.AccountKeeper

	txDecoder sdk.TxDecoder

	// CometBFT TimeoutCommit in milliseconds for fallback when AverageBlockTime is not set
	timeoutCommit uint64
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// NewKeeper returns a new instance of the hyperion keeper
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	memKey storetypes.StoreKey,
	stakingKeeper types.StakingKeeper,
	bankKeeper types.BankKeeper,
	slashingKeeper types.SlashingKeeper,
	distKeeper distrkeeper.Keeper,
	authority string,
	accountKeeper keeper.AccountKeeper,
	erc20Keeper erc20keeper.Keeper,
	logosKeeper logoskeeper.Keeper,
	chronosKeeper chronoskeeper.Keeper,
) Keeper {

	txConfig, err := authtx.NewTxConfigWithOptions(cdc, authtx.ConfigOptions{})
	if err != nil {
		panic("failed to update app tx config: " + err.Error())
	}

	k := Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
		memKey:         memKey,
		StakingKeeper:  stakingKeeper,
		bankKeeper:     bankKeeper,
		DistKeeper:     distKeeper,
		SlashingKeeper: slashingKeeper,
		authority:      authority,
		svcTags: metrics.Tags{
			"svc": "hyperion_k",
		},
		grpcTags: metrics.Tags{
			"svc": "hyperion_grpc",
		},
		accountKeeper: accountKeeper,
		erc20Keeper:   erc20Keeper,
		logosKeeper:   logosKeeper,
		chronosKeeper: chronosKeeper,
		txDecoder:     txConfig.TxDecoder(),
	}

	k.AttestationHandler = NewAttestationHandler(bankKeeper, k)

	return k
}

func (k *Keeper) GetAuthority() string {
	return k.authority
}

func (k *Keeper) Cdc() codec.Codec {
	return k.cdc
}

// GetTimeoutCommit returns the current TimeoutCommit value in milliseconds
func (k *Keeper) GetTimeoutCommit() uint64 {
	return k.timeoutCommit
}

// SetTimeoutCommit updates the TimeoutCommit value
func (k *Keeper) SetTimeoutCommit(timeoutCommit time.Duration) {
	k.timeoutCommit = uint64(timeoutCommit.Milliseconds())
}

/////////////////////////////
//     VALSET REQUESTS     //
/////////////////////////////

// SetValsetRequest returns a new instance of the Hyperion BridgeValidatorSet
// i.e. {"nonce": 1, "memebers": [{"eth_addr": "foo", "power": 11223}]}
func (k *Keeper) SetValsetRequest(ctx sdk.Context, hyperionId uint64, offsetValsetNonce uint64) *types.Valset {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	valset := k.GetCurrentValset(ctx, hyperionId)

	// If none of the bonded validators has registered eth key, then valset.Members = 0.
	if len(valset.Members) == 0 {
		return nil
	}

	k.StoreValset(ctx, valset)
	// Store the checkpoint as a legit past valset
	checkpoint := valset.GetCheckpoint(hyperionId)
	k.SetPastEthSignatureCheckpoint(ctx, hyperionId, checkpoint)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventValsetUpdateRequest{
		HyperionId:    hyperionId,
		ValsetNonce:   valset.Nonce + offsetValsetNonce,
		ValsetHeight:  valset.Height,
		ValsetMembers: valset.Members,
		RewardAmount:  valset.RewardAmount,
		RewardToken:   valset.RewardToken,
	})

	return valset
}

// StoreValset is for storing a valiator set at a given height
func (k *Keeper) StoreValset(ctx sdk.Context, valset *types.Valset) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	valset.Height = uint64(ctx.BlockHeight())
	store.Set(types.GetValsetKey(valset.HyperionId, valset.Nonce), k.cdc.MustMarshal(valset))
	k.SetLatestValsetNonce(ctx, valset.HyperionId, valset.Nonce)
}

// SetLatestValsetNonce sets the latest valset nonce
func (k *Keeper) SetLatestValsetNonce(ctx sdk.Context, hyperionId uint64, nonce uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetLatestValsetKey(hyperionId), types.UInt64Bytes(nonce))
}

// StoreValsetUnsafe is for storing a valiator set at a given height
func (k *Keeper) StoreValsetUnsafe(ctx sdk.Context, valset *types.Valset) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetValsetKey(valset.HyperionId, valset.Nonce), k.cdc.MustMarshal(valset))
	k.SetLatestValsetNonce(ctx, valset.HyperionId, valset.Nonce)
}

// HasValsetRequest returns true if a valset defined by a nonce exists
func (k *Keeper) HasValsetRequest(ctx sdk.Context, hyperionId uint64, nonce uint64) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetValsetKey(hyperionId, nonce))
}

// DeleteValset deletes the valset at a given nonce from state
func (k *Keeper) DeleteValset(ctx sdk.Context, hyperionId uint64, nonce uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	ctx.KVStore(k.storeKey).Delete(types.GetValsetKey(hyperionId, nonce))
}

func (k *Keeper) CleanValsets(ctx sdk.Context, hyperionId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.ValsetRequestKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}

// GetLatestValsetNonce returns the latest valset nonce
func (k *Keeper) GetLatestValsetNonce(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLatestValsetKey(hyperionId))

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// GetValset returns a valset by nonce
func (k *Keeper) GetValset(ctx sdk.Context, hyperionId uint64, nonce uint64) *types.Valset {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetValsetKey(hyperionId, nonce))
	if bz == nil {
		return nil
	}

	var valset types.Valset
	k.cdc.MustUnmarshal(bz, &valset)

	return &valset
}

// IterateValsets retruns all valsetRequests
func (k *Keeper) IterateValsets(ctx sdk.Context, cb func(key []byte, val *types.Valset) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetRequestKey)
	iter := prefixStore.ReverseIterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var valset types.Valset
		k.cdc.MustUnmarshal(iter.Value(), &valset)
		// cb returns true to stop early
		if cb(iter.Key(), &valset) {
			break
		}
	}
}

// GetValsets returns all the validator sets in state
func (k *Keeper) GetValsets(ctx sdk.Context, hyperionId uint64) (out []*types.Valset) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.IterateValsets(ctx, func(_ []byte, val *types.Valset) bool {
		if val.HyperionId == hyperionId {
			out = append(out, val)
		}
		return false
	})

	sort.Sort(types.Valsets(out))

	return
}

// GetLatestValset returns the latest validator set in state
func (k *Keeper) GetLatestValset(ctx sdk.Context, hyperionId uint64) (out *types.Valset) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	latestValsetNonce := k.GetLatestValsetNonce(ctx, hyperionId)
	out = k.GetValset(ctx, hyperionId, latestValsetNonce)

	return
}

// setLastSlashedValsetNonce sets the latest slashed valset nonce
func (k *Keeper) SetLastSlashedValsetNonce(ctx sdk.Context, hyperionId uint64, nonce uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetLastSlashedValsetNonceKey(hyperionId), types.UInt64Bytes(nonce))
}

// GetLastSlashedValsetNonce returns the latest slashed valset nonce
func (k *Keeper) GetLastSlashedValsetNonce(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLastSlashedValsetNonceKey(hyperionId))

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// SetLastUnbondingBlockHeight sets the last unbonding block height
func (k *Keeper) SetLastUnbondingBlockHeight(ctx sdk.Context, unbondingBlockHeight uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetLastUnbondingBlockHeightKey(), types.UInt64Bytes(unbondingBlockHeight))
}

// GetLastUnbondingBlockHeight returns the last unbonding block height
func (k *Keeper) GetLastUnbondingBlockHeight(ctx sdk.Context) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLastUnbondingBlockHeightKey())

	if len(bytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(bytes)
}

// GetUnslashedValsets returns all the unslashed validator sets in state
func (k *Keeper) GetUnslashedValsets(ctx sdk.Context, hyperionId uint64, maxHeight uint64) (out []*types.Valset) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	lastSlashedValsetNonce := k.GetLastSlashedValsetNonce(ctx, hyperionId)

	k.IterateValsetBySlashedValsetNonce(ctx, lastSlashedValsetNonce, hyperionId, maxHeight, func(_ []byte, valset *types.Valset) bool {
		if valset.Nonce > lastSlashedValsetNonce {
			out = append(out, valset)
		}
		return false
	})

	return
}

// IterateValsetBySlashedValsetNonce iterates through all valset by last slashed valset nonce in ASC order
func (k *Keeper) IterateValsetBySlashedValsetNonce(
	ctx sdk.Context,
	lastSlashedValsetNonce uint64,
	hyperionId uint64,
	maxHeight uint64,
	cb func(k []byte, v *types.Valset) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.ValsetRequestKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := prefixStore.Iterator(types.UInt64Bytes(lastSlashedValsetNonce), types.UInt64Bytes(maxHeight))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		valset := types.Valset{}
		k.cdc.MustUnmarshal(iter.Value(), &valset)

		if cb(iter.Key(), &valset) {
			break
		}
	}
}

/////////////////////////////
//     VALSET CONFIRMS     //
/////////////////////////////

// GetValsetConfirm returns a valset confirmation by a nonce and validator address
func (k *Keeper) GetValsetConfirm(ctx sdk.Context, hyperionId uint64, nonce uint64, validator sdk.AccAddress) *types.MsgValsetConfirm {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	entity := store.Get(types.GetValsetConfirmKey(hyperionId, nonce, validator))
	if entity == nil {
		return nil
	}

	valset := types.MsgValsetConfirm{}
	k.cdc.MustUnmarshal(entity, &valset)

	return &valset
}

// SetValsetConfirm sets a valset confirmation
func (k *Keeper) SetValsetConfirm(ctx sdk.Context, valset *types.MsgValsetConfirm) []byte {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	addr, err := sdk.AccAddressFromBech32(valset.Orchestrator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic(err)
	}

	key := types.GetValsetConfirmKey(valset.HyperionId, valset.Nonce, addr)
	store.Set(key, k.cdc.MustMarshal(valset))

	return key
}

func (k *Keeper) CleanValsetConfirms(ctx sdk.Context, hyperionId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), sdk.Uint64ToBigEndian(hyperionId))
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}

// GetValsetConfirms returns all validator set confirmations by nonce
func (k *Keeper) GetValsetConfirms(ctx sdk.Context, hyperionId uint64, nonce uint64) (valsetConfirms []*types.MsgValsetConfirm) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetConfirmKey)
	start, end := PrefixRange(types.GetValsetConfirmPrefixKey(hyperionId, nonce))
	iterator := prefixStore.Iterator(start, end)

	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valset := types.MsgValsetConfirm{}

		k.cdc.MustUnmarshal(iterator.Value(), &valset)
		valsetConfirms = append(valsetConfirms, &valset)
	}

	return valsetConfirms
}

// IterateValsetConfirmByNonce iterates through all valset confirms by validator set nonce in ASC order
func (k *Keeper) IterateValsetConfirmByNonce(ctx sdk.Context, hyperionId uint64, nonce uint64, cb func(k []byte, v *types.MsgValsetConfirm) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetConfirmKey)
	start, end := PrefixRange(types.GetValsetConfirmPrefixKey(hyperionId, nonce))
	iter := prefixStore.Iterator(start, end)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		valset := types.MsgValsetConfirm{}
		k.cdc.MustUnmarshal(iter.Value(), &valset)

		if cb(iter.Key(), &valset) {
			break
		}
	}
}

/////////////////////////////
//      BATCH CONFIRMS     //
/////////////////////////////

// GetBatchConfirm returns a batch confirmation given its nonce, the token contract, and a validator address
func (k *Keeper) GetBatchConfirm(ctx sdk.Context, hyperionId uint64, nonce uint64, tokenContract common.Address, validator sdk.AccAddress) *types.MsgConfirmBatch {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	entity := store.Get(types.GetBatchConfirmKey(hyperionId, tokenContract, nonce, validator))
	if entity == nil {
		return nil
	}

	batch := types.MsgConfirmBatch{}
	k.cdc.MustUnmarshal(entity, &batch)

	return &batch
}

// SetBatchConfirm sets a batch confirmation by a validator
func (k *Keeper) SetBatchConfirm(ctx sdk.Context, batch *types.MsgConfirmBatch) []byte {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// convert eth signer to hex string lol
	batch.EthSigner = common.HexToAddress(batch.EthSigner).Hex()
	tokenContract := common.HexToAddress(batch.TokenContract)
	store := ctx.KVStore(k.storeKey)

	acc, err := sdk.AccAddressFromBech32(batch.Orchestrator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic(err)
	}

	key := types.GetBatchConfirmKey(batch.HyperionId, tokenContract, batch.Nonce, acc)
	store.Set(key, k.cdc.MustMarshal(batch))

	return key
}

func (k *Keeper) CleanBatchConfirms(ctx sdk.Context, hyperionId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.BatchConfirmKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}

// IterateBatchConfirmByNonceAndTokenContract iterates through all batch confirmations
func (k *Keeper) IterateBatchConfirmByNonceAndTokenContract(
	ctx sdk.Context,
	hyperionId uint64,
	nonce uint64,
	tokenContract common.Address,
	cb func(k []byte, v *types.MsgConfirmBatch) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.BatchConfirmKey)

	batchPrefix := make([]byte, 0, 8+types.ETHContractAddressLen+8)
	batchPrefix = append(batchPrefix, types.UInt64Bytes(hyperionId)...)
	batchPrefix = append(batchPrefix, tokenContract.Bytes()...)
	batchPrefix = append(batchPrefix, types.UInt64Bytes(nonce)...)

	iter := prefixStore.Iterator(PrefixRange(batchPrefix))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		confirm := types.MsgConfirmBatch{}
		k.cdc.MustUnmarshal(iter.Value(), &confirm)

		if cb(iter.Key(), &confirm) {
			break
		}
	}
}

// GetBatchConfirmByNonceAndTokenContract returns the batch confirms
func (k *Keeper) GetBatchConfirmByNonceAndTokenContract(ctx sdk.Context, hyperionId uint64, nonce uint64, tokenContract common.Address) (out []*types.MsgConfirmBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.IterateBatchConfirmByNonceAndTokenContract(ctx, hyperionId, nonce, tokenContract, func(_ []byte, msg *types.MsgConfirmBatch) (stop bool) {
		out = append(out, msg)
		return false
	})

	return
}

/////////////////////////////
//    ADDRESS DELEGATION   //
/////////////////////////////

// SetOrchestratorValidator sets the Orchestrator key for a given validator
func (k *Keeper) SetOrchestratorValidator(ctx sdk.Context, hyperionId uint64, val sdk.ValAddress, orch sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetOrchestratorAddressKey(hyperionId, orch), val.Bytes())
}

// GetOrchestratorValidator returns the validator key associated with an orchestrator key
func (k *Keeper) GetOrchestratorValidator(ctx sdk.Context, hyperionId uint64, orch sdk.AccAddress) (sdk.ValAddress, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetOrchestratorAddressKey(hyperionId, orch))
	if bz == nil {
		return nil, false
	}

	return sdk.ValAddress(bz), true
}

// DeleteOrchestratorValidator deletes the orchestrator validator
func (k *Keeper) DeleteOrchestratorValidator(ctx sdk.Context, hyperionId uint64, orch sdk.AccAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetOrchestratorAddressKey(hyperionId, orch))
}

/////////////////////////////
//       ETH ADDRESS       //
/////////////////////////////

// SetEthAddressForValidator sets the ethereum address for a given validator
func (k *Keeper) SetEthAddressForValidator(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress, ethAddr common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetEthAddressByValidatorKey(hyperionId, validator), ethAddr.Bytes())
	store.Set(types.GetValidatorByEthAddressKey(hyperionId, ethAddr), validator.Bytes())
}

func (k *Keeper) SetEthAddressForValidatorWithFee(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress, ethAddr common.Address, fee sdk.Coin) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetEthAddressByValidatorKey(hyperionId, validator), ethAddr.Bytes())
	store.Set(types.GetValidatorByEthAddressKey(hyperionId, ethAddr), validator.Bytes())

	feeBytes, err := fee.Marshal()
	if err != nil {
		return err
	}
	store.Set(types.GetFeeByValidatorKey(hyperionId, validator), feeBytes)
	return nil
}

func (k *Keeper) SetFeeForValidator(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress, fee sdk.Coin) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetFeeByValidatorKey(hyperionId, validator), fee.Amount.BigInt().Bytes())
	return nil
}

func (k *Keeper) GetFeeByValidator(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress) *sdk.Coin {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	if !store.Has(types.GetFeeByValidatorKey(hyperionId, validator)) {
		return nil
	}
	bz := store.Get(types.GetFeeByValidatorKey(hyperionId, validator))
	if bz == nil {
		return nil
	}

	return &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewIntFromBigInt(new(big.Int).SetBytes(bz))}
}

func (k *Keeper) DeleteFeeForValidator(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	if store.Has(types.GetFeeByValidatorKey(hyperionId, validator)) {
		store.Delete(types.GetFeeByValidatorKey(hyperionId, validator))
	}
}

func (k *Keeper) GetLowestFeeValidator(ctx sdk.Context, hyperionId uint64) (val *sdk.ValAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(PrefixRange(append(types.FeeByValidatorKey, sdk.Uint64ToBigEndian(hyperionId)...)))
	defer iter.Close()

	lowestFee := sdk.Coin{}
	lowestFee.Amount = math.NewInt(-1)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()[len(types.FeeByValidatorKey)+8:]
		fee := sdk.Coin{Denom: sdk.DefaultBondDenom}
		fee.Amount = math.NewIntFromBigInt(new(big.Int).SetBytes(iter.Value()))
		if lowestFee.Amount.Equal(math.NewInt(-1)) || fee.Amount.LT(lowestFee.Amount) {
			lowestFee = fee
			addr := sdk.ValAddress(key)
			val = &addr
		}
	}

	return val
}

func (k *Keeper) GetHighestFeeValidator(ctx sdk.Context, hyperionId uint64) (val *sdk.ValAddress) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(PrefixRange(append(types.FeeByValidatorKey, sdk.Uint64ToBigEndian(hyperionId)...)))
	defer iter.Close()

	highestFee := sdk.Coin{}
	highestFee.Amount = math.NewInt(-1)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()[len(types.FeeByValidatorKey)+8:]
		fee := sdk.Coin{Denom: sdk.DefaultBondDenom}
		fee.Amount = math.NewIntFromBigInt(new(big.Int).SetBytes(iter.Value()))
		if highestFee.Amount.Equal(math.NewInt(-1)) || fee.Amount.GT(highestFee.Amount) {
			highestFee = fee
			addr := sdk.ValAddress(key)
			val = &addr
		}
	}

	return val
}

func (k *Keeper) GetLowestsFeeValidatorsFromFee(ctx sdk.Context, hyperionId uint64, fee sdk.Coin) []*sdk.ValAddress {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	iter := store.Iterator(PrefixRange(append(types.FeeByValidatorKey, sdk.Uint64ToBigEndian(hyperionId)...)))
	defer iter.Close()

	lowestFeeValidators := make([]*sdk.ValAddress, 0)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()[len(types.FeeByValidatorKey)+8:]
		feeOfValidator := sdk.Coin{}
		feeOfValidator.Amount = math.NewIntFromBigInt(new(big.Int).SetBytes(iter.Value()))
		if feeOfValidator.Amount.LT(fee.Amount) {
			addr := sdk.ValAddress(key)
			lowestFeeValidators = append(lowestFeeValidators, &addr)
		}
	}

	return lowestFeeValidators
}

// GetEthAddressByValidator returns the eth address for a given hyperion validator
func (k *Keeper) GetEthAddressByValidator(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress) (common.Address, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetEthAddressByValidatorKey(hyperionId, validator))
	if bz == nil {
		return common.Address{}, false
	}

	return common.BytesToAddress(bz), true
}

func (k *Keeper) DeleteEthAddressForValidator(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress, ethAddr common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetEthAddressByValidatorKey(hyperionId, validator))
	store.Delete(types.GetValidatorByEthAddressKey(hyperionId, ethAddr))
}

// GetValidatorByEthAddress returns the validator for a given eth address
func (k *Keeper) GetValidatorByEthAddress(ctx sdk.Context, hyperionId uint64, ethAddr common.Address) (validator stakingtypes.Validator, found bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	valAddr := store.Get(types.GetValidatorByEthAddressKey(hyperionId, ethAddr))
	if valAddr == nil {
		return stakingtypes.Validator{}, false
	}
	validator, err := k.StakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return stakingtypes.Validator{}, false
	}

	return validator, true
}

func (k *Keeper) GetCurrentValsetTotalPower(ctx sdk.Context, hyperionId uint64) math.Int {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	validators, _ := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	// allocate enough space for all validators, but len zero, we then append
	// so that we have an array with extra capacity but the correct length depending
	// on how many validators have keys set.
	totalPower := math.ZeroInt()
	for i := range validators {
		val, _ := sdk.ValAddressFromBech32(validators[i].GetOperator())
		vp, _ := k.StakingKeeper.GetLastValidatorPower(ctx, val)
		p := uint64(vp)

		if _, found := k.GetEthAddressByValidator(ctx, hyperionId, val); found {
			totalPower = totalPower.Add(math.NewInt(int64(p)))
		}
	}

	return totalPower
}

// GetCurrentValset gets powers from the store and normalizes them
// into an integer percentage with a resolution of uint32 Max meaning
// a given validators 'Hyperion power' is computed as
// Cosmos power for that validator / total cosmos power = x / uint32 Max
// where x is the voting power on the Hyperion contract. This allows us
// to only use integer division which produces a known rounding error
// from truncation equal to the ratio of the validators
// Cosmos power / total cosmos power ratio, leaving us at uint32 Max - 1
// total voting power. This is an acceptable rounding error since floating
// point may cause consensus problems if different floating point unit
// implementations are involved.
//
// 'total cosmos power' has an edge case, if a validator has not set their
// Ethereum key they are not included in the total. If they where control
// of the bridge could be lost in the following situation.
//
// If we have 100 total power, and 100 total power joins the validator set
// the new validators hold more than 33% of the bridge power, if we generate
// and submit a valset and they don't have their eth keys set they can never
// update the validator set again and the bridge and all its' funds are lost.
// For this reason we exclude validators with unset eth keys from validator sets
func (k *Keeper) GetCurrentValset(ctx sdk.Context, hyperionId uint64) *types.Valset {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	validators, _ := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	// allocate enough space for all validators, but len zero, we then append
	// so that we have an array with extra capacity but the correct length depending
	// on how many validators have keys set.
	bridgeValidators := make([]*types.BridgeValidator, 0, len(validators))
	var totalPower uint64
	for i := range validators {
		val, _ := sdk.ValAddressFromBech32(validators[i].GetOperator())
		vp, _ := k.StakingKeeper.GetLastValidatorPower(ctx, val)
		p := uint64(vp)

		if ethAddress, found := k.GetEthAddressByValidator(ctx, hyperionId, val); found {
			bv := &types.BridgeValidator{Power: p, EthereumAddress: ethAddress.Hex()}
			bridgeValidators = append(bridgeValidators, bv)
			totalPower += p
		}
	}

	// normalize power values
	for i := range bridgeValidators {

		bridgeValidators[i].Power = math.NewUint(bridgeValidators[i].Power).MulUint64(gomath.MaxUint32).QuoUint64(totalPower).Uint64()
	}

	// get the reward from the params store
	reward := k.GetValsetReward(ctx)[hyperionId]
	var rewardToken common.Address
	var rewardAmount math.Int
	if reward.Denom == "" {
		// the case where a validator has 'no reward'. The 'no reward' value is interpreted as having a zero
		// address for the ERC20 token and a zero value for the reward amount. Since we store a coin with the
		// params, a coin with a blank denom and/or zero amount is interpreted in this way.
		rewardToken = common.Address{0x0000000000000000000000000000000000000000}
		rewardAmount = math.NewIntFromUint64(0)

	} else {
		rewardAmount = reward.Amount
		tokenAddressToDenom, exists := k.GetTokenFromDenom(ctx, hyperionId, reward.Denom)
		if !exists { // force the reward to be zero
			rewardToken = common.Address{0x0000000000000000000000000000000000000000}
			rewardAmount = math.NewIntFromUint64(0)
		} else {
			rewardToken = common.HexToAddress(tokenAddressToDenom.TokenAddress)
		}
	}
	hyperionParams := k.GetCounterpartyChainParams(ctx)[hyperionId]
	// TODO: make the nonce an incrementing one (i.e. fetch last nonce from state, increment, set here)
	return types.NewValset(hyperionId, uint64(ctx.BlockHeight())+hyperionParams.OffsetValsetNonce, uint64(ctx.BlockHeight()), bridgeValidators, rewardAmount, rewardToken)
}

func (k *Keeper) GetLastValidatorPower(ctx sdk.Context, validator common.Address) (uint64, error) {
	val := cmn.ValAddressFromHexAddress(validator)
	vp, err := k.StakingKeeper.GetLastValidatorPower(ctx, val)
	if err != nil {
		return 0, err
	}
	return uint64(vp), nil
}

/////////////////////////////
//       HELPERS           //
/////////////////////////////

func (k *Keeper) getStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

func (k *Keeper) getMemStore(ctx sdk.Context) storetypes.KVStore { // TODO: using it for storing historical status txs, should be removed in the future
	memStore := ctx.KVStore(k.memKey)
	memStoreType := memStore.GetStoreType()

	if memStoreType != storetypes.StoreTypeMemory {
		panic(fmt.Sprintf("HyperionKeeper: invalid memory store type; got %s, expected: %s", memStoreType, storetypes.StoreTypeMemory))
	}
	return ctx.KVStore(k.memKey)
}

// SendToCommunityPool handles incorrect SendToCosmos calls to the community pool, since the calls
// have already been made on Ethereum there's nothing we can do to reverse them, and we should at least
// make use of the tokens which would otherwise be lost
func (k *Keeper) SendToCommunityPool(ctx sdk.Context, coins sdk.Coins) error {
	if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, distrtypes.ModuleName, coins); err != nil {
		return errors.Wrap(err, "transfer to community pool failed")
	}
	feePool, err := k.DistKeeper.FeePool.Get(ctx)

	if err != nil {
		return err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(coins...)...)
	err = k.DistKeeper.FeePool.Set(ctx, feePool)

	return err
}

/////////////////////////////
//       PARAMS        //
/////////////////////////////

// GetParams returns the parameters from the store
func (k *Keeper) GetParams(ctx sdk.Context) *types.Params {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := store.Get(types.ParamKey)
	if bz == nil {
		return nil
	}

	params := &types.Params{}
	k.cdc.MustUnmarshal(bz, params)

	return params
}

// SetParams sets the parameters in the store
func (k *Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := k.getStore(ctx)
	bz := k.cdc.MustMarshal(params)
	store.Set(types.ParamKey, bz)
}

// GetCounterpartyChainParams returns a mapping (hyperion id => the counterparty chain params)
func (k *Keeper) GetCounterpartyChainParams(ctx sdk.Context) map[uint64]*types.CounterpartyChainParams {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]*types.CounterpartyChainParams{}
	}

	counterpartyChainParamsMap := make(map[uint64]*types.CounterpartyChainParams)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		counterpartyChainParamsMap[counterpartyChainParams.HyperionId] = counterpartyChainParams
	}

	return counterpartyChainParamsMap
}

func (k *Keeper) SetCounterpartyChainParams(ctx sdk.Context, hyperionId uint64, newCounterpartyChainParams *types.CounterpartyChainParams) {
	params := k.GetParams(ctx)
	for i, counterpartyChainParams := range params.CounterpartyChainParams {
		if counterpartyChainParams.HyperionId == hyperionId {
			params.CounterpartyChainParams[i] = newCounterpartyChainParams
			break
		}
	}
	k.SetParams(ctx, params)
}

// GetBridgeContractAddress returns a mapping (hyperion id => the bridge contract address on the counterparty chain)
func (k *Keeper) GetBridgeContractAddress(ctx sdk.Context) map[uint64]common.Address {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]common.Address{}
	}

	bridgeContractAddressMap := make(map[uint64]common.Address)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		bridgeContractAddressMap[counterpartyChainParams.HyperionId] = common.HexToAddress(counterpartyChainParams.BridgeCounterpartyAddress)
	}

	return bridgeContractAddressMap
}

// GetBridgeChainID returns a mapping (hyperion id => the chain id of the counterparty chain)
func (k *Keeper) GetBridgeChainID(ctx sdk.Context) map[uint64]uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]uint64{}
	}

	bridgeChainIdMap := make(map[uint64]uint64)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		bridgeChainIdMap[counterpartyChainParams.HyperionId] = counterpartyChainParams.BridgeChainId
	}

	return bridgeChainIdMap
}

func (k *Keeper) GetHyperionID(ctx sdk.Context) map[uint64]uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]uint64{}
	}

	hyperionIdMap := make(map[uint64]uint64)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		hyperionIdMap[counterpartyChainParams.HyperionId] = counterpartyChainParams.HyperionId
	}

	return hyperionIdMap
}

func (k *Keeper) GetValsetReward(ctx sdk.Context) map[uint64]sdk.Coin {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]sdk.Coin{}
	}

	valsetRewardMap := make(map[uint64]sdk.Coin)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		valsetRewardMap[counterpartyChainParams.HyperionId] = counterpartyChainParams.ValsetReward
	}

	return valsetRewardMap
}

func (k *Keeper) UnpackAttestationClaim(attestation *types.Attestation) (types.EthereumClaim, error) {
	var msg types.EthereumClaim

	err := k.cdc.UnpackAny(attestation.Claim, &msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		err = errors.Wrap(err, "failed to unpack EthereumClaim")
		return nil, err
	} else {
		return msg, nil
	}
}

// GetOrchestratorAddresses iterates both the EthAddress and Orchestrator address indexes to produce
// a vector of MsgSetOrchestratorAddresses entires containing all the delgate keys for state
// export / import. This may seem at first glance to be excessively complicated, why not combine
// the EthAddress and Orchestrator address indexes and simply iterate one thing? The answer is that
// even though we set the Eth and Orchestrator address in the same place we use them differently we
// always go from Orchestrator address to Validator address and from validator address to Ethereum address
// we want to keep looking up the validator address for various reasons, so a direct Orchestrator to Ethereum
// address mapping will mean having to keep two of the same data around just to provide lookups.
//
// For the time being this will serve
func (k *Keeper) GetOrchestratorAddresses(ctx sdk.Context, hyperionId uint64) []*types.MsgSetOrchestratorAddresses {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	storePrefix := make([]byte, 0, len(types.EthAddressByValidatorKey)+8)
	storePrefix = append(storePrefix, types.EthAddressByValidatorKey...)
	storePrefix = append(storePrefix, types.UInt64Bytes(hyperionId)...)

	iter := store.Iterator(PrefixRange(storePrefix))
	defer iter.Close()

	ethAddresses := make(map[string]common.Address)

	for ; iter.Valid(); iter.Next() {
		// the 'key' contains both the prefix and the value, so we need
		// to cut off the starting bytes, if you don't do this a valid
		// cosmos key will be made out of EthAddressByValidatorKey + the startin bytes
		// of the actual key
		key := iter.Key()[len(types.EthAddressByValidatorKey)+8:]
		value := iter.Value()
		ethAddress := common.BytesToAddress(value)
		validatorAccount := sdk.AccAddress(key)
		ethAddresses[validatorAccount.String()] = ethAddress
	}

	store = ctx.KVStore(k.storeKey)
	storePrefix = make([]byte, 0, len(types.KeyOrchestratorAddress)+8)
	storePrefix = append(storePrefix, types.KeyOrchestratorAddress...)
	storePrefix = append(storePrefix, types.UInt64Bytes(hyperionId)...)
	iter = store.Iterator(PrefixRange(storePrefix))
	defer iter.Close()

	orchestratorAddresses := make(map[string]sdk.AccAddress)

	for ; iter.Valid(); iter.Next() {
		key := iter.Key()[len(types.KeyOrchestratorAddress)+8:]
		value := iter.Value()
		orchestratorAccount := sdk.AccAddress(key)
		validatorAccount := sdk.AccAddress(value)
		orchestratorAddresses[validatorAccount.String()] = orchestratorAccount
	}

	result := make([]*types.MsgSetOrchestratorAddresses, 0)

	for validatorAccount, ethAddress := range ethAddresses {
		orchestratorAccount, ok := orchestratorAddresses[validatorAccount]
		if !ok {
			metrics.ReportFuncError(k.svcTags)
			panic("cannot find validator account in orchestrator addresses mapping")
		}

		result = append(result, &types.MsgSetOrchestratorAddresses{
			Sender:       validatorAccount,
			Orchestrator: orchestratorAccount.String(),
			EthAddress:   ethAddress.Hex(),
		})
	}

	// we iterated over a map, so now we have to sort to ensure the
	// output here is deterministic, eth address chosen for no particular
	// reason
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].EthAddress < result[j].EthAddress
	})

	return result
}

// DeserializeValidatorIterator returns validators from the validator iterator.
// Adding here in gravity keeper as cdc is not available inside endblocker.
func (k *Keeper) DeserializeValidatorIterator(vals []byte) stakingtypes.ValAddresses {
	validators := stakingtypes.ValAddresses{}
	k.cdc.MustUnmarshal(vals, &validators)
	return validators
}

type PrefixStart []byte
type PrefixEnd []byte

// PrefixRange turns a prefix into a (start, end) range. The start is the given prefix value and
// the end is calculated by adding 1 bit to the start value. Nil is not allowed as prefix.
//
//	Example: []byte{1, 3, 4} becomes []byte{1, 3, 5}
//			 []byte{15, 42, 255, 255} becomes []byte{15, 43, 0, 0}
//
// In case of an overflow the end is set to nil.
//
//	Example: []byte{255, 255, 255, 255} becomes nil
//
// MARK finish-batches: this is where some crazy shit happens
func PrefixRange(proposedPrefix []byte) (PrefixStart, PrefixEnd) {
	if proposedPrefix == nil {
		panic("nil key not allowed")
	}

	// special case: no prefix is whole range
	if len(proposedPrefix) == 0 {
		return nil, nil
	}

	// copy the prefix and update last byte
	end := make([]byte, len(proposedPrefix))
	copy(end, proposedPrefix)
	l := len(end) - 1
	end[l]++

	// wait, what if that overflowed?....
	for end[l] == 0 && l > 0 {
		l--
		end[l]++
	}

	// okay, funny guy, you gave us FFF, no end to this range...
	if l == 0 && end[0] == 0 {
		end = nil
	}

	return proposedPrefix, end
}

// IsOnBlacklist checks that the address is black listed.
func (k *Keeper) IsOnBlacklist(ctx sdk.Context, addr common.Address) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.getStore(ctx).Has(types.GetBlacklistStoreKey(addr))
}

// SetBlacklistAddress sets the blacklist address.
func (k *Keeper) SetBlacklistAddress(ctx sdk.Context, addr common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// set boolean indicator
	k.getStore(ctx).Set(types.GetBlacklistStoreKey(addr), []byte{})
}

// GetAllBlacklistAddresses fetches all blacklisted addresses.
func (k *Keeper) GetAllBlacklistAddresses(ctx sdk.Context) []string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	blacklistedAddresses := make([]string, 0)
	store := ctx.KVStore(k.storeKey)
	blacklistAddressStore := prefix.NewStore(store, types.BlacklistKey)

	iterator := blacklistAddressStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		blacklistAddress := common.BytesToAddress(iterator.Key())
		blacklistedAddresses = append(blacklistedAddresses, blacklistAddress.String())
	}

	return blacklistedAddresses
}

// DeleteBlacklistAddress deletes the address from blacklist.
func (k *Keeper) DeleteBlacklistAddress(ctx sdk.Context, addr common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.getStore(ctx).Delete(types.GetBlacklistStoreKey(addr))
}

// InvalidSendToChainAddress Returns true if the provided address is invalid to send to EVM chain this could be
// for one of several reasons. (1) it is invalid in general like the Zero address, (2)
// it is invalid for a subset of ERC20 addresses or (3) it is on the governance deposit/withdraw
// blacklist. (2) is not yet implemented
// Blocking some addresses is technically motivated, if any ERC20 transfers in a batch fail the entire batch
// becomes impossible to execute.
func (k *Keeper) InvalidSendToChainAddress(ctx sdk.Context, addr common.Address) bool {
	return k.IsOnBlacklist(ctx, addr) || addr == types.ZeroAddress()
}

// CreateModuleAccount creates a module account with minting and burning capabilities
func (k *Keeper) CreateModuleAccount(ctx sdk.Context) {
	baseAcc := authtypes.NewEmptyModuleAccount(types.ModuleName, authtypes.Minter, authtypes.Burner)
	moduleAcc := (k.accountKeeper.NewAccount(ctx, baseAcc)).(sdk.ModuleAccountI) // set the account number
	k.accountKeeper.SetModuleAccount(ctx, moduleAcc)
}

func (k *Keeper) SetErc20Keeper(erc20Keeper erc20keeper.Keeper) {
	k.erc20Keeper = erc20Keeper
}

func (k *Keeper) GetHyperionParamsFromChainId(ctx sdk.Context, chainId uint64) *types.CounterpartyChainParams {
	params := k.GetParams(ctx)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == chainId {
			return counterpartyChainParam
		}
	}

	return nil
}

func (k *Keeper) GetChainIdFromHyperionId(ctx sdk.Context, hyperionId uint64) uint64 {
	params := k.GetParams(ctx)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.HyperionId == hyperionId {
			return counterpartyChainParam.BridgeChainId
		}
	}

	return 0
}

func (k *Keeper) GetProjectedCurrentEthereumHeight(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	counterpartyChainParams := k.GetCounterpartyChainParams(ctx)[hyperionId]
	currentCosmosHeight := ctx.BlockHeight()
	// we store the last observed Cosmos and Ethereum heights, we do not concern ourselves if these values
	// are zero because no batch can be produced if the last Ethereum block height is not first populated by a deposit event.
	heights := k.GetLastObservedEthereumBlockHeight(ctx, hyperionId)
	if heights.CosmosBlockHeight == 0 || heights.EthereumBlockHeight == 0 {
		return 0
	}

	// Use AverageBlockTime from params, or fallback to CometBFT TimeoutCommit
	averageBlockTime := k.GetEffectiveAverageBlockTime(ctx, counterpartyChainParams)

	// we project how long it has been in milliseconds since the last Ethereum block height was observed
	projectedMillis := (uint64(currentCosmosHeight) - heights.CosmosBlockHeight) * averageBlockTime
	// we convert that projection into the current Ethereum height using the average Ethereum block time in millis
	projectedCurrentEthereumHeight := (projectedMillis / counterpartyChainParams.AverageCounterpartyBlockTime) + heights.EthereumBlockHeight

	return projectedCurrentEthereumHeight
}

// GetEffectiveAverageBlockTime returns the effective average block time to use.
// If AverageBlockTime is set in params, it uses that value.
// Otherwise, it falls back to CometBFT's TimeoutCommit converted to milliseconds.
func (k *Keeper) GetEffectiveAverageBlockTime(ctx sdk.Context, params *types.CounterpartyChainParams) uint64 {
	// Fallback to CometBFT's TimeoutCommit stored in the keeper
	return k.timeoutCommit
}

func (k *Keeper) SearchAttestationsByEthereumAddress(ctx sdk.Context, hyperionId uint64, ethereumAddress string) ([]*types.Attestation, error) {
	attestations := k.GetAttestationMapping(ctx, hyperionId) // Assuming hyperionId is known or passed as a parameter
	var matchingAttestations []*types.Attestation

	for _, attestationList := range attestations {
		for _, attestation := range attestationList {
			claim, err := k.UnpackAttestationClaim(attestation)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unpack attestation claim")
			}

			// Check if the claim contains the specified Ethereum address
			switch claim := claim.(type) {
			case *types.MsgDepositClaim:
				if claim.EthereumSender == ethereumAddress || ethereumAddress == "" {
					matchingAttestations = append(matchingAttestations, attestation)
				}
			}
		}
	}

	return matchingAttestations, nil
}

func (k *Keeper) UpdateRpcUsed(ctx sdk.Context, hyperionId uint64, rpcUsed string, heightUsed uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	counterpartyChainParams := k.GetCounterpartyChainParams(ctx)[hyperionId]
	rpcList := make([]*types.Rpc, 0)
	found := false
	for _, rpc := range counterpartyChainParams.Rpcs {
		if rpc.Url == rpcUsed {
			found = true
			// Update the rpc's reputation and last height used
			rpc.Reputation++
			rpc.LastHeightUsed = heightUsed
		}
		rpcList = append(rpcList, rpc)
	}

	// order the rpc list by last height used
	sort.Slice(rpcList, func(i, j int) bool {
		return rpcList[i].LastHeightUsed < rpcList[j].LastHeightUsed
	})
	// keep the first 100 rpc
	if len(rpcList) > 100 {
		rpcList = rpcList[:100]
	}
	// If the rpc is not found, add it to the list
	if !found {
		rpcList = append(rpcList, &types.Rpc{Url: rpcUsed, Reputation: 1, LastHeightUsed: heightUsed})
	}
	counterpartyChainParams.Rpcs = rpcList
	k.SetCounterpartyChainParams(ctx, hyperionId, counterpartyChainParams)
}

func (k *Keeper) StoreNonceObserved(ctx sdk.Context, hyperionId uint64, nonce uint64, observedHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	if ctx.BlockHeight() < 244300 {
		return
	}
	nonceObservedStore := prefix.NewStore(store, types.NonceObservedKey)
	nonceObservedStore.Set(types.GetNonceObservedKey(hyperionId, nonce), sdk.Uint64ToBigEndian(observedHeight))
}

func (k *Keeper) NonceAlreadyObserved(ctx sdk.Context, hyperionId uint64, nonce uint64) bool {
	store := ctx.KVStore(k.storeKey)
	if ctx.BlockHeight() < 244300 {
		return false
	}
	nonceObservedStore := prefix.NewStore(store, types.NonceObservedKey)
	return nonceObservedStore.Has(types.GetNonceObservedKey(hyperionId, nonce))
}

func (k *Keeper) GetNonceObserved(ctx sdk.Context, hyperionId uint64, nonce uint64) uint64 {
	store := ctx.KVStore(k.storeKey)
	if ctx.BlockHeight() < 244300 {
		return 0
	}
	nonceObservedStore := prefix.NewStore(store, types.NonceObservedKey)
	return types.UInt64FromBytes(nonceObservedStore.Get(types.GetNonceObservedKey(hyperionId, nonce)))
}

func (k *Keeper) StoreSkippedNonce(ctx sdk.Context, hyperionId uint64, nonce uint64, startHeight uint64, endHeight uint64) {
	store := ctx.KVStore(k.storeKey)
	skippedNonceStore := prefix.NewStore(store, types.SkippedNonceKey)

	skippedNonceInfo := &types.SkippedNonceInfo{
		StartHeight: startHeight,
		EndHeight:   endHeight,
	}
	skippedNonceStore.Set(types.GetSkippedNonceKey(hyperionId, nonce), k.cdc.MustMarshal(skippedNonceInfo))
}

func (k *Keeper) HasSkippedNonce(ctx sdk.Context, hyperionId uint64, nonce uint64) bool {
	store := ctx.KVStore(k.storeKey)
	skippedNonceStore := prefix.NewStore(store, types.SkippedNonceKey)
	return skippedNonceStore.Has(types.GetSkippedNonceKey(hyperionId, nonce))
}

func (k *Keeper) GetSkippedNonceInfo(ctx sdk.Context, hyperionId uint64, nonce uint64) *types.SkippedNonceInfo {
	store := ctx.KVStore(k.storeKey)
	skippedNonceStore := prefix.NewStore(store, types.SkippedNonceKey)
	skippedNonceBytes := skippedNonceStore.Get(types.GetSkippedNonceKey(hyperionId, nonce))
	if skippedNonceBytes == nil {
		return nil
	}
	var skippedNonceInfo types.SkippedNonceInfo
	k.cdc.MustUnmarshal(skippedNonceBytes, &skippedNonceInfo)
	return &skippedNonceInfo
}

func (k *Keeper) RemoveSkippedNonce(ctx sdk.Context, hyperionId uint64, nonce uint64) {
	store := ctx.KVStore(k.storeKey)
	skippedNonceStore := prefix.NewStore(store, types.SkippedNonceKey)
	skippedNonceStore.Delete(types.GetSkippedNonceKey(hyperionId, nonce))
}

func (k *Keeper) CleanSkippedTxs(ctx sdk.Context, hyperionId uint64) {
	store := ctx.KVStore(k.storeKey)
	skippedNonceStore := prefix.NewStore(store, append(types.SkippedNonceKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := skippedNonceStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		skippedNonceStore.Delete(iter.Key())
	}
}

func (k *Keeper) CleanAllSkippedTxs(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	skippedNonceStore := prefix.NewStore(store, types.SkippedNonceKey)
	iter := skippedNonceStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		fmt.Println("iter.Key(): ", iter.Key())
		fmt.Println("iter.Value(): ", iter.Value())
		skippedNonceStore.Delete(iter.Key())
	}
	fmt.Println("cleaned all skipped txs")
}

func (k *Keeper) GetAllSkippedNonces(ctx sdk.Context, hyperionId uint64) []*types.SkippedNonceFullInfo {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.SkippedNonceKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	skippedNonceList := make([]*types.SkippedNonceFullInfo, 0)
	for ; iter.Valid(); iter.Next() {
		var skippedNonceInfo types.SkippedNonceInfo
		err := k.cdc.Unmarshal(iter.Value(), &skippedNonceInfo)
		if err != nil {
			skippedNonceInfo = types.SkippedNonceInfo{}
		}
		skippedNonceFullInfo := &types.SkippedNonceFullInfo{
			StartHeight: skippedNonceInfo.StartHeight,
			EndHeight:   skippedNonceInfo.EndHeight,
			Nonce:       types.UInt64FromBytes(iter.Key()),
		}
		skippedNonceList = append(skippedNonceList, skippedNonceFullInfo)
	}
	return skippedNonceList
}

func (k *Keeper) GetAllSkippedNoncesWithHyperionId(ctx sdk.Context) []*types.SkippedNonceFullInfoWithHyperionId {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.SkippedNonceKey)
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	skippedNonceList := make([]*types.SkippedNonceFullInfoWithHyperionId, 0)
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		hyperionId := types.UInt64FromBytes(key[:8])

		var skippedNonceInfo types.SkippedNonceInfo
		err := k.cdc.Unmarshal(iter.Value(), &skippedNonceInfo)
		if err != nil {
			skippedNonceInfo = types.SkippedNonceInfo{}
		}
		skippedNonceFullInfo := &types.SkippedNonceFullInfoWithHyperionId{
			HyperionId:  hyperionId,
			StartHeight: skippedNonceInfo.StartHeight,
			EndHeight:   skippedNonceInfo.EndHeight,
			Nonce:       types.UInt64FromBytes(iter.Key()),
		}
		skippedNonceList = append(skippedNonceList, skippedNonceFullInfo)
	}
	return skippedNonceList
}

func (k *Keeper) CleanAllNonceObserved(ctx sdk.Context, hyperionId uint64) {
	store := ctx.KVStore(k.storeKey)
	nonceObservedStore := prefix.NewStore(store, types.NonceObservedKey)
	iter := nonceObservedStore.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		nonceObservedStore.Delete(iter.Key())
	}
}

func (k *Keeper) CleanAllBatchesAndTxs(ctx sdk.Context, hyperionId uint64) {
	batches := k.GetOutgoingTxBatches(ctx, hyperionId)

	for _, batch := range batches {
		err := k.CancelOutgoingTXBatch(ctx, common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId)
		if err != nil {
			ctx.Logger().Error("failed to cancel outgoing tx batch", "error", err, "block", batch.Block, "batch_nonce", batch.BatchNonce)
		}
	}

	txs := k.GetPoolTransactions(ctx, hyperionId)

	for _, tx := range txs {
		alreadyInBatch := false
		batches := k.GetOutgoingTxBatches(ctx, hyperionId)
		for _, batch := range batches {
			for _, batchTx := range batch.Transactions {
				if batchTx.Id == tx.Id {
					alreadyInBatch = true
					break
				}
			}
		}

		if !alreadyInBatch { // we can process cancel
			sender, _ := sdk.AccAddressFromBech32(tx.Sender)
			err := k.RemoveFromOutgoingPoolAndRefund(ctx, hyperionId, tx.Id, sender)
			if err != nil {
				ctx.Logger().Error("failed to cancel outgoing tx", "error", err, "txId", tx.Id, "sender", tx.Sender)
			}
		}
	}
}

func (k *Keeper) SetWhitelistedAddresses(ctx sdk.Context, hyperionId uint64, whitelistedAddresses *types.WhitelistedAddresses) {
	store := ctx.KVStore(k.storeKey)
	whitelistStore := prefix.NewStore(store, types.WhitelistKey)
	whitelistBytes := k.cdc.MustMarshalJSON(whitelistedAddresses)
	whitelistStore.Set(types.GetWhitelistKey(hyperionId), whitelistBytes)
}

func (k *Keeper) GetWhitelistedAddresses(ctx sdk.Context, hyperionId uint64) *types.WhitelistedAddresses {
	store := ctx.KVStore(k.storeKey)
	whitelistStore := prefix.NewStore(store, types.WhitelistKey)
	whitelistBytes := whitelistStore.Get(types.GetWhitelistKey(hyperionId))
	if whitelistBytes == nil {
		return nil
	}
	whitelistedAddresses := &types.WhitelistedAddresses{}
	k.cdc.MustUnmarshalJSON(whitelistBytes, whitelistedAddresses)
	return whitelistedAddresses
}
