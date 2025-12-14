package keeper

import (
	"context"
	"fmt"
	"time"

	"helios-core/helios-chain/x/chaininfo/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
)

var _ types.QueryServer = Keeper{}

const (
	BlocksPerYear      = 60 * 60 * 24 * 365 / 5
	GenesisBlockHeight = 0
)

func (k Keeper) CoinInfo(c context.Context, req *types.QueryCoinInfoRequest) (*types.QueryCoinInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	bondDenom, err := k.stakingKeeper.BondDenom(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bond denom: %w", err)
	}

	totalSupply := k.bankKeeper.GetSupply(ctx, bondDenom)
	totalSupplyAmount := totalSupply.Amount

	if totalSupplyAmount.IsZero() {
		return &types.QueryCoinInfoResponse{
			TotalSupply:              "0",
			RewardsPerBlock:          "0",
			RewardsSinceGenesis:      "0",
			GenesisSupply:            "0",
			InflationPercentage_365D: math.LegacyZeroDec(),
			RewardsPerYear:           "0",
			LastRefreshDate:          time.Now().UTC().Format(time.RFC3339),
			ChainStatus:              "live",
			CurrentBlockHeight:       uint64(ctx.BlockHeight()),
			GenesisBlockHeight:       GenesisBlockHeight,
		}, nil
	}

	minter, err := k.mintKeeper.Minter.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get minter: %w", err)
	}

	blocksPerYearDec := math.LegacyNewDec(BlocksPerYear)
	rewardsPerBlockDec := minter.AnnualProvisions.Quo(blocksPerYearDec)
	rewardsPerBlock := rewardsPerBlockDec.TruncateInt()

	currentBlockHeight := uint64(ctx.BlockHeight())
	blocksSinceGenesis := int64(currentBlockHeight) - GenesisBlockHeight
	if blocksSinceGenesis < 0 {
		blocksSinceGenesis = 0
	}

	estimatedRewardsSinceGenesis := k.calculateRewardsSinceGenesis(
		ctx,
		totalSupplyAmount,
		blocksSinceGenesis,
		blocksPerYearDec,
	)

	var genesisSupplyAmount math.Int
	if totalSupplyAmount.GT(estimatedRewardsSinceGenesis) {
		genesisSupplyAmount = totalSupplyAmount.Sub(estimatedRewardsSinceGenesis)
	} else {
		genesisSupplyAmount = totalSupplyAmount.QuoRaw(2)
	}

	inflation365d := minter.Inflation.MulInt64(100)
	rewardsPerYear := minter.AnnualProvisions.TruncateInt()

	return &types.QueryCoinInfoResponse{
		TotalSupply:              totalSupplyAmount.String(),
		RewardsPerBlock:          rewardsPerBlock.String(),
		RewardsSinceGenesis:      estimatedRewardsSinceGenesis.String(),
		GenesisSupply:            genesisSupplyAmount.String(),
		InflationPercentage_365D: inflation365d,
		RewardsPerYear:           rewardsPerYear.String(),
		LastRefreshDate:          time.Now().UTC().Format(time.RFC3339),
		ChainStatus:              "live",
		CurrentBlockHeight:       currentBlockHeight,
		GenesisBlockHeight:       GenesisBlockHeight,
	}, nil
}

func (k Keeper) calculateRewardsSinceGenesis(
	ctx sdk.Context,
	totalSupply math.Int,
	blocksSinceGenesis int64,
	blocksPerYearDec math.LegacyDec,
) math.Int {
	if totalSupply.IsZero() {
		return math.ZeroInt()
	}

	earlyRate, _ := k.mintKeeper.GetEarlyPhaseInflationRate(ctx)
	growthRate, _ := k.mintKeeper.GetGrowthPhaseInflationRate(ctx)
	matureRate, _ := k.mintKeeper.GetMaturePhaseInflationRate(ctx)
	postCapRate, _ := k.mintKeeper.GetPostCapInflationRate(ctx)

	earlyThreshold := minttypes.HeliosToBaseUnits(minttypes.EarlyStageThreshold)
	growthThreshold := minttypes.HeliosToBaseUnits(minttypes.GrowthStageThreshold)
	matureThreshold := minttypes.HeliosToBaseUnits(minttypes.MatureStageThreshold)

	earlyRewardsPerBlock := math.LegacyNewDecFromInt(earlyThreshold).Mul(earlyRate).Quo(blocksPerYearDec)
	growthRewardsPerBlock := math.LegacyNewDecFromInt(growthThreshold).Mul(growthRate).Quo(blocksPerYearDec)
	matureRewardsPerBlock := math.LegacyNewDecFromInt(matureThreshold).Mul(matureRate).Quo(blocksPerYearDec)

	blocksSinceGenesisDec := math.LegacyNewDec(blocksSinceGenesis)
	totalSupplyDec := math.LegacyNewDecFromInt(totalSupply)

	if totalSupply.LT(earlyThreshold) {
		return earlyRewardsPerBlock.Mul(blocksSinceGenesisDec).TruncateInt()
	}

	var totalRewards math.LegacyDec
	phases := []struct {
		supply          math.Int
		rewardsPerBlock math.LegacyDec
	}{
		{earlyThreshold, earlyRewardsPerBlock},
		{growthThreshold.Sub(earlyThreshold), growthRewardsPerBlock},
		{matureThreshold.Sub(growthThreshold), matureRewardsPerBlock},
	}

	maxPhase := 0
	if totalSupply.LT(growthThreshold) {
		maxPhase = 1
		phases[1].supply = totalSupply.Sub(earlyThreshold)
	} else if totalSupply.LT(matureThreshold) {
		maxPhase = 2
		phases[2].supply = totalSupply.Sub(growthThreshold)
	} else {
		maxPhase = 2
	}

	for i := 0; i <= maxPhase; i++ {
		ratio := math.LegacyNewDecFromInt(phases[i].supply).Quo(totalSupplyDec)
		totalRewards = totalRewards.Add(phases[i].rewardsPerBlock.Mul(blocksSinceGenesisDec).Mul(ratio))
	}

	if totalSupply.GTE(matureThreshold) {
		postCapRewardsPerBlock := math.LegacyNewDecFromInt(totalSupply).Mul(postCapRate).Quo(blocksPerYearDec)
		postCapSupply := totalSupply.Sub(matureThreshold)
		postCapRatio := math.LegacyNewDecFromInt(postCapSupply).Quo(totalSupplyDec)
		totalRewards = totalRewards.Add(postCapRewardsPerBlock.Mul(blocksSinceGenesisDec).Mul(postCapRatio))
	}

	return totalRewards.TruncateInt()
}
