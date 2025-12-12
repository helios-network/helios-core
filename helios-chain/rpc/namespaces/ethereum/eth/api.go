package eth

import (
	"context"

	"github.com/ethereum/go-ethereum/signer/core/apitypes"

	"github.com/ethereum/go-ethereum/rpc"

	"cosmossdk.io/log"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"helios-core/helios-chain/rpc/backend"

	rpctypes "helios-core/helios-chain/rpc/types"
	"helios-core/helios-chain/types"
	chronostypes "helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// The Ethereum API allows applications to connect to an Evmos node that is
// part of the Evmos blockchain. Developers can interact with on-chain EVM data
// and send different types of transactions to the network by utilizing the
// endpoints provided by the API. The API follows a JSON-RPC standard. If not
// otherwise specified, the interface is derived from the Alchemy Ethereum API:
// https://docs.alchemy.com/alchemy/apis/ethereum
type EthereumAPI interface {
	// Getting Blocks
	//
	// Retrieves information from a particular block in the blockchain.
	BlockNumber() (hexutil.Uint64, error)
	GetBlockByNumber(ethBlockNum rpctypes.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint
	GetBlockTransactionCountByNumber(blockNum rpctypes.BlockNumber) *hexutil.Uint

	GetBlocksByPageAndSize(page hexutil.Uint64, size hexutil.Uint64, fullTx bool) ([]map[string]interface{}, error)

	// GetProposalBy(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error)
	GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalRPC, error)
	GetProposalsByPageAndSizeWithFilter(page hexutil.Uint64, size hexutil.Uint64, filter string) ([]*rpctypes.ProposalRPC, error)
	GetProposal(id hexutil.Uint64) (*rpctypes.ProposalRPC, error)
	GetProposalVotesByPageAndSize(id uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalVoteRPC, error)
	GetProposalsCount() (*hexutil.Uint64, error)
	GovCatalog() ([]rpctypes.MsgCatalogEntry, error)

	// Reading Transactions
	//
	// Retrieves information on the state data for addresses regardless of whether
	// it is a user or a smart contract.
	GetTransactionByHash(hash common.Hash) (*rpctypes.RPCTransaction, error)
	GetTransactionCount(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Uint64, error)
	GetTotalTransactionCount() (*hexutil.Uint64, error)
	GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error)
	GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error)
	GetLastTransactionsInfo(size hexutil.Uint64) ([]*rpctypes.ParsedRPCTransaction, error)
	GetAllTransactionReceiptsByBlockNumber(blockNum rpctypes.BlockNumber) ([]map[string]interface{}, error)
	// eth_getBlockReceipts

	// Writing Transactions
	//
	// Allows developers to both send ETH from one address to another, write data
	// on-chain, and interact with smart contracts.
	SendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	SendTransaction(args evmtypes.TransactionArgs) (common.Hash, error)
	// eth_sendPrivateTransaction
	// eth_cancel	PrivateTransaction

	// Account Information
	//
	// Returns information regarding an address's stored on-chain data.
	Accounts() ([]common.Address, error)
	GetAccountType(address common.Address) (string, error)
	GetHeliosAddress(address common.Address) (string, error)
	GetHeliosValoperAddress(address common.Address) (string, error)
	GetBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error)
	GetStorageAt(address common.Address, key string, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error)
	GetCode(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error)
	GetProof(address common.Address, storageKeys []string, blockNrOrHash rpctypes.BlockNumberOrHash) (*rpctypes.AccountResult, error)
	GetAccountTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error)
	GetAccountTokenBalance(address common.Address, tokenAddress common.Address) (*hexutil.Big, error)
	GetAccountTokensBalanceByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) (*rpctypes.AccountTokensBalance, error)
	GetAccountLastTransactionsInfo(address common.Address) ([]*rpctypes.ParsedRPCTransaction, error)

	// EVM/Smart Contract Execution
	//
	// Allows developers to read data from the blockchain which includes executing
	// smart contracts. However, no data is published to the Ethereum network.
	Call(args evmtypes.TransactionArgs, blockNrOrHash rpctypes.BlockNumberOrHash, _ *rpctypes.StateOverride) (hexutil.Bytes, error)

	// Chain Information
	//
	// Returns information on the Ethereum network and internal settings.
	ProtocolVersion() hexutil.Uint
	GasPrice() (*hexutil.Big, error)
	EstimateGas(args evmtypes.TransactionArgs, blockNrOptional *rpctypes.BlockNumber) (hexutil.Uint64, error)
	FeeHistory(blockCount rpc.DecimalOrHex, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*rpctypes.FeeHistoryResult, error)
	MaxPriorityFeePerGas() (*hexutil.Big, error)
	ChainId() (*hexutil.Big, error)
	ChainSize() (*rpctypes.ChainSize, error)

	// Tokens Information
	GetTokensByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error)
	GetTokenDetails(tokenAddress common.Address) (*banktypes.FullMetadata, error)
	GetTokensDetails(tokenAddresses []common.Address) ([]*banktypes.FullMetadata, error)
	GetTokensByChainIdAndPageAndSize(chainId uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error)

	// Getting Uncles
	//
	// Returns information on uncle blocks are which are network rejected blocks and replaced by a canonical block instead.
	GetUncleByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) map[string]interface{}
	GetUncleByBlockNumberAndIndex(number, idx hexutil.Uint) map[string]interface{}
	GetUncleCountByBlockHash(hash common.Hash) hexutil.Uint
	GetUncleCountByBlockNumber(blockNum rpctypes.BlockNumber) hexutil.Uint

	// Proof of Work
	Hashrate() hexutil.Uint64
	Mining() bool

	// Other
	Syncing() (interface{}, error)
	Coinbase() (string, error)
	Sign(address common.Address, data hexutil.Bytes) (hexutil.Bytes, error)
	GetTransactionLogs(txHash common.Hash) ([]*ethtypes.Log, error)
	SignTypedData(address common.Address, typedData apitypes.TypedData) (hexutil.Bytes, error)
	FillTransaction(args evmtypes.TransactionArgs) (*rpctypes.SignTransactionResult, error)
	Resend(ctx context.Context, args evmtypes.TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error)
	GetPendingTransactions() ([]*rpctypes.RPCTransaction, error)
	// eth_signTransaction (on Ethereum.org)
	// eth_getCompilers (on Ethereum.org)
	// eth_compileSolidity (on Ethereum.org)
	// eth_compileLLL (on Ethereum.org)
	// eth_compileSerpent (on Ethereum.org)
	// eth_getWork (on Ethereum.org)
	// eth_submitWork (on Ethereum.org)
	// eth_submitHashrate (on Ethereum.org)

	// Staking
	GetDelegations(address common.Address) ([]rpctypes.DelegationRPC, error)
	GetDelegation(address common.Address, validatorAddress common.Address) (*rpctypes.DelegationRPC, error)
	GetDelegationForValidators(address common.Address, validatorAddresses []string) ([]*rpctypes.DelegationRPC, error)
	GetValidator(address common.Address) (*rpctypes.ValidatorRPC, error)
	GetValidatorAndHisDelegation(address common.Address) (*rpctypes.ValidatorWithDelegationRPC, error)
	GetValidatorCommission(address common.Address) (*rpctypes.ValidatorCommissionRPC, error)
	GetValidatorOutStandingRewards(address common.Address) (*rpctypes.ValidatorRewardRPC, error)
	GetValidatorWithHisDelegationAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndDelegationRPC, error)
	GetValidatorWithHisAssetsAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndAssetsRPC, error)
	GetValidatorAndHisCommission(address common.Address) (*rpctypes.ValidatorWithCommissionRPC, error)
	GetValidatorsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorRPC, error)
	GetValidatorAPYDetails(address common.Address) (*rpctypes.ValidatorAPYDetailsRPC, error)
	GetValidatorsAPYByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorAPYDetailsRPC, error)
	GetActiveValidatorCount() (int, error)
	GetValidatorCount() (int, error)
	GetAllWhitelistedAssets() ([]rpctypes.WhitelistedAssetRPC, error)
	GetBlockSignatures(blockHeight hexutil.Uint64) ([]*rpctypes.ValidatorSignature, error)
	GetEpochComplete(epochId hexutil.Uint64) (*rpctypes.EpochCompleteResponse, error)
	GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorWithAssetsAndCommissionAndDelegationRPC, error)
	GetCoinInfo() (*rpctypes.CoinInfoRPC, error)
	// eth_getDelegations

	// cron
	GetCron(id uint64) (*chronostypes.Cron, error)
	GetCronByAddress(address common.Address) (*chronostypes.Cron, error)
	GetCronsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]chronostypes.Cron, error)
	GetAccountCronsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]chronostypes.Cron, error)
	GetCronTransactionByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionRPC, error)
	GetCronTransactionByHash(hash string) (*chronostypes.CronTransactionRPC, error)
	GetCronTransactionReceiptByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionReceiptRPC, error)
	GetCronTransactionReceiptByHash(hash string) (*chronostypes.CronTransactionReceiptRPC, error)
	GetCronTransactionReceiptsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error)
	GetCronTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error)
	GetAllCronTransactionReceiptsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error)
	GetAllCronTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error)
	GetAllCronTransactionReceiptsByBlockNumber(blockNum rpctypes.BlockNumber) ([]*chronostypes.CronTransactionReceiptRPC, error)
	GetBlockCronLogs(blockNum rpctypes.BlockNumber) ([]*ethtypes.Log, error)
	GetCronStatistics() (*chronostypes.CronStatistics, error)

	// hyperion
	GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error)
	GetAllHyperionTransferTxs(size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error)
	GetHyperionChains() ([]*rpctypes.HyperionChainRPC, error)
	GetHyperionHistoricalFees(hyperionId uint64) (*hyperiontypes.QueryHistoricalFeesResponse, error)
	GetValidatorHyperionData(address common.Address) (*hyperiontypes.OrchestratorData, error)
	GetWhitelistedAddresses(hyperionId uint64) ([]string, error)
	GetHyperionProjectedCurrentNetworkHeight(hyperionId uint64) (uint64, error)
	GetHyperionNonceAlreadyObserved(hyperionId uint64, nonce uint64) (bool, error)
	GetHyperionSkippedNonces(hyperionId uint64) ([]*hyperiontypes.SkippedNonceFullInfo, error)
	GetAllHyperionSkippedNonces() ([]*hyperiontypes.SkippedNonceFullInfoWithHyperionId, error)

	GetCosmosTransactionByHashFormatted(txHash string) (*map[string]interface{}, error)
}

