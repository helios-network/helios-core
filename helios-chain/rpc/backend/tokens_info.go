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

func (b *Backend) GetTokensByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]map[string]interface{}, error) {
	// Create pagination request
	pageReq := &query.PageRequest{
		Offset:     uint64((page - 1) * size),
		Limit:      uint64(size),
		CountTotal: true,
	}

	// Create the query request
	req := &erc20types.QueryTokenPairsRequest{
		Pagination: pageReq,
	}

	// Query token pairs using the ERC20 keeper
	res, err := b.queryClient.Erc20.TokenPairs(b.ctx, req)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to get token pairs")
	}

	tokens := make([]map[string]interface{}, 0, len(res.TokenPairs))

	// Format each token pair into a map
	for _, pair := range res.TokenPairs {
		token := make(map[string]interface{})

		// Convert ERC20 address to checksum format
		erc20Addr := common.HexToAddress(pair.Erc20Address)

		token["address"] = erc20Addr.Hex()
		token["denom"] = pair.Denom
		token["enabled"] = pair.Enabled
		token["owner"] = pair.ContractOwner

		tokens = append(tokens, token)
	}

	return tokens, nil
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

	bankRes, err := b.queryClient.Bank.DenomMetadata(b.ctx, &banktypes.QueryDenomMetadataRequest{
		Denom: erc20Res.TokenPair.Denom,
	})

	if err != nil {
		return &rpctypes.TokenDetails{
			Address: tokenAddress,
			Denom:   erc20Res.TokenPair.Denom,
		}, nil
	}

	logo := bankRes.Metadata.Logo

	if logo == "" {
		// set default logo
		logo, err = rpctypes.GenerateTokenLogoBase64(bankRes.Metadata.Symbol)

		if err != nil {
			logo = ""
		}
	}

	return &rpctypes.TokenDetails{
		Address:     tokenAddress,
		Denom:       erc20Res.TokenPair.Denom,
		Symbol:      bankRes.Metadata.Symbol,
		Decimals:    bankRes.Metadata.Decimals,
		Description: bankRes.Metadata.Description,
		Logo:        logo,
		Holders:     0,
	}, nil
}
