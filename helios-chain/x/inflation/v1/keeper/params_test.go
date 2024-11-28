package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	"helios-core/helios-chain/x/inflation/v1/types"
)

func TestParams(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()
	testCases := []struct {
		name      string
		mockFunc  func() types.Params
		expParams types.Params
	}{
		{
			"pass - default params",
			func() types.Params {
				params := nw.App.InflationKeeper.GetParams(ctx)
				return params
			},
			types.DefaultParams(),
		},
		{
			"pass - setting new params",
			func() types.Params {
				params := types.DefaultParams()
				err := nw.App.InflationKeeper.SetParams(ctx, params)
				require.NoError(t, err)
				return params
			},
			nw.App.InflationKeeper.GetParams(ctx),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := tc.mockFunc()
			require.Equal(t, tc.expParams, params)
		})
	}
}
