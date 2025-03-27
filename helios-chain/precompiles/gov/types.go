package gov

import (
	"fmt"
	"helios-core/helios-chain/utils"
	"math/big"
	"reflect"

	"helios-core/helios-chain/x/erc20/types"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	cmn "helios-core/helios-chain/precompiles/common"
)

// EventVote defines the event data for the Vote transaction.
type EventVote struct {
	Voter      common.Address
	ProposalId uint64 //nolint:revive,stylecheck
	Option     uint8
}

// EventVoteWeighted defines the event data for the VoteWeighted transaction.
type EventVoteWeighted struct {
	Voter      common.Address
	ProposalId uint64 //nolint:revive,stylecheck
	Options    WeightedVoteOptions
}

// VotesInput defines the input for the Votes query.
type VotesInput struct {
	ProposalId uint64 //nolint:revive,stylecheck
	Pagination query.PageRequest
}

// VotesOutput defines the output for the Votes query.
type VotesOutput struct {
	Votes        []WeightedVote
	PageResponse query.PageResponse
}

// VoteOutput is the output response returned by the vote query method.
type VoteOutput struct {
	Vote WeightedVote
}

// WeightedVote defines a struct of vote for vote split.
type WeightedVote struct {
	ProposalId uint64 //nolint:revive,stylecheck
	Voter      common.Address
	Options    []WeightedVoteOption
	Metadata   string
}

// WeightedVoteOption defines a unit of vote for vote split.
type WeightedVoteOption struct {
	Option uint8
	Weight string
}

// WeightedVoteOptions defines a slice of WeightedVoteOption.
type WeightedVoteOptions []WeightedVoteOption

// DepositInput defines the input for the Deposit query.
type DepositInput struct {
	ProposalId uint64 //nolint:revive,stylecheck
	Depositor  common.Address
}

// DepositOutput defines the output for the Deposit query.
type DepositOutput struct {
	Deposit DepositData
}

// DepositsInput defines the input for the Deposits query.
type DepositsInput struct {
	ProposalId uint64 //nolint:revive,stylecheck
	Pagination query.PageRequest
}

// DepositsOutput defines the output for the Deposits query.
type DepositsOutput struct {
	Deposits     []DepositData      `abi:"deposits"`
	PageResponse query.PageResponse `abi:"pageResponse"`
}

// TallyResultOutput defines the output for the TallyResult query.
type TallyResultOutput struct {
	TallyResult TallyResultData
}

// DepositData represents information about a deposit on a proposal
type DepositData struct {
	ProposalId uint64         `abi:"proposalId"` //nolint:revive,stylecheck
	Depositor  common.Address `abi:"depositor"`
	Amount     []cmn.Coin     `abi:"amount"`
}

// TallyResultData represents the tally result of a proposal
type TallyResultData struct {
	Yes        string
	Abstain    string
	No         string
	NoWithVeto string
}

// AddNewAssetConsensusProposalOutput defines the structure for the output of adding a new asset to the consensus.
type AddNewAssetConsensusProposalOutput struct {
	Proposal AddNewAssetConsensusProposalData
}

// AddNewAssetConsensusProposalData contains the detailed information of the proposal.
type AddNewAssetConsensusProposalData struct {
	Title       string      `json:"title"`       // Title of the proposal.
	Description string      `json:"description"` // Description of the proposal.
	Assets      []AssetData `json:"assets"`      // List of assets included in the proposal.
}

// AssetData defines the structure of an individual asset within the proposal.
type AssetData struct {
	Denom           string `json:"denom"`            // Denomination of the asset (e.g., 'USDT').
	ContractAddress string `json:"contract_address"` // Smart contract address associated with the asset.
	ChainId         string `json:"chain_id"`         // Chain where the asset is deployed (e.g., 'ethereum').
	Decimals        uint32 `json:"decimals"`         // Number of decimal places for the asset.
	BaseWeight      uint64 `json:"base_weight"`      // Base stake weight or value of the asset.
	Metadata        string `json:"metadata"`         // Additional metadata for the asset.
}

