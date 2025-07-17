package app

import (
	"context"
	"fmt"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "cosmossdk.io/x/auth/types"
	banktypes "cosmossdk.io/x/bank/types"
	stakingtypes "cosmossdk.io/x/staking/types"
	distrtypes "cosmossdk.io/x/distribution/types"
)

const (
	// UpgradeName defines the on-chain upgrade name for the v2.0 upgrade
	UpgradeNameV2 = "v2.0"
)

// CreateUpgradeHandlerV2 creates an upgrade handler for v2.0 upgrade
func CreateUpgradeHandlerV2(mm *module.Manager, configurator module.Configurator, app *HeliosApp) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		logger := app.Logger()
		logger.Info("🚀 Starting Helios Network v2.0 upgrade", "height", upgradeInfo.Height)

		// Step 1: Add new modules to version map
		logger.Info("📦 Adding new modules to version map...")
		fromVM = addNewModulesToVM(fromVM)

		// Step 2: Migrate delegation/undelegation logic
		logger.Info("🔗 Migrating delegation logic...")
		if err := migrateDelegationLogic(ctx, app); err != nil {
			return nil, errorsmod.Wrap(err, "failed to migrate delegation logic")
		}

		// Step 3: Initialize new features (fee discounts for long-term delegators)
		logger.Info("💎 Initializing fee discount system...")
		if err := initializeFeeDiscountSystem(ctx, app); err != nil {
			return nil, errorsmod.Wrap(err, "failed to initialize fee discount system")
		}

		// Step 4: Update network parameters
		logger.Info("⚙️ Updating network parameters...")
		if err := updateNetworkParameters(ctx, app); err != nil {
			return nil, errorsmod.Wrap(err, "failed to update network parameters")
		}

		// Step 5: Migrate validator metadata
		logger.Info("👥 Migrating validator metadata...")
		if err := migrateValidatorMetadata(ctx, app); err != nil {
			return nil, errorsmod.Wrap(err, "failed to migrate validator metadata")
		}

		// Step 6: Run module consensus version migrations
		logger.Info("🔄 Running module migrations...")
		versionMap, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, errorsmod.Wrap(err, "failed to run module migrations")
		}

		// Step 7: Post-migration validations
		logger.Info("✅ Running post-migration validations...")
		if err := validateUpgradeSuccess(ctx, app); err != nil {
			return nil, errorsmod.Wrap(err, "upgrade validation failed")
		}

		logger.Info("🎉 v2.0 upgrade completed successfully!", 
			"total_validators", len(app.StakingKeeper.GetAllValidators(ctx)),
			"upgrade_height", upgradeInfo.Height)

		return versionMap, nil
	}
}

// addNewModulesToVM adds new modules to the version map
func addNewModulesToVM(fromVM module.VersionMap) module.VersionMap {
	// In a real upgrade, you would add actual new modules here
	// Example: if you're adding a rewards module
	// fromVM[rewardstypes.ModuleName] = rewards.AppModule{}.ConsensusVersion()
	
	// For this POC, we simulate adding a hypothetical module
	// fromVM["feegrant"] = 1  // Enable fee grant module
	
	return fromVM
}

// migrateDelegationLogic implements enhanced delegation/undelegation logic
func migrateDelegationLogic(ctx context.Context, app *HeliosApp) error {
	// Get all validators
	validators, err := app.StakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get validators")
	}

	totalDelegationsProcessed := 0
	
	for _, validator := range validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			app.Logger().Error("invalid validator address", "validator", validator.OperatorAddress)
			continue
		}

		// Get all delegations to this validator
		delegations, err := app.StakingKeeper.GetValidatorDelegations(ctx, valAddr)
		if err != nil {
			app.Logger().Error("failed to get delegations", "validator", valAddr.String())
			continue
		}

		// Process each delegation
		for _, delegation := range delegations {
			// Add delegation timestamp tracking (new feature in v2.0)
			if err := addDelegationTimestamp(ctx, app, delegation); err != nil {
				app.Logger().Error("failed to add delegation timestamp", 
					"delegator", delegation.DelegatorAddress,
					"validator", delegation.ValidatorAddress,
					"error", err)
				continue
			}

			// Apply loyalty bonus for existing delegators
			if err := applyLoyaltyBonus(ctx, app, delegation); err != nil {
				app.Logger().Error("failed to apply loyalty bonus",
					"delegator", delegation.DelegatorAddress,
					"error", err)
				continue
			}

			totalDelegationsProcessed++
		}
	}

	app.Logger().Info("✅ Delegation logic migration completed", 
		"total_delegations_processed", totalDelegationsProcessed,
		"total_validators", len(validators))

	return nil
}

