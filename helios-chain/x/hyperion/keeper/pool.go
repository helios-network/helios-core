package keeper

import (
	"encoding/binary"
	"math/big"
	"sort"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

// AddToOutgoingPool
// - checks a counterpart denominator exists for the given voucher type
// - burns the voucher for transfer amount and fees
// - persists an OutgoingTx
// - adds the TX to the `available` TX pool via a second index
func (k *Keeper) AddToOutgoingPool(ctx sdk.Context, sender sdk.AccAddress, counterpartReceiver common.Address, amount, fee sdk.Coin, hyperionId uint64, txHash string) (uint64, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	totalAmount := amount.Add(fee)
	totalInVouchers := sdk.Coins{totalAmount}

	// If the coin is a hyperion voucher, burn the coins. If not, check if there is a deployed ERC20 contract representing it.
	// If there is, lock the coins.

	tokenAddressToDenom, exists := k.GetTokenFromDenom(ctx, hyperionId, totalAmount.Denom)
	if !exists {
		metrics.ReportFuncError(k.svcTags)
		return 0, errors.Wrapf(types.ErrInvalid, "token not found")
	}

	pendingTxs := k.GetPoolTransactions(ctx, hyperionId)
	for _, tx := range pendingTxs {
		if tx.Sender == sender.String() && tx.HyperionId == hyperionId {
			return 0, errors.Wrap(types.ErrInvalid, "sender already has a pending tx")
		}
	}

	isCosmosOriginated := tokenAddressToDenom.IsCosmosOriginated
	tokenContract := common.HexToAddress(tokenAddressToDenom.TokenAddress)

	if isCosmosOriginated {
		contractBalance := k.GetHyperionContractBalance(ctx, hyperionId, tokenContract)
		k.SetHyperionContractBalance(ctx, hyperionId, tokenContract, contractBalance.Add(totalAmount.Amount))
	}

	// send coins to module in prep for burn
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, totalInVouchers); err != nil {
		return 0, err
	}

	// burn vouchers to send them back to ETH
	if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, totalInVouchers); err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic(err)
	}

	// get next tx id from keeper
	nextID := k.AutoIncrementID(ctx, types.KeyLastTXPoolID)
	erc20Fee := types.NewSDKIntERC20Token(fee.Amount, tokenContract)

	// construct outgoing tx, as part of this process we represent
	// the token as an ERC20 token since it is preparing to go to ETH
	// rather than the denom that is the input to this function.
	outgoing := &types.OutgoingTransferTx{
		Id:          nextID,
		Sender:      sender.String(),
		HyperionId:  hyperionId,
		DestAddress: counterpartReceiver.Hex(),
		Token:       types.NewSDKIntERC20Token(amount.Amount, tokenContract),
		Fee:         erc20Fee,
		TxTimeout:   k.GetOutgoingTxTimeoutHeight(ctx, hyperionId),
		TxHash:      txHash,
	}

	// set the outgoing tx in the pool index
	if err := k.setPoolEntry(ctx, outgoing); err != nil {
		return 0, err
	}

	// add a second index with the fee
	k.appendToUnbatchedTXIndex(ctx, hyperionId, tokenContract, erc20Fee, nextID)

	return nextID, nil
}

func (k *Keeper) CleanPoolTransactions(ctx sdk.Context, hyperionId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	poolTx := k.GetPoolTransactions(ctx, hyperionId)
	for _, tx := range poolTx {
		txSender, err := sdk.AccAddressFromBech32(tx.Sender)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			continue
		}
		k.RemoveFromOutgoingPoolAndRefund(ctx, hyperionId, tx.Id, txSender)
	}
}

