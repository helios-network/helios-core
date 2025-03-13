package types

import (
	"context"
	"helios-core/helios-chain/x/evm/core/vm"
	"math/big"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"

	"helios-core/helios-chain/x/evm/statedb"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

type EVMKeeper interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	GetAccountWithoutBalance(ctx sdk.Context, addr common.Address) *statedb.Account
	EstimateGasInternal(c context.Context, req *evmtypes.EthCallRequest, fromType evmtypes.CallType) (*evmtypes.EstimateGasResponse, error)
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
	DeleteAccount(ctx sdk.Context, addr common.Address) error
	IsAvailableStaticPrecompile(params *evmtypes.Params, address common.Address) bool
	CallEVM(ctx sdk.Context, abi abi.ABI, from, contract common.Address, commit bool, method string, args ...interface{}) (*evmtypes.MsgEthereumTxResponse, error)
	CallEVMWithData(ctx sdk.Context, from common.Address, contract *common.Address, data []byte, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
	GetCode(ctx sdk.Context, hash common.Hash) []byte
	SetCode(ctx sdk.Context, hash []byte, bytecode []byte)
	SetAccount(ctx sdk.Context, address common.Address, account statedb.Account) error
	GetAccount(ctx sdk.Context, address common.Address) *statedb.Account

	ResetTransientGasUsed(ctx sdk.Context)
	GetTxIndexTransient(ctx sdk.Context) uint64
	// GetBaseFee returns the BaseFee param from the fee market module
	// adapted according to the evm denom decimals
	GetBaseFee(ctx sdk.Context) *big.Int
	// GetMinGasPrice returns the MinGasPrice param from the fee market module
	// adapted according to the evm denom decimals
	GetMinGasPrice(ctx sdk.Context) math.LegacyDec

	DeductTxCostsFromUserBalance(ctx sdk.Context, fees sdk.Coins, from common.Address) error

	RefundGas(ctx sdk.Context, msg core.Message, leftoverGas uint64, denom string) error
}

type DecoratorUtils struct {
	EvmParams          evmtypes.Params
	Rules              params.Rules
	Signer             ethtypes.Signer
	BaseFee            *big.Int
	MempoolMinGasPrice math.LegacyDec
	GlobalMinGasPrice  math.LegacyDec
	BlockTxIndex       uint64
	TxGasLimit         uint64
	GasWanted          uint64
	MinPriority        int64
	TxFee              *big.Int
}
