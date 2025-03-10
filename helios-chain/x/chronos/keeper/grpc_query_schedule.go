package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

// Cron queries a single scheduled EVM call by ID.
func (k Keeper) QueryGetCron(c context.Context, req *types.QueryGetCronRequest) (*types.QueryGetCronResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	scheduleStore := prefix.NewStore(store, types.CronKey)

	val := scheduleStore.Get(GetScheduleIDBytes(req.Id))
	if val == nil {
		return nil, status.Errorf(codes.NotFound, "cron with ID %d not found", req.Id)
	}

	var cron types.Cron
	if err := k.cdc.Unmarshal(val, &cron); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetCronResponse{Cron: cron}, nil
}

// Crons retrieves all scheduled EVM calls.
func (k Keeper) QueryGetCrons(c context.Context, req *types.QueryGetCronsRequest) (*types.QueryGetCronsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	cronStore := prefix.NewStore(store, types.CronKey)

	var crons []types.Cron
	pageRes, err := query.Paginate(cronStore, req.Pagination, func(_, value []byte) error {
		var cron types.Cron
		if err := k.cdc.Unmarshal(value, &cron); err != nil {
			return err
		}
		crons = append(crons, cron)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetCronsResponse{
		Crons:      crons,
		Pagination: pageRes,
	}, nil
}

// GetCronsByOwner retrieves schedules by owner address.
func (k Keeper) QueryGetCronsByOwner(c context.Context, req *types.QueryGetCronsByOwnerRequest) (*types.QueryGetCronsByOwnerResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	scheduleStore := prefix.NewStore(store, types.CronKey)

	var crons []types.Cron

	pageRes, err := query.Paginate(scheduleStore, req.Pagination, func(_, value []byte) error {
		var cron types.Cron
		if err := k.cdc.Unmarshal(value, &cron); err != nil {
			return err
		}
		if cron.OwnerAddress == req.OwnerAddress {
			crons = append(crons, cron)
		}
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetCronsByOwnerResponse{
		Crons:      crons,
		Pagination: pageRes,
	}, nil
}

func (k Keeper) QueryGetCronTransactionByNonce(c context.Context, req *types.QueryGetCronTransactionByNonceRequest) (*types.QueryGetCronTransactionByNonceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	tx, err := k.GetTransactionByNonce(sdk.UnwrapSDKContext(c), req.Nonce)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetCronTransactionByNonceResponse{
		Transaction: tx,
	}, nil
}

func (k Keeper) QueryGetCronTransactionReceiptLogsByBlockNumber(c context.Context, req *types.QueryGetCronTransactionReceiptLogsByBlockNumberRequest) (*types.QueryGetCronTransactionReceiptLogsByBlockNumberResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	tx, ok := k.GetCronTransactionLogsByBlockNumber(sdk.UnwrapSDKContext(c), req.BlockNumber)
	if !ok {
		return &types.QueryGetCronTransactionReceiptLogsByBlockNumberResponse{
			Logs: []*evmtypes.Log{},
		}, nil
	}

	return &types.QueryGetCronTransactionReceiptLogsByBlockNumberResponse{
		Logs: tx,
	}, nil
}

func (k Keeper) QueryGetCronTransactionReceiptsByBlockNumber(c context.Context, req *types.QueryGetCronTransactionReceiptsByBlockNumberRequest) (*types.QueryGetCronTransactionReceiptsByBlockNumberResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	txs, ok := k.GetCronTransactionReceiptsByBlockNumber(sdk.UnwrapSDKContext(c), req.BlockNumber)
	if !ok {
		return &types.QueryGetCronTransactionReceiptsByBlockNumberResponse{
			Transactions: []*types.CronTransactionReceiptRPC{},
		}, nil
	}

	return &types.QueryGetCronTransactionReceiptsByBlockNumberResponse{
		Transactions: txs,
	}, nil
}

func (k Keeper) QueryGetCronTransactionReceiptByHash(c context.Context, req *types.QueryGetCronTransactionReceiptByHashRequest) (*types.QueryGetCronTransactionReceiptByHashResponse, error) {
	tx, ok := k.GetCronTransactionReceiptByHash(sdk.UnwrapSDKContext(c), req.Hash)
	if !ok {
		return nil, status.Error(codes.NotFound, "Tx not found")
	}

	return &types.QueryGetCronTransactionReceiptByHashResponse{
		Transaction: tx,
	}, nil
}

func (k Keeper) QueryGetCronTransactionReceiptByNonce(c context.Context, req *types.QueryGetCronTransactionReceiptByNonceRequest) (*types.QueryGetCronTransactionReceiptByNonceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	tx, ok := k.GetCronTransactionReceiptByNonce(sdk.UnwrapSDKContext(c), req.Nonce)
	if !ok {
		return nil, status.Error(codes.NotFound, "cron transaction not found")
	}

	return &types.QueryGetCronTransactionReceiptByNonceResponse{
		Transaction: tx,
	}, nil
}

func (k Keeper) QueryGetCronTransactionReceiptsByPageAndSize(c context.Context, req *types.QueryGetCronTransactionReceiptsByPageAndSizeRequest) (*types.QueryGetCronTransactionReceiptsByPageAndSizeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	store := ctx.KVStore(k.storeKey)
	cronStore := prefix.NewStore(store, types.CronTransactionResultKey)

	var schedulesTxReceipts []*types.CronTransactionReceiptRPC
	pageRes, err := query.Paginate(cronStore, req.Pagination, func(_, value []byte) error {
		var tx types.CronTransactionResult
		if err := k.cdc.Unmarshal(value, &tx); err != nil {
			return err
		}
		txReceipt, err := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
		if err != nil {
			return err
		}
		schedulesTxReceipts = append(schedulesTxReceipts, txReceipt)
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryGetCronTransactionReceiptsByPageAndSizeResponse{
		Transactions: schedulesTxReceipts,
		Pagination:   pageRes,
	}, nil
}
