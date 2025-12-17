package backend

import (
	"context"

	rpctypes "helios-core/helios-chain/rpc/types"
	chaininfotypes "helios-core/helios-chain/x/chaininfo/types"
)

func (b *Backend) GetCoinInfo() (*rpctypes.CoinInfoRPC, error) {
	queryClient := chaininfotypes.NewQueryClient(b.clientCtx)
	req := &chaininfotypes.QueryCoinInfoRequest{}
	res, err := queryClient.CoinInfo(context.Background(), req)
	if err != nil {
		return nil, err
	}

	return &rpctypes.CoinInfoRPC{
		TotalSupply:             res.TotalSupply,
		RewardsPerBlock:         res.RewardsPerBlock,
		RewardsSinceGenesis:     res.RewardsSinceGenesis,
		GenesisSupply:           res.GenesisSupply,
		InflationPercentage365D: res.InflationPercentage_365D,
		RewardsPerYear:          res.RewardsPerYear,
		LastRefreshDate:         res.LastRefreshDate,
		ChainStatus:             res.ChainStatus,
		CurrentBlockHeight:      res.CurrentBlockHeight,
		GenesisBlockHeight:      res.GenesisBlockHeight,
	}, nil
}
