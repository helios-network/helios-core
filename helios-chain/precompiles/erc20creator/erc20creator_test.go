package erc20creator_test

import (
	"math/big"

	"helios-core/helios-chain/x/evm/core/vm"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"helios-core/helios-chain/x/evm/statedb"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

func (s *PrecompileTestSuite) TestRequiredGas() {
	testcases := []struct {
		name     string
		malleate func() []byte
		expGas   uint64
	}{
		{
			"success - createErc20 transaction with correct gas estimation",
			func() []byte {
				name := "TestToken"
				symbol := "TST"
				totalSupply := big.NewInt(1000000000000)
				decimals := uint8(18)
				input, err := s.precompile.Precompile.ABI.Pack(
					"createErc20",
					name,
					symbol,
					totalSupply,
					decimals,
				)
				s.Require().NoError(err, "failed to pack input")
				return input
			},
			200000,
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			s.SetupTest()

			// malleate contract input
			input := tc.malleate()
			gas := s.precompile.RequiredGas(input)

			s.Require().Equal(gas, tc.expGas)
		})
	}
}

// TestRun tests the precompile's Run method.
func (s *PrecompileTestSuite) TestRun() {
	testcases := []struct {
		name        string
		malleate    func() (common.Address, []byte)
		readOnly    bool
		expPass     bool
		errContains string
	}{
		{
			name: "pass - create a new erc20 token contract",
			malleate: func() (common.Address, []byte) {
				name := "TestToken"
				symbol := "TST"
				totalSupply := big.NewInt(1000000000000)
				decimals := uint8(18)
				input, err := s.precompile.Precompile.ABI.Pack(
					"createErc20",
					name,
					symbol,
					totalSupply,
					decimals,
				)
				s.Require().NoError(err, "failed to pack input")
				return s.keyring.GetAddr(0), input
			},
			readOnly: false,
			expPass:  true,
		},
		{
			name: "fail - decimals cannot exceed 18",
			malleate: func() (common.Address, []byte) {
				name := "TestToken"
				symbol := "TST"
				totalSupply := big.NewInt(1000000000000)
				decimals := uint8(19)
				input, err := s.precompile.Precompile.ABI.Pack(
					"createErc20",
					name,
					symbol,
					totalSupply,
					decimals,
				)
				s.Require().NoError(err, "failed to pack input")
				return s.keyring.GetAddr(0), input
			},
			readOnly: false,
			expPass:  false,
		},
		{
			name: "fail - symbol length cannot exceeds 32 characters",
			malleate: func() (common.Address, []byte) {
				name := "TestToken"
				symbol := "TSTTSTTSTTSTTSTTSTTSTTSTTSTTSTTST" // 33 characters
				totalSupply := big.NewInt(1000000000000)
				decimals := uint8(18)
				input, err := s.precompile.Precompile.ABI.Pack(
					"createErc20",
					name,
					symbol,
					totalSupply,
					decimals,
				)
				s.Require().NoError(err, "failed to pack input")
				return s.keyring.GetAddr(0), input
			},
			readOnly: false,
			expPass:  false,
		},
		{
			name: "fail - base denom length cannot exceeds 128 characters",
			malleate: func() (common.Address, []byte) {
				name := "TestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTestTokenTes" // 129 characters
				symbol := "TST"
				totalSupply := big.NewInt(1000000000000)
				decimals := uint8(18)
				input, err := s.precompile.Precompile.ABI.Pack(
					"createErc20",
					name,
					symbol,
					totalSupply,
					decimals,
				)
				s.Require().NoError(err, "failed to pack input")
				return s.keyring.GetAddr(0), input
			},
			readOnly: false,
			expPass:  false,
		},
		{
			name: "fail - total supply must be greater than zero",
			malleate: func() (common.Address, []byte) {
				name := "TestToken"
				symbol := "TST"
				totalSupply := big.NewInt(0)
				decimals := uint8(18)
				input, err := s.precompile.Precompile.ABI.Pack(
					"createErc20",
					name,
					symbol,
					totalSupply,
					decimals,
				)
				s.Require().NoError(err, "failed to pack input")
				return s.keyring.GetAddr(0), input
			},
			readOnly: false,
			expPass:  false,
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			// setup basic test suite
			s.SetupTest()
			ctx := s.network.GetContext()
			baseFee := s.network.App.EvmKeeper.GetBaseFee(ctx)

			// malleate testcase
			caller, input := tc.malleate()

			contract := vm.NewPrecompile(vm.AccountRef(caller), s.precompile, big.NewInt(0), uint64(1e6))
			contract.Input = input
			contractAddr := contract.Address()

			// Build and sign Ethereum transaction
			evmChainID := evmtypes.GetEthChainConfig().ChainID
			txArgs := evmtypes.EvmTxArgs{
				ChainID:   evmChainID,
				Nonce:     0,
				To:        &contractAddr,
				Amount:    nil,
				GasLimit:  100000,
				GasPrice:  big.NewInt(1e9),
				GasFeeCap: baseFee,
				GasTipCap: big.NewInt(1),
				Accesses:  &ethtypes.AccessList{},
			}
			msg, err := s.factory.GenerateGethCoreMsg(s.keyring.GetPrivKey(0), txArgs)
			s.Require().NoError(err)

			// Instantiate config
			proposerAddress := ctx.BlockHeader().ProposerAddress
			cfg, err := s.network.App.EvmKeeper.EVMConfig(ctx, proposerAddress)
			s.Require().NoError(err, "failed to instantiate EVM config")

			// Instantiate EVM
			headerHash := ctx.HeaderHash()
			stDB := statedb.New(
				ctx,
				s.network.App.EvmKeeper,
				statedb.NewEmptyTxConfig(common.BytesToHash(headerHash)),
			)
			evm := s.network.App.EvmKeeper.NewEVM(
				ctx, msg, cfg, nil, stDB,
			)

			precompiles, found, err := s.network.App.EvmKeeper.GetPrecompileInstance(ctx, contractAddr)
			s.Require().NoError(err, "failed to instantiate precompile")
			s.Require().True(found, "not found precompile")
			evm.WithPrecompiles(precompiles.Map, precompiles.Addresses)

			// Run precompiled contract
			bz, err := s.precompile.Run(evm, contract, tc.readOnly)

			// Check results
			if tc.expPass {
				s.Require().NoError(err, "expected no error when running the precompile")
				s.Require().NotNil(bz, "expected returned bytes not to be nil")
			} else {
				s.Require().Error(err, "expected error to be returned when running the precompile")
				s.Require().Nil(bz, "expected returned bytes to be nil")
				s.Require().ErrorContains(err, tc.errContains)
			}
		})
	}
}
