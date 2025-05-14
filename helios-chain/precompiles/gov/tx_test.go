package gov_test

import (
	"fmt"
	"math/big"

	"cosmossdk.io/math"

	erc20types "helios-core/helios-chain/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/precompiles/gov"
	"helios-core/helios-chain/precompiles/testutil"
	utiltx "helios-core/helios-chain/testutil/tx"
	"helios-core/helios-chain/x/evm/core/vm"
)

func (s *PrecompileTestSuite) TestVote() {
	var ctx sdk.Context
	method := s.precompile.Methods[gov.VoteMethod]
	newVoterAddr := utiltx.GenerateAddress()
	const proposalID uint64 = 1
	const option uint8 = 1
	const metadata = "metadata"

	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func()
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []interface{} {
				return []interface{}{}
			},
			func() {},
			200000,
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 4, 0),
		},
		{
			"fail - invalid voter address",
			func() []interface{} {
				return []interface{}{
					"",
					proposalID,
					option,
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"invalid voter address",
		},
		{
			"fail - invalid voter address",
			func() []interface{} {
				return []interface{}{
					common.Address{},
					proposalID,
					option,
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"invalid voter address",
		},
		{
			"fail - using a different voter address",
			func() []interface{} {
				return []interface{}{
					newVoterAddr,
					proposalID,
					option,
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"does not match the voter address",
		},
		{
			"fail - invalid vote option",
			func() []interface{} {
				return []interface{}{
					s.keyring.GetAddr(0),
					proposalID,
					option + 10,
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"invalid vote option",
		},
		{
			"success - vote proposal success",
			func() []interface{} {
				return []interface{}{
					s.keyring.GetAddr(0),
					proposalID,
					option,
					metadata,
				}
			},
			func() {
				proposal, _ := s.network.App.GovKeeper.Proposals.Get(ctx, proposalID)
				_, _, tallyResult, err := s.network.App.GovKeeper.Tally(ctx, proposal)
				s.Require().NoError(err)
				s.Require().Equal(math.NewInt(3e18).String(), tallyResult.YesCount)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			var contract *vm.Contract
			contract, ctx = testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)

			_, err := s.precompile.Vote(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())

			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck()
			}
		})
	}
}

func (s *PrecompileTestSuite) TestVoteWeighted() {
	var ctx sdk.Context
	method := s.precompile.Methods[gov.VoteWeightedMethod]
	newVoterAddr := utiltx.GenerateAddress()
	const proposalID uint64 = 1
	const metadata = "metadata"

	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func()
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"fail - empty input args",
			func() []interface{} {
				return []interface{}{}
			},
			func() {},
			200000,
			true,
			fmt.Sprintf(cmn.ErrInvalidNumberOfArgs, 4, 0),
		},
		{
			"fail - invalid voter address",
			func() []interface{} {
				return []interface{}{
					"",
					proposalID,
					[]gov.WeightedVoteOption{},
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"invalid voter address",
		},
		{
			"fail - using a different voter address",
			func() []interface{} {
				return []interface{}{
					newVoterAddr,
					proposalID,
					[]gov.WeightedVoteOption{},
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"does not match the voter address",
		},
		{
			"fail - invalid vote option",
			func() []interface{} {
				return []interface{}{
					s.keyring.GetAddr(0),
					proposalID,
					[]gov.WeightedVoteOption{{Option: 10, Weight: "1.0"}},
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"invalid vote option",
		},
		{
			"fail - invalid weight sum",
			func() []interface{} {
				return []interface{}{
					s.keyring.GetAddr(0),
					proposalID,
					[]gov.WeightedVoteOption{
						{Option: 1, Weight: "0.5"},
						{Option: 2, Weight: "0.6"},
					},
					metadata,
				}
			},
			func() {},
			200000,
			true,
			"total weight overflow 1.00",
		},
		{
			"success - vote weighted proposal",
			func() []interface{} {
				return []interface{}{
					s.keyring.GetAddr(0),
					proposalID,
					[]gov.WeightedVoteOption{
						{Option: 1, Weight: "0.7"},
						{Option: 2, Weight: "0.3"},
					},
					metadata,
				}
			},
			func() {
				proposal, _ := s.network.App.GovKeeper.Proposals.Get(ctx, proposalID)
				_, _, tallyResult, err := s.network.App.GovKeeper.Tally(ctx, proposal)
				s.Require().NoError(err)
				s.Require().Equal("2100000000000000000", tallyResult.YesCount)
				s.Require().Equal("900000000000000000", tallyResult.AbstainCount)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			var contract *vm.Contract
			contract, ctx = testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)

			_, err := s.precompile.VoteWeighted(ctx, s.keyring.GetAddr(0), contract, s.network.GetStateDB(), &method, tc.malleate())

			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck()
			}
		})
	}
}

func (s *PrecompileTestSuite) TestAddNewAssetProposal() {
	var ctx sdk.Context
	method := s.precompile.Methods[gov.AddNewAssetProposalMethod]
	title := "Whitelist USDT into the consensus with a base stake of power 100"
	description := "Explaining why USDT would be a good potential for Helios consensus and why it would secure the market"
	assets := func(contractAddr string) []interface{} {
		return []interface{}{
			struct {
				Denom           string `json:"denom"`
				ContractAddress string `json:"contractAddress"`
				ChainId         string `json:"chainId"`
				Decimals        uint32 `json:"decimals"`
				BaseWeight      uint64 `json:"baseWeight"`
				Metadata        string `json:"metadata"`
			}{
				Denom:           "USDT",
				ContractAddress: contractAddr,
				ChainId:         "ethereum",
				Decimals:        6,
				BaseWeight:      100,
				Metadata:        "Tether stablecoin",
			}}
	}

	initialDeposit := big.NewInt(1000000000000000000)

	testCases := []struct {
		name        string
		malleate    func(contractAddress string) []interface{}
		postCheck   func(proposalId uint64)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"success - create add new asset proposal success",
			func(contractAddress string) []interface{} {
				return []interface{}{
					title,
					description,
					assets(contractAddress),
					initialDeposit,
				}
			},
			func(proposalId uint64) {
				proposal, err := s.network.App.GovKeeper.Proposals.Get(ctx, proposalId)
				s.Require().NoError(err)
				s.Require().Equal(proposal.Title, title)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			var contract *vm.Contract
			contract, ctx = testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)

			// Deploy a mock USDT test Erc20 token to pass DoesERC20ContractExist check in HandleAddNewAssetConsensusProposal method
			testCoinMetadata := banktypes.Metadata{}
			testErc20AssetAddress, _ := s.network.App.Erc20Keeper.DeployERC20Contract(ctx, testCoinMetadata)

			output, err := s.precompile.AddNewAssetProposal(s.keyring.GetAddr(0), s.network.App.GovKeeper, ctx, &method, contract, tc.malleate(testErc20AssetAddress.String()))
			proposalId, _ := method.Outputs.Unpack(output)
			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck(proposalId[0].(uint64))
			}
		})
	}
}

func (s *PrecompileTestSuite) TestRemoveAssetProposal() {
	var ctx sdk.Context
	method := s.precompile.Methods[gov.RemoveAssetProposalMethod]
	title := "Remove USDT from the consensus weight"
	description := "Explaining why USDT should be removed and does not benefit the network anymore"
	denoms := []string{"USDT"}
	usdt := func(contractAddr string) erc20types.Asset {
		return erc20types.Asset{
			Denom:           "USDT",
			ContractAddress: contractAddr,
			ChainId:         "ethereum",
			Decimals:        6,
			BaseWeight:      100,
			Symbol:          "USDT",
		}
	}

	initialDeposit := big.NewInt(1000000000000000000)

	testCases := []struct {
		name        string
		malleate    func(contractAddress string) []interface{}
		postCheck   func(proposalId uint64)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"success - create remove asset proposal success",
			func(contractAddress string) []interface{} {
				return []interface{}{
					title,
					description,
					denoms,
					initialDeposit,
				}
			},
			func(proposalId uint64) {
				proposal, err := s.network.App.GovKeeper.Proposals.Get(ctx, proposalId)
				s.Require().NoError(err)
				s.Require().Equal(proposal.Title, title)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			var contract *vm.Contract
			contract, ctx = testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)

			// Deploy a mock USDT test Erc20 token
			testCoinMetadata := banktypes.Metadata{}
			testErc20AssetAddress, _ := s.network.App.Erc20Keeper.DeployERC20Contract(ctx, testCoinMetadata)

			// Mock adding an available USDT denom in consensus whitelist
			s.network.App.Erc20Keeper.AddAssetToConsensusWhitelist(ctx, usdt(testErc20AssetAddress.String()))
			// fmt.Println(s.network.App.Erc20Keeper.GetAllWhitelistedAssets(ctx))

			output, err := s.precompile.RemoveAssetProposal(s.keyring.GetAddr(0), s.network.App.GovKeeper, ctx, &method, contract, tc.malleate(testErc20AssetAddress.String()))
			proposalId, _ := method.Outputs.Unpack(output)
			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck(proposalId[0].(uint64))
			}
		})
	}
}

