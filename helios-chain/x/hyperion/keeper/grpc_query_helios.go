package keeper

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/samber/lo/mutable"

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

// [Not Used In Hyperion] DenomToTokenAddress queries the Cosmos Denom that maps to an Ethereum ERC20
func (k *Keeper) DenomToTokenAddress(c context.Context, req *types.QueryDenomToTokenAddressRequest) (*types.QueryDenomToTokenAddressResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	tokenAddressToDenom, exists := k.GetTokenFromDenom(ctx, req.HyperionId, req.Denom)
	if !exists {
		return nil, errors.Wrap(types.ErrInvalid, "token not found")
	}

	var ret types.QueryDenomToTokenAddressResponse
	ret.TokenAddress = tokenAddressToDenom.TokenAddress
	ret.CosmosOriginated = tokenAddressToDenom.IsCosmosOriginated

	return &ret, nil
}

// [Not Used In Hyperion] TokenAddressToDenom queries the ERC20 contract that maps to an Ethereum ERC20 if any
func (k *Keeper) TokenAddressToDenom(c context.Context, req *types.QueryTokenAddressToDenomRequest) (*types.QueryTokenAddressToDenomResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.grpcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	tokenAddressToDenom, exists := k.GetTokenFromAddress(ctx, req.HyperionId, common.HexToAddress(req.TokenAddress))
	if !exists { // simulate the case where the token address to denom is not found
		hyperionDenom := types.NewHyperionDenom(req.HyperionId, common.HexToAddress(req.TokenAddress))

		tokenAddressToDenom = &types.TokenAddressToDenom{
			TokenAddress:       common.HexToAddress(req.TokenAddress).String(),
			Denom:              hyperionDenom,
			IsCosmosOriginated: false,
		}
	}

	var ret types.QueryTokenAddressToDenomResponse
	ret.Denom = tokenAddressToDenom.Denom
	ret.CosmosOriginated = tokenAddressToDenom.IsCosmosOriginated

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
		if batch.HyperionId == req.HyperionId {
			res.TransfersInBatches = append(res.TransfersInBatches, batch.Transactions...)
		}
	}
	for _, tx := range unbatchedTx {
		if tx.HyperionId == req.HyperionId {
			res.UnbatchedTransfers = append(res.UnbatchedTransfers, tx)
		}
	}

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

