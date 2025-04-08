package keeper

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Helios-Chain-Labs/metrics"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/hyperion/types"
)

// [Not Used In Hyperion] ValsetConfirm queries the ValsetConfirm of the hyperion module
func (k *Keeper) ValsetConfirm(c context.Context, req *types.QueryValsetConfirmRequest) (*types.QueryValsetConfirmResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	addr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "address invalid")
	}

	return &types.QueryValsetConfirmResponse{Confirm: k.GetValsetConfirm(sdk.UnwrapSDKContext(c), req.HyperionId, req.Nonce, addr)}, nil
}

// [Not Used In Hyperion] BatchRequestByNonce queries the BatchRequestByNonce of the hyperion module
func (k *Keeper) BatchRequestByNonce(c context.Context, req *types.QueryBatchRequestByNonceRequest) (*types.QueryBatchRequestByNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	if err := types.ValidateEthAddress(req.ContractAddress); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrUnknownRequest, err.Error())
	}

	foundBatch := k.GetOutgoingTXBatch(sdk.UnwrapSDKContext(c), common.HexToAddress(req.ContractAddress), req.Nonce, req.HyperionId)
	if foundBatch == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrUnknownRequest, "Can not find tx batch")
	}

	return &types.QueryBatchRequestByNonceResponse{Batch: foundBatch}, nil
}

// [Not Used In Hyperion] DenomToERC20 queries the Cosmos Denom that maps to an Ethereum ERC20
func (k *Keeper) DenomToERC20(c context.Context, req *types.QueryDenomToERC20Request) (*types.QueryDenomToERC20Response, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	cosmosOriginated, erc20, err := k.DenomToERC20Lookup(ctx, req.Denom, req.HyperionId)

	var ret types.QueryDenomToERC20Response
	ret.Erc20 = erc20.Hex()
	ret.CosmosOriginated = cosmosOriginated

	return &ret, err
}

// [Not Used In Hyperion] ERC20ToDenom queries the ERC20 contract that maps to an Ethereum ERC20 if any
func (k *Keeper) ERC20ToDenom(c context.Context, req *types.QueryERC20ToDenomRequest) (*types.QueryERC20ToDenomResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	cosmosOriginated, name := k.ERC20ToDenomLookup(ctx, common.HexToAddress(req.Erc20), req.HyperionId)

	var ret types.QueryERC20ToDenomResponse
	ret.Denom = name
	ret.CosmosOriginated = cosmosOriginated

	return &ret, nil
}

// [Not Used In Hyperion]
func (k *Keeper) GetDelegateKeyByValidator(c context.Context, req *types.QueryDelegateKeysByValidatorAddress) (*types.QueryDelegateKeysByValidatorAddressResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	valAddress, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	valAccountAddr := sdk.AccAddress(valAddress.Bytes())
	keys := k.GetOrchestratorAddresses(ctx, req.HyperionId)

	for _, key := range keys {
		senderAddr, err := sdk.AccAddressFromBech32(key.Sender)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
		if valAccountAddr.Equals(senderAddr) {
			return &types.QueryDelegateKeysByValidatorAddressResponse{EthAddress: key.EthAddress, OrchestratorAddress: key.Orchestrator}, nil
		}
	}

	metrics.ReportFuncError(k.svcTags)
	return nil, errors.Wrap(types.ErrInvalid, "No validator")
}

// [Not Used In Hyperion]
func (k *Keeper) GetDelegateKeyByOrchestrator(c context.Context, req *types.QueryDelegateKeysByOrchestratorAddress) (*types.QueryDelegateKeysByOrchestratorAddressResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	keys := k.GetOrchestratorAddresses(ctx, req.HyperionId)

	_, err := sdk.AccAddressFromBech32(req.OrchestratorAddress)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	for _, key := range keys {
		if req.OrchestratorAddress == key.Orchestrator {
			return &types.QueryDelegateKeysByOrchestratorAddressResponse{ValidatorAddress: key.Sender, EthAddress: key.EthAddress}, nil
		}
	}

	metrics.ReportFuncError(k.svcTags)
	return nil, errors.Wrap(types.ErrInvalid, "No validator")
}

