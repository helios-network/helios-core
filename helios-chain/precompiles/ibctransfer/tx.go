package ibctransfer

import (
	"fmt"
	"helios-core/helios-chain/x/evm/core/vm"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func (p *Precompile) TransferIBC(
	ctx sdk.Context,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) (bz []byte, err error) {
	// 1. Parse and validate arguments
	if len(args) != 5 {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, fmt.Errorf("expected 5 arguments, got %d", len(args))
	}
	destinationChain, ok := args[0].(string)
	if !ok {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, fmt.Errorf("invalid destinationChain")
	}
	recipient, ok := args[1].(string)
	if !ok {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, fmt.Errorf("invalid recipient")
	}
	amount, ok := args[2].(*big.Int)
	if !ok {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, fmt.Errorf("invalid amount")
	}
	denom, ok := args[3].(string)
	if !ok {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, fmt.Errorf("invalid denom")
	}
	timeoutTimestamp, ok := args[4].(*big.Int)
	if !ok {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, fmt.Errorf("invalid timeoutTimestamp")
	}

	// 2. Find the channel for the destination chain
	channelID, err := p.findChannelForChain(ctx, destinationChain)
	if err != nil {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, err
	}

	// 3. Build the MsgTransfer
	sender := sdk.AccAddress(contract.Caller().Bytes()).String()
	msg := transfertypes.NewMsgTransfer(
		"transfer", // port
		channelID,  // channel
		sdk.NewCoin(denom, sdk.NewIntFromBigInt(amount)),
		sender,
		recipient,
		clienttypes.NewHeight(0, 0),
		uint64(timeoutTimestamp.Uint64()),
		"", // memo
	)

	// 4. Deliver the message
	_, err = p.ibcKeeper.Transfer(ctx, msg)
	if err != nil {
		bz, packErr := method.Outputs.Pack(false)
		if packErr != nil {
			return nil, packErr
		}
		return bz, err
	}

	// 5. Return success
	bz, packErr := method.Outputs.Pack(true)
	if packErr != nil {
		return nil, packErr
	}
	return bz, nil
}

func (p *Precompile) findChannelForChain(ctx sdk.Context, destinationChain string) (string, error) {
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
		// Unpack the client state to get the chain ID
		tmClientState, ok := clientState.(*ibctmtypes.ClientState)
		if !ok {
			continue
		}
		if tmClientState.ChainId == destinationChain {
			return ch.ChannelId, nil
		}
	}
	return "", fmt.Errorf("no channel found for chain %s", destinationChain)
}
