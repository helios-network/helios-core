package keeper

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/hyperion/types"

	chronostypes "helios-core/helios-chain/x/chronos/types"

	"github.com/Helios-Chain-Labs/metrics"
)

const OutgoingExternalDataxTxSize = 100

// BuildOutgoingTXBatch starts the following process chain:
//   - find bridged denominator for given voucher type
//   - determine if a an unexecuted batch is already waiting for this token type, if so confirm the new batch would
//     have a higher total fees. If not exit without creating a batch
//   - select available transactions from the outgoing transaction pool sorted by fee desc
//   - persist an outgoing batch object with an incrementing ID = nonce
//   - emit an event
func (k *Keeper) BuildOutgoingExternalDataTX(ctx sdk.Context, hyperionId uint64, cronId string, contractAddress common.Address, abiCall string, sender string, fee *types.Token, timeout uint64) (*types.OutgoingExternalDataTx, error) {
	fmt.Println("BuildOutgoingExternalDataTX for hyperionId: ", hyperionId)
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	nextID := k.AutoIncrementID(ctx, types.GetLastOutgoingBatchIDKey(hyperionId))
	tx := &types.OutgoingExternalDataTx{
		Nonce:                   nextID,
		Id:                      nextID,
		Block:                   uint64(ctx.BlockHeight()),
		Timeout:                 timeout,
		CronId:                  cronId,
		ExternalContractAddress: contractAddress.Hex(),
		AbiCallHex:              abiCall,
		Sender:                  sender,
		Fee:                     fee,
		HyperionId:              hyperionId,
		Claims:                  make([]*types.MsgExternalDataClaim, 0),
		Votes:                   make([]string, 0),
	}
	k.Logger(ctx).Info("StoreExternalData", "tx", tx)
	k.StoreExternalData(ctx, tx)

	return tx, nil
}

// / This gets the batch timeout height in the counterparty chain blocks.
func (k *Keeper) getExternalDataTimeoutHeight(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	counterpartyChainParams := k.GetCounterpartyChainParams(ctx)[hyperionId]
	projectedCurrentEthereumHeight := k.GetProjectedCurrentEthereumHeight(ctx, hyperionId)
	// we convert our target time for block timeouts (lets say 12 hours) into a number of blocks to
	// place on top of our projection of the current Ethereum block height.
	blocksToAdd := counterpartyChainParams.TargetBatchTimeout / counterpartyChainParams.AverageCounterpartyBlockTime

	return projectedCurrentEthereumHeight + blocksToAdd
}

