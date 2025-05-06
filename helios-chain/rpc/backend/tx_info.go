package backend

import (
	"fmt"
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	rpctypes "helios-core/helios-chain/rpc/types"
	"helios-core/helios-chain/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	"helios-core/helios-chain/precompiles/chronos"
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

	if page == 0 || size == 0 || size > 100 {
		return nil, errors.New("invalid page or size parameters")
	}

	// Transactions accumulator
	transactions := make([]*rpctypes.RPCTransaction, 0, size)

	// utilities
	counter := uint64(0)
	skip := (uint64(page) - 1) * uint64(size)

	// Handle pending transactions first
	pendingTxs, _ := b.PendingTransactions()
	for _, tx := range pendingTxs {
		msg, err := evmtypes.UnwrapEthereumMsgTx(tx)
		if err != nil {
			continue
		}
		rpctx, err := rpctypes.NewTransactionFromMsg(msg, common.Hash{}, 0, 0, nil, b.chainID)
		if err != nil {
			continue
		}
		if counter >= skip {
			transactions = append(transactions, rpctx)
			if uint64(len(transactions)) >= uint64(size) {
				return transactions, nil
			}
		}
		counter++
	}

	// Fetch relevant blocks heights once
	heightsWhereFindingTx := make([]int64, 0)
	batchSize := int64(1000)
	latestBlockNumber, err := b.BlockNumber()
	if err != nil {
		return nil, err
	}
	maxHeight := int64(latestBlockNumber)
	totalMatchedTxs := uint64(0)

	for maxHeight > 0 && totalMatchedTxs < skip+uint64(size) {
		minHeight := maxHeight - batchSize + 1
		if minHeight < 1 {
			minHeight = 1
		}

		info, err := b.clientCtx.Client.BlockchainLocateNotEmptyBlocksInfo(b.ctx, minHeight, maxHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to get block metadata: %w", err)
		}

		for _, blockMeta := range info.BlockMetas {
			if blockMeta.NumTxs > 0 {
				heightsWhereFindingTx = append(heightsWhereFindingTx, blockMeta.Header.Height)
				totalMatchedTxs += uint64(blockMeta.NumTxs) // Compteur explicite ici
			}
		}
		maxHeight = minHeight - 1
		if minHeight == 1 {
			break
		}
	}

	// Reverse iterate relevant heights to maintain asc order
	for i := 0; i < len(heightsWhereFindingTx) && uint64(len(transactions)) < uint64(size); i++ {
		blockHeight := heightsWhereFindingTx[i]
		block, err := b.GetBlockByNumber(rpctypes.BlockNumber(blockHeight), true)
		if err != nil {
			continue
		}
		txs, ok := block["transactions"].([]interface{})
		if !ok {
			continue
		}
		for _, tx := range txs {
			ethTx, ok := tx.(*rpctypes.RPCTransaction)
			if !ok {
				continue
			}
			if counter >= skip {
				transactions = append(transactions, ethTx)
				if uint64(len(transactions)) >= uint64(size) {
					return transactions, nil
				}
			}
			counter++
		}
	}

	return transactions, nil
}

func (b *Backend) ParseTransactions(txs []*rpctypes.RPCTransaction) ([]*rpctypes.ParsedRPCTransaction, error) {
	transactions := make([]*rpctypes.ParsedRPCTransaction, 0, len(txs))

	for _, transaction := range txs {
		if transaction == nil {
			return nil, errors.New("transaction is nil")
		}

		tx := &rpctypes.ParsedRPCTransaction{
			RawTransaction: *transaction,
		}
		to := transaction.To

		switch to.Hex() {
		case "0x0000000000000000000000000000000000000800": // Staking Tx
			txData, err := b.decodeStakingTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode staking transaction")
			}
			tx.ParsedInfo = txData
		case "0x0000000000000000000000000000000000000801": // Distribution Tx
			txData, err := b.decodeDistributionTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode distribution transaction")
			}
			tx.ParsedInfo = txData
		case "0x0000000000000000000000000000000000000805": // Proposal Tx
			txData, err := b.decodeGovTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode gov transaction")
			}
			tx.ParsedInfo = txData
		case "0x0000000000000000000000000000000000000806": // ERC20 Creation Tx
			txData, err := b.decodeErc20CreationTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode erc20 creation transaction")
			}
			tx.ParsedInfo = txData
		case "0x0000000000000000000000000000000000000830": // Cron Tx
			txData, err := b.decodeCronTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode bridge transaction")
			}
			tx.ParsedInfo = txData
		case "0x0000000000000000000000000000000000000900": // Bridge Tx
			txData, err := b.decodeBridgeTransaction(transaction)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode bridge transaction")
			}
			tx.ParsedInfo = txData
		default:
			txData := make(map[string]interface{})
			txData["type"] = "UNKNOWN"
			tx.ParsedInfo = txData

		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (b *Backend) GetLastTransactionsInfo(size hexutil.Uint64) ([]*rpctypes.ParsedRPCTransaction, error) {
	if !(size > 0 && size <= 50) {
		return nil, errors.New("size must be between 1 and 50")
	}

	rawTxs, err := b.GetTransactionsByPageAndSize(1, size)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return b.ParseTransactions(rawTxs)
}

