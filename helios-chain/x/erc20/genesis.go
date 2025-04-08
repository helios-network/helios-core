package erc20

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"
)

// InitGenesis import module genesis
func InitGenesis(
	ctx sdk.Context,
	k keeper.Keeper,
	accountKeeper authkeeper.AccountKeeper,
	data types.GenesisState,
	bankKeeper bankkeeper.Keeper,
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

		coinMetadata := banktypes.Metadata{
			Description: fmt.Sprintf("Token %s created with erc20 genesis", pair.Denom),
			Base:        pair.Denom,
			Name:        "Helios",
			Symbol:      "HLS",
			Decimals:    18,
			Display:     "HLS",
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    pair.Denom,
					Exponent: uint32(0),
				},
				{
					Denom:    "HLS",
					Exponent: uint32(18),
				},
			},
			Logo: "807ff0e6f9c51651b04710e61a15ded84be227d9afe812613b871a8d75ac0d4a",
		}

		// validate metadata
		if err := coinMetadata.Validate(); err != nil {
			panic(fmt.Errorf("failed to validate metadata: %w", err))
		}
		bankKeeper.SetDenomMetaData(ctx, coinMetadata)
	}
}

// ExportGenesis export module status
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		TokenPairs: k.GetTokenPairs(ctx),
	}
}