var _ EthereumAPI = (*PublicAPI)(nil)

// PublicAPI is the eth_ prefixed set of APIs in the Web3 JSON-RPC spec.
type PublicAPI struct {
	ctx     context.Context
	logger  log.Logger
	backend backend.EVMBackend
}

// NewPublicAPI creates an instance of the public ETH Web3 API.
func NewPublicAPI(logger log.Logger, backend backend.EVMBackend) *PublicAPI {
	api := &PublicAPI{
		ctx:     context.Background(),
		logger:  logger.With("client", "json-rpc"),
		backend: backend,
	}

	return api
}

///////////////////////////////////////////////////////////////////////////////
///                           Blocks						                            ///
///////////////////////////////////////////////////////////////////////////////

// BlockNumber returns the current block number.
func (e *PublicAPI) BlockNumber() (hexutil.Uint64, error) {
	e.logger.Debug("eth_blockNumber")
	return e.backend.BlockNumber()
}

// GetBlockByNumber returns the block identified by number.
func (e *PublicAPI) GetBlockByNumber(ethBlockNum rpctypes.BlockNumber, fullTx bool) (map[string]interface{}, error) {
	e.logger.Debug("eth_getBlockByNumber", "number", ethBlockNum, "full", fullTx)
	return e.backend.GetBlockByNumber(ethBlockNum, fullTx)
}

