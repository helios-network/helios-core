// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package keeper

import (
	"fmt"
	"helios-core/helios-chain/x/evm/types"
	"os"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var _ types.EvmHooks = MultiEvmHooks{}

// MultiEvmHooks combine multiple evm hooks, all hook functions are run in array sequence
type MultiEvmHooks []types.EvmHooks

// NewMultiEvmHooks combine multiple evm hooks
func NewMultiEvmHooks(hooks ...types.EvmHooks) MultiEvmHooks {
	return hooks
}

// PostTxProcessing delegate the call to underlying hooks
func (mh MultiEvmHooks) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	logFile, _ := os.OpenFile("/tmp/helios-debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	fmt.Fprintf(logFile, "======> EVM MultiEvmHooks: Processing %d hooks\n", len(mh))
	for i := range mh {
		fmt.Fprintf(logFile, "======> EVM MultiEvmHooks: Executing hook %d of type %T\n", i, mh[i])
		if err := mh[i].PostTxProcessing(ctx, msg, receipt); err != nil {
			fmt.Fprintf(logFile, "======> EVM MultiEvmHooks: Hook %d failed: %v\n", i, err)
			return errorsmod.Wrapf(err, "EVM hook %T failed", mh[i])
		}
	}
	fmt.Fprintf(logFile, "======> EVM MultiEvmHooks: Successfully processed all hooks\n")
	return nil
}
