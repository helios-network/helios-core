package gov

import (
	"embed"
	"fmt"

	erc20keeper "helios-core/helios-chain/x/erc20/keeper"
	hyperionkeeper "helios-core/helios-chain/x/hyperion/keeper"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/evm/core/vm"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

var _ vm.PrecompiledContract = &Precompile{}

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

// Precompile defines the precompiled contract for gov.
type Precompile struct {
	cmn.Precompile
	govKeeper      govkeeper.Keeper
	bankKeeper     bankkeeper.Keeper
	erc20Keeper    erc20keeper.Keeper
	hyperionKeeper hyperionkeeper.Keeper
}

// LoadABI loads the gov ABI from the embedded abi.json file
// for the gov precompile.
func LoadABI() (abi.ABI, error) {
	return cmn.LoadABI(f, "abi.json")
}

// NewPrecompile creates a new gov Precompile instance as a
// PrecompiledContract interface.
func NewPrecompile(
	govKeeper govkeeper.Keeper,
	authzKeeper authzkeeper.Keeper,
	bankKeeper bankkeeper.Keeper,
	erc20Keeper erc20keeper.Keeper,
	hyperionKeeper hyperionkeeper.Keeper,
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
		govKeeper:      govKeeper,
		bankKeeper:     bankKeeper,
		erc20Keeper:    erc20Keeper,
		hyperionKeeper: hyperionKeeper,
	}

	// SetAddress defines the address of the gov precompiled contract.
	p.SetAddress(common.HexToAddress(evmtypes.GovPrecompileAddress))

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

// Run executes the precompiled contract gov methods defined in the ABI.
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
	case VoteMethod:
		bz, err = p.Vote(ctx, evm.Origin, contract, stateDB, method, args)
	case VoteWeightedMethod:
		bz, err = p.VoteWeighted(ctx, evm.Origin, contract, stateDB, method, args)
	// gov queries
	case GetVoteMethod:
		bz, err = p.GetVote(ctx, method, contract, args)
	case GetVotesMethod:
		bz, err = p.GetVotes(ctx, method, contract, args)
	case GetDepositMethod:
		bz, err = p.GetDeposit(ctx, method, contract, args)
	case GetDepositsMethod:
		bz, err = p.GetDeposits(ctx, method, contract, args)
	case GetTallyResultMethod:
		bz, err = p.GetTallyResult(ctx, method, contract, args)
	case AddNewAssetProposalMethod:
		bz, err = p.AddNewAssetProposal(
			evm.Origin,
			p.govKeeper,
			ctx, method, contract, args)
	case UpdateAssetProposalMethod:
		bz, err = p.UpdateAssetProposal(
			evm.Origin,
			p.govKeeper,
			ctx, method, contract, args)
	case RemoveAssetProposalMethod:
		bz, err = p.RemoveAssetProposal(
			evm.Origin,
			p.govKeeper,
			ctx, method, contract, args)
	case UpdateBlockParamsProposalMethod:
		bz, err = p.UpdateBlockParamsProposal(
			ctx,
			evm.Origin,
			p.govKeeper,
			contract,
			method,
			args)
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

// IsTransaction checks if the given method name corresponds to a transaction or query.
//
// Available gov transactions are:
//   - Vote
//   - VoteWeighted
func (Precompile) IsTransaction(method *abi.Method) bool {
	switch method.Name {
	case VoteMethod, VoteWeightedMethod, AddNewAssetProposalMethod, UpdateAssetProposalMethod, RemoveAssetProposalMethod:
		return true
	default:
		return false
	}
}

// Logger returns a precompile-specific logger.
func (p Precompile) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("evm extension", "gov")
}
