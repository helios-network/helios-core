package backend

import (
	"fmt"
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	rpctypes "helios-core/helios-chain/rpc/types"
	"helios-core/helios-chain/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	"helios-core/helios-chain/precompiles/distribution"
	"helios-core/helios-chain/precompiles/erc20creator"
	"helios-core/helios-chain/precompiles/gov"
	"helios-core/helios-chain/precompiles/hyperion"
	"helios-core/helios-chain/precompiles/staking"

	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

// GetTransactionByHash returns the Ethereum format transaction identified by Ethereum transaction hash
func (b *Backend) GetTransactionByHash(txHash common.Hash) (*rpctypes.RPCTransaction, error) {
	res, err := b.GetTxByEthHash(txHash)
	hexTx := txHash.Hex()

	if err != nil {
		return b.getTransactionByHashPending(txHash)
	}

	block, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil {
		return nil, err
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
	if err != nil {
		return nil, err
	}

	// the `res.MsgIndex` is inferred from tx index, should be within the bound.
	msg, ok := tx.GetMsgs()[res.MsgIndex].(*evmtypes.MsgEthereumTx)
	if !ok {
		return nil, errors.New("invalid ethereum tx")
	}

	blockRes, err := b.rpcClient.BlockResults(b.ctx, &block.Block.Height)
	if err != nil {
		b.logger.Debug("block result not found", "height", block.Block.Height, "error", err.Error())
		return nil, nil
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		msgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		for i := range msgs {
			if msgs[i].Hash == hexTx {
				if i > math.MaxInt32 {
					return nil, errors.New("tx index overflow")
				}
				res.EthTxIndex = int32(i) //nolint:gosec // G115 G115 -- checked for int overflow already
				break
			}
		}
	}
	// if we still unable to find the eth tx index, return error, shouldn't happen.
	if res.EthTxIndex == -1 {
		return nil, errors.New("can't find index of ethereum tx")
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle the error for pruned node.
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", blockRes.Height, "error", err)
	}

	height := uint64(res.Height)    //#nosec G701 G115 -- checked for int overflow already
	index := uint64(res.EthTxIndex) //#nosec G701 G115 -- checked for int overflow already
	return rpctypes.NewTransactionFromMsg(
		msg,
		common.BytesToHash(block.BlockID.Hash.Bytes()),
		height,
		index,
		baseFee,
		b.chainID,
	)
}

// getTransactionByHashPending find pending tx from mempool
func (b *Backend) getTransactionByHashPending(txHash common.Hash) (*rpctypes.RPCTransaction, error) {
	hexTx := txHash.Hex()
	// try to find tx in mempool
	txs, err := b.PendingTransactions()
	if err != nil {
		b.logger.Debug("tx not found", "hash", hexTx, "error", err.Error())
		return nil, nil
	}

	for _, tx := range txs {
		msg, err := evmtypes.UnwrapEthereumMsg(tx, txHash)
		if err != nil {
			// not ethereum tx
			continue
		}

		if msg.Hash == hexTx {
			// use zero block values since it's not included in a block yet
			rpctx, err := rpctypes.NewTransactionFromMsg(
				msg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				b.chainID,
			)
			if err != nil {
				return nil, err
			}
			return rpctx, nil
		}
	}

	b.logger.Debug("tx not found", "hash", hexTx)
	return nil, nil
}

// GetGasUsed returns gasUsed from transaction
func (b *Backend) GetGasUsed(res *types.TxResult, price *big.Int, gas uint64) uint64 {
	// patch gasUsed if tx is reverted and happened before height on which fixed was introduced
	// to return real gas charged
	// more info at https://github.com/evmos/ethermint/pull/1557
	if res.Failed && res.Height < b.cfg.JSONRPC.FixRevertGasRefundHeight {
		return new(big.Int).Mul(price, new(big.Int).SetUint64(gas)).Uint64()
	}
	return res.GasUsed
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (b *Backend) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	hexTx := hash.Hex()
	b.logger.Debug("eth_getTransactionReceipt", "hash", hexTx)

	res, err := b.GetTxByEthHash(hash)
	if err != nil {
		b.logger.Debug("tx not found", "hash", hexTx, "error", err.Error())
		return nil, nil
	}

	resBlock, err := b.TendermintBlockByNumber(rpctypes.BlockNumber(res.Height))
	if err != nil {
		b.logger.Debug("block not found", "height", res.Height, "error", err.Error())
		return nil, nil
	}

	tx, err := b.clientCtx.TxConfig.TxDecoder()(resBlock.Block.Txs[res.TxIndex])
	if err != nil {
		b.logger.Debug("decoding failed", "error", err.Error())
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	ethMsg := tx.GetMsgs()[res.MsgIndex].(*evmtypes.MsgEthereumTx)

	txData, err := evmtypes.UnpackTxData(ethMsg.Data)
	if err != nil {
		b.logger.Error("failed to unpack tx data", "error", err.Error())
		return nil, err
	}

	cumulativeGasUsed := uint64(0)
	blockRes, err := b.rpcClient.BlockResults(b.ctx, &res.Height)
	if err != nil {
		b.logger.Debug("failed to retrieve block results", "height", res.Height, "error", err.Error())
		return nil, nil
	}

	for _, txResult := range blockRes.TxsResults[0:res.TxIndex] {
		cumulativeGasUsed += uint64(txResult.GasUsed) //nolint:gosec // G115 -- checked for int overflow already
	}

	cumulativeGasUsed += res.CumulativeGasUsed

	var status hexutil.Uint
	if res.Failed {
		status = hexutil.Uint(ethtypes.ReceiptStatusFailed)
	} else {
		status = hexutil.Uint(ethtypes.ReceiptStatusSuccessful)
	}

	chainID, err := b.ChainID()
	if err != nil {
		return nil, err
	}

	from, err := ethMsg.GetSender(chainID.ToInt())
	if err != nil {
		return nil, err
	}

	// parse tx logs from events
	msgIndex := int(res.MsgIndex) // #nosec G701 -- checked for int overflow already
	logs, err := TxLogsFromEvents(blockRes.TxsResults[res.TxIndex].Events, msgIndex)
	if err != nil {
		b.logger.Debug("failed to parse logs", "hash", hexTx, "error", err.Error())
	}

	if res.EthTxIndex == -1 {
		// Fallback to find tx index by iterating all valid eth transactions
		msgs := b.EthMsgsFromTendermintBlock(resBlock, blockRes)
		for i := range msgs {
			if msgs[i].Hash == hexTx {
				res.EthTxIndex = int32(i) //nolint:gosec // G115 G115
				break
			}
		}
	}
	// return error if still unable to find the eth tx index
	if res.EthTxIndex == -1 {
		return nil, errors.New("can't find index of ethereum tx")
	}

	receipt := map[string]interface{}{
		// Consensus fields: These fields are defined by the Yellow Paper
		"status":            status,
		"cumulativeGasUsed": hexutil.Uint64(cumulativeGasUsed),
		"logsBloom":         ethtypes.BytesToBloom(ethtypes.LogsBloom(logs)),
		"logs":              logs,

		// Implementation fields: These fields are added by geth when processing a transaction.
		// They are stored in the chain database.
		"transactionHash": hash,
		"contractAddress": nil,
		"gasUsed":         hexutil.Uint64(b.GetGasUsed(res, txData.GetGasPrice(), txData.GetGas())),

		// Inclusion information: These fields provide information about the inclusion of the
		// transaction corresponding to this receipt.
		"blockHash":        common.BytesToHash(resBlock.Block.Header.Hash()).Hex(),
		"blockNumber":      hexutil.Uint64(res.Height),     //nolint:gosec // G115
		"transactionIndex": hexutil.Uint64(res.EthTxIndex), //nolint:gosec // G115

		// sender and receiver (contract or EOA) addreses
		"from": from,
		"to":   txData.GetTo(),
		"type": hexutil.Uint(ethMsg.AsTransaction().Type()),
	}

	if logs == nil {
		receipt["logs"] = [][]*ethtypes.Log{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if txData.GetTo() == nil {
		receipt["contractAddress"] = crypto.CreateAddress(from, txData.GetNonce())
	}

	if dynamicTx, ok := txData.(*evmtypes.DynamicFeeTx); ok {
		baseFee, err := b.BaseFee(blockRes)
		if err != nil {
			// tolerate the error for pruned node.
			b.logger.Error("fetch basefee failed, node is pruned?", "height", res.Height, "error", err)
		} else {
			receipt["effectiveGasPrice"] = hexutil.Big(*dynamicTx.EffectiveGasPrice(baseFee))
		}
	}

	return receipt, nil
}

// GetTransactionLogs returns the transaction logs identified by hash.
func (b *Backend) GetTransactionLogs(hash common.Hash) ([]*ethtypes.Log, error) {
	hexTx := hash.Hex()

	res, err := b.GetTxByEthHash(hash)
	if err != nil {
		b.logger.Debug("tx not found", "hash", hexTx, "error", err.Error())
		return nil, nil
	}

	if res.Failed {
		// failed, return empty logs
		return nil, nil
	}

	resBlockResult, err := b.rpcClient.BlockResults(b.ctx, &res.Height)
	if err != nil {
		b.logger.Debug("block result not found", "number", res.Height, "error", err.Error())
		return nil, nil
	}

	// parse tx logs from events
	index := int(res.MsgIndex) // #nosec G701
	return TxLogsFromEvents(resBlockResult.TxsResults[res.TxIndex].Events, index)
}

// GetTransactionByBlockHashAndIndex returns the transaction identified by hash and index.
func (b *Backend) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	b.logger.Debug("eth_getTransactionByBlockHashAndIndex", "hash", hash.Hex(), "index", idx)
	sc, ok := b.clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		return nil, errors.New("invalid rpc client")
	}

	block, err := sc.BlockByHash(b.ctx, hash.Bytes())
	if err != nil {
		b.logger.Debug("block not found", "hash", hash.Hex(), "error", err.Error())
		return nil, nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "hash", hash.Hex())
		return nil, nil
	}

	return b.GetTransactionByBlockAndIndex(block, idx)
}

// GetTransactionByBlockNumberAndIndex returns the transaction identified by number and index.
func (b *Backend) GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	b.logger.Debug("eth_getTransactionByBlockNumberAndIndex", "number", blockNum, "index", idx)

	block, err := b.TendermintBlockByNumber(blockNum)
	if err != nil {
		b.logger.Debug("block not found", "height", blockNum.Int64(), "error", err.Error())
		return nil, nil
	}

	if block.Block == nil {
		b.logger.Debug("block not found", "height", blockNum.Int64())
		return nil, nil
	}

	return b.GetTransactionByBlockAndIndex(block, idx)
}

// GetTxByEthHash uses `/tx_query` to find transaction by ethereum tx hash
// TODO: Don't need to convert once hashing is fixed on Tendermint
// https://github.com/cometbft/cometbft/issues/6539
func (b *Backend) GetTxByEthHash(hash common.Hash) (*types.TxResult, error) {
	if b.indexer != nil {
		return b.indexer.GetByTxHash(hash)
	}

	// fallback to tendermint tx indexer
	query := fmt.Sprintf("%s.%s='%s'", evmtypes.TypeMsgEthereumTx, evmtypes.AttributeKeyEthereumTxHash, hash.Hex())
	txResult, err := b.queryTendermintTxIndexer(query, func(txs *rpctypes.ParsedTxs) *rpctypes.ParsedTx {
		return txs.GetTxByHash(hash)
	})
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetTxByEthHash %s", hash.Hex())
	}
	return txResult, nil
}