func (b *Backend) decodeStakingTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "UNKNOWN"
	if len(transaction.Input) == 0 {
		return decodedValues, nil
	}

	stakingAbi, err := staking.LoadABI()

	if err != nil {
		b.logger.Error("failed to load staking ABI", "error", err)
		return decodedValues, nil
	}

	if len(transaction.Input) < 4 {
		b.logger.Error("invalid transaction input length", "input", transaction.Input)
		return decodedValues, nil
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := stakingAbi.MethodById(sigdata)

	if err != nil {
		b.logger.Error("failed to find method by id", "error", err)
		return decodedValues, nil
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		b.logger.Error("failed to decode staking transaction input", "error", err)
		return decodedValues, nil
	}

	methodName := method.Name

	if methodName == "delegate" && len(decodedInput) >= 3 || methodName == "undelegate" && len(decodedInput) >= 3 {
		amount, ok := decodedInput[2].(*big.Int)
		if !ok {
			b.logger.Error("invalid amount", "input", transaction.Input)
			return decodedValues, nil
		}
		denom, ok := decodedInput[3].(string)
		if !ok {
			b.logger.Error("invalid denom", "input", transaction.Input)
			return decodedValues, nil
		}
		// Create a map to hold the decoded values
		if methodName == "delegate" {
			decodedValues["type"] = "STAKE_IN"
		} else {
			decodedValues["type"] = "STAKE_OUT"
		}
		decodedValues["amount"] = amount.String()
		decodedValues["denom"] = denom
		return decodedValues, nil
	}
	return decodedValues, nil
}

func (b *Backend) decodeErc20CreationTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "UNKNOWN"
	if len(transaction.Input) == 0 {
		return decodedValues, nil
	}

	erc20Abi, err := erc20creator.LoadABI()

	if err != nil {
		b.logger.Error("failed to load erc20 ABI", "error", err)
		return decodedValues, nil
	}

	if len(transaction.Input) < 4 {
		b.logger.Error("invalid transaction input length", "input", transaction.Input)
		return decodedValues, nil
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := erc20Abi.MethodById(sigdata)

	if err != nil {
		b.logger.Error("failed to find method by id", "error", err)
		return decodedValues, nil
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		b.logger.Error("failed to decode erc20 transaction input", "error", err)
		return decodedValues, nil
	}

	methodName := method.Name

	if methodName == "createErc20" && len(decodedInput) >= 4 {
		name, ok := decodedInput[0].(string)
		if !ok {
			b.logger.Error("invalid name", "input", transaction.Input)
			return decodedValues, nil
		}
		symbol, ok := decodedInput[1].(string)
		if !ok {
			b.logger.Error("invalid symbol", "input", transaction.Input)
			return decodedValues, nil
		}
		totalSupply, ok := decodedInput[2].(*big.Int)
		if !ok {
			b.logger.Error("invalid total supply", "input", transaction.Input)
			return decodedValues, nil
		}
		decimals, ok := decodedInput[3].(uint8)
		if !ok {
			b.logger.Error("invalid decimals", "input", transaction.Input)
			return decodedValues, nil
		}
		// Create a map to hold the decoded values
		decodedValues["type"] = "CREATE_ERC20"
		decodedValues["name"] = name
		decodedValues["symbol"] = symbol
		decodedValues["totalSupply"] = totalSupply.String()
		decodedValues["decimals"] = decimals

		return decodedValues, nil
	}
	return decodedValues, nil
}

