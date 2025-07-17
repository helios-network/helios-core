package app

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// Upgrade names
const (
	UpgradeNameV1132 = "v1.13.2" // Current production version
	UpgradeNameV2    = "v2.0"    // Next planned upgrade
)

func (app *HeliosApp) registerUpgradeHandlers() {
	// Register current production upgrade handler (v1.13.2)
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV1132,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			// Simple parameter update for current version
			mintParams, err := app.MintKeeper.Params.Get(ctx)
			if err != nil {
				return nil, err
			}
			mintParams.BlocksPerYear = 42_048_000 // from 35040000 to 42048000
			err = app.MintKeeper.Params.Set(ctx, mintParams)
			if err != nil {
				return nil, err
			}

			return app.mm.RunMigrations(ctx, app.configurator, fromVM)
		},
	)

	// Register v2.0 upgrade handler (next planned upgrade)
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV2, 
		CreateUpgradeHandlerV2(app.mm, app.configurator, app))

	// Apply store upgrades based on upgrade plan
	app.applyStoreUpgrades()
}

// applyStoreUpgrades configures store upgrades based on the detected upgrade plan
func (app *HeliosApp) applyStoreUpgrades() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	var storeUpgrades *storetypes.StoreUpgrades

	// Apply appropriate store upgrades based on upgrade name
	switch upgradeInfo.Name {
	case UpgradeNameV1132:
		// No store upgrades needed for v1.13.2
		storeUpgrades = &storetypes.StoreUpgrades{
			Added:   nil,
			Renamed: nil,
			Deleted: nil,
		}
	case UpgradeNameV2:
		// Store upgrades for v2.0
		storeUpgrades = GetStoreUpgradesV2()
	default:
		// No store upgrades for unknown versions
		app.Logger().Info("No store upgrades defined for upgrade", "name", upgradeInfo.Name)
		return
	}

	// Configure store loader
	app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, storeUpgrades))
	app.Logger().Info("Store upgrades configured", "upgrade", upgradeInfo.Name, "height", upgradeInfo.Height)
}