// GetTxByTxIndex uses `/tx_query` to find transaction by tx index of valid ethereum txs
func (b *Backend) GetTxByTxIndex(height int64, index uint) (*types.TxResult, error) {
	int32Index := int32(index) //nolint:gosec // G115 G115 -- checked for int overflow already
	if b.indexer != nil {
		return b.indexer.GetByBlockAndIndex(height, int32Index)
	}

	// fallback to tendermint tx indexer
	query := fmt.Sprintf("tx.height=%d AND %s.%s=%d",
		height, evmtypes.TypeMsgEthereumTx,
		evmtypes.AttributeKeyTxIndex, index,
	)
	txResult, err := b.queryTendermintTxIndexer(query, func(txs *rpctypes.ParsedTxs) *rpctypes.ParsedTx {
		return txs.GetTxByTxIndex(int(index)) //#nosec G701 G115 -- checked for int overflow already
	})
	if err != nil {
		return nil, errorsmod.Wrapf(err, "GetTxByTxIndex %d %d", height, index)
	}
	return txResult, nil
}

// queryTendermintTxIndexer query tx in tendermint tx indexer
func (b *Backend) queryTendermintTxIndexer(query string, txGetter func(*rpctypes.ParsedTxs) *rpctypes.ParsedTx) (*types.TxResult, error) {
	resTxs, err := b.clientCtx.Client.TxSearch(b.ctx, query, false, nil, nil, "")
	if err != nil {
		return nil, err
	}
	if len(resTxs.Txs) == 0 {
		return nil, errors.New("ethereum tx not found")
	}
	txResult := resTxs.Txs[0]
	if !rpctypes.TxSucessOrExpectedFailure(&txResult.TxResult) {
		return nil, errors.New("invalid ethereum tx")
	}

	var tx sdk.Tx
	if txResult.TxResult.Code != 0 {
		// it's only needed when the tx exceeds block gas limit
		tx, err = b.clientCtx.TxConfig.TxDecoder()(txResult.Tx)
		if err != nil {
			return nil, fmt.Errorf("invalid ethereum tx")
		}
	}

	return rpctypes.ParseTxIndexerResult(txResult, tx, txGetter)
}

