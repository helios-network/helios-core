package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"helios-core/helios-chain/x/chronos/types"
)

// Schedule queries a single scheduled EVM call by ID.
func (k Keeper) Schedule(c context.Context, req *types.QueryGetScheduleRequest) (*types.QueryGetScheduleResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	scheduleStore := prefix.NewStore(store, types.ScheduleKey)

	val := scheduleStore.Get(GetScheduleIDBytes(req.Id))
	if val == nil {
		return nil, status.Errorf(codes.NotFound, "schedule with ID %d not found", req.Id)
	}

	var schedule types.Schedule
	if err := k.cdc.Unmarshal(val, &schedule); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetScheduleResponse{Schedule: schedule}, nil
}

// Schedules retrieves all scheduled EVM calls.
func (k Keeper) Schedules(c context.Context, req *types.QuerySchedulesRequest) (*types.QuerySchedulesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	scheduleStore := prefix.NewStore(store, types.ScheduleKey)

	var schedules []types.Schedule
	pageRes, err := query.Paginate(scheduleStore, req.Pagination, func(_, value []byte) error {
		var schedule types.Schedule
		if err := k.cdc.Unmarshal(value, &schedule); err != nil {
			return err
		}
		schedules = append(schedules, schedule)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QuerySchedulesResponse{
		Schedules:  schedules,
		Pagination: pageRes,
	}, nil
}

// ScheduledCallsByOwner retrieves schedules by owner address.
func (k Keeper) ScheduledCallsByOwner(c context.Context, req *types.QueryScheduledCallsByOwnerRequest) (*types.QueryScheduledCallsByOwnerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	scheduleStore := prefix.NewStore(store, types.ScheduleKey)

	var schedules []types.Schedule

	pageRes, err := query.Paginate(scheduleStore, req.Pagination, func(_, value []byte) error {
		var schedule types.Schedule
		if err := k.cdc.Unmarshal(value, &schedule); err != nil {
			return err
		}

		if schedule.OwnerAddress == req.OwnerAddress {
			schedules = append(schedules, schedule)
		}

		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryScheduledCallsByOwnerResponse{
		Schedules:  schedules,
		Pagination: pageRes,
	}, nil
}
