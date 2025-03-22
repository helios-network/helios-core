package chronos

import (
	"embed"
	"fmt"

	cmn "helios-core/helios-chain/precompiles/common"
	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	"helios-core/helios-chain/x/evm/core/vm"
	evmtypes "helios-core/helios-chain/x/evm/types"

	storetypes "cosmossdk.io/store/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/ethereum/go-ethereum/common"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

type Precompile struct {
	cmn.Precompile
	chronosKeeper chronoskeeper.Keeper
}

// LoadABI loads the gov ABI from the embedded abi.json file
// for the precompile.
func LoadABI() (abi.ABI, error) {
	return cmn.LoadABI(f, "abi.json")
}

func NewPrecompile(
	chronosKeeper chronoskeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	abi, err := LoadABI()
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  abi,
			AuthzKeeper:          authzKeeper,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
			ApprovalExpiration:   cmn.DefaultExpirationDuration, // should be configurable in the future.
		},
		chronosKeeper: chronosKeeper,
	}

	// SetAddress defines the address of the gov precompiled contract.
	p.SetAddress(common.HexToAddress(evmtypes.ChronosPrecompileAddress))

	return p, nil
}

// RequiredGas calculates the precompiled contract's base gas rate.
func (p Precompile) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}
	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}
	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	if err := stateDB.Commit(); err != nil {
		return nil, err
	}

	switch method.Name {
	// gov transactions
	case CreateCronMethod:
		bz, err = p.CreateCron(ctx, evm.Origin, contract, stateDB, method, args)
	case UpdateCronMethod:
		bz, err = p.UpdateCron(ctx, evm.Origin, contract, stateDB, method, args)
	case CancelCronMethod:
		bz, err = p.CancelCron(ctx, evm.Origin, contract, stateDB, method, args)
	case CreateCallbackConditionedCronMethod:
		bz, err = p.CreateCallbackConditionedCron(ctx, evm.Origin, contract, stateDB, method, args)
	default:
		return nil, fmt.Errorf(cmn.ErrUnknownMethod, method.Name)
	}

	if err != nil {
		return nil, err
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
		return nil, err
	}

	return bz, nil
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case CreateCronMethod:
		return true
	case UpdateCronMethod:
		return true
	case CancelCronMethod:
		return true
	case CreateCallbackConditionedCronMethod:
		return true
	default:
		return false
	}
}
