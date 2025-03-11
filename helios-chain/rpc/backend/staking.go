package backend

import (
	"fmt"
	"slices"

	cmn "helios-core/helios-chain/precompiles/common"
	erc20types "helios-core/helios-chain/x/erc20/types"

	"cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *Backend) GetValidatorCommission(address common.Address) (map[string]interface{}, error) {
	queryMsg := &distributiontypes.QueryValidatorCommissionRequest{
		ValidatorAddress: cmn.ValAddressFromHexAddress(address).String(),
	}
	res, err := b.queryClient.Distribution.ValidatorCommission(b.ctx, queryMsg)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"amount": res.Commission.Commission.AmountOf("ahelios").TruncateInt(),
		"denom":  "ahelios",
	}, nil
}

func (b *Backend) GetValidatorOutStandingRewards(address common.Address) (map[string]interface{}, error) {
	queryMsg := &distributiontypes.QueryValidatorOutstandingRewardsRequest{
		ValidatorAddress: cmn.ValAddressFromHexAddress(address).String(),
	}
	res, err := b.queryClient.Distribution.ValidatorOutstandingRewards(b.ctx, queryMsg)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"amount": res.Rewards.Rewards.AmountOf("ahelios").TruncateInt(),
		"denom":  "ahelios",
	}, nil
}

func (b *Backend) GetValidator(address common.Address) (map[string]interface{}, error) {
	queryMsg := &stakingtypes.QueryValidatorRequest{
		ValidatorAddr: cmn.ValAddressFromHexAddress(address).String(),
	}
	res, err := b.queryClient.Staking.Validator(b.ctx, queryMsg)
	if err != nil {
		return nil, err
	}
	validator := res.Validator
	valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
	if err != nil {
		b.logger.Error("GetDelegations", "err", err)
		return nil, err
	}
	cosmosAddressOfTheValidator := sdk.AccAddress(valAddr.Bytes())
	evmAddressOfTheValidator := common.BytesToAddress(cosmosAddressOfTheValidator.Bytes()).String()

	apr, err := b.GetValidatorAPR(validator.OperatorAddress)

	return map[string]interface{}{
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
	}, nil
}

func (b *Backend) GetValidatorAndHisCommission(address common.Address) (map[string]interface{}, error) {
	validator, err := b.GetValidator(address)

	if err != nil {
		return nil, err
	}
	commission, err := b.GetValidatorCommission(address)

	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"validator":  validator,
		"commission": commission,
	}, nil
}

func (b *Backend) GetValidatorAndHisDelegation(address common.Address) (map[string]interface{}, error) {
	validator, err := b.GetValidator(address)

	if err != nil {
		return nil, err
	}
	delegation, err := b.GetDelegation(address, address)

	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"validator":  validator,
		"delegation": delegation,
	}, nil
}

func (b *Backend) GetValidatorWithHisDelegationAndCommission(address common.Address) (map[string]interface{}, error) {
	validator, err := b.GetValidator(address)

	if err != nil {
		return nil, err
	}
	delegation, err := b.GetDelegation(address, address)

	if err != nil {
		return nil, err
	}
	commission, err := b.GetValidatorCommission(address)

	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"validator":  validator,
		"delegation": delegation,
		"commission": commission,
	}, nil
}

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

