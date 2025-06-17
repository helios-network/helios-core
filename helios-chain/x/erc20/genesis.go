package erc20

import (
	"fmt"
	"slices"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"helios-core/helios-chain/utils"
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

		if pair.Denom == "ahelios" {
			createHeliosMetadata(ctx, k, pair, bankKeeper)
			continue
		}

		k.SetToken(ctx, pair)

		var chainsByDefaultInConsensus = []string{
			"hyperion-11155111-0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9",
			"hyperion-97-0xC689BF5a007F44410676109f8aa8E3562da1c9Ba",
			"hyperion-43113-0xd00ae08403B9bbb9124bB305C09058E32C39A48c",
			"hyperion-80002-0xA5733b3A8e62A8faF43b0376d5fAF46E89B3033E",
		}

		if slices.Contains(chainsByDefaultInConsensus, pair.Denom) {
			addTokenToConsensusWhitelist(ctx, k, pair, bankKeeper)
			continue
		}

		// if strings.Contains(pair.Denom, "helios") {
		// 	createHeliosMetadata(ctx, k, pair, bankKeeper)
		// 	continue
		// }

		// metadata, found := bankKeeper.GetDenomMetaData(ctx, pair.Denom)
		// if !found { // creation of the metadata
		// 	panic(fmt.Errorf("denom metadata not found for %s", pair.Denom))
		// }

	}
}

func addTokenToConsensusWhitelist(ctx sdk.Context, k keeper.Keeper, pair types.TokenPair, bankKeeper bankkeeper.Keeper) {

	metadata, found := bankKeeper.GetDenomMetaData(ctx, pair.Denom)
	if !found {
		return
	}

	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.IsOriginated {

			asset := types.Asset{
				Denom:           metadata.Base,
				ContractAddress: pair.Erc20Address,
				ChainId:         strconv.FormatUint(chainMetadata.ChainId, 10),
				ChainName:       strconv.FormatUint(chainMetadata.ChainId, 10),
				Decimals:        uint64(metadata.Decimals),
				BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
				Symbol:          metadata.Symbol,
			}

			k.AddAssetToConsensusWhitelist(ctx, asset)
			break
		}
	}
}

func createHeliosMetadata(ctx sdk.Context, k keeper.Keeper, pair types.TokenPair, bankKeeper bankkeeper.Keeper) {
	// // TODO REMOVE AFTER
	asset := types.Asset{
		Denom:           pair.Denom,
		ContractAddress: pair.Erc20Address,
		ChainId:         utils.MainnetChainID,
		ChainName:       "Helios",
		Decimals:        uint64(18),
		BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
		Symbol:          "HLS",
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
		Logo:            "807ff0e6f9c51651b04710e61a15ded84be227d9afe812613b871a8d75ac0d4a",
		ContractAddress: pair.Erc20Address,
	}

	// validate metadata
	if err := coinMetadata.Validate(); err != nil {
		panic(fmt.Errorf("failed to validate metadata: %w", err))
	}
	if metadata, found := bankKeeper.GetDenomMetaData(ctx, pair.Denom); found {
		coinMetadata.ChainsMetadatas = metadata.ChainsMetadatas
	}
	bankKeeper.SetDenomMetaData(ctx, coinMetadata)
}

// ExportGenesis export module status
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	return &types.GenesisState{
		Params:     k.GetParams(ctx),
		TokenPairs: k.GetTokenPairs(ctx),
	}
}
