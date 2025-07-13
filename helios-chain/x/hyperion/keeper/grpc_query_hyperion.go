package keeper

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

// [Used In Hyperion] Params queries the params of the hyperion module
func (k *Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	return &types.QueryParamsResponse{Params: *k.GetParams(sdk.UnwrapSDKContext(c))}, nil
}

// [Used In Hyperion] CurrentValset queries the CurrentValset of the hyperion module
func (k *Keeper) CurrentValset(c context.Context, req *types.QueryCurrentValsetRequest) (*types.QueryCurrentValsetResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	return &types.QueryCurrentValsetResponse{Valset: k.GetCurrentValset(sdk.UnwrapSDKContext(c), req.HyperionId)}, nil
}

// [Used In Hyperion] ValsetRequest queries the ValsetRequest of the hyperion module
func (k *Keeper) ValsetRequest(c context.Context, req *types.QueryValsetRequestRequest) (*types.QueryValsetRequestResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	return &types.QueryValsetRequestResponse{Valset: k.GetValset(sdk.UnwrapSDKContext(c), req.HyperionId, req.Nonce)}, nil
}

// [Used In Hyperion] ValsetConfirmsByNonce queries the ValsetConfirmsByNonce of the hyperion module
func (k *Keeper) ValsetConfirmsByNonce(c context.Context, req *types.QueryValsetConfirmsByNonceRequest) (*types.QueryValsetConfirmsByNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	confirms := make([]*types.MsgValsetConfirm, 0)

	k.IterateValsetConfirmByNonce(sdk.UnwrapSDKContext(c), req.HyperionId, req.Nonce, func(_ []byte, valset *types.MsgValsetConfirm) (stop bool) {
		confirms = append(confirms, valset)

		return false
	})

	return &types.QueryValsetConfirmsByNonceResponse{Confirms: confirms}, nil
}

// [Used In Hyperion] LastValsetRequests queries the LastValsetRequests of the hyperion module
func (k *Keeper) LastValsetRequests(c context.Context, req *types.QueryLastValsetRequestsRequest) (*types.QueryLastValsetRequestsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	maxValsetRequestsReturned := 5

	valReq := k.GetValsets(sdk.UnwrapSDKContext(c), req.HyperionId)
	valReqLen := len(valReq)
	retLen := 0

	if valReqLen < maxValsetRequestsReturned {
		retLen = valReqLen
	} else {
		retLen = maxValsetRequestsReturned
	}

	return &types.QueryLastValsetRequestsResponse{Valsets: valReq[0:retLen]}, nil
}

// [Used In Hyperion] LastPendingValsetRequestByAddr queries the LastPendingValsetRequestByAddr of the hyperion module
func (k *Keeper) LastPendingValsetRequestByAddr(c context.Context, req *types.QueryLastPendingValsetRequestByAddrRequest) (*types.QueryLastPendingValsetRequestByAddrResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "address invalid")
	}

	pendingValsetReq := make([]*types.Valset, 0)
	k.IterateValsets(sdk.UnwrapSDKContext(c), func(_ []byte, val *types.Valset) bool {
		if val.HyperionId != req.HyperionId {
			// return false to continue the loop
			return false
		}
		// foundConfirm is true if the operatorAddr has signed the valset we are currently looking at
		foundConfirm := k.GetValsetConfirm(sdk.UnwrapSDKContext(c), val.HyperionId, val.Nonce, addr) != nil
		// if this valset has NOT been signed by operatorAddr, store it in pendingValsetReq
		// and exit the loop
		if !foundConfirm {
			pendingValsetReq = append(pendingValsetReq, val)
		}
		// if we have more than 100 unconfirmed requests in
		// our array we should exit, TODO pagination
		if len(pendingValsetReq) > 100 {
			return true
		}
		// return false to continue the loop
		return false
	})

	return &types.QueryLastPendingValsetRequestByAddrResponse{Valsets: pendingValsetReq}, nil
}

// [Used In Hyperion] BatchFees queries the batch fees from unbatched pool
func (k *Keeper) BatchFees(c context.Context, req *types.QueryBatchFeeRequest) (*types.QueryBatchFeeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	return &types.QueryBatchFeeResponse{BatchFees: k.GetAllBatchFees(sdk.UnwrapSDKContext(c), req.HyperionId, sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0)), sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0)))}, nil
}

// [Used In Hyperion] BatchFeesWithMinimumFee queries the batch fees from unbatched pool with a minimum fee
func (k *Keeper) BatchFeesWithMinimumFee(c context.Context, req *types.QueryBatchFeeWithMinimumFeeRequest) (*types.QueryBatchFeeWithMinimumFeeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	return &types.QueryBatchFeeWithMinimumFeeResponse{BatchFees: k.GetAllBatchFees(sdk.UnwrapSDKContext(c), req.HyperionId, req.MinimumBatchFee, req.MinimumTxFee)}, nil
}

