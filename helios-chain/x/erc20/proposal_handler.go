package erc20

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"

	"github.com/ethereum/go-ethereum/common"
)

// NewErc20ProposalHandler creates a governance handler to manage all erc20 proposal types.
func NewErc20ProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.RemoveAssetConsensusProposal:
			return HandleRemoveAssetConsensusProposal(ctx, k, c)
		case *types.AddNewAssetConsensusProposal:
			return handleAddNewAssetConsensusProposal(ctx, k, c)
		case *types.UpdateAssetConsensusProposal:
			return HandleUpdateAssetConsensusProposal(ctx, k, c)
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
		contractAddress := common.HexToAddress(asset.ContractAddress)
		exist, err := k.DoesERC20ContractExist(ctx, contractAddress)
		if err != nil {
			return errors.Wrapf(types.ErrAssetNotFound, "failed to check if ERC20 contract exists for asset %s: %v", asset.Denom, err)
		}
		if !exist {
			return errors.Wrapf(types.ErrAssetNotFound, "failed to add asset %s in consensus whitelist as the ERC20 contract does not exist", asset.Denom)
		}

		if err := k.AddAssetToConsensusWhitelist(ctx, *asset); err != nil {
			return errors.Wrapf(err, "failed to add asset %s to consensus whitelist", asset.Denom)
		}
	}

	return nil
}

func HandleRemoveAssetConsensusProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.RemoveAssetConsensusProposal) error {
	// Validate the proposal
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

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

func HandleUpdateAssetConsensusProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.UpdateAssetConsensusProposal) error {
	// Validate the proposal
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

	//TODO: Check the min avg score of the sender to make sure no useless proposals

	// Iterate over the updates in the proposal
	for _, update := range proposal.Updates {
		// Check if the asset is already whitelisted
		if !k.IsAssetWhitelisted(ctx, update.Denom) {
			return errors.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted", update.Denom)
		}

		// Retrieve the asset from the whitelist
		asset, err := k.GetAssetFromWhitelist(ctx, update.Denom)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve asset %s from whitelist", update.Denom)
		}

		// Determine the adjustment factor based on the magnitude
		var adjustmentFactor float64
		switch update.Magnitude {
		case "small":
			adjustmentFactor = 0.05
		case "medium":
			adjustmentFactor = 0.15
		case "high":
			adjustmentFactor = 0.30
		default:
			return errors.Wrapf(types.ErrInvalidLengthQuery, "invalid magnitude: %s", update.Magnitude)
		}

		// Apply the adjustment
		if update.Direction == "up" {
			asset.BaseWeight += uint64(float64(asset.BaseWeight) * adjustmentFactor)
		} else if update.Direction == "down" {
			asset.BaseWeight -= uint64(float64(asset.BaseWeight) * adjustmentFactor)
		} else {
			return errors.Wrapf(types.ErrInvalidLengthQuery, "invalid direction: %s", update.Direction)
		}

		// Update the asset in the whitelist
		if err := k.UpdateAssetInConsensusWhitelist(ctx, asset); err != nil {
			return errors.Wrapf(err, "failed to update asset %s in whitelist", update.Denom)
		}
	}

	return nil
}
