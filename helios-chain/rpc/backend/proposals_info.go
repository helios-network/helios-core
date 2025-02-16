// Copyright Jeremy Guyet
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"

	govprecompilestypes "helios-core/helios-chain/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *Backend) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error) {
	statusTypes := map[govtypes.ProposalStatus]interface{}{
		govtypes.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED:    "UNSPECIFIED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_DEPOSIT_PERIOD: "DEPOSIT_PERIOD",
		govtypes.ProposalStatus_PROPOSAL_STATUS_VOTING_PERIOD:  "VOTING_PERIOD",
		govtypes.ProposalStatus_PROPOSAL_STATUS_PASSED:         "PASSED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_REJECTED:       "REJECTED",
		govtypes.ProposalStatus_PROPOSAL_STATUS_FAILED:         "FAILED",
	}
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
			// TODO: manage unknow proposals
			details = append(details, map[string]interface{}{
				"type": "UnknownProposalType",
			})
		}
		proposalsResult = append(proposalsResult, map[string]interface{}{
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
			"votingStartTime":  proposal.VotingStartTime,
			"votingEndTime":    proposal.VotingEndTime,
			"submitTime":       proposal.SubmitTime,
			"totalDeposit":     proposal.TotalDeposit,
			"minDeposit":       proposal.GetMinDepositFromParams(*resParams.Params),
			"finalTallyResult": proposal.FinalTallyResult,
		})
	}

	return proposalsResult, nil
}
