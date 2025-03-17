package keeper

import (
	"context"

	// errorsmod "cosmossdk.io/errors"
	// sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/inflation/v1/types"
)

var _ types.QueryServer = Keeper{}

// Params implements the Query/Params gRPC method
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

func (k Keeper) Period(c context.Context, req *types.QueryPeriodRequest) (*types.QueryPeriodResponse, error) {
	return nil, nil
}

func (k Keeper) CirculatingSupply(c context.Context, req *types.QueryCirculatingSupplyRequest) (*types.QueryCirculatingSupplyResponse, error) {
	return &types.QueryCirculatingSupplyResponse{}, nil
}

func (k Keeper) EpochMintProvision(c context.Context, req *types.QueryEpochMintProvisionRequest) (*types.QueryEpochMintProvisionResponse, error) {
	return &types.QueryEpochMintProvisionResponse{}, nil
}

func (k Keeper) InflationRate(c context.Context, req *types.QueryInflationRateRequest) (*types.QueryInflationRateResponse, error) {
	return &types.QueryInflationRateResponse{}, nil
}

func (k Keeper) SkippedEpochs(c context.Context, req *types.QuerySkippedEpochsRequest) (*types.QuerySkippedEpochsResponse, error) {
	return &types.QuerySkippedEpochsResponse{}, nil
}
