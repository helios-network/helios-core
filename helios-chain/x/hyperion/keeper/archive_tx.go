package keeper

import (
	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/testnet"
	"helios-core/helios-chain/x/hyperion/types"

	"cosmossdk.io/store/prefix"
	"github.com/Helios-Chain-Labs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

func (k *Keeper) StoreFinalizedTx(ctx sdk.Context, tx *types.TransferTx) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	lastIndex, err := k.FindLastFinalizedTxIndex(ctx, cmn.AnyToHexAddress(tx.Sender))
	if err != nil {
		return
	}
	tx.Index = lastIndex + 1

	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		store := ctx.ArchiveStore(k.storeKey)
		store.Set(append(types.ArchiveStoreFinalizedTxKey, types.GetArchiveStoreFinalizedTxKey(cmn.AnyToHexAddress(tx.Sender), tx.Index)...), k.cdc.MustMarshal(tx))
	} else {
		store := ctx.KVStore(k.storeKey)
		finalizedTxStore := prefix.NewStore(store, types.FinalizedTxKey)
		finalizedTxStore.Set(types.GetFinalizedTxKey(cmn.AnyToHexAddress(tx.Sender), tx.Index), k.cdc.MustMarshal(tx))
	}

	k.StoreLastFinalizedTxIndex(ctx, tx)

	if tx.DestAddress != tx.Sender {
		lastIndexOfDestAddress, err := k.FindLastFinalizedTxIndex(ctx, cmn.AnyToHexAddress(tx.DestAddress))
		if err != nil {
			return
		}
		tx.Index = lastIndexOfDestAddress + 1

		if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
			store := ctx.ArchiveStore(k.storeKey)
			store.Set(append(types.ArchiveStoreFinalizedTxKey, types.GetArchiveStoreFinalizedTxKey(cmn.AnyToHexAddress(tx.DestAddress), tx.Index)...), k.cdc.MustMarshal(tx))
		} else {
			store := ctx.KVStore(k.storeKey)
			finalizedTxStore := prefix.NewStore(store, types.FinalizedTxKey)
			finalizedTxStore.Set(types.GetFinalizedTxKey(cmn.AnyToHexAddress(tx.DestAddress), tx.Index), k.cdc.MustMarshal(tx))
		}
	}
}

func (k *Keeper) StoreLastFinalizedTxIndex(ctx sdk.Context, tx *types.TransferTx) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		lastFinalizedTxs, err := k.GetLastFinalizedTxIndex(ctx)
		if err != nil {
			return
		}

		lastFinalizedTxs.Txs = append(lastFinalizedTxs.Txs, tx)
		if len(lastFinalizedTxs.Txs) > 100 { // save only the last 100 txs
			lastFinalizedTxs.Txs = lastFinalizedTxs.Txs[len(lastFinalizedTxs.Txs)-100:]
		}

		store := ctx.ArchiveStore(k.storeKey)
		store.Set(types.ArchiveStoreLastFinalizedTxIndexKey, k.cdc.MustMarshal(&lastFinalizedTxs))
	} else {
		store := ctx.KVStore(k.storeKey)
		lastFinalizedTxIndexStore := prefix.NewStore(store, types.LastFinalizedTxIndexKey)

		lastFinalizedTxs, err := k.GetLastFinalizedTxIndex(ctx)
		if err != nil {
			return
		}

		lastFinalizedTxs.Txs = append(lastFinalizedTxs.Txs, tx)
		if len(lastFinalizedTxs.Txs) > 100 { // save only the last 100 txs
			lastFinalizedTxs.Txs = lastFinalizedTxs.Txs[len(lastFinalizedTxs.Txs)-100:]
		}

		lastFinalizedTxIndexStore.Set([]byte{0x0}, k.cdc.MustMarshal(&lastFinalizedTxs))
	}
}

func (k *Keeper) GetLastFinalizedTxIndex(ctx sdk.Context) (types.LastFinalizedTxIndex, error) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		store := ctx.ArchiveStore(k.storeKey)
		lastFinalizedTxs := store.Get(types.ArchiveStoreLastFinalizedTxIndexKey)
		if lastFinalizedTxs == nil {
			return types.LastFinalizedTxIndex{}, nil
		}
		var txs types.LastFinalizedTxIndex
		k.cdc.MustUnmarshal(lastFinalizedTxs, &txs)
		return txs, nil
	}

	store := ctx.KVStore(k.storeKey)
	lastFinalizedTxIndexStore := prefix.NewStore(store, types.LastFinalizedTxIndexKey)
	lastFinalizedTxs := lastFinalizedTxIndexStore.Get([]byte{0x0})
	if lastFinalizedTxs == nil {
		return types.LastFinalizedTxIndex{}, nil
	}
	var txs types.LastFinalizedTxIndex
	k.cdc.MustUnmarshal(lastFinalizedTxs, &txs)
	return txs, nil
}

