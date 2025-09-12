package keeper

import (
	"helios-core/helios-chain/x/erc20/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

var isTrue = []byte("0x01")

const addressLength = 42

// GetParams returns the total set of erc20 parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	enableErc20 := k.IsERC20Enabled(ctx)
	return types.NewParams(enableErc20, []string{}, []string{})
}

// SetParams sets the erc20 parameters to the param space.
func (k Keeper) SetParams(ctx sdk.Context, newParams types.Params) error {
	if err := newParams.Validate(); err != nil {
		return err
	}

	// Direct storage without expensive operations
	k.setERC20Enabled(ctx, newParams.EnableErc20)

	return nil
}

// IsERC20Enabled returns true if the module logic is enabled
func (k Keeper) IsERC20Enabled(ctx sdk.Context) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(types.ParamStoreKeyEnableErc20)
}

// setERC20Enabled sets the EnableERC20 param in the store
func (k Keeper) setERC20Enabled(ctx sdk.Context, enable bool) {
	store := ctx.KVStore(k.storeKey)
	if enable {
		store.Set(types.ParamStoreKeyEnableErc20, isTrue)
		return
	}
	store.Delete(types.ParamStoreKeyEnableErc20)
}

func (k Keeper) IsDynamicPrecompileEnabled(ctx sdk.Context, address common.Address) bool {
	store := ctx.KVStore(k.storeKey)
	exists := store.Has(append(types.ParamStoreKeyDynamicPrecompilePrefix, address.Bytes()...))
	return exists
}

func (k Keeper) SetDynamicPrecompileEnabled(ctx sdk.Context, address common.Address) {
	store := ctx.KVStore(k.storeKey)
	store.Set(append(types.ParamStoreKeyDynamicPrecompilePrefix, address.Bytes()...), isTrue)
}

func (k Keeper) IsNativePrecompileEnabled(ctx sdk.Context, address common.Address) bool {
	store := ctx.KVStore(k.storeKey)
	exists := store.Has(append(types.ParamStoreKeyNativePrecompilePrefix, address.Bytes()...))
	return exists
}

func (k Keeper) SetNativePrecompileEnabled(ctx sdk.Context, address common.Address) {
	store := ctx.KVStore(k.storeKey)
	store.Set(append(types.ParamStoreKeyNativePrecompilePrefix, address.Bytes()...), isTrue)
}

// setDynamicPrecompiles sets the DynamicPrecompiles param in the store
func (k Keeper) SetOldDynamicPrecompiles(ctx sdk.Context, dynamicPrecompiles []string) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 0, addressLength*len(dynamicPrecompiles))
	for _, str := range dynamicPrecompiles {
		bz = append(bz, []byte(str)...)
	}
	store.Set(types.ParamStoreKeyDynamicPrecompiles, bz)
}

// getDynamicPrecompiles returns the DynamicPrecompiles param from the store
func (k Keeper) GetOldDynamicPrecompiles(ctx sdk.Context) (dynamicPrecompiles []string) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamStoreKeyDynamicPrecompiles)

	for i := 0; i < len(bz); i += addressLength {
		dynamicPrecompiles = append(dynamicPrecompiles, string(bz[i:i+addressLength]))
	}
	return dynamicPrecompiles
}

// setNativePrecompiles sets the NativePrecompiles param in the store
func (k Keeper) SetOldNativePrecompiles(ctx sdk.Context, nativePrecompiles []string) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 0, addressLength*len(nativePrecompiles))
	for _, str := range nativePrecompiles {
		bz = append(bz, []byte(str)...)
	}
	store.Set(types.ParamStoreKeyNativePrecompiles, bz)
}

// getNativePrecompiles returns the NativePrecompiles param from the store
func (k Keeper) GetOldNativePrecompiles(ctx sdk.Context) (nativePrecompiles []string) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamStoreKeyNativePrecompiles)
	for i := 0; i < len(bz); i += addressLength {
		nativePrecompiles = append(nativePrecompiles, string(bz[i:i+addressLength]))
	}
	return nativePrecompiles
}

// func (k Keeper) IsOldDynamicPrecompileEnabled(ctx sdk.Context, address common.Address) bool {
// 	dynamicPrecompiles := k.GetOldDynamicPrecompiles(ctx)
// 	for _, precompile := range dynamicPrecompiles {
// 		if precompile == address.Hex() {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (k Keeper) IsOldNativePrecompileEnabled(ctx sdk.Context, address common.Address) bool {
// 	nativePrecompiles := k.GetOldNativePrecompiles(ctx)
// 	for _, precompile := range nativePrecompiles {
// 		if precompile == address.Hex() {
// 			return true
// 		}
// 	}
// 	return false
// }
