// Copyright Jeremy Guyet

package backend

import (
	rpctypes "helios-core/helios-chain/rpc/types"

	erc20types "helios-core/helios-chain/x/erc20/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *Backend) GetTokensByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]banktypes.FullMetadata, error) {
	// Create pagination request
	pageReq := &query.PageRequest{
		Offset:     uint64((page - 1) * size),
		Limit:      uint64(size),
		CountTotal: true,
	}

	res, err := b.queryClient.Bank.DenomsFullMetadata(b.ctx, &banktypes.QueryDenomsFullMetadataRequest{
		Pagination: pageReq,
	})
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to get token pairs")
	}

	return res.Metadatas, nil
}

func (b *Backend) GetTokenDetails(tokenAddress common.Address) (*rpctypes.TokenDetails, error) {

	// Get ERC20 balances using erc20 query
	// Create the query request
	erc20Req := &erc20types.QueryTokenPairRequest{
		Token: tokenAddress.String(),
	}
	erc20Res, err := b.queryClient.Erc20.TokenPair(b.ctx, erc20Req)
	if err != nil {
		return nil, err
	}

	bankRes, err := b.queryClient.Bank.DenomFullMetadata(b.ctx, &banktypes.QueryDenomFullMetadataRequest{
		Denom: erc20Res.TokenPair.Denom,
	})

	if err != nil {
		return &rpctypes.TokenDetails{
			Address: tokenAddress,
			Denom:   erc20Res.TokenPair.Denom,
		}, nil
	}

	return &rpctypes.TokenDetails{
		Address:       tokenAddress,
		Denom:         erc20Res.TokenPair.Denom,
		Symbol:        bankRes.Metadata.Metadata.Symbol,
		Decimals:      bankRes.Metadata.Metadata.Decimals,
		Description:   bankRes.Metadata.Metadata.Description,
		Logo:          bankRes.Metadata.Metadata.Logo,
		Holders:       bankRes.Metadata.HoldersCount,
		TotalSupply:   (*hexutil.Big)(bankRes.Metadata.TotalSupply.BigInt()),
		TotalSupplyUI: bankRes.Metadata.TotalSupply.String(),
	}, nil
}