// NewMsgVote creates a new MsgVote instance.
func NewMsgVote(args []interface{}) (*govv1.MsgVote, common.Address, error) {
	if len(args) != 4 {
		return nil, common.Address{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	voterAddress, ok := args[0].(common.Address)
	if !ok || voterAddress == (common.Address{}) {
		return nil, common.Address{}, fmt.Errorf(ErrInvalidVoter, args[0])
	}

	proposalID, ok := args[1].(uint64)
	if !ok {
		return nil, common.Address{}, fmt.Errorf(ErrInvalidProposalID, args[1])
	}

	option, ok := args[2].(uint8)
	if !ok {
		return nil, common.Address{}, fmt.Errorf(ErrInvalidOption, args[2])
	}

	metadata, ok := args[3].(string)
	if !ok {
		return nil, common.Address{}, fmt.Errorf(ErrInvalidMetadata, args[3])
	}

	msg := &govv1.MsgVote{
		ProposalId: proposalID,
		Voter:      sdk.AccAddress(voterAddress.Bytes()).String(),
		Option:     govv1.VoteOption(option),
		Metadata:   metadata,
	}

	return msg, voterAddress, nil
}

// NewMsgVoteWeighted creates a new MsgVoteWeighted instance.
func NewMsgVoteWeighted(method *abi.Method, args []interface{}) (*govv1.MsgVoteWeighted, common.Address, WeightedVoteOptions, error) {
	if len(args) != 4 {
		return nil, common.Address{}, WeightedVoteOptions{}, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 4, len(args))
	}

	voterAddress, ok := args[0].(common.Address)
	if !ok || voterAddress == (common.Address{}) {
		return nil, common.Address{}, WeightedVoteOptions{}, fmt.Errorf(ErrInvalidVoter, args[0])
	}

	proposalID, ok := args[1].(uint64)
	if !ok {
		return nil, common.Address{}, WeightedVoteOptions{}, fmt.Errorf(ErrInvalidProposalID, args[1])
	}

	// Unpack the input struct
	var options WeightedVoteOptions
	arguments := abi.Arguments{method.Inputs[2]}
	if err := arguments.Copy(&options, []interface{}{args[2]}); err != nil {
		return nil, common.Address{}, WeightedVoteOptions{}, fmt.Errorf("error while unpacking args to Options struct: %s", err)
	}

	weightedOptions := make([]*govv1.WeightedVoteOption, len(options))
	for i, option := range options {
		weightedOptions[i] = &govv1.WeightedVoteOption{
			Option: govv1.VoteOption(option.Option),
			Weight: option.Weight,
		}
	}

	metadata, ok := args[3].(string)
	if !ok {
		return nil, common.Address{}, WeightedVoteOptions{}, fmt.Errorf(ErrInvalidMetadata, args[3])
	}

	msg := &govv1.MsgVoteWeighted{
		ProposalId: proposalID,
		Voter:      sdk.AccAddress(voterAddress.Bytes()).String(),
		Options:    weightedOptions,
		Metadata:   metadata,
	}

	return msg, voterAddress, options, nil
}

// ParseVotesArgs parses the arguments for the Votes query.
func ParseVotesArgs(method *abi.Method, args []interface{}) (*govv1.QueryVotesRequest, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	var input VotesInput
	if err := method.Inputs.Copy(&input, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to VotesInput: %s", err)
	}

	return &govv1.QueryVotesRequest{
		ProposalId: input.ProposalId,
		Pagination: &input.Pagination,
	}, nil
}

func (vo *VotesOutput) FromResponse(res *govv1.QueryVotesResponse) *VotesOutput {
	vo.Votes = make([]WeightedVote, len(res.Votes))
	for i, v := range res.Votes {
		hexAddr, err := utils.Bech32ToHexAddr(v.Voter)
		if err != nil {
			return nil
		}
		options := make([]WeightedVoteOption, len(v.Options))
		for j, opt := range v.Options {
			options[j] = WeightedVoteOption{
				Option: uint8(opt.Option), //nolint:gosec // G115
				Weight: opt.Weight,
			}
		}
		vo.Votes[i] = WeightedVote{
			ProposalId: v.ProposalId,
			Voter:      hexAddr,
			Options:    options,
			Metadata:   v.Metadata,
		}
	}
	if res.Pagination != nil {
		vo.PageResponse = query.PageResponse{
			NextKey: res.Pagination.NextKey,
			Total:   res.Pagination.Total,
		}
	}
	return vo
}

// ParseVoteArgs parses the arguments for the Votes query.
func ParseVoteArgs(args []interface{}) (*govv1.QueryVoteRequest, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	proposalID, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf(ErrInvalidProposalID, args[0])
	}

	voter, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(ErrInvalidVoter, args[1])
	}

	voterAccAddr := sdk.AccAddress(voter.Bytes())
	return &govv1.QueryVoteRequest{
		ProposalId: proposalID,
		Voter:      voterAccAddr.String(),
	}, nil
}

