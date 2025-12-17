package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {}