// [Not Used In Hyperion]
func (k *Keeper) GetPendingSendToChain(c context.Context, req *types.QueryPendingSendToChain) (*types.QueryPendingSendToChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	batches := k.GetOutgoingTxBatches(ctx, req.HyperionId)
	unbatchedTx := k.GetPoolTransactions(ctx, req.HyperionId)
	senderAddress := req.SenderAddress

	res := &types.QueryPendingSendToChainResponse{}
	res.TransfersInBatches = make([]*types.OutgoingTransferTx, 0)
	res.UnbatchedTransfers = make([]*types.OutgoingTransferTx, 0)

	for _, batch := range batches {
		for _, tx := range batch.Transactions {
			if tx.Sender == senderAddress {
				res.TransfersInBatches = append(res.TransfersInBatches, tx)
			}
		}
	}

	for _, tx := range unbatchedTx {
		if strings.EqualFold(tx.Sender, senderAddress) {
			res.UnbatchedTransfers = append(res.UnbatchedTransfers, tx)

		}
	}

	return res, nil
}

// [Not Used In Hyperion]
func (k *Keeper) GetAllPendingSendToChain(c context.Context, req *types.QueryAllPendingSendToChainRequest) (*types.QueryAllPendingSendToChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	batches := k.GetAllOutgoingTxBatches(ctx)
	unbatchedTx := k.GetAllPoolTransactions(ctx)

	res := &types.QueryAllPendingSendToChainResponse{}
	res.TransfersInBatches = make([]*types.OutgoingTransferTx, 0)
	res.UnbatchedTransfers = make([]*types.OutgoingTransferTx, 0)

	for _, batch := range batches {
		res.TransfersInBatches = append(res.TransfersInBatches, batch.Transactions...)
	}
	res.UnbatchedTransfers = append(res.UnbatchedTransfers, unbatchedTx...)

	return res, nil
}

// [Not Used In Hyperion]
func (k *Keeper) HyperionModuleState(c context.Context, req *types.QueryModuleStateRequest) (*types.QueryModuleStateResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	state := ExportGenesis(ctx, *k)

	res := &types.QueryModuleStateResponse{
		State: &state,
	}

	return res, nil
}

// [Not Used In Hyperion]
func (k *Keeper) MissingHyperionNonces(c context.Context, req *types.MissingNoncesRequest) (*types.MissingNoncesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	var res []string

	bondedValidators, err := k.StakingKeeper.GetBondedValidatorsByPower(ctx)
	if err != nil {
		return nil, err
	}
	for i := range bondedValidators {
		val, _ := sdk.ValAddressFromBech32(bondedValidators[i].GetOperator())

		ev := k.GetLastEventByValidatorAndHyperionId(ctx, req.HyperionId, val)
		if ev.EthereumEventNonce == 0 && ev.EthereumEventHeight == 0 {
			res = append(res, bondedValidators[i].GetOperator())
		}
	}

	return &types.MissingNoncesResponse{OperatorAddresses: res}, nil
}

// [Not Used In Hyperion]
func (k *Keeper) GetHyperionIdFromChainId(c context.Context, req *types.QueryGetHyperionIdFromChainIdRequest) (*types.QueryGetHyperionIdFromChainIdResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetHyperionParamsFromChainId(ctx, req.ChainId)

	if params != nil {
		return &types.QueryGetHyperionIdFromChainIdResponse{
			HyperionId: params.HyperionId,
		}, nil
	}

	return nil, errors.Wrap(types.ErrDuplicate, "BridgeChainId not found")
}