func (vo *VoteOutput) FromResponse(res *govv1.QueryVoteResponse) *VoteOutput {
	hexVoter, err := utils.Bech32ToHexAddr(res.Vote.Voter)
	if err != nil {
		return nil
	}
	vo.Vote.Voter = hexVoter
	vo.Vote.Metadata = res.Vote.Metadata
	vo.Vote.ProposalId = res.Vote.ProposalId

	options := make([]WeightedVoteOption, len(res.Vote.Options))
	for j, opt := range res.Vote.Options {
		options[j] = WeightedVoteOption{
			Option: uint8(opt.Option), //nolint:gosec // G115
			Weight: opt.Weight,
		}
	}
	vo.Vote.Options = options
	return vo
}

// ParseDepositArgs parses the arguments for the Deposit query.
func ParseDepositArgs(args []interface{}) (*govv1.QueryDepositRequest, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	proposalID, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf(ErrInvalidProposalID, args[0])
	}

	depositor, ok := args[1].(common.Address)
	if !ok {
		return nil, fmt.Errorf(ErrInvalidDepositor, args[1])
	}

	depositorAccAddr := sdk.AccAddress(depositor.Bytes())
	return &govv1.QueryDepositRequest{
		ProposalId: proposalID,
		Depositor:  depositorAccAddr.String(),
	}, nil
}

// ParseDepositsArgs parses the arguments for the Deposits query.
func ParseDepositsArgs(method *abi.Method, args []interface{}) (*govv1.QueryDepositsRequest, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 2, len(args))
	}

	var input DepositsInput
	if err := method.Inputs.Copy(&input, args); err != nil {
		return nil, fmt.Errorf("error while unpacking args to DepositsInput: %s", err)
	}

	return &govv1.QueryDepositsRequest{
		ProposalId: input.ProposalId,
		Pagination: &input.Pagination,
	}, nil
}

// ParseTallyResultArgs parses the arguments for the TallyResult query.
func ParseTallyResultArgs(args []interface{}) (*govv1.QueryTallyResultRequest, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	proposalID, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf(ErrInvalidProposalID, args[0])
	}

	return &govv1.QueryTallyResultRequest{
		ProposalId: proposalID,
	}, nil
}

// Define a struct for parsing the assets
type ParsedAsset struct {
	Denom           string `json:"denom"`
	ContractAddress string `json:"contractAddress"`
	ChainId         string `json:"chainId"`
	Decimals        uint64 `json:"decimals"`
	BaseWeight      uint64 `json:"baseWeight"`
	Metadata        string `json:"metadata"`
}