func (b *Backend) GetAllWhitelistedAssets() ([]map[string]interface{}, error) {
	whitelistedAssets := make([]map[string]interface{}, 0)
	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})
	if err != nil {
		b.logger.Error("GetAllWhitelistedAssets", "err", err)
		return nil, err
	}
	repartitionMap, err := b.queryClient.Staking.ShareRepartitionMap(b.ctx, &stakingtypes.QueryShareRepartitionMapRequest{})
	if err != nil {
		b.logger.Error("GetAllWhitelistedAssets", "err", err)
		return nil, err
	}
	for _, asset := range whitelistedAssetsResp.Assets {
		whitelistedAssets = append(whitelistedAssets, map[string]interface{}{
			"denom":                         asset.Denom,
			"baseWeight":                    asset.BaseWeight,
			"chainId":                       asset.ChainId,
			"decimals":                      asset.Decimals,
			"metadata":                      asset.Metadata,
			"contractAddress":               asset.ContractAddress,
			"totalShares":                   repartitionMap.SharesRepartitionMap[asset.Denom].NetworkShares,
			"networkPercentageSecurisation": repartitionMap.SharesRepartitionMap[asset.Denom].NetworkPercentageSecurisation,
		})
	}

	return whitelistedAssets, err
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

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})

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

			idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == asset.Denom })
			baseWeight := math.NewIntFromUint64(1)
			if idx != -1 {
				baseWeight = math.NewIntFromUint64(whitelistedAssetsResp.Assets[idx].GetBaseWeight())
			}

			assets = append(assets, map[string]interface{}{
				"symbol":         asset.Denom,
				"baseAmount":     asset.BaseAmount,
				"amount":         asset.WeightedAmount.Quo(baseWeight),
				"weightedAmount": asset.WeightedAmount,
			})
		}

		delegationRewardsResponse, err := b.queryClient.Distribution.DelegationRewards(b.ctx, &distributiontypes.QueryDelegationRewardsRequest{
			DelegatorAddress: sdk.AccAddress(delegatorAddress.Bytes()).String(),
			ValidatorAddress: delegation.ValidatorAddress,
		})

		if err != nil {
			return delegations, err
		}

		// delegationTotalRewardsResponse, err := b.queryClient.Distribution.DelegationTotalRewards(b.ctx, &distributiontypes.QueryDelegationTotalRewardsRequest{
		// 	DelegatorAddress: sdk.AccAddress(delegatorAddress.Bytes()).String(),
		// })

		delegations = append(delegations, map[string]interface{}{
			"validator_address": evmAddressOfTheValidator,
			"shares":            delegation.Shares.TruncateInt(),
			"assets":            assets,
			"rewards": map[string]interface{}{
				"denom":  delegationRewardsResponse.Rewards[0].Denom,
				"amount": delegationRewardsResponse.Rewards[0].Amount.TruncateInt(),
			},
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

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})

	assets := make([]map[string]interface{}, 0)
	for _, asset := range delegation.AssetWeights {

		idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == asset.Denom })
		baseWeight := math.NewIntFromUint64(1)
		if idx != -1 {
			baseWeight = math.NewIntFromUint64(whitelistedAssetsResp.Assets[idx].GetBaseWeight())
		}

		assets = append(assets, map[string]interface{}{
			"symbol":         asset.Denom,
			"baseAmount":     asset.BaseAmount,
			"amount":         asset.WeightedAmount.Quo(baseWeight),
			"weightedAmount": asset.WeightedAmount,
		})
	}

	delegationRewardsResponse, err := b.queryClient.Distribution.DelegationRewards(b.ctx, &distributiontypes.QueryDelegationRewardsRequest{
		DelegatorAddress: sdk.AccAddress(address.Bytes()).String(),
		ValidatorAddress: delegation.ValidatorAddress,
	})

	if err != nil {
		return nil, err
	}

	if len(delegationRewardsResponse.Rewards) == 0 {
		delegationRewardsResponse.Rewards = append(delegationRewardsResponse.Rewards, sdk.DecCoin{Denom: "ahelios", Amount: math.NewInt(0).ToLegacyDec()})
	}

	return map[string]interface{}{
		"validator_address": validatorAddress,
		"shares":            delegation.Shares.TruncateInt(),
		"assets":            assets,
		"rewards": map[string]interface{}{
			"denom":  delegationRewardsResponse.Rewards[0].Denom,
			"amount": delegationRewardsResponse.Rewards[0].Amount.TruncateInt(),
		},
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
