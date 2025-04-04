// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:LGPL-3.0-only

package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"helios-core/helios-chain/x/erc20/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

var _ evmtypes.EvmHooks = Hooks{}

// Hooks wrapper struct for fees keeper
type Hooks struct {
	k Keeper
}

// Hooks return the wrapper hooks struct for the Keeper
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

func (h Hooks) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	return h.k.PostTxProcessing(ctx, msg, receipt)
}

func (k *Keeper) PostTxProcessing(
	ctx sdk.Context,
	msg core.Message,
	receipt *ethtypes.Receipt,
) error {
	// If it's a contract deployment, receipt.ContractAddress will contain the contract address
	if receipt != nil && receipt.ContractAddress != (common.Address{}) {
		contractAddress := receipt.ContractAddress
		ctx.Logger().Info("Detected contract deployment", "address", contractAddress.String())

		// Use a CacheContext to avoid altering the state if something fails
		cacheCtx, _ := ctx.CacheContext()

		// Check if it's an ERC20 using QueryERC20
		metadata, err := k.QueryERC20(cacheCtx, contractAddress)
		if err != nil {
			// If the error is due to a non-ERC20 contract, simply return
			// without error to allow other hooks to execute
			ctx.Logger().Debug("Contract is not an ERC20", "address", contractAddress.String(), "error", err.Error())
			return nil
		}

		// If we get here, the contract responded correctly to ERC20 queries
		// Check essential ERC20 fields
		if metadata.Name == "" || metadata.Symbol == "" {
			ctx.Logger().Debug("Contract is missing ERC20 metadata", "address", contractAddress.String())
			return nil
		}

		// Check if this ERC20 contract is already registered
		if k.IsERC20Registered(ctx, contractAddress) {
			ctx.Logger().Info("ERC20 contract already registered", "address", contractAddress.String())
			// Ensure precompile is enabled even if already registered
			if err := k.EnableDynamicPrecompiles(ctx, contractAddress); err != nil {
				ctx.Logger().Error("Failed to enable precompile for registered ERC20",
					"address", contractAddress.String(),
					"error", err.Error())
			}
			return nil
		}

		ctx.Logger().Info("Detected new ERC20 token",
			"address", contractAddress.String(),
			"name", metadata.Name,
			"symbol", metadata.Symbol)

		// Get the contract deployer address (equivalent to evm.Origin)
		// In the EVM hook, msg.From() contains the deployer's address
		deployer := msg.From()
		if msg.From() == (common.Address{}) {
			// Fallback to module address if deployer is not available
			deployer = types.ModuleAddress
		}

		// Convert ethereum address to cosmos address
		recipient := sdk.AccAddress(deployer.Bytes())

		// Create token metadata for the native coin equivalent
		base := metadata.Symbol // Use symbol as the denom
		coinMetadata := banktypes.Metadata{
			Description: fmt.Sprintf("Token %s detected by ERC20 hook", metadata.Name),
			Base:        base,
			Name:        metadata.Name,
			Symbol:      metadata.Symbol,
			Decimals:    uint32(metadata.Decimals),
			Display:     base,
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    base,
					Exponent: 0,
				},
				{
					Denom:    base,
					Exponent: uint32(metadata.Decimals),
				},
			},
		}

		// Set the denom metadata in the bank keeper
		k.bankKeeper.SetDenomMetaData(ctx, coinMetadata)

		supply, err := k.TotalSupply(ctx, contractAddress)
		coins := sdk.NewCoins(sdk.NewCoin(base, sdkmath.NewIntFromBigInt(supply)))

		// Register the token pair for cross-chain usage
		pair := types.NewTokenPair(contractAddress, base, types.OWNER_EXTERNAL)
		k.SetToken(ctx, pair)

		// Emit an event for the ERC20 detection
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"erc20_detected",
				sdk.NewAttribute("contract_address", contractAddress.String()),
				sdk.NewAttribute("name", metadata.Name),
				sdk.NewAttribute("symbol", metadata.Symbol),
				sdk.NewAttribute("decimals", fmt.Sprintf("%d", metadata.Decimals)),
				sdk.NewAttribute("denom", base),
			),
		)

		if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
			ctx.Logger().Error("Failed to mint native coins",
				"address", contractAddress.String(),
				"error", err.Error())
		} else {
			// If minting succeeds, send tokens to the deployer
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, recipient, coins); err != nil {
				ctx.Logger().Error("Failed to send native coins to deployer",
					"address", contractAddress.String(),
					"deployer", deployer.String(),
					"error", err.Error())
			} else {
				ctx.Logger().Info("Successfully minted and sent native tokens",
					"address", contractAddress.String(),
					"deployer", deployer.String(),
					"amount", supply.String(),
					"denom", base)
			}
		}

		// Important: Enable dynamic precompiles for the detected ERC20
		if err := k.EnableDynamicPrecompiles(ctx, pair.GetERC20Contract()); err != nil {
			ctx.Logger().Error("Failed to enable precompile for ERC20",
				"address", contractAddress.String(),
				"error", err.Error())
			// Continue even if this fails
		}

		// Verify that the precompile was properly initialized
		precompile, found, err := k.GetERC20PrecompileInstance(ctx, contractAddress)
		if err != nil || !found {
			ctx.Logger().Error("Failed to verify ERC20 precompile initialization",
				"address", contractAddress.String(),
				"found", found,
				"error", err)
		} else {
			ctx.Logger().Info("Successfully initialized ERC20 precompile",
				"address", contractAddress.String(),
				"precompile", precompile)
		}
	}

	// For normal transactions (non-deployment)
	contract := msg.To()
	if contract == nil || msg.GasPrice().Sign() <= 0 {
		return nil
	}
	return nil
}