func (s *PrecompileTestSuite) TestUpdateAssetProposal() {
	var ctx sdk.Context
	method := s.precompile.Methods[gov.UpdateAssetProposalMethod]
	title := "Increase USDT Weight in Consensus"
	description := "Proposal to increase USDT weight with high magnitude for increased staking power."
	usdt := func(contractAddr string) erc20types.Asset {
		return erc20types.Asset{
			Denom:           "USDT",
			ContractAddress: contractAddr,
			ChainId:         "ethereum",
			Decimals:        6,
			BaseWeight:      100,
			Symbol:          "USDT",
		}
	}
	usdtUpdate := []interface{}{struct {
		Denom     string `json:"denom"`
		Magnitude string `json:"magnitude"`
		Direction string `json:"direction"`
	}{
		Denom:     "USDT",
		Magnitude: "high",
		Direction: "up",
	}}

	initialDeposit := big.NewInt(1000000000000000000)

	testCases := []struct {
		name        string
		malleate    func() []interface{}
		postCheck   func(proposalId uint64)
		gas         uint64
		expError    bool
		errContains string
	}{
		{
			"success - create update asset proposal success",
			func() []interface{} {
				return []interface{}{
					title,
					description,
					usdtUpdate,
					initialDeposit,
				}
			},
			func(proposalId uint64) {
				proposal, err := s.network.App.GovKeeper.Proposals.Get(ctx, proposalId)
				s.Require().NoError(err)
				s.Require().Equal(proposal.Title, title)
			},
			200000,
			false,
			"",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			ctx = s.network.GetContext()

			var contract *vm.Contract
			contract, ctx = testutil.NewPrecompileContract(s.T(), ctx, s.keyring.GetAddr(0), s.precompile, tc.gas)

			// Deploy a mock USDT test Erc20 token
			testCoinMetadata := banktypes.Metadata{}
			testErc20AssetAddress, _ := s.network.App.Erc20Keeper.DeployERC20Contract(ctx, testCoinMetadata)

			// Mock adding an available USDT denom in consensus whitelist
			s.network.App.Erc20Keeper.AddAssetToConsensusWhitelist(ctx, usdt(testErc20AssetAddress.String()))

			output, err := s.precompile.UpdateAssetProposal(s.keyring.GetAddr(0), s.network.App.GovKeeper, ctx, &method, contract, tc.malleate())
			proposalId, _ := method.Outputs.Unpack(output)
			if tc.expError {
				s.Require().ErrorContains(err, tc.errContains)
			} else {
				s.Require().NoError(err)
				tc.postCheck(proposalId[0].(uint64))
			}
		})
	}
}
