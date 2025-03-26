package types

import (
	"math/big"

	evmtypes "helios-core/helios-chain/x/evm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// BankKeeper defines the expected interface needed to retrieve account balances.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx sdk.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

// EVMKeeper defines the expected EVM keeper interface for supporting fee distribution.
type EVMKeeper interface {
	GetParams(ctx sdk.Context) evmtypes.Params
	ChainID() *big.Int
	GetContractDeployerAddress(ctx sdk.Context, contract common.Address) (sdk.AccAddress, bool)
	IsPrecompile(addr common.Address) bool
	GetEvmDenom(ctx sdk.Context) string
}

// AccountKeeper defines the expected account keeper interface for fee distribution.
type AccountKeeper interface {
	GetModuleAddress(moduleName string) sdk.AccAddress
}
