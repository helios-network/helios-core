package types

import (
	"math/big"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// Helios defines the default coin denomination used in Ethermint in:
	//
	// - Staking parameters: denomination used as stake in the dPoS chain
	// - Mint parameters: denomination minted due to fee distribution rewards
	// - Governance parameters: denomination used for spam prevention in proposal deposits
	// - Crisis parameters: constant fee denomination used for spam prevention to check broken invariant
	// - EVM parameters: denomination used for running EVM state transitions in Ethermint.
	HeliosCoin string = "helios"

	BaseDenom        string = "ahelios"
	BaseDenomTestnet string = "athelios"

	// BaseDenomUnit defines the base denomination unit for helios.
	// 1 evmos = 1x10^{BaseDenomUnit} aevmos
	BaseDenomUnit = 18

	// DisplayDenom defines the denomination displayed to users in client applications.
	DisplayDenom        string = "helios"
	DisplayDenomTestnet string = "thelios"
)

var PowerReduction = sdkmath.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(BaseDenomUnit), nil))

// NewHeliosCoin is a utility function that returns an "helios" coin with the given math.Int amount.
// The function will panic if the provided amount is negative.
func NewHeliosCoin(amount math.Int) sdk.Coin {
	return sdk.NewCoin(HeliosCoin, amount)
}

// NewHeliosCoinInt64 is a utility function that returns an "helios" coin with the given int64 amount.
// The function will panic if the provided amount is negative.
func NewHeliosCoinInt64(amount int64) sdk.Coin {
	return sdk.NewInt64Coin(HeliosCoin, amount)
}
