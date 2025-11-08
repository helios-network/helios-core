package hyperion

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	cmn "helios-core/helios-chain/precompiles/common"

	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	"helios-core/helios-chain/x/evm/core/vm"
	hyperionkeeper "helios-core/helios-chain/x/hyperion/keeper"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	cosmosmath "cosmossdk.io/math"

	chronostypes "helios-core/helios-chain/x/chronos/types"

	evmtypes "helios-core/helios-chain/x/evm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	AddCounterpartyChainParamsMethod         = "addCounterpartyChainParams"
	UpdateCounterpartyChainInfosParamsMethod = "updateCounterpartyChainInfosParams"
	SetOrchestratorAddressesMethod           = "setOrchestratorAddresses"
	SendToChainMethod                        = "sendToChain"
	RequestDataHyperion                      = "requestData"
	CancelSendToChainMethod                  = "cancelSendToChain"
)

func (p Precompile) AddCounterpartyChainParams(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 7 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 7, len(args))
	}

	hyperionId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 hyperionId")
	}

	bridgeChainName, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string bridgeChainName")
	}

	contractSourceHash, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string contractSourceHash")
	}

	bridgeCounterpartyAddress, ok := args[3].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string bridgeCounterpartyAddress")
	}

	bridgeChainId, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 bridgeChainId")
	}

	bridgeContractStartHeight, ok := args[5].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 bridgeContractStartHeight")
	}

	logoBase64, ok := args[6].(string)
	err := fmt.Errorf("invalid string logoBase64")
	if !ok {
		return nil, err
	}

	logoHash := ""
	if logoBase64 != "" {
		logoHash, err = p.logosKeeper.StoreLogo(ctx, logoBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to store logo: %w", err)
		}
	}

	msg := &hyperiontypes.MsgAddCounterpartyChainParams{
		Authority: cmn.AccAddressFromHexAddress(origin).String(),
		CounterpartyChainParams: &hyperiontypes.CounterpartyChainParams{
			HyperionId:                    hyperionId,
			ContractSourceHash:            contractSourceHash, // hash of the BridgeCounterparty Smart Contract
			BridgeCounterpartyAddress:     bridgeCounterpartyAddress,
			BridgeChainId:                 bridgeChainId,
			BridgeChainLogo:               logoHash,
			BridgeChainType:               "evm",
			BridgeChainName:               bridgeChainName,
			SignedValsetsWindow:           25000,
			SignedBatchesWindow:           25000,
			SignedClaimsWindow:            25000,
			TargetBatchTimeout:            3600000, // 1 hour
			AverageBlockTime:              2000,
			AverageCounterpartyBlockTime:  15000,
			SlashFractionValset:           cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionBatch:            cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionClaim:            cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionConflictingClaim: cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			UnbondSlashingValsetsWindow:   25000,
			SlashFractionBadEthSignature:  cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			BridgeContractStartHeight:     bridgeContractStartHeight,
			ValsetReward:                  sdk.Coin{Denom: "ahelios", Amount: cosmosmath.NewInt(0)},
			Initializer:                   cmn.AccAddressFromHexAddress(origin).String(),
			DefaultTokens:                 []*hyperiontypes.TokenAddressToDenomWithGenesisInfos{},
			Rpcs:                          []*hyperiontypes.Rpc{},
			OffsetValsetNonce:             0,
			MinCallExternalDataGas:        10000000, // 10M Gas
			Paused:                        true,
		},
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err = msgSrv.AddCounterpartyChainParams(ctx, msg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) SetOrchestratorAddresses(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	orchestratorAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	hyperionId, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid hyperionId uint64")
	}

	msg := &hyperiontypes.MsgSetOrchestratorAddresses{
		Sender:       cmn.AccAddressFromHexAddress(origin).String(),
		Orchestrator: cmn.AccAddressFromHexAddress(orchestratorAddress).String(),
		EthAddress:   orchestratorAddress.String(),
		HyperionId:   hyperionId,
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.SetOrchestratorAddresses(ctx, msg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) SendToChain(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	chainId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 chainId")
	}

	dest, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string dest address")
	}

	contractAddress, ok := args[2].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid contract address")
	}

	amount, ok := args[3].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint256 for AmountToDeposit")
	}
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	bridgeFee, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint256 for AmountToDeposit")
	}
	if bridgeFee.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	amountV := cosmosmath.NewIntFromBigInt(amount)
	bridgeFeeV := cosmosmath.NewIntFromBigInt(bridgeFee)

	tokenPair, ok := p.erc20Keeper.GetTokenPair(ctx, p.erc20Keeper.GetTokenPairID(ctx, contractAddress.String()))

	if !ok {
		// no erc20 exists for this token
		return nil, fmt.Errorf("invalid token %s not registered as tokenPair", contractAddress.String())
	}

	msg := &hyperiontypes.MsgSendToChain{
		Sender:      cmn.AccAddressFromHexAddress(origin).String(),
		DestChainId: chainId,
		Dest:        dest,
		Amount:      sdk.NewCoin(tokenPair.Denom, amountV),
		BridgeFee:   sdk.NewCoin(sdk.DefaultBondDenom, bridgeFeeV), // HLS
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.SendToChain(ctx, msg)
	if err != nil {
		return nil, err
	}

	// if err := p.EmitCronCreatedEvent(ctx, stateDB, origin, p.Address(), resp.CronId); err != nil {
	// 	return nil, err
	// }

	return method.Outputs.Pack(true)
}