// GetBlockByHash returns the block identified by hash.
func (e *PublicAPI) GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error) {
	e.logger.Debug("eth_getBlockByHash", "hash", hash.Hex(), "full", fullTx)
	return e.backend.GetBlockByHash(hash, fullTx)
}

func (e *PublicAPI) GetBlocksByPageAndSize(page hexutil.Uint64, size hexutil.Uint64, fullTx bool) ([]map[string]interface{}, error) {
	e.logger.Debug("eth_getLatestBlocks", "page", page, "size", size, "full", fullTx)
	return e.backend.GetBlocksByPageAndSize(page, size, fullTx)
}

// Proposals

func (e *PublicAPI) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalRPC, error) {
	e.logger.Debug("eth_getProposalsByPageAndSize", "page", page, "size", size, "full")
	return e.backend.GetProposalsByPageAndSize(page, size)
}

func (e *PublicAPI) GetProposalsByPageAndSizeWithFilter(page hexutil.Uint64, size hexutil.Uint64, filter string) ([]*rpctypes.ProposalRPC, error) {
	e.logger.Debug("eth_getProposalsByPageAndSizeWithFilter", "page", page, "size", size, "filter", filter)
	return e.backend.GetProposalsByPageAndSizeWithFilter(page, size, filter)
}

func (e *PublicAPI) GetProposal(id hexutil.Uint64) (*rpctypes.ProposalRPC, error) {
	e.logger.Debug("eth_getProposal", "id", id)
	return e.backend.GetProposal(id)
}

func (e *PublicAPI) GetProposalVotesByPageAndSize(id uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalVoteRPC, error) {
	e.logger.Debug("eth_getProposalVotesByPageAndSize", "id", id, "page", page, "size", size)
	return e.backend.GetProposalVotesByPageAndSize(id, page, size)
}

func (e *PublicAPI) GetProposalsCount() (*hexutil.Uint64, error) {
	e.logger.Debug("eth_getProposalsCount")
	return e.backend.GetProposalsCount()
}

func (e *PublicAPI) GovCatalog() ([]rpctypes.MsgCatalogEntry, error) {
	e.logger.Debug("eth_govCatalog")
	return e.backend.GovCatalog()
}

///////////////////////////////////////////////////////////////////////////////
///                           Read Txs					                            ///
///////////////////////////////////////////////////////////////////////////////

// GetTransactionByHash returns the transaction identified by hash.
func (e *PublicAPI) GetTransactionByHash(hash common.Hash) (*rpctypes.RPCTransaction, error) {
	e.logger.Debug("eth_getTransactionByHash", "hash", hash.Hex())
	return e.backend.GetTransactionByHash(hash)
}

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (e *PublicAPI) GetTransactionCount(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Uint64, error) {
	e.logger.Debug("eth_getTransactionCount", "address", address.Hex(), "block number or hash", blockNrOrHash)
	blockNum, err := e.backend.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}
	return e.backend.GetTransactionCount(address, blockNum)
}

// GetTotalTransactionCount returns the total number of transactions in the blockchain.
func (e *PublicAPI) GetTotalTransactionCount() (*hexutil.Uint64, error) {
	e.logger.Debug("eth_getTotalTransactionCount")
	return e.backend.GetTotalTransactionCount()
}

// GetTransactionReceipt returns the transaction receipt identified by hash.
func (e *PublicAPI) GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error) {
	hexTx := hash.Hex()
	e.logger.Debug("eth_getTransactionReceipt", "hash", hexTx)
	return e.backend.GetTransactionReceipt(hash)
}

// GetBlockTransactionCountByHash returns the number of transactions in the block identified by hash.
func (e *PublicAPI) GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint {
	e.logger.Debug("eth_getBlockTransactionCountByHash", "hash", hash.Hex())
	return e.backend.GetBlockTransactionCountByHash(hash)
}

// GetBlockTransactionCountByNumber returns the number of transactions in the block identified by number.
func (e *PublicAPI) GetBlockTransactionCountByNumber(blockNum rpctypes.BlockNumber) *hexutil.Uint {
	e.logger.Debug("eth_getBlockTransactionCountByNumber", "height", blockNum.Int64())
	return e.backend.GetBlockTransactionCountByNumber(blockNum)
}

// GetTransactionByBlockHashAndIndex returns the transaction identified by hash and index.
func (e *PublicAPI) GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	e.logger.Debug("eth_getTransactionByBlockHashAndIndex", "hash", hash.Hex(), "index", idx)
	return e.backend.GetTransactionByBlockHashAndIndex(hash, idx)
}

// GetTransactionByBlockNumberAndIndex returns the transaction identified by number and index.
func (e *PublicAPI) GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error) {
	e.logger.Debug("eth_getTransactionByBlockNumberAndIndex", "number", blockNum, "index", idx)
	return e.backend.GetTransactionByBlockNumberAndIndex(blockNum, idx)
}

