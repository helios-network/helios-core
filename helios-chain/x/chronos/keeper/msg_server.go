package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/chronos/types"

	errors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

type msgServer struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateCron create a new cron
func (k msgServer) CreateCron(goCtx context.Context, req *types.MsgCreateCron) (*types.MsgCreateCronResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, fmt.Sprintf("only the owner can schedule an EVM call %s != %s", req.OwnerAddress, req.Sender))
	}

	newID := k.keeper.StoreGetNextCronID(ctx)

	newCron := types.Cron{
		Id:                 newID,
		OwnerAddress:       req.OwnerAddress,
		ContractAddress:    req.ContractAddress,
		AbiJson:            req.AbiJson,
		MethodName:         req.MethodName,
		Params:             req.Params,
		Frequency:          req.Frequency,
		NextExecutionBlock: uint64(ctx.BlockHeight()) + req.Frequency,
		ExpirationBlock:    req.ExpirationBlock,
		GasLimit:           req.GasLimit,
		MaxGasPrice:        req.MaxGasPrice,
	}

	if err := k.keeper.AddCron(ctx, newCron); err != nil {
		return nil, errors.Wrap(err, "failed to add schedule")
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"CreateCron",
			sdk.NewAttribute("cron_id", fmt.Sprintf("%d", newCron.Id)),
			sdk.NewAttribute("owner_address", req.OwnerAddress),
			sdk.NewAttribute("contract_address", req.ContractAddress),
			sdk.NewAttribute("method_name", req.MethodName),
		),
	)

	return &types.MsgCreateCronResponse{CronId: newCron.Id}, nil
}

// UpdateCron modifies an existing cron
func (k msgServer) UpdateCron(goCtx context.Context, req *types.MsgUpdateCron) (*types.MsgUpdateCronResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cron, found := k.keeper.GetCron(ctx, req.CronId)
	if !found {
		return nil, errors.Wrapf(errortypes.ErrNotFound, "cron %d not found", req.CronId)
	}

	if cron.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, "only the owner can edit")
	}
	cron.Frequency = req.NewFrequency
	cron.Params = req.NewParams
	cron.ExpirationBlock = req.NewExpirationBlock
	cron.GasLimit = req.NewGasLimit
	cron.MaxGasPrice = req.NewMaxGasPrice

	k.keeper.StoreSetCron(ctx, cron)

	return &types.MsgUpdateCronResponse{Success: true}, nil
}

// CancelCron cancels a cron
func (k msgServer) CancelCron(goCtx context.Context, req *types.MsgCancelCron) (*types.MsgCancelCronResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := k.keeper.GetCron(ctx, req.CronId)
	if !found {
		return nil, errors.Wrapf(errortypes.ErrNotFound, "cron %d not found", req.CronId)
	}

	if schedule.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, "only owner can cancel the schedule")
	}

	if err := k.keeper.RemoveCron(ctx, req.CronId, sdk.MustAccAddressFromBech32(req.OwnerAddress)); err != nil {
		return nil, errors.Wrap(err, "failed to remove schedule")
	}

	return &types.MsgCancelCronResponse{Success: true}, nil
}
