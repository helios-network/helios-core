package transfer

import (
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	"helios-core/helios-chain/x/ibc/transfer/keeper"
)

var _ porttypes.IBCModule = IBCModule{}

// IBCModule implements the ICS26 interface for transfer given the transfer keeper.
type IBCModule struct {
	*ibctransfer.IBCModule
}

// NewIBCModule creates a new IBCModule given the keeper
func NewIBCModule(k keeper.Keeper) IBCModule {
	transferModule := ibctransfer.NewIBCModule(*k.Keeper)
	return IBCModule{
		IBCModule: &transferModule,
	}
}
