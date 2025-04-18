package utils_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	testkeyring "helios-core/helios-chain/testutil/integration/evmos/keyring"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	"helios-core/helios-chain/testutil/integration/evmos/utils"
)

func TestCheckBalances(t *testing.T) {
	testDenom := "atest"
	keyring := testkeyring.New(1)
	nw := network.New(
		network.WithDenom(testDenom),
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)

	testcases := []struct {
		name        string
		address     string
		expAmount   math.Int
		expPass     bool
		errContains string
	}{
		{
			name:      "pass",
			address:   keyring.GetAccAddr(0).String(),
			expAmount: network.PrefundedAccountInitialBalance,
			expPass:   true,
		},
		{
			name:        "fail - wrong amount",
			address:     keyring.GetAccAddr(0).String(),
			expAmount:   math.NewInt(1),
			errContains: "expected balance",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			balances := []banktypes.Balance{{
				Address: tc.address,
				Coins: sdk.NewCoins(
					sdk.NewCoin(testDenom, tc.expAmount),
				),
			}}

			err := utils.CheckBalances(nw.GetContext(), nw.GetBankClient(), balances)
			if tc.expPass {
				require.NoError(t, err, "unexpected error checking balances")
			} else {
				require.Error(t, err, "expected error checking balances")
				require.ErrorContains(t, err, tc.errContains, "expected different error checking balances")
			}
		})
	}
}