func formatErc20TransferTxs(ctx sdk.Context, k *Keeper, txs []*types.QueryTransferTx) []*types.QueryTransferTx {
	tokenPairHLS, existsHLS := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, sdk.DefaultBondDenom))
	for _, tx := range txs {
		if tx.ReceivedToken == nil {
			tx.ReceivedToken = &types.Token{
				Amount:   math.NewInt(0),
				Contract: sdk.DefaultBondDenom,
			}
		}
		if tx.SentToken == nil {
			tx.SentToken = &types.Token{
				Amount:   math.NewInt(0),
				Contract: sdk.DefaultBondDenom,
			}
		}
		if tx.SentFee == nil {
			tx.SentFee = &types.Token{
				Amount:   math.NewInt(0),
				Contract: sdk.DefaultBondDenom,
			}
		}
		if tx.ReceivedFee == nil {
			tx.ReceivedFee = &types.Token{
				Amount:   math.NewInt(0),
				Contract: sdk.DefaultBondDenom,
			}
		}
		if tx.SentToken.Contract != "" {
			tokenPair, exists := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, tx.SentToken.Contract))
			if exists {
				tx.SentToken.Contract = tokenPair.Erc20Address
			}
		}
		if tx.ReceivedToken.Contract != "" {
			tokenPair, exists := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, tx.ReceivedToken.Contract))
			if exists {
				tx.ReceivedToken.Contract = tokenPair.Erc20Address
			}
		}
		if existsHLS {
			tx.SentFee.Contract = tokenPairHLS.Erc20Address
			tx.ReceivedFee.Contract = tokenPairHLS.Erc20Address
		}
	}
	return txs
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
	txs := make([]*types.QueryTransferTx, 0)
	currentCount := uint64(0) // Track how many transactions we've counted

	// 1. HIGHEST PRIORITY: Process incoming transactions from attestations
	params := k.GetParams(ctx)

	// Collect all incoming transactions first (to process newest first)
	incomingTxs := make([]*types.QueryTransferTx, 0)

	// 1. First PRIORITY: Process outgoing transactions (unbatched)
	// Adjust indices based on incoming transactions
	remainingSlots := req.Pagination.Limit - uint64(len(txs))
	if remainingSlots > 0 {
		// Get outgoing transactions
		allOuts := k.GetAllPoolTransactions(ctx)
		// Filter by address and convert to TransferTx
		outgoingTxs := make([]*types.QueryTransferTx, 0)
		for _, tx := range allOuts {
			if cmn.AnyToHexAddress(tx.Sender).String() == req.Address || req.Address == "" {
				receivedTokenToDenom := ""
				if tx.Token.Contract != "" {
					denom, _ := k.GetTokenFromAddress(ctx, tx.HyperionId, common.HexToAddress(tx.Token.Contract))
					receivedTokenToDenom = denom.Denom
				}

				outgoingTxs = append(outgoingTxs, &types.QueryTransferTx{
					HyperionId:  tx.HyperionId,
					Id:          tx.Id,
					Sender:      cmn.AnyToHexAddress(tx.Sender).String(),
					DestAddress: cmn.AnyToHexAddress(tx.DestAddress).String(),
					ReceivedToken: &types.Token{
						Amount:   tx.Token.Amount,
						Contract: receivedTokenToDenom,
					},
					ReceivedFee: &types.Token{
						Amount:   tx.Fee.Amount,
						Contract: sdk.DefaultBondDenom,
					},
					SentToken: &types.Token{
						Amount:   tx.Token.Amount,
						Contract: receivedTokenToDenom,
					},
					SentFee: &types.Token{
						Amount:   tx.Fee.Amount,
						Contract: sdk.DefaultBondDenom,
					},
					Status:    "PROGRESS_UNBATCHED",
					Direction: "OUT",
					ChainId:   k.GetBridgeChainID(ctx)[tx.HyperionId],
					Height:    uint64(ctx.BlockHeight()),
					Proof:     &types.Proof{},
					TxHash:    tx.TxHash,
					Timeout:   tx.TxTimeout,
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
				break
			}
		}
	}

	// 2. SECOND PRIORITY: Process batched transactions
	remainingSlots = req.Pagination.Limit - uint64(len(txs))
	if remainingSlots > 0 {
		// Get outgoing transactions
		batches := k.GetAllOutgoingTxBatches(ctx)

		outTransfersInBatches := make([]*types.OutgoingTransferTx, 0)
		for _, batch := range batches {
			outTransfersInBatches = append(outTransfersInBatches, batch.Transactions...)
		}
		allOuts := outTransfersInBatches

		// Filter by address and convert to TransferTx
		outgoingTxs := make([]*types.QueryTransferTx, 0)
		for _, tx := range allOuts {
			if cmn.AnyToHexAddress(tx.Sender).String() == req.Address || req.Address == "" {
				receivedTokenToDenom := ""
				if tx.Token.Contract != "" {
					denom, _ := k.GetTokenFromAddress(ctx, tx.HyperionId, common.HexToAddress(tx.Token.Contract))
					receivedTokenToDenom = denom.Denom
				}

				outgoingTxs = append(outgoingTxs, &types.QueryTransferTx{
					HyperionId:  tx.HyperionId,
					Id:          tx.Id,
					Sender:      cmn.AnyToHexAddress(tx.Sender).String(),
					DestAddress: cmn.AnyToHexAddress(tx.DestAddress).String(),
					ReceivedToken: &types.Token{
						Amount:   tx.Token.Amount,
						Contract: receivedTokenToDenom,
					},
					ReceivedFee: &types.Token{
						Amount:   tx.Fee.Amount,
						Contract: sdk.DefaultBondDenom,
					},
					SentToken: &types.Token{
						Amount:   tx.Token.Amount,
						Contract: receivedTokenToDenom,
					},
					SentFee: &types.Token{
						Amount:   tx.Fee.Amount,
						Contract: sdk.DefaultBondDenom,
					},
					Status:    "PROGRESS_BATCHED",
					Direction: "OUT",
					ChainId:   k.GetBridgeChainID(ctx)[tx.HyperionId],
					Height:    uint64(ctx.BlockHeight()),
					Proof:     &types.Proof{},
					TxHash:    tx.TxHash,
					Timeout:   tx.TxTimeout,
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
				break
			}
		}
	}

	// 3. THIRD PRIORITY: Process incoming transactions
	remainingSlots = req.Pagination.Limit - uint64(len(txs))
	if remainingSlots > 0 {
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

					receivedTokenToDenom := ""
					if claim.TokenContract != "" {
						denom, _ := k.GetTokenFromAddress(ctx, claim.HyperionId, common.HexToAddress(claim.TokenContract))
						receivedTokenToDenom = denom.Denom
					}

					incomingTxs = append(incomingTxs, &types.QueryTransferTx{
						HyperionId:  claim.HyperionId,
						Id:          claim.EventNonce,
						Height:      claim.BlockHeight,
						Sender:      cmn.AnyToHexAddress(claim.EthereumSender).String(),
						DestAddress: cmn.AnyToHexAddress(claim.CosmosReceiver).String(),
						SentToken: &types.Token{
							Amount:   claim.Amount,
							Contract: claim.TokenContract,
						},
						SentFee: &types.Token{
							Amount:   math.NewInt(0),
							Contract: sdk.DefaultBondDenom,
						},
						ReceivedToken: &types.Token{
							Amount:   claim.Amount,
							Contract: receivedTokenToDenom,
						},
						ReceivedFee: &types.Token{
							Amount:   math.NewInt(0),
							Contract: sdk.DefaultBondDenom,
						},
						Status:    status,
						Direction: "IN",
						ChainId:   counterpartyChainParam.BridgeChainId,
						Proof:     proof,
						TxHash:    claim.TxHash,
						Timeout:   0,
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
				break
			}
		}
	}

	// 4. LOWEST PRIORITY: Get finalized transactions
	// Adjust indices based on prior transactions
	remainingSlots = req.Pagination.Limit - uint64(len(txs))
	if remainingSlots > 0 && req.Address != "" {
		lastIndex, err := k.FindLastFinalizedTxIndex(ctx, cmn.AnyToHexAddress(req.Address))
		if err != nil {
			return nil, errors.Wrap(err, "failed to find last finalized tx index")
		}
		finalizedStartIndex := lastIndex - remainingSlots

		if lastIndex < remainingSlots {
			finalizedStartIndex = 0
		}

		finalizedTxs, err := k.FindFinalizedTxsByIndexToIndex(ctx, common.HexToAddress(req.Address), finalizedStartIndex, lastIndex)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find finalized txs")
		}

		// reverse the txs
		mutable.Reverse(finalizedTxs)

		txs = append(txs, formatQueryTransferTxs(finalizedTxs)...)
	} else if req.Address == "" {
		finalizedTxs, err := k.GetLastFinalizedTxIndex(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to search finalized txs")
		}
		mutable.Reverse(finalizedTxs.Txs)
		txs = append(txs, formatQueryTransferTxs(finalizedTxs.Txs)...)
	}

	// Format ERC20 tokens if requested
	if req.FormatErc20 {
		formatErc20TransferTxs(ctx, k, txs)
	}

	return &types.QueryGetTransactionsByPageAndSizeResponse{
		Txs: txs,
	}, nil
}

func formatQueryTransferTxs(txs []*types.TransferTx) []*types.QueryTransferTx {
	queryTxs := make([]*types.QueryTransferTx, 0)
	for _, tx := range txs {
		queryTxs = append(queryTxs, &types.QueryTransferTx{
			HyperionId:    tx.HyperionId,
			Id:            tx.Id,
			Height:        tx.Height,
			Sender:        tx.Sender,
			DestAddress:   tx.DestAddress,
			SentToken:     tx.SentToken,
			SentFee:       tx.SentFee,
			ReceivedToken: tx.ReceivedToken,
			ReceivedFee:   tx.ReceivedFee,
			Status:        tx.Status,
			Direction:     tx.Direction,
			ChainId:       tx.ChainId,
			Proof:         tx.Proof,
			TxHash:        tx.TxHash,
			Timeout:       0,
		})
	}
	return queryTxs
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

func (k *Keeper) QueryGetRpcListByChainId(c context.Context, req *types.QueryGetRpcListByChainIdRequest) (*types.QueryGetRpcListByChainIdResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == req.ChainId {
			return &types.QueryGetRpcListByChainIdResponse{
				Rpcs: counterpartyChainParam.Rpcs,
			}, nil
		}
	}

	return &types.QueryGetRpcListByChainIdResponse{
		Rpcs: make([]*types.Rpc, 0),
	}, nil
}

func (k *Keeper) QueryHistoricalFees(c context.Context, req *types.QueryHistoricalFeesRequest) (*types.QueryHistoricalFeesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	lowestFeeValidator := k.GetLowestFeeValidator(ctx, req.HyperionId)
	lowestFee := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(-1)}
	if lowestFeeValidator != nil {
		lowestFee = k.GetFeeByValidator(ctx, req.HyperionId, *lowestFeeValidator)
	} else {
		lowestFee = &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(-1)}
	}

	highestFeeValidator := k.GetHighestFeeValidator(ctx, req.HyperionId)
	highestFee := &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(-1)}
	if highestFeeValidator != nil {
		highestFee = k.GetFeeByValidator(ctx, req.HyperionId, *highestFeeValidator)
	} else {
		highestFee = &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(-1)}
	}

	averageFee := sdk.Coin{}
	averageFee.Amount = lowestFee.Amount.Add(highestFee.Amount).Quo(math.NewInt(2))

	lastFinalizedTxs, err := k.GetLastFinalizedTxIndex(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find last finalized tx index")
	}

	historicalFees := make([]*sdk.Coin, 0)

	for _, tx := range lastFinalizedTxs.Txs {
		if tx.Direction == "OUT" && tx.HyperionId == req.HyperionId {
			historicalFees = append(historicalFees, &sdk.Coin{
				Denom:  sdk.DefaultBondDenom,
				Amount: tx.SentFee.Amount,
			})
		}
	}

	sort.Slice(historicalFees, func(i, j int) bool {
		return historicalFees[i].Amount.GT(historicalFees[j].Amount)
	})

	if lowestFee.Amount.LTE(math.NewInt(0)) && highestFee.Amount.LTE(math.NewInt(0)) && averageFee.Amount.LTE(math.NewInt(0)) && len(historicalFees) > 0 {
		// calculate average fee from historical fees
		averageFee = sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(0)}
		for _, fee := range historicalFees {
			averageFee.Amount = averageFee.Amount.Add(fee.Amount)
		}
		averageFee.Amount = averageFee.Amount.Quo(math.NewInt(int64(len(historicalFees))))
		// set lowest fee to 25% lower than average fee
		lowestFee = &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: averageFee.Amount.Mul(math.NewInt(75)).Quo(math.NewInt(100))}
		// set highest fee to 25% higher than average fee
		highestFee = &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: averageFee.Amount.Mul(math.NewInt(125)).Quo(math.NewInt(100))}
	}

	if lowestFee.Amount.LTE(math.NewInt(0)) && highestFee.Amount.LTE(math.NewInt(0)) && averageFee.Amount.LTE(math.NewInt(0)) && len(historicalFees) == 0 {
		// set lowest fee to 0. HLS will be used as default fee
		lowestFee = &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(250000000000000000)}
		highestFee = &sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(750000000000000000)}
		averageFee = sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: math.NewInt(500000000000000000)}
	}

	return &types.QueryHistoricalFeesResponse{
		HistoricalFees: historicalFees,
		Low:            lowestFee,
		High:           highestFee,
		Average:        &averageFee,
	}, nil
}

