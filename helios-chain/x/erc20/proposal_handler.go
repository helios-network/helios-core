package erc20

import (
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
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
			return HandleAddNewAssetConsensusProposal(ctx, k, c)
		case *types.UpdateAssetConsensusProposal:
			return HandleUpdateAssetConsensusProposal(ctx, k, c)
		default:
			return errors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized erc20 proposal content type: %T", c)
		}
	}
}
func HandleAddNewAssetConsensusProposal(ctx sdk.Context, k keeper.Keeper, p *types.AddNewAssetConsensusProposal) error {
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

	// TODO: Check the min avg score of the sender to ensure no useless proposals

	// Iterate over the updates in the proposal
	for _, update := range proposal.Updates {
		// Validate and retrieve the asset
		if !k.IsAssetWhitelisted(ctx, update.Denom) {
			return errors.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted", update.Denom)
		}

		asset, err := k.GetAssetFromWhitelist(ctx, update.Denom)
		if err != nil {
			return errors.Wrapf(err, "failed to retrieve asset %s from whitelist", update.Denom)
		}

		// Determine the adjustment factor
		percentFactor, adjustmentFactor, err := getAdjustmentFactors(update.Magnitude)
		if err != nil {
			return err
		}

		// Apply the adjustment to the asset weight
		updatedAsset, increaseWeight, err := applyWeightAdjustment(asset, update.Direction, adjustmentFactor)
		if err != nil {
			return err
		}

		// Update the asset in the whitelist
		if err := k.UpdateAssetInConsensusWhitelist(ctx, updatedAsset); err != nil {
			return errors.Wrapf(err, "failed to update asset %s in whitelist", update.Denom)
		}

		// Update delegation stakes with the new weighted amount
		if err := k.UpdateAssetNativeSharesWeight(ctx, update.Denom, percentFactor, increaseWeight); err != nil {
			return errors.Wrapf(err, "failed to update native delegation shares weight: %s", err)
		}
	}

	return nil
}

// Helper to determine adjustment factors based on magnitude
func getAdjustmentFactors(magnitude string) (math.LegacyDec, float64, error) {
	switch magnitude {
	case "small":
		return math.LegacyMustNewDecFromStr("0.05"), 0.05, nil
	case "medium":
		return math.LegacyMustNewDecFromStr("0.15"), 0.15, nil
	case "high":
		return math.LegacyMustNewDecFromStr("0.30"), 0.30, nil
	default:
		return math.LegacyDec{}, 0, errors.Wrapf(types.ErrInvalidLengthQuery, "invalid magnitude: %s", magnitude)
	}
}

// Helper to apply weight adjustment based on direction
func applyWeightAdjustment(asset types.Asset, direction string, adjustmentFactor float64) (types.Asset, bool, error) {
	increaseWeight := false
	switch direction {
	case "up":
		asset.BaseWeight += uint64(float64(asset.BaseWeight) * adjustmentFactor)
		increaseWeight = true
	case "down":
		asset.BaseWeight -= uint64(float64(asset.BaseWeight) * adjustmentFactor)
	default:
		return asset, false, errors.Wrapf(types.ErrInvalidLengthQuery, "invalid direction: %s", direction)
	}
	return asset, increaseWeight, nil
}
