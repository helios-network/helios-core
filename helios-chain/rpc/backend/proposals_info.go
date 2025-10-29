// Copyright Jeremy Guyet

package backend

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/gogoproto/proto"
	"github.com/pkg/errors"

	cmn "helios-core/helios-chain/precompiles/common"
	rpctypes "helios-core/helios-chain/rpc/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func ParseProposal(proposal *govtypes.Proposal, govParams *govtypes.Params, codec codec.Codec) (*rpctypes.ProposalRPC, error) {
	statusTypes := map[govtypes.ProposalStatus]string{
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
			details = append(details, map[string]interface{}{
				"type":  "UnknownProposalType",
				"error": err.Error(),
			})
			continue
		}

		contentJson, err := codec.MarshalInterfaceJSON(msg)
		if err != nil {
			details = append(details, map[string]interface{}{
				"type":  msg.Content.TypeUrl,
				"error": err.Error(),
			})
			continue
		}
		// json to interface
		var content map[string]interface{}
		err = json.Unmarshal(contentJson, &content)
		if err != nil {
			details = append(details, map[string]interface{}{
				"type":  msg.Content.TypeUrl,
				"error": err.Error(),
			})
			continue
		}
		decodedContent := content["content"].(map[string]interface{})

		// check if msg field exists and is string
		if decodedContent["msg"] != nil && decodedContent["msg"].(string) != "" {
			var interfaceMsgMap map[string]interface{}
			err = json.Unmarshal([]byte(decodedContent["msg"].(string)), &interfaceMsgMap)
			if err != nil {
				details = append(details, map[string]interface{}{
					"type":  msg.Content.TypeUrl,
					"error": err.Error(),
				})
				continue
			}
			decodedContent["msg"] = interfaceMsgMap
		}

		details = append(details, content["content"].(map[string]interface{}))
	}

	// return map[string]interface{}{
	// 	"id":         proposal.Id,
	// 	"statusCode": proposal.Status,
	// 	"status":     statusTypes[proposal.Status],
	// 	"proposer":   common.BytesToAddress(proposerAddr.Bytes()).String(),
	// 	"title":      proposal.Title,
	// 	"metadata":   proposal.Metadata,
	// 	"summary":    proposal.Summary,
	// 	"details":    details,
	// 	"options": []*govtypes.WeightedVoteOption{
	// 		{Option: govtypes.OptionYes, Weight: "Yes"},
	// 		{Option: govtypes.OptionAbstain, Weight: "Abstain"},
	// 		{Option: govtypes.OptionNo, Weight: "No"},
	// 		{Option: govtypes.OptionNoWithVeto, Weight: "No With Veto"},
	// 	},
	// 	"votingStartTime":    proposal.VotingStartTime,
	// 	"votingEndTime":      proposal.VotingEndTime,
	// 	"submitTime":         proposal.SubmitTime,
	// 	"totalDeposit":       proposal.TotalDeposit,
	// 	"minDeposit":         proposal.GetMinDepositFromParams(*govParams),
	// 	"finalTallyResult":   proposal.FinalTallyResult,
	// 	"currentTallyResult": proposal.CurrentTallyResult,
	// }, nil

	return &rpctypes.ProposalRPC{
		Id:         proposal.Id,
		StatusCode: proposal.Status.String(),
		Status:     statusTypes[proposal.Status],
		Proposer:   common.BytesToAddress(proposerAddr.Bytes()).String(),
		Title:      proposal.Title,
		Metadata:   proposal.Metadata,
		Summary:    proposal.Summary,
		Details:    details,
		Options: []rpctypes.ProposalVoteOptionRPC{
			{Option: govtypes.OptionYes.String(), Weight: "Yes"},
			{Option: govtypes.OptionAbstain.String(), Weight: "Abstain"},
			{Option: govtypes.OptionNo.String(), Weight: "No"},
			{Option: govtypes.OptionNoWithVeto.String(), Weight: "No With Veto"},
		},
		VotingStartTime:    *proposal.VotingStartTime,
		VotingEndTime:      *proposal.VotingEndTime,
		SubmitTime:         *proposal.SubmitTime,
		TotalDeposit:       proposal.TotalDeposit,
		MinDeposit:         proposal.GetMinDepositFromParams(*govParams),
		FinalTallyResult:   *proposal.FinalTallyResult,
		CurrentTallyResult: *proposal.CurrentTallyResult,
	}, nil
}

