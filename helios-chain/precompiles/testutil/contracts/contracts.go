package contracts

import (
	"errors"
	"fmt"
	"math/big"

	heliosapp "helios-core/helios-chain/app"
	"helios-core/helios-chain/crypto/ethsecp256k1"
	"helios-core/helios-chain/precompiles/testutil"
	evmosutil "helios-core/helios-chain/testutil"
	evmtypes "helios-core/helios-chain/x/evm/types"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Call is a helper function to call any arbitrary smart contract.
func Call(ctx sdk.Context, app *heliosapp.HeliosApp, args CallArgs) (res abci.ExecTxResult, ethRes *evmtypes.MsgEthereumTxResponse, err error) {
	var (
		nonce    uint64
		gasLimit = args.GasLimit
	)

	if args.PrivKey == nil {
		return abci.ExecTxResult{}, nil, fmt.Errorf("private key is required; got: %v", args.PrivKey)
	}

	pk, ok := args.PrivKey.(*ethsecp256k1.PrivKey)
	if !ok {
		return abci.ExecTxResult{}, nil, errors.New("error while casting type ethsecp256k1.PrivKey on provided private key")
	}

	key, err := pk.ToECDSA()
	if err != nil {
		return abci.ExecTxResult{}, nil, fmt.Errorf("error while converting private key to ecdsa: %v", err)
	}

	addr := crypto.PubkeyToAddress(key.PublicKey)

	if args.Nonce == nil {
		nonce = app.EvmKeeper.GetNonce(ctx, addr)
	} else {
		nonce = args.Nonce.Uint64()
	}

	// if gas limit not provided
	// use default
	if args.GasLimit == 0 {
		gasLimit = 1000000
	}

	// if gas price not provided
	var gasPrice *big.Int
	if args.GasPrice == nil {
		baseFeeRes, err := app.EvmKeeper.BaseFee(ctx, &evmtypes.QueryBaseFeeRequest{})
		if err != nil {
			return abci.ExecTxResult{}, nil, err
		}
		gasPrice = baseFeeRes.BaseFee.BigInt() // default gas price == block base fee
	} else {
		gasPrice = args.GasPrice
	}
	// Create MsgEthereumTx that calls the contract
	input, err := args.ContractABI.Pack(args.MethodName, args.Args...)
	if err != nil {
		return abci.ExecTxResult{}, nil, fmt.Errorf("error while packing the input: %v", err)
	}

	// Create MsgEthereumTx that calls the contract
	msg := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		ChainID:   evmtypes.GetEthChainConfig().ChainID,
		Nonce:     nonce,
		To:        &args.ContractAddr,
		Amount:    args.Amount,
		GasLimit:  gasLimit,
		GasPrice:  gasPrice,
		GasFeeCap: args.GasFeeCap,
		GasTipCap: args.GasTipCap,
		Input:     input,
		Accesses:  args.AccessList,
	})
	msg.From = addr.Hex()

	res, err = evmosutil.DeliverEthTx(app, args.PrivKey, msg)
	if err != nil {
		return res, nil, fmt.Errorf("error during deliver tx: %s", err)
	}
	if !res.IsOK() {
		return res, nil, fmt.Errorf("error during deliver tx: %v", res.Log)
	}

	ethRes, err = evmtypes.DecodeTxResponse(res.Data)
	if err != nil {
		return res, nil, fmt.Errorf("error while decoding tx response: %v", err)
	}

	return res, ethRes, nil
}

// CallContractAndCheckLogs is a helper function to call any arbitrary smart contract and check that the logs
// contain the expected events.
func CallContractAndCheckLogs(ctx sdk.Context, app *heliosapp.HeliosApp, cArgs CallArgs, logCheckArgs testutil.LogCheckArgs) (abci.ExecTxResult, *evmtypes.MsgEthereumTxResponse, error) {
	res, ethRes, err := Call(ctx, app, cArgs)
	if err != nil {
		return res, nil, err
	}

	logCheckArgs.Res = res
	return res, ethRes, testutil.CheckLogs(logCheckArgs)
}