func (k *Keeper) QueryGetOrchestratorsData(c context.Context, req *types.QueryGetOrchestratorsDataRequest) (*types.QueryGetOrchestratorsDataResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	orchestratorsData := make([]*types.OrchestratorData, 0)

	for _, orchestrator := range req.Orchestrators {
		orchestratorAddr := cmn.AnyToHexAddress(orchestrator)
		orchestratorAddrAcc := cmn.AccAddressFromHexAddress(orchestratorAddr)
		orchestratorData, err := k.GetOrchestratorData(ctx, orchestratorAddrAcc)
		if err != nil {
			orchestratorsData = append(orchestratorsData, &types.OrchestratorData{
				Orchestrator:             orchestrator,
				OrchestratorHyperionData: make([]*types.OrchestratorHyperionData, 0),
			})
			continue
		}
		orchestratorsData = append(orchestratorsData, orchestratorData)
	}

	return &types.QueryGetOrchestratorsDataResponse{
		OrchestratorsData: orchestratorsData,
	}, nil
}

func (k *Keeper) QueryGetOrchestratorData(c context.Context, req *types.QueryGetOrchestratorDataRequest) (*types.QueryGetOrchestratorDataResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	orchestratorAddr := cmn.AnyToHexAddress(req.OrchestratorAddress)
	orchestratorAddrAcc := cmn.AccAddressFromHexAddress(orchestratorAddr)
	orchestratorData, err := k.GetOrchestratorData(ctx, orchestratorAddrAcc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get orchestrator data")
	}

	return &types.QueryGetOrchestratorDataResponse{
		OrchestratorData: orchestratorData,
	}, nil
}

