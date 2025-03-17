package erc20

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	data types.GenesisState,
) {
	err := k.SetParams(ctx, data.Params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}

	// ensure erc20 module account is set on genesis
	if acc := accountKeeper.GetModuleAccount(ctx, types.ModuleName); acc == nil {
		// NOTE: shouldn't occur
		panic("the erc20 module account has not been set")
	}
	for _, pair := range data.TokenPairs {
		// TODO REMOVE AFTER
		asset := types.Asset{
			Denom:           pair.Denom,
			ContractAddress: pair.Erc20Address,
			ChainId:         "helios",
			Decimals:        uint64(18),
			BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
			Metadata:        fmt.Sprintf("Token %s metadata", "ahelios"),
		}

		// TODO : remove this !!
		k.AddAssetToConsensusWhitelist(ctx, asset)
		k.SetToken(ctx, pair)
	}
}

// ExportGenesis export module status
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		TokenPairs: k.GetTokenPairs(ctx),
	}
}
