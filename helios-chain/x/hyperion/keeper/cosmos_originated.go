package keeper

import (
	"encoding/json"

	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

func (k *Keeper) GetCosmosOriginatedDenom(ctx sdk.Context, tokenContract common.Address) (string, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetERC20ToCosmosDenomKey(tokenContract))
	if bz == nil {
		return "", false
	}

	return string(bz), true
}

func (k *Keeper) GetCosmosOriginatedERC20(ctx sdk.Context, denom string) (common.Address, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetCosmosDenomToERC20Key(denom))
	if bz == nil {
		return common.Address{}, false
	}

	return common.BytesToAddress(bz), true
}

func (k *Keeper) SetCosmosOriginatedDenomToERC20(ctx sdk.Context, denom string, tokenContract common.Address) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetCosmosDenomToERC20Key(denom), tokenContract.Bytes())
	store.Set(types.GetERC20ToCosmosDenomKey(tokenContract), []byte(denom))
}

// DenomToERC20 returns if an asset is native to Cosmos or Ethereum, and get its corresponding ERC20 address
// This will return an error if it cant parse the denom as a hyperion denom, and then also can't find the denom
// in an index of ERC20 contracts deployed on Ethereum to serve as synthetic Cosmos assets.
func (k *Keeper) DenomToERC20Lookup(ctx sdk.Context, denomStr string, hyperionId uint64) (isCosmosOriginated bool, tokenContract common.Address, err error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// First try parsing the ERC20 out of the denom
	hyperionDenom, denomErr := types.NewHyperionDenomFromString(hyperionId, denomStr)
	if denomErr == nil {
		// This is an Ethereum-originated asset
		tokenContractFromDenom, _ := hyperionDenom.TokenContract()
		return false, tokenContractFromDenom, nil
	}

	// If denom is native cosmos coin denom, return Cosmos coin ERC20 contract address from Params
	if denomStr == k.GetCosmosCoinDenom(ctx)[hyperionId] {
		// isCosmosOriginated assumed to be false, since the native cosmos coin
		// expected to be mapped from Ethereum mainnet in first place, i.e. its origin
		// is still from Ethereum.
		return false, k.GetCosmosCoinERC20Contract(ctx)[hyperionId], nil
	}

	// Look up ERC20 contract in index and error if it's not in there
	tokenContract, exists := k.GetCosmosOriginatedERC20(ctx, denomStr)
	if !exists {
		err = errors.Errorf(
			"denom (%s) not a hyperion voucher coin (parse error: %s), and also not in cosmos-originated ERC20 index",
			denomStr, denomErr.Error(),
		)

		metrics.ReportFuncError(k.svcTags)
		return false, common.Address{}, err
	}

	isCosmosOriginated = true
	return isCosmosOriginated, tokenContract, nil
}

// RewardToERC20Lookup is a specialized function wrapping DenomToERC20Lookup designed to validate
// the validator set reward any time we generate a validator set
func (k *Keeper) RewardToERC20Lookup(ctx sdk.Context, coin sdk.Coin, hyperionId uint64) (common.Address, math.Int) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	if coin.Denom == "" || coin.Amount.BigInt() == nil || coin.Amount == math.NewInt(0) {
		metrics.ReportFuncError(k.svcTags)
		panic("Bad validator set relaying reward!")
	} else {
		// reward case, pass to DenomToERC20Lookup
		_, addressStr, err := k.DenomToERC20Lookup(ctx, coin.Denom, hyperionId)
		if err != nil {
			// This can only ever happen if governance sets a value for the reward
			// which is not a valid ERC20 that as been bridged before (either from or to Cosmos)
			// We'll classify that as operator error and just panic
			metrics.ReportFuncError(k.svcTags)
			panic("Invalid Valset reward! Correct or remove the paramater value")
		}
		err = types.ValidateEthAddress(addressStr.Hex())
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			panic("Invalid Valset reward! Correct or remove the paramater value")
		}
		return addressStr, coin.Amount
	}
}

func (k *Keeper) ValidateTokenMetaData(ctx sdk.Context, metadata *types.TokenMetadata) (*types.TokenMetadata, error) {

	if metadata == nil {
		return nil, errors.Errorf("undefined metadata")
	}

	if metadata.Decimals < 0 {
		return nil, errors.Errorf("claim data is not a valid Decimals: %v", metadata.Decimals)
	}

	if metadata.Decimals > 18 {
		return nil, errors.Errorf("claim data is not a valid Decimals: %v", metadata.Decimals)
	}

	if len(metadata.Name) > 30 {
		return nil, errors.Errorf("claim data is not a valid Name: %v len superior to 30", metadata.Name)
	}

	if len(metadata.Symbol) > 30 {
		return nil, errors.Errorf("claim data is not a valid Symbol: %v len superior to 30", metadata.Name)
	}

	return metadata, nil
}

func (k *Keeper) parseClaimData(ctx sdk.Context, claimData string) (*types.TokenMetadata, *sdk.Msg, error) {
	var data types.ClaimData
	var msg sdk.Msg

	claimDataFull := claimData

	if err := json.Unmarshal([]byte(claimData), &data); err != nil {
		return nil, nil, errors.Errorf("claim data is not a json valid or empty (%s)", claimData)
	}

	if data.Metadata != nil {
		claimDataFull = data.Data
		if _, err := k.ValidateTokenMetaData(ctx, data.Metadata); err != nil {
			return nil, nil, err
		}
		if claimDataFull == "" { // metadata alone
			return data.Metadata, nil, nil
		}
	}
	// Check if the claim data is a valid sdk msg
	if err := k.cdc.UnmarshalInterfaceJSON([]byte(claimDataFull), &msg); err != nil {
		return data.Metadata, nil, err
	}

	return data.Metadata, &msg, nil
}

func (k *Keeper) handleValidateMsg(ctx sdk.Context, msg *sdk.Msg) (bool, error) {
	switch (*msg).(type) {
	case *types.MsgSendToChain:
		return true, nil
	}
	return false, errors.Errorf("Message %s not managed", msg)
}

// ERC20ToDenom returns if an ERC20 address represents an asset is native to Cosmos or Ethereum,
// and get its corresponding hyperion denom.
func (k *Keeper) ERC20ToDenomLookup(ctx sdk.Context, tokenContract common.Address, hyperionId uint64) (isCosmosOriginated bool, denom string) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// First try looking up tokenContract in index
	denomStr, exists := k.GetCosmosOriginatedDenom(ctx, tokenContract)
	if exists {
		isCosmosOriginated = true
		return isCosmosOriginated, denomStr
	} else if tokenContract == k.GetCosmosCoinERC20Contract(ctx)[hyperionId] {
		return false, k.GetCosmosCoinDenom(ctx)[hyperionId]
	}

	// If it is not in there, it is not a cosmos originated token, turn the ERC20 into a hyperion denom
	return false, types.NewHyperionDenom(hyperionId, tokenContract).String()
}

// IterateERC20ToDenom iterates over erc20 to denom relations
func (k *Keeper) IterateERC20ToDenom(ctx sdk.Context, cb func(k []byte, v *types.ERC20ToDenom) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ERC20ToDenomKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		erc20ToDenom := types.ERC20ToDenom{
			Erc20: common.BytesToAddress(iter.Key()).Hex(),
			Denom: string(iter.Value()),
		}

		if cb(iter.Key(), &erc20ToDenom) {
			break
		}
	}
}
