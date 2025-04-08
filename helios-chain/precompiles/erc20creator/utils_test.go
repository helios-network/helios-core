package erc20creator_test

import (
	"helios-core/helios-chain/precompiles/erc20creator"
	"helios-core/helios-chain/testutil/integration/evmos/factory"
	evmtypes "helios-core/helios-chain/x/evm/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	//nolint:revive // dot imports are fine for Ginkgo
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	. "github.com/onsi/gomega"
)

func (s *PrecompileTestSuite) setupErc20CreatorPrecompile() *erc20creator.Precompile {
	precompile, err := erc20creator.NewPrecompile(
		s.network.App.Erc20Keeper,
		s.network.App.BankKeeper,
		s.network.App.LogosKeeper,
	)

	s.Require().NoError(err, "failed to create erc20creator precompile")
	return precompile
}

func (is *IntegrationTestSuite) setupErc20CreatorPrecompile() *erc20creator.Precompile {
	precompile, err := erc20creator.NewPrecompile(
		is.network.App.Erc20Keeper,
		is.network.App.BankKeeper,
		is.network.App.LogosKeeper,
	)

	Expect(err).ToNot(HaveOccurred(), "failed to create erc20creator precompile")
	return precompile
}

// callType constants to differentiate between direct calls and calls through a contract.
const (
	directCall = iota + 1
	contractCall
)

// ContractData is a helper struct to hold the addresses and ABIs for the
// different contract instances that are subject to testing here.
type ContractData struct {
	ownerPriv cryptotypes.PrivKey

	contractAddr   common.Address
	contractABI    abi.ABI
	precompileAddr common.Address
	precompileABI  abi.ABI
}

// getTxAndCallArgs is a helper function to return the correct call arguments for a given call type.
// In case of a direct call to the precompile, the precompile's ABI is used. Otherwise a caller contract is used.
func getTxAndCallArgs(
	callType int,
	contractData ContractData,
	methodName string,
	args ...interface{},
) (evmtypes.EvmTxArgs, factory.CallArgs) {
	txArgs := evmtypes.EvmTxArgs{}
	callArgs := factory.CallArgs{}

	switch callType {
	case directCall:
		txArgs.To = &contractData.precompileAddr
		callArgs.ContractABI = contractData.precompileABI
	case contractCall:
		txArgs.To = &contractData.contractAddr
		callArgs.ContractABI = contractData.contractABI
	}

	callArgs.MethodName = methodName
	callArgs.Args = args

	return txArgs, callArgs
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
