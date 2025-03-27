package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "helios-core/helios-chain/x/evm/types"
	feedistributiontypes "helios-core/helios-chain/x/feedistribution/types"
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
	if contract == nil || msg.GasPrice().Sign() <= 0 {
		return nil
	}

	// check if the fees are globally enabled or if the
	// developer shares are set to zero
	params := k.GetParams(ctx)
	if !params.EnableFeeDistribution || params.DeveloperShares.IsZero() {
		return nil
	}

	// !TODO: fix this
	// Check if the address is a precompile
	// evmParams := k.evmKeeper.GetParams(ctx)
	// containsPrecompile := slices.Contains(evmParams.ActivePrecompiles, contract.String())
	// if containsPrecompile {
	// 	return nil
	// }

	// Verify this is actually a contract
	if !k.IsContract(ctx, *contract) {
		return nil
	}

	// if the contract is not registered to receive fees, do nothing
	revenue, found := k.GetRevenue(ctx, *contract)
	if !found {
		return nil
	}

	withdrawer := revenue.GetWithdrawerAddr()
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
		return errorsmod.Wrapf(
			err,
			"fee collector account failed to distribute developer fees (%s) to withdraw address %s. contract %s",
			fees, withdrawer, contract,
		)
	}

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				feedistributiontypes.EventTypeDistributeFees,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.From().String()),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyContract, contract.String()),
				sdk.NewAttribute(feedistributiontypes.AttributeKeyWithdrawerAddress, withdrawer.String()),
				sdk.NewAttribute(sdk.AttributeKeyAmount, developerFee.String()),
			),
		},
	)

	return nil
}
