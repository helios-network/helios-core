package ibctransfer

import (
	"embed"
	"fmt"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/evm/core/vm"
	evmtypes "helios-core/helios-chain/x/evm/types"
	ibckeeper "helios-core/helios-chain/x/ibc/transfer/keeper"

	storetypes "cosmossdk.io/store/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	abiPath               string = "abi.json"
	TransferMethod        string = "ibcTransfer"
	GetSupportedChains    string = "getSupportedChains"
	GasIbcTransfer               = 3_000_000 // Example value, adjust as needed
	GasGetSupportedChains        = 10_000    // Example value, adjust as needed
)

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

type Precompile struct {
	cmn.Precompile
	ibcKeeper ibckeeper.Keeper
}

func NewPrecompile(
	ibcKeeper ibckeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
) (*Precompile, error) {
	newABI, err := cmn.LoadABI(f, abiPath)
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newABI,
			AuthzKeeper:          authzKeeper,
			ApprovalExpiration:   cmn.DefaultExpirationDuration,
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		ibcKeeper: ibcKeeper,
	}
	p.SetAddress(p.GetContractAddress())
	return p, nil
}

func (p *Precompile) RequiredGas(input []byte) uint64 {
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]
	method, err := p.MethodById(methodID)
	if err != nil {
		return 0
	}

	switch method.Name {
	case TransferMethod:
		return GasIbcTransfer
	case "getSupportedChains":
		return GasGetSupportedChains
	default:
		return 0
	}
}

func (p *Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	// 1. Reject value sent to the contract (no payable)
	if value := contract.Value(); value.Sign() == 1 {
		return nil, fmt.Errorf("ibcTransfer precompile cannot receive funds: %s", contract.Value().String())
	}

	// 2. Setup context, stateDB, method, args, etc.
	ctx, stateDB, snapshot, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// 3. Defer gas error handling
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	// 4. Dispatch to the correct handler
	switch method.Name {
	case "ibcTransfer":
		if readOnly {
			// Transfers are not allowed in read-only mode
			return nil, fmt.Errorf("ibcTransfer cannot be called in read-only mode")
		}
		bz, err = p.TransferIBC(ctx, contract, stateDB, method, args)
	case "getSupportedChains":
		bz, err = p.GetSupportedChains(ctx, contract, stateDB, method, args)
	default:
		return nil, fmt.Errorf("unknown method: %s", method.Name)
	}

	// 5. Gas accounting
	cost := ctx.GasMeter().GasConsumed() - initialGas
	if !contract.UseGas(cost) {
		return nil, vm.ErrOutOfGas
	}

	// 6. Add journal entries (if needed)
	if err := p.AddJournalEntries(stateDB, snapshot); err != nil {
		return nil, err
	}

	return bz, err
}

func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case TransferMethod:
		return true
	default:
		return false
	}
}

func (p *Precompile) GetContractAddress() common.Address {
	address := common.HexToAddress(evmtypes.IBCTransferPrecompileAddress)
	return address
}