// OutgoingTxBatchExecuted is run when the Cosmos chain detects that a batch has been executed on Ethereum
// It frees all the transactions in the batch, then cancels all earlier batches, this function panics instead
// of returning errors because any failure will cause a double spend.
func (k *Keeper) OutgoingExternalDataTxExecuted(ctx sdk.Context, tx *types.OutgoingExternalDataTx, claim *types.MsgExternalDataClaim, att *types.Attestation) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	contractAddress := common.HexToAddress(tx.ExternalContractAddress)

	// Iterate through remaining txs
	k.IterateOutgoingExternalDataTXs(ctx, tx.HyperionId, func(key []byte, iter_tx *types.OutgoingExternalDataTx) bool {
		// If the iterated txs nonce is lower than the one that was just executed, cancel it
		if iter_tx.HyperionId == tx.HyperionId && iter_tx.Nonce < tx.Nonce && common.HexToAddress(iter_tx.ExternalContractAddress) == contractAddress {
			err := k.CancelOutgoingExternalDataTX(ctx, contractAddress, iter_tx.Nonce, iter_tx.HyperionId)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				panic(fmt.Sprintf("Failed cancel out externalData tx %s %d while trying to execute %s %d with %s", contractAddress, iter_tx.Nonce, contractAddress, tx.Nonce, err))
			}
		}
		return false
	})

	// Delete batch since it is finished
	k.DeleteExternalData(ctx, *tx)

	tokenAddressToDenomFee, _ := k.GetTokenFromAddress(ctx, tx.HyperionId, common.HexToAddress(tx.Fee.Contract))

	tokenAddressFee := ""
	if tokenAddressToDenomFee != nil {
		tokenAddressFee = tokenAddressToDenomFee.Denom
	}

	hashs := []string{}
	for _, claim := range tx.Claims {
		hashs = append(hashs, hex.EncodeToString(claim.ClaimHash()))
	}

	k.StoreFinalizedTx(ctx, &types.TransferTx{
		HyperionId:    tx.HyperionId,
		Id:            tx.Id,
		Sender:        cmn.AnyToHexAddress(tx.Sender).String(),
		DestAddress:   cmn.AnyToHexAddress(tx.ExternalContractAddress).String(),
		SentToken:     nil,
		SentFee:       &types.Token{Amount: tx.Fee.Amount, Contract: tokenAddressFee},
		ReceivedToken: nil,
		ReceivedFee:   tx.Fee,
		Status:        "BRIDGED",
		Direction:     "DATA",
		ChainId:       k.GetBridgeChainID(ctx)[tx.HyperionId],
		Height:        claim.BlockHeight,
		TxHash:        "",
		Proof: &types.Proof{
			Orchestrators: strings.Join(tx.Votes, ";"),
			Hashs:         strings.Join(hashs, ";"),
		},
	})

	k.rewardVotersOfExternalDataTx(ctx, tx.HyperionId, tx, att)

	cronId, _ := strconv.ParseUint(tx.CronId, 10, 64)
	bytesData, err := hex.DecodeString(claim.CallDataResult)
	if err != nil {
		k.chronosKeeper.StoreCronCallBackData(ctx, cronId, &chronostypes.CronCallBackData{
			Data:  []byte{},
			Error: []byte(fmt.Sprintf("parse error: %v", err)),
		})
		return
	}
	errBytesData, err := hex.DecodeString(claim.CallDataResultError)
	if err != nil {
		if len(bytesData) > 0 {
			k.chronosKeeper.StoreCronCallBackData(ctx, cronId, &chronostypes.CronCallBackData{
				Data:  bytesData,
				Error: []byte{},
			})
		} else {
			k.chronosKeeper.StoreCronCallBackData(ctx, cronId, &chronostypes.CronCallBackData{
				Data:  []byte{},
				Error: []byte(fmt.Sprintf("JSON parse error: %v", err)),
			})
		}
		return
	}
	k.chronosKeeper.StoreCronCallBackData(ctx, cronId, &chronostypes.CronCallBackData{
		Data:  bytesData,
		Error: errBytesData,
	})
}

func (k *Keeper) rewardVotersOfExternalDataTx(ctx sdk.Context, hyperionId uint64, tx *types.OutgoingExternalDataTx, att *types.Attestation) {

	pairId := k.erc20Keeper.GetERC20Map(ctx, common.HexToAddress(tx.Fee.Contract))
	tokenPair, ok := k.erc20Keeper.GetTokenPair(ctx, pairId)
	if !ok {
		return
	}

	numberOfVotes := len(att.Votes)
	rewardPerVote := big.NewInt(0).Div(tx.Fee.Amount.BigInt(), big.NewInt(int64(numberOfVotes)))

	for _, vote := range att.Votes {
		voteSplit := strings.Split(vote, ":")
		orchestrator := cmn.AnyToHexAddress(voteSplit[0])
		orchestratorAddr := cmn.AccAddressFromHexAddress(orchestrator)
		k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, orchestratorAddr, sdk.NewCoins(sdk.NewCoin(tokenPair.Denom, sdkmath.NewIntFromBigInt(rewardPerVote))))
		k.Logger(ctx).Info("HYPERION - ABCI.go - rewardVotersOfExternalDataTx -> rewardPerVote", "rewardPerVote", rewardPerVote, "orchestratorAddr", orchestratorAddr)

		// Update orchestrator data
		orchestratorData, err := k.GetOrchestratorHyperionData(ctx, orchestratorAddr, tx.HyperionId)
		if err != nil {
			k.Logger(ctx).Error("failed to get orchestrator data", "error", err, "hyperion_id", tx.HyperionId, "orchestrator", orchestratorAddr)
			continue
		}
		orchestratorData.ExternalDataTxExecuted++
		orchestratorData.ExternalDataTxFeeCollected = orchestratorData.ExternalDataTxFeeCollected.Add(tx.Fee.Amount)
		k.SetOrchestratorHyperionData(ctx, orchestratorAddr, tx.HyperionId, *orchestratorData)
	}
}

// StoreExternalData stores an external data transaction
func (k *Keeper) StoreExternalData(ctx sdk.Context, tx *types.OutgoingExternalDataTx) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetOutgoingExternalDataKey(tx.HyperionId, tx.Nonce, common.HexToAddress(tx.ExternalContractAddress))
	store.Set(key, k.cdc.MustMarshal(tx))

	blockKey := types.GetOutgoingExternalDataBlockKey(tx.HyperionId, tx.Block)
	store.Set(blockKey, k.cdc.MustMarshal(tx))
}

