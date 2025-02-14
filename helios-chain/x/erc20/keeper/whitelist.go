package keeper

import (
	"helios-core/helios-chain/x/erc20/types"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// AddAssetToConsensusWhitelist adds an asset to the consensus whitelist
func (k Keeper) AddAssetToConsensusWhitelist(ctx sdk.Context, asset types.Asset) error {
	store := k.GetStore(ctx)

	// Check if the asset is already whitelisted
	if k.IsAssetWhitelisted(ctx, asset.Denom) {
		return errors.Wrapf(types.ErrAssetAlreadyWhitelisted, "asset %s is already whitelisted", asset.Denom)
	}

	// Marshal and store the asset in the whitelist
	assetKey := types.GetAssetKey(asset.Denom)
	bz := k.cdc.MustMarshal(&asset)
	store.Set(assetKey, bz)

	return nil
}

// IsAssetWhitelisted checks if an asset is already in the whitelist
func (k Keeper) IsAssetWhitelisted(ctx sdk.Context, denom string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.GetAssetKey(denom))
}

// GetAssetFromWhitelist retrieves an asset from the whitelist
func (k Keeper) GetAssetFromWhitelist(ctx sdk.Context, denom string) (types.Asset, error) {
	store := ctx.KVStore(k.storeKey)

	// Check if the asset exists
	if !k.IsAssetWhitelisted(ctx, denom) {
		return types.Asset{}, errors.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted", denom)
	}

	// Retrieve and unmarshal the asset
	bz := store.Get(types.GetAssetKey(denom))
	var asset types.Asset
	k.cdc.MustUnmarshal(bz, &asset)

	return asset, nil
}

// RemoveAssetFromConsensusWhitelist removes an asset from the whitelist
func (k Keeper) RemoveAssetFromConsensusWhitelist(ctx sdk.Context, denom string) error {
	store := ctx.KVStore(k.storeKey)

	// Check if the asset exists
	if !k.IsAssetWhitelisted(ctx, denom) {
		return errors.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted", denom)
	}

	// Delete the asset from the store
	store.Delete(types.GetAssetKey(denom))
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

func ConvertAssetsToErc20Assets(assets []types.Asset) []stakingtypes.Erc20Asset {
	erc20Assets := make([]stakingtypes.Erc20Asset, len(assets))
	for i, asset := range assets {
		erc20Assets[i] = assetAdapter{asset: asset}
	}
	return erc20Assets
}

func (k Keeper) GetAllStakingAssets(ctx sdk.Context) []stakingtypes.Erc20Asset {
	erc20Assets := k.GetAllWhitelistedAssets(ctx)
	return ConvertAssetsToErc20Assets(erc20Assets)
}

// GetAllWhitelistedAssets retrieves all assets from the whitelist
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

	// Check if the asset is already whitelisted
	if !k.IsAssetWhitelisted(ctx, asset.Denom) {
		return errors.Wrapf(types.ErrAssetNotFound, "asset %s is not whitelisted", asset.Denom)
	}

	// Marshal and store the updated asset
	assetKey := types.GetAssetKey(asset.Denom)
	bz := k.cdc.MustMarshal(&asset)
	store.Set(assetKey, bz)

	return nil
}

func (k Keeper) UpdateAssetNativeSharesWeight(ctx sdk.Context, denom string, percentage math.LegacyDec, increase bool) error {
	return k.stakingKeeper.UpdateAssetWeight(ctx, denom, percentage, increase)
}
