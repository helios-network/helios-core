// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package gov_test

import (
	"fmt"
	"helios-core/helios-chain/precompiles/erc20/testdata"
	"helios-core/helios-chain/precompiles/gov"
	"helios-core/helios-chain/precompiles/testutil"
	"helios-core/helios-chain/testutil/integration/evmos/factory"
	testutiltx "helios-core/helios-chain/testutil/tx"
	erc20types "helios-core/helios-chain/x/erc20/types"
	"helios-core/helios-chain/x/evm/core/vm"
	evmtypes "helios-core/helios-chain/x/evm/types"
	"math/big"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"
)

// General variables used for integration tests
var (
	// differentAddr is an address generated for testing purposes that e.g. raises the different origin error
	differentAddr = testutiltx.GenerateAddress()
	// defaultCallArgs  are the default arguments for calling the smart contract
	//
	// NOTE: this has to be populated in a BeforeEach block because the contractAddr would otherwise be a nil address.
	callArgs factory.CallArgs
	// txArgs are the EVM transaction arguments to use in the transactions
	txArgs evmtypes.EvmTxArgs
	// defaultLogCheck instantiates a log check arguments struct with the precompile ABI events populated.
	defaultLogCheck testutil.LogCheckArgs
	// passCheck defines the arguments to check if the precompile returns no error
	passCheck testutil.LogCheckArgs
	// outOfGasCheck defines the arguments to check if the precompile returns out of gas error
	outOfGasCheck testutil.LogCheckArgs
	// mock erc20 contract address
	usdtErc20Addr common.Address
)