// [Not Used In Hyperion]
func (k Keeper) Attestation(c context.Context, req *types.QueryAttestationRequest) (*types.QueryAttestationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	att := k.GetAttestation(ctx, req.HyperionId, req.Nonce, req.ClaimHash)
	if att == nil {
		return nil, status.Error(codes.NotFound, "attestation not found")
	}

	return &types.QueryAttestationResponse{
		Attestation: att,
	}, nil
}

func (k *Keeper) QueryGetTransactionsByPageAndSize(c context.Context, req *types.QueryGetTransactionsByPageAndSizeRequest) (*types.QueryGetTransactionsByPageAndSizeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	// Validate pagination parameters
	if req.Pagination == nil {
		return nil, errors.Wrap(types.ErrInvalid, "pagination is required")
	}

	startIndex := req.Pagination.Offset
	endIndex := req.Pagination.Offset + req.Pagination.Limit
	txs := make([]*types.TransferTx, 0)
	currentCount := uint64(0) // Track how many transactions we've counted

	// 1. HIGHEST PRIORITY: Process incoming transactions from attestations
	params := k.GetParams(ctx)

	// Collect all incoming transactions first (to process newest first)
	incomingTxs := make([]*types.TransferTx, 0)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		attestations, err := k.SearchAttestationsByEthereumAddress(ctx, counterpartyChainParam.HyperionId, req.Address)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search attestations by Ethereum address")
		}

		for _, attestation := range attestations {
			claim, err := k.UnpackAttestationClaim(attestation)
			if err != nil {
				return nil, errors.Wrap(err, "failed to unpack attestation claim")
			}

			switch claim := claim.(type) {
			case *types.MsgDepositClaim:
				status := "PROGRESS"
				proof := &types.Proof{}
				if attestation.Observed {
					status = "BRIDGED"

					validators := []string{}
					proofs := []string{}
					for _, validator := range attestation.Votes {
						validatorSplitted := strings.Split(validator, ":")
						validators = append(validators, cmn.AnyToHexAddress(validatorSplitted[0]).String())
						proofs = append(proofs, validatorSplitted[1])
					}
					proof = &types.Proof{
						Orchestrators: strings.Join(validators, ","),
						Hashs:         strings.Join(proofs, ","),
					}
				}

				incomingTxs = append(incomingTxs, &types.TransferTx{
					HyperionId:  claim.HyperionId,
					Id:          claim.EventNonce,
					Height:      claim.BlockHeight,
					Sender:      cmn.AnyToHexAddress(claim.EthereumSender).String(),
					DestAddress: cmn.AnyToHexAddress(claim.CosmosReceiver).String(),
					Erc20Token: &types.ERC20Token{
						Amount:   claim.Amount,
						Contract: claim.TokenContract,
					},
					Erc20Fee: &types.ERC20Token{
						Amount:   math.NewInt(0),
						Contract: claim.TokenContract,
					},
					Status:    status,
					Direction: "IN",
					ChainId:   counterpartyChainParam.BridgeChainId,
					Proof:     proof,
					TxHash:    claim.TxHash,
				})
			}
		}
	}

	// Sort incoming transactions by height (newest first)
	sort.Slice(incomingTxs, func(i, j int) bool {
		return incomingTxs[i].Height > incomingTxs[j].Height
	})

	// Add incoming transactions to result with pagination
	for i, tx := range incomingTxs {
		if uint64(i) >= startIndex && uint64(i) < endIndex {
			txs = append(txs, tx)
		}
		currentCount++

		// If we've filled our page, return early
		if uint64(len(txs)) >= req.Pagination.Limit {
			return &types.QueryGetTransactionsByPageAndSizeResponse{
				Txs: txs,
			}, nil
		}
	}

	// 2. SECOND PRIORITY: Process outgoing transactions
	// Adjust indices based on incoming transactions
	remainingSlots := req.Pagination.Limit - uint64(len(txs))
	if remainingSlots > 0 {
		// Get outgoing transactions
		batches := k.GetAllOutgoingTxBatches(ctx)
		unbatchedTx := k.GetAllPoolTransactions(ctx)

		outTransfersInBatches := make([]*types.OutgoingTransferTx, 0)
		for _, batch := range batches {
			outTransfersInBatches = append(outTransfersInBatches, batch.Transactions...)
		}
		allOuts := append(outTransfersInBatches, unbatchedTx...)

		// Filter by address and convert to TransferTx
		outgoingTxs := make([]*types.TransferTx, 0)
		for _, tx := range allOuts {
			if cmn.AnyToHexAddress(tx.Sender).String() == req.Address {
				outgoingTxs = append(outgoingTxs, &types.TransferTx{
					HyperionId:  tx.HyperionId,
					Id:          tx.Id,
					Sender:      cmn.AnyToHexAddress(tx.Sender).String(),
					DestAddress: cmn.AnyToHexAddress(tx.DestAddress).String(),
					Erc20Token:  tx.Erc20Token,
					Erc20Fee:    tx.Erc20Fee,
					Status:      "PROGRESS",
					Direction:   "OUT",
					ChainId:     k.GetBridgeChainID(ctx)[tx.HyperionId],
					Height:      uint64(ctx.BlockHeight()),
					Proof:       &types.Proof{},
					TxHash:      tx.TxHash,
				})
			}
		}

		// Sort outgoing transactions by ID (newest first)
		sort.Slice(outgoingTxs, func(i, j int) bool {
			return outgoingTxs[i].Id > outgoingTxs[j].Id
		})

		// Calculate adjusted indices for outgoing transactions
		outgoingStartIndex := uint64(0)
		if startIndex > currentCount {
			outgoingStartIndex = startIndex - currentCount
		} else {
			outgoingStartIndex = 0
		}
		outgoingEndIndex := outgoingStartIndex + remainingSlots

		// Add outgoing transactions with pagination
		for i, tx := range outgoingTxs {
			if uint64(i) >= outgoingStartIndex && uint64(i) < outgoingEndIndex {
				txs = append(txs, tx)
			}
			currentCount++

			// If we've filled our page, break
			if uint64(len(txs)) >= req.Pagination.Limit {
				return &types.QueryGetTransactionsByPageAndSizeResponse{
					Txs: txs,
				}, nil
			}
		}
	}

	// 3. LOWEST PRIORITY: Get finalized transactions
	// Adjust indices based on prior transactions
	remainingSlots = req.Pagination.Limit - uint64(len(txs))
	if remainingSlots > 0 {
		// Calculate adjusted indices for finalized transactions
		finalizedStartIndex := uint64(0)
		if startIndex > currentCount {
			finalizedStartIndex = startIndex - currentCount
		} else {
			finalizedStartIndex = 0
		}
		finalizedEndIndex := finalizedStartIndex + remainingSlots

		finalizedTxs, err := k.FindFinalizedTxsByIndexToIndex(ctx, common.HexToAddress(req.Address), finalizedStartIndex+1, finalizedEndIndex+1)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find finalized txs")
		}

		txs = append(txs, finalizedTxs...)
	}

	return &types.QueryGetTransactionsByPageAndSizeResponse{
		Txs: txs,
	}, nil
}

func (k *Keeper) QueryGetCounterpartyChainParamsByChainId(c context.Context, req *types.QueryGetCounterpartyChainParamsByChainIdRequest) (*types.QueryGetCounterpartyChainParamsByChainIdResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetHyperionParamsFromChainId(ctx, req.ChainId)

	if params == nil {
		return nil, errors.Wrap(types.ErrInvalid, "chainId not found "+strconv.FormatUint(req.ChainId, 10))
	}

	return &types.QueryGetCounterpartyChainParamsByChainIdResponse{
		CounterpartyChainParams: params,
	}, nil
}
