package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/modules/hyperion/types"
)

// NormalizeGenesis takes care of formatting in the internal structures, as they're used as values
// in the keeper eventually, while having raw strings in them.
func NormalizeGenesis(data *types.GenesisState) {
	for _, counterpartyParams := range data.Params.CounterpartyChainParams {
		counterpartyParams.BridgeCounterpartyAddress = common.HexToAddress(counterpartyParams.BridgeCounterpartyAddress).Hex()
		counterpartyParams.CosmosCoinErc20Contract = common.HexToAddress(counterpartyParams.CosmosCoinErc20Contract).Hex()
	}
}

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data *types.GenesisState) {
	k.CreateModuleAccount(ctx)

	NormalizeGenesis(data)

	k.SetParams(ctx, data.Params)
	// populate state with cosmos originated denom-erc20 mapping
	for _, item := range data.Params.CounterpartyChainParams {
		for _, denom := range item.Erc20ToDenoms {
			k.SetCosmosOriginatedDenomToERC20ByHyperionID(ctx, denom.Denom, common.HexToAddress(denom.Erc20), item.HyperionId)
		}
	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	var (
		p                               = k.GetParams(ctx)
	)
	return types.GenesisState{
		Params:                     p,
	}
}
