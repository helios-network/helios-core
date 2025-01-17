package erc20creator_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/precompiles/erc20creator"
	"helios-core/helios-chain/precompiles/erc20creator/testdata"
	"helios-core/helios-chain/precompiles/testutil"
	"helios-core/helios-chain/testutil/integration/evmos/factory"
	"helios-core/helios-chain/testutil/integration/evmos/grpc"
	"helios-core/helios-chain/testutil/integration/evmos/keyring"
	"helios-core/helios-chain/testutil/integration/evmos/network"
	evmtypes "helios-core/helios-chain/x/evm/types"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"
)

var is *IntegrationTestSuite

// IntegrationTestSuite is the implementation of the TestSuite interface for Erc20Creator precompile
// unit testis.
type IntegrationTestSuite struct {
	network     *network.UnitTestNetwork
	factory     factory.TxFactory
	grpcHandler grpc.Handler
	keyring     keyring.Keyring

	precompile *erc20creator.Precompile
}

func (is *IntegrationTestSuite) SetupTest() {
	keyring := keyring.New(2)

	integrationNetwork := network.NewUnitTestNetwork(
		network.WithPreFundedAccounts(keyring.GetAllAccAddrs()...),
	)
	grpcHandler := grpc.NewIntegrationHandler(integrationNetwork)
	txFactory := factory.New(integrationNetwork, grpcHandler)

	is.factory = txFactory
	is.grpcHandler = grpcHandler
	is.keyring = keyring
	is.network = integrationNetwork

	is.precompile = is.setupErc20CreatorPrecompile()
}

func TestIntegrationSuite(t *testing.T) {
	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Erc20Creator Extension Suite")
}

var _ = Describe("Erc20Creator Extension -", func() {
	var (
		erc20CreatorCallerContractAddr      common.Address
		erc20CreatorCallerContract          evmtypes.CompiledContract
		createErc20Method                   = "createErc20"
		createErc20FromCallerContractMethod = "callCreateErc20"
		err                                 error
		sender                              keyring.Key

		// contractData is a helper struct to hold the addresses and ABIs for the
		// different contract instances that are subject to testing here.
		contractData ContractData
		passCheck    testutil.LogCheckArgs
	)

	BeforeEach(func() {
		is = new(IntegrationTestSuite)
		is.SetupTest()

		sender = is.keyring.GetKey(0)

		erc20CreatorCallerContract, err = testdata.LoadErc20CreatorCallerContract()
		Expect(err).ToNot(HaveOccurred(), "failed to load Erc20CreatorCaller contract")

		erc20CreatorCallerContractAddr, err = is.factory.DeployContract(
			sender.Priv,
			evmtypes.EvmTxArgs{}, // NOTE: passing empty struct to use default values
			factory.ContractDeploymentData{
				Contract: erc20CreatorCallerContract,
			},
		)
		Expect(err).ToNot(HaveOccurred(), "failed to deploy Erc20CreatorCaller contract")

		contractData = ContractData{
			ownerPriv:      sender.Priv,
			precompileAddr: is.precompile.Address(),
			precompileABI:  is.precompile.Precompile.ABI,
			contractAddr:   erc20CreatorCallerContractAddr,
			contractABI:    erc20CreatorCallerContract.ABI,
		}

		passCheck = testutil.LogCheckArgs{}.WithExpPass(true)

		err = is.network.NextBlock()
		Expect(err).ToNot(HaveOccurred(), "failed to advance block")
	})

	Context("Direct precompile queries", func() {
		It("should return a deployed erc20 token contract address", func() {
			name := "TestToken"
			symbol := "TST"
			totalSupply := big.NewInt(1000000000000)
			decimals := uint8(18)

			txArgs, callArgs := getTxAndCallArgs(directCall, contractData, createErc20Method)
			callArgs.Args = []interface{}{name, symbol, totalSupply, decimals}

			_, ethRes, err := is.factory.CallContractAndCheckLogs(sender.Priv, txArgs, callArgs, passCheck)
			Expect(err).ToNot(HaveOccurred(), "unexpected result calling contract")
			var tokenAddress common.Address
			err = is.precompile.Precompile.ABI.UnpackIntoInterface(&tokenAddress, createErc20Method, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack erc20 token contract address")

			is.network.NextBlock()
			cAcc := is.network.App.EvmKeeper.GetAccount(is.network.GetContext(), tokenAddress)
			Expect(cAcc.IsContract()).To(Equal(true), "expected the erc20 contract to be deployed")
		})

	})

	Context("Calls from a contract", func() {
		It("should return a deployed erc20 token contract address", func() {
			name := "TestToken"
			symbol := "TST"
			totalSupply := big.NewInt(1000000000000)
			decimals := uint8(18)

			txArgs, callArgs := getTxAndCallArgs(contractCall, contractData, createErc20FromCallerContractMethod)
			callArgs.Args = []interface{}{name, symbol, totalSupply, decimals}

			_, ethRes, err := is.factory.CallContractAndCheckLogs(sender.Priv, txArgs, callArgs, passCheck)
			Expect(err).ToNot(HaveOccurred(), "unexpected result calling contract")
			var tokenAddress common.Address
			err = is.precompile.Precompile.ABI.UnpackIntoInterface(&tokenAddress, createErc20Method, ethRes.Ret)
			Expect(err).ToNot(HaveOccurred(), "failed to unpack erc20 token contract address")

			is.network.NextBlock()
			cAcc := is.network.App.EvmKeeper.GetAccount(is.network.GetContext(), tokenAddress)
			Expect(cAcc.IsContract()).To(Equal(true), "expected the erc20 contract to be deployed")
		})

	})
})
