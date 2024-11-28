package keeper_test

import (
	"github.com/stretchr/testify/suite"

	"helios-core/helios-chain/testutil/integration/evmos/factory"
	"helios-core/helios-chain/testutil/integration/evmos/grpc"
	"helios-core/helios-chain/testutil/integration/evmos/keyring"
	"helios-core/helios-chain/testutil/integration/evmos/network"
)

type KeeperTestSuite struct {
	suite.Suite

	network *network.UnitTestNetwork
	handler grpc.Handler
	keyring keyring.Keyring
	factory factory.TxFactory
}

func (suite *KeeperTestSuite) SetupTest() {
	keys := keyring.New(2)
	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keys.GetAllAccAddrs()...),
	)
	gh := grpc.NewIntegrationHandler(nw)
	tf := factory.New(nw, gh)
	suite.network = nw
	suite.factory = tf
	suite.handler = gh
	suite.keyring = keys
}
