package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"helios-core/helios-chain/x/vesting/types"
)

var _ types.QueryServer = Keeper{}

// Balances returns the locked, unvested and vested amount of tokens for a
// clawback vesting account
func (k Keeper) Balances(
	goCtx context.Context,
	req *types.QueryBalancesRequest,
) (*types.QueryBalancesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	clawbackAccount, err := k.GetClawbackVestingAccount(goCtx, addr)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"account at address '%s' either does not exist or is not a vesting account ", addr.String(),
		)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	locked := clawbackAccount.GetLockedUpCoins(ctx.BlockTime())
	unvested := clawbackAccount.GetVestingCoins(ctx.BlockTime())
	vested := clawbackAccount.GetVestedCoins(ctx.BlockTime())

	return &types.QueryBalancesResponse{
		Locked:   locked,
		Unvested: unvested,
		Vested:   vested,
	}, nil
}