func (e *PublicAPI) GetTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error) {
	e.logger.Info("eth_getTransactionsByPageAndSize", "page", page, "size", size)
	return e.backend.GetTransactionsByPageAndSize(page, size)
}

func (e *PublicAPI) GetLastTransactionsInfo(size hexutil.Uint64) ([]*rpctypes.ParsedRPCTransaction, error) {
	e.logger.Info("eth_getLastTransactionsInfo", "size", size)
	return e.backend.GetLastTransactionsInfo(size)
}

///////////////////////////////////////////////////////////////////////////////
///                           Write Txs					                            ///
///////////////////////////////////////////////////////////////////////////////

// SendRawTransaction send a raw Ethereum transaction.
func (e *PublicAPI) SendRawTransaction(data hexutil.Bytes) (common.Hash, error) {
	e.logger.Debug("eth_sendRawTransaction", "length", len(data))
	return e.backend.SendRawTransaction(data)
}

// SendTransaction sends an Ethereum transaction.
func (e *PublicAPI) SendTransaction(args evmtypes.TransactionArgs) (common.Hash, error) {
	e.logger.Debug("eth_sendTransaction", "args", args.String())
	return e.backend.SendTransaction(args)
}

///////////////////////////////////////////////////////////////////////////////
///                           Account Information				                    ///
///////////////////////////////////////////////////////////////////////////////

// Accounts returns the list of accounts available to this node.
func (e *PublicAPI) Accounts() ([]common.Address, error) {
	e.logger.Debug("eth_accounts")
	return e.backend.Accounts()
}

func (e *PublicAPI) GetAccountType(address common.Address) (string, error) {
	e.logger.Debug("eth_getAccountType", "address", address.String())
	return e.backend.GetAccountType(address)
}

func (e *PublicAPI) GetHeliosAddress(address common.Address) (string, error) {
	e.logger.Debug("eth_getHeliosAddress", "address", address.String())
	return e.backend.GetHeliosAddress(address)
}

func (e *PublicAPI) GetHeliosValoperAddress(address common.Address) (string, error) {
	e.logger.Debug("eth_getHeliosValoperAddress", "address", address.String())
	return e.backend.GetHeliosValoperAddress(address)
}

// GetBalance returns the provided account's balance up to the provided block number.
func (e *PublicAPI) GetBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error) {
	e.logger.Debug("eth_getBalance", "address", address.String(), "block number or hash", blockNrOrHash)
	return e.backend.GetBalance(address, blockNrOrHash)
}

// GetStorageAt returns the contract storage at the given address, block number, and key.
func (e *PublicAPI) GetStorageAt(address common.Address, key string, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error) {
	e.logger.Debug("eth_getStorageAt", "address", address.Hex(), "key", key, "block number or hash", blockNrOrHash)
	return e.backend.GetStorageAt(address, key, blockNrOrHash)
}

// GetCode returns the contract code at the given address and block number.
func (e *PublicAPI) GetCode(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error) {
	e.logger.Debug("eth_getCode", "address", address.Hex(), "block number or hash", blockNrOrHash)
	return e.backend.GetCode(address, blockNrOrHash)
}

// GetProof returns an account object with proof and any storage proofs
func (e *PublicAPI) GetProof(address common.Address,
	storageKeys []string,
	blockNrOrHash rpctypes.BlockNumberOrHash,
) (*rpctypes.AccountResult, error) {
	e.logger.Debug("eth_getProof", "address", address.Hex(), "keys", storageKeys, "block number or hash", blockNrOrHash)
	return e.backend.GetProof(address, storageKeys, blockNrOrHash)
}

func (e *PublicAPI) GetAccountTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error) {
	e.logger.Debug("eth_getProof", "address", address.Hex(), "page", page, "size", size)
	return e.backend.GetAccountTransactionsByPageAndSize(address, page, size)
}

func (e *PublicAPI) GetAccountTokenBalance(address common.Address, tokenAddress common.Address) (*hexutil.Big, error) {
	e.logger.Debug("eth_getAccountTokenBalance", "address", address.String(), "tokenAddress", tokenAddress.String())
	return e.backend.GetAccountTokenBalance(address, tokenAddress)
}

func (e *PublicAPI) GetAccountTokensBalanceByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) (*rpctypes.AccountTokensBalance, error) {
	e.logger.Debug("eth_getAccountTokensBalanceByPageAndSize", "address", address.String(), "page", page, "size", size)
	return e.backend.GetAccountTokensBalanceByPageAndSize(address, page, size)
}

func (e *PublicAPI) GetAccountLastTransactionsInfo(address common.Address) ([]*rpctypes.ParsedRPCTransaction, error) {
	e.logger.Debug("eth_getAccountLastTransactionsInfo", "address", address.String())
	return e.backend.GetAccountLastTransactionsInfo(address)
}

///////////////////////////////////////////////////////////////////////////////
///                           EVM/Smart Contract Execution				          ///
///////////////////////////////////////////////////////////////////////////////

// Call performs a raw contract call.
func (e *PublicAPI) Call(args evmtypes.TransactionArgs,
	blockNrOrHash rpctypes.BlockNumberOrHash,
	_ *rpctypes.StateOverride,
) (hexutil.Bytes, error) {
	e.logger.Debug("eth_call", "args", args.String(), "block number or hash", blockNrOrHash)

	blockNum, err := e.backend.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}
	data, err := e.backend.DoCall(args, blockNum)
	if err != nil {
		return []byte{}, err
	}

	return (hexutil.Bytes)(data.Ret), nil
}

