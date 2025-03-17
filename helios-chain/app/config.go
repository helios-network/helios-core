//go:build !test
// +build !test

package app

import (
	"strings"

	"helios-core/helios-chain/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var sealed = false

// InitializeAppConfiguration allows to setup the global configuration
// for the Evmos EVM.
func InitializeAppConfiguration(chainID string) error {
	if sealed {
		return nil
	}

	// When calling any CLI command, it creates a tempApp inside RootCmdHandler with an empty chainID.
	if chainID == "" {
		return nil
	}

	id := strings.Split(chainID, "-")[0]
	coinInfo, found := evmtypes.ChainsCoinInfo[id]
	if !found {
		// default to mainnet coin info
		coinInfo = evmtypes.ChainsCoinInfo[utils.MainnetChainID]
	}

	if err := setBaseDenom(coinInfo); err != nil {
		return err
	}

	baseDenom, err := sdk.GetBaseDenom()
	if err != nil {
		return err
	}

	ethCfg := evmtypes.DefaultChainConfig(chainID)

	err = evmtypes.NewEVMConfigurator().
		WithExtendedEips(evmosActivators).
		WithChainConfig(ethCfg).
		WithEVMCoinInfo(baseDenom, uint8(coinInfo.Decimals)).
		Configure()
	if err != nil {
		return err
	}

	sealed = true
	return nil
}

// setBaseDenom registers the display denom and base denom and sets the
// base denom for the chain. The function registers different values based on
// the EvmCoinInfo to allow different configurations in mainnet and testnet.
func setBaseDenom(ci evmtypes.EvmCoinInfo) error {
	if err := sdk.RegisterDenom(ci.DisplayDenom, math.LegacyOneDec()); err != nil {
		return err
	}
	// sdk.RegisterDenom will automatically overwrite the base denom when the new denom units are lower than the current base denom's units.
	return sdk.RegisterDenom(ci.Denom, math.LegacyNewDecWithPrec(1, int64(ci.Decimals)))
}
