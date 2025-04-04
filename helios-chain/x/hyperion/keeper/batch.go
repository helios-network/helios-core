package keeper

import (
	"bytes"
	"fmt"

	"cosmossdk.io/errors"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/hyperion/types"

	"github.com/Helios-Chain-Labs/metrics"
)

const OutgoingTxBatchSize = 100

// BuildOutgoingTXBatch starts the following process chain:
//   - find bridged denominator for given voucher type
//   - determine if a an unexecuted batch is already waiting for this token type, if so confirm the new batch would
//     have a higher total fees. If not exit without creating a batch
//   - select available transactions from the outgoing transaction pool sorted by fee desc
//   - persist an outgoing batch object with an incrementing ID = nonce
//   - emit an event
func (k *Keeper) BuildOutgoingTXBatch(ctx sdk.Context, tokenContract common.Address, hyperionId uint64, maxElements int) (*types.OutgoingTxBatch, error) {
	fmt.Println("BuildOutgoingTXBatch for hyperionId: ", hyperionId)
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if maxElements == 0 {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "max elements value")
	}

	lastBatch := k.GetLastOutgoingBatchByTokenType(ctx, hyperionId, tokenContract)

	// lastBatch may be nil if there are no existing batches, we only need
	// to perform this check if a previous batch exists
	if lastBatch != nil {
		// this traverses the current tx pool for this token type and determines what
		// fees a hypothetical batch would have if created
		currentFees := k.GetBatchFeesByTokenType(ctx, hyperionId, tokenContract)
		if currentFees == nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrap(types.ErrInvalid, "error getting fees from tx pool")
		}

		lastFees := lastBatch.GetFees()
		if lastFees.GT(currentFees.TotalFees) {
			metrics.ReportFuncError(k.svcTags)
			return nil, errors.Wrap(types.ErrInvalid, "new batch would not be more profitable")
		}
	}

	selectedTx, err := k.pickUnbatchedTX(ctx, tokenContract, maxElements, hyperionId)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	nextID := k.AutoIncrementID(ctx, types.GetLastOutgoingBatchIDKey(hyperionId))
	batch := &types.OutgoingTxBatch{
		BatchNonce:    nextID,
		BatchTimeout:  k.getBatchTimeoutHeight(ctx, hyperionId),
		Transactions:  selectedTx,
		TokenContract: tokenContract.Hex(),
		HyperionId:    hyperionId,
	}
	k.StoreBatch(ctx, batch)

	// Get the checkpoint and store it as a legit past batch
	checkpoint := batch.GetCheckpoint(hyperionId)
	k.SetPastEthSignatureCheckpoint(ctx, hyperionId, checkpoint)

	return batch, nil
}

// / This gets the batch timeout height in the counterparty chain blocks.
func (k *Keeper) getBatchTimeoutHeight(ctx sdk.Context, hyperionId uint64) uint64 {
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
func (k *Keeper) OutgoingTxBatchExecuted(ctx sdk.Context, tokenContract common.Address, nonce uint64, hyperionId uint64, claim *types.MsgWithdrawClaim) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	b := k.GetOutgoingTXBatch(ctx, tokenContract, nonce, hyperionId)
	if b == nil {
		metrics.ReportFuncError(k.svcTags)
		return
		// panic(fmt.Sprintf("unknown batch nonce for outgoing tx batch %s %d", tokenContract, nonce))
	}

	// cleanup outgoing TX pool, while these transactions where hidden from GetPoolTransactions
	// they still exist in the pool and need to be cleaned up.
	for _, tx := range b.Transactions {
		k.removePoolEntry(ctx, tx.HyperionId, tx.Id)
	}

	// Iterate through remaining batches
	k.IterateOutgoingTXBatches(ctx, hyperionId, func(key []byte, iter_batch *types.OutgoingTxBatch) bool {
		// If the iterated batches nonce is lower than the one that was just executed, cancel it
		if iter_batch.HyperionId == hyperionId && iter_batch.BatchNonce < b.BatchNonce && common.HexToAddress(iter_batch.TokenContract) == tokenContract {
			err := k.CancelOutgoingTXBatch(ctx, tokenContract, iter_batch.BatchNonce, iter_batch.HyperionId)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				panic(fmt.Sprintf("Failed cancel out batch %s %d while trying to execute %s %d with %s", tokenContract, iter_batch.BatchNonce, tokenContract, nonce, err))
			}
		}
		return false
	})

	// Delete batch since it is finished
	k.DeleteBatch(ctx, *b)

	for _, tx := range b.Transactions {
		k.StoreFinalizedTx(ctx, &types.TransferTx{
			HyperionId:  tx.HyperionId,
			Id:          tx.Id,
			Sender:      cmn.AnyToHexAddress(tx.Sender).String(),
			DestAddress: cmn.AnyToHexAddress(tx.DestAddress).String(),
			Erc20Token:  tx.Erc20Token,
			Erc20Fee:    tx.Erc20Fee,
			Status:      "BRIDGED",
			Direction:   "OUT",
			ChainId:     k.GetBridgeChainID(ctx)[tx.HyperionId],
			Height:      claim.BlockHeight,
			TxHash:      tx.TxHash,
			Proof: &types.Proof{
				Orchestrators: cmn.AnyToHexAddress(claim.Orchestrator).String(),
				Hashs:         claim.TxHash,
			},
		})
	}
}

