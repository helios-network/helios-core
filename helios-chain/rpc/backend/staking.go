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

	rpctypes "helios-core/helios-chain/rpc/types"
)

func (b *Backend) GetValidatorCommission(address common.Address) (*rpctypes.ValidatorCommissionRPC, error) {
	queryMsg := &distributiontypes.QueryValidatorCommissionRequest{
		ValidatorAddress: cmn.ValAddressFromHexAddress(address).String(),
	}
	res, err := b.queryClient.Distribution.ValidatorCommission(b.ctx, queryMsg)
	if err != nil {
		return nil, err
	}

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})
	if err != nil {
		b.logger.Error("GetDelegation", "err", err)
		return nil, nil
	}

	idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == sdk.DefaultBondDenom })
	heliosContractAddress := ""
	if idx != -1 {
		heliosContractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
	}

	return &rpctypes.ValidatorCommissionRPC{
		Amount:          res.Commission.Commission.AmountOf(sdk.DefaultBondDenom).TruncateInt(),
		Denom:           sdk.DefaultBondDenom,
		ContractAddress: heliosContractAddress,
	}, nil
}

func (b *Backend) GetValidatorOutStandingRewards(address common.Address) (*rpctypes.ValidatorRewardRPC, error) {
	queryMsg := &distributiontypes.QueryValidatorOutstandingRewardsRequest{
		ValidatorAddress: cmn.ValAddressFromHexAddress(address).String(),
	}
	res, err := b.queryClient.Distribution.ValidatorOutstandingRewards(b.ctx, queryMsg)
	if err != nil {
		return nil, err
	}

	return &rpctypes.ValidatorRewardRPC{
		Amount: res.Rewards.Rewards.AmountOf(sdk.DefaultBondDenom).TruncateInt(),
		Denom:  sdk.DefaultBondDenom,
	}, nil
}

func (b *Backend) GetValidator(address common.Address) (*rpctypes.ValidatorRPC, error) {
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
	if err != nil {
		b.logger.Error("GetValidatorAPR", "err", err)
		return nil, err
	}

	boostQuery := &stakingtypes.QueryTotalBoostedValidatorRequest{
		ValidatorAddr: validator.OperatorAddress,
	}

	boostRes, err := b.queryClient.Staking.TotalBoostedValidator(b.ctx, boostQuery)
	totalBoost := "0"
	boostPercentage := "0"
	if err == nil {
		totalBoost = boostRes.TotalBoost
		boostPercentage = boostRes.BoostPercentage
	}
	formattedValidatorResp := formatValidatorResponse(validator, evmAddressOfTheValidator, apr, totalBoost, boostPercentage)
	return &formattedValidatorResp, nil
}

func (b *Backend) GetValidatorAndHisCommission(address common.Address) (*rpctypes.ValidatorWithCommissionRPC, error) {
	validator, err := b.GetValidator(address)
	if err != nil {
		return nil, err
	}
	commission, err := b.GetValidatorCommission(address)
	if err != nil {
		return nil, err
	}
	return &rpctypes.ValidatorWithCommissionRPC{
		Validator:  *validator,
		Commission: *commission,
	}, nil
}

func (b *Backend) GetValidatorAndHisDelegation(address common.Address) (*rpctypes.ValidatorWithDelegationRPC, error) {
	validator, err := b.GetValidator(address)
	if err != nil {
		return nil, err
	}
	delegation, err := b.GetDelegation(address, address)
	if err != nil {
		return nil, err
	}
	return &rpctypes.ValidatorWithDelegationRPC{
		Validator:  *validator,
		Delegation: *delegation,
	}, nil
}

func (b *Backend) GetValidatorDelegation(address common.Address) (*rpctypes.DelegationRPC, error) {
	delegation, err := b.GetDelegation(address, address)
	if err != nil {
		return nil, err
	}
	return delegation, nil
}

func (b *Backend) GetValidatorWithHisDelegationAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndDelegationRPC, error) {
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
	return &rpctypes.ValidatorWithCommissionAndDelegationRPC{
		Validator:  *validator,
		Delegation: *delegation,
		Commission: *commission,
	}, nil
}

