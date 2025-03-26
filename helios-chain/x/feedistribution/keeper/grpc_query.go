package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/feedistribution/types"
)

// Custom error types for common error scenarios
var (
	ErrEmptyRequest   = status.Error(codes.InvalidArgument, "empty request")
	ErrInvalidAddress = func(err error) error {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid address: %v", err))
	}
	ErrContractNotFound = func(contract string) error {
		return status.Error(codes.NotFound, fmt.Sprintf("contract not found: %s", contract))
	}
	ErrInvalidPagination = status.Error(codes.InvalidArgument, "invalid pagination parameters")
)

// Helper functions for common validations
func (k Keeper) validateRequest(req interface{}) error {
	if req == nil {
		return ErrEmptyRequest
	}
	return nil
}

func (k Keeper) validateAddress(address string) error {
	if err := types.ValidateAddress(address); err != nil {
		return ErrInvalidAddress(err)
	}
	return nil
}

func (k Keeper) validateContractAddress(address string) error {
	if err := k.validateAddress(address); err != nil {
		return err
	}
	return nil
}

// validateAndUnwrapContext validates the context and unwraps it to sdk.Context
func (k Keeper) validateAndUnwrapContext(c context.Context) (sdk.Context, error) {
	if c == nil {
		return sdk.Context{}, status.Error(codes.Internal, "nil context")
	}
	return sdk.UnwrapSDKContext(c), nil
}

var _ types.QueryServer = Keeper{}

// Params returns the module parameters.
// Returns codes.InvalidArgument if the request is nil.
// Returns codes.Internal if there's an error with context handling.
func (k Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	params := k.GetParams(ctx)
	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}

// BlockFees returns the accumulated fees for all contracts in the current block.
// Returns codes.InvalidArgument if the request is nil.
// Returns codes.Internal if there's an error during iteration.
func (k Keeper) BlockFees(c context.Context, req *types.QueryBlockFeesRequest) (*types.QueryBlockFeesResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	var blockFees []types.BlockFees
	k.IterateRevenues(ctx, func(contract common.Address, revenue types.Revenue) bool {
		fees := k.GetBlockFees(ctx, contract)
		if !fees.AccumulatedFees.IsZero() {
			blockFees = append(blockFees, fees)
		}
		return false
	})

	return &types.QueryBlockFeesResponse{
		BlockFees: blockFees,
	}, nil
}

