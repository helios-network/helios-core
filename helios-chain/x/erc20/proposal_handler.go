package erc20

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"
)

// NewErc20ProposalHandler creates a governance handler to manage new erc20 proposal types.
func NewErc20ProposalHandler(k keeper.Keeper) govtypes.Handler {

	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.RemoveAssetConsensusProposal:
			return HandleRemoveAssetConsensusProposal(ctx, k, c)
		case *types.AddNewAssetConsensusProposal:
			return handleAddNewAssetConsensusProposal(ctx, k, c)
		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized erc20 proposal content type: %T", c)
		}
	}
}

func handleAddNewAssetConsensusProposal(ctx sdk.Context, k keeper.Keeper, p *types.AddNewAssetConsensusProposal) error {
	// Validate the proposal
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	// Iterate over the assets in the proposal and add them to the consensus whitelist
	for _, asset := range p.Assets {
		if err := k.AddAssetToConsensusWhitelist(ctx, *asset); err != nil {
			return errors.Wrapf(err, "failed to add asset %s to consensus whitelist", asset.Denom)
		}
	}

	return nil
}

func HandleRemoveAssetConsensusProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.RemoveAssetConsensusProposal) error {
	for _, denom := range proposal.Denoms {
		if !k.IsAssetWhitelisted(ctx, denom) {
			return errors.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted", denom)
		}

		// Remove asset from whitelist
		err := k.RemoveAssetFromConsensusWhitelist(ctx, denom)
		if err != nil {
			return err
		}
	}
	return nil
}
