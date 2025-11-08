package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global erc20 module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding.
	//
	// The actual codec used for serialization should be provided to modules/erc20 and
	// defined at the application level.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// AminoCdc is a amino codec created to support amino JSON compatible msgs.
	AminoCdc = codec.NewAminoCodec(amino) //nolint:staticcheck
)

const (
	// Amino names
	convertERC20Name       = "helios/MsgConvertERC20"
	convertCoinName        = "helios/MsgConvertCoin" // keep it for backwards compatibility when querying txs
	updateParams           = "helios/erc20/MsgUpdateParams"
	registerERC20          = "helios/erc20/MsgRegisterERC20"
	toggleConversion       = "helios/erc20/MsgToggleConversion"
	registerAssetConsensus = "helios/erc20/NewAssetConsensusProposal"
	removeAssetConsensus   = "helios/erc20/RemoveAssetConsensusProposal"
	updateAssetConsensus   = "helios/erc20/UpdateAssetConsensusProposal"
	// New Msg types for generic proposal system
	addAssetConsensusMsg    = "evmos/x/erc20/MsgAddAssetConsensus"
	removeAssetConsensusMsg = "evmos/x/erc20/MsgRemoveAssetConsensus"
	updateAssetConsensusMsg = "evmos/x/erc20/MsgUpdateAssetConsensus"
)

// NOTE: This is required for the GetSignBytes function
func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}

// RegisterInterfaces register implementations
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgConvertCoin{}, // keep it for backwards compatibility when querying txs
		&MsgConvertERC20{},
		&MsgUpdateParams{},
		&MsgRegisterERC20{},
		&MsgToggleConversion{},
		&MsgAddAssetConsensus{},
		&MsgRemoveAssetConsensus{},
		&MsgUpdateAssetConsensus{},
	)
	registry.RegisterImplementations(
		(*govv1beta1.Content)(nil),
		&RegisterCoinProposal{}, // Keep interface for backwards compatibility on proposals query
		&RegisterERC20Proposal{},
		&ToggleTokenConversionProposal{},
		&AddNewAssetConsensusProposal{},
		&RemoveAssetConsensusProposal{},
		&UpdateAssetConsensusProposal{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the necessary x/erc20 interfaces and
// concrete types on the provided LegacyAmino codec. These types are used for
// Amino JSON serialization and EIP-712 compatibility.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgUpdateParams{}, updateParams, nil)
	cdc.RegisterConcrete(&MsgConvertERC20{}, convertERC20Name, nil)
	cdc.RegisterConcrete(&MsgConvertCoin{}, convertCoinName, nil)
	cdc.RegisterConcrete(&MsgRegisterERC20{}, registerERC20, nil)
	cdc.RegisterConcrete(&MsgToggleConversion{}, toggleConversion, nil)
	cdc.RegisterConcrete(&AddNewAssetConsensusProposal{}, registerAssetConsensus, nil)
	cdc.RegisterConcrete(&RemoveAssetConsensusProposal{}, removeAssetConsensus, nil)
	cdc.RegisterConcrete(&UpdateAssetConsensusProposal{}, updateAssetConsensus, nil)
	// New Msg types for generic proposal system
	cdc.RegisterConcrete(&MsgAddAssetConsensus{}, addAssetConsensusMsg, nil)
	cdc.RegisterConcrete(&MsgRemoveAssetConsensus{}, removeAssetConsensusMsg, nil)
	cdc.RegisterConcrete(&MsgUpdateAssetConsensus{}, updateAssetConsensusMsg, nil)
}
