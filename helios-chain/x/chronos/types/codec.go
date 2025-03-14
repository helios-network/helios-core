package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// Legacy Amino Codec for JSON Serialization
var amino = codec.NewLegacyAmino()

// Protobuf Codec for Message Serialization
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// Amino JSON codec for backwards compatibility
var AminoCdc = codec.NewAminoCodec(amino) //nolint:staticcheck

// Constants for Amino encoding (used for JSON compatibility)
const (
	createCron = "helios/chronos/MsgCreateCron"
	updateCron = "helios/chronos/MsgUpdateCron"
	cancelCron = "helios/chronos/MsgCancelCron"
)

// Init function to register codecs and seal Amino
func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}

// RegisterInterfaces registers message and proposal implementations.
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// Register Chronos Messages
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgCreateCron{},
		&MsgUpdateCron{},
		&MsgCancelCron{},
	)

	// Register MsgService Descriptor
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the necessary Chronos messages for JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateCron{}, createCron, nil)
	cdc.RegisterConcrete(&MsgUpdateCron{}, updateCron, nil)
	cdc.RegisterConcrete(&MsgCancelCron{}, cancelCron, nil)
}
