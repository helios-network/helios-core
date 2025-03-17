package keeper

import (
	"helios-core/helios-chain/x/chronos/types"
)

var _ types.QueryServer = Keeper{}
