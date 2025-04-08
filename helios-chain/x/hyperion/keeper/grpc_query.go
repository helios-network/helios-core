package keeper

import (
	"helios-core/helios-chain/x/hyperion/types"
)

var _ types.QueryServer = &Keeper{}
