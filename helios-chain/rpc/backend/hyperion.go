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
func (b *Backend) GetHyperionAccountTransferTxsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*hyperiontypes.TransferTx, error) {
	if page == 0 || size == 0 || size > 100 {
		return nil, errors.New("invalid page or size parameters")
	}

	b.logger.Info("GetHyperionAccountTransferTxsByPageAndSize", "address", address, "offset", (uint64(page)-1)*uint64(size), "limit", uint64(size))

	req := &hyperiontypes.QueryGetTransactionsByPageAndSizeRequest{
		Address: address.String(),
		Pagination: &query.PageRequest{
			Offset: (uint64(page) - 1) * uint64(size),
			Limit:  uint64(size),
		},
		FormatErc20: true,
	}
	res, err := b.queryClient.Hyperion.QueryGetTransactionsByPageAndSize(b.ctx, req)
	if err != nil || res.Txs == nil {
		b.logger.Error("GetHyperionAccountTransferTxsByPageAndSize", "error", err)
		return []*hyperiontypes.TransferTx{}, err
	}

	return res.Txs, nil
}

func (b *Backend) GetAllHyperionTransferTxs(size hexutil.Uint64) ([]*hyperiontypes.TransferTx, error) {
	if size == 0 || size > 100 {
		return nil, errors.New("invalid size parameters")
	}

	req := &hyperiontypes.QueryGetTransactionsByPageAndSizeRequest{
		Address: "",
		Pagination: &query.PageRequest{
			Offset: (uint64(1) - 1) * uint64(size),
			Limit:  uint64(size),
		},
		FormatErc20: true,
	}
	res, err := b.queryClient.Hyperion.QueryGetTransactionsByPageAndSize(b.ctx, req)
	if err != nil || res.Txs == nil {
		return []*hyperiontypes.TransferTx{}, err
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
	})

	return counterpartyChainParams, nil
}