// addDelegationTimestamp adds timestamp tracking for delegations (simulated)
func addDelegationTimestamp(ctx context.Context, app *HeliosApp, delegation stakingtypes.Delegation) error {
	// In a real implementation, you would store this in a new KV store
	// For this POC, we just log the action
	
	blockTime := sdk.UnwrapSDKContext(ctx).BlockTime()
	app.Logger().Info("📅 Adding delegation timestamp",
		"delegator", delegation.DelegatorAddress,
		"validator", delegation.ValidatorAddress,
		"timestamp", blockTime.Unix(),
		"shares", delegation.Shares.String())
	
	// Here you would actually store the timestamp:
	// store := ctx.KVStore(app.keys[customstaking.StoreKey])
	// key := types.GetDelegationTimeKey(delAddr, valAddr)
	// store.Set(key, sdk.Uint64ToBigEndian(uint64(blockTime.Unix())))
	
	return nil
}

// applyLoyaltyBonus gives a bonus to existing long-term delegators
func applyLoyaltyBonus(ctx context.Context, app *HeliosApp, delegation stakingtypes.Delegation) error {
	// Simulate loyalty bonus calculation
	// In reality, you'd check how long they've been delegating
	
	minShares := math.LegacyNewDec(1000) // 1000 share minimum
	bonusRate := math.LegacyNewDecWithPrec(1, 2) // 1% bonus
	
	if delegation.Shares.GTE(minShares) {
		bonus := delegation.Shares.Mul(bonusRate)
		
		// In a real implementation, you would add this bonus
		// newShares := delegation.Shares.Add(bonus)
		// delegation.Shares = newShares
		// app.StakingKeeper.SetDelegation(ctx, delegation)
		
		app.Logger().Info("🎁 Loyalty bonus calculated",
			"delegator", delegation.DelegatorAddress,
			"validator", delegation.ValidatorAddress,
			"original_shares", delegation.Shares.String(),
			"bonus", bonus.String())
	}
	
	return nil
}

// initializeFeeDiscountSystem creates a fee discount system for loyal delegators
func initializeFeeDiscountSystem(ctx context.Context, app *HeliosApp) error {
	// Create a special account for fee discounts
	feeDiscountAddr := authtypes.NewModuleAddress("fee_discounts")
	
	// Allocate initial funds for fee discounts (1M HELIOS)
	initialFunds := sdk.NewCoins(sdk.NewCoin("uhelios", math.NewInt(1_000_000_000_000_000_000_000_000))) // 1M HELIOS
	
	if err := app.BankKeeper.MintCoins(ctx, authtypes.ModuleName, initialFunds); err != nil {
		return errorsmod.Wrap(err, "failed to mint coins for fee discount system")
	}
	
	if err := app.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.ModuleName, feeDiscountAddr, initialFunds); err != nil {
		return errorsmod.Wrap(err, "failed to send coins to fee discount account")
	}
	
	app.Logger().Info("✅ Fee discount system initialized",
		"account", feeDiscountAddr.String(),
		"initial_funds", initialFunds.String())
	
	return nil
}