// [Used In Hyperion] LastPendingBatchRequestByAddr queries the LastPendingBatchRequestByAddr of the hyperion module
func (k *Keeper) LastPendingBatchRequestByAddr(c context.Context, req *types.QueryLastPendingBatchRequestByAddrRequest) (*types.QueryLastPendingBatchRequestByAddrResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "address invalid")
	}

	var pendingBatchReq *types.OutgoingTxBatch
	k.IterateOutgoingTXBatches(sdk.UnwrapSDKContext(c), req.HyperionId, func(_ []byte, batch *types.OutgoingTxBatch) (stop bool) {
		foundConfirm := k.GetBatchConfirm(sdk.UnwrapSDKContext(c), batch.HyperionId, batch.BatchNonce, common.HexToAddress(batch.TokenContract), addr) != nil
		if !foundConfirm {
			pendingBatchReq = batch
			return true
		}

		return false
	})

	return &types.QueryLastPendingBatchRequestByAddrResponse{Batch: pendingBatchReq}, nil
}

func (k *Keeper) LastPendingBatchsRequestByAddr(c context.Context, req *types.QueryLastPendingBatchsRequestByAddrRequest) (*types.QueryLastPendingBatchsRequestByAddrResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "address invalid")
	}

	MaxResults := 100 // todo: impl pagination
	pendingBatchReqs := make([]*types.OutgoingTxBatch, 0)

	k.IterateOutgoingTXBatches(sdk.UnwrapSDKContext(c), req.HyperionId, func(_ []byte, batch *types.OutgoingTxBatch) (stop bool) {
		foundConfirm := k.GetBatchConfirm(sdk.UnwrapSDKContext(c), batch.HyperionId, batch.BatchNonce, common.HexToAddress(batch.TokenContract), addr) != nil
		if !foundConfirm {
			pendingBatchReqs = append(pendingBatchReqs, batch)
		}
		if len(pendingBatchReqs) == MaxResults {
			return true
		}
		return false
	})

	return &types.QueryLastPendingBatchsRequestByAddrResponse{Batchs: pendingBatchReqs}, nil
}

// [Used In Hyperion] OutgoingTxBatches queries the OutgoingTxBatches of the hyperion module
func (k *Keeper) OutgoingTxBatches(c context.Context, req *types.QueryOutgoingTxBatchesRequest) (*types.QueryOutgoingTxBatchesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	MaxResults := 100 // todo: impl pagination

	batches := make([]*types.OutgoingTxBatch, 0)
	k.IterateOutgoingTXBatches(sdk.UnwrapSDKContext(c), req.HyperionId, func(_ []byte, batch *types.OutgoingTxBatch) bool {
		batches = append(batches, batch)
		return len(batches) == MaxResults
	})

	return &types.QueryOutgoingTxBatchesResponse{Batches: batches}, nil
}

func (k *Keeper) OutgoingTxBatchesCount(c context.Context, req *types.QueryOutgoingTxBatchesCountRequest) (*types.QueryOutgoingTxBatchesCountResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	MaxResults := uint64(100) // todo: impl pagination

	txCount := uint64(0)
	batchCount := uint64(0)

	k.IterateOutgoingTXBatches(sdk.UnwrapSDKContext(c), req.HyperionId, func(_ []byte, batch *types.OutgoingTxBatch) bool {
		batchCount++
		txCount += uint64(len(batch.Transactions))
		return batchCount == MaxResults
	})

	return &types.QueryOutgoingTxBatchesCountResponse{TxCount: txCount, BatchCount: batchCount}, nil
}

// [Used In Hyperion] OutgoingExternalDataTxs queries the OutgoingExternalDataTxs of the hyperion module
func (k *Keeper) OutgoingExternalDataTxs(c context.Context, req *types.QueryOutgoingExternalDataTxsRequest) (*types.QueryOutgoingExternalDataTxsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	MaxResults := 100 // todo: impl pagination

	txs := make([]*types.OutgoingExternalDataTx, 0)
	k.IterateOutgoingExternalDataTXs(sdk.UnwrapSDKContext(c), req.HyperionId, func(_ []byte, tx *types.OutgoingExternalDataTx) bool {
		txs = append(txs, tx)
		return len(txs) == MaxResults
	})

	return &types.QueryOutgoingExternalDataTxsResponse{Txs: txs}, nil
}

// [Used In Hyperion] BatchConfirms returns the batch confirmations by nonce and token contract
func (k *Keeper) BatchConfirms(c context.Context, req *types.QueryBatchConfirmsRequest) (*types.QueryBatchConfirmsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	confirms := make([]*types.MsgConfirmBatch, 0)
	k.IterateBatchConfirmByNonceAndTokenContract(sdk.UnwrapSDKContext(c), req.HyperionId, req.Nonce, common.HexToAddress(req.ContractAddress),
		func(_ []byte, batch *types.MsgConfirmBatch) (stop bool) {
			confirms = append(confirms, batch)
			return false
		})

	return &types.QueryBatchConfirmsResponse{Confirms: confirms}, nil
}

