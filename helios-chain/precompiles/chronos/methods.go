package chronos

import (
	"fmt"

	cmn "helios-core/helios-chain/precompiles/common"

	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	chronostypes "helios-core/helios-chain/x/chronos/types"
	"helios-core/helios-chain/x/evm/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	CreateCronMethod = "createCron"
	UpdateCronMethod = "updateCron"
	CancelCronMethod = "cancelCron"
)

func (p Precompile) CreateCron(
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

	contractAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	abi, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	methodName, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newFrequency")
	}

	params, ok := args[3].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid []string for newParams")
	}

	frequency, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newFrequency")
	}

	expirationBlock, ok := args[5].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newExpirationBlock")
	}

	gasLimit, ok := args[6].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newGasLimit")
	}

	msg := &chronostypes.MsgCreateCron{
		OwnerAddress:    cmn.AccAddressFromHexAddress(origin).String(),
		ContractAddress: contractAddress.String(),
		AbiJson:         abi,
		MethodName:      methodName,
		Params:          params,
		Frequency:       frequency,
		ExpirationBlock: expirationBlock,
		GasLimit:        gasLimit,
		Sender:          cmn.AccAddressFromHexAddress(origin).String(),
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

	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	// Récupérer les valeurs des arguments dans le bon ordre
	cronId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for cronId")
	}

	newFrequency, ok := args[1].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newFrequency")
	}

	newParams, ok := args[2].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid []string for newParams")
	}

	newExpirationBlock, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newExpirationBlock")
	}

	newGasLimit, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for newGasLimit")
	}

	msg := &chronostypes.MsgUpdateCron{
		OwnerAddress:       cmn.AccAddressFromHexAddress(origin).String(),
		CronId:             cronId,
		NewFrequency:       newFrequency,
		NewParams:          newParams,
		NewExpirationBlock: newExpirationBlock,
		NewGasLimit:        newGasLimit,
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