// GetTransactionByBlockAndIndex is the common code shared by `GetTransactionByBlockNumberAndIndex` and `GetTransactionByBlockHashAndIndex`.
func (b *Backend) GetTransactionByBlockAndIndex(block *tmrpctypes.ResultBlock, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	blockRes, err := b.rpcClient.BlockResults(b.ctx, &block.Block.Height)
	if err != nil {
		return nil, nil
	}

	var msg *evmtypes.MsgEthereumTx
	// find in tx indexer
	res, err := b.GetTxByTxIndex(block.Block.Height, uint(idx))
	if err == nil {
		tx, err := b.clientCtx.TxConfig.TxDecoder()(block.Block.Txs[res.TxIndex])
		if err != nil {
			b.logger.Debug("invalid ethereum tx", "height", block.Block.Header, "index", idx)
			return nil, nil
		}

		var ok bool
		// msgIndex is inferred from tx events, should be within bound.
		msg, ok = tx.GetMsgs()[res.MsgIndex].(*evmtypes.MsgEthereumTx)
		if !ok {
			b.logger.Debug("invalid ethereum tx", "height", block.Block.Header, "index", idx)
			return nil, nil
		}
	} else {
		i := int(idx) //#nosec G115 G701
		ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		if i >= len(ethMsgs) {
			b.logger.Debug("block txs index out of bound", "index", i)
			return nil, nil
		}

		msg = ethMsgs[i]
	}

	baseFee, err := b.BaseFee(blockRes)
	if err != nil {
		// handle the error for pruned node.
		b.logger.Error("failed to fetch Base Fee from prunned block. Check node prunning configuration", "height", block.Block.Height, "error", err)
	}

	height := uint64(block.Block.Height) //nolint:gosec // G115 -- checked for int overflow already
	index := uint64(idx)                 // #nosec G701 -- checked for int overflow already
	return rpctypes.NewTransactionFromMsg(
		msg,
		common.BytesToHash(block.Block.Hash()),
		height,
		index,
		baseFee,
		b.chainID,
	)
}