// RemoveFromOutgoingPoolAndRefund
// - checks that the provided tx actually exists
// - deletes the unbatched tx from the pool
// - issues the tokens back to the sender
func (k *Keeper) RemoveFromOutgoingPoolAndRefund(ctx sdk.Context, hyperionId uint64, txId uint64, sender sdk.AccAddress) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// check that we actually have a tx with that id and what it's details are
	tx, err := k.getPoolEntry(ctx, hyperionId, txId)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	txSender, err := sdk.AccAddressFromBech32(tx.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	if !sender.Equals(txSender) {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalid, "Invalid sender address")
	}

	// An inconsistent entry should never enter the store, but this is the ideal place to exploit
	// it such a bug if it did ever occur, so we should double check to be really sure
	if tx.Fee.Contract != tx.Token.Contract {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalid, "Inconsistent tokens to cancel!: %s %s", tx.Fee.Contract, tx.Token.Contract)
	}

	found := false
	poolTx := k.GetPoolTransactions(ctx, tx.HyperionId)
	for _, pTx := range poolTx {
		if pTx.Id == txId {
			found = true
		}
	}
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalid, "txId %d is not in unbatched pool! Must be in batch!", txId)
	}

	// delete this tx from both indexes
	err = k.removeFromUnbatchedTXIndex(ctx, tx.HyperionId, common.HexToAddress(tx.Token.Contract), tx.Fee, txId)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalid, "txId %d not in unbatched index! Must be in a batch!", txId)
	}
	k.removePoolEntry(ctx, tx.HyperionId, txId)

	// reissue the amount and the fee
	var totalToRefundCoins sdk.Coins
	tokenAddressToDenom, exists := k.GetTokenFromAddress(ctx, tx.HyperionId, common.HexToAddress(tx.Token.Contract))
	if !exists {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(types.ErrInvalid, "txId %d not in unbatched index! Must be in a batch!", txId)
	}
	// native cosmos coin denom
	totalToRefund := sdk.NewCoin(tokenAddressToDenom.Denom, tx.Token.Amount)
	totalToRefund.Amount = totalToRefund.Amount.Add(tx.Fee.Amount)
	totalToRefundCoins = sdk.NewCoins(totalToRefund)

	if tokenAddressToDenom.IsCosmosOriginated { // update the contract balance
		contractBalance := k.GetHyperionContractBalance(ctx, tx.HyperionId, common.HexToAddress(tx.Token.Contract))

		if contractBalance.LT(tx.Token.Amount.Add(tx.Fee.Amount)) {
			metrics.ReportFuncError(k.svcTags)
			return errors.Wrap(types.ErrSupplyOverflow, "invalid supply on the source network")
		}
		k.SetHyperionContractBalance(ctx, tx.HyperionId, common.HexToAddress(tx.Token.Contract), contractBalance.Sub(tx.Token.Amount.Add(tx.Fee.Amount)))
	}

	// mint coins in module for prep to send
	if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, totalToRefundCoins); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrapf(err, "mint vouchers coins: %s", totalToRefundCoins)
	}
	if err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sender, totalToRefundCoins); err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(err, "transfer vouchers")
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventBridgeWithdrawCanceled{
		BridgeContract: k.GetBridgeContractAddress(ctx)[tx.HyperionId].Hex(),
		BridgeChainId:  k.GetBridgeChainID(ctx)[tx.HyperionId],
	})

	return nil
}

// appendToUnbatchedTXIndex add at the end when tx with same fee exists
func (k *Keeper) appendToUnbatchedTXIndex(ctx sdk.Context, hyperionId uint64, tokenContract common.Address, fee *types.Token, txID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	idxKey := types.GetFeeSecondIndexKey(hyperionId, tokenContract, fee)
	var idSet types.IDSet
	if store.Has(idxKey) {
		bz := store.Get(idxKey)
		k.cdc.MustUnmarshal(bz, &idSet)
	}
	idSet.Ids = append(idSet.Ids, txID)
	store.Set(idxKey, k.cdc.MustMarshal(&idSet))
}

