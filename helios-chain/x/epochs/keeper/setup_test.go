package keeper_test

import (
	"time"

	"helios-core/helios-chain/testutil/integration/evmos/grpc"
	"helios-core/helios-chain/testutil/integration/evmos/keyring"
	"helios-core/helios-chain/testutil/integration/evmos/network"

	"helios-core/helios-chain/x/epochs/types"
)

const (
	day             = time.Hour * 24
	week            = time.Hour * 24 * 7
	month           = time.Hour * 24 * 31
	monthIdentifier = "month"
)

// KeeperTestSuite is the implementation of the test suite for the
// Epochs module.
type KeeperTestSuite struct {
	network *network.UnitTestNetwork
	keyring keyring.Keyring
	handler grpc.Handler
}

// SetupTest is the setup function for epoch module tests. If epochsInfo is provided empty
// the default genesis for the epoch module is used.
func SetupTest(epochsInfo []types.EpochInfo) *KeeperTestSuite {
	keys := keyring.New(1)

	customGenesis := network.CustomGenesisState{}
	epochsGenesis := types.DefaultGenesisState()

	if len(epochsInfo) > 0 {
		epochsGenesis = types.NewGenesisState(epochsInfo)
	}

	customGenesis[types.ModuleName] = epochsGenesis

	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
		network.WithCustomGenesis(customGenesis),
	)

	gh := grpc.NewIntegrationHandler(nw)

	return &KeeperTestSuite{
		network: nw,
		keyring: keys,
		handler: gh,
	}
}