func (b *Backend) GetTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error) {
	if page == 0 || size == 0 {
		return nil, errors.New("page and size must be greater than 0")
	}
	pageNum := uint64(page)
	pageSize := uint64(size)

	transactions := make([]*rpctypes.RPCTransaction, 0, pageSize)

	// Get pending transactions
	pendingTxs, err := b.PendingTransactions()
	if err != nil {
		return nil, err
	}
	// offSet is the index of the first transaction in the page (0-indexed)
	offSet := (pageNum - 1) * pageSize

	// pendingTxCount is the number of pending transactions in the mempool
	pendingTxCount := uint64(len(pendingTxs))

	// add mempool txs if we are in the page range
	if offSet < pendingTxCount {
		pendingEndPosition := min(offSet+pageSize, pendingTxCount)

		for i := offSet; i < pendingEndPosition; i++ {
			msg, err := evmtypes.UnwrapEthereumMsgTx(pendingTxs[i])
			if err != nil {
				return nil, fmt.Errorf("failed to unwrap ethereum tx: %w", err)
			}

			rpctx, err := rpctypes.NewTransactionFromMsg(
				msg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				b.chainID,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create transaction from msg: %w", err)
			}
			transactions = append(transactions, rpctx)

			if uint64(len(transactions)) >= pageSize {
				return transactions, nil
			}
		}
	}

	// we need to find txs in confirmed blocks
	// get latest block number
	latestBlockNumber, err := b.BlockNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block number: %w", err)
	}

	// Calculate how many more transactions we need
	txCount := uint64(0)

	// use BlockchainInfo to estimate the target block based on num_txs using batches of 100
	const batchSize = int64(100)
	maxHeight := int64(latestBlockNumber)
	for maxHeight > 0 && uint64(len(transactions)) < pageSize {
		// Calculate minHeight safely
		minHeight := int64(1)
		if maxHeight > int64(batchSize) {
			minHeight = maxHeight - int64(batchSize) + 1
		}

		b.logger.Debug("Querying BlockchainInfo",
			"minHeight", minHeight,
			"maxHeight", maxHeight)

		// get block metadata for the current batch
		info, err := b.clientCtx.Client.BlockchainInfo(b.ctx, int64(minHeight), int64(maxHeight))
		if err != nil {
			return nil, fmt.Errorf("failed to get block metadata: %w", err)
		}

		var targetBlock int64
		// inspect the number of txs in each block and find the block within the page range
		for i := 0; i < len(info.BlockMetas); i++ {
			txCount += uint64(info.BlockMetas[i].NumTxs)
			if uint64(txCount) >= offSet { // this block contains txs we need
				targetBlock = info.BlockMetas[i].Header.Height
				break
			}
		}
		if targetBlock == 0 {
			return nil, fmt.Errorf("txs not found for page %d and size %d", pageNum, pageSize)
		}

		for blockHeight := targetBlock; blockHeight > 0 && uint64(len(transactions)) < pageSize; blockHeight-- {

			block, err := b.rpcClient.Block(b.ctx, &blockHeight)
			if err != nil {
				return nil, fmt.Errorf("failed to get block %d: %w", blockHeight, err)
			}

			blockRes, err := b.rpcClient.BlockResults(b.ctx, &blockHeight)
			if err != nil {
				return nil, fmt.Errorf("failed to get block results for block %d: %w", blockHeight, err)
			}

			updatedOffset := offSet + uint64(len(transactions))
			txCount := uint64(0)
			ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
			for i, msg := range ethMsgs {

				// Skip eth messages until we reach our start position
				if txCount+uint64(i) < updatedOffset {
					continue
				}

				// Check if we've reached our page size
				if uint64(len(transactions)) >= pageSize {
					return transactions, nil
				}

				// construct the rpc transaction
				baseFee, err := b.BaseFee(blockRes)
				if err != nil {
					return nil, fmt.Errorf("failed to fetch Base Fee from prunned block. Check node prunning configuration: %w", err)
				}

				rpctx, err := rpctypes.NewTransactionFromMsg(
					msg,
					common.BytesToHash(block.Block.Hash()),
					uint64(block.Block.Height),
					uint64(i),
					baseFee,
					b.chainID,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create transaction from msg: %w", err)
				}
				transactions = append(transactions, rpctx)
			}
			txCount += uint64(len(ethMsgs))
		}
		maxHeight -= batchSize
	}

	return transactions, nil

}

