package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"helios-core/helios-chain/contracts"
	evmostypes "helios-core/helios-chain/types"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/erc20/types"
)

var _ types.QueryServer = Keeper{}

// TokenPairs returns all registered pairs
func (k Keeper) TokenPairs(c context.Context, req *types.QueryTokenPairsRequest) (*types.QueryTokenPairsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	var pairs []types.TokenPair
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.KeyPrefixTokenPair)

	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var pair types.TokenPair
		if err := k.cdc.Unmarshal(value, &pair); err != nil {
			return err
		}
		pairs = append(pairs, pair)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryTokenPairsResponse{
		TokenPairs: pairs,
		Pagination: pageRes,
	}, nil
}

// TokenPair returns a given registered token pair
func (k Keeper) TokenPair(c context.Context, req *types.QueryTokenPairRequest) (*types.QueryTokenPairResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	// check if the token is a hex address, if not, check if it is a valid SDK
	// denom
	if err := evmostypes.ValidateAddress(req.Token); err != nil {
		if err := sdk.ValidateDenom(req.Token); err != nil {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"invalid format for token %s, should be either hex ('0x...') cosmos denom", req.Token,
			)
		}
	}

	id := k.GetTokenPairID(ctx, req.Token)

	if len(id) == 0 {
		return nil, status.Errorf(codes.NotFound, "token pair with token '%s'", req.Token)
	}

	pair, found := k.GetTokenPair(ctx, id)
	if !found {
		return nil, status.Errorf(codes.NotFound, "token pair with token '%s'", req.Token)
	}

	return &types.QueryTokenPairResponse{TokenPair: pair}, nil
}

// BalanceOf implements the Query/BalanceOf gRPC method
func (k Keeper) ERC20BalanceOf(c context.Context, req *types.QueryERC20BalanceOfRequest) (*types.QueryERC20BalanceOfResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if err := evmostypes.ValidateAddress(req.Address); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address")
	}

	if err := evmostypes.ValidateAddress(req.Token); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid token address")
	}

	ctx := sdk.UnwrapSDKContext(c)

	contract := common.HexToAddress(req.Token)
	account := common.HexToAddress(req.Address)

	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI
	balance := k.BalanceOf(ctx, erc20, contract, account)

	if balance == nil {
		return nil, status.Error(codes.Internal, "failed to get balance")
	}

	return &types.QueryERC20BalanceOfResponse{
		Balance: balance.String(),
	}, nil
}

// Params returns the params of the erc20 module
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{Params: params}, nil
}

// WhitelistedAssets returns all whitelisted assets with optional pagination
func (k Keeper) WhitelistedAssets(c context.Context, req *types.QueryWhitelistedAssetsRequest) (*types.QueryWhitelistedAssetsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	// Create a prefix store for whitelisted assets
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.WhitelistPrefix))

	var assets []types.Asset

	// Paginate through the whitelisted assets
	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		var asset types.Asset
		if err := k.cdc.Unmarshal(value, &asset); err != nil {
			return err
		}
		assets = append(assets, asset)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryWhitelistedAssetsResponse{
		Assets:     assets,
		Pagination: pageRes,
	}, nil
}
