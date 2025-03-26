package keeper

import (
	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/feedistribution/types"
)

// SetContractInfo stores contract information in the module's state
func (k Keeper) SetContractInfo(ctx sdk.Context, contractInfo types.ContractInfo) error {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixContract)
	key := []byte(contractInfo.ContractAddress)

	bz := k.cdc.MustMarshal(&contractInfo)
	store.Set(key, bz)
	return nil
}

// GetContractInfo retrieves contract information from the module's state
func (k Keeper) GetContractInfo(ctx sdk.Context, contractAddr string) (types.ContractInfo, bool) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixContract)
	key := []byte(contractAddr)

	bz := store.Get(key)
	if bz == nil {
		return types.ContractInfo{}, false
	}

	var contractInfo types.ContractInfo
	k.cdc.MustUnmarshal(bz, &contractInfo)
	return contractInfo, true
}

// DeleteContractInfo removes contract information from the module's state
func (k Keeper) DeleteContractInfo(ctx sdk.Context, contractAddr string) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixContract)
	key := []byte(contractAddr)
	store.Delete(key)
}

// GetAllContracts returns all registered contracts with pagination
func (k Keeper) GetAllContracts(ctx sdk.Context, offset, limit uint64) ([]types.ContractInfo, uint64) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixContract)

	var contracts []types.ContractInfo
	var count uint64

	// Skip to the offset
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid() && count < offset; iterator.Next() {
		count++
	}

	// Collect up to limit items
	for ; iterator.Valid() && uint64(len(contracts)) < limit; iterator.Next() {
		var contract types.ContractInfo
		k.cdc.MustUnmarshal(iterator.Value(), &contract)
		contracts = append(contracts, contract)
	}

	// Count total items
	var total uint64
	iterator = store.Iterator(nil, nil)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		total++
	}

	return contracts, total
}

// GetDeployerContracts returns all contracts deployed by a specific address with pagination
func (k Keeper) GetDeployerContracts(ctx sdk.Context, deployerAddr string, offset, limit uint64) ([]types.ContractInfo, uint64) {
	store := prefix.NewStore(k.GetStore(ctx), types.KeyPrefixContract)

	var contracts []types.ContractInfo
	var count uint64
	var matchCount uint64

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	// Iterate through all contracts
	for ; iterator.Valid(); iterator.Next() {
		var contract types.ContractInfo
		k.cdc.MustUnmarshal(iterator.Value(), &contract)

		// If this contract matches our deployer
		if contract.DeployerAddress == deployerAddr {
			matchCount++
			// If we've reached our offset
			if matchCount > offset {
				// And we haven't hit our limit
				if uint64(len(contracts)) < limit {
					contracts = append(contracts, contract)
				}
			}
		}
		count++
	}

	return contracts, matchCount
}
