package logos

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/evm/core/vm"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	cmn "helios-core/helios-chain/precompiles/common"
)

const (
	EventTypeLogoUploaded = "LogoUploaded"
)

func (p Precompile) EmitLogoUploadedEvent(ctx sdk.Context, stateDB vm.StateDB, from, to common.Address, logoHash string) error {
	// Prepare the event topics
	event := p.ABI.Events[EventTypeLogoUploaded]
	topics := make([]common.Hash, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID

	var err error
	topics[1], err = cmn.MakeTopic(from) // index 1
	if err != nil {
		return err
	}

	topics[2], err = cmn.MakeTopic(to) // index 2
	if err != nil {
		return err
	}

	arguments := abi.Arguments{event.Inputs[2]} // cronId
	packed, err := arguments.Pack(logoHash)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()), //nolint:gosec // G115
	})

	return nil
}