func (b *Backend) GetLastTransactionsInfo(size hexutil.Uint64) (map[string]interface{}, error) {
	if !(size > 0 && size <= 50) {
		return nil, errors.New("size must be between 1 and 50")
	}

	b.logger.Debug("GetLastTransactionsInfo", "size", size)

	transactions, err := b.GetTransactionsByPageAndSize(1, size)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	for _, transaction := range transactions {
		if transaction == nil {
			return nil, errors.New("transaction is nil")
		}

		to := transaction.To

		switch to.Hex() {
		case "0x0000000000000000000000000000000000000800": // Staking Tx
			txData, err := b.decodeStakingTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode staking transaction")
			}
			return txData, nil
		case "0x0000000000000000000000000000000000000801": // Distribution Tx
		case "0x0000000000000000000000000000000000000802": // ICS Tx
		case "0x0000000000000000000000000000000000000803": // Vesting tx
		case "0x0000000000000000000000000000000000000804": // Bank Tx
		case "0x0000000000000000000000000000000000000805": // Proposal Tx
			txData, err := b.decodeGovTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode gov transaction")
			}
			return txData, nil
		case "0x0000000000000000000000000000000000000806": // ERC20 Creation Tx
			txData, err := b.decodeErc20CreationTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode erc20 creation transaction")
			}
			return txData, nil
		case "0x0000000000000000000000000000000000000830": // Cron Tx
		case "0x0000000000000000000000000000000000000900": // Bridge Tx
			txData, err := b.decodeBridgeTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode bridge transaction")
			}
			return txData, nil

		default:
		}
	}

	return nil, nil
}

