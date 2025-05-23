// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:LGPL-3.0-only

package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"golang.org/x/exp/slices"

	evmtypes "helios-core/helios-chain/x/evm/types"

	types "helios-core/helios-chain/x/revenue/v1/types"
)

var _ evmtypes.EvmHooks = Hooks{}

// Hooks wrapper struct for fees keeper
type Hooks struct {
	k Keeper
}

// Hooks return the wrapper hooks struct for the Keeper
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// PostTxProcessing is a wrapper for calling the EVM PostTxProcessing hook on
// the module keeper
func (h Hooks) PostTxProcessing(ctx sdk.Context, msg core.Message, receipt *ethtypes.Receipt) error {
	return h.k.PostTxProcessing(ctx, msg, receipt)
}

func (h Hooks) PostContractCreation(ctx sdk.Context, contractAddress common.Address, deployerAddress sdk.AccAddress) error {
	return h.k.PostContractCreation(ctx, contractAddress, deployerAddress)
}

// PostTxProcessing implements EvmHooks.PostTxProcessing. After each successful
// interaction with a registered contract, the contract deployer (or, if set,
// the withdraw address) receives a share from the transaction fees paid by the
// transaction sender.
func (k Keeper) PostTxProcessing(
	ctx sdk.Context,
	msg core.Message,
	receipt *ethtypes.Receipt,
) error {
	contract := msg.To()
	// when baseFee and minGasPrice in freemarker module are both 0
	// the user may send a transaction with gasPrice of 0 to the precompiled contract
	if contract == nil || msg.GasPrice().Sign() <= 0 {
		return nil
	}

	// check if the fees are globally enabled or if the
	// developer shares are set to zero
	params := k.GetParams(ctx)
	if !params.EnableRevenue || params.DeveloperShares.IsZero() {
		return nil
	}

	evmParams := k.evmKeeper.GetParams(ctx)

	var withdrawer sdk.AccAddress
	containsPrecompile := slices.Contains(evmParams.ActiveStaticPrecompiles, contract.String())

	if containsPrecompile {
		return nil
	}

	// if the contract is not a precompile, check if the contract is registered in the revenue module.
	// else, return and avoid performing unnecessary logic
	revenue, found := k.GetRevenue(ctx, *contract)
	if !found {
		return nil
	}

	withdrawer = revenue.GetWithdrawerAddr()
	if len(withdrawer) == 0 {
		withdrawer = revenue.GetDeployerAddr()
	}

	// calculate fees to be paid
	txFee := math.NewIntFromUint64(receipt.GasUsed).Mul(math.NewIntFromBigInt(msg.GasPrice()))
	developerFee := (params.DeveloperShares).MulInt(txFee).TruncateInt()
	evmDenom := k.evmKeeper.GetParams(ctx).EvmDenom
	fees := sdk.Coins{{Denom: evmDenom, Amount: developerFee}}

	// distribute the fees to the contract deployer / withdraw address
	err := k.bankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		k.feeCollectorName,
		withdrawer,
		fees,
	)
	if err != nil {
		fmt.Println("Error distributing developer fees:", err)
		// return errorsmod.Wrapf(
		// 	err,
		// 	"fee collector account failed to distribute developer fees (%s %s) to withdraw address %s. contract %s",
		// 	fees, evmDenom, withdrawer, contract,
		// )
		return nil
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeDistributeDevRevenue,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.From().String()),
				sdk.NewAttribute(types.AttributeKeyContract, contract.String()),
				sdk.NewAttribute(types.AttributeKeyWithdrawerAddress, withdrawer.String()),
				sdk.NewAttribute(sdk.AttributeKeyAmount, developerFee.String()),
			),
		},
	)

	return nil
}

// PostContractCreation is a hook that is called after a contract is created in the EVM.
// It automatically registers the contract for revenue distribution if revenue is enabled.
// The contract is registered with the deployer address as both the deployer and withdrawer.
// This allows for automatic fee distribution to the contract deployer without requiring
// a separate registration transaction.
func (k Keeper) PostContractCreation(
	ctx sdk.Context,
	contractAddress common.Address,
	deployerAddress sdk.AccAddress,
) error {

	// check if the fees are globally enabled or if the
	// developer shares are set to zero
	params := k.GetParams(ctx)
	if !params.EnableRevenue || params.DeveloperShares.IsZero() {
		return nil
	}
	if k.IsRevenueRegistered(ctx, contractAddress) {
		return nil
	}

	// prevent storing the same address for deployer and withdrawer
	revenue := types.NewRevenue(contractAddress, deployerAddress, deployerAddress)
	k.SetRevenue(ctx, revenue)
	k.SetDeployerMap(ctx, deployerAddress, contractAddress)

	return nil
}
