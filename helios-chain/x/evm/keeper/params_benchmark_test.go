package keeper_test

import (
	"testing"

	"helios-core/helios-chain/x/evm/types"
)

func BenchmarkSetParams(b *testing.B) {
	suite := KeeperTestSuite{}
	suite.SetupTest()
	params := types.DefaultParams()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = suite.network.App.EvmKeeper.SetParams(suite.network.GetContext(), params)
	}
}

func BenchmarkGetParams(b *testing.B) {
	suite := KeeperTestSuite{}
	suite.SetupTest()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = suite.network.App.EvmKeeper.GetParams(suite.network.GetContext())
	}
}