///////////////////////////////////////////////////////////////////////////////
///                           Event Logs													          ///
///////////////////////////////////////////////////////////////////////////////
// FILTER API at ./filters/api.go

///////////////////////////////////////////////////////////////////////////////
///                           Chain Information										          ///
///////////////////////////////////////////////////////////////////////////////

// ProtocolVersion returns the supported Ethereum protocol version.
func (e *PublicAPI) ProtocolVersion() hexutil.Uint {
	e.logger.Debug("eth_protocolVersion")
	return hexutil.Uint(types.ProtocolVersion)
}

// GasPrice returns the current gas price based on Ethermint's gas price oracle.
func (e *PublicAPI) GasPrice() (*hexutil.Big, error) {
	e.logger.Debug("eth_gasPrice")
	return e.backend.GasPrice()
}

// EstimateGas returns an estimate of gas usage for the given smart contract call.
func (e *PublicAPI) EstimateGas(args evmtypes.TransactionArgs, blockNrOptional *rpctypes.BlockNumber) (hexutil.Uint64, error) {
	e.logger.Debug("eth_estimateGas")
	return e.backend.EstimateGas(args, blockNrOptional)
}

func (e *PublicAPI) FeeHistory(blockCount rpc.DecimalOrHex,
	lastBlock rpc.BlockNumber,
	rewardPercentiles []float64,
) (*rpctypes.FeeHistoryResult, error) {
	e.logger.Debug("eth_feeHistory")
	return e.backend.FeeHistory(blockCount, lastBlock, rewardPercentiles)
}

// MaxPriorityFeePerGas returns a suggestion for a gas tip cap for dynamic fee transactions.
func (e *PublicAPI) MaxPriorityFeePerGas() (*hexutil.Big, error) {
	e.logger.Debug("eth_maxPriorityFeePerGas")
	head, err := e.backend.CurrentHeader()
	if err != nil {
		return nil, err
	}
	tipcap, err := e.backend.SuggestGasTipCap(head.BaseFee)
	if err != nil {
		return nil, err
	}
	return (*hexutil.Big)(tipcap), nil
}

// ChainId is the EIP-155 replay-protection chain id for the current ethereum chain config.
func (e *PublicAPI) ChainId() (*hexutil.Big, error) { //nolint
	e.logger.Debug("eth_chainId")
	return e.backend.ChainID()
}

func (e *PublicAPI) ChainSize() (*rpctypes.ChainSize, error) {
	e.logger.Debug("eth_chainSize")
	return e.backend.ChainSize()
}

///////////////////////////////////////////////////////////////////////////////
///                           Tokens									    ///
///////////////////////////////////////////////////////////////////////////////

func (e *PublicAPI) GetTokensByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error) {
	e.logger.Debug("eth_getTokensByPageAndSize", "page", page, "size", size)
	return e.backend.GetTokensByPageAndSize(page, size)
}

func (e *PublicAPI) GetTokenDetails(tokenAddress common.Address) (*banktypes.FullMetadata, error) {
	e.logger.Debug("eth_getTokenDetails", "tokenAddress", tokenAddress.String())
	return e.backend.GetTokenDetails(tokenAddress)
}

func (e *PublicAPI) GetTokensDetails(tokenAddresses []common.Address) ([]*banktypes.FullMetadata, error) {
	e.logger.Debug("eth_getTokensDetails", "tokenAddresses", tokenAddresses)
	return e.backend.GetTokensDetails(tokenAddresses)
}

func (e *PublicAPI) GetTokensByChainIdAndPageAndSize(chainId uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error) {
	e.logger.Debug("eth_getTokensByChainIdAndPageAndSize", "chainId", chainId, "page", page, "size", size)
	return e.backend.GetTokensByChainIdAndPageAndSize(chainId, page, size)
}

///////////////////////////////////////////////////////////////////////////////
///                           Uncles										///
///////////////////////////////////////////////////////////////////////////////

// GetUncleByBlockHashAndIndex returns the uncle identified by hash and index. Always returns nil.
func (e *PublicAPI) GetUncleByBlockHashAndIndex(_ common.Hash, _ hexutil.Uint) map[string]interface{} {
	return nil
}

// GetUncleByBlockNumberAndIndex returns the uncle identified by number and index. Always returns nil.
func (e *PublicAPI) GetUncleByBlockNumberAndIndex(_, _ hexutil.Uint) map[string]interface{} {
	return nil
}

// GetUncleCountByBlockHash returns the number of uncles in the block identified by hash. Always zero.
func (e *PublicAPI) GetUncleCountByBlockHash(_ common.Hash) hexutil.Uint {
	return 0
}

// GetUncleCountByBlockNumber returns the number of uncles in the block identified by number. Always zero.
func (e *PublicAPI) GetUncleCountByBlockNumber(_ rpctypes.BlockNumber) hexutil.Uint {
	return 0
}

///////////////////////////////////////////////////////////////////////////////
///                           Proof of Work												          ///
///////////////////////////////////////////////////////////////////////////////

// Hashrate returns the current node's hashrate. Always 0.
func (e *PublicAPI) Hashrate() hexutil.Uint64 {
	e.logger.Debug("eth_hashrate")
	return 0
}

// Mining returns whether or not this node is currently mining. Always false.
func (e *PublicAPI) Mining() bool {
	e.logger.Debug("eth_mining")
	return false
}

///////////////////////////////////////////////////////////////////////////////
///                           Other 															          ///
///////////////////////////////////////////////////////////////////////////////

