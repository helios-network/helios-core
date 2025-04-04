package hyperion

import (
	"fmt"
	"math/big"

	cmn "helios-core/helios-chain/precompiles/common"

	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	"helios-core/helios-chain/x/evm/core/vm"
	hyperionkeeper "helios-core/helios-chain/x/hyperion/keeper"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	cosmosmath "cosmossdk.io/math"

	chronostypes "helios-core/helios-chain/x/chronos/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	AddCounterpartyChainParamsMethod = "addCounterpartyChainParams"
	SetOrchestratorAddressesMethod   = "setOrchestratorAddresses"
	SendToChainMethod                = "sendToChain"
	RequestDataHyperion              = "requestData"
)

func (p Precompile) AddCounterpartyChainParams(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	ctx.Logger().Info("AddCounterpartyChainParams -10")

	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	ctx.Logger().Info("AddCounterpartyChainParams -9")

	hyperionId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 hyperionId")
	}

	contractSourceHash, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string contractSourceHash")
	}

	bridgeCounterpartyAddress, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string bridgeCounterpartyAddress")
	}

	bridgeChainId, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 bridgeChainId")
	}

	bridgeContractStartHeight, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 bridgeContractStartHeight")
	}

	ctx.Logger().Info("AddCounterpartyChainParams -8")

	msg := &hyperiontypes.MsgAddCounterpartyChainParams{
		Orchestrator: cmn.AccAddressFromHexAddress(origin).String(),
		CounterpartyChainParams: &hyperiontypes.CounterpartyChainParams{
			HyperionId:                    hyperionId,
			ContractSourceHash:            contractSourceHash, // hash of the BridgeCounterparty Smart Contract
			BridgeCounterpartyAddress:     bridgeCounterpartyAddress,
			BridgeChainId:                 bridgeChainId,
			SignedValsetsWindow:           25000,
			SignedBatchesWindow:           25000,
			SignedClaimsWindow:            25000,
			TargetBatchTimeout:            43200000,
			AverageBlockTime:              2000,
			AverageCounterpartyBlockTime:  15000,
			SlashFractionValset:           cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionBatch:            cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionClaim:            cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionConflictingClaim: cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			UnbondSlashingValsetsWindow:   25000,
			SlashFractionBadEthSignature:  cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			CosmosCoinDenom:               "ahelios",
			CosmosCoinErc20Contract:       "",
			ClaimSlashingEnabled:          false,
			BridgeContractStartHeight:     bridgeContractStartHeight,
			ValsetReward:                  sdk.Coin{Denom: "ahelios", Amount: cosmosmath.NewInt(0)},
		},
	}

	ctx.Logger().Info("AddCounterpartyChainParams -7", "msg", msg)

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.AddCounterpartyChainParams(ctx, msg)
	if err != nil {
		return nil, err
	}

	ctx.Logger().Info("AddCounterpartyChainParams -6")

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
		BridgeFee:   sdk.NewCoin(tokenPair.Denom, bridgeFeeV),
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

func (p Precompile) RequestData(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	// value sent to the precompile for overall execution fee
	value := contract.Value()

	if value.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("insufficient funds for execution")
	}

	// Extract args
	_, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid chainId type")
	}

	source, ok := args[1].(common.Address) // the contract to callback
	if !ok {
		return nil, fmt.Errorf("invalid source address type")
	}

	// TODO: send to hyperion for calling on external chains
	// abiCall, ok := args[2].([]byte)
	// if !ok {
	// 	return nil, fmt.Errorf("invalid abiCall type")
	// }

	//TODO: check if callbackSelector exist
	callbackSelector, ok := args[3].(string)
	if !ok {
		return nil, fmt.Errorf("invalid callbackSelector function name")
	}

	maxCallbackGas, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid maxCallbackGas type")
	}

	gasLimit, ok := args[5].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid gasLimit type")
	}

	// Create parameters for chronos cron job
	expirationBlock := ctx.BlockHeight() + 100 // Set reasonable expiration

	hyperionFee := new(big.Int).Div(value, big.NewInt(2))
	evmExecutionFee := new(big.Int).Div(value, big.NewInt(2))

	// Transfert des fonds au module Hyperion
	hyperionFeeCoins := sdk.NewCoins(sdk.NewCoin("ahelios", cosmosmath.NewIntFromBigInt(hyperionFee)))
	if err := p.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		cmn.AccAddressFromHexAddress(origin),
		hyperiontypes.ModuleName,
		hyperionFeeCoins,
	); err != nil {
		return nil, err
	}

	maxGasPrice := cosmosmath.NewIntFromBigInt(maxCallbackGas)
	maxExecutionFee := cosmosmath.NewIntFromBigInt(evmExecutionFee)

	// Create the chronos message
	msg := &chronostypes.MsgCreateCallBackConditionedCron{
		OwnerAddress:    cmn.AccAddressFromHexAddress(origin).String(),
		ContractAddress: contract.CallerAddress.String(),
		MethodName:      callbackSelector,
		ExpirationBlock: uint64(expirationBlock),
		GasLimit:        gasLimit.Uint64(),
		MaxGasPrice:     &maxGasPrice,
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

	//TODO: REMOVE TEST
	// Simple mock price
	ethPrice := big.NewInt(2000) // Mock price of $2000

	// Convert price to bytes (as uint256)
	priceBytes := common.BigToHash(ethPrice).Bytes()
	p.chronosKeeper.StoreCronCallBackData(ctx, response.CronId, &chronostypes.CronCallBackData{
		Data:  priceBytes,
		Error: []byte{}, //[]byte(fmt.Sprintf("JSON parse error: %v", err)),
	})

	//TODO: REMOVE TEST

	//TODO: instead call Hyperion with Task ID to execute with chain_id and abiCall and source(contract hash to call)

	ctx.Logger().Debug("Created conditional cron job",
		"taskId", taskId.String(),
		"origin", origin.String(),
		"callerAddress", contract.CallerAddress.String(),
		"contract.Address()", contract.Address().String(),
		"cronId", response.CronId,
		"source", source.Hex(),
		"methodName", callbackSelector)

	// Return the task ID as uint256
	return method.Outputs.Pack(taskId)
}