// appendToUnbatchedTXIndex add at the top when tx with same fee exists
func (k *Keeper) prependToUnbatchedTXIndex(ctx sdk.Context, hyperionId uint64, tokenContract common.Address, fee *types.Token, txID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	idxKey := types.GetFeeSecondIndexKey(hyperionId, tokenContract, fee)
	var idSet types.IDSet
	if store.Has(idxKey) {
		bz := store.Get(idxKey)
		k.cdc.MustUnmarshal(bz, &idSet)
	}

	idSet.Ids = append([]uint64{txID}, idSet.Ids...)
	store.Set(idxKey, k.cdc.MustMarshal(&idSet))
}

// removeFromUnbatchedTXIndex removes the tx from the index and also removes it from the iterator
// GetPoolTransactions, making this tx implicitly invisible without a direct request. We remove a tx
// from the pool for good in OutgoingTxBatchExecuted, but if a batch is canceled or timed out we 'reactivate'
// an entry by adding it back to the second index.
func (k *Keeper) removeFromUnbatchedTXIndex(ctx sdk.Context, hyperionId uint64, tokenContract common.Address, fee *types.Token, txID uint64) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	idxKey := types.GetFeeSecondIndexKey(hyperionId, tokenContract, fee)

	var idSet types.IDSet
	bz := store.Get(idxKey)
	if bz == nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrUnknown, "fee")
	}

	k.cdc.MustUnmarshal(bz, &idSet)
	for i := range idSet.Ids {
		if idSet.Ids[i] == txID {
			idSet.Ids = append(idSet.Ids[0:i], idSet.Ids[i+1:]...)
			if len(idSet.Ids) != 0 {
				store.Set(idxKey, k.cdc.MustMarshal(&idSet))
			} else {
				store.Delete(idxKey)
			}
			return nil
		}
	}

	metrics.ReportFuncError(k.svcTags)
	return errors.Wrap(types.ErrUnknown, "tx id")
}

func (k *Keeper) setPoolEntry(ctx sdk.Context, outgoingTransferTx *types.OutgoingTransferTx) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	bz, err := k.cdc.Marshal(outgoingTransferTx)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return err
	}

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetOutgoingTxPoolKey(outgoingTransferTx.HyperionId, outgoingTransferTx.Id), bz)

	return nil
}

// getPoolEntry grabs an entry from the tx pool, this *does* include transactions in batches
// so check the UnbatchedTxIndex or call GetPoolTransactions for that purpose
func (k *Keeper) getPoolEntry(ctx sdk.Context, hyperionId uint64, id uint64) (*types.OutgoingTransferTx, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetOutgoingTxPoolKey(hyperionId, id))
	if bz == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, types.ErrUnknown
	}

	var r types.OutgoingTransferTx
	err := k.cdc.Unmarshal(bz, &r)

	if err != nil {
		return nil, err
	}

	return &r, nil
}

// removePoolEntry removes an entry from the tx pool, this *does* include transactions in batches
// so you will need to run it when cleaning up after a executed batch
func (k *Keeper) removePoolEntry(ctx sdk.Context, hyperionId uint64, id uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetOutgoingTxPoolKey(hyperionId, id))
}

// GetPoolTransactions, grabs all transactions from the tx pool, useful for queries or genesis save/load
// this does not include all transactions in batches, because it iterates using the second index key
func (k *Keeper) GetPoolTransactions(ctx sdk.Context, hyperionId uint64) []*types.OutgoingTransferTx {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := ctx.KVStore(k.storeKey)
	iter := prefixStore.ReverseIterator(PrefixRange(append(types.SecondIndexOutgoingTXFeeKey, sdk.Uint64ToBigEndian(hyperionId)...)))

	var ret []*types.OutgoingTransferTx
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ids types.IDSet
		k.cdc.MustUnmarshal(iter.Value(), &ids)
		for _, id := range ids.Ids {
			tx, err := k.getPoolEntry(ctx, hyperionId, id)
			if tx.HyperionId != hyperionId {
				continue
			}
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				continue
			}
			ret = append(ret, tx)
		}
	}

	return ret
}