// Syncing returns false in case the node is currently not syncing with the network. It can be up to date or has not
// yet received the latest block headers from its pears. In case it is synchronizing:
// - startingBlock: block number this node started to synchronize from
// - currentBlock:  block number this node is currently importing
// - highestBlock:  block number of the highest block header this node has received from peers
// - pulledStates:  number of state entries processed until now
// - knownStates:   number of known state entries that still need to be pulled
func (e *PublicAPI) Syncing() (interface{}, error) {
	e.logger.Debug("eth_syncing")
	return e.backend.Syncing()
}

// Coinbase is the address that staking rewards will be send to (alias for Etherbase).
func (e *PublicAPI) Coinbase() (string, error) {
	e.logger.Debug("eth_coinbase")

	coinbase, err := e.backend.GetCoinbase()
	if err != nil {
		return "", err
	}
	ethAddr := common.BytesToAddress(coinbase.Bytes())
	return ethAddr.Hex(), nil
}

// Sign signs the provided data using the private key of address via Geth's signature standard.
func (e *PublicAPI) Sign(address common.Address, data hexutil.Bytes) (hexutil.Bytes, error) {
	e.logger.Debug("eth_sign", "address", address.Hex(), "data", common.Bytes2Hex(data))
	return e.backend.Sign(address, data)
}

// GetTransactionLogs returns the logs given a transaction hash.
func (e *PublicAPI) GetTransactionLogs(txHash common.Hash) ([]*ethtypes.Log, error) {
	e.logger.Debug("eth_getTransactionLogs", "hash", txHash)

	return e.backend.GetTransactionLogs(txHash)
}

// SignTypedData signs EIP-712 conformant typed data
func (e *PublicAPI) SignTypedData(address common.Address, typedData apitypes.TypedData) (hexutil.Bytes, error) {
	e.logger.Debug("eth_signTypedData", "address", address.Hex(), "data", typedData)
	return e.backend.SignTypedData(address, typedData)
}

// FillTransaction fills the defaults (nonce, gas, gasPrice or 1559 fields)
// on a given unsigned transaction, and returns it to the caller for further
// processing (signing + broadcast).
func (e *PublicAPI) FillTransaction(args evmtypes.TransactionArgs) (*rpctypes.SignTransactionResult, error) {
	// Set some sanity defaults and terminate on failure
	args, err := e.backend.SetTxDefaults(args)
	if err != nil {
		return nil, err
	}

	// Assemble the transaction and obtain rlp
	tx := args.ToTransaction().AsTransaction()

	data, err := tx.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return &rpctypes.SignTransactionResult{
		Raw: data,
		Tx:  tx,
	}, nil
}

// Resend accepts an existing transaction and a new gas price and limit. It will remove
// the given transaction from the pool and reinsert it with the new gas price and limit.
func (e *PublicAPI) Resend(_ context.Context,
	args evmtypes.TransactionArgs,
	gasPrice *hexutil.Big,
	gasLimit *hexutil.Uint64,
) (common.Hash, error) {
	e.logger.Debug("eth_resend", "args", args.String())
	return e.backend.Resend(args, gasPrice, gasLimit)
}

// GetPendingTransactions returns the transactions that are in the transaction pool
// and have a from address that is one of the accounts this node manages.
func (e *PublicAPI) GetPendingTransactions() ([]*rpctypes.RPCTransaction, error) {
	e.logger.Debug("eth_getPendingTransactions")

	txs, err := e.backend.PendingTransactions()
	if err != nil {
		return nil, err
	}

	chainIDHex, err := e.backend.ChainID()
	if err != nil {
		return nil, err
	}

	chainID := chainIDHex.ToInt()

	result := make([]*rpctypes.RPCTransaction, 0, len(txs))
	for _, tx := range txs {
		for _, msg := range (*tx).GetMsgs() {
			ethMsg, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				// not valid ethereum tx
				break
			}

			rpctx, err := rpctypes.NewTransactionFromMsg(
				ethMsg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				chainID,
			)
			if err != nil {
				return nil, err
			}

			result = append(result, rpctx)
		}
	}

	return result, nil
}

func (e *PublicAPI) GetDelegations(address common.Address) ([]rpctypes.DelegationRPC, error) {
	e.logger.Debug("eth_getDelegations", "address", address.Hex())
	return e.backend.GetDelegations(address)
}

func (e *PublicAPI) GetValidator(address common.Address) (*rpctypes.ValidatorRPC, error) {
	e.logger.Debug("eth_getValidator", "address", address)
	return e.backend.GetValidator(address)
}

func (e *PublicAPI) GetValidatorAndHisDelegation(address common.Address) (*rpctypes.ValidatorWithDelegationRPC, error) {
	e.logger.Debug("eth_getValidatorAndHisDelegation", "address", address)
	return e.backend.GetValidatorAndHisDelegation(address)
}

func (e *PublicAPI) GetValidatorCommission(address common.Address) (*rpctypes.ValidatorCommissionRPC, error) {
	e.logger.Debug("eth_getValidatorCommission", "address", address)
	return e.backend.GetValidatorCommission(address)
}

func (e *PublicAPI) GetValidatorOutStandingRewards(address common.Address) (*rpctypes.ValidatorRewardRPC, error) {
	e.logger.Debug("eth_getValidatorOutStandingRewards", "address", address)
	return e.backend.GetValidatorOutStandingRewards(address)
}

func (e *PublicAPI) GetValidatorWithHisDelegationAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndDelegationRPC, error) {
	e.logger.Debug("eth_getValidatorWithHisDelegationAndCommission", "address", address)
	return e.backend.GetValidatorWithHisDelegationAndCommission(address)
}

func (e *PublicAPI) GetValidatorWithHisAssetsAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndAssetsRPC, error) {
	e.logger.Debug("eth_getValidatorWithHisAssetsAndCommission", "address", address)
	return e.backend.GetValidatorWithHisAssetsAndCommission(address)
}