func (k *Keeper) RefundExternalData(ctx sdk.Context, tx types.OutgoingExternalDataTx) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	pairId := k.erc20Keeper.GetERC20Map(ctx, common.HexToAddress(tx.Fee.Contract))
	tokenPair, ok := k.erc20Keeper.GetTokenPair(ctx, pairId)
	if !ok {
		return
	}
	k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, cmn.AccAddressFromHexAddressString(tx.Sender), sdk.NewCoins(sdk.NewCoin(tokenPair.Denom, tx.Fee.Amount)))
	k.Logger(ctx).Info("HYPERION - ABCI.go - RefundExternalData -> refund", "txId", tx.Id, "sender", tx.Sender, "amount", tx.Fee.Amount, "tokenAddressFee", tokenPair.Denom)
}

// DeleteBatch deletes an outgoing transaction batch
func (k *Keeper) DeleteExternalData(ctx sdk.Context, tx types.OutgoingExternalDataTx) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetOutgoingExternalDataKey(tx.HyperionId, tx.Nonce, common.HexToAddress(tx.ExternalContractAddress)))
	store.Delete(types.GetOutgoingExternalDataBlockKey(tx.HyperionId, tx.Block))
}

// GetOutgoingExternalDataTX loads a batch object. Returns nil when not exists.
func (k *Keeper) GetOutgoingExternalDataTX(ctx sdk.Context, contractAddress common.Address, nonce uint64, hyperionId uint64) *types.OutgoingExternalDataTx {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetOutgoingExternalDataKey(hyperionId, nonce, contractAddress)
	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	var b types.OutgoingExternalDataTx
	k.cdc.MustUnmarshal(bz, &b)

	return &b
}

// CancelOutgoingExternalDataTX deletes the external data tx
func (k *Keeper) CancelOutgoingExternalDataTX(ctx sdk.Context, tokenContract common.Address, nonce uint64, hyperionId uint64) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	tx := k.GetOutgoingExternalDataTX(ctx, tokenContract, nonce, hyperionId)
	if tx == nil {
		return types.ErrUnknown
	}

	// Delete batch since it is finished
	k.DeleteExternalData(ctx, *tx)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventOutgoingBatchCanceled{
		HyperionId:     hyperionId,
		BridgeContract: k.GetBridgeContractAddress(ctx)[hyperionId].Hex(),
		BridgeChainId:  k.GetBridgeChainID(ctx)[hyperionId],
		BatchId:        nonce,
		Nonce:          nonce,
	})

	return nil
}

// IterateOutgoingExternalDataTXs iterates through all outgoing external data txs in DESC order.
func (k *Keeper) IterateOutgoingExternalDataTXs(ctx sdk.Context, hyperionId uint64, cb func(key []byte, tx *types.OutgoingExternalDataTx) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.OutgoingExternalDataKey)
	iter := prefixStore.ReverseIterator(PrefixRange(types.UInt64Bytes(hyperionId)))
	// iterate over [OutgoingExternalDataKey][hyperionId]
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var tx types.OutgoingExternalDataTx
		k.cdc.MustUnmarshal(iter.Value(), &tx)
		// cb returns true to stop early
		if cb(iter.Key(), &tx) {
			break
		}
	}
}

func (k *Keeper) IterateAllOutgoingExternalDataTXs(ctx sdk.Context, cb func(key []byte, tx *types.OutgoingExternalDataTx) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.OutgoingExternalDataKey)
	iter := prefixStore.ReverseIterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var tx types.OutgoingExternalDataTx
		k.cdc.MustUnmarshal(iter.Value(), &tx)
		// cb returns true to stop early
		if cb(iter.Key(), &tx) {
			break
		}
	}
}

// GetOutgoingExternalDataTXs returns the outgoing external data txs
func (k *Keeper) GetOutgoingExternalDataTXs(ctx sdk.Context, hyperionId uint64) (out []*types.OutgoingExternalDataTx) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.IterateOutgoingExternalDataTXs(ctx, hyperionId, func(_ []byte, tx *types.OutgoingExternalDataTx) bool {
		out = append(out, tx)
		return false
	})

	return
}

func (k *Keeper) GetAllOutgoingExternalDataTXs(ctx sdk.Context) (out []*types.OutgoingExternalDataTx) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.IterateAllOutgoingExternalDataTXs(ctx, func(_ []byte, tx *types.OutgoingExternalDataTx) bool {
		out = append(out, tx)
		return false
	})

	return
}
