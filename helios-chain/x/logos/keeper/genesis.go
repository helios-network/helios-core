package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/logos/types"
)

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data *types.GenesisState) {
	k.SetParams(ctx, *data.Params)

	for _, logo := range data.Logos {
		k.SetLogo(ctx, *logo)
	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	p := k.GetParams(ctx)

	state := types.GenesisState{
		Params: &p,
		Logos:  make([]*types.Logo, 0),
	}

	logos := k.GetAllLogos(ctx)

	for _, logo := range logos {
		state.Logos = append(state.Logos, &logo)
	}

	return state
}
