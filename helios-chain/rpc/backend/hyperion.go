package backend

import (
	"math"
	"time"

	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	rpctypes "helios-core/helios-chain/rpc/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

// GetAccountTransactionsByPageAndSize returns the transactions at the given page and size for the filtered by address
func (b *Backend) GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error) {
	if page == 0 || size == 0 || size > 100 {
		return nil, errors.New("invalid page or size parameters")
	}

	req := &hyperiontypes.QueryGetTransactionsByPageAndSizeRequest{
		Address: address.String(),
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
		FormatErc20: true,
	}
	res, err := b.queryClient.Hyperion.QueryGetTransactionsByPageAndSize(b.ctx, req)

	if err != nil {
		b.logger.Error("GetHyperionAccountTransferTxsByPageAndSize", "error", err)
		return []*hyperiontypes.QueryTransferTx{}, err
	}

	if res.Txs == nil {
		return []*hyperiontypes.QueryTransferTx{}, nil
	}

	return res.Txs, nil
}

func (b *Backend) GetAllHyperionTransferTxs(size hexutil.Uint64) ([]*hyperiontypes.QueryTransferTx, error) {
	if size == 0 || size > 100 {
		return nil, errors.New("invalid size parameters")
	}

	req := &hyperiontypes.QueryGetTransactionsByPageAndSizeRequest{
		Address: "",
		Pagination: &query.PageRequest{
			Offset: 0,
			Limit:  uint64(size),
		},
		FormatErc20: true,
	}
	res, err := b.queryClient.Hyperion.QueryGetTransactionsByPageAndSize(b.ctx, req)
	if err != nil || res.Txs == nil {
		return []*hyperiontypes.QueryTransferTx{}, err
	}

	return res.Txs, nil
}

func (b *Backend) GetHyperionChains() ([]*rpctypes.HyperionChainRPC, error) {
	res, err := b.queryClient.Hyperion.QueryGetCounterpartyChainParamsWithComplemetaryInfo(b.ctx, &hyperiontypes.QueryGetCounterpartyChainParamsWithComplemetaryInfoRequest{})
	if err != nil {
		b.logger.Error("GetHyperionChains", "error", err)
		return nil, err
	}

	counterpartyChainParams := make([]*rpctypes.HyperionChainRPC, 0)

	for _, chain := range res.CounterpartyChainParamsWithComplemetaryInfo {

		estimatedCurrentBlock := uint64(0)

		if chain.ComplemetaryInfo.LatestObservedBlockTime != 0 {
			currentTime := time.Now().UnixMilli()
			lastObservedTimeMs := chain.ComplemetaryInfo.LatestObservedBlockTime * 1000
			timeSinceLastObservedMs := currentTime - int64(lastObservedTimeMs)
			blocksSinceLastObserved := math.Floor(float64(uint64(timeSinceLastObservedMs) / chain.ComplemetaryInfo.AverageCounterpartyBlockTime))
			estimatedCurrentBlock = chain.ComplemetaryInfo.LatestObservedBlockHeight + uint64(blocksSinceLastObserved)
		}

		counterpartyChainParams = append(counterpartyChainParams, &rpctypes.HyperionChainRPC{
			HyperionContractAddress:           chain.CounterpartyChainParams.BridgeCounterpartyAddress,
			ChainId:                           chain.CounterpartyChainParams.BridgeChainId,
			Name:                              chain.CounterpartyChainParams.BridgeChainName,
			ChainType:                         chain.CounterpartyChainParams.BridgeChainType,
			Logo:                              chain.CounterpartyChainParams.BridgeChainLogo,
			HyperionId:                        chain.CounterpartyChainParams.HyperionId,
			Paused:                            chain.CounterpartyChainParams.Paused,
			AverageCounterpartyBlockTime:      chain.ComplemetaryInfo.AverageCounterpartyBlockTime,
			LatestObservedBlockHeight:         chain.ComplemetaryInfo.LatestObservedBlockHeight,
			LatestObservedBlockTime:           chain.ComplemetaryInfo.LatestObservedBlockTime,
			TargetBatchTimeout:                chain.CounterpartyChainParams.TargetBatchTimeout,
			TargetOutgoingTxTimeout:           chain.CounterpartyChainParams.TargetOutgoingTxTimeout,
			EstimatedCounterpartyCurrentBlock: estimatedCurrentBlock,
		})
	}

	bankRes, _ := b.queryClient.Bank.DenomFullMetadata(b.ctx, &banktypes.QueryDenomFullMetadataRequest{
		Denom: evmtypes.DefaultEVMDenom,
	})

	resAvgBlockTime, err := b.queryClient.Hyperion.QueryGetHeliosEffectiveAverageBlockTime(b.ctx, &hyperiontypes.QueryGetHeliosEffectiveAverageBlockTimeRequest{})
	if err != nil {
		b.logger.Error("GetHyperionEffectiveAverageBlockTime", "error", err)
		return nil, err
	}

	latestBlock, err := b.GetBlockByNumber(rpctypes.EthLatestBlockNumber, false)
	if err != nil {
		b.logger.Error("GetHyperionEffectiveAverageBlockTime", "error", err)
		return nil, err
	}

	counterpartyChainParams = append(counterpartyChainParams, &rpctypes.HyperionChainRPC{
		HyperionContractAddress:           evmtypes.HyperionPrecompileAddress,
		ChainId:                           uint64(b.chainID.Int64()),
		Name:                              "Helios",
		ChainType:                         "evm",
		Logo:                              bankRes.Metadata.Metadata.Logo,
		HyperionId:                        0,
		Paused:                            false,
		AverageCounterpartyBlockTime:      resAvgBlockTime.AverageBlockTime,
		LatestObservedBlockHeight:         uint64(latestBlock["number"].(hexutil.Uint64)),
		LatestObservedBlockTime:           uint64(latestBlock["timestamp"].(hexutil.Uint64)),
		TargetBatchTimeout:                0,
		TargetOutgoingTxTimeout:           0,
		EstimatedCounterpartyCurrentBlock: uint64(latestBlock["number"].(hexutil.Uint64)),
	})

	return counterpartyChainParams, nil
}

