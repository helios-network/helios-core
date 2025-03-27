package chronos

import (
	"fmt"
	"math/big"
	"strings"

	cmn "helios-core/helios-chain/precompiles/common"

	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	chronostypes "helios-core/helios-chain/x/chronos/types"
	"helios-core/helios-chain/x/evm/core/vm"

	cosmosmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	CreateCronMethod                    = "createCron"
	UpdateCronMethod                    = "updateCron"
	CancelCronMethod                    = "cancelCron"
	CreateCallbackConditionedCronMethod = "createCallbackConditionedCron"
)

func (p Precompile) CreateCron(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 9 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 8, len(args))
	}

	contractAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	abiStr, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}
	abiContract, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, fmt.Errorf("invalid ABI JSON")
	}

	methodName, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newFrequency")
	}
	methodABI, exists := abiContract.Methods[methodName]
	if !exists {
		return nil, fmt.Errorf("method %s does not exist in ABI", methodName)
	}

	params, ok := args[3].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid []string for newParams")
	}
	if len(params) != len(methodABI.Inputs) {
		return nil, fmt.Errorf("invalid number of params: expected %d, got %d", len(methodABI.Inputs), len(params))
	}

	frequency, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for Frequency")
	}
	if frequency == 0 {
		return nil, fmt.Errorf("invalid min Frequency should be 1")
	}

	expirationBlock, ok := args[5].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for ExpirationBlock")
	}

	gasLimit, ok := args[6].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for GasLimit")
	}
	if gasLimit == 0 {
		return nil, fmt.Errorf("invalid zero GasLimit")
	}

	maxGasPrice, ok := args[7].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for MaxGasPrice")
	}
	if maxGasPrice.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero MaxGasPrice")
	}

	amountToDeposit, ok := args[8].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for AmountToDeposit")
	}
	if amountToDeposit.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	maxGasPriceV := cosmosmath.NewIntFromBigInt(maxGasPrice)
	amountToDepositV := cosmosmath.NewIntFromBigInt(amountToDeposit)

	msg := &chronostypes.MsgCreateCron{
		OwnerAddress:    cmn.AccAddressFromHexAddress(origin).String(),
		ContractAddress: contractAddress.String(),
		AbiJson:         abiStr,
		MethodName:      methodName,
		Params:          params,
		Frequency:       frequency,
		ExpirationBlock: expirationBlock,
		GasLimit:        gasLimit,
		MaxGasPrice:     &maxGasPriceV,
		Sender:          cmn.AccAddressFromHexAddress(origin).String(),
		AmountToDeposit: &amountToDepositV,
	}

	msgSrv := chronoskeeper.NewMsgServerImpl(p.chronosKeeper)
	resp, err := msgSrv.CreateCron(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := p.EmitCronCreatedEvent(ctx, stateDB, origin, p.Address(), resp.CronId); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) UpdateCron(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 6 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	// Récupérer les valeurs des arguments dans le bon ordre
	cronId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for cronId")
	}
	if !p.chronosKeeper.StoreCronExists(ctx, cronId) {
		return nil, fmt.Errorf("invalid cron doesn't exists")
	}
	cron, _ := p.chronosKeeper.GetCron(ctx, cronId)

	newFrequency, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newFrequency")
	}
	if newFrequency == 0 {
		return nil, fmt.Errorf("invalid min Frequency should be 1")
	}

	abiContract, err := abi.JSON(strings.NewReader(cron.AbiJson))
	if err != nil {
		return nil, fmt.Errorf("invalid ABI JSON")
	}
	methodABI, exists := abiContract.Methods[cron.MethodName]
	if !exists {
		return nil, fmt.Errorf("method %s does not exist in ABI", cron.MethodName)
	}
	newParams, ok := args[2].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid []string for newParams")
	}
	if len(newParams) != len(methodABI.Inputs) {
		return nil, fmt.Errorf("invalid number of params: expected %d, got %d", len(methodABI.Inputs), len(newParams))
	}

	newExpirationBlock, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newExpirationBlock")
	}

	newGasLimit, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newGasLimit")
	}
	if newGasLimit == 0 {
		return nil, fmt.Errorf("invalid zero GasLimit")
	}

	newMaxGasPrice, ok := args[5].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for MaxGasPrice")
	}
	if newMaxGasPrice.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero MaxGasPrice")
	}

	newMaxGasPriceV := cosmosmath.NewIntFromBigInt(newMaxGasPrice)

	msg := &chronostypes.MsgUpdateCron{
		OwnerAddress:       cmn.AccAddressFromHexAddress(origin).String(),
		CronId:             cronId,
		NewFrequency:       newFrequency,
		NewParams:          newParams,
		NewExpirationBlock: newExpirationBlock,
		NewGasLimit:        newGasLimit,
		NewMaxGasPrice:     &newMaxGasPriceV,
		Sender:             cmn.AccAddressFromHexAddress(origin).String(),
	}

	msgSrv := chronoskeeper.NewMsgServerImpl(p.chronosKeeper)
	resp, err := msgSrv.UpdateCron(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := p.EmitCronUpdatedEvent(ctx, stateDB, origin, p.Address(), cronId, resp.Success); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) CancelCron(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	cronId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64")
	}
	if !p.chronosKeeper.StoreCronExists(ctx, cronId) {
		return nil, fmt.Errorf("invalid cron doesn't exists")
	}

	msg := &chronostypes.MsgCancelCron{
		OwnerAddress: cmn.AccAddressFromHexAddress(origin).String(),
		CronId:       cronId,
		Sender:       cmn.AccAddressFromHexAddress(origin).String(),
	}

	msgSrv := chronoskeeper.NewMsgServerImpl(p.chronosKeeper)
	resp, err := msgSrv.CancelCron(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := p.EmitCronCanceledEvent(ctx, stateDB, origin, p.Address(), cronId, resp.Success); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) CreateCallbackConditionedCron(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 6 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 6, len(args))
	}

	contractAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	methodName, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newFrequency")
	}

	expirationBlock, ok := args[2].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for ExpirationBlock")
	}

	gasLimit, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for GasLimit")
	}
	if gasLimit == 0 {
		return nil, fmt.Errorf("invalid zero GasLimit")
	}

	maxGasPrice, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for MaxGasPrice")
	}
	if maxGasPrice.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero MaxGasPrice")
	}

	amountToDeposit, ok := args[5].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for AmountToDeposit")
	}
	if amountToDeposit.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	maxGasPriceV := cosmosmath.NewIntFromBigInt(maxGasPrice)
	amountToDepositV := cosmosmath.NewIntFromBigInt(amountToDeposit)

	msg := &chronostypes.MsgCreateCallBackConditionedCron{
		OwnerAddress:    cmn.AccAddressFromHexAddress(origin).String(),
		ContractAddress: contractAddress.String(),
		MethodName:      methodName,
		ExpirationBlock: expirationBlock,
		GasLimit:        gasLimit,
		MaxGasPrice:     &maxGasPriceV,
		Sender:          cmn.AccAddressFromHexAddress(origin).String(),
		AmountToDeposit: &amountToDepositV,
	}

	msgSrv := chronoskeeper.NewMsgServerImpl(p.chronosKeeper)
	resp, err := msgSrv.CreateCallBackConditionedCron(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := p.EmitCronCreatedEvent(ctx, stateDB, origin, p.Address(), resp.CronId); err != nil {
		return nil, err
	}

	// exemple test envoi d'une valeur de uint256(1000) dans data et []byte{} vide dans error

	p.chronosKeeper.StoreCronCallBackData(ctx, resp.CronId, &chronostypes.CronCallBackData{
		Data:  common.BigToHash(big.NewInt(1000)).Bytes(),
		Error: []byte{},
	})

	return method.Outputs.Pack(true)
}
