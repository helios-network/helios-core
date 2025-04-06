package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	authzcdc "github.com/cosmos/cosmos-sdk/x/authz/codec"
)

// Legacy Amino Codec for JSON Serialization
var amino = codec.NewLegacyAmino()

// Protobuf Codec for Message Serialization
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// Amino JSON codec for backwards compatibility
var AminoCdc = codec.NewAminoCodec(amino) //nolint:staticcheck

func init() {
	RegisterLegacyAminoCodec(amino)
	RegisterLegacyAminoCodec(authzcdc.Amino)
	amino.Seal()
}

// RegisterInterfaces registers the interfaces for the proto stuff
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// Register Hyperion Messages
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgStoreLogoRequest{},
	)

	// Register MsgService Descriptor
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the necessary Hyperion messages for JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgStoreLogoRequest{}, "logos/MsgStoreLogoRequest", nil)
	cdc.RegisterConcrete(&Logo{}, "logos/Logo", nil)
}
