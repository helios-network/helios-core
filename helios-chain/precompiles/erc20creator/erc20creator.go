// Package erc20creator provides a precompiled contract for creating ERC20 tokens.
//
// This precompile allows for the creation of ERC20 tokens directly through a precompiled contract.
// To use this precompile, it must be:
//
// 1. Loaded in the static_precompiles.go file
// 2. Have its contract address defined in:
//   - x/evm/types/precompiles.go
//   - x/evm/types/params.go
//
// Note: Each precompile contract address must be unique and assigned in sequential order
// to maintain consistency across the system.
//
// Current address: 0x0000000000000000000000000000000000000806
//
// The precompile exposes a single method:
//   - createErc20(string name, string symbol, uint256 totalSupply, uint8 decimals) returns (address)
package erc20creator

import (
	"embed"
	"fmt"
	"math/big"

	erc20keeper "helios-core/helios-chain/x/erc20/keeper"

	errorsmod "cosmossdk.io/errors"

	"helios-core/helios-chain/x/erc20/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	cmn "helios-core/helios-chain/precompiles/common"
	vm "helios-core/helios-chain/x/evm/core/vm"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var _ vm.PrecompiledContract = &Precompile{}

//go:embed abi.json
var f embed.FS

const (
	abiPath = "abi.json"

	GasErc20Creator = 200_000
)

type Precompile struct {
	cmn.Precompile
	abi.ABI
	erc20Keeper erc20keeper.Keeper
	bankKeeper  bankkeeper.Keeper
}

func NewPrecompile(erc20Keeper erc20keeper.Keeper, bankKeeper bankkeeper.Keeper) (*Precompile, error) {

	newABI, err := cmn.LoadABI(f, abiPath)
	if err != nil {
		return nil, err
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newABI,
			KvGasConfig:          storetypes.GasConfig{}, // no need because we have static gas
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		erc20Keeper: erc20Keeper,
		bankKeeper:  bankKeeper,
	}

	p.SetAddress(common.HexToAddress(evmtypes.Erc20CreatorPrecompileAddress))

	return p, nil

}

func (p Precompile) RequiredGas(_ []byte) uint64 {
	return GasErc20Creator
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {

	ctx, _, _, method, _, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	// Check if metadata already exists
	_, found := p.bankKeeper.GetDenomMetaData(ctx, args[0].(string))
	if found {
		return nil, errorsmod.Wrap(
			types.ErrInternalTokenPair, "denom metadata already registered",
		)
	}

	base := args[0].(string)
	symbol := args[1].(string)
	decimals := uint32(args[3].(uint8))
	supply := args[2].(*big.Int)

	coinMetadata := banktypes.Metadata{
		Description: fmt.Sprintf("Token %s created with ERC20Creator", base),
		Base:        base,
		Symbol:      symbol,
		Decimals:    decimals,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    base,
				Exponent: 0,
			},
			{
				Denom:    base,
				Exponent: decimals,
			},
		},
	}

	contractAddr, err := p.erc20Keeper.DeployERC20Contract(ctx, coinMetadata)
	if err != nil {
		return nil, err
	}

	coins := sdk.NewCoins(sdk.NewCoin(base, sdkmath.NewIntFromBigInt(supply)))

	err = p.bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	if err != nil {
		return nil, fmt.Errorf("failed to mint coins: %w", err)
	}

	err = p.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, sdk.AccAddress(evm.Origin.Bytes()), coins)

	if err != nil {
		return nil, fmt.Errorf("failed to send coins to owner: %w", err)
	}

	tokenPair := types.NewTokenPair(contractAddr, base, types.OWNER_MODULE)
	p.erc20Keeper.SetTokenPair(ctx, tokenPair)

	fmt.Println("addr owner", evm.Origin.String())
	fmt.Println("addr contract", contractAddr)

	return method.Outputs.Pack(contractAddr)
}

func (Precompile) IsTransaction(_ *abi.Method) bool {
	return true
}
