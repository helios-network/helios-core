package backend

import (
	chronostypes "helios-core/helios-chain/x/chronos/types"

	rpctypes "helios-core/helios-chain/rpc/types"

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

func (b *Backend) GetCronTransactionReceiptsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error) {
	req := &chronostypes.QueryGetCronTransactionReceiptsByPageAndSizeRequest{
		Address: address.String(),
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

func (b *Backend) GetCronTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error) {
	req := &chronostypes.QueryGetCronTransactionsByPageAndSizeRequest{
		Address: address.String(),
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

func (b *Backend) GetAllCronTransactionReceiptsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error) {
	req := &chronostypes.QueryGetAllCronTransactionReceiptsByPageAndSizeRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	}
	res, err := b.queryClient.Chronos.QueryGetAllCronTransactionReceiptsByPageAndSize(b.ctx, req)
	if err != nil || res.Transactions == nil {
		return []*chronostypes.CronTransactionReceiptRPC{}, err
	}
	return res.Transactions, nil
}

func (b *Backend) GetAllCronTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error) {
	req := &chronostypes.QueryGetAllCronTransactionsByPageAndSizeRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	}
	res, err := b.queryClient.Chronos.QueryGetAllCronTransactionsByPageAndSize(b.ctx, req)
	if err != nil || res.Transactions == nil {
		return []*chronostypes.CronTransactionRPC{}, err
	}
	return res.Transactions, nil
}

func (b *Backend) GetAllCronTransactionReceiptsByBlockNumber(blockNum rpctypes.BlockNumber) ([]*chronostypes.CronTransactionReceiptRPC, error) {

	blockNumber := blockNum.TmHeight()

	if blockNumber == nil {
		block, err := b.TendermintBlockByNumber(blockNum)
		if err != nil {
			b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
			return []*chronostypes.CronTransactionReceiptRPC{}, nil
		}
		if block.Block == nil {
			b.logger.Debug("block not found", "height", blockNum.Int64())
			return []*chronostypes.CronTransactionReceiptRPC{}, nil
		}
		blockNumber = &block.Block.Height
	}

	req := &chronostypes.QueryGetCronTransactionReceiptsByBlockNumberRequest{
		BlockNumber: uint64(*blockNumber),
	}
	res, err := b.queryClient.Chronos.QueryGetCronTransactionReceiptsByBlockNumber(b.ctx, req)
	if err != nil || res.Transactions == nil {
		return []*chronostypes.CronTransactionReceiptRPC{}, err
	}
	return res.Transactions, nil
}

func (b *Backend) GetBlockCronLogs(blockNum rpctypes.BlockNumber) ([]*ethtypes.Log, error) {
	unfiltered := make([]*ethtypes.Log, 0)
	blockNumber := blockNum.TmHeight()

	if blockNumber == nil {
		block, err := b.TendermintBlockByNumber(blockNum)
		if err != nil {
			b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
			return unfiltered, nil
		}
		if block.Block == nil {
			b.logger.Debug("block not found", "height", blockNum.Int64())
			return unfiltered, nil
		}
		blockNumber = &block.Block.Height
	}

	req := &chronostypes.QueryGetCronTransactionReceiptLogsByBlockNumberRequest{
		BlockNumber: uint64(*blockNumber),
	}
	res, _ := b.queryClient.Chronos.QueryGetCronTransactionReceiptLogsByBlockNumber(b.ctx, req)

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

func (b *Backend) GetCronStatistics() (*chronostypes.CronStatistics, error) {
	req := &chronostypes.QueryGetCronStatisticsRequest{}
	res, err := b.queryClient.Chronos.QueryGetCronStatistics(b.ctx, req)
	if err != nil {
		return nil, err
	}
	return &res.Statistics, nil
}
