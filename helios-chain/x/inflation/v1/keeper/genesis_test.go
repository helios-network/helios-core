package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	"helios-core/helios-chain/x/inflation/v1/types"
)

func TestInitGenesis(t *testing.T) {
	nw := network.NewUnitTestNetwork()
	ctx := nw.GetContext()
	// check calculated epochMintProvision at genesis
	epochMintProvision := nw.App.InflationKeeper.GetEpochMintProvision(ctx)
	expMintProvision := math.LegacyMustNewDecFromStr("282534246575342465753425").Quo(math.LegacyNewDec(types.ReductionFactor))
	require.Equal(t, expMintProvision, epochMintProvision)
}
