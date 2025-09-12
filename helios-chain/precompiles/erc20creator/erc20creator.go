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

	// Extract arguments in expected order: name (name), symbol, totalSupply, decimals
	name, okName := args[0].(string)
	symbol, okSymbol := args[1].(string)
	denom, okDenom := args[2].(string)
	supply, okSupply := args[3].(*big.Int)
	decimals, okDecimals := args[4].(uint8)
	logoBase64, okLogo := args[5].(string)

	if !okName || !okSymbol || !okSupply || !okDecimals || !okDenom || !okLogo {
		return nil, fmt.Errorf("invalid argument types")
	}

	logoHash := ""

	if logoBase64 != "" {
		logoHash, err = p.logosKeeper.StoreLogo(ctx, logoBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to store logo: %w", err)
		}
	}

	// Validate arguments
	if err := p.validateArguments(name, symbol, denom, supply, decimals); err != nil {
		return nil, err
	}

	// Ensure the evm.Origin is not the zero address (common check for authenticity)
	if evm.Origin == (common.Address{}) {
		return nil, fmt.Errorf("origin address is zero address")
	}

	found := true
	// Check if metadata already exists for this base denom permit to create ~100 000 same denoms maximum
	_, found = p.bankKeeper.GetDenomMetaData(ctx, denom)
	if found {
		return nil, errorsmod.Wrap(
			types.ErrInternalTokenPair,
			"denom metadata already registered, choose a unique base denomination",
		)
	}

	coinMetadata := banktypes.Metadata{
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
	}

	// validate metadata
	if err := coinMetadata.Validate(); err != nil {
		return nil, fmt.Errorf("failed to deploy ERC20 contract: %w", err)
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

	recipient := sdktypes.AccAddress(evm.Origin.Bytes())
	coins := sdktypes.NewCoins(sdktypes.NewCoin(denom, sdkmath.NewIntFromBigInt(supply)))

	// Mint native coins to the module account
	if err := p.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return nil, fmt.Errorf("failed to mint coins on-chain: %w", err)
	}

	// Transfer minted coins to the recipient
	if err := p.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, coins); err != nil {
		return nil, fmt.Errorf("failed to send minted coins to recipient: %w", err)
	}

	// Register the token pair for cross-chain usage
	tokenPair := types.NewTokenPair(contractAddr, denom, types.OWNER_MODULE)
	p.erc20Keeper.SetToken(ctx, tokenPair)

	// Enable dynamic precompiles for the deployed ERC20 contract
	if err = p.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract()); err != nil {
		return nil, fmt.Errorf("failed to EnableDynamicPrecompiles: %w", err)
	}

	// ctx.EventManager().EmitEvent(
	// 	sdktypes.NewEvent(
	// 		"erc20_created", // todo: add to sdktypes
	// 		sdktypes.NewAttribute("denom", denom),
	// 		sdktypes.NewAttribute("symbol", symbol),
	// 		sdktypes.NewAttribute("contract_address", contractAddr.String()),
	// 		sdktypes.NewAttribute("decimals", fmt.Sprintf("%d", decimals)),
	// 		sdktypes.NewAttribute("supply", supply.String()),
	// 	),
	// )

	// TODO REMOVE AFTER
	// asset := types.Asset{
	// 	Denom:           denom,
	// 	ContractAddress: contractAddr.Hex(),
	// 	ChainId:         utils.MainnetChainID, // Exemple de chainId, à ajuster si nécessaire
	// 	ChainName:       "Helios",
	// 	Decimals:        uint64(decimals),
	// 	BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
	// 	Symbol:          symbol,
	// }

	// TODO : remove this !!
	// if err := p.erc20Keeper.AddAssetToConsensusWhitelist(ctx, asset); err != nil {
	// 	return nil, fmt.Errorf("failed to add ERC20 asset to whitelist: %w", err)
	// }

	// write the log to the stateDB
	stateDB.AddLog(&ethtypes.Log{
		Address: p.Address(), // ou une autre adresse
		Topics: []common.Hash{
			crypto.Keccak256Hash([]byte("createErc20(address)")),
		},
		Data: contractAddr.Bytes(), // ou encode en abi si besoin
	})

	return method.Outputs.Pack(contractAddr)
}

func (Precompile) IsTransaction(_ *abi.Method) bool {
	return true
}

// validateArguments checks the token parameters for basic safety and correctness
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