func (b *Backend) decodeStakingTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	if transaction.Input == nil || len(transaction.Input) == 0 {
		return nil, errors.New("transaction input is empty")
	}

	stakingAbi, err := staking.LoadABI()

	if err != nil {
		return nil, errors.Wrap(err, "failed to load staking ABI")
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := stakingAbi.MethodById(sigdata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to find method by id")
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode staking transaction input")
	}

	methodName := method.Name

	if methodName == "delegate" || methodName == "undelegate" {
		amount, ok := decodedInput[2].(*big.Int)
		if !ok {
			return nil, errors.New("invalid amount")
		}
		denom, ok := decodedInput[3].(string)
		if !ok {
			return nil, errors.New("invalid denom")
		}
		// Create a map to hold the decoded values
		decodedValues := make(map[string]interface{})
		if methodName == "delegate" {
			decodedValues["type"] = "STAKE_IN"
		} else {
			decodedValues["type"] = "STAKE_OUT"
		}
		decodedValues["amount"] = amount.String()
		decodedValues["denom"] = denom
		return decodedValues, nil
	}

	return nil, nil
}

func (b *Backend) decodeErc20CreationTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	if transaction.Input == nil || len(transaction.Input) == 0 {
		return nil, errors.New("transaction input is empty")
	}

	erc20Abi, err := erc20creator.LoadABI()

	if err != nil {
		return nil, errors.Wrap(err, "failed to load erc20 ABI")
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := erc20Abi.MethodById(sigdata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to find method by id")
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode erc20 transaction input")
	}

	name, ok := decodedInput[0].(string)
	if !ok {
		return nil, errors.New("invalid name")
	}
	symbol, ok := decodedInput[1].(string)
	if !ok {
		return nil, errors.New("invalid symbol")
	}
	totalSupply, ok := decodedInput[2].(*big.Int)
	if !ok {
		return nil, errors.New("invalid total supply")
	}
	decimals, ok := decodedInput[3].(uint8)
	if !ok {
		return nil, errors.New("invalid decimals")
	}
	// Create a map to hold the decoded values
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "CREATE_ERC20"
	decodedValues["name"] = name
	decodedValues["symbol"] = symbol
	decodedValues["totalSupply"] = totalSupply.String()
	decodedValues["decimals"] = decimals

	return decodedValues, nil
}

