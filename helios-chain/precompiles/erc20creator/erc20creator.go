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
	"strings"

	erc20keeper "helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "helios-core/helios-chain/precompiles/common"
	vm "helios-core/helios-chain/x/evm/core/vm"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

var _ vm.PrecompiledContract = &Precompile{}

//go:embed abi.json
var f embed.FS

const (
	abiPath = "abi.json"

	GasErc20Creator = 200_000

	// Basic validation constraints
	MaxNameLength   = 128
	MaxSymbolLength = 32
	MaxDecimals     = 18
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
		return nil, fmt.Errorf("failed to load ABI for ERC20 creator precompile: %w", err)
	}

	p := &Precompile{
		Precompile: cmn.Precompile{
			ABI:                  newABI,
			KvGasConfig:          storetypes.GasConfig{}, // no key/value store gas since we have a static cost
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

	// Set up call context and arguments
	ctx, _, _, method, _, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to run setup for ERC20 precompile: %w", err)
	}

	// Extract arguments in expected order: base (name), symbol, totalSupply, decimals
	base, okBase := args[0].(string)
	symbol, okSymbol := args[1].(string)
	supply, okSupply := args[2].(*big.Int)
	decimals, okDecimals := args[3].(uint8)

	if !okBase || !okSymbol || !okSupply || !okDecimals {
		return nil, fmt.Errorf("invalid argument types")
	}

	// Validate arguments
	if err := p.validateArguments(base, symbol, supply, decimals); err != nil {
		return nil, err
	}

	// Ensure the evm.Origin is not the zero address (common check for authenticity)
	if evm.Origin == (common.Address{}) {
		return nil, fmt.Errorf("origin address is zero address")
	}

	// Check if metadata already exists for this base denom
	_, found := p.bankKeeper.GetDenomMetaData(ctx, base)
	if found {
		return nil, errorsmod.Wrap(
			types.ErrInternalTokenPair,
			"denom metadata already registered, choose a unique base denomination",
		)
	}

	coinMetadata := banktypes.Metadata{
		Description: fmt.Sprintf("Token %s created with ERC20Creator precompile", base),
		Base:        base,
		Symbol:      symbol,
		Decimals:    uint32(decimals),
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    base,
				Exponent: 0,
			},
			{
				Denom:    base,
				Exponent: uint32(decimals),
			},
		},
	}

	// Deploy the ERC20 contract
	contractAddr, err := p.erc20Keeper.DeployERC20Contract(ctx, coinMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy ERC20 contract: %w", err)
	}

	// Mint tokens in the ERC20 contract
	if err := p.erc20Keeper.MintERC20Tokens(ctx, contractAddr, evm.Origin, supply); err != nil {
		return nil, fmt.Errorf("failed to mint ERC20 tokens: %w", err)
	}

	recipient := sdk.AccAddress(evm.Origin.Bytes())
	coins := sdk.NewCoins(sdk.NewCoin(base, sdkmath.NewIntFromBigInt(supply)))

	// Mint native coins to the module account
	if err := p.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to mint coins on-chain: %w", err)
	}

	// Transfer minted coins to the recipient
	if err := p.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, coins); err != nil {
		return nil, fmt.Errorf("failed to send minted coins to recipient: %w", err)
	}

	// Register the token pair for cross-chain usage
	tokenPair := types.NewTokenPair(contractAddr, base, types.OWNER_MODULE)
	p.erc20Keeper.SetToken(ctx, tokenPair)

	// Enable dynamic precompiles for the deployed ERC20 contract
	p.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())

	// TODO REMOVE AFTER
	asset := types.Asset{
		Denom:           base,
		ContractAddress: contractAddr.Hex(),
		ChainId:         "ethereum", // Exemple de chainId, à ajuster si nécessaire
		Decimals:        uint64(decimals),
		BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
		Metadata:        fmt.Sprintf("Token %s metadata", symbol),
	}

	// TODO : remove this !!
	if err := p.erc20Keeper.AddAssetToConsensusWhitelist(ctx, asset); err != nil {
		return nil, fmt.Errorf("failed to add ERC20 asset to whitelist: %w", err)
	}

	return method.Outputs.Pack(contractAddr)
}

func (Precompile) IsTransaction(_ *abi.Method) bool {
	return true
}

// validateArguments checks the token parameters for basic safety and correctness
func (p *Precompile) validateArguments(base, symbol string, supply *big.Int, decimals uint8) error {
	// Check non-empty fields
	if strings.TrimSpace(base) == "" {
		return fmt.Errorf("base denom cannot be empty")
	}
	if strings.TrimSpace(symbol) == "" {
		return fmt.Errorf("symbol cannot be empty")
	}

	// Check length constraints
	if len(base) > MaxNameLength {
		return fmt.Errorf("base denom length exceeds %d characters", MaxNameLength)
	}
	if len(symbol) > MaxSymbolLength {
		return fmt.Errorf("symbol length exceeds %d characters", MaxSymbolLength)
	}

	// Check supply validity
	if supply == nil || supply.Sign() <= 0 {
		return fmt.Errorf("total supply must be greater than zero")
	}

	// Check decimals range
	if decimals > MaxDecimals {
		return fmt.Errorf("decimals cannot exceed %d", MaxDecimals)
	}

	return nil
}
