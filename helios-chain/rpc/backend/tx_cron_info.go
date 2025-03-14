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
	if err != nil {
		return nil, err
	}
	return &res.Cron, err
}

func (b *Backend) GetCronByAddress(address common.Address) (*chronostypes.Cron, error) {

	req := &chronostypes.QueryGetCronByAddressRequest{
		Address: address.String(),
	}
	res, err := b.queryClient.Chronos.QueryGetCronByAddress(b.ctx, req)
	if err != nil {
		return nil, err
	}
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
	if err != nil || res.Crons == nil {
		return []chronostypes.Cron{}, err
	}
	return res.Crons, nil
}

func (b *Backend) GetAccountCronsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]chronostypes.Cron, error) {

	req := &chronostypes.QueryGetCronsByOwnerRequest{
		OwnerAddress: address.String(),
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	}
	res, err := b.queryClient.Chronos.QueryGetCronsByOwner(b.ctx, req)
	if err != nil || res.Crons == nil {
		return []chronostypes.Cron{}, err
	}
	return res.Crons, nil
}

func (b *Backend) GetCronTransactionByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionRPC, error) {

	req := &chronostypes.QueryGetCronTransactionByNonceRequest{
		Nonce: uint64(nonce),
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionByNonce(b.ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Transaction, nil
}

func (b *Backend) GetCronTransactionByHash(hash string) (*chronostypes.CronTransactionRPC, error) {

	req := &chronostypes.QueryGetCronTransactionByHashRequest{
		Hash: hash,
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionByHash(b.ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Transaction, nil
}

func (b *Backend) GetCronTransactionReceiptByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionReceiptRPC, error) {

	req := &chronostypes.QueryGetCronTransactionReceiptByNonceRequest{
		Nonce: uint64(nonce),
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionReceiptByNonce(b.ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Transaction, nil
}

func (b *Backend) GetCronTransactionReceiptByHash(hash string) (*chronostypes.CronTransactionReceiptRPC, error) {

	req := &chronostypes.QueryGetCronTransactionReceiptByHashRequest{
		Hash: hash,
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionReceiptByHash(b.ctx, req)
	if err != nil {
		return nil, err
	}
	return res.Transaction, nil
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
	if err != nil || res.Transactions == nil {
		return []*chronostypes.CronTransactionReceiptRPC{}, err
	}
	return res.Transactions, nil
}

func (b *Backend) GetCronTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error) {
	req := &chronostypes.QueryGetCronTransactionsByPageAndSizeRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionsByPageAndSize(b.ctx, req)
	if err != nil || res.Transactions == nil {
		return []*chronostypes.CronTransactionRPC{}, err
	}
	return res.Transactions, nil
}

func (b *Backend) GetBlockCronLogs(blockNumber uint64) ([]*ethtypes.Log, error) {

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
