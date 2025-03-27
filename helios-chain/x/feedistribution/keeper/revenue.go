package keeper

import (
	"helios-core/helios-chain/x/feedistribution/types"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetRevenue returns the Revenue for a registered contract
func (k Keeper) GetRevenue(ctx sdk.Context, contract common.Address) (types.Revenue, bool) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixRevenue)

	var revenue types.Revenue
	bz := store.Get(contract.Bytes())
	if len(bz) == 0 {
		return revenue, false
	}

	k.cdc.MustUnmarshal(bz, &revenue)
	return revenue, true
}

// SetRevenue stores a contract for fee distribution
func (k Keeper) SetRevenue(ctx sdk.Context, revenue types.Revenue) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixRevenue)
	contract := common.HexToAddress(revenue.ContractAddress)
	bz := k.cdc.MustMarshal(&revenue)
	store.Set(contract.Bytes(), bz)
}

// IterateRevenues iterates over all registered contracts
func (k Keeper) IterateRevenues(ctx sdk.Context, handler func(contract common.Address, revenue types.Revenue) (stop bool)) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixRevenue)
	prefixLength := len(types.KeyPrefixRevenue)

	iterator := storetypes.KVStorePrefixIterator(store, types.KeyPrefixRevenue)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		contract := common.BytesToAddress(iterator.Key()[prefixLength:])

		var revenue types.Revenue
		k.cdc.MustUnmarshal(iterator.Value(), &revenue)

		if handler(contract, revenue) {
			break
		}
	}
}

// SetDeployerMap stores a contract-by-deployer mapping
func (k Keeper) SetDeployerMap(
	ctx sdk.Context,
	deployer sdk.AccAddress,
	contract common.Address,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixDeployer)
	key := append(deployer.Bytes(), contract.Bytes()...)
	store.Set(key, []byte{1})
}

// SetWithdrawerMap stores a contract-by-withdrawer mapping
func (k Keeper) SetWithdrawerMap(
	ctx sdk.Context,
	withdrawer sdk.AccAddress,
	contract common.Address,
) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixWithdrawer)
	key := append(withdrawer.Bytes(), contract.Bytes()...)
	store.Set(key, []byte{1})
}
