// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (b *Backend) GetDelegations(delegatorAddress common.Address) ([]stakingtypes.Delegation, error) {
	delegations := make([]map[string]interface{}, 0)
	queryMsg := &stakingtypes.QueryGetDelegationsRequest{
		DelegatorAddr: sdk.AccAddress(delegatorAddress.Bytes()).String(),
	}
	res, err := b.queryClient.Staking.GetDelegations(b.ctx, queryMsg)
	if err != nil {
		b.logger.Error("GetDelegations", "err", err)
		return make([]stakingtypes.Delegation, 0), nil
	}

	for _, delegation := range res.Delegations {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			b.logger.Error("GetDelegations", "err", err)
			return make([]stakingtypes.Delegation, 0), nil
		}
		b.logger.Info("GetDelegations", "valAddr", valAddr)
		cosmosAddressOfTheValidator := sdk.AccAddress(valAddr.Bytes())
		b.logger.Info("GetDelegations", "cosmosAddressOfTheValidator", cosmosAddressOfTheValidator.String())
		evmAddressOfTheValidator := common.BytesToAddress(cosmosAddressOfTheValidator.Bytes()).String()
		b.logger.Info("GetDelegations", "evmAddressOfTheValidator", evmAddressOfTheValidator)
		delegations = append(delegations, map[string]interface{}{
			"validator_address": evmAddressOfTheValidator,
			"shares":            delegation.Shares.String(),
			// "balance": map[string]interface{}{
			// 	"denom": delegation.AssetWeights//.GetBalance().Denom,
			// 	"amount": delegation.GetBalance().Amount.String(),
			// },
		})
	}

	return res.Delegations, nil
}