// GetValidatorWithHisAssetsAndCommission returns validator info with assets and commission
func (b *Backend) GetValidatorWithHisAssetsAndCommission(address common.Address) (*rpctypes.ValidatorWithCommissionAndAssetsRPC, error) {
	validator, err := b.GetValidator(address)
	if err != nil {
		return nil, err
	}

	commission, err := b.GetValidatorCommission(address)
	if err != nil {
		return nil, err
	}

	validatorBech32Addr := sdk.AccAddress(address.Bytes())
	valAddr := sdk.ValAddress(validatorBech32Addr)

	assetsResp, err := b.queryClient.Staking.ValidatorAssets(b.ctx, &stakingtypes.QueryValidatorAssetsRequest{
		ValidatorAddr: valAddr.String(),
	})
	if err != nil {
		return nil, err
	}

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})
	if err != nil {
		return nil, err
	}

	enrichedAssets := make([]rpctypes.ValidatorAssetRPC, 0, len(assetsResp.Assets))
	for _, asset := range assetsResp.Assets {
		idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == asset.Denom })
		contractAddress := ""
		if idx != -1 {
			contractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
		}

		enrichedAssets = append(enrichedAssets, rpctypes.ValidatorAssetRPC{
			Denom:           asset.Denom,
			BaseAmount:      asset.BaseAmount,
			WeightedAmount:  asset.WeightedAmount,
			ContractAddress: contractAddress,
		})
	}

	return &rpctypes.ValidatorWithCommissionAndAssetsRPC{
		Validator:  *validator,
		Assets:     enrichedAssets,
		Commission: *commission,
	}, nil
}

// GetValidatorWithHisAssets returns validator info with assets
func (b *Backend) GetValidatorAssets(address common.Address) ([]rpctypes.ValidatorAssetRPC, error) {
	validatorBech32Addr := sdk.AccAddress(address.Bytes())
	valAddr := sdk.ValAddress(validatorBech32Addr)

	assetsResp, err := b.queryClient.Staking.ValidatorAssets(b.ctx, &stakingtypes.QueryValidatorAssetsRequest{
		ValidatorAddr: valAddr.String(),
	})
	if err != nil {
		return nil, err
	}

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})
	if err != nil {
		return nil, err
	}

	enrichedAssets := make([]rpctypes.ValidatorAssetRPC, 0, len(assetsResp.Assets))
	for _, asset := range assetsResp.Assets {
		idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == asset.Denom })
		contractAddress := ""
		if idx != -1 {
			contractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
		}

		enrichedAssets = append(enrichedAssets, rpctypes.ValidatorAssetRPC{
			Denom:           asset.Denom,
			BaseAmount:      asset.BaseAmount,
			WeightedAmount:  asset.WeightedAmount,
			ContractAddress: contractAddress,
		})
	}

	return enrichedAssets, nil
}

