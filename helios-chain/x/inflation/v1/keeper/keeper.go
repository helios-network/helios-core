package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
)

// Keeper is a wrapper around the Cosmos SDK staking keeper.
type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec
}

// NewKeeper creates a new staking Keeper wrapper instance.
func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
) *Keeper {
	return &Keeper{
		cdc:      cdc,
		storeKey: storeKey,
	}
}