// ContractInfo returns the deployer information for a given contract address.
// Returns codes.InvalidArgument if the request is nil or the address is invalid.
// Returns codes.NotFound if the contract is not registered.
func (k Keeper) ContractInfo(c context.Context, req *types.QueryContractInfoRequest) (*types.QueryContractInfoResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	if err := k.validateContractAddress(req.ContractAddress); err != nil {
		return nil, err
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	contractInfo, found := k.GetContractInfo(ctx, req.ContractAddress)
	if !found {
		return nil, ErrContractNotFound(req.ContractAddress)
	}

	return &types.QueryContractInfoResponse{
		ContractInfo: contractInfo,
	}, nil
}

// Contracts returns the list of all registered contracts and their deployers.
// Returns codes.InvalidArgument if the request is nil or pagination parameters are invalid.
// Returns codes.Internal if there's an error during pagination.
func (k Keeper) Contracts(c context.Context, req *types.QueryContractsRequest) (*types.QueryContractsResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	if req.Limit == 0 {
		req.Limit = 100 // default limit
	}

	if req.Limit > 1000 {
		return nil, ErrInvalidPagination
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	contracts, total := k.GetAllContracts(ctx, req.Offset, req.Limit)
	if contracts == nil {
		contracts = []types.ContractInfo{} // return empty slice instead of nil
	}

	return &types.QueryContractsResponse{
		Contracts: contracts,
		Total:     total,
	}, nil
}

// DeployerContracts returns all contracts deployed by a given address.
// Returns codes.InvalidArgument if the request is nil, the address is invalid, or pagination parameters are invalid.
// Returns codes.Internal if there's an error during pagination.
func (k Keeper) DeployerContracts(c context.Context, req *types.QueryDeployerContractsRequest) (*types.QueryDeployerContractsResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	deployer, err := sdk.AccAddressFromBech32(req.DeployerAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid deployer address: %v", err))
	}

	if req.Limit == 0 {
		req.Limit = 100 // default limit
	}

	if req.Limit > 1000 {
		return nil, ErrInvalidPagination
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	contracts, total := k.GetDeployerContracts(ctx, deployer.String(), req.Offset, req.Limit)
	if contracts == nil {
		contracts = []types.ContractInfo{} // return empty slice instead of nil
	}

	return &types.QueryDeployerContractsResponse{
		Contracts: contracts,
		Total:     total,
	}, nil
}

// Revenues returns all active fee distribution contracts.
// Returns codes.InvalidArgument if the request is nil or pagination parameters are invalid.
// Returns codes.Internal if there's an error during pagination.
func (k Keeper) Revenues(c context.Context, req *types.QueryRevenuesRequest) (*types.QueryRevenuesResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	if req.Pagination != nil {
		if req.Pagination.Limit > 1000 {
			return nil, ErrInvalidPagination
		}
	}

	store := k.GetStore(ctx)
	revenueStore := prefix.NewStore(store, types.KeyPrefixRevenue)

	var revenues []types.Revenue
	pageRes, err := k.Paginate(revenueStore, req.Pagination, func(key, value []byte) error {
		var revenue types.Revenue
		if err := k.cdc.Unmarshal(value, &revenue); err != nil {
			return fmt.Errorf("failed to unmarshal revenue: %w", err)
		}
		revenues = append(revenues, revenue)
		return nil
	})
	if err != nil {
		// Check if the error message indicates a pagination error
		if err.Error() == "invalid pagination parameters" {
			return nil, ErrInvalidPagination
		}
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to paginate revenues: %v", err))
	}

	if revenues == nil {
		revenues = []types.Revenue{} // return empty slice instead of nil
	}

	return &types.QueryRevenuesResponse{
		Revenues:   revenues,
		Pagination: pageRes,
	}, nil
}

// Revenue returns the revenue configuration for a given contract.
// Returns codes.InvalidArgument if the request is nil or the contract address is invalid.
// Returns codes.NotFound if the revenue configuration is not found.
func (k Keeper) Revenue(c context.Context, req *types.QueryRevenueRequest) (*types.QueryRevenueResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	if err := k.validateContractAddress(req.ContractAddress); err != nil {
		return nil, err
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	revenue, found := k.GetRevenue(ctx, common.HexToAddress(req.ContractAddress))
	if !found {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("revenue not found for contract: %s", req.ContractAddress))
	}

	return &types.QueryRevenueResponse{
		Revenue: revenue,
	}, nil
}

// DeployerRevenues returns all active fee distribution contracts for a given deployer.
// Returns codes.InvalidArgument if the request is nil, the deployer address is invalid, or pagination parameters are invalid.
// Returns codes.Internal if there's an error during pagination.
func (k Keeper) DeployerRevenues(c context.Context, req *types.QueryDeployerRevenuesRequest) (*types.QueryDeployerRevenuesResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	deployer, err := sdk.AccAddressFromBech32(req.DeployerAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid deployer address: %v", err))
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	if req.Pagination != nil {
		if req.Pagination.Limit > 1000 {
			return nil, ErrInvalidPagination
		}
	}

	var revenues []types.Revenue
	k.IterateRevenues(ctx, func(contract common.Address, revenue types.Revenue) bool {
		if revenue.DeployerAddress == deployer.String() {
			revenues = append(revenues, revenue)
		}
		return false
	})

	// Handle pagination
	start, end := k.GetPaginatedIndexes(len(revenues), req.Pagination)
	if start >= len(revenues) {
		revenues = []types.Revenue{} // return empty slice instead of nil
	} else {
		revenues = revenues[start:end]
	}

	return &types.QueryDeployerRevenuesResponse{
		Revenues:   revenues,
		Pagination: k.GetPageResponse(len(revenues), req.Pagination),
	}, nil
}

// WithdrawerRevenues returns all active fee distribution contracts for a given withdrawer.
// Returns codes.InvalidArgument if the request is nil, the withdrawer address is invalid, or pagination parameters are invalid.
// Returns codes.Internal if there's an error during pagination.
func (k Keeper) WithdrawerRevenues(c context.Context, req *types.QueryWithdrawerRevenuesRequest) (*types.QueryWithdrawerRevenuesResponse, error) {
	if err := k.validateRequest(req); err != nil {
		return nil, err
	}

	withdrawer, err := sdk.AccAddressFromBech32(req.WithdrawerAddress)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid withdrawer address: %v", err))
	}

	ctx, err := k.validateAndUnwrapContext(c)
	if err != nil {
		return nil, err
	}

	if req.Pagination != nil {
		if req.Pagination.Limit > 1000 {
			return nil, ErrInvalidPagination
		}
	}

	var revenues []types.Revenue
	k.IterateRevenues(ctx, func(contract common.Address, revenue types.Revenue) bool {
		if revenue.WithdrawerAddress == withdrawer.String() {
			revenues = append(revenues, revenue)
		}
		return false
	})

	// Handle pagination
	start, end := k.GetPaginatedIndexes(len(revenues), req.Pagination)
	if start >= len(revenues) {
		revenues = []types.Revenue{} // return empty slice instead of nil
	} else {
		revenues = revenues[start:end]
	}

	return &types.QueryWithdrawerRevenuesResponse{
		Revenues:   revenues,
		Pagination: k.GetPageResponse(len(revenues), req.Pagination),
	}, nil
}
