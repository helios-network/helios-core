package types

import (
	"context"
	"math/big"
	"time"

	erc20types "helios-core/helios-chain/x/erc20/types"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
)

// StakingKeeper defines the expected staking keeper methods
type StakingKeeper interface {
	GetBondedValidatorsByPower(ctx context.Context) ([]stakingtypes.Validator, error)
	GetLastValidatorPower(ctx context.Context, operator sdk.ValAddress) (int64, error)
	GetLastTotalPower(ctx context.Context) (power math.Int, err error)
	IterateValidators(context.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
	ValidatorQueueIterator(ctx context.Context, endTime time.Time, endHeight int64) (storetypes.Iterator, error)
	GetParams(ctx context.Context) (stakingtypes.Params, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	IterateBondedValidatorsByPower(context.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
	IterateLastValidators(context.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
	Validator(context.Context, sdk.ValAddress) (stakingtypes.ValidatorI, error)
	ValidatorByConsAddr(context.Context, sdk.ConsAddress) (stakingtypes.ValidatorI, error)
	Slash(context.Context, sdk.ConsAddress, int64, int64, math.LegacyDec) (math.Int, error)
	Jail(context.Context, sdk.ConsAddress) error
	PowerReduction(ctx context.Context) (res math.Int)
}

// BankKeeper defines the expected bank keeper methods
type BankKeeper interface {
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	MintCoins(ctx context.Context, name string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetDenomMetaData(ctx context.Context, denom string) (bank.Metadata, bool)
	SetDenomMetaData(ctx context.Context, denomMetaData bank.Metadata)
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

type ERC20Keeper interface {
	GetTokenPairID(ctx sdk.Context, token string) []byte
	GetTokenPair(ctx sdk.Context, id []byte) (erc20types.TokenPair, bool)
	ConvertCoinNativeERC20(
		ctx sdk.Context,
		pair erc20types.TokenPair,
		amount math.Int,
		receiver common.Address,
		sender sdk.AccAddress,
	) error
	DeployERC20Contract(
		ctx sdk.Context,
		coinMetadata bank.Metadata,
	) (common.Address, error) 
	SetToken(ctx sdk.Context, pair erc20types.TokenPair)
	EnableDynamicPrecompiles(ctx sdk.Context, addresses ...common.Address) error
	MintERC20Tokens(
		ctx sdk.Context,
		contractAddr common.Address,
		recipient common.Address,
		amount *big.Int,
	) error
}

type SlashingKeeper interface {
	GetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress) (info slashingtypes.ValidatorSigningInfo, err error)
}

type DistributionKeeper interface {
	FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error
	GetFeePool(ctx context.Context) (feePool types.FeePool)
	SetFeePool(ctx context.Context, feePool types.FeePool)
}
