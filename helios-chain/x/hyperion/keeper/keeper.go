package keeper

import (
	gomath "math"
	"sort"

	erc20keeper "helios-core/helios-chain/x/erc20/keeper"

	"github.com/cosmos/cosmos-sdk/x/auth/keeper"
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

	StakingKeeper  types.StakingKeeper
	bankKeeper     types.BankKeeper
	DistKeeper     distrkeeper.Keeper
	SlashingKeeper types.SlashingKeeper
	erc20Keeper    erc20keeper.Keeper

	AttestationHandler interface {
		Handle(sdk.Context, types.EthereumClaim) error
	}

	svcTags  metrics.Tags
	grpcTags metrics.Tags

	// address authorized to execute MsgUpdateParams. Default: gov module
	authority     string
	accountKeeper keeper.AccountKeeper
}

func (k *Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// NewKeeper returns a new instance of the hyperion keeper
func NewKeeper(
	cdc codec.Codec,
	storeKey storetypes.StoreKey,
	stakingKeeper types.StakingKeeper,
	bankKeeper types.BankKeeper,
	slashingKeeper types.SlashingKeeper,
	distKeeper distrkeeper.Keeper,
	authority string,
	accountKeeper keeper.AccountKeeper,
	erc20Keeper erc20keeper.Keeper,
) Keeper {

	k := Keeper{
		cdc:            cdc,
		storeKey:       storeKey,
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
	}

	k.AttestationHandler = NewAttestationHandler(bankKeeper, k)

	return k
}

/////////////////////////////
//     VALSET REQUESTS     //
/////////////////////////////

// SetValsetRequest returns a new instance of the Hyperion BridgeValidatorSet
// i.e. {"nonce": 1, "memebers": [{"eth_addr": "foo", "power": 11223}]}
func (k *Keeper) SetValsetRequest(ctx sdk.Context, hyperionId uint64) *types.Valset {
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
	k.SetPastEthSignatureCheckpoint(ctx, checkpoint)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventValsetUpdateRequest{
		HyperionId:    hyperionId,
		ValsetNonce:   valset.Nonce,
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

	k.IterateValsetBySlashedValsetNonce(ctx, lastSlashedValsetNonce, maxHeight, func(_ []byte, valset *types.Valset) bool {
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
	maxHeight uint64,
	cb func(k []byte, v *types.Valset) (stop bool),
) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ValsetRequestKey)
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
	iter := prefixStore.Iterator(PrefixRange(types.GetValsetConfirmPrefixKey(hyperionId, nonce)))
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
		rewardToken, rewardAmount = k.RewardToERC20Lookup(ctx, reward, hyperionId)
	}
	// TODO: make the nonce an incrementing one (i.e. fetch last nonce from state, increment, set here)
	return types.NewValset(hyperionId, uint64(ctx.BlockHeight()), uint64(ctx.BlockHeight()), bridgeValidators, rewardAmount, rewardToken)
}

/////////////////////////////
//       HELPERS           //
/////////////////////////////

func (k *Keeper) getStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
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

// GetCosmosCoinDenom returns a mapping (hyperion id => the Cosmos native coin)
func (k *Keeper) GetCosmosCoinDenom(ctx sdk.Context) map[uint64]string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]string{}
	}
	cosmosCoinDenom := make(map[uint64]string)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		cosmosCoinDenom[counterpartyChainParams.HyperionId] = counterpartyChainParams.CosmosCoinDenom
	}

	return cosmosCoinDenom
}

// GetCosmosCoinERC20Contract returns a mapping (hyperion id => the Cosmos coin ERC20 contract address of the counterparty chain)
func (k *Keeper) GetCosmosCoinERC20Contract(ctx sdk.Context) map[uint64]common.Address {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	params := k.GetParams(ctx)
	if params == nil {
		return map[uint64]common.Address{}
	}
	cosmosCoinErc20ContractMap := make(map[uint64]common.Address)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		cosmosCoinErc20ContractMap[counterpartyChainParams.HyperionId] = common.HexToAddress(counterpartyChainParams.CosmosCoinErc20Contract)
	}

	return cosmosCoinErc20ContractMap
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

// IsOnBlacklist checks that the Ethereum Address is black listed.
func (k *Keeper) IsOnBlacklist(ctx sdk.Context, addr common.Address) bool {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	return k.getStore(ctx).Has(types.GetEthereumBlacklistStoreKey(addr))
}

// SetEthereumBlacklistAddress sets the ethereum blacklist address.
func (k *Keeper) SetEthereumBlacklistAddress(ctx sdk.Context, addr common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// set boolean indicator
	k.getStore(ctx).Set(types.GetEthereumBlacklistStoreKey(addr), []byte{})
}

// GetAllEthereumBlacklistAddresses fetches all etheruem blacklisted addresses.
func (k *Keeper) GetAllEthereumBlacklistAddresses(ctx sdk.Context) []string {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	blacklistedAddresses := make([]string, 0)
	store := ctx.KVStore(k.storeKey)
	blacklistAddressStore := prefix.NewStore(store, types.EthereumBlacklistKey)

	iterator := blacklistAddressStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		blacklistAddress := common.BytesToAddress(iterator.Key())
		blacklistedAddresses = append(blacklistedAddresses, blacklistAddress.String())
	}

	return blacklistedAddresses
}

// DeleteEthereumBlacklistAddress deletes the address from blacklist.
func (k *Keeper) DeleteEthereumBlacklistAddress(ctx sdk.Context, addr common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.getStore(ctx).Delete(types.GetEthereumBlacklistStoreKey(addr))
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

func (k *Keeper) isAdmin(ctx sdk.Context, addr string) bool {
	for _, adminAddress := range k.GetParams(ctx).Admins {
		if adminAddress == addr {
			return true
		}
	}
	return false
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
	// we project how long it has been in milliseconds since the last Ethereum block height was observed
	projectedMillis := (uint64(currentCosmosHeight) - heights.CosmosBlockHeight) * counterpartyChainParams.AverageBlockTime
	// we convert that projection into the current Ethereum height using the average Ethereum block time in millis
	projectedCurrentEthereumHeight := (projectedMillis / counterpartyChainParams.AverageCounterpartyBlockTime) + heights.EthereumBlockHeight

	return projectedCurrentEthereumHeight
}