func (b *Backend) GetValidatorsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorRPC, error) {
	if page <= 0 {
		return nil, fmt.Errorf("page must be greater than 0")
	}
	if size <= 0 || size > 100 { // prevent excessive page sizes
		return nil, fmt.Errorf("size must be between 1 and 100")
	}

	inflationRes, err := b.queryClient.Mint.Inflation(b.ctx, &minttypes.QueryInflationRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get inflation: %w", err)
	}

	distributionParams, err := b.queryClient.Distribution.Params(b.ctx, &distributiontypes.QueryParamsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get distribution params: %w", err)
	}

	stakingPool, err := b.queryClient.Staking.Pool(b.ctx, &stakingtypes.QueryPoolRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get staking pool: %w", err)
	}

	supply, err := b.queryClient.Bank.SupplyOf(b.ctx, &banktypes.QuerySupplyOfRequest{
		Denom: "ahelios",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get supply: %w", err)
	}

	// Calculate common values once
	inflation := inflationRes.Inflation.MustFloat64()
	communityTax := distributionParams.Params.CommunityTax.MustFloat64()
	bondedRatio, err := stakingPool.Pool.BondedTokens.ToLegacyDec().Quo(supply.Amount.Amount.ToLegacyDec()).Float64()
	if err != nil {
		return nil, fmt.Errorf("failed to calculate bonded ratio: %w", err)
	}

	// Get validators with pagination
	queryMsg := &stakingtypes.QueryValidatorsRequest{
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
	}
	validatorsResp, err := b.queryClient.Staking.Validators(b.ctx, queryMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	validatorsResult := make([]rpctypes.ValidatorRPC, 0, len(validatorsResp.Validators))

	for _, validator := range validatorsResp.Validators {
		valAddr, err := sdk.ValAddressFromBech32(validator.OperatorAddress)
		if err != nil {
			b.logger.Error("failed to parse validator address",
				"validator", validator.OperatorAddress,
				"error", err)
			continue // Skip invalid validators instead of failing entirely
		}

		// calculate APR inline using the pre calculated common factors
		commissionRate := validator.Commission.CommissionRates.Rate.MustFloat64()
		apr := calculateAPR(inflation, communityTax, commissionRate, bondedRatio)

		validatorCosmosAddress := sdk.AccAddress(valAddr.Bytes())
		validatorEVMAddress := common.BytesToAddress(validatorCosmosAddress.Bytes()).String()

		boostQuery := &stakingtypes.QueryTotalBoostedValidatorRequest{
			ValidatorAddr: validator.OperatorAddress,
		}

		boostRes, err := b.queryClient.Staking.TotalBoostedValidator(b.ctx, boostQuery)
		totalBoost := "0"
		boostPercentage := "0"
		if err == nil {
			totalBoost = boostRes.TotalBoost
			boostPercentage = boostRes.BoostPercentage
		}

		validatorsResult = append(validatorsResult, formatValidatorResponse(validator, validatorEVMAddress, apr, totalBoost, boostPercentage))
	}
	return validatorsResult, nil
}

func (b *Backend) GetValidatorsByPageAndSizeWithHisAssetsAndCommissionAndDelegation(page hexutil.Uint64, size hexutil.Uint64) ([]rpctypes.ValidatorWithAssetsAndCommissionAndDelegationRPC, error) {
	validatorsResp, err := b.GetValidatorsByPageAndSize(page, size)
	if err != nil {
		return nil, err
	}
	validatorsResult := make([]rpctypes.ValidatorWithAssetsAndCommissionAndDelegationRPC, 0, len(validatorsResp))
	for _, validator := range validatorsResp {
		assets, err := b.GetValidatorAssets(common.HexToAddress(validator.ValidatorAddress))
		if err != nil {
			return nil, err
		}
		delegation, err := b.GetValidatorDelegation(common.HexToAddress(validator.ValidatorAddress))
		if err != nil {
			return nil, err
		}
		commission, err := b.GetValidatorCommission(common.HexToAddress(validator.ValidatorAddress))
		if err != nil {
			return nil, err
		}
		validatorsResult = append(validatorsResult, rpctypes.ValidatorWithAssetsAndCommissionAndDelegationRPC{
			Validator:  validator,
			Assets:     assets,
			Commission: *commission,
			Delegation: *delegation,
		})
	}
	return validatorsResult, nil
}

func (b *Backend) GetActiveValidatorCount() (int, error) {
	queryMsg := &stakingtypes.QueryValidatorsRequest{}
	validatorsResp, err := b.queryClient.Staking.Validators(b.ctx, queryMsg)
	if err != nil {
		return 0, fmt.Errorf("failed to get validators: %w", err)
	}

	validatorCount := 0

	for _, validator := range validatorsResp.Validators {
		if validator.Status == 3 {
			validatorCount++
		}
	}

	return validatorCount, nil
}

// Helper function to calculate APR
func calculateAPR(inflation, communityTax, commissionRate, bondedRatio float64) string {
	apr := (1 - communityTax) * (1 - commissionRate) * inflation / bondedRatio
	return fmt.Sprintf("%f%%", apr*100.0)
}

// Helper function to format validator response
func formatValidatorResponse(validator stakingtypes.Validator, evmAddress string, apr string, totalBoost string, boostPercentage string) rpctypes.ValidatorRPC {
	return rpctypes.ValidatorRPC{
		ValidatorAddress:        evmAddress,
		Shares:                  validator.DelegatorShares.String(),
		Moniker:                 validator.GetMoniker(),
		Commission:              validator.Commission,
		Description:             validator.Description,
		Status:                  validator.Status,
		UnbondingHeight:         validator.UnbondingHeight,
		UnbondingIds:            validator.UnbondingIds,
		Jailed:                  validator.Jailed,
		UnbondingOnHoldRefCount: validator.UnbondingOnHoldRefCount,
		UnbondingTime:           validator.UnbondingTime,
		MinSelfDelegation:       validator.MinSelfDelegation,
		Apr:                     apr,
		MinDelegation:           validator.MinDelegation,
		DelegationAuthorization: validator.DelegateAuthorization,
		TotalBoost:              totalBoost,
		BoostPercentage:         boostPercentage,
	}
}

func (b *Backend) GetAllWhitelistedAssets() ([]rpctypes.WhitelistedAssetRPC, error) {
	whitelistedAssets := make([]rpctypes.WhitelistedAssetRPC, 0)
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
		whitelistedAssets = append(whitelistedAssets, rpctypes.WhitelistedAssetRPC{
			Denom:                         asset.Denom,
			BaseWeight:                    asset.BaseWeight,
			ChainId:                       asset.ChainId,
			ChainName:                     asset.ChainName,
			Decimals:                      asset.Decimals,
			Symbol:                        asset.Symbol,
			ContractAddress:               asset.ContractAddress,
			TotalShares:                   repartitionMap.SharesRepartitionMap[asset.Denom].NetworkShares,
			NetworkPercentageSecurisation: repartitionMap.SharesRepartitionMap[asset.Denom].NetworkPercentageSecurisation,
		})
	}

	return whitelistedAssets, err
}

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (b *Backend) GetDelegations(delegatorAddress common.Address) ([]rpctypes.DelegationRPC, error) {
	delegations := make([]rpctypes.DelegationRPC, 0)
	queryMsg := &stakingtypes.QueryGetDelegationsRequest{
		DelegatorAddr: sdk.AccAddress(delegatorAddress.Bytes()).String(),
	}
	res, err := b.queryClient.Staking.GetDelegations(b.ctx, queryMsg)
	if err != nil {
		b.logger.Error("GetDelegations", "err", err)
		return delegations, nil
	}

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})
	if err != nil {
		b.logger.Error("GetDelegations", "err", err)
		return delegations, nil
	}

	idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == sdk.DefaultBondDenom })
	heliosContractAddress := ""
	if idx != -1 {
		heliosContractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
	}

	for _, delegation := range res.Delegations {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			b.logger.Error("GetDelegations", "err", err)
			return delegations, nil
		}
		cosmosAddressOfTheValidator := sdk.AccAddress(valAddr.Bytes())
		evmAddressOfTheValidator := common.BytesToAddress(cosmosAddressOfTheValidator.Bytes()).String()

		assets := make([]rpctypes.DelegationAsset, 0)
		for _, asset := range delegation.AssetWeights {

			idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == asset.Denom })
			baseWeight := math.NewIntFromUint64(1)
			contractAddress := ""
			if idx != -1 {
				baseWeight = math.NewIntFromUint64(whitelistedAssetsResp.Assets[idx].GetBaseWeight())
				contractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
			}

			assets = append(assets, rpctypes.DelegationAsset{
				Denom:           asset.Denom,
				BaseAmount:      asset.BaseAmount,
				Amount:          asset.WeightedAmount.Quo(baseWeight),
				WeightedAmount:  asset.WeightedAmount,
				ContractAddress: contractAddress,
			})
		}

		delegationRewardsResponse, err := b.queryClient.Distribution.DelegationRewards(b.ctx, &distributiontypes.QueryDelegationRewardsRequest{
			DelegatorAddress: sdk.AccAddress(delegatorAddress.Bytes()).String(),
			ValidatorAddress: delegation.ValidatorAddress,
		})
		if err != nil {
			return delegations, err
		}

		if len(delegationRewardsResponse.Rewards) == 0 {
			delegationRewardsResponse.Rewards = append(delegationRewardsResponse.Rewards, sdk.DecCoin{Denom: "ahelios", Amount: math.NewInt(0).ToLegacyDec()})
		}

		boostQuery := &stakingtypes.QueryTotalBoostedDelegationRequest{
			ValidatorAddr: delegation.ValidatorAddress,
			DelegatorAddr: sdk.AccAddress(delegatorAddress.Bytes()).String(),
		}

		boostRes, err := b.queryClient.Staking.TotalBoostedDelegation(b.ctx, boostQuery)
		totalBoost := "0"
		if err == nil {
			totalBoost = boostRes.TotalBoost
		}

		delegations = append(delegations, rpctypes.DelegationRPC{
			ValidatorAddress: evmAddressOfTheValidator,
			Shares:           delegation.Shares.TruncateInt().String(),
			Assets:           assets,
			Rewards: rpctypes.DelegationRewardRPC{
				Denom:           delegationRewardsResponse.Rewards[0].Denom,
				Amount:          delegationRewardsResponse.Rewards[0].Amount.TruncateInt(),
				ContractAddress: heliosContractAddress,
			},
			TotalBoost: totalBoost,
		})
	}

	boostQuery := &stakingtypes.QueryTotalBoostedDelegationsRequest{
		DelegatorAddr: sdk.AccAddress(delegatorAddress.Bytes()).String(),
	}

	boostRes, err := b.queryClient.Staking.TotalBoostedDelegations(b.ctx, boostQuery)
	if err != nil {
		b.logger.Error("TotalBoostedDelegation", "err", err)
		return delegations, err
	}

	for _, boost := range boostRes.DelegationBoosts {
		found := false
		for _, delegation := range delegations {
			if delegation.ValidatorAddress == cmn.AnyToHexAddress(boost.ValidatorAddress).Hex() {
				found = true
				break
			}
		}

		if !found {
			delegations = append(delegations, rpctypes.DelegationRPC{
				ValidatorAddress: cmn.AnyToHexAddress(boost.ValidatorAddress).Hex(),
				Shares:           "0",
				Assets:           []rpctypes.DelegationAsset{},
				Rewards: rpctypes.DelegationRewardRPC{
					Denom:           sdk.DefaultBondDenom,
					Amount:          math.NewInt(0),
					ContractAddress: heliosContractAddress,
				},
				TotalBoost: boost.Amount.String(),
			})
		}
	}

	return delegations, nil
}

