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
	// HexToBech32Method defines the ABI method name to convert a EIP-55
	// hex formatted address to bech32 address string.
	ScheduleEVMCallMethod = "scheduleEVMCall"
)

// HexToBech32 converts a hex address to its corresponding Bech32 format. The Human Readable Prefix
// (HRP) must be provided in the arguments. This function fails if the address is invalid or if the
// bech32 conversion fails.
func (p Precompile) ScheduleEVMCall(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	ctx.Logger().Info("HAHHAHAHAHAHHAHAHAHAHAH")
	if len(args) != 3 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	address, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	contractAddress, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	abi, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	msg := &chronostypes.MsgScheduleEVMCall{
		OwnerAddress:    cmn.AccAddressFromHexAddress(address).String(),
		ContractAddress: contractAddress.String(),
		AbiJson:         abi,
		MethodName:      "increment",
		Params:          []string{},
		Frequency:       uint64(1),
		ExpirationBlock: uint64(0),
		GasLimit:        uint64(300000),
	}

	msgSrv := chronoskeeper.NewMsgServerImpl(p.chronosKeeper)
	if _, err := msgSrv.ScheduleEVMCall(ctx, msg); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
