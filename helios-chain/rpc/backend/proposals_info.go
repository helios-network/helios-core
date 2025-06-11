// Copyright Jeremy Guyet

package backend

import (
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	cmn "helios-core/helios-chain/precompiles/common"
	rpctypes "helios-core/helios-chain/rpc/types"

	govprecompilestypes "helios-core/helios-chain/x/erc20/types"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ParseProposal(proposal *govtypes.Proposal, govParams *govtypes.Params) (map[string]interface{}, error) {
	statusTypes := map[govtypes.ProposalStatus]interface{}{
		govtypes.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED:    "UNSPECIFIED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD: "DEPOSIT_PERIOD",
		govtypes.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD:  "VOTING_PERIOD",
		govtypes.ProposalStatus_PROPOSAL_STATUS_PASSED:         "PASSED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_REJECTED:       "REJECTED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_FAILED:         "FAILED",
	}

	proposerAddr, err := sdk.AccAddressFromBech32(proposal.Proposer)
	if err != nil {
		return nil, err
	}
	details := make([]map[string]interface{}, 0)
	for _, anyJSON := range proposal.Messages {
		msg := &govtypes.MsgExecLegacyContent{}
		err := proto.Unmarshal(anyJSON.Value, msg)
		if err != nil {
			continue
		}
		newAssetConsensusProposal := &govprecompilestypes.AddNewAssetConsensusProposal{}
		err = proto.Unmarshal(msg.Content.Value, newAssetConsensusProposal)
		if err == nil {
			details = append(details, map[string]interface{}{
				"type":   "AddNewAssetConsensusProposal",
				"assets": newAssetConsensusProposal.Assets,
			})
			continue
		}
		updateAssetConsensusProposal := &govprecompilestypes.UpdateAssetConsensusProposal{}
		err = proto.Unmarshal(msg.Content.Value, updateAssetConsensusProposal)
		if err == nil {
			details = append(details, map[string]interface{}{
				"type":    "UpdateAssetConsensusProposal",
				"updates": updateAssetConsensusProposal.Updates,
			})
			continue
		}
		removeAssetConsensusProposal := &govprecompilestypes.RemoveAssetConsensusProposal{}
		err = proto.Unmarshal(msg.Content.Value, removeAssetConsensusProposal)
		if err == nil {
			details = append(details, map[string]interface{}{
				"type":   "RemoveAssetConsensusProposal",
				"denoms": removeAssetConsensusProposal.Denoms,
			})
			continue
		}
		hyperionProposal := &hyperiontypes.HyperionProposal{}
		err = proto.Unmarshal(msg.Content.Value, hyperionProposal)
		if err == nil {
			details = append(details, map[string]interface{}{
				"type": "HyperionProposal",
				"msg":  hyperionProposal.Msg,
			})
			continue
		}
		// TODO: manage unknow proposals
		details = append(details, map[string]interface{}{
			"type": "UnknownProposalType",
		})
	}

	return map[string]interface{}{
		"id":         proposal.Id,
		"statusCode": proposal.Status,
		"status":     statusTypes[proposal.Status],
		"proposer":   common.BytesToAddress(proposerAddr.Bytes()).String(),
		"title":      proposal.Title,
		"metadata":   proposal.Metadata,
		"summary":    proposal.Summary,
		"details":    details,
		"options": []*govtypes.WeightedVoteOption{
			{Option: govtypes.OptionYes, Weight: "Yes"},
			{Option: govtypes.OptionAbstain, Weight: "Abstain"},
			{Option: govtypes.OptionNo, Weight: "No"},
			{Option: govtypes.OptionNoWithVeto, Weight: "No With Veto"},
		},
		"votingStartTime":    proposal.VotingStartTime,
		"votingEndTime":      proposal.VotingEndTime,
		"submitTime":         proposal.SubmitTime,
		"totalDeposit":       proposal.TotalDeposit,
		"minDeposit":         proposal.GetMinDepositFromParams(*govParams),
		"finalTallyResult":   proposal.FinalTallyResult,
		"currentTallyResult": proposal.CurrentTallyResult,
	}, nil
}

func (b *Backend) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error) {
	proposalsResult := make([]map[string]interface{}, 0)
	proposals, err := b.queryClient.Gov.Proposals(b.ctx, &govtypes.QueryProposalsRequest{
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	})
	if err != nil {
		return nil, err
	}

	msg := &govtypes.QueryParamsRequest{
		ParamsType: "voting",
	}
	resParams, err := b.queryClient.Gov.Params(b.ctx, msg)
	if err != nil {
		return nil, err
	}
	for _, proposal := range proposals.Proposals {
		formattedProposal, err := ParseProposal(proposal, resParams.Params)
		if err != nil {
			continue
		}
		proposalsResult = append(proposalsResult, formattedProposal)
	}
	return proposalsResult, nil
}

func (b *Backend) GetProposal(id hexutil.Uint64) (map[string]interface{}, error) {
	proposalResponse, err := b.queryClient.Gov.Proposal(b.ctx, &govtypes.QueryProposalRequest{
		ProposalId: uint64(id),
	})
	if err != nil {
		return nil, err
	}
	msg := &govtypes.QueryParamsRequest{
		ParamsType: "voting",
	}
	resParams, err := b.queryClient.Gov.Params(b.ctx, msg)
	if err != nil {
		return nil, err
	}
	formattedProposal, err := ParseProposal(proposalResponse.Proposal, resParams.Params)
	if err != nil {
		return nil, err
	}
	return formattedProposal, nil
}

func (b *Backend) GetProposalVotesByPageAndSize(id uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalVoteRPC, error) {
	proposalResponse, err := b.queryClient.Gov.Proposal(b.ctx, &govtypes.QueryProposalRequest{
		ProposalId: uint64(id),
	})
	if err != nil {
		return nil, err
	}
	if proposalResponse.Proposal == nil {
		return nil, errors.New("proposal not found")
	}
	proposalVotes, err := b.queryClient.Gov.Votes(b.ctx, &govtypes.QueryVotesRequest{
		ProposalId: uint64(id),
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	})
	if err != nil {
		return nil, err
	}
	proposalVotesResult := make([]*rpctypes.ProposalVoteRPC, 0)
	for _, vote := range proposalVotes.Votes {
		options := make([]rpctypes.ProposalVoteOptionRPC, 0)
		for _, option := range vote.Options {
			options = append(options, rpctypes.ProposalVoteOptionRPC{
				Option: option.Option.String(),
				Weight: option.Weight,
			})
		}
		proposalVotesResult = append(proposalVotesResult, &rpctypes.ProposalVoteRPC{
			Voter:    cmn.AnyToHexAddress(vote.Voter).Hex(),
			Options:  options,
			Metadata: vote.Metadata,
		})
	}
	return proposalVotesResult, nil
}
