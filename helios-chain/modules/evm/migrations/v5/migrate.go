package v5

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v0types "helios-core/helios-chain/modules/evm/migrations/v0/types"
	v4types "helios-core/helios-chain/modules/evm/migrations/v4/types"
	"helios-core/helios-chain/modules/evm/types"
)

// MigrateStore migrates the x/evm module state from the consensus version 4 to
// version 5. Specifically, it takes the parameters that are currently stored
// in separate keys and stores them directly into the x/evm module state using
// a single params key.
func MigrateStore(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var (
		chainConfig v0types.V0ChainConfig
		extraEIPs   v4types.ExtraEIPs
		params      v4types.V4Params
	)
	store := ctx.KVStore(storeKey)
	chainCfgBz := store.Get(v0types.ParamStoreKeyChainConfig)
	cdc.MustUnmarshal(chainCfgBz, &chainConfig)
	params.ChainConfig = chainConfig
	extraEIPsBz := store.Get(v0types.ParamStoreKeyExtraEIPs)
	cdc.MustUnmarshal(extraEIPsBz, &extraEIPs)
	params.ExtraEIPs = extraEIPs
	params.EvmDenom = string(store.Get(v0types.ParamStoreKeyEVMDenom))
	params.EnableCreate = store.Has(v0types.ParamStoreKeyEnableCreate)
	params.EnableCall = store.Has(v0types.ParamStoreKeyEnableCall)
	params.AllowUnprotectedTxs = store.Has(v0types.ParamStoreKeyAllowUnprotectedTxs)
	if err := params.Validate(); err != nil {
		return err
	}
	bz := cdc.MustMarshal(&params)
	store.Set(types.KeyPrefixParams, bz)
	store.Delete(v0types.ParamStoreKeyChainConfig)
	store.Delete(v0types.ParamStoreKeyExtraEIPs)
	store.Delete(v0types.ParamStoreKeyEVMDenom)
	store.Delete(v0types.ParamStoreKeyEnableCreate)
	store.Delete(v0types.ParamStoreKeyEnableCall)
	store.Delete(v0types.ParamStoreKeyAllowUnprotectedTxs)
	return nil
}
