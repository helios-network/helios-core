package backend

import (
	chronostypes "helios-core/helios-chain/x/chronos/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (b *Backend) GetCron(id uint64) (*chronostypes.Cron, error) {

	req := &chronostypes.QueryGetCronRequest{
		Id: id,
	}
	res, err := b.queryClient.Chronos.QueryGetCron(b.ctx, req)
	return &res.Cron, err
}

func (b *Backend) GetCronsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]chronostypes.Cron, error) {

	req := &chronostypes.QueryGetCronsRequest{
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	}
	res, err := b.queryClient.Chronos.QueryGetCrons(b.ctx, req)
	return res.Crons, err
}

func (b *Backend) GetCronTransactionByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionRPC, error) {

	req := &chronostypes.QueryGetCronTransactionByNonceRequest{
		Nonce: uint64(nonce),
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionByNonce(b.ctx, req)
	return res.Transaction, err
}

func (b *Backend) GetCronTransactionReceiptByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionReceiptRPC, error) {

	req := &chronostypes.QueryGetCronTransactionReceiptByNonceRequest{
		Nonce: uint64(nonce),
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionReceiptByNonce(b.ctx, req)
	return res.Transaction, err
}

func (b *Backend) GetCronTransactionReceiptByHash(hash string) (*chronostypes.CronTransactionReceiptRPC, error) {

	req := &chronostypes.QueryGetCronTransactionReceiptByHashRequest{
		Hash: hash,
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionReceiptByHash(b.ctx, req)
	return res.Transaction, err
}

func (b *Backend) GetCronTransactionReceiptsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error) {
	req := &chronostypes.QueryGetCronTransactionReceiptsByPageAndSizeRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionReceiptsByPageAndSize(b.ctx, req)
	return res.Transactions, err
}

func (b *Backend) BlockCronLogs(blockNumber uint64) ([]*ethtypes.Log, error) {

	req := &chronostypes.QueryGetCronTransactionReceiptLogsByBlockNumberRequest{
		BlockNumber: blockNumber,
	}
	res, _ := b.queryClient.Chronos.QueryGetCronTransactionReceiptLogsByBlockNumber(b.ctx, req)
	unfiltered := make([]*ethtypes.Log, 0)

	for _, log := range res.Logs {
		topics := make([]common.Hash, len(log.Topics))
		for i, topic := range log.Topics {
			topics[i] = common.HexToHash(topic)
		}

		unfiltered = append(unfiltered, &ethtypes.Log{
			Address:     common.HexToAddress(log.Address),
			Topics:      topics,
			Data:        log.Data,
			BlockNumber: log.BlockNumber,
			TxHash:      common.HexToHash(log.TxHash),
			TxIndex:     uint(0),
			BlockHash:   common.HexToHash(log.BlockHash),
			Index:       uint(0),
			Removed:     log.Removed,
		})
	}

	return unfiltered, nil
}