// StoreBatch stores a transaction batch
func (k *Keeper) StoreBatch(ctx sdk.Context, batch *types.OutgoingTxBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	// set the current block height when storing the batch
	batch.Block = uint64(ctx.BlockHeight())
	key := types.GetOutgoingTxBatchKey(common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId)
	fmt.Println("StoreBatch - key: ", key)
	store.Set(key, k.cdc.MustMarshal(batch))

	blockKey := types.GetOutgoingTxBatchBlockKey(batch.HyperionId, batch.Block)
	store.Set(blockKey, k.cdc.MustMarshal(batch))
}

// StoreBatchUnsafe stores a transaction batch w/o setting the height
func (k *Keeper) StoreBatchUnsafe(ctx sdk.Context, batch *types.OutgoingTxBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetOutgoingTxBatchKey(common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId)
	store.Set(key, k.cdc.MustMarshal(batch))

	blockKey := types.GetOutgoingTxBatchBlockKey(batch.HyperionId, batch.Block)
	store.Set(blockKey, k.cdc.MustMarshal(batch))

	// make sure transactions are indexed with OutgoingTXPoolKey
	for _, tx := range batch.Transactions {
		if err := k.setPoolEntry(ctx, tx); err != nil {
			panic("cannot index batch tx")
		}
	}
}

// DeleteBatch deletes an outgoing transaction batch
func (k *Keeper) DeleteBatch(ctx sdk.Context, batch types.OutgoingTxBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetOutgoingTxBatchKey(common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId))
	store.Delete(types.GetOutgoingTxBatchBlockKey(batch.HyperionId, batch.Block))
}

// pickUnbatchedTX find TX in pool and remove from "available" second index
func (k *Keeper) pickUnbatchedTX(ctx sdk.Context, tokenContract common.Address, maxElements int, hyperionId uint64) ([]*types.OutgoingTransferTx, error) {
	fmt.Println("pickUnbatchedTX for hyperionId: ", hyperionId)
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	selectedTx := make([]*types.OutgoingTransferTx, 0)
	var err error

	k.IterateOnSpecificalTokenContractOutgoingPoolByFee(ctx, hyperionId, tokenContract, func(txID uint64, tx *types.OutgoingTransferTx) bool {
		fmt.Println("pickUnbatchedTX - tx: ", tx)
		if tx != nil && tx.Erc20Fee != nil {
			selectedTx = append(selectedTx, tx)
			err = k.removeFromUnbatchedTXIndex(ctx, hyperionId, tokenContract, tx.Erc20Fee, txID)
			return err != nil || len(selectedTx) == maxElements
		} else {
			// we found a nil, exit
			return true
		}
	})

	if len(selectedTx) == 0 {
		return nil, types.ErrNoUnbatchedTxsFound
	}

	return selectedTx, nil
}

// GetOutgoingTXBatch loads a batch object. Returns nil when not exists.
func (k *Keeper) GetOutgoingTXBatch(ctx sdk.Context, tokenContract common.Address, nonce uint64, hyperionId uint64) *types.OutgoingTxBatch {
	fmt.Println("GetOutgoingTXBatch: ", tokenContract, nonce, hyperionId)
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetOutgoingTxBatchKey(tokenContract, nonce, hyperionId)
	fmt.Println("GetOutgoingTxBatchKey - key: ", key)
	bz := store.Get(key)
	if len(bz) == 0 {
		return nil
	}

	var b types.OutgoingTxBatch
	k.cdc.MustUnmarshal(bz, &b)
	fmt.Printf("GetOutgoingTxBatch - b: %+v\n", b)
	for _, tx := range b.Transactions {
		tx.Erc20Token.Contract = tokenContract.Hex()
		tx.Erc20Fee.Contract = tokenContract.Hex()
	}

	return &b
}

// CancelOutgoingTXBatch releases all TX in the batch and deletes the batch
func (k *Keeper) CancelOutgoingTXBatch(ctx sdk.Context, tokenContract common.Address, nonce uint64, hyperionId uint64) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	batch := k.GetOutgoingTXBatch(ctx, tokenContract, nonce, hyperionId)
	if batch == nil {
		return types.ErrUnknown
	}

	for _, tx := range batch.Transactions {
		tx.Erc20Fee.Contract = tokenContract.Hex()
		k.prependToUnbatchedTXIndex(ctx, hyperionId, tokenContract, tx.Erc20Fee, tx.Id)
	}

	// Delete batch since it is finished
	k.DeleteBatch(ctx, *batch)

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

