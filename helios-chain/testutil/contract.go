package testutil

import (
	"fmt"
	"math/big"

	"github.com/cosmos/gogoproto/proto"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"helios-core/helios-chain/app"
	"helios-core/helios-chain/testutil/tx"
	evm "helios-core/helios-chain/x/evm/types"
)

// ContractArgs are the params used for calling a smart contract.
type ContractArgs struct {
	// Addr is the address of the contract to call.
	Addr common.Address
	// ABI is the ABI of the contract to call.
	ABI abi.ABI
	// MethodName is the name of the method to call.
	MethodName string
	// Args are the arguments to pass to the method.
	Args []interface{}
}

// ContractCallArgs is the arguments for calling a smart contract.
type ContractCallArgs struct {
	// Contract are the contract-specific arguments required for the contract call.
	Contract ContractArgs
	// Nonce is the nonce to use for the transaction.
	Nonce *big.Int
	// Amount is the aevmos amount to send in the transaction.
	Amount *big.Int
	// GasLimit to use for the transaction
	GasLimit uint64
	// PrivKey is the private key to be used for the transaction.
	PrivKey cryptotypes.PrivKey
}

// DeployContract deploys a contract with the provided private key,
// compiled contract data and constructor arguments
func DeployContract(
	ctx sdk.Context,
	heliosApp *app.HeliosApp,
	priv cryptotypes.PrivKey,
	queryClientEvm evm.QueryClient,
	contract evm.CompiledContract,
	constructorArgs ...interface{},
) (common.Address, error) {
	chainID := evm.GetEthChainConfig().ChainID
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := heliosApp.EvmKeeper.GetNonce(ctx, from)

	ctorArgs, err := contract.ABI.Pack("", constructorArgs...)
	if err != nil {
		return common.Address{}, err
	}

	data := append(contract.Bin, ctorArgs...) //nolint:gocritic
	gas, err := tx.GasLimit(ctx, from, data, queryClientEvm)
	if err != nil {
		return common.Address{}, err
	}

	baseFeeRes, err := heliosApp.EvmKeeper.BaseFee(ctx, &evm.QueryBaseFeeRequest{})
	if err != nil {
		return common.Address{}, err
	}
	baseFee := baseFeeRes.BaseFee.BigInt()
	if baseFee.Uint64() == 0 {
		baseFee = big.NewInt(1)
	}

	msgEthereumTx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     nonce,
		GasLimit:  gas,
		GasFeeCap: baseFee,
		GasTipCap: big.NewInt(1),
		Input:     data,
		Accesses:  &ethtypes.AccessList{},
	})
	msgEthereumTx.From = from.String()

	res, err := DeliverEthTx(heliosApp, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, err
	}

	if _, err := CheckEthTxResponse(res, heliosApp.AppCodec()); err != nil {
		return common.Address{}, err
	}

	return crypto.CreateAddress(from, nonce), nil
}

// DeployContractWithFactory deploys a contract using a contract factory
// with the provided factoryAddress
func DeployContractWithFactory(
	ctx sdk.Context,
	heliosApp *app.HeliosApp,
	priv cryptotypes.PrivKey,
	factoryAddress common.Address,
) (common.Address, abci.ExecTxResult, error) {
	chainID := evm.GetEthChainConfig().ChainID
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	factoryNonce := heliosApp.EvmKeeper.GetNonce(ctx, factoryAddress)
	nonce := heliosApp.EvmKeeper.GetNonce(ctx, from)

	msgEthereumTx := evm.NewTx(&evm.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    nonce,
		To:       &factoryAddress,
		GasLimit: uint64(100000),
		GasPrice: big.NewInt(1000000000),
	})
	msgEthereumTx.From = from.String()

	res, err := DeliverEthTx(heliosApp, priv, msgEthereumTx)
	if err != nil {
		return common.Address{}, abci.ExecTxResult{}, err
	}

	if _, err := CheckEthTxResponse(res, heliosApp.AppCodec()); err != nil {
		return common.Address{}, abci.ExecTxResult{}, err
	}

	return crypto.CreateAddress(factoryAddress, factoryNonce), res, err
}

// CheckEthTxResponse checks that the transaction was executed successfully
func CheckEthTxResponse(r abci.ExecTxResult, cdc codec.Codec) ([]*evm.MsgEthereumTxResponse, error) {
	if !r.IsOK() {
		return nil, fmt.Errorf("tx failed. Code: %d, Logs: %s", r.Code, r.Log)
	}

	var txData sdk.TxMsgData
	if err := cdc.Unmarshal(r.Data, &txData); err != nil {
		return nil, err
	}

	if len(txData.MsgResponses) == 0 {
		return nil, fmt.Errorf("no message responses found")
	}

	responses := make([]*evm.MsgEthereumTxResponse, 0, len(txData.MsgResponses))
	for i := range txData.MsgResponses {
		var res evm.MsgEthereumTxResponse
		if err := proto.Unmarshal(txData.MsgResponses[i].Value, &res); err != nil {
			return nil, err
		}

		if res.Failed() {
			return nil, fmt.Errorf("tx failed. VmError: %s", res.VmError)
		}
		responses = append(responses, &res)
	}

	return responses, nil
}
