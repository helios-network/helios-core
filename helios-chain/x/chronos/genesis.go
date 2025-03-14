package chronos

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/chronos/keeper"
	"helios-core/helios-chain/x/chronos/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k keeper.Keeper, genState types.GenesisState) {
	// Set all the crons
	for _, cron := range genState.CronList {
		k.AddCron(ctx, cron)
		// err := k.AddCron(ctx, cron)
		// if err != nil {
		// 	panic(err)
		// }
	}

	// this line is used by starport scaffolding # genesis/module/init
	err := k.SetParams(ctx, genState.Params)
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis
func ExportGenesis(ctx sdk.Context, k keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// Get all crons from store
	crons := k.GetAllCrons(ctx)
	genesis.CronList = crons

	return genesis
}