func (b *Backend) decodeDistributionTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "UNKNOWN"
	if len(transaction.Input) == 0 {
		return decodedValues, nil
	}

	distributionAbi, err := distribution.LoadABI()

	if err != nil {
		b.logger.Error("failed to load distribution ABI", "error", err)
		return decodedValues, nil
	}

	if len(transaction.Input) < 4 {
		b.logger.Error("invalid transaction input length", "input", transaction.Input)
		return decodedValues, nil
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := distributionAbi.MethodById(sigdata)

	if err != nil {
		b.logger.Error("failed to find method by id", "error", err)
		return decodedValues, nil
	}

	methodName := method.Name

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		b.logger.Error("failed to decode distribution transaction input", "error", err)
		return decodedValues, nil
	}

	if methodName == "withdrawDelegatorRewards" && len(decodedInput) >= 2 {
		delegatorAddress, ok := decodedInput[0].(common.Address)
		if !ok {
			b.logger.Error("invalid delegator address", "input", transaction.Input)
			return decodedValues, nil
		}
		validatorAddress, ok := decodedInput[1].(common.Address)
		if !ok {
			b.logger.Error("invalid validator address", "input", transaction.Input)
			return decodedValues, nil
		}
		// Create a map to hold the decoded values
		decodedValues["delegatorAddress"] = delegatorAddress
		decodedValues["validatorAddress"] = validatorAddress

		return decodedValues, nil
	}
	return decodedValues, nil
}

func (b *Backend) decodeBridgeTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "UNKNOWN"
	if len(transaction.Input) == 0 {
		return decodedValues, nil
	}

	bridgeAbi, err := hyperion.LoadABI()

	if err != nil {
		b.logger.Error("failed to load bridge ABI", "error", err)
		return decodedValues, nil
	}

	if len(transaction.Input) < 4 {
		b.logger.Error("invalid transaction input length", "input", transaction.Input)
		return decodedValues, nil
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := bridgeAbi.MethodById(sigdata)

	if err != nil {
		b.logger.Error("failed to find method by id", "error", err)
		return decodedValues, nil
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		b.logger.Error("failed to decode bridge transaction input", "error", err)
		return decodedValues, nil
	}

	methodName := method.Name

	if methodName == "sendToChain" && len(decodedInput) >= 5 {
		chainId, ok := decodedInput[0].(uint64)
		if !ok {
			b.logger.Error("invalid chainId", "input", transaction.Input)
			return decodedValues, nil
		}
		destAddress, ok := decodedInput[1].(string)
		if !ok {
			b.logger.Error("invalid destAddress", "input", transaction.Input)
			return decodedValues, nil
		}
		contractAddress, ok := decodedInput[2].(common.Address)
		if !ok {
			b.logger.Error("invalid contractAddress", "input", transaction.Input)
			return decodedValues, nil
		}
		amount, ok := decodedInput[3].(*big.Int)
		if !ok {
			b.logger.Error("invalid amount", "input", transaction.Input)
			return decodedValues, nil
		}
		bridgeFee, ok := decodedInput[4].(*big.Int)
		if !ok {
			b.logger.Error("invalid bridgeFee", "input", transaction.Input)
			return decodedValues, nil
		}
		// Create a map to hold the decoded values
		decodedValues["type"] = "BRIDGE_IN"
		decodedValues["chainId"] = chainId
		decodedValues["destAddress"] = destAddress
		decodedValues["contractAddress"] = contractAddress.String()
		decodedValues["amount"] = amount.String()
		decodedValues["bridgeFee"] = bridgeFee.String()
		return decodedValues, nil
	}
	return decodedValues, nil
}

