package erc20creator_test

import (
	"testing"

	"helios-core/helios-chain/precompiles/erc20creator"

	"helios-core/helios-chain/testutil/integration/evmos/factory"
	"helios-core/helios-chain/testutil/integration/evmos/grpc"
	testkeyring "helios-core/helios-chain/testutil/integration/evmos/keyring"
	"helios-core/helios-chain/testutil/integration/evmos/network"

	"github.com/stretchr/testify/suite"
)

var s *PrecompileTestSuite

// PrecompileTestSuite is the implementation of the TestSuite interface for ERC20 precompile
// unit tests.
type PrecompileTestSuite struct {
	suite.Suite

	factory     factory.TxFactory
	grpcHandler grpc.Handler
	network     *network.UnitTestNetwork
	keyring     testkeyring.Keyring

	precompile *erc20creator.Precompile
}

func TestPrecompileTestSuite(t *testing.T) {
	s = new(PrecompileTestSuite)
	suite.Run(t, s)
}

func (s *PrecompileTestSuite) SetupTest() {
	keyring := testkeyring.New(2)
	nw := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(nw)
	txFactory := factory.New(nw, grpcHandler)

	s.factory = txFactory
	s.grpcHandler = grpcHandler
	s.keyring = keyring
	s.network = nw

	s.precompile = s.setupErc20CreatorPrecompile()
}
