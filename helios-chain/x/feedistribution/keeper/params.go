package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/feedistribution/types"
)

// GetParams returns the current feedistribution module parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := k.GetStore(ctx)
	bz := store.Get(types.KeyPrefixParams)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the feedistribution module parameters.
// It implements the GovKeeper.ParamSetter interface.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	store := k.GetStore(ctx)
	bz := k.cdc.MustMarshal(&params)
	store.Set(types.KeyPrefixParams, bz)

	return nil
}
