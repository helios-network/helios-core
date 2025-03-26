package keeper

import (
	"fmt"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/feedistribution/types"
)

// Keeper of this module maintains collections of fees and contract registrations.
type Keeper struct {
	cdc              codec.BinaryCodec
	storeKey         storetypes.StoreKey
	memKey           storetypes.StoreKey
	paramstore       paramtypes.Subspace
	authority        sdk.AccAddress
	accountKeeper    types.AccountKeeper
	bankKeeper       types.BankKeeper
	evmKeeper        types.EVMKeeper
	feeCollectorName string
}

// NewKeeper creates a new Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	ps paramtypes.Subspace,
	authority sdk.AccAddress,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	evmKeeper types.EVMKeeper,
	feeCollectorName string,
) *Keeper {
	// set KeyTable if it has not already been set
	if !ps.HasKeyTable() {
		ps = ps.WithKeyTable(types.ParamKeyTable())
	}

	// ensure gov module account is set and is not nil
	if err := sdk.VerifyAddressFormat(authority); err != nil {
		panic(err)
	}

	return &Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		memKey:           memKey,
		paramstore:       ps,
		authority:        authority,
		accountKeeper:    accountKeeper,
		bankKeeper:       bankKeeper,
		evmKeeper:        evmKeeper,
		feeCollectorName: feeCollectorName,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", types.ModuleName)
}

// GetStore returns the module's KVStore.
func (k Keeper) GetStore(ctx sdk.Context) storetypes.KVStore {
	return ctx.KVStore(k.storeKey)
}

// ClearBlockFees removes the accumulated fees for a contract
func (k Keeper) ClearBlockFees(ctx sdk.Context, contract common.Address) {
	store := k.GetStore(ctx)
	key := types.GetBlockFeesKey(contract)
	store.Delete(key)
}

// DistributeFees sends fees from the fee collector to a recipient
func (k Keeper) DistributeFees(ctx sdk.Context, recipient sdk.AccAddress, fees sdk.Coins) error {
	return k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		k.feeCollectorName,
		recipient,
		fees,
	)
}

// IterateBlockFees iterates over all block fees
func (k Keeper) IterateBlockFees(ctx sdk.Context, fn func(common.Address, types.BlockFees) bool) {
	store := k.GetStore(ctx)
	prefixStore := prefix.NewStore(store, types.KeyPrefixBlockFees)

	iterator := prefixStore.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var blockFees types.BlockFees
		k.cdc.MustUnmarshal(iterator.Value(), &blockFees)

		// Convert the key (contract address bytes) to common.Address
		contract := common.BytesToAddress(iterator.Key())

		if stop := fn(contract, blockFees); stop {
			break
		}
	}
}

// GetBlockFees returns the block fees for a contract
func (k Keeper) GetBlockFees(ctx sdk.Context, contract common.Address) types.BlockFees {
	store := k.GetStore(ctx)
	key := types.GetBlockFeesKey(contract)
	bz := store.Get(key)
	if bz == nil {
		return types.BlockFees{
			ContractAddress: contract.String(),
			AccumulatedFees: math.ZeroInt(),
		}
	}

	var fees types.BlockFees
	k.cdc.MustUnmarshal(bz, &fees)
	return fees
}

// SetBlockFees stores the accumulated fees for a contract
func (k Keeper) SetBlockFees(ctx sdk.Context, contract common.Address, fees types.BlockFees) error {
	store := k.GetStore(ctx)
	key := types.GetBlockFeesKey(contract)
	bz := k.cdc.MustMarshal(&fees)
	store.Set(key, bz)
	return nil
}

// Paginate is a helper function for pagination
func (k Keeper) Paginate(
	store storetypes.KVStore,
	pageRequest *query.PageRequest,
	onResult func(key, value []byte) error,
) (*query.PageResponse, error) {
	// if no page request, return all results
	if pageRequest == nil {
		var count uint64
		iterator := store.Iterator(nil, nil)
		defer iterator.Close()

		for ; iterator.Valid(); iterator.Next() {
			if err := onResult(iterator.Key(), iterator.Value()); err != nil {
				return nil, err
			}
			count++
		}

		return &query.PageResponse{
			Total: count,
		}, nil
	}

	offset := pageRequest.Offset
	key := pageRequest.Key
	limit := pageRequest.Limit
	countTotal := pageRequest.CountTotal

	if offset > 0 && key != nil {
		return nil, fmt.Errorf("invalid request, either offset or key is expected, got both")
	}

	var iterator storetypes.Iterator
	if key == nil {
		iterator = store.Iterator(nil, nil)
	} else {
		iterator = store.Iterator(key, nil)
	}
	defer iterator.Close()

	var count uint64
	var nextKey []byte

	// skip offset
	for i := uint64(0); i < offset && iterator.Valid(); i++ {
		iterator.Next()
	}

	// gather results
	for ; iterator.Valid() && count < limit; iterator.Next() {
		if err := onResult(iterator.Key(), iterator.Value()); err != nil {
			return nil, err
		}
		count++
	}

	if iterator.Valid() {
		nextKey = iterator.Key()
	}

	// count total if requested
	var total uint64
	if countTotal {
		total = offset + count
		if iterator.Valid() {
			for ; iterator.Valid(); iterator.Next() {
				total++
			}
		}
	}

	return &query.PageResponse{
		NextKey: nextKey,
		Total:   total,
	}, nil
}

// GetPaginatedIndexes returns the start and end indexes for pagination
func (k Keeper) GetPaginatedIndexes(total int, pageRequest *query.PageRequest) (start, end int) {
	if pageRequest == nil {
		return 0, total
	}

	start = int(pageRequest.Offset)
	if start >= total {
		return total, total
	}

	end = int(pageRequest.Offset + pageRequest.Limit)
	if end > total {
		end = total
	}

	return start, end
}

// GetPageResponse returns a page response with the next key and total if requested
func (k Keeper) GetPageResponse(total int, pageRequest *query.PageRequest) *query.PageResponse {
	if pageRequest == nil {
		return &query.PageResponse{
			Total: uint64(total),
		}
	}

	_, end := k.GetPaginatedIndexes(total, pageRequest)
	nextKey := []byte(nil)
	if end < total {
		nextKey = []byte(fmt.Sprintf("%d", end))
	}

	return &query.PageResponse{
		NextKey: nextKey,
		Total:   uint64(total),
	}
}

// IsContract determines if the given address is a smart contract by checking if it has code associated with it
func (k Keeper) IsContract(ctx sdk.Context, addr common.Address) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixCodeHash)
	return store.Has(addr.Bytes())
}