// RequestData is the function that will be called by the precompile to request data from the hyperion
//
// example payable call in solidity:
//
//	function requestData(address _source, bytes _abiCall, uint256 _chainId, string memory _callbackSelector, uint256 _maxGasPrice, uint256 _gasLimit) external payable {
//	    hyperion.requestData(_source, _abiCall, _chainId, _callbackSelector, _maxGasPrice, _gasLimit);
//	}
func (p Precompile) RequestData(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// Extract args
	chainId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid chainId type")
	}

	externalContract, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid source address type")
	}

	abiCall, ok := args[2].([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid abiCall type")
	}
	if len(abiCall) == 0 {
		return nil, fmt.Errorf("invalid abiCall type empty")
	}

	callbackSelector, ok := args[3].(string)
	if !ok {
		return nil, fmt.Errorf("invalid callbackSelector function name")
	}

	code := stateDB.GetCode(contract.CallerAddress)
	if !evmtypes.FunctionExists(code, callbackSelector+"(bytes,bytes)") {
		return nil, fmt.Errorf("invalid callbackSelector function %s does not exist", callbackSelector)
	}

	maxGasPrice, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid gasPrice type")
	}

	baseGasLimit, ok := args[5].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid gasLimit type")
	}

	cronGasLimit := big.NewInt(0).Div(baseGasLimit, big.NewInt(2)) // TODO: get from hyperion params
	hyperionMaxGasLimit := big.NewInt(0).Div(baseGasLimit, big.NewInt(2))

	hyperionParams := p.hyperionKeeper.GetHyperionParamsFromChainId(ctx, chainId)

	if hyperionParams == nil {
		return nil, fmt.Errorf("invalid chainId")
	}

	if hyperionMaxGasLimit.Cmp(big.NewInt(int64(hyperionParams.MinCallExternalDataGas))) < 0 {
		return nil, fmt.Errorf("invalid hyperionMaxGasLimit")
	}

	// Create parameters for chronos cron job
	expirationBlock := ctx.BlockHeight() + 100 // Set reasonable expiration
	actualGasPrice := p.chronosKeeper.EvmKeeper.GetBaseFee(ctx)
	baseDenom := evmtypes.GetEVMCoinDenom()

	heliosTokenAddress, err := p.erc20Keeper.GetCoinAddress(ctx, baseDenom)
	if err != nil {
		return nil, err
	}

	if maxGasPrice.Cmp(actualGasPrice) < 0 {
		return nil, fmt.Errorf("gasPrice is too low")
	}

	hyperionFee := big.NewInt(0).Mul(hyperionMaxGasLimit, maxGasPrice)
	evmExecutionFee := big.NewInt(0).Mul(cronGasLimit, maxGasPrice)

	balance, err := p.bankKeeper.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: cmn.AccAddressFromHexAddress(origin).String(),
		Denom:   baseDenom,
	})
	if err != nil {
		return nil, err
	}

	if balance.Balance.Amount.LT(cosmosmath.NewIntFromBigInt(hyperionFee).Add(cosmosmath.NewIntFromBigInt(evmExecutionFee))) {
		return nil, fmt.Errorf("insufficient balance for hyperion fee and evm execution fee %s needed", cosmosmath.NewIntFromBigInt(hyperionFee).Add(cosmosmath.NewIntFromBigInt(evmExecutionFee)).Sub(balance.Balance.Amount))
	}

	// Transfert des fonds au module Hyperion
	hyperionFeeCoins := sdk.NewCoins(sdk.NewCoin(baseDenom, cosmosmath.NewIntFromBigInt(hyperionFee)))
	if err := p.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		cmn.AccAddressFromHexAddress(origin),
		hyperiontypes.ModuleName,
		hyperionFeeCoins,
	); err != nil {
		return nil, err
	}

	ptMaxGasPrice := cosmosmath.NewIntFromBigInt(maxGasPrice)
	maxExecutionFee := cosmosmath.NewIntFromBigInt(evmExecutionFee)

	// Create the chronos message
	msg := &chronostypes.MsgCreateCallBackConditionedCron{
		OwnerAddress:    cmn.AccAddressFromHexAddress(origin).String(),
		ContractAddress: contract.CallerAddress.String(),
		MethodName:      callbackSelector,
		ExpirationBlock: uint64(expirationBlock),
		GasLimit:        cronGasLimit.Uint64(),
		MaxGasPrice:     &ptMaxGasPrice,
		Sender:          cmn.AccAddressFromHexAddress(origin).String(),
		AmountToDeposit: &maxExecutionFee,
	}

	// Submit the message to create the cron job
	msgServer := chronoskeeper.NewMsgServerImpl(p.chronosKeeper)
	response, err := msgServer.CreateCallBackConditionedCron(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create cron job: %w", err)
	}
	// Generate a task ID - convert the cron ID to uint256
	taskId := new(big.Int).SetUint64(response.CronId)

	hyperionFeeToken := hyperiontypes.Token{
		Contract: heliosTokenAddress.Hex(),
		Amount:   cosmosmath.NewIntFromBigInt(hyperionFee),
	}
	abiCallHex := hex.EncodeToString(abiCall)

	outgoingTx, err := p.hyperionKeeper.BuildOutgoingExternalDataTX(ctx, hyperionParams.HyperionId, strconv.FormatUint(response.CronId, 10), externalContract, abiCallHex, origin.Hex(), &hyperionFeeToken, uint64(expirationBlock)-1)
	if err != nil {
		return nil, fmt.Errorf("failed to create hyperionoutgoing tx: %w", err)
	}

	ctx.Logger().Debug("Created conditional cron job",
		"taskId", taskId.String(),
		"origin", origin.String(),
		"callerAddress", contract.CallerAddress.String(),
		"contract.Address()", contract.Address().String(),
		"cronId", response.CronId,
		"methodName", callbackSelector,
		"outgoingTxId", outgoingTx.Id,
	)

	// Return the task ID as uint256
	return method.Outputs.Pack(taskId)
}

func (p Precompile) UpdateCounterpartyChainInfosParams(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 3 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	chainId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 chainId")
	}

	logo, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string logo")
	}

	name, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string name")
	}

	msg := &hyperiontypes.MsgUpdateCounterpartyChainInfosParams{
		Signer:          cmn.AccAddressFromHexAddress(origin).String(),
		BridgeChainId:   chainId,
		BridgeChainLogo: logo,
		BridgeChainName: name,
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.UpdateCounterpartyChainInfosParams(ctx, msg)
	if err != nil {
		return nil, err
	}

	// if err := p.EmitCronCreatedEvent(ctx, stateDB, origin, p.Address(), resp.CronId); err != nil {
	// 	return nil, err
	// }

	return method.Outputs.Pack(true)
}

func (p Precompile) CancelSendToChain(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	chainId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 chainId")
	}

	transactionId, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 transactionId")
	}

	msg := &hyperiontypes.MsgCancelSendToChain{
		TransactionId: transactionId,
		Sender:        cmn.AccAddressFromHexAddress(origin).String(),
		ChainId:       chainId,
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.CancelSendToChain(ctx, msg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
