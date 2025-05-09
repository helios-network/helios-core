package ibctransfer

import (
	"fmt"
	"helios-core/helios-chain/x/evm/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// GetSupportedChains returns a list of all unique destination chain IDs with active IBC channels.
func (p *Precompile) GetSupportedChains(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	chains := make(map[string]struct{})

	// Iterate all channels
	channels := p.ibcKeeper.ChannelKeeper.GetAllChannels(ctx)
	for _, ch := range channels {
		if ch.PortId != "transfer" {
			continue
		}
		channel, found := p.ibcKeeper.ChannelKeeper.GetChannel(ctx, ch.PortId, ch.ChannelId)
		if !found || len(channel.ConnectionHops) == 0 {
			continue
		}
		connectionID := channel.ConnectionHops[0]
		connection, found := p.ibcKeeper.ConnectionKeeper.GetConnection(ctx, connectionID)
		if !found {
			continue
		}
		clientID := connection.ClientId
		clientState, found := p.ibcKeeper.ClientKeeper.GetClientState(ctx, clientID)
		if !found {
			continue
		}
		tmClientState, ok := clientState.(*ibctmtypes.ClientState)
		if !ok {
			continue
		}
		chains[tmClientState.ChainId] = struct{}{}
	}

	// Convert map to slice
	chainList := make([]string, 0, len(chains))
	for chainID := range chains {
		chainList = append(chainList, chainID)
	}

	// Pack as string[] using ABI
	bz, err := method.Outputs.Pack(chainList)
	if err != nil {
		return nil, fmt.Errorf("failed to pack supported chains: %w", err)
	}
	return bz, nil
}