func (k *Keeper) FindLastFinalizedTxIndex(ctx sdk.Context, addr common.Address) (uint64, error) {

	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		store := ctx.ArchiveStore(k.storeKey)
		finalizedTxStore := prefix.NewStore(store, types.ArchiveStoreFinalizedTxKey)
		iter := finalizedTxStore.ReverseIterator(PrefixRange(types.GetArchiveStoreFinalizedTxAddressPrefixKey(addr)))
		defer iter.Close()

		for ; iter.Valid(); iter.Next() {
			var tx types.TransferTx
			k.cdc.MustUnmarshal(iter.Value(), &tx)

			return tx.Index, nil
		}
		return 0, nil
	}

	store := ctx.KVStore(k.storeKey)
	finalizedTxStore := prefix.NewStore(store, types.FinalizedTxKey)
	iter := finalizedTxStore.ReverseIterator(PrefixRange(types.GetFinalizedTxAddressPrefixKey(addr)))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var tx types.TransferTx
		k.cdc.MustUnmarshal(iter.Value(), &tx)

		return tx.Index, nil
	}
	return 0, nil
}

func (k *Keeper) FindFinalizedTxs(ctx sdk.Context, addr common.Address) ([]*types.TransferTx, error) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		store := ctx.ArchiveStore(k.storeKey)
		finalizedTxStore := prefix.NewStore(store, types.ArchiveStoreFinalizedTxKey)
		iter := finalizedTxStore.Iterator(PrefixRange(types.GetArchiveStoreFinalizedTxAddressPrefixKey(addr)))
		defer iter.Close()

		txs := make([]*types.TransferTx, 0)
		for ; iter.Valid(); iter.Next() {
			var tx types.TransferTx
			k.cdc.MustUnmarshal(iter.Value(), &tx)
			txs = append(txs, &tx)
		}
		return txs, nil
	}

	store := ctx.KVStore(k.storeKey)
	finalizedTxStore := prefix.NewStore(store, types.FinalizedTxKey)
	iter := finalizedTxStore.Iterator(PrefixRange(types.GetFinalizedTxAddressPrefixKey(addr)))
	defer iter.Close()

	txs := make([]*types.TransferTx, 0)

	for ; iter.Valid(); iter.Next() {
		var tx types.TransferTx
		k.cdc.MustUnmarshal(iter.Value(), &tx)
		txs = append(txs, &tx)
	}

	return txs, nil
}

func (k *Keeper) FindFinalizedTxsByIndexToIndex(ctx sdk.Context, addr common.Address, startIndex uint64, endIndex uint64) ([]*types.TransferTx, error) {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		store := ctx.ArchiveStore(k.storeKey)
		finalizedTxStore := prefix.NewStore(store, types.ArchiveStoreFinalizedTxKey)
		start, _ := PrefixRange(types.GetArchiveStoreFinalizedTxAddressAndTxIndexPrefixKey(addr, startIndex+1))
		end, _ := PrefixRange(types.GetArchiveStoreFinalizedTxAddressAndTxIndexPrefixKey(addr, endIndex+1))
		iter := finalizedTxStore.Iterator(start, end)
		defer iter.Close()

		txs := make([]*types.TransferTx, 0)

		for ; iter.Valid(); iter.Next() {
			var tx types.TransferTx
			k.cdc.MustUnmarshal(iter.Value(), &tx)

			if tx.Index > endIndex+1 {
				break
			}
			txs = append(txs, &tx)
		}
		return txs, nil
	}

	store := ctx.KVStore(k.storeKey)
	finalizedTxStore := prefix.NewStore(store, types.FinalizedTxKey)
	start, _ := PrefixRange(types.GetFinalizedTxAddressAndTxIndexPrefixKey(addr, startIndex+1))
	end, _ := PrefixRange(types.GetFinalizedTxAddressAndTxIndexPrefixKey(addr, endIndex+1))
	iter := finalizedTxStore.Iterator(start, end)
	defer iter.Close()

	txs := make([]*types.TransferTx, 0)

	for ; iter.Valid(); iter.Next() {
		var tx types.TransferTx
		k.cdc.MustUnmarshal(iter.Value(), &tx)

		if tx.Index > endIndex+1 {
			break
		}
		txs = append(txs, &tx)
	}

	return txs, nil
}
