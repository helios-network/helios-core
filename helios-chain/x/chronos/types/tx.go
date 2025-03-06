package types

import (
	"strings"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgScheduleEVMCall{}

func (msg *MsgScheduleEVMCall) Route() string {
	return RouterKey
}

func (msg *MsgScheduleEVMCall) Type() string {
	return "schedule-evm-call"
}

func (msg *MsgScheduleEVMCall) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		panic(err.Error())
	}
	return []sdk.AccAddress{owner}
}

func (msg *MsgScheduleEVMCall) GetSignBytes() []byte {
	return ModuleCdc.MustMarshalJSON(msg)
}

// Validate checks MsgScheduleEVMCall validity
func (msg *MsgScheduleEVMCall) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return errors.Wrap(err, "owner_address is invalid")
	}

	if !strings.HasPrefix(msg.ContractAddress, "0x") || len(msg.ContractAddress) != 42 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "contract_address is invalid")
	}

	if msg.AbiJson == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "abi_json cannot be empty")
	}

	if msg.MethodName == "" {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "method_name cannot be empty")
	}

	if msg.Frequency == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "frequency must be greater than zero")
	}

	// Optional expiration_block, 0 means no expiration
	if msg.ExpirationBlock != 0 && msg.ExpirationBlock <= msg.Frequency {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "expiration_block must be greater than frequency")
	}

	return nil
}

//----------------------------------------------------------------

var _ sdk.Msg = &MsgModifyScheduledEVMCall{}

func (msg *MsgModifyScheduledEVMCall) Route() string {
	return RouterKey
}

func (msg *MsgModifyScheduledEVMCall) Type() string {
	return "modify-scheduled-evm-call"
}

func (msg *MsgModifyScheduledEVMCall) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		panic(err.Error())
	}
	return []sdk.AccAddress{owner}
}

func (msg *MsgModifyScheduledEVMCall) GetSignBytes() []byte {
	return ModuleCdc.MustMarshalJSON(msg)
}

func (msg *MsgModifyScheduledEVMCall) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return errors.Wrap(err, "owner_address is invalid")
	}

	if msg.ScheduleId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "schedule_id must be valid")
	}

	if msg.NewFrequency == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "new_frequency must be greater than zero")
	}

	return nil
}

//----------------------------------------------------------------

var _ sdk.Msg = &MsgCancelScheduledEVMCall{}

func (msg *MsgCancelScheduledEVMCall) Route() string {
	return RouterKey
}

func (msg *MsgCancelScheduledEVMCall) Type() string {
	return "cancel-scheduled-evm-call"
}

func (msg *MsgCancelScheduledEVMCall) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		panic(err.Error())
	}
	return []sdk.AccAddress{owner}
}

func (msg *MsgCancelScheduledEVMCall) GetSignBytes() []byte {
	return ModuleCdc.MustMarshalJSON(msg)
}

func (msg *MsgCancelScheduledEVMCall) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return errors.Wrap(err, "owner_address is invalid")
	}

	if msg.ScheduleId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "schedule_id must be valid")
	}

	return nil
}
