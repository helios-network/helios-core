package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authzcdc "github.com/cosmos/cosmos-sdk/x/authz/codec"
	// this line is used by starport scaffolding # 1
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateDenom{}, "helios/tokenfactory/create-denom", nil)
	cdc.RegisterConcrete(&MsgMint{}, "helios/tokenfactory/mint", nil)
	cdc.RegisterConcrete(&MsgBurn{}, "helios/tokenfactory/burn", nil)
	// nolint:all
	// cdc.RegisterConcrete(&MsgForceTransfer{}, "helios/tokenfactory/force-transfer", nil)
	cdc.RegisterConcrete(&MsgChangeAdmin{}, "helios/tokenfactory/change-admin", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "helios/tokenfactory/update-params", nil)
	cdc.RegisterConcrete(&MsgSetDenomMetadata{}, "helios/tokenfactory/set-denom-metadata", nil)
	cdc.RegisterConcrete(&Params{}, "helios/tokenfactory/Params", nil)

}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreateDenom{},
		&MsgMint{},
		&MsgBurn{},
		// &MsgForceTransfer{},
		&MsgChangeAdmin{},
		&MsgUpdateParams{},
		&MsgSetDenomMetadata{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	ModuleCdc = codec.NewLegacyAmino()
)

func init() {
	RegisterCodec(ModuleCdc)
	// Register all Amino interfaces and concrete types on the authz Amino codec so that this can later be
	// used to properly serialize MsgGrant and MsgExec instances
	sdk.RegisterLegacyAminoCodec(ModuleCdc)
	RegisterCodec(authzcdc.Amino)

	ModuleCdc.Seal()
}