func ParseAddNewAssetProposalArgs(args []interface{}) (*types.AddNewAssetConsensusProposal, error) {
	// Validate the number of arguments; the method expects exactly 3.
	if len(args) != 4 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	// Extract the title argument and ensure it is a non-empty string.
	title, ok := args[0].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("invalid title argument: %v", args[0])
	}

	// Extract the description argument and ensure it is a non-empty string.
	description, ok := args[1].(string)
	if !ok || description == "" {
		return nil, fmt.Errorf("invalid description argument: %v", args[1])
	}

	// Cast args[2] to a slice of interfaces
	rawAssets := reflect.ValueOf(args[2])
	if rawAssets.Kind() != reflect.Slice || rawAssets.Len() == 0 {
		return nil, fmt.Errorf("invalid or empty assets argument: %v", args[2])
	}

	// Convert each element of rawAssets into ParsedAsset and validate
	protoAssets := make([]*types.Asset, rawAssets.Len())
	for i := 0; i < rawAssets.Len(); i++ {
		rawAsset := rawAssets.Index(i).Interface()

		// Ensure the rawAsset is a struct and extract its fields
		assetStruct, ok := rawAsset.(struct {
			Denom           string `json:"denom"`
			ContractAddress string `json:"contractAddress"`
			ChainId         string `json:"chainId"`
			Decimals        uint32 `json:"decimals"`
			BaseWeight      uint64 `json:"baseWeight"`
			Metadata        string `json:"metadata"`
		})
		if !ok {
			return nil, fmt.Errorf("invalid asset structure at index %d: %v", i, rawAsset)
		}

		// Validate fields in the struct.
		if assetStruct.Denom == "" {
			return nil, fmt.Errorf("invalid denom for asset at index %d: %+v", i, assetStruct)
		}
		if assetStruct.ContractAddress == "" {
			return nil, fmt.Errorf("invalid contractAddress for asset at index %d: %+v", i, assetStruct)
		}
		if assetStruct.ChainId == "" {
			return nil, fmt.Errorf("invalid chainId for asset at index %d: %+v", i, assetStruct)
		}
		if assetStruct.Decimals == 0 {
			return nil, fmt.Errorf("invalid decimals for asset at index %d: %+v", i, assetStruct)
		}
		if assetStruct.BaseWeight == 0 {
			return nil, fmt.Errorf("invalid baseWeight for asset at index %d: %+v", i, assetStruct)
		}
		// Add the validated asset to the list.
		protoAssets[i] = &types.Asset{
			Denom:           assetStruct.Denom,
			ContractAddress: assetStruct.ContractAddress,
			ChainId:         assetStruct.ChainId,
			Decimals:        uint64(assetStruct.Decimals),
			BaseWeight:      assetStruct.BaseWeight,
			Metadata:        assetStruct.Metadata,
		}
	}

	initialDeposit, ok := args[3].(*big.Int)
	if !ok || initialDeposit == nil || initialDeposit.Sign() < 0 {
		return nil, fmt.Errorf("invalid or missing initialDeposit argument: %v", args[3])
	}

	// Vérifie si la valeur tient dans un uint64
	if initialDeposit.BitLen() > 64 {
		return nil, fmt.Errorf("initialDeposit value out of range for uint64: %v", initialDeposit)
	}

	// Construct and return the AddNewAssetConsensusProposal object.
	return &types.AddNewAssetConsensusProposal{
		Title:          title,
		Description:    description,
		Assets:         protoAssets,
		InitialDeposit: initialDeposit.Uint64(),
	}, nil
}