func (e *PublicAPI) GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorWithAssetsAndCommissionAndDelegationRPC, error) {
	e.logger.Debug("eth_getValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation", "page", page, "size", size)
	return e.backend.GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation(page, size)
}

func (e *PublicAPI) GetValidatorAndHisCommission(address common.Address) (*rpctypes.ValidatorWithCommissionRPC, error) {
	e.logger.Debug("eth_getValidatorAndHisCommission", "address", address)
	return e.backend.GetValidatorAndHisCommission(address)
}

func (e *PublicAPI) GetValidatorsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorRPC, error) {
	e.logger.Debug("eth_getValidatorsByPageAndSize", "page", page, "size", size)
	return e.backend.GetValidatorsByPageAndSize(page, size)
}

func (e *PublicAPI) GetActiveValidatorCount() (int, error) {
	e.logger.Debug("eth_GetActiveValidatorCount")
	return e.backend.GetActiveValidatorCount()
}

func (e *PublicAPI) GetValidatorCount() (int, error) {
	e.logger.Debug("eth_getValidatorCount")
	return e.backend.GetValidatorCount()
}

func (e *PublicAPI) GetCoinInfo() (*rpctypes.CoinInfoRPC, error) {
	return e.backend.GetCoinInfo()
}

func (e *PublicAPI) GetDelegation(address common.Address, validatorAddress common.Address) (*rpctypes.DelegationRPC, error) {
	e.logger.Debug("eth_getDelegation", "address", address.Hex(), "validatorAddress", validatorAddress)
	return e.backend.GetDelegation(address, validatorAddress)
}

func (e *PublicAPI) GetDelegationForValidators(address common.Address, validatorAddresses []string) ([]*rpctypes.DelegationRPC, error) {
	e.logger.Debug("eth_getDelegationForValidators", "address", address.Hex(), "validatorAddresses", validatorAddresses)
	return e.backend.GetDelegationForValidators(address, validatorAddresses)
}

func (e *PublicAPI) GetAllWhitelistedAssets() ([]rpctypes.WhitelistedAssetRPC, error) {
	e.logger.Debug("eth_getAllWhitelistedAssets")
	return e.backend.GetAllWhitelistedAssets()
}

func (e *PublicAPI) GetValidatorAPYDetails(address common.Address) (*rpctypes.ValidatorAPYDetailsRPC, error) {
	e.logger.Debug("eth_getValidatorAPYDetails", "address", address.Hex())
	return e.backend.GetValidatorAPYDetails(address)
}

func (e *PublicAPI) GetValidatorsAPYByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorAPYDetailsRPC, error) {
	e.logger.Debug("eth_getValidatorsAPYByPageAndSize", "page", page, "size", size)
	return e.backend.GetValidatorsAPYByPageAndSize(page, size)
}

func (e *PublicAPI) GetCron(id uint64) (*chronostypes.Cron, error) {
	e.logger.Debug("eth_getCron", "id", id)
	return e.backend.GetCron(id)
}

func (e *PublicAPI) GetCronByAddress(address common.Address) (*chronostypes.Cron, error) {
	e.logger.Debug("eth_getCronByAddress", "address", address.String())
	return e.backend.GetCronByAddress(address)
}

func (e *PublicAPI) GetCronsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]chronostypes.Cron, error) {
	e.logger.Debug("eth_getCronsByPageAndSize", "page", page, "size", size)
	return e.backend.GetCronsByPageAndSize(page, size)
}

func (e *PublicAPI) GetAccountCronsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]chronostypes.Cron, error) {
	e.logger.Debug("eth_getAccountCronsByPageAndSize", "address", address, "page", page, "size", size)
	return e.backend.GetAccountCronsByPageAndSize(address, page, size)
}

func (e *PublicAPI) GetCronTransactionByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionRPC, error) {
	e.logger.Debug("eth_getCronTransactionByNonce", "nonce", nonce)
	return e.backend.GetCronTransactionByNonce(nonce)
}

func (e *PublicAPI) GetCronTransactionByHash(hash string) (*chronostypes.CronTransactionRPC, error) {
	e.logger.Debug("eth_getCronTransactionByHash", "hash", hash)
	return e.backend.GetCronTransactionByHash(hash)
}

func (e *PublicAPI) GetCronTransactionReceiptByNonce(nonce hexutil.Uint64) (*chronostypes.CronTransactionReceiptRPC, error) {
	e.logger.Debug("eth_getCronTransactionReceiptByNonce", "nonce", nonce)
	return e.backend.GetCronTransactionReceiptByNonce(nonce)
}

func (e *PublicAPI) GetCronTransactionReceiptByHash(hash string) (*chronostypes.CronTransactionReceiptRPC, error) {
	e.logger.Debug("eth_getCronTransactionReceiptByHash", "hash", hash)
	return e.backend.GetCronTransactionReceiptByHash(hash)
}

func (e *PublicAPI) GetCronTransactionReceiptsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error) {
	e.logger.Debug("eth_getCronTransactionReceiptsByPageAndSize", "address", address, "page", page, "size", size)
	return e.backend.GetCronTransactionReceiptsByPageAndSize(address, page, size)
}

func (e *PublicAPI) GetCronTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error) {
	e.logger.Debug("eth_getCronTransactionsByPageAndSize", "address", address, "page", page, "size", size)
	return e.backend.GetCronTransactionsByPageAndSize(address, page, size)
}