func TestKeeperIntegrationTestSuite(t *testing.T) {
	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

var _ = Describe("Calling governance precompile from EOA", func() {
	var s *PrecompileTestSuite
	const (
		proposalID uint64 = 1
		option     uint8  = 1
		metadata          = "metadata"
	)
	BeforeEach(func() {
		s = new(PrecompileTestSuite)
		s.SetupTest()

		// set the default call arguments
		callArgs = factory.CallArgs{
			ContractABI: s.precompile.ABI,
		}
		defaultLogCheck = testutil.LogCheckArgs{
			ABIEvents: s.precompile.ABI.Events,
		}
		passCheck = defaultLogCheck.WithExpPass(true)
		outOfGasCheck = defaultLogCheck.WithErrContains(vm.ErrOutOfGas.Error())

		// reset tx args each test to avoid keeping custom
		// values of previous tests (e.g. gasLimit)
		precompileAddr := s.precompile.Address()
		txArgs = evmtypes.EvmTxArgs{
			To: &precompileAddr,
		}
	})

	// =====================================
	// 				TRANSACTIONS
	// =====================================
	Describe("Execute Vote transaction", func() {
		const method = gov.VoteMethod

		BeforeEach(func() {
			// set the default call arguments
			callArgs.MethodName = method
		})

		It("should return error if the provided gasLimit is too low", func() {
			txArgs.GasLimit = 30000
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0), proposalID, option, metadata,
			}

			_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, outOfGasCheck)
			Expect(err).To(BeNil())

			// tally result yes count should remain unchanged
			proposal, _ := s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalID)
			_, _, tallyResult, err := s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())
			Expect(tallyResult.YesCount).To(Equal("0"), "expected tally result yes count to remain unchanged")
		})

		It("should return error if the origin is different than the voter", func() {
			callArgs.Args = []interface{}{
				differentAddr, proposalID, option, metadata,
			}

			voterSetCheck := defaultLogCheck.WithErrContains(gov.ErrDifferentOrigin, s.keyring.GetAddr(0).String(), differentAddr.String())

			_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil())
		})

		It("should vote success", func() {
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0), proposalID, option, metadata,
			}

			voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVote)

			_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			// tally result yes count should updated
			proposal, _ := s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalID)
			_, _, tallyResult, err := s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())

			Expect(tallyResult.YesCount).To(Equal(math.NewInt(3e18).String()), "expected tally result yes count updated")
		})
	})

	Describe("Execute VoteWeighted transaction", func() {
		const method = gov.VoteWeightedMethod

		BeforeEach(func() {
			callArgs.MethodName = method
		})

		It("should return error if the provided gasLimit is too low", func() {
			txArgs.GasLimit = 30000
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0),
				proposalID,
				[]gov.WeightedVoteOption{
					{Option: 1, Weight: "0.5"},
					{Option: 2, Weight: "0.5"},
				},
				metadata,
			}

			_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, outOfGasCheck)
			Expect(err).To(BeNil())

			// tally result should remain unchanged
			proposal, _ := s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalID)
			_, _, tallyResult, err := s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())
			Expect(tallyResult.YesCount).To(Equal("0"), "expected tally result to remain unchanged")
		})

		It("should return error if the origin is different than the voter", func() {
			callArgs.Args = []interface{}{
				differentAddr,
				proposalID,
				[]gov.WeightedVoteOption{
					{Option: 1, Weight: "0.5"},
					{Option: 2, Weight: "0.5"},
				},
				metadata,
			}

			voterSetCheck := defaultLogCheck.WithErrContains(gov.ErrDifferentOrigin, s.keyring.GetAddr(0).String(), differentAddr.String())

			_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil())
		})

		It("should vote weighted success", func() {
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0),
				proposalID,
				[]gov.WeightedVoteOption{
					{Option: 1, Weight: "0.7"},
					{Option: 2, Weight: "0.3"},
				},
				metadata,
			}

			voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVoteWeighted)

			_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			// tally result should be updated
			proposal, _ := s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalID)
			_, _, tallyResult, err := s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())

			expectedYesCount := math.NewInt(21e17) // 70% of 3e18
			Expect(tallyResult.YesCount).To(Equal(expectedYesCount.String()), "expected tally result yes count updated")

			expectedAbstainCount := math.NewInt(9e17) // 30% of 3e18
			Expect(tallyResult.AbstainCount).To(Equal(expectedAbstainCount.String()), "expected tally result no count updated")
		})
	})

	Describe("Full flow Execute Add/Update/Remove Asset Proposal and Vote for them", func() {
		const voteMethod = gov.VoteMethod
		const addNewAssetProposalMethod = gov.AddNewAssetProposalMethod
		const updateAssetProposalMethod = gov.UpdateAssetProposalMethod
		const removeAssetProposalMethod = gov.RemoveAssetProposalMethod
		var proposalId uint64

		BeforeEach(func() {
			// To bypass voting period of genesis proposal
			s.network.NextBlockAfter(time.Hour)
			erc20MinterV5Contract, err := testdata.LoadERC20MinterV5Contract()
			Expect(err).ToNot(HaveOccurred(), "failed to load ERC20 minter contract")

			contractOwner := s.keyring.GetKey(0)

			// Deploy an test ERC20 USDT contract (for adding)
			usdtErc20Addr, err = s.factory.DeployContract(
				contractOwner.Priv,
				evmtypes.EvmTxArgs{}, // NOTE: passing empty struct to use default values
				factory.ContractDeploymentData{
					Contract: erc20MinterV5Contract,
					ConstructorArgs: []interface{}{
						"TetherUSD", "USDT",
					},
				},
			)
			Expect(err).ToNot(HaveOccurred(), "failed to deploy contract")
			s.network.NextBlock()
		})

		It("should create add/update/remove asset proposal success and vote for them", func() {
			// Stage 1: Create Add New Asset Proposal
			callArgs.MethodName = addNewAssetProposalMethod
			title := "Whitelist USDT into the consensus with a base stake of power 100"
			description := "Explaining why USDT would be a good potential for Helios consensus and why it would secure the market"
			assets := func(contractAddr string) []gov.AssetData {
				return []gov.AssetData{{
					Denom:           "USDT",
					ContractAddress: contractAddr,
					ChainId:         "ethereum",
					Decimals:        6,
					BaseWeight:      100,
					Metadata:        "Tether stablecoin",
				}}
			}

			initialDeposit := big.NewInt(1000000000000000000)
			callArgs.Args = []interface{}{
				title, description, assets(usdtErc20Addr.String()), initialDeposit,
			}

			_, ethRes, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, passCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			err = s.precompile.UnpackIntoInterface(&proposalId, addNewAssetProposalMethod, ethRes.Ret)
			Expect(err).To(BeNil())

			// Stage 2: Vote for Add New Asset Proposal
			s.network.NextBlock()
			callArgs.MethodName = voteMethod
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0), proposalId, option, metadata,
			}

			voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVote)

			_, _, err = s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			// Check the asset whitelist before the voting period elapses
			whitelist := s.network.App.Erc20Keeper.GetAllWhitelistedAssets(s.network.GetContext())
			Expect(len(whitelist)).To(BeIdenticalTo(0))

			// tally result yes count should updated
			proposal, _ := s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalId)
			_, _, tallyResult, err := s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())
			Expect(tallyResult.YesCount).To(Equal(math.NewInt(3e18).String()), "expected tally result yes count updated")
			// Stage 3: Voting Period Of Add New Asset Proposal has passed
			s.network.NextBlockAfter(48 * time.Hour)

			proposal, _ = s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalId)
			Expect(proposal.Status.String()).To(BeIdenticalTo("PROPOSAL_STATUS_PASSED"))

			// Check the asset whitelist after the add new asset proposal has passed
			whitelist = s.network.App.Erc20Keeper.GetAllWhitelistedAssets(s.network.GetContext())
			fmt.Println(whitelist)
			Expect(whitelist[0].Denom).To(BeIdenticalTo("USDT"))

			// Stage 4: Create Update Asset Proposal
			callArgs.MethodName = updateAssetProposalMethod
			title = "Increase USDT Weight in Consensus"
			description = "Proposal to increase USDT weight with high magnitude for increased staking power."
			usdtUpdate := []erc20types.WeightUpdate{{
				Denom:     "USDT",
				Magnitude: "high",
				Direction: "up",
			}}

			initialDeposit = big.NewInt(1000000000000000000)
			callArgs.Args = []interface{}{
				title, description, usdtUpdate, initialDeposit,
			}

			_, ethRes, err = s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, passCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			err = s.precompile.UnpackIntoInterface(&proposalId, updateAssetProposalMethod, ethRes.Ret)
			Expect(err).To(BeNil())

			// Stage 5: Vote for Update Asset Proposal
			s.network.NextBlock()
			callArgs.MethodName = voteMethod
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0), proposalId, option, metadata,
			}

			// Check the asset whitelist before the voting period elapses
			whitelist = s.network.App.Erc20Keeper.GetAllWhitelistedAssets(s.network.GetContext())
			Expect(len(whitelist)).To(BeIdenticalTo(1))
			oldBaseWeight := whitelist[0].BaseWeight

			_, _, err = s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			// tally result yes count should updated
			proposal, _ = s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalId)
			_, _, tallyResult, err = s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())
			Expect(tallyResult.YesCount).To(Equal(math.NewInt(3e18).String()), "expected tally result yes count updated")

			// Stage 6: Voting Period Of Update Asset Proposal has passed
			s.network.NextBlockAfter(48 * time.Hour)

			proposal, _ = s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalId)
			Expect(proposal.Status.String()).To(BeIdenticalTo("PROPOSAL_STATUS_PASSED"))

			// Check the asset whitelist after the update asset proposal has passed
			whitelist = s.network.App.Erc20Keeper.GetAllWhitelistedAssets(s.network.GetContext())
			fmt.Println(whitelist)
			Expect(whitelist[0].BaseWeight).To(BeIdenticalTo(oldBaseWeight * 130 / 100)) // Magnitude: "high",

			// Stage 7: Create Remove Asset Proposal
			callArgs.MethodName = removeAssetProposalMethod
			title = "Remove USDT from the consensus weight"
			description = "Explaining why USDT should be removed and does not benefit the network anymore"
			denoms := []string{"USDT"}
			initialDeposit = big.NewInt(1000000000000000000)
			callArgs.Args = []interface{}{
				title, description, denoms, initialDeposit,
			}

			_, ethRes, err = s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, passCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			err = s.precompile.UnpackIntoInterface(&proposalId, removeAssetProposalMethod, ethRes.Ret)
			Expect(err).To(BeNil())

			// Stage 8: Vote for Remove Asset Proposal
			s.network.NextBlock()
			callArgs.MethodName = voteMethod
			callArgs.Args = []interface{}{
				s.keyring.GetAddr(0), proposalId, option, metadata,
			}

			// Check the asset whitelist before the voting period elapses
			whitelist = s.network.App.Erc20Keeper.GetAllWhitelistedAssets(s.network.GetContext())
			Expect(len(whitelist)).To(BeIdenticalTo(1))

			_, _, err = s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, callArgs, voterSetCheck)
			Expect(err).To(BeNil(), "error while calling the precompile")

			// tally result yes count should updated
			proposal, _ = s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalId)
			_, _, tallyResult, err = s.network.App.GovKeeper.Tally(s.network.GetContext(), proposal)
			Expect(err).To(BeNil())
			Expect(tallyResult.YesCount).To(Equal(math.NewInt(3e18).String()), "expected tally result yes count updated")

			// Stage 9: Voting Period Of Update Asset Proposal has passed
			s.network.NextBlockAfter(48 * time.Hour)

			proposal, _ = s.network.App.GovKeeper.Proposals.Get(s.network.GetContext(), proposalId)
			Expect(proposal.Status.String()).To(BeIdenticalTo("PROPOSAL_STATUS_PASSED"))

			// Check the asset whitelist after the update asset proposal has passed
			whitelist = s.network.App.Erc20Keeper.GetAllWhitelistedAssets(s.network.GetContext())
			Expect(len(whitelist)).To(BeIdenticalTo(0)) // empty whitelist
		})
	})

	// =====================================
	// 				QUERIES
	// =====================================
	Describe("Execute queries", func() {
		Context("vote query", func() {
			method := gov.GetVoteMethod
			BeforeEach(func() {
				// submit a vote
				voteArgs := factory.CallArgs{
					ContractABI: s.precompile.ABI,
					MethodName:  gov.VoteMethod,
					Args: []interface{}{
						s.keyring.GetAddr(0), proposalID, option, metadata,
					},
				}

				voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVote)

				_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, voteArgs, voterSetCheck)
				Expect(err).To(BeNil(), "error while calling the precompile")
				Expect(s.network.NextBlock()).To(BeNil())
			})
			It("should return a vote", func() {
				callArgs.MethodName = method
				callArgs.Args = []interface{}{proposalID, s.keyring.GetAddr(0)}
				txArgs.GasLimit = 200_000

				_, ethRes, err := s.factory.CallContractAndCheckLogs(
					s.keyring.GetPrivKey(0),
					txArgs,
					callArgs,
					passCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out gov.VoteOutput
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil())

				Expect(out.Vote.Voter).To(Equal(s.keyring.GetAddr(0)))
				Expect(out.Vote.ProposalId).To(Equal(proposalID))
				Expect(out.Vote.Metadata).To(Equal(metadata))
				Expect(out.Vote.Options).To(HaveLen(1))
				Expect(out.Vote.Options[0].Option).To(Equal(option))
				Expect(out.Vote.Options[0].Weight).To(Equal(math.LegacyOneDec().String()))
			})
		})

		Context("weighted vote query", func() {
			method := gov.GetVoteMethod
			BeforeEach(func() {
				// submit a weighted vote
				voteArgs := factory.CallArgs{
					ContractABI: s.precompile.ABI,
					MethodName:  gov.VoteWeightedMethod,
					Args: []interface{}{
						s.keyring.GetAddr(0),
						proposalID,
						[]gov.WeightedVoteOption{
							{Option: 1, Weight: "0.7"},
							{Option: 2, Weight: "0.3"},
						},
						metadata,
					},
				}

				voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVoteWeighted)

				_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, voteArgs, voterSetCheck)
				Expect(err).To(BeNil(), "error while calling the precompile")
				Expect(s.network.NextBlock()).To(BeNil())
			})

			It("should return a weighted vote", func() {
				callArgs.MethodName = method
				callArgs.Args = []interface{}{proposalID, s.keyring.GetAddr(0)}
				txArgs.GasLimit = 200_000

				_, ethRes, err := s.factory.CallContractAndCheckLogs(
					s.keyring.GetPrivKey(0),
					txArgs,
					callArgs,
					passCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out gov.VoteOutput
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil())

				Expect(out.Vote.Voter).To(Equal(s.keyring.GetAddr(0)))
				Expect(out.Vote.ProposalId).To(Equal(proposalID))
				Expect(out.Vote.Metadata).To(Equal(metadata))
				Expect(out.Vote.Options).To(HaveLen(2))
				Expect(out.Vote.Options[0].Option).To(Equal(uint8(1)))
				Expect(out.Vote.Options[0].Weight).To(Equal("0.7"))
				Expect(out.Vote.Options[1].Option).To(Equal(uint8(2)))
				Expect(out.Vote.Options[1].Weight).To(Equal("0.3"))
			})
		})

		Context("votes query", func() {
			method := gov.GetVotesMethod
			BeforeEach(func() {
				// submit votes
				for _, key := range s.keyring.GetKeys() {
					voteArgs := factory.CallArgs{
						ContractABI: s.precompile.ABI,
						MethodName:  gov.VoteMethod,
						Args: []interface{}{
							key.Addr, proposalID, option, metadata,
						},
					}

					voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVote)

					_, _, err := s.factory.CallContractAndCheckLogs(key.Priv, txArgs, voteArgs, voterSetCheck)
					Expect(err).To(BeNil(), "error while calling the precompile")
					Expect(s.network.NextBlock()).To(BeNil())
				}
			})
			It("should return all votes", func() {
				callArgs.MethodName = method
				callArgs.Args = []interface{}{
					proposalID,
					query.PageRequest{
						Limit:      10,
						CountTotal: true,
					},
				}
				txArgs.GasLimit = 200_000

				_, ethRes, err := s.factory.CallContractAndCheckLogs(
					s.keyring.GetPrivKey(0),
					txArgs,
					callArgs,
					passCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out gov.VotesOutput
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil())

				votersCount := len(s.keyring.GetKeys())
				Expect(out.PageResponse.Total).To(Equal(uint64(votersCount)))
				Expect(out.PageResponse.NextKey).To(Equal([]byte{}))
				Expect(out.Votes).To(HaveLen(votersCount))
				for _, v := range out.Votes {
					Expect(v.ProposalId).To(Equal(proposalID))
					Expect(v.Metadata).To(Equal(metadata))
					Expect(v.Options).To(HaveLen(1))
					Expect(v.Options[0].Option).To(Equal(option))
					Expect(v.Options[0].Weight).To(Equal(math.LegacyOneDec().String()))
				}
			})
		})

		Context("deposit query", func() {
			method := gov.GetDepositMethod
			BeforeEach(func() {
				callArgs.MethodName = method
			})

			It("should return a deposit", func() {
				callArgs.Args = []interface{}{proposalID, s.keyring.GetAddr(0)}
				txArgs.GasLimit = 200_000

				_, ethRes, err := s.factory.CallContractAndCheckLogs(
					s.keyring.GetPrivKey(0),
					txArgs,
					callArgs,
					passCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out gov.DepositOutput
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil())

				Expect(out.Deposit.ProposalId).To(Equal(proposalID))
				Expect(out.Deposit.Depositor).To(Equal(s.keyring.GetAddr(0)))
				Expect(out.Deposit.Amount).To(HaveLen(1))
				Expect(out.Deposit.Amount[0].Denom).To(Equal(s.network.GetDenom()))
				Expect(out.Deposit.Amount[0].Amount.Cmp(big.NewInt(100))).To(Equal(0))
			})
		})

		Context("deposits query", func() {
			method := gov.GetDepositsMethod
			BeforeEach(func() {
				callArgs.MethodName = method
			})

			It("should return all deposits", func() {
				callArgs.Args = []interface{}{
					proposalID,
					query.PageRequest{
						Limit:      10,
						CountTotal: true,
					},
				}
				txArgs.GasLimit = 200_000

				_, ethRes, err := s.factory.CallContractAndCheckLogs(
					s.keyring.GetPrivKey(0),
					txArgs,
					callArgs,
					passCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out gov.DepositsOutput
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil())

				Expect(out.PageResponse.Total).To(Equal(uint64(1)))
				Expect(out.PageResponse.NextKey).To(Equal([]byte{}))
				Expect(out.Deposits).To(HaveLen(1))
				for _, d := range out.Deposits {
					Expect(d.ProposalId).To(Equal(proposalID))
					Expect(d.Amount).To(HaveLen(1))
					Expect(d.Amount[0].Denom).To(Equal(s.network.GetDenom()))
					Expect(d.Amount[0].Amount.Cmp(big.NewInt(100))).To(Equal(0))
				}
			})
		})

		Context("tally result query", func() {
			method := gov.GetTallyResultMethod
			BeforeEach(func() {
				callArgs.MethodName = method
				voteArgs := factory.CallArgs{
					ContractABI: s.precompile.ABI,
					MethodName:  gov.VoteMethod,
					Args: []interface{}{
						s.keyring.GetAddr(0), proposalID, option, metadata,
					},
				}

				voterSetCheck := passCheck.WithExpEvents(gov.EventTypeVote)

				_, _, err := s.factory.CallContractAndCheckLogs(s.keyring.GetPrivKey(0), txArgs, voteArgs, voterSetCheck)
				Expect(err).To(BeNil(), "error while calling the precompile")
				Expect(s.network.NextBlock()).To(BeNil())
			})

			It("should return the tally result", func() {
				callArgs.Args = []interface{}{proposalID}
				txArgs.GasLimit = 200_000

				_, ethRes, err := s.factory.CallContractAndCheckLogs(
					s.keyring.GetPrivKey(0),
					txArgs,
					callArgs,
					passCheck,
				)
				Expect(err).To(BeNil(), "error while calling the smart contract: %v", err)

				var out gov.TallyResultOutput
				err = s.precompile.UnpackIntoInterface(&out, method, ethRes.Ret)
				Expect(err).To(BeNil())

				Expect(out.TallyResult.Yes).To(Equal("3000000000000000000"))
				Expect(out.TallyResult.Abstain).To(Equal("0"))
				Expect(out.TallyResult.No).To(Equal("0"))
				Expect(out.TallyResult.NoWithVeto).To(Equal("0"))
			})
		})
	})
})