// [Used In Hyperion] LastEventByAddr returns the last event for the given validator address, this allows eth oracles to figure out where they left off
func (k *Keeper) LastEventByAddr(c context.Context, req *types.QueryLastEventByAddrRequest) (*types.QueryLastEventByAddrResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	var ret types.QueryLastEventByAddrResponse

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrInvalidAddress, req.Address)
	}

	validator, found := k.GetOrchestratorValidator(ctx, req.HyperionId, addr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "address")
	}

	lastClaimEvent := k.GetLastEventByValidatorAndHyperionId(ctx, req.HyperionId, validator)
	if lastClaimEvent.EthereumEventNonce == 0 && lastClaimEvent.EthereumEventHeight == 0 {
		// if hyperion happens to query too early without a bonded validator even existing setup the base event
		lowestObservedNonce := k.GetLastObservedEventNonce(ctx, req.HyperionId)
		blockHeight := k.GetLastObservedEthereumBlockHeight(ctx, req.HyperionId).EthereumBlockHeight

		k.setLastEventByValidatorAndHyperionId(
			ctx,
			req.HyperionId,
			validator,
			lowestObservedNonce,
			blockHeight,
		)
		lastClaimEvent = k.GetLastEventByValidatorAndHyperionId(ctx, req.HyperionId, validator)
	}

	ret.LastClaimEvent = &lastClaimEvent

	return &ret, nil
}

// [Used In Hyperion]
func (k *Keeper) GetDelegateKeyByEth(c context.Context, req *types.QueryDelegateKeysByEthAddress) (*types.QueryDelegateKeysByEthAddressResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	keys := k.GetOrchestratorAddresses(ctx, req.HyperionId)
	if err := types.ValidateEthAddress(req.EthAddress); err != nil {
		return nil, errors.Wrap(err, "invalid eth address")
	}

	for _, key := range keys {
		if req.EthAddress == key.EthAddress {
			return &types.QueryDelegateKeysByEthAddressResponse{
				ValidatorAddress:    key.Sender,
				OrchestratorAddress: key.Orchestrator}, nil
		}
	}

	metrics.ReportFuncError(k.svcTags)
	return nil, errors.Wrap(types.ErrInvalid, "No validator")
}

func (k *Keeper) QueryGetDelegateKeysByAddress(c context.Context, req *types.QueryGetDelegateKeysByAddressRequest) (*types.QueryGetDelegateKeysByAddressResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)
	chainWhereKeyIsRegistered := make([]uint64, 0)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		keys := k.GetOrchestratorAddresses(ctx, counterpartyChainParam.HyperionId)
		for _, key := range keys {
			if common.HexToAddress(key.EthAddress).Hex() == common.HexToAddress(req.EthAddress).Hex() {
				chainWhereKeyIsRegistered = append(chainWhereKeyIsRegistered, counterpartyChainParam.BridgeChainId)
			}
		}
	}

	return &types.QueryGetDelegateKeysByAddressResponse{
		ChainIds: chainWhereKeyIsRegistered,
	}, nil
}

func (k *Keeper) QueryGetLastObservedEthereumBlockHeight(c context.Context, req *types.QueryGetLastObservedEthereumBlockHeightRequest) (*types.QueryGetLastObservedEthereumBlockHeightResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	lastObservedHeight := k.GetLastObservedEthereumBlockHeight(ctx, req.HyperionId)

	return &types.QueryGetLastObservedEthereumBlockHeightResponse{
		LastObservedHeight: &lastObservedHeight,
	}, nil
}

func (k *Keeper) QueryGetLastObservedEventNonce(c context.Context, req *types.QueryGetLastObservedEventNonceRequest) (*types.QueryGetLastObservedEventNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	lastObservedEventNonce := k.GetLastObservedEventNonce(ctx, req.HyperionId)

	return &types.QueryGetLastObservedEventNonceResponse{
		LastObservedEventNonce: lastObservedEventNonce,
	}, nil
}

func (k *Keeper) QueryGetTokensOfChain(c context.Context, req *types.QueryGetTokensOfChainRequest) (*types.QueryGetTokensOfChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	pageReq := &query.PageRequest{
		Offset:     uint64((req.Page - 1) * uint64(req.Size_)),
		Limit:      uint64(req.Size_),
		CountTotal: true,
	}

	res, err := k.bankKeeper.DenomsByChainId(ctx, &banktypes.QueryDenomsByChainIdRequest{
		ChainId:             req.ChainId,
		Pagination:          pageReq,
		OrderByHoldersCount: true,
	})

	if err != nil {
		return nil, err
	}

	formattedTokens := make([]*types.FullMetadataToken, 0)

	for _, token := range res.Metadatas {
		formattedTokens = append(formattedTokens, &types.FullMetadataToken{
			Metadata:     token.Metadata,
			HoldersCount: token.HoldersCount,
			TotalSupply:  token.TotalSupply,
		})
	}

	return &types.QueryGetTokensOfChainResponse{
		Tokens:     formattedTokens,
		Pagination: res.Pagination,
	}, nil
}
