package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
)

type BankKeeper interface {
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

type MintKeeper = mintkeeper.Keeper

type StakingKeeper interface {
	BondDenom(ctx context.Context) (string, error)
}
