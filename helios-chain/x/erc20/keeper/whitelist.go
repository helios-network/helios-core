package keeper

import (
	"fmt"
	"helios-core/helios-chain/x/erc20/types"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AddAssetToConsensusWhitelist adds an asset to the consensus whitelist or unarchives an existing one
func (k Keeper) AddAssetToConsensusWhitelist(ctx sdk.Context, asset types.Asset) error {
	// Check if the asset already exists
	existingAsset, err := k.GetAssetFromWhitelist(ctx, asset.Denom)
	if err == nil {
		// Asset exists
		if !existingAsset.Archived {
			// Asset exists and is not archived, return error
			return errorsmod.Wrapf(types.ErrAssetAlreadyWhitelisted, "asset %s is already whitelisted and active", asset.Denom)
		}
		// Asset exists and is archived, unarchive it
		return k.UnarchiveAssetInConsensusWhitelist(ctx, asset.Denom, asset.BaseWeight)
	} else if _, ok := err.(types.AssetNotFoundError); !ok {
		// An unexpected error occurred when checking for the asset
		return errorsmod.Wrapf(err, "failed to check existence of asset %s", asset.Denom)
	}

	// making sure the asset is not archived by default received call from the proposal handler
	asset.Archived = false

	// Asset does not exist, proceed to add it
	store := k.GetStore(ctx)
	assetKey := types.GetAssetKey(asset.Denom)
	bz := k.cdc.MustMarshal(&asset) // New assets are never archived initially
	store.Set(assetKey, bz)

	return nil
}

// RemoveAssetFromConsensusWhitelist archives an asset instead of completely removing it
// This ensures users can still undelegate any staked amount of this asset
func (k Keeper) RemoveAssetFromConsensusWhitelist(ctx sdk.Context, denom string) error {
	// Check if the asset exists and is not already archived
	asset, err := k.GetAssetFromWhitelist(ctx, denom)
	if err != nil {
		return err // Handles ErrAssetNotFound
	}

	if asset.Archived {
		// Asset is already archived, so it's already effectively removed
		return nil
	}

	// Archive the asset instead of removing it completely
	// This reduces its weight to 1 and marks it as archived
	return k.ArchiveAssetInConsensusWhitelist(ctx, denom)
}

// IsAssetWhitelisted checks if an asset is already in the whitelist (archived or not)
func (k Keeper) IsAssetWhitelisted(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetAssetKey(denom))
}

// GetAssetFromWhitelist retrieves an asset from the whitelist (archived or not)
func (k Keeper) GetAssetFromWhitelist(ctx sdk.Context, denom string) (types.Asset, error) {
	store := ctx.KVStore(k.storeKey)

	// Check if the asset exists
	assetKey := types.GetAssetKey(denom)
	if !store.Has(assetKey) {
		return types.Asset{}, types.AssetNotFoundError{Denom: denom}
	}

	// Retrieve and unmarshal the asset
	bz := store.Get(assetKey)
	var asset types.Asset
	k.cdc.MustUnmarshal(bz, &asset)

	return asset, nil
}

// ArchiveAssetInConsensusWhitelist marks an asset as archived, sets its weight to 1, and updates delegations.
func (k Keeper) ArchiveAssetInConsensusWhitelist(ctx sdk.Context, denom string) error {
	asset, err := k.GetAssetFromWhitelist(ctx, denom)
	if err != nil {
		return err // Handles ErrAssetNotFound
	}

	if asset.Archived {
		return errorsmod.Wrapf(types.ErrAssetAlreadyArchived, "asset %s is already archived", denom)
	}

	originalWeight := asset.BaseWeight
	if originalWeight <= 1 {
		// Cannot archive an asset with weight 1 or less, or already archived with weight 1.
		// Mark as archived just in case, but no weight change needed.
		asset.Archived = true
		asset.BaseWeight = 1 // Ensure weight is 1
		if err := k.UpdateAssetInConsensusWhitelist(ctx, asset); err != nil {
			return errorsmod.Wrapf(err, "failed to update asset %s during archival (weight <= 1)", denom)
		}
		// No need to call UpdateAssetNativeSharesWeight if weight was already <= 1
		return nil
	}

	// Update asset state
	asset.Archived = true
	asset.BaseWeight = 1

	// Save the updated asset state first
	if err := k.UpdateAssetInConsensusWhitelist(ctx, asset); err != nil {
		return errorsmod.Wrapf(err, "failed to update asset %s during archival", denom)
	}

	// Update delegation shares based on the weight change
	if err := k.UpdateAssetNativeSharesWeight(ctx, denom, math.LegacyNewDec(0), false, originalWeight); err != nil {
		return errorsmod.Wrapf(err, "failed to update native delegation shares weight during archival for asset %s", denom)
	}

	return nil
}