func (b *Backend) GetDelegation(address common.Address, validatorAddress common.Address) (*rpctypes.DelegationRPC, error) {
	// transform evm address to validator cosmos address
	validatorBech32Addr := sdk.AccAddress(validatorAddress.Bytes())
	valAddr := sdk.ValAddress(validatorBech32Addr)

	whitelistedAssetsResp, err := b.queryClient.Erc20.WhitelistedAssets(b.ctx, &erc20types.QueryWhitelistedAssetsRequest{})
	if err != nil {
		b.logger.Error("GetDelegation", "err", err)
		return nil, nil
	}

	idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == sdk.DefaultBondDenom })
	heliosContractAddress := ""
	if idx != -1 {
		heliosContractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
	}

	queryMsg := &stakingtypes.QueryDelegationRequest{
		DelegatorAddr: sdk.AccAddress(address.Bytes()).String(),
		ValidatorAddr: valAddr.String(),
	}
	res, err := b.queryClient.Staking.Delegation(b.ctx, queryMsg)
	if err != nil {
		boostQuery := &stakingtypes.QueryTotalBoostedDelegationRequest{
			ValidatorAddr: valAddr.String(),
			DelegatorAddr: sdk.AccAddress(address.Bytes()).String(),
		}

		boostRes, err := b.queryClient.Staking.TotalBoostedDelegation(b.ctx, boostQuery)
		if err == nil { // if the delegation doesnt exists and boost exists, return the boost
			return &rpctypes.DelegationRPC{
				ValidatorAddress: validatorAddress.String(),
				Shares:           "0",
				Assets:           []rpctypes.DelegationAsset{},
				Rewards: rpctypes.DelegationRewardRPC{
					Denom:           sdk.DefaultBondDenom,
					Amount:          math.NewInt(0),
					ContractAddress: heliosContractAddress,
				},
				TotalBoost: boostRes.TotalBoost,
			}, nil
		}

		return nil, nil
	}
	delegation := res.DelegationResponse.Delegation

	assets := make([]rpctypes.DelegationAsset, 0)
	for _, asset := range delegation.AssetWeights {

		idx := slices.IndexFunc(whitelistedAssetsResp.Assets, func(c erc20types.Asset) bool { return c.Denom == asset.Denom })
		baseWeight := math.NewIntFromUint64(1)
		contractAddress := ""
		if idx != -1 {
			baseWeight = math.NewIntFromUint64(whitelistedAssetsResp.Assets[idx].GetBaseWeight())
			contractAddress = whitelistedAssetsResp.Assets[idx].ContractAddress
		}

		assets = append(assets, rpctypes.DelegationAsset{
			Denom:           asset.Denom,
			BaseAmount:      asset.BaseAmount,
			Amount:          asset.WeightedAmount.Quo(baseWeight),
			WeightedAmount:  asset.WeightedAmount,
			ContractAddress: contractAddress,
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

	boostQuery := &stakingtypes.QueryTotalBoostedDelegationRequest{
		ValidatorAddr: delegation.ValidatorAddress,
		DelegatorAddr: sdk.AccAddress(address.Bytes()).String(),
	}

	boostRes, err := b.queryClient.Staking.TotalBoostedDelegation(b.ctx, boostQuery)
	totalBoost := "0"
	if err == nil {
		totalBoost = boostRes.TotalBoost
	}

	return &rpctypes.DelegationRPC{
		ValidatorAddress: validatorAddress.String(),
		Shares:           delegation.Shares.TruncateInt().String(),
		Assets:           assets,
		Rewards: rpctypes.DelegationRewardRPC{
			Denom:           delegationRewardsResponse.Rewards[0].Denom,
			Amount:          delegationRewardsResponse.Rewards[0].Amount.TruncateInt(),
			ContractAddress: heliosContractAddress,
		},
		TotalBoost: totalBoost,
	}, nil
}

func (b *Backend) GetValidatorAPR(validatorAddress string) (string, error) {
	// Get validator commission rate
	validator, err := b.queryClient.Staking.Validator(b.ctx, &stakingtypes.QueryValidatorRequest{
		ValidatorAddr: validatorAddress,
	})
	if err != nil {
		return "0%", err
	}
	commissionRate := validator.Validator.Commission.CommissionRates.Rate.MustFloat64()

	// Get inflation rate
	inflationRes, err := b.queryClient.Mint.Inflation(b.ctx, &minttypes.QueryInflationRequest{})
	if err != nil {
		return "0%", err
	}
	inflation := inflationRes.Inflation.MustFloat64()

	// Get community tax
	distributionParams, err := b.queryClient.Distribution.Params(b.ctx, &distributiontypes.QueryParamsRequest{})
	if err != nil {
		return "0%", err
	}
	communityTax := distributionParams.Params.CommunityTax.MustFloat64()

	// Get bonded ratio
	stakingPool, err := b.queryClient.Staking.Pool(b.ctx, &stakingtypes.QueryPoolRequest{})
	if err != nil {
		return "0%", err
	}

	supply, err := b.queryClient.Bank.SupplyOf(b.ctx, &banktypes.QuerySupplyOfRequest{
		Denom: "ahelios", // stakingPool.Pool.BondedTokens.Denom,
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

// GetBlockSignatures queries block signatures with address conversion
func (b *Backend) GetBlockSignatures(blockHeight hexutil.Uint64) ([]*rpctypes.ValidatorSignature, error) {
	// Call the gRPC query
	req := &stakingtypes.QueryBlockSignaturesRequest{
		Height: int64(blockHeight),
	}

	res, err := b.queryClient.Staking.BlockSignatures(b.ctx, req)
	if err != nil {
		b.logger.Error("GetBlockSignatures", "err", err)
		return nil, err
	}

	// Convert response with address conversion
	signatures := make([]*rpctypes.ValidatorSignature, len(res.Signatures))
	for i, sig := range res.Signatures {
		// Convert operator address to ETH format - CORRECTION ICI
		valAddr, err := sdk.ValAddressFromBech32(sig.Address)
		var ethAddr string
		if err != nil {
			b.logger.Error("failed to decode validator address", "addr", sig.Address, "err", err)
			ethAddr = ""
		} else {
			ethAddr = common.BytesToAddress(valAddr.Bytes()).Hex()
		}

		// Convert AssetWeights
		var assetWeights []*rpctypes.AssetWeight
		for _, aw := range sig.AssetWeights {
			assetWeights = append(assetWeights, &rpctypes.AssetWeight{
				Denom:          aw.Denom,
				BaseAmount:     aw.BaseAmount.String(),
				WeightedAmount: aw.WeightedAmount.String(),
			})
		}

		signatures[i] = &rpctypes.ValidatorSignature{
			Address:          ethAddr,
			Signed:           sig.Signed,
			IndexOffset:      sig.IndexOffset,
			TotalTokens:      sig.TotalTokens,
			AssetWeights:     assetWeights,
			EpochNumber:      sig.EpochNumber,
			Status:           sig.Status,
			Jailed:           sig.Jailed,
			MissedBlockCount: sig.MissedBlockCount,
		}
	}

	return signatures, nil
}

// GetEpochComplete queries complete epoch data with address conversion
func (b *Backend) GetEpochComplete(epochId hexutil.Uint64) (*rpctypes.EpochCompleteResponse, error) {
	// Call the gRPC query
	req := &stakingtypes.QueryEpochCompleteRequest{
		EpochId: uint64(epochId),
	}
	res, err := b.queryClient.Staking.EpochComplete(b.ctx, req)
	if err != nil {
		b.logger.Error("GetEpochComplete", "err", err)
		return nil, err
	}
	// Convert response with address conversion
	validators := make([]*rpctypes.EpochValidatorDetail, len(res.Validators))
	for i, val := range res.Validators {
		// Convert operator address to ETH format
		valAddr, err := sdk.ValAddressFromBech32(val.OperatorAddress)
		var ethAddress string
		if err != nil {
			b.logger.Error("failed to decode validator address", "addr", val.OperatorAddress, "err", err)
			ethAddress = ""
		} else {
			ethAddress = common.BytesToAddress(valAddr.Bytes()).Hex()
		}

		// Convert BlocksSigned
		var blocksSigned []*rpctypes.EpochValidatorSignature
		for _, bs := range val.BlocksSigned {
			blocksSigned = append(blocksSigned, &rpctypes.EpochValidatorSignature{
				Signature: bs.Signature,
				Height:    bs.Height,
			})
		}
		// Convert AssetWeights
		var assetWeights []*rpctypes.AssetWeight
		for _, aw := range val.AssetWeights {
			assetWeights = append(assetWeights, &rpctypes.AssetWeight{
				Denom:          aw.Denom,
				BaseAmount:     aw.BaseAmount.String(),
				WeightedAmount: aw.WeightedAmount.String(),
			})
		}
		validators[i] = &rpctypes.EpochValidatorDetail{
			ValidatorAddress:       ethAddress, // Même adresse que l'opérateur
			OperatorAddress:        ethAddress, // L'adresse de l'opérateur
			BlocksSigned:           blocksSigned,
			BlocksMissed:           val.BlocksMissed,
			BondedTokens:           val.BondedTokens.String(),
			Status:                 val.Status,
			IsSlashed:              val.IsSlashed,
			IsJailed:               val.IsJailed,
			AssetWeights:           assetWeights,
			MissedBlockCount:       val.MissedBlockCount,
			SigningInfoStartHeight: val.SigningInfoStartHeight,
			CommissionRate:         val.CommissionRate,
			VotingPower:            val.VotingPower.String(),
		}
	}
	return &rpctypes.EpochCompleteResponse{
		Epoch:                res.Epoch,
		EpochLength:          res.EpochLength,
		StartHeight:          res.StartHeight,
		EndHeight:            res.EndHeight,
		CurrentHeight:        res.CurrentHeight,
		BlocksValidated:      res.BlocksValidated,
		BlocksRemaining:      res.BlocksRemaining,
		BlocksUntilNextEpoch: res.BlocksUntilNextEpoch,
		Validators:           validators,
		TotalTokens:          res.TotalTokens.String(),
		TotalVotingPower:     res.TotalVotingPower.String(),
	}, nil
}
