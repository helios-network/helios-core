// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	rpctypes "helios-core/helios-chain/rpc/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (b *Backend) GetStakedPowers(delegatorAddress common.Address, blockNum rpctypes.BlockNumber) (string, error) {

	delegatorBech32Addr := sdk.AccAddress(delegatorAddress.Bytes())
	// valAddr := sdk.ValAddress(delegatorBech32Addr)

	// delegatorEthAddr := common.BytesToAddress(delegatorBech32Addr.Bytes())

	// req := &stakingtypes.QueryDelegationRequest{
	// 	DelegatorAddr: addr.String(),
	// 	ValidatorAddr: addr.String(),
	// }
	// res, err := b.queryClient.Staking.Delegation(b.ctx, req)
	// if err != nil {
	// 	return nil, nil
	// }

	// res.DelegationResponse.Balance.Amount

	return delegatorBech32Addr.String(), nil
}
