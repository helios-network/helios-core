package auction

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Helios-Chain-Labs/metrics"

	auctiontypes "helios-core/helios-chain/modules/auction/types"
	chaintypes "helios-core/helios-chain/types"
)

func (am AppModule) EndBlocker(ctx sdk.Context) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, am.svcTags)
	defer doneFn()

	// trigger auction settlement
	endingTimeStamp := am.keeper.GetEndingTimeStamp(ctx)

	if ctx.BlockTime().Unix() < endingTimeStamp {
		return
	}

	logger := ctx.Logger().With("module", "auction", "EndBlocker", ctx.BlockHeight())
	logger.Info("Settling auction round...", "blockTimestamp", ctx.BlockTime().Unix(), "endingTimeStamp", endingTimeStamp)
	auctionModuleAddress := am.accountKeeper.GetModuleAddress(auctiontypes.ModuleName)

	// get and validate highest bid
	lastBid := am.keeper.GetHighestBid(ctx)
	lastBidAmount := lastBid.Amount.Amount

	maxHeliosCap := am.keeper.GetParams(ctx).HeliosBasketMaxCap

	// settle auction round
	if lastBidAmount.IsPositive() && lastBid.Bidder != "" {
		lastBidder, err := sdk.AccAddressFromBech32(lastBid.Bidder)
		if err != nil {
			metrics.ReportFuncError(am.svcTags)
			logger.Info(err.Error())
			return
		}

		// burn exactly module's helios amount received from bid
		heliosBalanceInAuctionModule := am.bankKeeper.GetBalance(ctx, auctionModuleAddress, chaintypes.HeliosCoin)
		if heliosBalanceInAuctionModule.IsPositive() {
			heliosBurnAmount := sdk.NewCoin(chaintypes.HeliosCoin, lastBidAmount)
			err = am.bankKeeper.BurnCoins(ctx, auctiontypes.ModuleName, sdk.NewCoins(heliosBurnAmount))

			if err != nil {
				metrics.ReportFuncError(am.svcTags)
				logger.Info(err.Error())
			}
		}

		// send tokens to winner or append to next auction round
		coins := am.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)
		for _, coin := range coins {
			// cap the amount of Helios that can be sent to the winner
			if coin.Denom == chaintypes.HeliosCoin {
				if coin.Amount.GT(maxHeliosCap) {
					coin.Amount = maxHeliosCap
				}
			}

			if err := am.bankKeeper.SendCoinsFromModuleToAccount(ctx, auctiontypes.ModuleName, lastBidder, sdk.NewCoins(coin)); err != nil {
				metrics.ReportFuncError(am.svcTags)
				am.keeper.Logger(ctx).Error("Transferring coins to winner failed")
			}
		}

		// emit typed event for auction result
		auctionRound := am.keeper.GetAuctionRound(ctx)

		// Store the auction result, so that it can be queried later
		am.keeper.SetLastAuctionResult(ctx, auctiontypes.LastAuctionResult{
			Winner: lastBid.Bidder,
			Amount: lastBid.Amount,
			Round:  auctionRound,
		})

		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&auctiontypes.EventAuctionResult{
			Winner: lastBid.Bidder,
			Amount: lastBid.Amount,
			Round:  auctionRound,
		})

		// clear bid
		am.keeper.DeleteBid(ctx)
	}

	// advance auctionRound, endingTimestamp
	nextRound := am.keeper.AdvanceNextAuctionRound(ctx)
	nextEndingTimestamp := am.keeper.AdvanceNextEndingTimeStamp(ctx)
	// ping exchange module to flush fee for next round
	balances := am.exchangeKeeper.WithdrawAllAuctionBalances(ctx)

	newBasket := am.bankKeeper.GetAllBalances(ctx, auctionModuleAddress)

	// for correctness, emit the correct HELIOS value in the new basket in the event the HELIOS balances exceed the cap
	newHeliosAmount := newBasket.AmountOf(chaintypes.HeliosCoin)
	if newHeliosAmount.GT(maxHeliosCap) {
		excessHelios := newHeliosAmount.Sub(maxHeliosCap)
		newBasket = newBasket.Sub(sdk.NewCoin(chaintypes.HeliosCoin, excessHelios))
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&auctiontypes.EventAuctionStart{
		Round:           nextRound,
		EndingTimestamp: nextEndingTimestamp,
		NewBasket:       newBasket,
	})

	if len(balances) == 0 {
		logger.Info("😢 Received empty coin basket from exchange")
	} else {
		logger.Info("💰 Auction module received", balances.String(), "new auction basket is now", newBasket.String())
	}
}
