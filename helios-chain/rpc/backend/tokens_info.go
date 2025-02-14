// Copyright Jeremy Guyet
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	"helios-core/helios-chain/x/erc20/types"

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
	req := &types.QueryTokenPairsRequest{
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
		token["symbol"] = pair.Denom
		token["enabled"] = pair.Enabled
		token["owner"] = pair.ContractOwner

		tokens = append(tokens, token)
	}

	return tokens, nil
}