// UnarchiveAssetInConsensusWhitelist marks an asset as active, sets its new weight, and updates delegations.
func (k Keeper) UnarchiveAssetInConsensusWhitelist(ctx sdk.Context, denom string, newWeight uint64) error {
	asset, err := k.GetAssetFromWhitelist(ctx, denom)
	if err != nil {
		return err // Handles ErrAssetNotFound
	}

	if !asset.Archived {
		return errorsmod.Wrapf(types.ErrAssetNotArchived, "asset %s is not archived, cannot unarchive", denom)
	}

	if newWeight == 0 {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "cannot unarchive asset %s with zero weight", denom)
	}

	// The asset must currently have BaseWeight=1 as it's archived
	if asset.BaseWeight != 1 {
		// This indicates an inconsistent state, potentially log a warning
		k.Logger(ctx).Error("Archived asset has unexpected weight", "denom", denom, "weight", asset.BaseWeight)
		// Proceed to set the new weight anyway
	}

	// Calculate percentage increase: (newWeight - 1) / 1 = newWeight - 1
	// Use LegacyDec for precision
	decNewWeight := math.LegacyNewDecFromInt(math.NewIntFromUint64(newWeight))
	decOne := math.LegacyNewDec(1)
	percentageIncrease := decNewWeight.Sub(decOne)

	if percentageIncrease.IsNegative() {
		// Should not happen if newWeight >= 1 (newWeight=0 checked above)
		return fmt.Errorf("calculated negative percentage increase for asset %s during unarchival", denom)
	}

	// Update asset state
	asset.Archived = false
	asset.BaseWeight = newWeight

	// Save the updated asset state first
	if err := k.UpdateAssetInConsensusWhitelist(ctx, asset); err != nil {
		return errorsmod.Wrapf(err, "failed to update asset %s during unarchival", denom)
	}

	// If newWeight is 1, percentageIncrease is 0, so no update needed.
	if !percentageIncrease.IsZero() {
		// Update delegation shares based on the weight change
		if err := k.UpdateAssetNativeSharesWeight(ctx, denom, percentageIncrease, true, 0); err != nil {
			return errorsmod.Wrapf(err, "failed to update native delegation shares weight during unarchival for asset %s", denom)
		}
	}

	return nil
}

type assetAdapter struct {
	asset types.Asset
}

func (a assetAdapter) GetDenom() string {
	return a.asset.Denom
}

func (a assetAdapter) GetContractAddress() string {
	return a.asset.ContractAddress
}

func (a assetAdapter) GetBaseWeight() uint64 {
	return a.asset.BaseWeight
}

// ConvertAssetsToErc20Assets converts Asset types to Erc20Asset interface types
func ConvertAssetsToErc20Assets(assets []types.Asset) []stakingtypes.Erc20Asset {
	erc20Assets := make([]stakingtypes.Erc20Asset, 0, len(assets)) // Initialize with 0 length, capacity len(assets)
	for _, asset := range assets {
		// Explicitly check if the asset is archived before adding
		if !asset.Archived {
			erc20Assets = append(erc20Assets, assetAdapter{asset: asset})
		}
	}
	return erc20Assets
}

func (k Keeper) GetAllStakingAssets(ctx sdk.Context) []stakingtypes.Erc20Asset {
	allAssets := k.GetAllWhitelistedAssets(ctx)
	activeAssets := make([]types.Asset, 0, len(allAssets))
	for _, asset := range allAssets {
		if !asset.Archived {
			activeAssets = append(activeAssets, asset)
		}
	}
	return ConvertAssetsToErc20Assets(activeAssets) // Convert only active assets
}

// GetAllWhitelistedAssets retrieves all assets from the whitelist (including archived)
func (k Keeper) GetAllWhitelistedAssets(ctx sdk.Context) []types.Asset {
	// Wrap the store with the prefix to isolate whitelist entries
	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.WhitelistPrefix))
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	var assets []types.Asset
	for ; iterator.Valid(); iterator.Next() {
		var asset types.Asset
		k.cdc.MustUnmarshal(iterator.Value(), &asset)
		assets = append(assets, asset)
	}
	return assets
}

// UpdateAssetInConsensusWhitelist updates an existing asset in the consensus whitelist
func (k Keeper) UpdateAssetInConsensusWhitelist(ctx sdk.Context, asset types.Asset) error {
	store := k.GetStore(ctx)

	// Check if the asset is actually whitelisted (using IsAssetWhitelisted which checks presence)
	if !k.IsAssetWhitelisted(ctx, asset.Denom) {
		return errorsmod.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted, cannot update", asset.Denom)
	}

	// Marshal and store the updated asset
	assetKey := types.GetAssetKey(asset.Denom)
	bz := k.cdc.MustMarshal(&asset)
	store.Set(assetKey, bz)

	return nil
}

func (k Keeper) UpdateAssetNativeSharesWeight(ctx sdk.Context, denom string, percentage math.LegacyDec, increase bool, originalWeight uint64) error {
	// If percentage is zero, no update needed
	if percentage.IsZero() {
		return nil
	}

	// Ensure percentage is not negative
	if percentage.IsNegative() {
		return fmt.Errorf("invalid negative percentage provided for weight update: %s", percentage.String())
	}

	return k.stakingKeeper.UpdateAssetWeight(ctx, denom, percentage, increase, originalWeight)
}
