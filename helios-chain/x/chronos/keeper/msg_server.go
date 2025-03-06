package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	errors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"helios-core/helios-chain/x/chronos/types"
)

type msgServer struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// ScheduleEVMCall schedules a new EVM call
func (k msgServer) ScheduleEVMCall(goCtx context.Context, req *types.MsgScheduleEVMCall) (*types.MsgScheduleEVMCallResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errorsmod.Wrap(err, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	newID := k.keeper.GetNextScheduleID(ctx)

	newSchedule := types.Schedule{
		Id:                 newID,
		OwnerAddress:       req.OwnerAddress,
		ContractAddress:    req.ContractAddress,
		AbiJson:            req.AbiJson,
		MethodName:         req.MethodName,
		Params:             req.Params,
		Frequency:          req.Frequency,
		NextExecutionBlock: uint64(ctx.BlockHeight()) + req.Frequency,
		ExpirationBlock:    req.ExpirationBlock,
	}

	if err := k.keeper.AddSchedule(ctx, newSchedule); err != nil {
		return nil, errorsmod.Wrap(err, "failed to add schedule")
	}

	return &types.MsgScheduleEVMCallResponse{ScheduleId: newSchedule.Id}, nil
}

// ModifyScheduledEVMCall modifies an existing scheduled EVM call
func (k msgServer) ModifyScheduledEVMCall(goCtx context.Context, req *types.MsgModifyScheduledEVMCall) (*types.MsgModifyScheduledEVMCallResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := k.keeper.GetSchedule(ctx, req.ScheduleId)
	if !found {
		return nil, errors.Wrapf(errortypes.ErrNotFound, "schedule %d not found", req.ScheduleId)
	}

	if schedule.OwnerAddress != req.OwnerAddress {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, "only owner can modify the schedule")
	}
	schedule.Frequency = req.NewFrequency
	schedule.Params = req.NewParams
	schedule.ExpirationBlock = req.NewExpirationBlock

	k.keeper.StoreSchedule(ctx, schedule)

	return &types.MsgModifyScheduledEVMCallResponse{Success: true}, nil
}

// CancelScheduledEVMCall cancels a scheduled EVM call
func (k msgServer) CancelScheduledEVMCall(goCtx context.Context, req *types.MsgCancelScheduledEVMCall) (*types.MsgCancelScheduledEVMCallResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := k.keeper.GetSchedule(ctx, req.ScheduleId)
	if !found {
		return nil, errors.Wrapf(errortypes.ErrNotFound, "schedule %d not found", req.ScheduleId)
	}

	if schedule.OwnerAddress != req.OwnerAddress {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, "only owner can cancel the schedule")
	}

	if err := k.keeper.RemoveSchedule(ctx, req.ScheduleId, sdk.MustAccAddressFromBech32(req.OwnerAddress)); err != nil {
		return nil, errorsmod.Wrap(err, "failed to remove schedule")
	}

	return &types.MsgCancelScheduledEVMCallResponse{Success: true}, nil
}