func ParseUpdateAssetProposalArgs(args []interface{}) (*types.UpdateAssetConsensusProposal, error) {
	// Validate the number of arguments; the method expects exactly 3.
	if len(args) != 4 {
		return nil, fmt.Errorf("invalid number of arguments, expected 3, got %d", len(args))
	}

	// Extract the title argument and ensure it is a non-empty string.
	title, ok := args[0].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("invalid title argument: %v", args[0])
	}

	// Extract the description argument and ensure it is a non-empty string.
	description, ok := args[1].(string)
	if !ok || description == "" {
		return nil, fmt.Errorf("invalid description argument: %v", args[1])
	}

	// Extract the updates array from args[2].
	rawUpdates := reflect.ValueOf(args[2])
	if rawUpdates.Kind() != reflect.Slice || rawUpdates.Len() == 0 {
		return nil, fmt.Errorf("invalid or empty updates argument: %v", args[2])
	}

	// Convert each element of rawUpdates into WeightUpdateData and validate.
	parsedUpdates := make([]*types.WeightUpdate, rawUpdates.Len())
	for i := 0; i < rawUpdates.Len(); i++ {
		rawUpdate := rawUpdates.Index(i).Interface()

		// Ensure the rawUpdate is a struct and extract its fields.
		updateStruct, ok := rawUpdate.(struct {
			Denom     string `json:"denom"`
			Magnitude string `json:"magnitude"`
			Direction string `json:"direction"`
		})
		if !ok {
			return nil, fmt.Errorf("invalid update structure at index %d: %v", i, rawUpdate)
		}

		// Validate fields in the struct.
		if updateStruct.Denom == "" {
			return nil, fmt.Errorf("invalid denom for update at index %d: %+v", i, updateStruct)
		}
		if updateStruct.Magnitude == "" {
			return nil, fmt.Errorf("invalid magnitude for update at index %d: %+v", i, updateStruct)
		}
		if updateStruct.Direction != "up" && updateStruct.Direction != "down" {
			return nil, fmt.Errorf("invalid direction for update at index %d: %+v", i, updateStruct)
		}

		// Add the validated update to the list.
		parsedUpdates[i] = &types.WeightUpdate{
			Denom:     updateStruct.Denom,
			Magnitude: updateStruct.Magnitude,
			Direction: updateStruct.Direction,
		}
	}

	initialDeposit, ok := args[3].(*big.Int)
	if !ok || initialDeposit == nil || initialDeposit.Sign() < 0 {
		return nil, fmt.Errorf("invalid or missing initialDeposit argument: %v", args[3])
	}

	// Vérifie si la valeur tient dans un uint64
	if initialDeposit.BitLen() > 64 {
		return nil, fmt.Errorf("initialDeposit value out of range for uint64: %v", initialDeposit)
	}

	// Construct and return the UpdateAssetConsensusProposal object.
	return &types.UpdateAssetConsensusProposal{
		Title:          title,
		Description:    description,
		Updates:        parsedUpdates,
		InitialDeposit: initialDeposit.Uint64(),
	}, nil
}

func ParseRemoveAssetProposalArgs(args []interface{}) (*types.RemoveAssetConsensusProposal, error) {
	// Validate the number of arguments; the function expects exactly 4.
	// Arguments: title (string), description (string), denoms ([]string), initialDeposit (*big.Int).
	if len(args) != 4 {
		return nil, fmt.Errorf("invalid number of arguments, expected 4, got %d", len(args))
	}

	// Extract and validate the title (must be a non-empty string).
	title, ok := args[0].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("invalid title argument: %v", args[0])
	}

	// Extract and validate the description (must be a non-empty string).
	description, ok := args[1].(string)
	if !ok || description == "" {
		return nil, fmt.Errorf("invalid description argument: %v", args[1])
	}

	// Extract and validate the denoms (must be a non-empty slice of strings).
	rawDenoms := reflect.ValueOf(args[2])
	if rawDenoms.Kind() != reflect.Slice || rawDenoms.Len() == 0 {
		return nil, fmt.Errorf("invalid or empty denoms argument: %v", args[2])
	}

	// Convert denoms into a slice of strings and validate each entry.
	parsedDenoms := make([]string, rawDenoms.Len())
	for i := 0; i < rawDenoms.Len(); i++ {
		denom, ok := rawDenoms.Index(i).Interface().(string)
		if !ok || denom == "" {
			return nil, fmt.Errorf("invalid denom at index %d: %v", i, rawDenoms.Index(i))
		}
		parsedDenoms[i] = denom
	}

	// Extract and validate the initialDeposit (must be a non-negative *big.Int).
	initialDeposit, ok := args[3].(*big.Int)
	if !ok || initialDeposit == nil || initialDeposit.Sign() < 0 {
		return nil, fmt.Errorf("invalid or missing initialDeposit argument: %v", args[3])
	}

	// Ensure the initialDeposit value fits within a uint64.
	if initialDeposit.BitLen() > 64 {
		return nil, fmt.Errorf("initialDeposit value out of range for uint64: %v", initialDeposit)
	}

	// Construct and return the RemoveAssetConsensusProposal object.
	return &types.RemoveAssetConsensusProposal{
		Title:          title,
		Description:    description,
		Denoms:         parsedDenoms,
		InitialDeposit: initialDeposit.Uint64(),
	}, nil
}

