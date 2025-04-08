package hyperion

import (
	"embed"
	"fmt"

	cmn "helios-core/helios-chain/precompiles/common"
	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	"helios-core/helios-chain/x/evm/core/vm"
	evmtypes "helios-core/helios-chain/x/evm/types"
	hyperionkeeper "helios-core/helios-chain/x/hyperion/keeper"

	erc20keeper "helios-core/helios-chain/x/erc20/keeper"

	logoskeeper "helios-core/helios-chain/x/logos/keeper"

	storetypes "cosmossdk.io/store/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
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
	hyperionKeeper hyperionkeeper.Keeper
	erc20Keeper    erc20keeper.Keeper
	bankKeeper     bankkeeper.Keeper
	chronosKeeper  chronoskeeper.Keeper
	logosKeeper    logoskeeper.Keeper
}

// LoadABI loads the gov ABI from the embedded abi.json file
// for the precompile.
func LoadABI() (abi.ABI, error) {
	return cmn.LoadABI(f, "abi.json")
}

func NewPrecompile(
	hyperionKeeper hyperionkeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
	erc20Keeper erc20keeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	chronosKeeper chronoskeeper.Keeper,
	logosKeeper logoskeeper.Keeper,
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
		hyperionKeeper: hyperionKeeper,
		erc20Keeper:    erc20Keeper,
		bankKeeper:     bankKeeper,
		chronosKeeper:  chronosKeeper,
		logosKeeper:    logosKeeper,
	}

	// SetAddress defines the address of the gov precompiled contract.
	p.SetAddress(common.HexToAddress(evmtypes.HyperionPrecompileAddress))

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
	case AddCounterpartyChainParamsMethod:
		bz, err = p.AddCounterpartyChainParams(ctx, evm.Origin, contract, stateDB, method, args)
	case SendToChainMethod:
		bz, err = p.SendToChain(ctx, evm.Origin, contract, stateDB, method, args)
	case SetOrchestratorAddressesMethod:
		bz, err = p.SetOrchestratorAddresses(ctx, evm.Origin, contract, stateDB, method, args)
	case UpdateCounterpartyChainInfosParamsMethod:
		bz, err = p.UpdateCounterpartyChainInfosParams(ctx, evm.Origin, contract, stateDB, method, args)
	// ask for external chain datas
	case RequestDataHyperion:
		bz, err = p.RequestData(ctx, evm.Origin, contract, stateDB, method, args)

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
	case AddCounterpartyChainParamsMethod:
		return true
	case SendToChainMethod:
		return true
	case SetOrchestratorAddressesMethod:
		return true
	case RequestDataHyperion:
		return true
	case UpdateCounterpartyChainInfosParamsMethod:
		return true
	default:
		return false
	}
}
