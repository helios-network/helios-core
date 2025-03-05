package ante_test

import (
	storetypes "cosmossdk.io/store/types"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	evmante "helios-core/helios-chain/x/evm/ante"
)

func (suite *EvmAnteTestSuite) TestBuildEvmExecutionCtx() {
	network := network.New()

	ctx := evmante.BuildEvmExecutionCtx(network.GetContext())

	suite.Equal(storetypes.GasConfig{}, ctx.KVGasConfig())
	suite.Equal(storetypes.GasConfig{}, ctx.TransientKVGasConfig())
}
