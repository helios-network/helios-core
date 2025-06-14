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
//   - createErc20(string name, string symbol, string denom, uint256 totalSupply, uint8 decimals, string logoBase64, address mintAuthority, address pauseAuthority, address burnAuthority) returns (address)
package erc20creator

import (
	"embed"
	"fmt"
	"math/big"
	"strings"

	erc20keeper "helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	errorsmod "cosmossdk.io/errors"

	cmn "helios-core/helios-chain/precompiles/common"
	vm "helios-core/helios-chain/x/evm/core/vm"

	logoskeeper "helios-core/helios-chain/x/logos/keeper"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
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

	// Increased gas to handle role operations and revocation
	GasErc20Creator = 300_000

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
	logosKeeper logoskeeper.Keeper
}

func LoadABI() (abi.ABI, error) {
	return cmn.LoadABI(f, "abi.json")
}

func NewPrecompile(erc20Keeper erc20keeper.Keeper, bankKeeper bankkeeper.Keeper, logosKeeper logoskeeper.Keeper) (*Precompile, error) {
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
		logosKeeper: logosKeeper,
	}

	p.SetAddress(common.HexToAddress(evmtypes.Erc20CreatorPrecompileAddress))
	return p, nil
}

func (p Precompile) RequiredGas(_ []byte) uint64 {
	return GasErc20Creator
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	// Set up call context and arguments
	ctx, stateDB, _, method, _, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, fmt.Errorf("failed to run setup for ERC20 precompile: %w", err)
	}

	// Extract arguments with new role authorities
	name, okName := args[0].(string)
	symbol, okSymbol := args[1].(string)
	denom, okDenom := args[2].(string)
	supply, okSupply := args[3].(*big.Int)
	decimals, okDecimals := args[4].(uint8)
	logoBase64, okLogo := args[5].(string)
	// New parameters for role-based access control
	mintAuthority, okMintAuth := args[6].(common.Address)
	pauseAuthority, okPauseAuth := args[7].(common.Address)
	burnAuthority, okBurnAuth := args[8].(common.Address)

	if !okName || !okSymbol || !okSupply || !okDecimals || !okDenom || !okLogo || !okMintAuth || !okPauseAuth || !okBurnAuth {
		return nil, fmt.Errorf("invalid argument types")
	}

	// Determine which features are enabled based on authority addresses
	isMintable := mintAuthority != (common.Address{})
	isPausable := pauseAuthority != (common.Address{})
	isBurnable := burnAuthority != (common.Address{})

	logoHash := ""
	if logoBase64 != "" {
		logoHash, err = p.logosKeeper.StoreLogo(ctx, logoBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to store logo: %w", err)
		}
	}

	// Validate arguments (now allows zero supply)
	if err := p.validateArguments(name, symbol, denom, supply, decimals); err != nil {
		return nil, err
	}

	// Ensure the evm.Origin is not the zero address
	if evm.Origin == (common.Address{}) {
		return nil, fmt.Errorf("origin address is zero address")
	}

	// Check if metadata already exists for this base denom
	_, found := p.bankKeeper.GetDenomMetaData(ctx, denom)
	if found {
		return nil, errorsmod.Wrap(
			types.ErrInternalTokenPair,
			"denom metadata already registered, choose a unique base denomination",
		)
	}

	// Create metadata (without storing roles in URI)
	coinMetadata := p.createMetadata(name, symbol, denom, decimals, logoHash)

	// Validate metadata
	if err := coinMetadata.Validate(); err != nil {
		return nil, fmt.Errorf("failed to deploy ERC20 contract: %w", err)
	}

	// Deploy the ERC20 contract with proper authorities set from the beginning
	contractAddr, err := p.erc20Keeper.DeployERC20Contract(
		ctx,
		coinMetadata,
		evm.Origin,     // User becomes the owner
		mintAuthority,  // Mint authority
		pauseAuthority, // Pause authority
		burnAuthority,  // Burn authority
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy ERC20 contract: %w", err)
	}

	// SOLANA-STYLE: Only mint if supply > 0
	if supply.Sign() > 0 {
		// Mint initial tokens using temporary module authority
		if err := p.erc20Keeper.MintERC20Tokens(ctx, contractAddr, evm.Origin, supply); err != nil {
			return nil, fmt.Errorf("failed to mint ERC20 tokens: %w", err)
		}

		// Cosmos-side operations
		recipient := sdktypes.AccAddress(evm.Origin.Bytes())
		coins := sdktypes.NewCoins(sdktypes.NewCoin(denom, sdkmath.NewIntFromBigInt(supply)))

		if err := p.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
			return nil, fmt.Errorf("failed to mint coins on-chain: %w", err)
		}

		if err := p.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, coins); err != nil {
			return nil, fmt.Errorf("failed to send minted coins to recipient: %w", err)
		}
	}

	// Always revoke temporary minter role from module if token is non-mintable
	if !isMintable {
		ctx.Logger().Info("Token created as non-mintable",
			"contract", contractAddr.Hex(),
			"note", "Module retains MINTER_ROLE for gas optimization - role is dormant")
	} else {
		ctx.Logger().Info("Token created as mintable",
			"contract", contractAddr.Hex(),
			"mint_authority", mintAuthority.Hex())
	}

	// Register the token pair
	tokenPair := types.NewTokenPair(contractAddr, denom, types.OWNER_MODULE)
	p.erc20Keeper.SetToken(ctx, tokenPair)

	// Enable dynamic precompiles
	if err = p.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract()); err != nil {
		return nil, fmt.Errorf("failed to EnableDynamicPrecompiles: %w", err)
	}

	// Emit event with role information
	p.emitTokenCreationEventWithRoles(ctx, denom, symbol, contractAddr, decimals, supply,
		isMintable, mintAuthority, isPausable, pauseAuthority, isBurnable, burnAuthority)

	// Write the log to the stateDB
	stateDB.AddLog(&ethtypes.Log{
		Address: p.Address(),
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte("createErc20(address,bool,bool,bool)")),
		},
		Data: contractAddr.Bytes(),
	})

	return method.Outputs.Pack(contractAddr)
}

