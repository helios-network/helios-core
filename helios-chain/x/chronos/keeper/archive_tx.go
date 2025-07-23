package keeper

import (
	"encoding/json"
	"helios-core/helios-chain/testnet"
	"helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
	"strconv"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *Keeper) GetArchivedCron(ctx sdk.Context, id uint64) (types.Cron, bool) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		bz := k.archiveStore.Get([]byte(strconv.FormatUint(id, 10)))
		if bz == nil {
			return types.Cron{}, false
		}
		var cron types.Cron
		k.cdc.MustUnmarshal(bz, &cron)
		return cron, true
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronArchivedKey)
	bz := store.Get(GetCronIDBytes(id))
	if bz == nil {
		return types.Cron{}, false
	}

	var cron types.Cron
	k.cdc.MustUnmarshal(bz, &cron)
	return cron, true
}

func (k *Keeper) GetCronTransactionResultByNonce(ctx sdk.Context, nonce uint64) (types.CronTransactionResult, bool) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		bz := k.archiveStore.Get(append(types.ArchiveStoreTxKey, strconv.FormatUint(nonce, 10)...))
		if bz == nil {
			return types.CronTransactionResult{}, false
		}
		var txResult types.CronTransactionResult
		k.cdc.MustUnmarshal(bz, &txResult)
		return txResult, true
	} else {
		store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionResultKey)
		bz := store.Get(GetTxIDBytes(nonce))
		if bz == nil {
			return types.CronTransactionResult{}, false
		}
		var txResult types.CronTransactionResult
		k.cdc.MustUnmarshal(bz, &txResult)
		return txResult, true
	}
}

func (k *Keeper) GetCronTransactionResultByHash(ctx sdk.Context, hash string) (types.CronTransactionResult, bool) {
	nonce, ok := k.GetTxNonceByHash(ctx, hash)
	if !ok {
		return types.CronTransactionResult{}, false
	}
	return k.GetCronTransactionResultByNonce(ctx, nonce)
}

func (k *Keeper) GetCronTransactionResultsByBlockNumber(ctx sdk.Context, blockNumber uint64) ([]types.CronTransactionResult, bool) {
	txHashs, ok := k.GetBlockTxHashs(ctx, blockNumber)
	if !ok {
		return []types.CronTransactionResult{}, false
	}
	txs := make([]types.CronTransactionResult, 0)
	max := 100 // todo pagination
	for _, txHash := range txHashs {
		tx, ok := k.GetCronTransactionResultByHash(ctx, txHash)
		if !ok {
			continue
		}
		txs = append(txs, tx)
		if len(txs) >= max {
			break
		}
	}
	return txs, true
}

func (k *Keeper) GetCronTransactionReceiptByHash(ctx sdk.Context, hash string) (*types.CronTransactionReceiptRPC, bool) {
	tx, ok := k.GetCronTransactionResultByHash(ctx, hash)
	if !ok {
		return nil, false
	}
	receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
	return receiptTx, true
}

func (k *Keeper) GetCronTransactionReceiptByNonce(ctx sdk.Context, nonce uint64) (*types.CronTransactionReceiptRPC, bool) {
	tx, ok := k.GetCronTransactionResultByNonce(ctx, nonce)
	if !ok {
		return nil, false
	}
	receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
	return receiptTx, true
}

func (k *Keeper) GetCronTransactionReceiptsByBlockNumber(ctx sdk.Context, blockNumber uint64) ([]*types.CronTransactionReceiptRPC, bool) {
	txHashs, ok := k.GetBlockTxHashs(ctx, blockNumber)
	if !ok {
		return []*types.CronTransactionReceiptRPC{}, false
	}
	txs := make([]*types.CronTransactionReceiptRPC, 0)
	max := 100 // todo pagination
	for _, txHash := range txHashs {
		tx, ok := k.GetCronTransactionResultByHash(ctx, txHash)
		if !ok {
			continue
		}
		receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
		txs = append(txs, receiptTx)
		if len(txs) >= max {
			break
		}
	}
	return txs, true
}

