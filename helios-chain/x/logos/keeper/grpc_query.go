package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/logos/types"
)

var _ types.QueryServer = Keeper{}

// Params implements the Query/Params request
func (k Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}

// Logo implements the Query/Logo request
func (k Keeper) Logo(c context.Context, req *types.QueryLogoRequest) (*types.QueryLogoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	logo, found := k.GetLogo(ctx, req.Hash)
	if !found {
		return nil, status.Error(codes.NotFound, "logo not found")
	}

	return &types.QueryLogoResponse{Logo: &logo}, nil
}
