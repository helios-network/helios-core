// Copyright Jeremy Guyet

package backend

import (
	erc20types "helios-core/helios-chain/x/erc20/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *Backend) GetTokensByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error) {
	// Create pagination request
	pageReq := &query.PageRequest{
		Offset:     uint64((page - 1) * size),
		Limit:      uint64(size),
		CountTotal: false,
	}

	res, err := b.queryClient.Bank.DenomsFullMetadata(b.ctx, &banktypes.QueryDenomsFullMetadataRequest{
		Pagination: pageReq,
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to get token pairs")
	}

	metadatas := make([]*banktypes.FullMetadata, len(res.Metadatas))

	for i, metadata := range res.Metadatas {
		metadatas[i] = &metadata
	}

	return metadatas, nil
}

func (b *Backend) GetTokenDetails(tokenAddress common.Address) (*banktypes.FullMetadata, error) {

	erc20Res, err := b.queryClient.Erc20.TokenPair(b.ctx, &erc20types.QueryTokenPairRequest{
		Token: tokenAddress.String(),
	})
	if err != nil {
		return nil, err
	}

	bankRes, err := b.queryClient.Bank.DenomFullMetadata(b.ctx, &banktypes.QueryDenomFullMetadataRequest{
		Denom: erc20Res.TokenPair.Denom,
	})

	if err != nil {
		return nil, err
	}

	return &bankRes.Metadata, nil
}

func (b *Backend) GetTokensDetails(tokenAddresses []common.Address) ([]*banktypes.FullMetadata, error) {
	// todo optimize this
	metadatas := make([]*banktypes.FullMetadata, len(tokenAddresses))

	for i, tokenAddress := range tokenAddresses {
		metadata, err := b.GetTokenDetails(tokenAddress)
		if err != nil {
			return nil, err
		}
		metadatas[i] = metadata
	}

	return metadatas, nil
}

func (b *Backend) GetTokensByChainIdAndPageAndSize(chainId uint64, page hexutil.Uint64, size hexutil.Uint64) ([]*banktypes.FullMetadata, error) {

	pageReq := &query.PageRequest{
		Offset:     uint64((page - 1) * size),
		Limit:      uint64(size),
		CountTotal: true,
	}

	res, err := b.queryClient.Bank.DenomsByChainId(b.ctx, &banktypes.QueryDenomsByChainIdRequest{
		ChainId:             chainId,
		Pagination:          pageReq,
		OrderByHoldersCount: true,
	})

	metadatas := make([]*banktypes.FullMetadata, len(res.Metadatas))

	if err != nil {
		return metadatas, err
	}

	for i, metadata := range res.Metadatas {
		metadatas[i] = &metadata
	}

	return metadatas, nil
}