func (do *DepositOutput) FromResponse(res *govv1.QueryDepositResponse) *DepositOutput {
	hexDepositor, err := utils.Bech32ToHexAddr(res.Deposit.Depositor)
	if err != nil {
		return nil
	}
	coins := make([]cmn.Coin, len(res.Deposit.Amount))
	for i, c := range res.Deposit.Amount {
		coins[i] = cmn.Coin{
			Denom:  c.Denom,
			Amount: c.Amount.BigInt(),
		}
	}
	do.Deposit = DepositData{
		ProposalId: res.Deposit.ProposalId,
		Depositor:  hexDepositor,
		Amount:     coins,
	}
	return do
}

func (do *DepositsOutput) FromResponse(res *govv1.QueryDepositsResponse) *DepositsOutput {
	do.Deposits = make([]DepositData, len(res.Deposits))
	for i, d := range res.Deposits {
		hexDepositor, err := utils.Bech32ToHexAddr(d.Depositor)
		if err != nil {
			return nil
		}
		coins := make([]cmn.Coin, len(d.Amount))
		for j, c := range d.Amount {
			coins[j] = cmn.Coin{
				Denom:  c.Denom,
				Amount: c.Amount.BigInt(),
			}
		}
		do.Deposits[i] = DepositData{
			ProposalId: d.ProposalId,
			Depositor:  hexDepositor,
			Amount:     coins,
		}
	}
	if res.Pagination != nil {
		do.PageResponse = query.PageResponse{
			NextKey: res.Pagination.NextKey,
			Total:   res.Pagination.Total,
		}
	}
	return do
}

func (tro *TallyResultOutput) FromResponse(res *govv1.QueryTallyResultResponse) *TallyResultOutput {
	tro.TallyResult = TallyResultData{
		Yes:        res.Tally.YesCount,
		Abstain:    res.Tally.AbstainCount,
		No:         res.Tally.NoCount,
		NoWithVeto: res.Tally.NoWithVetoCount,
	}
	return tro
}

// UpdateParamsProposalArgs holds the arguments for the UpdateParamsProposal method
type UpdateParamsProposalArgs struct {
	Title          string
	Description    string
	MaxGas         int64
	MaxBytes       int64
	InitialDeposit *big.Int
}

// parseUpdateParamsProposalArgs parses the arguments for the UpdateParamsProposal method
func (p *Precompile) parseUpdateBlockParamsArgs(args []interface{}) (*UpdateParamsProposalArgs, error) {
	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	// Extract the title argument and ensure it is a non-empty string.
	title, ok := args[0].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("invalid title argument: %v", args[0])
	}

	// Extract the description argument and ensure it is a non-empty string.
	description, ok := args[1].(string)
	if !ok || description == "" {
		return nil, fmt.Errorf("invalid description argument: %v", args[1])
	}

	maxGas, ok := args[2].(int64)
	if !ok || maxGas < 0 {
		return nil, fmt.Errorf("invalid maxGas: %v", maxGas)
	}

	maxBytes, ok := args[3].(int64)
	if !ok || maxBytes < 0 {
		return nil, fmt.Errorf("invalid maxBytes: %v", maxBytes)
	}

	initialDeposit, ok := args[4].(*big.Int)
	if !ok || initialDeposit == nil || initialDeposit.Sign() < 0 {
		return nil, fmt.Errorf("invalid or missing initialDeposit argument: %v", args[4])
	}

	return &UpdateParamsProposalArgs{
		Title:          title,
		Description:    description,
		MaxGas:         maxGas,
		MaxBytes:       maxBytes,
		InitialDeposit: initialDeposit,
	}, nil
}