func (b *Backend) decodeGovTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "UNKNOWN"
	if len(transaction.Input) == 0 {
		return decodedValues, nil
	}

	govAbi, err := gov.LoadABI()

	if err != nil {
		b.logger.Error("failed to load gov ABI", "error", err)
		return decodedValues, nil
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := govAbi.MethodById(sigdata)

	if err != nil {
		b.logger.Error("failed to find method by id", "error", err)
		return decodedValues, nil
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		b.logger.Error("failed to decode gov transaction input", "error", err)
		return decodedValues, nil
	}

	methodName := method.Name

	if methodName == "vote" && len(decodedInput) >= 3 {
		voter, ok := decodedInput[0].(common.Address)
		if !ok {
			b.logger.Error("invalid voter", "input", transaction.Input)
			return decodedValues, nil
		}
		proposalId, ok := decodedInput[1].(uint64)
		if !ok {
			b.logger.Error("Invalid proposalId", "input", transaction.Input)
			return decodedValues, nil
		}
		option, ok := decodedInput[2].(uint8)
		if !ok {
			b.logger.Error("invalid option", "input", transaction.Input)
			return decodedValues, nil
		}
		metadata, ok := decodedInput[3].(string)
		if !ok {
			b.logger.Error("invalid metadata", "input", transaction.Input)
			return decodedValues, nil
		}
		// Create a map to hold the decoded values
		decodedValues["type"] = "GOV_VOTE"
		decodedValues["voter"] = voter.String()
		decodedValues["proposalId"] = proposalId
		decodedValues["option"] = option
		decodedValues["metadata"] = metadata
		return decodedValues, nil
	}
	return decodedValues, nil
}

func (b *Backend) decodeCronTransaction(transaction *rpctypes.RPCTransaction) (map[string]interface{}, error) {
	decodedValues := make(map[string]interface{})
	decodedValues["type"] = "UNKNOWN"
	if len(transaction.Input) == 0 {
		return decodedValues, nil
	}

	chronosAbi, err := chronos.LoadABI()

	if err != nil {
		b.logger.Error("failed to load chronos ABI", "error", err)
		return decodedValues, nil
	}

	sigdata := transaction.Input[:4]
	inputData := transaction.Input[4:]

	method, err := chronosAbi.MethodById(sigdata)

	if err != nil {
		b.logger.Error("failed to find method by id", "error", err)
		return decodedValues, nil
	}

	decodedInput, err := method.Inputs.Unpack(inputData)
	if err != nil {
		b.logger.Error("failed to decode chronos transaction input", "error", err)
		return decodedValues, nil
	}

	methodName := method.Name

	if methodName == "createCron" && len(decodedInput) >= 9 {
		contractAddress, ok := decodedInput[0].(common.Address)
		if !ok {
			b.logger.Error("invalid contractAddress", "input", transaction.Input)
			return decodedValues, nil
		}
		abi, ok := decodedInput[1].(string)
		if !ok {
			b.logger.Error("invalid abi", "input", transaction.Input)
			return decodedValues, nil
		}
		methodName, ok := decodedInput[2].(string)
		if !ok {
			b.logger.Error("invalid methodName", "input", transaction.Input)
			return decodedValues, nil
		}
		frequency, ok := decodedInput[4].(uint64)
		if !ok {
			b.logger.Error("invalid frequency", "input", transaction.Input)
			return decodedValues, nil
		}
		expirationBlock, ok := decodedInput[5].(uint64)
		if !ok {
			b.logger.Error("invalid expirationBlock", "input", transaction.Input)
			return decodedValues, nil
		}
		gasLimit, ok := decodedInput[6].(uint64)
		if !ok {
			b.logger.Error("invalid gasLimit", "input", transaction.Input)
			return decodedValues, nil
		}
		maxGasPrice, ok := decodedInput[7].(*big.Int)
		if !ok {
			b.logger.Error("invalid maxGasPrice", "input", transaction.Input)
			return decodedValues, nil
		}
		amountToDeposit, ok := decodedInput[8].(*big.Int)
		if !ok {
			b.logger.Error("invalid amountToDeposit", "input", transaction.Input)
			return decodedValues, nil
		}
		// Create a map to hold the decoded values
		decodedValues["type"] = "CREATE_CRON"
		decodedValues["contractAddress"] = contractAddress.String()
		decodedValues["abi"] = abi
		decodedValues["methodName"] = methodName
		decodedValues["frequency"] = frequency
		decodedValues["expirationBlock"] = expirationBlock
		decodedValues["gasLimit"] = gasLimit
		decodedValues["maxGasPrice"] = maxGasPrice.String()
		decodedValues["amountToDeposit"] = amountToDeposit.String()
		return decodedValues, nil
	}
	return decodedValues, nil
}
