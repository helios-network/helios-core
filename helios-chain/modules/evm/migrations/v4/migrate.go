package v4

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	v0types "helios-core/helios-chain/modules/evm/migrations/v0/types"
	v4types "helios-core/helios-chain/modules/evm/migrations/v4/types"
	"helios-core/helios-chain/modules/evm/types"
)

// MigrateStore migrates the x/evm module state from the consensus version 3 to
// version 4. Specifically, it takes the parameters that are currently stored
// and managed by the Cosmos SDK params module and stores them directly into the x/evm module state.
func MigrateStore(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	legacySubspace types.Subspace,
	cdc codec.BinaryCodec,
) error {
	var params v0types.V0Params
	legacySubspace.GetParamSetIfExists(ctx, &params)
	if err := params.Validate(); err != nil {
		return err
	}
	chainCfgBz := cdc.MustMarshal(&params.ChainConfig)
	extraEIPsBz := cdc.MustMarshal(&v4types.ExtraEIPs{
		EIPs: params.ExtraEIPs,
	})
	store := ctx.KVStore(storeKey)
	store.Set(v0types.ParamStoreKeyEVMDenom, []byte(params.EvmDenom))
	store.Set(v0types.ParamStoreKeyExtraEIPs, extraEIPsBz)
	store.Set(v0types.ParamStoreKeyChainConfig, chainCfgBz)
	if params.AllowUnprotectedTxs {
		store.Set(v0types.ParamStoreKeyAllowUnprotectedTxs, []byte{0x01})
	}
	if params.EnableCall {
		store.Set(v0types.ParamStoreKeyEnableCall, []byte{0x01})
	}
	if params.EnableCreate {
		store.Set(v0types.ParamStoreKeyEnableCreate, []byte{0x01})
	}
	return nil
}
