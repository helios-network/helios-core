package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// Register messages
	cdc.RegisterConcrete(&MsgRegisterRevenue{}, "helios/feedistribution/MsgRegisterRevenue", nil)
	cdc.RegisterConcrete(&MsgUpdateRevenue{}, "helios/feedistribution/MsgUpdateRevenue", nil)
	cdc.RegisterConcrete(&MsgCancelRevenue{}, "helios/feedistribution/MsgCancelRevenue", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "helios/feedistribution/MsgUpdateParams", nil)
}

// RegisterInterfaces registers the interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// Register message implementations
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterRevenue{},
		&MsgUpdateRevenue{},
		&MsgCancelRevenue{},
		&MsgUpdateParams{},
	)

	// Register the msgservice descriptor
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