func (k *Keeper) QueryGetOrchestratorHyperionData(c context.Context, req *types.QueryGetOrchestratorHyperionDataRequest) (*types.QueryGetOrchestratorHyperionDataResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	orchestratorAddr := cmn.AnyToHexAddress(req.OrchestratorAddress)
	orchestratorAddrAcc := cmn.AccAddressFromHexAddress(orchestratorAddr)

	orchestratorData, err := k.GetOrchestratorHyperionData(ctx, orchestratorAddrAcc, req.HyperionId)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get orchestrator hyperion data")
	}

	return &types.QueryGetOrchestratorHyperionDataResponse{
		OrchestratorHyperionData: orchestratorData,
	}, nil
}

func (k *Keeper) QueryGetWhitelistedAddresses(c context.Context, req *types.QueryGetWhitelistedAddressesRequest) (*types.QueryGetWhitelistedAddressesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	whitelistedAddresses := k.GetWhitelistedAddresses(ctx, req.HyperionId)
	if whitelistedAddresses == nil {
		return nil, errors.Wrap(types.ErrInvalid, "WhitelistedAddresses not found")
	}

	return &types.QueryGetWhitelistedAddressesResponse{Addresses: whitelistedAddresses.Addresses}, nil
}

func (k *Keeper) QueryEstimateLatestBlockOfChain(c context.Context, req *types.QueryEstimateLatestBlockOfChainRequest) (*types.QueryEstimateLatestBlockOfChainResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryEstimateLatestBlockOfChainResponse{LatestBlock: k.GetProjectedCurrentEthereumHeight(ctx, req.HyperionId)}, nil
}

