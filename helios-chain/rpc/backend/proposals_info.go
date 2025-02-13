// Copyright Jeremy Guyet
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (b *Backend) GetProposalsByPageAndSize(page hexutil.Uint64, size hexutil.Uint64) ([]*govtypes.Proposal, error) {
	proposals, err := b.queryClient.Gov.Proposals(b.ctx, &govtypes.QueryProposalsRequest{}) //.Params(b.ctx, &govtypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}
	return proposals.Proposals, nil
}
