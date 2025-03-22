package gov

import (
	"fmt"

	"cosmossdk.io/math"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/erc20/types"
	"helios-core/helios-chain/x/evm/core/vm"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
)

const (
	// VoteMethod defines the ABI method name for the gov Vote transaction.
	VoteMethod = "vote"
	// VoteWeightedMethod defines the ABI method name for the gov VoteWeighted transaction.
	VoteWeightedMethod = "voteWeighted"
	// AddNewAssetProposalMethod defines the method name for add new proposal
	AddNewAssetProposalMethod = "addNewAssetProposal"
	// UpdateAssetProposalMethod defines the method name for add new proposal
	UpdateAssetProposalMethod = "updateAssetProposal"
	// RemoveAssetProposalMethod defines the method name for add new proposal
	RemoveAssetProposalMethod = "removeAssetProposal"
	// UpdateBlockParamsProposalMethod defines the method name for updating consensus parameters
	UpdateBlockParamsProposalMethod = "updateBlockParamsProposal"
)

// Vote defines a method to add a vote on a specific proposal.
func (p Precompile) Vote(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, voterHexAddr, err := NewMsgVote(args)
	if err != nil {
		return nil, err
	}

	// If the contract is the voter, we don't need an origin check
	// Otherwise check if the origin matches the voter address
	isContractVoter := contract.CallerAddress == voterHexAddr && contract.CallerAddress != origin
	if !isContractVoter && origin != voterHexAddr {
		return nil, fmt.Errorf(ErrDifferentOrigin, origin.String(), voterHexAddr.String())
	}

	msgSrv := govkeeper.NewMsgServerImpl(&p.govKeeper)

	if _, err = msgSrv.Vote(ctx, msg); err != nil {
		return nil, err
	}

	if err = p.EmitVoteEvent(ctx, stateDB, voterHexAddr, msg.ProposalId, int32(msg.Option)); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

// VoteWeighted defines a method to add a vote on a specific proposal.
func (p Precompile) VoteWeighted(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	msg, voterHexAddr, options, err := NewMsgVoteWeighted(method, args)
	if err != nil {
		return nil, err
	}

	// If the contract is the voter, we don't need an origin check
	// Otherwise check if the origin matches the voter address
	isContractVoter := contract.CallerAddress == voterHexAddr && contract.CallerAddress != origin
	if !isContractVoter && origin != voterHexAddr {
		return nil, fmt.Errorf(ErrDifferentOrigin, origin.String(), voterHexAddr.String())
	}

	msgSrv := govkeeper.NewMsgServerImpl(&p.govKeeper)
	if _, err = msgSrv.VoteWeighted(ctx, msg); err != nil {
		return nil, err
	}

	if err = p.EmitVoteWeightedEvent(ctx, stateDB, voterHexAddr, msg.ProposalId, options); err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p *Precompile) AddNewAssetProposal(
	origin common.Address,
	govKeeper govkeeper.Keeper,
	ctx sdk.Context,
	method *abi.Method,
	_ *vm.Contract,
	args []interface{},
) ([]byte, error) {
	// Parse arguments into the AddNewAssetConsensusProposal type.
	addNewAssetProposalReq, err := ParseAddNewAssetProposalArgs(args)
	if err != nil {
		return nil, fmt.Errorf("failed to parse addNewAssetProposal arguments: %w", err)
	}

	// check if baseWeight is well superior of zero
	for _, asset := range addNewAssetProposalReq.Assets {
		if asset.BaseWeight == uint64(0) {
			return nil, fmt.Errorf("failed criterial BaseWeight of %s can't be equals to zero.", asset.Denom)
		}
	}

	proposer := sdk.AccAddress(origin.Bytes())

	proposalContent := &types.AddNewAssetConsensusProposal{
		Title:       addNewAssetProposalReq.Title,
		Description: addNewAssetProposalReq.Description,
		Assets:      addNewAssetProposalReq.Assets,
	}

	contentMsg, err := v1.NewLegacyContent(proposalContent, govKeeper.GetAuthority()) // todo : recheck here
	if err != nil {
		return nil, fmt.Errorf("error converting legacy content into proposal message: %w", err)
	}

	// Convert sdk.Msg to *types.Any
	contentAny, err := codectypes.NewAnyWithValue(contentMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to pack content message: %w", err)
	}

	msg := &v1.MsgSubmitProposal{
		Messages: []*codectypes.Any{contentAny},
		InitialDeposit: sdk.NewCoins(
			sdk.NewCoin("ahelios", math.NewInt(int64(addNewAssetProposalReq.InitialDeposit))), // todo: change ahelios by default var
		),
		Proposer: proposer.String(),
		Metadata: "Optional metadata", // todo update !!
		Title:    addNewAssetProposalReq.Title,
		Summary:  addNewAssetProposalReq.Description,
	}

	msgSrv := govkeeper.NewMsgServerImpl(&p.govKeeper)
	proposal, err := msgSrv.SubmitProposal(ctx, msg)
	if err != nil {
		// Log the error or handle it in a specific way
		fmt.Printf("Failed to submit proposal: %v\n", err)
		return nil, err
	}

	fmt.Println("proposalId: ", proposal.ProposalId)
	//TODO: update weight erc20
	// Pack and return a success response with the proposal ID
	return method.Outputs.Pack(proposal.ProposalId)
}

func (p *Precompile) UpdateAssetProposal(
	origin common.Address,
	govKeeper govkeeper.Keeper,
	ctx sdk.Context,
	method *abi.Method,
	_ *vm.Contract,
	args []interface{},
) ([]byte, error) {
	// Parse arguments into the AddNewAssetConsensusProposal type.
	updateProposalReq, err := ParseUpdateAssetProposalArgs(args)

	if err != nil {
		return nil, fmt.Errorf("failed to parse addNewAssetProposal arguments: %w", err)
	}

	proposer := sdk.AccAddress(origin.Bytes())

	proposalContent := &types.UpdateAssetConsensusProposal{
		Title:       updateProposalReq.Title,
		Description: updateProposalReq.Description,
		Updates:     updateProposalReq.Updates,
	}

	contentMsg, err := v1.NewLegacyContent(proposalContent, govKeeper.GetAuthority()) // todo : recheck here
	if err != nil {
		return nil, fmt.Errorf("error converting legacy content into proposal message: %w", err)
	}

	// Convert sdk.Msg to *types.Any
	contentAny, err := codectypes.NewAnyWithValue(contentMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to pack content message: %w", err)
	}

	msg := &v1.MsgSubmitProposal{
		Messages: []*codectypes.Any{contentAny},
		InitialDeposit: sdk.NewCoins(
			sdk.NewCoin("ahelios", math.NewInt(int64(updateProposalReq.InitialDeposit))), // todo: change ahelios by default var
		),
		Proposer: proposer.String(),
		Metadata: "Optional metadata", // todo update !!
		Title:    updateProposalReq.Title,
		Summary:  updateProposalReq.Description,
	}

	msgSrv := govkeeper.NewMsgServerImpl(&p.govKeeper)
	proposal, err := msgSrv.SubmitProposal(ctx, msg)
	if err != nil {
		// Log the error or handle it in a specific way
		fmt.Printf("Failed to submit proposal: %v\n", err)
		return nil, err
	}
	// YAMI -> :
	//TODO: check denom is whitelisted
	//TODO: check BaseWeight not equals to 1 if direction is down (Already checked in simulation but should be better to have also here).
	// Pack and return a success response with the proposal ID
	return method.Outputs.Pack(proposal.ProposalId)
}

func (p *Precompile) RemoveAssetProposal(
	origin common.Address,
	govKeeper govkeeper.Keeper,
	ctx sdk.Context,
	method *abi.Method,
	_ *vm.Contract,
	args []interface{},
) ([]byte, error) {
	// Parse arguments into the AddNewAssetConsensusProposal type.
	removeProposalReq, err := ParseRemoveAssetProposalArgs(args)

	if err != nil {
		return nil, fmt.Errorf("failed to parse addNewAssetProposal arguments: %w", err)
	}

	proposer := sdk.AccAddress(origin.Bytes())

	proposalContent := &types.RemoveAssetConsensusProposal{
		Title:          removeProposalReq.Title,
		Description:    removeProposalReq.Description,
		Denoms:         removeProposalReq.Denoms,
		InitialDeposit: removeProposalReq.InitialDeposit,
	}

	contentMsg, err := v1.NewLegacyContent(proposalContent, govKeeper.GetAuthority()) // todo : recheck here
	if err != nil {
		return nil, fmt.Errorf("error converting legacy content into proposal message: %w", err)
	}

	// Convert sdk.Msg to *types.Any
	contentAny, err := codectypes.NewAnyWithValue(contentMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to pack content message: %w", err)
	}

	msg := &v1.MsgSubmitProposal{
		Messages: []*codectypes.Any{contentAny},
		InitialDeposit: sdk.NewCoins(
			sdk.NewCoin("ahelios", math.NewInt(int64(proposalContent.InitialDeposit))), // todo: change ahelios by default var
		),
		Proposer: proposer.String(),
		Metadata: "Optional metadata", // todo update !!
		Title:    proposalContent.Title,
		Summary:  proposalContent.Description,
	}

	msgSrv := govkeeper.NewMsgServerImpl(&p.govKeeper)
	proposal, err := msgSrv.SubmitProposal(ctx, msg)
	if err != nil {
		// Log the error or handle it in a specific way
		fmt.Printf("Failed to submit proposal: %v\n", err)
		return nil, err
	}

	//TODO: update weight erc20

	// Pack and return a success response with the proposal ID
	return method.Outputs.Pack(proposal.ProposalId)
}

// UpdateBlockParamsProposal submits a proposal to update block parameters
func (p *Precompile) UpdateBlockParamsProposal(
	ctx sdk.Context,
	origin common.Address,
	govKeeper govkeeper.Keeper,
	_ *vm.Contract,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {
	updateArgs, err := p.parseUpdateBlockParamsArgs(args)
	if err != nil {
		return nil, err
	}

	proposer := sdk.AccAddress(origin.Bytes())

	// Create the consensus params update message
	updateParamsMsg := &consensustypes.MsgUpdateParams{
		Authority: govKeeper.GetAuthority(),
		Block: &tmproto.BlockParams{
			MaxBytes: updateArgs.MaxBytes,
			MaxGas:   updateArgs.MaxGas,
		},
		// Keep existing values for other parameters
		Evidence:  ctx.ConsensusParams().Evidence,
		Validator: ctx.ConsensusParams().Validator,
		Abci:      ctx.ConsensusParams().Abci,
	}

	// Pack the update params message into Any type
	msgAny, err := codectypes.NewAnyWithValue(updateParamsMsg)
	if err != nil {
		return nil, err
	}

	// Create the proposal message
	msg := &v1.MsgSubmitProposal{
		Messages: []*codectypes.Any{msgAny},
		InitialDeposit: sdk.NewCoins(
			sdk.NewCoin("ahelios", math.NewInt(updateArgs.InitialDeposit.Int64())),
		),
		Proposer: proposer.String(),
		Title:    updateArgs.Title,
		Summary:  updateArgs.Description,
	}

	// Submit the proposal
	msgServer := govkeeper.NewMsgServerImpl(&govKeeper)
	res, err := msgServer.SubmitProposal(ctx, msg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.ProposalId)
}
