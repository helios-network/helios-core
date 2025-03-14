package types

import (
	"context"

	"cosmossdk.io/core/address"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected account keeper used for simulations (noalias)
type AccountKeeper interface {
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	GetModuleAddress(moduleName string) sdk.AccAddress
	IterateAccounts(ctx context.Context, cb func(account sdk.AccountI) bool)
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, account sdk.AccountI)
	RemoveAccount(ctx context.Context, account sdk.AccountI)
	// GetParams(ctx context.Context) (params authtypes.Params)
	GetSequence(ctx context.Context, account sdk.AccAddress) (uint64, error)
	AddressCodec() address.Codec
	// Methods imported from account should be defined here
}

type WasmMsgServer interface {
	ExecuteContract(context.Context, *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error)
	// Methods imported from account should be defined here
}