func (k *Keeper) GetAllPoolTransactions(ctx sdk.Context) []*types.OutgoingTransferTx {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.SecondIndexOutgoingTXFeeKey)
	// we must use the second index key here because transactions are left in the store, but removed
	// from the tx sorting key, while in batches
	iter := prefixStore.ReverseIterator(nil, nil)

	var ret []*types.OutgoingTransferTx
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ids types.IDSet
		k.cdc.MustUnmarshal(iter.Value(), &ids)
		for _, id := range ids.Ids {
			key := iter.Key()
			HyperionIDLen := 8
			// 1. hyperionId (uint64)
			hyperionIdBytes := key[:HyperionIDLen]
			hyperionIdOfTheTx := binary.BigEndian.Uint64(hyperionIdBytes)

			tx, err := k.getPoolEntry(ctx, hyperionIdOfTheTx, id)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				continue
			}
			ret = append(ret, tx)
		}
	}

	return ret
}

// IterateOnSpecificalTokenContractOutgoingPoolByFee itetates over the outgoing pool which is sorted by fee
func (k *Keeper) IterateOnSpecificalTokenContractOutgoingPoolByFee(ctx sdk.Context, hyperionId uint64, tokenContract common.Address, cb func(uint64, *types.OutgoingTransferTx) bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.SecondIndexOutgoingTXFeeKey)
	iter := prefixStore.ReverseIterator(PrefixRange(types.GetPrefixRangeForGetFeeSecondIndexKeyOnSpecificalTokenContract(hyperionId, tokenContract)))
	// iterate over all [SecondIndexOutgoingTXFeeKey][hyperionId][tokenContract] prefixed []bytes

	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var ids types.IDSet
		k.cdc.MustUnmarshal(iter.Value(), &ids)
		// cb returns true to stop early
		for _, id := range ids.Ids {
			tx, err := k.getPoolEntry(ctx, hyperionId, id)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				continue
			}
			if cb(id, tx) {
				return
			}
		}
	}
}

// GetBatchFeesByTokenType gets the fees the next batch of a given token type would
// have if created. This info is both presented to relayers for the purpose of determining
// when to request batches and also used by the batch creation process to decide not to create
// a new batch
func (k *Keeper) GetBatchFeesByTokenType(ctx sdk.Context, hyperionId uint64, tokenContractAddr common.Address) *types.BatchFees {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	batchFeesMap := k.createBatchFees(ctx, hyperionId)
	return batchFeesMap[tokenContractAddr]
}

// GetAllBatchFees creates a fee entry for every batch type currently in the store
// this can be used by relayers to determine what batch types are desirable to request
func (k *Keeper) GetAllBatchFees(ctx sdk.Context, hyperionId uint64) (batchFees []*types.BatchFees) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	batchFeesMap := k.createBatchFees(ctx, hyperionId)
	// create array of batchFees
	for _, batchFee := range batchFeesMap {
		batchFees = append(batchFees, batchFee)
	}

	// quick sort by token to make this function safe for use
	// in consensus computations
	sort.SliceStable(batchFees, func(i, j int) bool {
		return batchFees[i].Token < batchFees[j].Token
	})

	return batchFees
}