func (k *Keeper) QueryIsNonceAlreadyObserved(c context.Context, req *types.QueryIsNonceAlreadyObservedRequest) (*types.QueryIsNonceAlreadyObservedResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	return &types.QueryIsNonceAlreadyObservedResponse{IsNonceAlreadyObserved: k.NonceAlreadyObserved(ctx, req.HyperionId, req.Nonce)}, nil
}

func (k *Keeper) QueryGetUnObservedNonces(c context.Context, req *types.QueryGetUnObservedNoncesRequest) (*types.QueryGetUnObservedNoncesResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetHyperionParamsFromChainId(ctx, req.HyperionId)

	if params.HyperionId != req.HyperionId {
		return nil, errors.Wrap(types.ErrInvalid, "hyperionId not found")
	}
	latestObservedNonce := k.GetLastObservedEventNonce(ctx, req.HyperionId)

	unObservedNonces := make([]uint64, 0)
	for nonce := req.StartNonce; nonce <= req.EndNonce; nonce++ {
		if nonce > latestObservedNonce {
			break
		}
		if !k.NonceAlreadyObserved(ctx, req.HyperionId, nonce) {
			unObservedNonces = append(unObservedNonces, nonce)
		}
	}

	return &types.QueryGetUnObservedNoncesResponse{
		UnObservedNonces: unObservedNonces,
	}, nil
}