// IterateOutgoingTXBatches iterates through all outgoing batches in DESC order.
func (k *Keeper) IterateOutgoingTXBatches(ctx sdk.Context, hyperionId uint64, cb func(key []byte, batch *types.OutgoingTxBatch) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.OutgoingTXBatchKey)
	iter := prefixStore.ReverseIterator(PrefixRange(types.UInt64Bytes(hyperionId)))
	// iterate over [OutgoingTXBatchKey][hyperionId]
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var batch types.OutgoingTxBatch
		k.cdc.MustUnmarshal(iter.Value(), &batch)

		k.Logger(ctx).Info("IterateOutgoingTXBatches", "batch", batch, "hyperionId", hyperionId)
		// cb returns true to stop early
		if cb(iter.Key(), &batch) {
			break
		}
	}
}

func (k *Keeper) IterateAllOutgoingTXBatches(ctx sdk.Context, cb func(key []byte, batch *types.OutgoingTxBatch) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.OutgoingTXBatchKey)
	iter := prefixStore.ReverseIterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var batch types.OutgoingTxBatch
		k.cdc.MustUnmarshal(iter.Value(), &batch)

		fmt.Println("IterateOutgoingTXBatches - batch: ", batch)
		// cb returns true to stop early
		if cb(iter.Key(), &batch) {
			break
		}
	}
}

// GetOutgoingTxBatches returns the outgoing tx batches
func (k *Keeper) GetOutgoingTxBatches(ctx sdk.Context, hyperionId uint64) (out []*types.OutgoingTxBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.IterateOutgoingTXBatches(ctx, hyperionId, func(_ []byte, batch *types.OutgoingTxBatch) bool {
		out = append(out, batch)
		return false
	})

	return
}

func (k *Keeper) GetAllOutgoingTxBatches(ctx sdk.Context) (out []*types.OutgoingTxBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	k.IterateAllOutgoingTXBatches(ctx, func(_ []byte, batch *types.OutgoingTxBatch) bool {
		out = append(out, batch)
		return false
	})

	return
}

// GetLastOutgoingBatchByTokenType gets the latest outgoing tx batch by token type
func (k *Keeper) GetLastOutgoingBatchByTokenType(ctx sdk.Context, hyperionId uint64, token common.Address) *types.OutgoingTxBatch {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	batches := k.GetOutgoingTxBatches(ctx, hyperionId)
	var lastBatch *types.OutgoingTxBatch = nil
	lastNonce := uint64(0)

	for _, batch := range batches {
		if bytes.Equal(common.HexToAddress(batch.TokenContract).Bytes(), token.Bytes()) && batch.BatchNonce > lastNonce {
			lastBatch = batch
			lastNonce = batch.BatchNonce
		}
	}

	return lastBatch
}

// SetLastSlashedBatchBlock sets the latest slashed Batch block height
func (k *Keeper) SetLastSlashedBatchBlock(ctx sdk.Context, hyperionId uint64, blockHeight uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetLastSlashedBatchBlockKey(hyperionId), types.UInt64Bytes(blockHeight))
}

// GetLastSlashedBatchBlock returns the latest slashed Batch block
func (k *Keeper) GetLastSlashedBatchBlock(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	storedBytes := store.Get(types.GetLastSlashedBatchBlockKey(hyperionId))

	if len(storedBytes) == 0 {
		return 0
	}

	return types.UInt64FromBytes(storedBytes)
}

// GetUnslashedBatches returns all the unslashed batches in state
func (k *Keeper) GetUnslashedBatches(ctx sdk.Context, hyperionId uint64, maxHeight uint64) (out []*types.OutgoingTxBatch) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	lastSlashedBatchBlock := k.GetLastSlashedBatchBlock(ctx, hyperionId)
	k.IterateBatchBySlashedBatchBlock(ctx, lastSlashedBatchBlock, maxHeight, func(_ []byte, batch *types.OutgoingTxBatch) bool {
		if batch.Block > lastSlashedBatchBlock {
			out = append(out, batch)
		}
		return false
	})

	return
}

// IterateBatchBySlashedBatchBlock iterates through all Batch by last slashed Batch block in ASC order
func (k *Keeper) IterateBatchBySlashedBatchBlock(ctx sdk.Context, lastSlashedBatchBlock, maxHeight uint64, cb func([]byte, *types.OutgoingTxBatch) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.OutgoingTXBatchBlockKey)
	iter := prefixStore.Iterator(types.UInt64Bytes(lastSlashedBatchBlock), types.UInt64Bytes(maxHeight))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var Batch types.OutgoingTxBatch
		k.cdc.MustUnmarshal(iter.Value(), &Batch)
		// cb returns true to stop early
		if cb(iter.Key(), &Batch) {
			break
		}
	}
}