// CreateBatchFees iterates over the outgoing pool and creates batch token fee map
func (k *Keeper) createBatchFees(ctx sdk.Context, hyperionId uint64) map[common.Address]*types.BatchFees {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.SecondIndexOutgoingTXFeeKey)
	iter := prefixStore.Iterator(PrefixRange(types.GetPrefixRangeForGetFeeSecondIndexKey(hyperionId)))
	// iterate over [SecondIndexOutgoingTXFeeKey][hyperionId] prefixes
	defer iter.Close()

	batchFeesMap := make(map[common.Address]*types.BatchFees)
	txCountMap := make(map[common.Address]int)

	for ; iter.Valid(); iter.Next() {
		var ids types.IDSet
		k.cdc.MustUnmarshal(iter.Value(), &ids)

		// create a map to store the token contract address and its total fee
		// Parse the iterator key to get contract address & fee
		// If len(ids.Ids) > 1, multiply fee amount with len(ids.Ids) and add it to total fee amount

		key := iter.Key()

		HyperionIDLen := 8
		ETHContractAddressLen := 20
		FeeAmountLen := 32

		// 1. hyperionId (uint64)
		// hyperionIdBytes := key[:HyperionIDLen]
		// hyperionIdOfTheTx := binary.BigEndian.Uint64(hyperionIdBytes)

		// 2. tokenContractAddr (common.Address)
		tokenContractBytes := key[HyperionIDLen : HyperionIDLen+ETHContractAddressLen]
		tokenContractAddr := common.BytesToAddress(tokenContractBytes)

		// 3. feeAmount (*big.Int)
		feeAmountBytes := key[HyperionIDLen+ETHContractAddressLen : HyperionIDLen+ETHContractAddressLen+FeeAmountLen]
		feeAmount := big.NewInt(0).SetBytes(feeAmountBytes)

		for i := 0; i < len(ids.Ids); i++ {
			if txCountMap[tokenContractAddr] >= OutgoingTxBatchSize {
				break
			} else {
				// add fee amount
				if _, ok := batchFeesMap[tokenContractAddr]; ok {
					totalFees := batchFeesMap[tokenContractAddr].TotalFees
					totalFees = totalFees.Add(math.NewIntFromBigInt(feeAmount))
					batchFeesMap[tokenContractAddr].TotalFees = totalFees
				} else {
					batchFeesMap[tokenContractAddr] = &types.BatchFees{
						Token:     tokenContractAddr.Hex(),
						TotalFees: math.NewIntFromBigInt(feeAmount)}
				}

				txCountMap[tokenContractAddr]++
			}
		}
	}

	return batchFeesMap
}

func (k *Keeper) AutoIncrementID(ctx sdk.Context, idKey []byte) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bz := store.Get(idKey)

	var id uint64
	if bz != nil {
		id = binary.BigEndian.Uint64(bz) + 1
	} else {
		id = 1
	}

	bz = sdk.Uint64ToBigEndian(id)
	store.Set(idKey, bz)

	return id
}

func (k *Keeper) SetID(ctx sdk.Context, idKey []byte, id uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bz := sdk.Uint64ToBigEndian(id)
	store.Set(idKey, bz)
}

func (k *Keeper) GetLastOutgoingBatchID(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetLastOutgoingBatchIDKey(hyperionId)
	var id uint64
	bz := store.Get(key)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	return id
}

func (k *Keeper) SetLastOutgoingBatchID(ctx sdk.Context, hyperionId uint64, lastOutgoingBatchID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetLastOutgoingBatchIDKey(hyperionId)
	bz := sdk.Uint64ToBigEndian(lastOutgoingBatchID)
	store.Set(key, bz)
}

func (k *Keeper) GetLastOutgoingPoolID(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetLastTXPoolIDKey(hyperionId)
	var id uint64
	bz := store.Get(key)
	if bz != nil {
		id = binary.BigEndian.Uint64(bz)
	}
	return id
}

func (k *Keeper) SetLastOutgoingPoolID(ctx sdk.Context, hyperionId uint64, lastOutgoingPoolID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetLastTXPoolIDKey(hyperionId)
	bz := sdk.Uint64ToBigEndian(lastOutgoingPoolID)
	store.Set(key, bz)
}

// / This gets the batch timeout height in the counterparty chain blocks.
func (k *Keeper) GetOutgoingTxTimeoutHeight(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	counterpartyChainParams := k.GetCounterpartyChainParams(ctx)[hyperionId]
	projectedCurrentEthereumHeight := k.GetProjectedCurrentEthereumHeight(ctx, hyperionId)

	blocksToAdd := counterpartyChainParams.TargetOutgoingTxTimeout / counterpartyChainParams.AverageCounterpartyBlockTime
	return projectedCurrentEthereumHeight + blocksToAdd
}