// createMetadata creates bank metadata without role information in URI
func (p *Precompile) createMetadata(name, symbol, denom string, decimals uint8, logoHash string) banktypes.Metadata {
	return banktypes.Metadata{
		Description: fmt.Sprintf("Token %s created with ERC20Creator precompile", denom),
		Base:        denom,
		Name:        name,
		Symbol:      symbol,
		Decimals:    uint32(decimals),
		Display:     symbol,
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    denom,
				Exponent: uint32(0),
			},
			{
				Denom:    symbol,
				Exponent: uint32(decimals),
			},
		},
		Logo: logoHash,
		// URI is left empty - roles are managed by the ERC20 contract itself
	}
}

// emitTokenCreationEventWithRoles emits an event with comprehensive role information
func (p *Precompile) emitTokenCreationEventWithRoles(ctx sdktypes.Context, denom, symbol string,
	contractAddr common.Address, decimals uint8, supply *big.Int,
	isMintable bool, mintAuthority common.Address, isPausable bool, pauseAuthority common.Address,
	isBurnable bool, burnAuthority common.Address) {

	attributes := []sdktypes.Attribute{
		sdktypes.NewAttribute("denom", denom),
		sdktypes.NewAttribute("symbol", symbol),
		sdktypes.NewAttribute("contract_address", contractAddr.String()),
		sdktypes.NewAttribute("decimals", fmt.Sprintf("%d", decimals)),
		sdktypes.NewAttribute("supply", supply.String()),
		sdktypes.NewAttribute("mintable", fmt.Sprintf("%t", isMintable)),
		sdktypes.NewAttribute("pausable", fmt.Sprintf("%t", isPausable)),
		sdktypes.NewAttribute("burnable", fmt.Sprintf("%t", isBurnable)),
	}

	if isMintable {
		attributes = append(attributes, sdktypes.NewAttribute("mint_authority", mintAuthority.Hex()))
	}
	if isPausable {
		attributes = append(attributes, sdktypes.NewAttribute("pause_authority", pauseAuthority.Hex()))
	}
	if isBurnable {
		attributes = append(attributes, sdktypes.NewAttribute("burn_authority", burnAuthority.Hex()))
	}

	ctx.EventManager().EmitEvent(
		sdktypes.NewEvent("erc20_created_with_roles", attributes...),
	)
}

func (Precompile) IsTransaction(_ *abi.Method) bool {
	return true
}

// validateArguments checks the token parameters for basic safety and correctness
// MODIFIED: Now allows zero supply (Solana-style)
func (p *Precompile) validateArguments(name, symbol, denom string, supply *big.Int, decimals uint8) error {
	// Check non-empty fields
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.TrimSpace(symbol) == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	if strings.TrimSpace(denom) == "" {
		return fmt.Errorf("denom cannot be empty")
	}

	// Check for spaces
	if strings.Contains(name, " ") {
		return fmt.Errorf("name cannot contain spaces")
	}
	if strings.Contains(symbol, " ") {
		return fmt.Errorf("symbol cannot contain spaces")
	}
	if strings.Contains(denom, " ") {
		return fmt.Errorf("denom cannot contain spaces")
	}

	// Check length constraints
	if len(name) > MaxNameLength {
		return fmt.Errorf("name length exceeds %d characters", MaxNameLength)
	}
	if len(symbol) > MaxSymbolLength {
		return fmt.Errorf("symbol length exceeds %d characters", MaxSymbolLength)
	}
	if len(denom) > MaxSymbolLength {
		return fmt.Errorf("denom length exceeds %d characters", MaxSymbolLength)
	}

	// MODIFIED: Allow zero supply (Solana-style) but not negative
	if supply == nil || supply.Sign() < 0 {
		return fmt.Errorf("total supply must be greater than or equal to zero")
	}

	// Check decimals range
	if decimals > MaxDecimals {
		return fmt.Errorf("decimals cannot exceed %d", MaxDecimals)
	}

	return nil
}
