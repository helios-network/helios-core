package keeper

import (
	"helios-core/helios-chain/x/hyperion/exported"
	v2 "helios-core/helios-chain/x/hyperion/migrations/v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Migrator struct {
	keeper   Keeper
	subspace exported.Subspace
}

func NewMigrator(k Keeper, ss exported.Subspace) Migrator {
	return Migrator{
		keeper:   k,
		subspace: ss,
	}
}

// Migrate1to2 migrates hyperion's consensus version from 1 to 2. Specifically, it migrates
// Params kept in x/params directly to hyperion's module state
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.Migrate(
		ctx,
		ctx.KVStore(m.keeper.storeKey),
		m.subspace,
		m.keeper.cdc,
	)
}
