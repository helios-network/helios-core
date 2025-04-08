package backend

import (
	"context"
	"fmt"
	"math/big"
	"time"

	rpctypes "helios-core/helios-chain/rpc/types"
	"helios-core/helios-chain/server/config"
	evmostypes "helios-core/helios-chain/types"
	chronostypes "helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	"cosmossdk.io/log"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	tmrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// BackendI implements the Cosmos and EVM backend.
type BackendI interface { //nolint: revive
	EVMBackend
}

// EVMBackend implements the functionality shared within ethereum namespaces
// as defined by EIP-1474: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-1474.md
// Implemented by Backend.
type EVMBackend interface {
	// Node specific queries
	Accounts() ([]common.Address, error)
	Syncing() (interface{}, error)
	SetEtherbase(etherbase common.Address) bool
	SetGasPrice(gasPrice hexutil.Big) bool
	ImportRawKey(privkey, password string) (common.Address, error)
	ListAccounts() ([]common.Address, error)
	NewMnemonic(uid string, language keyring.Language, hdPath, bip39Passphrase string, algo keyring.SignatureAlgo) (*keyring.Record, error)
	UnprotectedAllowed() bool
	RPCGasCap() uint64            // global gas cap for eth_call over rpc: DoS protection
	RPCEVMTimeout() time.Duration // global timeout for eth_call over rpc: DoS protection
	RPCTxFeeCap() float64         // RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for send-transaction variants. The unit is ether.
	RPCMinGasPrice() *big.Int

	// Sign Tx
	Sign(address common.Address, data hexutil.Bytes) (hexutil.Bytes, error)
	SendTransaction(args evmtypes.TransactionArgs) (common.Hash, error)
	SignTypedData(address common.Address, typedData apitypes.TypedData) (hexutil.Bytes, error)

	// Blocks Info
	BlockNumber() (hexutil.Uint64, error)
	GetBlockByNumber(blockNum rpctypes.BlockNumber, fullTx bool) (map[string]interface{}, error)
	GetBlockByHash(hash common.Hash, fullTx bool) (map[string]interface{}, error)
	GetBlocksByPageAndSize(page hexutil.Uint64, size hexutil.Uint64, fullTx bool) ([]map[string]interface{}, error)

	// Proposals Info
	GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error)
	GetProposal(id hexutil.Uint64) (map[string]interface{}, error)

	GetBlockTransactionCountByHash(hash common.Hash) *hexutil.Uint
	GetBlockTransactionCountByNumber(blockNum rpctypes.BlockNumber) *hexutil.Uint
	TendermintBlockByNumber(blockNum rpctypes.BlockNumber) (*tmrpctypes.ResultBlock, error)
	TendermintBlockByHash(blockHash common.Hash) (*tmrpctypes.ResultBlock, error)
	BlockNumberFromTendermint(blockNrOrHash rpctypes.BlockNumberOrHash) (rpctypes.BlockNumber, error)
	BlockNumberFromTendermintByHash(blockHash common.Hash) (*big.Int, error)
	EthMsgsFromTendermintBlock(block *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults) []*evmtypes.MsgEthereumTx
	BlockBloom(blockRes *tmrpctypes.ResultBlockResults) (ethtypes.Bloom, error)
	HeaderByNumber(blockNum rpctypes.BlockNumber) (*ethtypes.Header, error)
	HeaderByHash(blockHash common.Hash) (*ethtypes.Header, error)
	RPCBlockFromTendermintBlock(resBlock *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults, fullTx bool) (map[string]interface{}, error)
	EthBlockByNumber(blockNum rpctypes.BlockNumber) (*ethtypes.Block, error)
	EthBlockFromTendermintBlock(resBlock *tmrpctypes.ResultBlock, blockRes *tmrpctypes.ResultBlockResults) (*ethtypes.Block, error)

	// Account Info
	GetAccountType(address common.Address) (string, error)
	GetHeliosAddress(address common.Address) (string, error)
	GetHeliosValoperAddress(address common.Address) (string, error)
	GetCode(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error)
	GetBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error)
	GetStorageAt(address common.Address, key string, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error)
	GetProof(address common.Address, storageKeys []string, blockNrOrHash rpctypes.BlockNumberOrHash) (*rpctypes.AccountResult, error)
	GetTransactionCount(address common.Address, blockNum rpctypes.BlockNumber) (*hexutil.Uint64, error)
	GetAccountTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error)
	GetTokenBalance(address common.Address, tokenAddress common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error)
	GetTokensBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) ([]rpctypes.TokenBalance, error)

	// Chain Info
	ChainID() (*hexutil.Big, error)
	ChainConfig() *params.ChainConfig
	GlobalMinGasPrice() (*big.Int, error)
	BaseFee(blockRes *tmrpctypes.ResultBlockResults) (*big.Int, error)
	CurrentHeader() (*ethtypes.Header, error)
	PendingTransactions() ([]*sdk.Tx, error)
	GetCoinbase() (sdk.AccAddress, error)
	FeeHistory(blockCount rpc.DecimalOrHex, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (*rpctypes.FeeHistoryResult, error)
	SuggestGasTipCap(baseFee *big.Int) (*big.Int, error)
	ChainSize() (*rpctypes.ChainSize, error)

	// Tokens Info
	GetTokensByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error)
	GetTokenDetails(tokenAddress common.Address) (*rpctypes.TokenDetails, error)

	// Tx Info
	GetTransactionByHash(txHash common.Hash) (*rpctypes.RPCTransaction, error)
	GetTxByEthHash(txHash common.Hash) (*evmostypes.TxResult, error)
	GetTxByTxIndex(height int64, txIndex uint) (*evmostypes.TxResult, error)
	GetTransactionByBlockAndIndex(block *tmrpctypes.ResultBlock, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionReceipt(hash common.Hash) (map[string]interface{}, error)
	GetTransactionLogs(hash common.Hash) ([]*ethtypes.Log, error)
	GetTransactionByBlockHashAndIndex(hash common.Hash, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionByBlockNumberAndIndex(blockNum rpctypes.BlockNumber, idx hexutil.Uint) (*rpctypes.RPCTransaction, error)
	GetTransactionsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error)
	GetLastTransactionsInfo(size hexutil.Uint64) (map[string]interface{}, error)

	// Send Transaction
	Resend(args evmtypes.TransactionArgs, gasPrice *hexutil.Big, gasLimit *hexutil.Uint64) (common.Hash, error)
	SendRawTransaction(data hexutil.Bytes) (common.Hash, error)
	SetTxDefaults(args evmtypes.TransactionArgs) (evmtypes.TransactionArgs, error)
	EstimateGas(args evmtypes.TransactionArgs, blockNrOptional *rpctypes.BlockNumber) (hexutil.Uint64, error)
	DoCall(args evmtypes.TransactionArgs, blockNr rpctypes.BlockNumber) (*evmtypes.MsgEthereumTxResponse, error)
	GasPrice() (*hexutil.Big, error)

	// Filter API
	GetLogs(hash common.Hash) ([][]*ethtypes.Log, error)
	GetLogsByHeight(height *int64) ([][]*ethtypes.Log, error)
	BloomStatus() (uint64, uint64)

	// Tracing
	TraceTransaction(hash common.Hash, config *evmtypes.TraceConfig) (interface{}, error)
	TraceBlock(height rpctypes.BlockNumber, config *evmtypes.TraceConfig, block *tmrpctypes.ResultBlock) ([]*evmtypes.TxTraceResult, error)

	// Staking [to update]
	GetDelegations(address common.Address) ([]rpctypes.DelegationRPC, error)
	GetDelegation(address common.Address, validatorAddress common.Address) (*rpctypes.DelegationRPC, error)
	GetValidator(address common.Address) (*rpctypes.ValidatorRPC, error)
	GetValidatorAndHisDelegation(address common.Address) (*rpctypes.ValidatorWithDelegationRPC, error)
	GetValidatorCommission(address common.Address) (*rpctypes.ValidatorCommissionRPC, error)
	GetValidatorOutStandingRewards(address common.Address) (*rpctypes.ValidatorRewardRPC, error)
	GetValidatorWithHisDelegationAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndDelegationRPC, error)
	GetValidatorAndHisCommission(address common.Address) (*rpctypes.ValidatorWithCommissionRPC, error)
	GetValidatorsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorRPC, error)
	GetActiveValidatorCount() (int, error)
	GetAllWhitelistedAssets() ([]rpctypes.WhitelistedAssetRPC, error)

	//cron
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

	// hyperion
	GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.TransferTx, error)
	GetHyperionChains() ([]*rpctypes.HyperionChainRPC, error)
}

