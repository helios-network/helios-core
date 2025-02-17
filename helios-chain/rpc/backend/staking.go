// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	"fmt"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *Backend) GetValidatorsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error) {
	validatorsResult := make([]map[string]interface{}, 0)
	queryMsg := &stakingtypes.QueryValidatorsRequest{
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	}
	res, err := b.queryClient.Staking.Validators(b.ctx, queryMsg)
	if err != nil {
		return nil, err
	}

	for _, validator := range res.Validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			b.logger.Error("GetDelegations", "err", err)
			return validatorsResult, nil
		}
		cosmosAddressOfTheValidator := sdk.AccAddress(valAddr.Bytes())
		evmAddressOfTheValidator := common.BytesToAddress(cosmosAddressOfTheValidator.Bytes()).String()

		apr, err := b.GetValidatorAPR(validator.OperatorAddress)

		validatorsResult = append(validatorsResult, map[string]interface{}{
			"validatorAddress":        evmAddressOfTheValidator,
			"shares":                  validator.DelegatorShares.String(),
			"moniker":                 validator.GetMoniker(),
			"commission":              validator.Commission,
			"description":             validator.Description,
			"status":                  validator.Status,
			"unbondingHeight":         validator.UnbondingHeight,
			"unbondingIds":            validator.UnbondingIds,
			"jailed":                  validator.Jailed,
			"unbondingOnHoldRefCount": validator.UnbondingOnHoldRefCount,
			"unbondingTime":           validator.UnbondingTime,
			"minSelfDelegation":       validator.MinSelfDelegation,
			"apr":                     apr,
			// todo details of the staking
		})
	}
	return validatorsResult, nil
}

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (b *Backend) GetDelegations(delegatorAddress common.Address) ([]map[string]interface{}, error) {
	delegations := make([]map[string]interface{}, 0)
	queryMsg := &stakingtypes.QueryGetDelegationsRequest{
		DelegatorAddr: sdk.AccAddress(delegatorAddress.Bytes()).String(),
	}
	res, err := b.queryClient.Staking.GetDelegations(b.ctx, queryMsg)
	if err != nil {
		b.logger.Error("GetDelegations", "err", err)
		return delegations, nil
	}

	for _, delegation := range res.Delegations {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			b.logger.Error("GetDelegations", "err", err)
			return delegations, nil
		}
		b.logger.Info("GetDelegations", "valAddr", valAddr)
		cosmosAddressOfTheValidator := sdk.AccAddress(valAddr.Bytes())
		b.logger.Info("GetDelegations", "cosmosAddressOfTheValidator", cosmosAddressOfTheValidator.String())
		evmAddressOfTheValidator := common.BytesToAddress(cosmosAddressOfTheValidator.Bytes()).String()
		b.logger.Info("GetDelegations", "evmAddressOfTheValidator", evmAddressOfTheValidator)

		assets := make([]map[string]interface{}, 0)
		for _, asset := range delegation.AssetWeights {
			assets = append(assets, map[string]interface{}{
				"symbol": asset.Denom,
				"amount": asset.BaseAmount,
				// un nececary for front-end
				// "weightedAmount": asset.WeightedAmount,
			})
		}

		delegationRewardsResponse, err := b.queryClient.Distribution.DelegationRewards(b.ctx, &distributiontypes.QueryDelegationRewardsRequest{
			DelegatorAddress: sdk.AccAddress(delegatorAddress.Bytes()).String(),
			ValidatorAddress: delegation.ValidatorAddress,
		})

		// delegationTotalRewardsResponse, err := b.queryClient.Distribution.DelegationTotalRewards(b.ctx, &distributiontypes.QueryDelegationTotalRewardsRequest{
		// 	DelegatorAddress: sdk.AccAddress(delegatorAddress.Bytes()).String(),
		// })

		delegations = append(delegations, map[string]interface{}{
			"validator_address": evmAddressOfTheValidator,
			"shares":            delegation.Shares.String(),
			"assets":            assets,
			"rewards":           delegationRewardsResponse.Rewards,
		})
	}

	return delegations, nil
}

