// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"

	"helios-core/helios-chain/testutil"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	utiltx "helios-core/helios-chain/testutil/tx"
	evmostypes "helios-core/helios-chain/types"
	"helios-core/helios-chain/x/staking/keeper"

	"github.com/stretchr/testify/require"
)

func TestMsgDelegate(t *testing.T) {
	var (
		ctx              sdk.Context
		nw               *network.UnitTestNetwork
		defaultDelCoin   = sdk.NewCoin(evmostypes.BaseDenom, math.NewInt(1e18))
		delegatorAddr, _ = utiltx.NewAccAddressAndKey()
	)

	testCases := []struct { //nolint:dupl
		name   string
		setup  func() sdk.Coin
		expErr bool
		errMsg string
	}{
		{
			name: "can delegate from a common account",
			setup: func() sdk.Coin {
				// Send some funds to delegator account
				err := testutil.FundAccountWithBaseDenom(ctx, nw.App.BankKeeper, delegatorAddr, defaultDelCoin.Amount.Int64())
				require.NoError(t, err)
				return defaultDelCoin
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			delCoin := tc.setup()

			srv := keeper.NewMsgServerImpl(nw.App.StakingKeeper)
			res, err := srv.Delegate(ctx, &types.MsgDelegate{
				DelegatorAddress: delegatorAddr.String(),
				ValidatorAddress: nw.GetValidators()[0].OperatorAddress,
				Amount:           delCoin,
			})

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}
		})
	}
}

func TestMsgCreateValidator(t *testing.T) {
	var (
		ctx              sdk.Context
		nw               *network.UnitTestNetwork
		defaultDelCoin   = sdk.NewCoin(evmostypes.BaseDenom, math.NewInt(1e18))
		validatorAddr, _ = utiltx.NewAccAddressAndKey()
	)

	testCases := []struct { //nolint:dupl
		name   string
		setup  func() sdk.Coin
		expErr bool
		errMsg string
	}{
		{
			name: "can create a validator using a common account",
			setup: func() sdk.Coin {
				// Send some funds to delegator account
				err := testutil.FundAccountWithBaseDenom(ctx, nw.App.BankKeeper, validatorAddr, defaultDelCoin.Amount.Int64())
				require.NoError(t, err)
				return defaultDelCoin
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nw = network.NewUnitTestNetwork()
			ctx = nw.GetContext()
			coinToSelfBond := tc.setup()

			pubKey := ed25519.GenPrivKey().PubKey()
			commissions := types.NewCommissionRates(
				math.LegacyNewDecWithPrec(5, 2),
				math.LegacyNewDecWithPrec(2, 1),
				math.LegacyNewDecWithPrec(5, 2),
			)
			msg, err := types.NewMsgCreateValidator(
				sdk.ValAddress(validatorAddr).String(),
				pubKey,
				coinToSelfBond,
				types.NewDescription("T", "E", "S", "T", "Z"),
				commissions,
				math.OneInt(),
			)
			require.NoError(t, err)
			srv := keeper.NewMsgServerImpl(nw.App.StakingKeeper)
			res, err := srv.CreateValidator(ctx, msg)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)
			}
		})
	}
}