func (e *PublicAPI) GetAllCronTransactionReceiptsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionReceiptRPC, error) {
	e.logger.Debug("eth_getAllCronTransactionReceiptsByPageAndSize", "page", page, "size", size)
	return e.backend.GetAllCronTransactionReceiptsByPageAndSize(page, size)
}

func (e *PublicAPI) GetAllCronTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*chronostypes.CronTransactionRPC, error) {
	e.logger.Debug("eth_getAllCronTransactionsByPageAndSize", "page", page, "size", size)
	return e.backend.GetAllCronTransactionsByPageAndSize(page, size)
}

func (e *PublicAPI) GetAllCronTransactionReceiptsByBlockNumber(blockNum rpctypes.BlockNumber) ([]*chronostypes.CronTransactionReceiptRPC, error) {
	e.logger.Debug("eth_getAllCronTransactionReceiptsByBlockNumber", "height", blockNum.Int64())
	return e.backend.GetAllCronTransactionReceiptsByBlockNumber(blockNum)
}

func (e *PublicAPI) GetBlockCronLogs(blockNum rpctypes.BlockNumber) ([]*ethtypes.Log, error) {
	e.logger.Debug("eth_getBlockCronLogs", "height", blockNum.Int64())
	return e.backend.GetBlockCronLogs(blockNum)
}

func (e *PublicAPI) GetCronStatistics() (*chronostypes.CronStatistics, error) {
	e.logger.Debug("eth_getCronStatistics")
	return e.backend.GetCronStatistics()
}

func (e *PublicAPI) GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error) {
	e.logger.Debug("eth_getHyperionAccountTransferTxsByPageAndSize", "address", address, "page", page, "size", size)
	return e.backend.GetHyperionAccountTransferTxsByPageAndSize(address, page, size)
}

func (e *PublicAPI) GetAllHyperionTransferTxs(size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error) {
	e.logger.Debug("eth_getAllHyperionTransferTxs", "size", size)
	return e.backend.GetAllHyperionTransferTxs(size)
}

func (e *PublicAPI) GetHyperionChains() ([]*rpctypes.HyperionChainRPC, error) {
	e.logger.Debug("eth_getHyperionChains")
	return e.backend.GetHyperionChains()
}

func (e *PublicAPI) GetHyperionHistoricalFees(hyperionId uint64) (*hyperiontypes.QueryHistoricalFeesResponse, error) {
	e.logger.Debug("eth_getHyperionHistoricalFees", "hyperionId", hyperionId)
	return e.backend.GetHyperionHistoricalFees(hyperionId)
}

func (e *PublicAPI) GetValidatorHyperionData(address common.Address) (*hyperiontypes.OrchestratorData, error) {
	e.logger.Debug("eth_getValidatorHyperionData", "address", address)
	return e.backend.GetValidatorHyperionData(address)
}

func (e *PublicAPI) GetWhitelistedAddresses(hyperionId uint64) ([]string, error) {
	e.logger.Debug("eth_getWhitelistedAddresses", "hyperionId", hyperionId)
	return e.backend.GetWhitelistedAddresses(hyperionId)
}

func (e *PublicAPI) GetHyperionProjectedCurrentNetworkHeight(hyperionId uint64) (uint64, error) {
	e.logger.Debug("eth_getHyperionProjectedCurrentNetworkHeight", "hyperionId", hyperionId)
	return e.backend.GetHyperionProjectedCurrentNetworkHeight(hyperionId)
}

func (e *PublicAPI) GetHyperionNonceAlreadyObserved(hyperionId uint64, nonce uint64) (bool, error) {
	e.logger.Debug("eth_getHyperionNonceAlreadyObserved", "hyperionId", hyperionId, "nonce", nonce)
	return e.backend.GetHyperionNonceAlreadyObserved(hyperionId, nonce)
}

func (e *PublicAPI) GetHyperionSkippedNonces(hyperionId uint64) ([]*hyperiontypes.SkippedNonceFullInfo, error) {
	e.logger.Debug("eth_getHyperionSkippedNonces", "hyperionId", hyperionId)
	return e.backend.GetHyperionSkippedNonces(hyperionId)
}

func (e *PublicAPI) GetAllHyperionSkippedNonces() ([]*hyperiontypes.SkippedNonceFullInfoWithHyperionId, error) {
	e.logger.Debug("eth_getAllHyperionSkippedNonces")
	return e.backend.GetAllHyperionSkippedNonces()
}

// Dans helios-chain/rpc/namespaces/ethereum/eth/api.go

func (e *PublicAPI) GetBlockSignatures(blockHeight hexutil.Uint64) ([]*rpctypes.ValidatorSignature, error) {
	e.logger.Debug("eth_getBlockSignatures", "height", blockHeight)
	return e.backend.GetBlockSignatures(blockHeight)
}

func (e *PublicAPI) GetEpochComplete(epochId hexutil.Uint64) (*rpctypes.EpochCompleteResponse, error) {
	e.logger.Debug("eth_getEpochComplete", "epochId", epochId)
	return e.backend.GetEpochComplete(epochId)
}

func (e *PublicAPI) GetCosmosTransactionByHashFormatted(txHash string) (*map[string]interface{}, error) {
	e.logger.Debug("eth_getCosmosTransactionByHashFormatted", "txHash", txHash)
	return e.backend.GetCosmosTransactionByHashFormatted(txHash)
}

func (e *PublicAPI) GetAllTransactionReceiptsByBlockNumber(blockNum rpctypes.BlockNumber) ([]map[string]interface{}, error) {
	e.logger.Debug("eth_getAllTransactionReceiptsByBlockNumber", "height", blockNum.Int64())
	return e.backend.GetAllTransactionReceiptsByBlockNumber(blockNum)
}