func (b *Backend) decodeDistributionTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	if transaction.Input == nil || len(transaction.Input) == 0 {
		return nil, errors.New("transaction input is empty")
	}

	distributionAbi, err := distribution.LoadABI()

	if err != nil {
		return nil, errors.Wrap(err, "failed to load distribution ABI")
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := distributionAbi.MethodById(sigdata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to find method by id")
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode distribution transaction input")
	}

	validatorAddress, ok := decodedInput[1].(string)
	if !ok {
		return nil, errors.New("invalid validator address")
	}
	// Create a map to hold the decoded values
	decodedValues := make(map[string]interface{})
	decodedValues["validatorAddress"] = validatorAddress

	return decodedValues, nil
}

func (b *Backend) decodeBridgeTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	if transaction.Input == nil || len(transaction.Input) == 0 {
		return nil, errors.New("transaction input is empty")
	}

	bridgeAbi, err := hyperion.LoadABI()

	if err != nil {
		return nil, errors.Wrap(err, "failed to load bridge ABI")
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := bridgeAbi.MethodById(sigdata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to find method by id")
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode bridge transaction input")
	}

	methodName := method.Name

	if methodName == "delegate" || methodName == "undelegate" {
		amount, ok := decodedInput[2].(*big.Int)
		if !ok {
			return nil, errors.New("invalid amount")
		}
		denom, ok := decodedInput[3].(string)
		if !ok {
			return nil, errors.New("invalid denom")
		}
		// Create a map to hold the decoded values
		decodedValues := make(map[string]interface{})
		if methodName == "delegate" {
			decodedValues["type"] = "STAKE_IN"
		} else {
			decodedValues["type"] = "STAKE_OUT"
		}
		decodedValues["amount"] = amount.String()
		decodedValues["denom"] = denom
		return decodedValues, nil
	}

	return nil, nil
}

func (b *Backend) decodeGovTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	if transaction.Input == nil || len(transaction.Input) == 0 {
		return nil, errors.New("transaction input is empty")
	}

	govAbi, err := gov.LoadABI()

	if err != nil {
		return nil, errors.Wrap(err, "failed to load gov ABI")
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := govAbi.MethodById(sigdata)

	if err != nil {
		return nil, errors.Wrap(err, "failed to find method by id")
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode gov transaction input")
	}

	methodName := method.Name

	if methodName == "vote" {
		voter, ok := decodedInput[0].(common.Address)
		if !ok {
			return nil, errors.New("invalid voter")
		}
		proposalId, ok := decodedInput[1].(uint64)
		if !ok {
			return nil, errors.New("Invalid proposalId")
		}
		option, ok := decodedInput[2].(uint8)
		if !ok {
			return nil, errors.New("invalid option")
		}
		metadata, ok := decodedInput[3].(string)
		if !ok {
			return nil, errors.New("invalid metadata")
		}
		// Create a map to hold the decoded values
		decodedValues := make(map[string]interface{})
		decodedValues["type"] = "GOV_VOTE"
		decodedValues["voter"] = voter.String()
		decodedValues["proposalId"] = proposalId
		decodedValues["option"] = option
		decodedValues["metadata"] = metadata
		return decodedValues, nil
	}

	return nil, nil
}