// updateNetworkParameters updates various network parameters for v2.0
func updateNetworkParameters(ctx context.Context, app *HeliosApp) error {
	// Update staking parameters
	stakingParams, err := app.StakingKeeper.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get staking params")
	}
	
	// Reduce unbonding time from 21 days to 14 days (more user-friendly)
	stakingParams.UnbondingTime = 14 * 24 * 60 * 60 * 1000000000 // 14 days in nanoseconds
	
	// Increase max validators from current to 125
	stakingParams.MaxValidators = 125
	
	// Set minimum commission to 2%
	stakingParams.MinCommissionRate = math.LegacyNewDecWithPrec(2, 2) // 2%
	
	if err := app.StakingKeeper.SetParams(ctx, stakingParams); err != nil {
		return errorsmod.Wrap(err, "failed to update staking params")
	}
	
	// Update distribution parameters
	distrParams, err := app.DistrKeeper.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get distribution params")
	}
	
	// Reduce community tax from 2% to 1%
	distrParams.CommunityTax = math.LegacyNewDecWithPrec(1, 2) // 1%
	
	if err := app.DistrKeeper.Params.Set(ctx, distrParams); err != nil {
		return errorsmod.Wrap(err, "failed to update distribution params")
	}
	
	app.Logger().Info("✅ Network parameters updated",
		"unbonding_time", "14 days",
		"max_validators", stakingParams.MaxValidators,
		"min_commission", "2%",
		"community_tax", "1%")
	
	return nil
}

// migrateValidatorMetadata migrates validator metadata to new format
func migrateValidatorMetadata(ctx context.Context, app *HeliosApp) error {
	validators, err := app.StakingKeeper.GetAllValidators(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to get validators")
	}
	
	migratedCount := 0
	
	for _, validator := range validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			continue
		}
		
		// Cap commission rates above 10% to 10%
		if validator.Commission.Rate.GT(math.LegacyNewDecWithPrec(10, 2)) {
			validator.Commission.Rate = math.LegacyNewDecWithPrec(10, 2) // 10%
			validator.Commission.MaxRate = math.LegacyNewDecWithPrec(10, 2) // 10%
			
			if err := app.StakingKeeper.SetValidator(ctx, validator); err != nil {
				app.Logger().Error("failed to update validator commission",
					"validator", valAddr.String(),
					"error", err)
				continue
			}
			
			app.Logger().Info("📊 Validator commission capped",
				"validator", valAddr.String(),
				"new_rate", "10%")
			
			migratedCount++
		}
	}
	
	app.Logger().Info("✅ Validator metadata migration completed",
		"validators_migrated", migratedCount,
		"total_validators", len(validators))
	
	return nil
}

// validateUpgradeSuccess performs post-upgrade validations
func validateUpgradeSuccess(ctx context.Context, app *HeliosApp) error {
	// Validate staking parameters
	stakingParams, err := app.StakingKeeper.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to validate staking params")
	}
	
	if stakingParams.MaxValidators != 125 {
		return fmt.Errorf("expected max validators to be 125, got %d", stakingParams.MaxValidators)
	}
	
	// Validate distribution parameters
	distrParams, err := app.DistrKeeper.Params.Get(ctx)
	if err != nil {
		return errorsmod.Wrap(err, "failed to validate distribution params")
	}
	
	expectedTax := math.LegacyNewDecWithPrec(1, 2) // 1%
	if !distrParams.CommunityTax.Equal(expectedTax) {
		return fmt.Errorf("expected community tax to be 1%%, got %s", distrParams.CommunityTax.String())
	}
	
	// Validate fee discount account exists
	feeDiscountAddr := authtypes.NewModuleAddress("fee_discounts")
	balance := app.BankKeeper.GetBalance(ctx, feeDiscountAddr, "uhelios")
	
	expectedBalance := math.NewInt(1_000_000_000_000_000_000_000_000) // 1M HELIOS
	if !balance.Amount.Equal(expectedBalance) {
		return fmt.Errorf("expected fee discount balance to be %s, got %s", 
			expectedBalance.String(), balance.Amount.String())
	}
	
	app.Logger().Info("✅ All upgrade validations passed")
	return nil
}

// GetStoreUpgradesV2 returns the store upgrades for v2.0
func GetStoreUpgradesV2() *storetypes.StoreUpgrades {
	return &storetypes.StoreUpgrades{
		Added: []string{
			// Add new module stores here when adding actual modules
			// "feegrant", // Example: if enabling fee grant module
		},
		Renamed: []storetypes.StoreRename{
			// No store renames needed for v2.0
		},
		Deleted: []string{
			// No stores to delete for v2.0
		},
	}
} 