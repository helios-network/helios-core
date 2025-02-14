package app

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// nolint:all
const (
	upgradeName = "v1.13.2"
)

func (app *HeliosApp) registerUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeName,
		func(ctx context.Context, upgradeInfo upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
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

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}
	// nolint:all
	if upgradeInfo.Name == upgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		// add any store upgrades here
		storeUpgrades := storetypes.StoreUpgrades{
			Added:   nil,
			Renamed: nil,
			Deleted: nil,
		}

		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}