func (k *Keeper) GetCronTransactionLogsByBlockNumber(ctx sdk.Context, blockNumber uint64) ([]*evmtypes.Log, bool) {
	txHashs, ok := k.GetBlockTxHashs(ctx, blockNumber)
	if !ok {
		return []*evmtypes.Log{}, false
	}
	txs := make([]*evmtypes.Log, 0)
	max := 100 // todo pagination
	for _, txHash := range txHashs {
		tx, ok := k.GetCronTransactionResultByHash(ctx, txHash)
		if !ok {
			continue
		}
		receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
		txs = append(txs, receiptTx.Logs...)
		if len(txs) >= max {
			break
		}
	}
	return txs, true
}

func (k *Keeper) SetTotalCronCount(ctx sdk.Context, count uint64) {
	k.archiveStore.Set(types.ArchiveStoreCronCountKey, sdk.Uint64ToBigEndian(count))
}

func (k *Keeper) GetTotalCronCount(ctx sdk.Context) uint64 {
	bz := k.archiveStore.Get(types.ArchiveStoreCronCountKey)
	if bz == nil {
		return 0
	}
	return sdk.BigEndianToUint64(bz)
}

func (k *Keeper) StoreChangeArchivedTotalCount(ctx sdk.Context, increment int32) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		count := k.GetArchivedCronCount(ctx) + increment
		k.archiveStore.Set(types.ArchiveStoreArchivedCronCountKey, sdk.Uint64ToBigEndian(uint64(count)))
	} else {
		store := ctx.KVStore(k.storeKey)
		count := k.GetArchivedCronCount(ctx) + increment
		store.Set(types.CronArchivedCountKey, sdk.Uint64ToBigEndian(uint64(count)))
	}
}

func (k *Keeper) GetArchivedCronCount(ctx sdk.Context) int32 {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		bz := k.archiveStore.Get(types.ArchiveStoreArchivedCronCountKey)
		if bz == nil {
			return 0
		}
		return int32(sdk.BigEndianToUint64(bz))
	} else {
		store := ctx.KVStore(k.storeKey)
		bz := store.Get(types.CronArchivedCountKey)
		if bz == nil {
			return 0
		}
		return int32(sdk.BigEndianToUint64(bz))
	}
}

func (k *Keeper) StoreChangeCronRefundedLastBlockTotalCount(ctx sdk.Context, count uint64) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		k.archiveStore.Set(types.ArchiveStoreRefundedLastBlockCountKey, sdk.Uint64ToBigEndian(count))
	} else {
		store := ctx.KVStore(k.storeKey)
		store.Set(types.CronRefundedLastBlockCountKey, sdk.Uint64ToBigEndian(count))
	}
}

func (k *Keeper) GetCronRefundedLastBlockCount(ctx sdk.Context) uint64 {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		bz := k.archiveStore.Get(types.ArchiveStoreRefundedLastBlockCountKey)
		if bz == nil {
			return 0
		}
		return sdk.BigEndianToUint64(bz)
	} else {
		store := ctx.KVStore(k.storeKey)
		bz := store.Get(types.CronRefundedLastBlockCountKey)
		if bz == nil {
			return 0
		}
		return sdk.BigEndianToUint64(bz)
	}
}

func (k *Keeper) StoreChangeCronExecutedLastBlockTotalCount(ctx sdk.Context, count uint64) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		k.archiveStore.Set(types.ArchiveStoreExecutedLastBlockCountKey, sdk.Uint64ToBigEndian(count))
	} else {
		store := ctx.KVStore(k.storeKey)
		store.Set(types.CronExecutedLastBlockCountKey, sdk.Uint64ToBigEndian(count))
	}
}

func (k *Keeper) GetCronExecutedLastBlockCount(ctx sdk.Context) uint64 {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		bz := k.archiveStore.Get(types.ArchiveStoreExecutedLastBlockCountKey)
		if bz == nil {
			return 0
		}
		return sdk.BigEndianToUint64(bz)
	} else {
		store := ctx.KVStore(k.storeKey)
		bz := store.Get(types.CronExecutedLastBlockCountKey)
		if bz == nil {
			return 0
		}
		return sdk.BigEndianToUint64(bz)
	}
}

