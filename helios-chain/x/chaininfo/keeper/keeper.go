package keeper

import (
	"helios-core/helios-chain/x/chaininfo/types"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	bankKeeper    types.BankKeeper
	mintKeeper    types.MintKeeper
	stakingKeeper types.StakingKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	bankKeeper types.BankKeeper,
	mintKeeper types.MintKeeper,
	stakingKeeper types.StakingKeeper,
) Keeper {
	return Keeper{
		cdc:           cdc,
		bankKeeper:    bankKeeper,
		mintKeeper:    mintKeeper,
		stakingKeeper: stakingKeeper,
	}
}

func (k Keeper) Logger(ctx log.Logger) log.Logger {
	return ctx.With("module", "x/"+types.ModuleName)
}