func (b *Backend) GetHyperionHistoricalFees(hyperionId uint64) (*hyperiontypes.QueryHistoricalFeesResponse, error) {
	res, err := b.queryClient.Hyperion.QueryHistoricalFees(b.ctx, &hyperiontypes.QueryHistoricalFeesRequest{
		HyperionId: hyperionId,
	})
	if err != nil {
		b.logger.Error("GetHyperionHistoricalFees", "error", err)
		return nil, err
	}
	return res, nil
}

func (b *Backend) GetValidatorHyperionData(address common.Address) (*hyperiontypes.OrchestratorData, error) {
	res, err := b.queryClient.Hyperion.QueryGetOrchestratorData(b.ctx, &hyperiontypes.QueryGetOrchestratorDataRequest{
		OrchestratorAddress: address.String(),
	})
	if err != nil {
		b.logger.Error("GetValidatorHyperionData", "error", err)
		return nil, err
	}
	return res.OrchestratorData, nil
}

func (b *Backend) GetWhitelistedAddresses(hyperionId uint64) ([]string, error) {
	res, err := b.queryClient.Hyperion.QueryGetWhitelistedAddresses(b.ctx, &hyperiontypes.QueryGetWhitelistedAddressesRequest{
		HyperionId: hyperionId,
	})
	if err != nil {
		b.logger.Error("GetWhitelistedAddresses", "error", err)
		return nil, err
	}
	return res.Addresses, nil
}

func (b *Backend) GetHyperionProjectedCurrentNetworkHeight(hyperionId uint64) (uint64, error) {
	res, err := b.queryClient.Hyperion.QueryEstimateLatestBlockOfChain(b.ctx, &hyperiontypes.QueryEstimateLatestBlockOfChainRequest{
		HyperionId: hyperionId,
	})
	if err != nil {
		b.logger.Error("HyperionEstimateLatestBlockOfChain", "error", err)
		return 0, err
	}
	return res.LatestBlock, nil
}

func (b *Backend) GetHyperionNonceAlreadyObserved(hyperionId uint64, nonce uint64) (bool, error) {
	res, err := b.queryClient.Hyperion.QueryIsNonceAlreadyObserved(b.ctx, &hyperiontypes.QueryIsNonceAlreadyObservedRequest{
		HyperionId: hyperionId,
		Nonce:      nonce,
	})
	if err != nil {
		b.logger.Error("GetHyperionNonceAlreadyObserved", "error", err)
		return false, err
	}
	return res.IsNonceAlreadyObserved, nil
}

func (b *Backend) GetHyperionSkippedNonces(hyperionId uint64) ([]*hyperiontypes.SkippedNonceFullInfo, error) {
	res, err := b.queryClient.Hyperion.QueryGetSkippedNonces(b.ctx, &hyperiontypes.QueryGetSkippedNoncesRequest{
		HyperionId: hyperionId,
	})
	if err != nil {
		b.logger.Error("GetHyperionSkippedNonces", "error", err)
		return nil, err
	}
	if res.SkippedNonces == nil {
		return []*hyperiontypes.SkippedNonceFullInfo{}, nil
	}
	if len(res.SkippedNonces) == 0 {
		return []*hyperiontypes.SkippedNonceFullInfo{}, nil
	}
	return res.SkippedNonces, nil
}

func (b *Backend) GetAllHyperionSkippedNonces() ([]*hyperiontypes.SkippedNonceFullInfoWithHyperionId, error) {
	res, err := b.queryClient.Hyperion.QueryGetAllSkippedNonces(b.ctx, &hyperiontypes.QueryGetAllSkippedNoncesRequest{})
	if err != nil {
		b.logger.Error("GetAllHyperionSkippedNonces", "error", err)
		return nil, err
	}
	return res.SkippedNonces, nil
}