func (k *Keeper) StoreArchiveCron(ctx sdk.Context, cron types.Cron) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		cron.Archived = true
		k.archiveStore.Set([]byte(strconv.FormatUint(cron.Id, 10)), k.cdc.MustMarshal(&cron))
		return
	}
	cron.Archived = true
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronArchivedKey)
	bz := k.cdc.MustMarshal(&cron)
	store.Set(GetCronIDBytes(cron.Id), bz)
}

func (k *Keeper) GetBlockTxHashs(ctx sdk.Context, blockNumber uint64) ([]string, bool) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		key := append(types.ArchiveStoreBlockTransactionHashsKey, strconv.FormatUint(blockNumber, 10)...)
		bz := k.archiveStore.Get(key)
		if bz == nil {
			return []string{}, false
		}
		var txHashes []string
		err := json.Unmarshal(bz, &txHashes)
		if err != nil {
			return []string{}, false
		}
		return txHashes, true
	} else {
		store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronBlockTransactionHashsKey)
		bz := store.Get(GetBlockIDBytes(blockNumber))
		if bz == nil {
			return []string{}, false
		}

		var txHashes []string
		err := json.Unmarshal(bz, &txHashes)
		if err != nil {
			return []string{}, false
		}
		return txHashes, true
	}
}

func (k *Keeper) StoreSetTransactionHashInBlock(ctx sdk.Context, blockNumber uint64, txHash string) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		key := append(types.ArchiveStoreBlockTransactionHashsKey, strconv.FormatUint(blockNumber, 10)...)
		txHashes, _ := k.GetBlockTxHashs(ctx, blockNumber)
		txHashes = append(txHashes, txHash)

		bz, _ := json.Marshal(&txHashes)
		k.archiveStore.Set(key, bz)
	} else {
		store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronBlockTransactionHashsKey)

		txHashes, _ := k.GetBlockTxHashs(ctx, blockNumber)
		txHashes = append(txHashes, txHash)

		bz, _ := json.Marshal(&txHashes)
		store.Set(GetBlockIDBytes(blockNumber), bz)
	}
}

func (k *Keeper) GetTxNonceByHash(ctx sdk.Context, txHash string) (uint64, bool) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		bz := k.archiveStore.Get([]byte(txHash))
		if bz == nil {
			return 0, false
		}
		return sdk.BigEndianToUint64(bz), true
	} else {
		store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionHashToNonceKey)
		bz := store.Get([]byte(txHash))
		if bz == nil {
			return 0, false
		}

		nonce := sdk.BigEndianToUint64(bz)
		return nonce, true
	}
}

func (k *Keeper) StoreSetTransactionNonceByHash(ctx sdk.Context, txHash string, nonce uint64) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		k.archiveStore.Set([]byte(txHash), sdk.Uint64ToBigEndian(nonce))
	} else {
		store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionHashToNonceKey)
		store.Set([]byte(txHash), sdk.Uint64ToBigEndian(nonce))
	}
}

func (k *Keeper) StoreCronTransactionResult(ctx sdk.Context, cron types.Cron, tx types.CronTransactionResult) {

	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		k.archiveStore.Set(append(types.ArchiveStoreTxKey, strconv.FormatUint(tx.Nonce, 10)...), k.cdc.MustMarshal(&tx))
		k.archiveStore.Set(append(append(types.ArchiveStoreCronTxNonceKey, strconv.FormatUint(cron.Id, 10)...), strconv.FormatUint(tx.Nonce, 10)...), []byte{})
		return
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionResultKey)
	bz := k.cdc.MustMarshal(&tx)
	store.Set(GetTxIDBytes(tx.Nonce), bz)

	// Stockage uniquement du nonce dans l'index secondaire pour éviter les doublons
	storeByCronId := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.CronTransactionResultByCronIdKey, sdk.Uint64ToBigEndian(cron.Id)...))

	// ici on ne stocke que le nonce (très léger) comme référence
	storeByCronId.Set(GetTxIDBytes(tx.Nonce), []byte{}) // pas besoin de valeur car on récupère la donnée via le nonce dans le store principal
}
