package insurance

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/modules/insurance/keeper"
	"github.com/Helios-Chain-Labs/metrics"
)

func (am AppModule) EndBlocker(ctx sdk.Context, k keeper.Keeper) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, am.svcTags)
	defer doneFn()
	// call automatic withdraw keeper function
	err := k.WithdrawAllMaturedRedemptions(ctx)
	if err != nil {
		metrics.ReportFuncError(metrics.Tags{
			"svc": "insurance_abci",
		})
	}
}