func (b *Backend) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.ProposalRPC, error) {
	proposalsResult := make([]*rpctypes.ProposalRPC, 0)
	proposals, err := b.queryClient.Gov.Proposals(b.ctx, &govtypes.QueryProposalsRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
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
		formattedProposal, err := ParseProposal(proposal, resParams.Params, b.clientCtx.Codec)
		if err != nil {
			continue
		}
		proposalsResult = append(proposalsResult, formattedProposal)
	}
	return proposalsResult, nil
}

func (b *Backend) GetProposal(id hexutil.Uint64) (*rpctypes.ProposalRPC, error) {
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
	formattedProposal, err := ParseProposal(proposalResponse.Proposal, resParams.Params, b.clientCtx.Codec)
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

func (b *Backend) GetProposalsCount() (*hexutil.Uint64, error) {
	response, err := b.queryClient.Gov.ProposalsCount(b.ctx, &govtypes.QueryProposalsCountRequest{})
	if err != nil {
		return nil, err
	}
	totalCount := hexutil.Uint64(response.Count)
	return &totalCount, nil
}

type ProposalFilter struct {
	Status      govtypes.ProposalStatus
	Proposer    string
	Title       string
	Description string
	Voter       string
	Depositor   string
	Request     *govtypes.QueryProposalsRequest
}

func (b *Backend) GetProposalsByPageAndSizeWithFilter(page hexutil.Uint64, size hexutil.Uint64, filter string) ([]*rpctypes.ProposalRPC, error) {
	// filters examples:
	// status=1
	// status=2
	// status=3
	// proposer=0xffffffffffffffffffffffffffffffffffffffff
	// title-matches=test
	// description-matches=test
	// status=1&proposer=0xffffffffffffffffffffffffffffffffffffffff&title-matches=test&description-matches=test

	var proposalFilter ProposalFilter
	proposalFilter.Status = govtypes.ProposalStatus_PROPOSAL_STATUS_UNSPECIFIED
	proposalFilter.Proposer = ""
	proposalFilter.Voter = ""
	proposalFilter.Depositor = ""
	proposalFilter.Title = ""
	proposalFilter.Description = ""
	proposalFilter.Request = &govtypes.QueryProposalsRequest{
		Pagination: &query.PageRequest{
			Offset:  (uint64(page) - 1) * uint64(size),
			Limit:   uint64(size),
			Reverse: true,
		},
	}

	for _, filter := range strings.Split(filter, "&") {
		parts := strings.Split(filter, "=")
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "status":
			parsedStatus, err := strconv.ParseUint(parts[1], 10, 32)
			if err != nil {
				continue
			}
			proposalFilter.Status = govtypes.ProposalStatus(uint32(parsedStatus))
			proposalFilter.Request.ProposalStatus = proposalFilter.Status
		case "proposer":
			proposalFilter.Proposer = cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(parts[1])).String()
			proposalFilter.Request.Proposer = proposalFilter.Proposer
		case "depositor":
			proposalFilter.Depositor = cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(parts[1])).String()
			proposalFilter.Request.Depositor = proposalFilter.Depositor
		case "voter":
			proposalFilter.Voter = cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(parts[1])).String()
			proposalFilter.Request.Voter = proposalFilter.Voter
		case "title-matches":
			proposalFilter.Title = parts[1]
		case "description-matches":
			proposalFilter.Description = parts[1]
		}
	}

	proposalsResult := make([]*rpctypes.ProposalRPC, 0)
	proposals, err := b.queryClient.Gov.Proposals(b.ctx, proposalFilter.Request)
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
		formattedProposal, err := ParseProposal(proposal, resParams.Params, b.clientCtx.Codec)
		if err != nil {
			continue
		}
		proposalsResult = append(proposalsResult, formattedProposal)
	}
	return proposalsResult, nil
}
