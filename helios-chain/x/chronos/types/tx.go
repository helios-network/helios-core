package types

import (
	"strings"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgCreateCron{}

func (msg *MsgCreateCron) Route() string {
	return RouterKey
}

func (msg *MsgCreateCron) Type() string {
	return "create-cron"
}

func (msg *MsgCreateCron) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		panic(err.Error())
	}
	return []sdk.AccAddress{owner}
}

func (msg *MsgCreateCron) GetSignBytes() []byte {
	return ModuleCdc.MustMarshalJSON(msg)
}

// Validate checks MsgScheduleEVMCall validity
func (msg *MsgCreateCron) Validate() error {
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

var _ sdk.Msg = &MsgUpdateCron{}

func (msg *MsgUpdateCron) Route() string {
	return RouterKey
}

func (msg *MsgUpdateCron) Type() string {
	return "update-cron"
}

func (msg *MsgUpdateCron) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		panic(err.Error())
	}
	return []sdk.AccAddress{owner}
}

func (msg *MsgUpdateCron) GetSignBytes() []byte {
	return ModuleCdc.MustMarshalJSON(msg)
}

func (msg *MsgUpdateCron) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return errors.Wrap(err, "owner_address is invalid")
	}

	if msg.CronId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "cron_id must be valid")
	}

	if msg.NewFrequency == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "new_frequency must be greater than zero")
	}

	return nil
}

//----------------------------------------------------------------

var _ sdk.Msg = &MsgCancelCron{}

func (msg *MsgCancelCron) Route() string {
	return RouterKey
}

func (msg *MsgCancelCron) Type() string {
	return "cancel-cron"
}

func (msg *MsgCancelCron) GetSigners() []sdk.AccAddress {
	owner, err := sdk.AccAddressFromBech32(msg.OwnerAddress)
	if err != nil {
		panic(err.Error())
	}
	return []sdk.AccAddress{owner}
}

func (msg *MsgCancelCron) GetSignBytes() []byte {
	return ModuleCdc.MustMarshalJSON(msg)
}

func (msg *MsgCancelCron) Validate() error {
	if _, err := sdk.AccAddressFromBech32(msg.OwnerAddress); err != nil {
		return errors.Wrap(err, "owner_address is invalid")
	}

	if msg.CronId == 0 {
		return errors.Wrap(sdkerrors.ErrInvalidRequest, "cron_id must be valid")
	}

	return nil
}
