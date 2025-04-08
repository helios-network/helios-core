package keeper

import (
	"encoding/json"
	"reflect"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

func (k *Keeper) GetTokenFromAddress(ctx sdk.Context, hyperionId uint64, tokenContract common.Address) (*types.TokenAddressToDenom, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetTokenAddressToCosmosDenomKey(hyperionId, tokenContract))
	if bz == nil {
		return nil, false
	}

	tokenAddressToDenom := types.TokenAddressToDenom{}
	k.cdc.MustUnmarshal(bz, &tokenAddressToDenom)
	return &tokenAddressToDenom, true
}

func (k *Keeper) GetTokenFromDenom(ctx sdk.Context, hyperionId uint64, denom string) (*types.TokenAddressToDenom, bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.GetCosmosDenomToTokenAddressKey(hyperionId, denom))
	if bz == nil {
		return nil, false
	}

	tokenAddressToDenom := types.TokenAddressToDenom{}
	k.cdc.MustUnmarshal(bz, &tokenAddressToDenom)
	return &tokenAddressToDenom, true
}

func (k *Keeper) SetToken(ctx sdk.Context, hyperionId uint64, tokenAddressToDenom *types.TokenAddressToDenom) *types.TokenAddressToDenom {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetCosmosDenomToTokenAddressKey(hyperionId, tokenAddressToDenom.Denom), k.cdc.MustMarshal(tokenAddressToDenom))
	store.Set(types.GetTokenAddressToCosmosDenomKey(hyperionId, common.HexToAddress(tokenAddressToDenom.TokenAddress)), k.cdc.MustMarshal(tokenAddressToDenom))

	return tokenAddressToDenom
}

// func (k *Keeper) SearchTokenFromDenom(ctx sdk.Context, denomStr string, hyperionId uint64) (tokenAddressToDenom *types.TokenAddressToDenom, err error) {
// 	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
// 	defer doneFn()

// 	// First try parsing the ERC20 out of the denom
// 	// hyperionDenom, denomErr := types.NewHyperionDenomFromString(hyperionId, denomStr)
// 	// if denomErr == nil {
// 	// 	// This is an Ethereum-originated asset
// 	// 	tokenContractFromDenom, _ := hyperionDenom.TokenContract()
// 	// 	return false, tokenContractFromDenom, nil
// 	// }

// 	// Look up ERC20 contract in index and error if it's not in there
// 	tokenAddressToDenom, exists := k.GetTokenFromDenom(ctx, hyperionId, denomStr)
// 	if !exists {
// 		err = errors.Errorf(
// 			"denom (%s) not a hyperion voucher coin, and also not in cosmos-originated token address index",
// 			denomStr,
// 		)
// 		metrics.ReportFuncError(k.svcTags)
// 		return nil, err
// 	}

// 	return tokenAddressToDenom, nil
// }

// RewardToTokenAddressLookup is a specialized function wrapping DenomToERC20Lookup designed to validate
// the validator set reward any time we generate a validator set
// func (k *Keeper) RewardToTokenAddressLookup(ctx sdk.Context, coin sdk.Coin, hyperionId uint64) (common.Address, math.Int) {
// 	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
// 	defer doneFn()

// 	if coin.Denom == "" || coin.Amount.BigInt() == nil || coin.Amount == math.NewInt(0) {
// 		metrics.ReportFuncError(k.svcTags)
// 		panic("Bad validator set relaying reward!")
// 	} else {
// 		// reward case, pass to DenomToERC20Lookup
// 		_, addressStr, err := k.DenomToTokenAddressLookup(ctx, coin.Denom, hyperionId)
// 		if err != nil {
// 			// This can only ever happen if governance sets a value for the reward
// 			// which is not a valid ERC20 that as been bridged before (either from or to Cosmos)
// 			// We'll classify that as operator error and just panic
// 			metrics.ReportFuncError(k.svcTags)
// 			panic("Invalid Valset reward! Correct or remove the paramater value")
// 		}
// 		err = types.ValidateEthAddress(addressStr.Hex())
// 		if err != nil {
// 			metrics.ReportFuncError(k.svcTags)
// 			panic("Invalid Valset reward! Correct or remove the paramater value")
// 		}
// 		return addressStr, coin.Amount
// 	}
// }

func (k *Keeper) ValidateTokenMetaData(ctx sdk.Context, metadata *types.TokenMetadata) (*types.TokenMetadata, error) {

	if metadata == nil {
		return nil, errors.Errorf("undefined metadata")
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

func (k *Keeper) handleValidateMsg(_ sdk.Context, msg *sdk.Msg) (bool, error) {
	switch (*msg).(type) {
	case *types.MsgSendToChain:
		return true, nil
	}
	return false, errors.Errorf("Message %s not managed", reflect.TypeOf(msg))
}

// TokenAddressToDenom returns if a token address represents an asset is native to Cosmos or Ethereum,
// and get its corresponding hyperion denom.
// func (k *Keeper) TokenAddressToDenomLookup(ctx sdk.Context, tokenContract common.Address, hyperionId uint64) (*types.TokenAddressToDenom, error) {
// 	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
// 	defer doneFn()

// 	// First try looking up tokenContract in index
// 	tokenAddressToDenom, exists := k.GetCosmosOriginatedDenom(ctx, hyperionId, tokenContract)
// 	if exists {
// 		return tokenAddressToDenom, nil
// 	}

// 	// If it is not in there, it is not a cosmos originated token, turn the ERC20 into a hyperion denom
// 	return false, types.NewHyperionDenom(hyperionId, tokenContract).String()
// }

// IterateTokenAddressToDenom iterates over token address to denom relations
func (k *Keeper) IterateTokens(ctx sdk.Context, hyperionId uint64, cb func(k []byte, v *types.TokenAddressToDenom) (stop bool)) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	prefixKey := append(types.TokenAddressToDenomKey, types.UInt64Bytes(hyperionId)...)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), prefixKey)
	iter := prefixStore.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		tokenAddressToDenom := types.TokenAddressToDenom{
			TokenAddress: common.BytesToAddress(iter.Key()).Hex(),
			Denom:        string(iter.Value()),
		}

		if cb(iter.Key(), &tokenAddressToDenom) {
			break
		}
	}
}

func (k *Keeper) GetAllTokens(ctx sdk.Context, hyperionId uint64) []*types.TokenAddressToDenom {
	tokenAddressToDenoms := []*types.TokenAddressToDenom{}

	k.IterateTokens(ctx, hyperionId, func(_ []byte, tokenAddressToDenom *types.TokenAddressToDenom) bool {
		tokenAddressToDenoms = append(tokenAddressToDenoms, tokenAddressToDenom)
		return false
	})

	return tokenAddressToDenoms
}
