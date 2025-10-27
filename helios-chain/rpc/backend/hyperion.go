package backend

import (
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
	res, err := b.queryClient.Hyperion.Params(b.ctx, &hyperiontypes.QueryParamsRequest{})
	if err != nil {
		b.logger.Error("GetHyperionChains", "error", err)
		return nil, err
	}

	counterpartyChainParams := make([]*rpctypes.HyperionChainRPC, 0)

	for _, chain := range res.Params.CounterpartyChainParams {
		counterpartyChainParams = append(counterpartyChainParams, &rpctypes.HyperionChainRPC{
			HyperionContractAddress: chain.BridgeCounterpartyAddress,
			ChainId:                 chain.BridgeChainId,
			Name:                    chain.BridgeChainName,
			ChainType:               chain.BridgeChainType,
			Logo:                    chain.BridgeChainLogo,
			HyperionId:              chain.HyperionId,
			Paused:                  chain.Paused,
		})
	}

	bankRes, _ := b.queryClient.Bank.DenomFullMetadata(b.ctx, &banktypes.QueryDenomFullMetadataRequest{
		Denom: evmtypes.DefaultEVMDenom,
	})

	counterpartyChainParams = append(counterpartyChainParams, &rpctypes.HyperionChainRPC{
		HyperionContractAddress: evmtypes.HyperionPrecompileAddress,
		ChainId:                 uint64(b.chainID.Int64()),
		Name:                    "Helios",
		ChainType:               "evm",
		Logo:                    bankRes.Metadata.Metadata.Logo,
		HyperionId:              0,
		Paused:                  false,
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