var _ BackendI = (*Backend)(nil)

// Backend implements the BackendI interface
type Backend struct {
	ctx                 context.Context
	clientCtx           client.Context
	rpcClient           tmrpcclient.SignClient
	queryClient         *rpctypes.QueryClient // gRPC query client
	logger              log.Logger
	chainID             *big.Int
	cfg                 config.Config
	allowUnprotectedTxs bool
	indexer             evmostypes.EVMTxIndexer
}

// NewBackend creates a new Backend instance for cosmos and ethereum namespaces
func NewBackend(
	ctx *server.Context,
	logger log.Logger,
	clientCtx client.Context,
	allowUnprotectedTxs bool,
	indexer evmostypes.EVMTxIndexer,
) *Backend {

	chainID, err := evmostypes.ParseChainID(clientCtx.ChainID)
	if err != nil {
		panic(err)
	}

	logger.Info("Creating Backend",
		"chain_id", clientCtx.ChainID,
		"allow_unprotected_txs", allowUnprotectedTxs,
	)

	appConf, err := config.GetConfig(ctx.Viper)
	if err != nil {
		panic(err)
	}

	rpcClient, ok := clientCtx.Client.(tmrpcclient.SignClient)
	if !ok {
		panic(fmt.Sprintf("invalid rpc client, expected: tmrpcclient.SignClient, got: %T", clientCtx.Client))
	}

	return &Backend{
		ctx:                 context.Background(),
		clientCtx:           clientCtx,
		rpcClient:           rpcClient,
		queryClient:         rpctypes.NewQueryClient(clientCtx),
		logger:              logger.With("module", "backend"),
		chainID:             chainID,
		cfg:                 appConf,
		allowUnprotectedTxs: allowUnprotectedTxs,
		indexer:             indexer,
	}
}
