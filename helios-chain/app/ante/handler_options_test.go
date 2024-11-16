package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	ethante "helios-core/helios-chain/app/ante/evm"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	"helios-core/helios-chain/types"

	"helios-core/helios-chain/app/ante"
)

func TestValidateHandlerOptions(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	cases := []struct {
		name    string
		options ante.HandlerOptions
		expPass bool
	}{
		{
			"fail - empty options",
			ante.HandlerOptions{},
			false,
		},
		{
			"fail - empty account keeper",
			ante.HandlerOptions{
				Cdc:           nw.App.AppCodec(),
				AccountKeeper: nil,
			},
			false,
		},
		{
			"fail - empty bank keeper",
			ante.HandlerOptions{
				Cdc:           nw.App.AppCodec(),
				AccountKeeper: nw.App.AccountKeeper,
				BankKeeper:    nil,
			},
			false,
		},
		{
			"fail - empty distribution keeper",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nil,

				IBCKeeper: nil,
			},
			false,
		},
		{
			"fail - empty IBC keeper",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,

				IBCKeeper: nil,
			},
			false,
		},
		{
			"fail - empty staking keeper",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,

				IBCKeeper:     nw.App.IBCKeeper,
				StakingKeeper: nil,
			},
			false,
		},
		{
			"fail - empty fee market keeper",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,

				IBCKeeper:       nw.App.IBCKeeper,
				StakingKeeper:   nw.App.StakingKeeper,
				FeeMarketKeeper: nil,
			},
			false,
		},
		{
			"fail - empty EVM keeper",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,
				IBCKeeper:          nw.App.IBCKeeper,
				StakingKeeper:      nw.App.StakingKeeper,
				FeeMarketKeeper:    nw.App.FeeMarketKeeper,
				EvmKeeper:          nil,
			},
			false,
		},
		{
			"fail - empty signature gas consumer",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,
				IBCKeeper:          nw.App.IBCKeeper,
				StakingKeeper:      nw.App.StakingKeeper,
				FeeMarketKeeper:    nw.App.FeeMarketKeeper,
				EvmKeeper:          nw.App.EvmKeeper,
				SigGasConsumer:     nil,
			},
			false,
		},
		{
			"fail - empty signature mode handler",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,
				IBCKeeper:          nw.App.IBCKeeper,
				StakingKeeper:      nw.App.StakingKeeper,
				FeeMarketKeeper:    nw.App.FeeMarketKeeper,
				EvmKeeper:          nw.App.EvmKeeper,
				SigGasConsumer:     ante.SigVerificationGasConsumer,
				SignModeHandler:    nil,
			},
			false,
		},
		{
			"fail - empty tx fee checker",
			ante.HandlerOptions{
				Cdc:                nw.App.AppCodec(),
				AccountKeeper:      nw.App.AccountKeeper,
				BankKeeper:         nw.App.BankKeeper,
				DistributionKeeper: nw.App.DistrKeeper,
				IBCKeeper:          nw.App.IBCKeeper,
				StakingKeeper:      nw.App.StakingKeeper,
				FeeMarketKeeper:    nw.App.FeeMarketKeeper,
				EvmKeeper:          nw.App.EvmKeeper,
				SigGasConsumer:     ante.SigVerificationGasConsumer,
				SignModeHandler:    nw.App.GetTxConfig().SignModeHandler(),
				TxFeeChecker:       nil,
			},
			false,
		},
		{
			"success - default app options",
			ante.HandlerOptions{
				Cdc:                    nw.App.AppCodec(),
				AccountKeeper:          nw.App.AccountKeeper,
				BankKeeper:             nw.App.BankKeeper,
				DistributionKeeper:     nw.App.DistrKeeper,
				ExtensionOptionChecker: types.HasDynamicFeeExtensionOption,
				EvmKeeper:              nw.App.EvmKeeper,
				StakingKeeper:          nw.App.StakingKeeper,
				FeegrantKeeper:         nw.App.FeeGrantKeeper,
				IBCKeeper:              nw.App.IBCKeeper,
				FeeMarketKeeper:        nw.App.FeeMarketKeeper,
				SignModeHandler:        nw.GetEncodingConfig().TxConfig.SignModeHandler(),
				SigGasConsumer:         ante.SigVerificationGasConsumer,
				MaxTxGasWanted:         40000000,
				TxFeeChecker:           ethante.NewDynamicFeeChecker(nw.App.FeeMarketKeeper),
			},
			true,
		},
	}

	for _, tc := range cases {
		err := tc.options.Validate()
		if tc.expPass {
			require.NoError(t, err, tc.name)
		} else {
			require.Error(t, err, tc.name)
		}
	}
}