func (b *Backend) GetDelegation(address common.Address, validatorAddress common.Address) (map[string]interface{}, error) {

	// transform evm address to validator cosmos address
	validatorBech32Addr := sdk.AccAddress(validatorAddress.Bytes())
	valAddr := sdk.ValAddress(validatorBech32Addr)

	queryMsg := &stakingtypes.QueryDelegationRequest{
		DelegatorAddr: sdk.AccAddress(address.Bytes()).String(),
		ValidatorAddr: valAddr.String(),
	}
	res, err := b.queryClient.Staking.Delegation(b.ctx, queryMsg)
	if err != nil {
		b.logger.Error("GetDelegation", "err", err)
		return nil, nil
	}
	delegation := res.DelegationResponse.Delegation

	assets := make([]map[string]interface{}, 0)
	assets = append(assets, map[string]interface{}{
		"symbol": res.DelegationResponse.Balance.Denom,
		"amount": res.DelegationResponse.Balance.Amount,
	})
	for _, asset := range delegation.AssetWeights {
		assets = append(assets, map[string]interface{}{
			"symbol": asset.Denom,
			"amount": asset.BaseAmount,
			// un nececary for front-end
			// "weightedAmount": asset.WeightedAmount,
		})
	}

	delegationRewardsResponse, err := b.queryClient.Distribution.DelegationRewards(b.ctx, &distributiontypes.QueryDelegationRewardsRequest{
		DelegatorAddress: sdk.AccAddress(address.Bytes()).String(),
		ValidatorAddress: delegation.ValidatorAddress,
	})

	return map[string]interface{}{
		"validator_address": validatorAddress,
		"shares":            delegation.Shares.String(),
		"assets":            assets,
		"rewards":           delegationRewardsResponse.Rewards,
	}, nil
}

func (b *Backend) GetValidatorAPR(validatorAddress string) (string, error) {
	// Récupérer le taux de commission du validateur
	validator, err := b.queryClient.Staking.Validator(b.ctx, &stakingtypes.QueryValidatorRequest{
		ValidatorAddr: validatorAddress,
	})
	if err != nil {
		return "0%", err
	}
	commissionRate := validator.Validator.Commission.CommissionRates.Rate.MustFloat64()

	// Récupérer l'inflation
	inflationRes, err := b.queryClient.Mint.Inflation(b.ctx, &minttypes.QueryInflationRequest{})
	if err != nil {
		return "0%", err
	}
	inflation := inflationRes.Inflation.MustFloat64()

	// Récupérer la community tax
	distributionParams, err := b.queryClient.Distribution.Params(b.ctx, &distributiontypes.QueryParamsRequest{})
	if err != nil {
		return "0%", err
	}
	communityTax := distributionParams.Params.CommunityTax.MustFloat64()

	// Récupérer le bonded ratio
	stakingPool, err := b.queryClient.Staking.Pool(b.ctx, &stakingtypes.QueryPoolRequest{})
	if err != nil {
		return "0%", err
	}

	supply, err := b.queryClient.Bank.SupplyOf(b.ctx, &banktypes.QuerySupplyOfRequest{
		Denom: "ahelios", //stakingPool.Pool.BondedTokens.Denom,
	})
	if err != nil {
		return "0%", err
	}

	// todo: voir si le montant est bien comparable au shared bonded tokens par apport a l'inflation
	bondedRatio, err := stakingPool.Pool.BondedTokens.ToLegacyDec().Quo(supply.Amount.Amount.ToLegacyDec()).Float64()
	if err != nil {
		return "0%", err
	}
	// Calculer l'APR
	apr := (1 - communityTax) * (1 - commissionRate) * inflation / bondedRatio

	return fmt.Sprintf("%f%%", apr*100.0), nil
}
