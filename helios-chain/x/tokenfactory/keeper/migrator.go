package keeper

import (
	"helios-core/helios-chain/x/tokenfactory/exported"
	v2 "helios-core/helios-chain/x/tokenfactory/migrations/v2"

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

func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return v2.Migrate(
		ctx,
		ctx.KVStore(m.keeper.storeKey),
		m.subspace,
	)
}
