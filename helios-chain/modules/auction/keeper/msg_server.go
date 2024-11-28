package keeper

import (
	"context"

	"cosmossdk.io/math"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	"github.com/Helios-Chain-Labs/metrics"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"helios-core/helios-chain/modules/auction/types"
	chaintypes "helios-core/helios-chain/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	Keeper
	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,
		svcTags: metrics.Tags{
			"svc": "auction_h",
		},
	}
}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	k.SetParams(sdk.UnwrapSDKContext(c), msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) Bid(goCtx context.Context, msg *types.MsgBid) (*types.MsgBidResponse, error) {
	goCtx, doneFn := metrics.ReportFuncCallAndTimingCtx(goCtx, k.svcTags)
	defer doneFn()

	// prepare context
	ctx := sdk.UnwrapSDKContext(goCtx)

	round := k.GetAuctionRound(ctx)
	if msg.Round != round {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrBidRound, "current round is %d but got bid for %d", round, msg.Round)
	}
	// check valid bid
	lastBid := k.GetHighestBid(ctx)
	if msg.BidAmount.Amount.LT(lastBid.Amount.Amount) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrInvalidRequest, "Bid must exceed current highest bid")
	}

	// ensure last_bid * (1+min_next_increment_rate) <= new_bid
	params := k.GetParams(ctx)
	if lastBid.Amount.Amount.ToLegacyDec().Mul(math.LegacyOneDec().Add(params.MinNextBidIncrementRate)).GT(msg.BidAmount.Amount.ToLegacyDec()) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(sdkerrors.ErrInvalidRequest, "new bid should be bigger than last bid + min increment percentage")
	}

	// process bid
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "invalid sender address")
	}

	// deposit new bid
	newBidAmount := sdk.NewCoins(msg.BidAmount)
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, types.ModuleName, newBidAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("Bidder deposit failed", "senderAddr", senderAddr.String(), "coin", msg.BidAmount.String())
		return nil, errors.Wrap(err, "deposit failed")
	}

	// check first bidder
	isFirstBidder := !lastBid.Amount.Amount.IsPositive()
	if !isFirstBidder {
		err := k.refundLastBidder(ctx)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			return nil, err
		}
	}

	// set new bid to store
	k.SetBid(ctx, msg.Sender, msg.BidAmount)

	// emit typed event for bid
	auctionRound := k.GetAuctionRound(ctx)
	_ = ctx.EventManager().EmitTypedEvent(&types.EventBid{
		Bidder: msg.Sender,
		Amount: msg.BidAmount,
		Round:  auctionRound,
	})
	return &types.MsgBidResponse{}, nil
}

func (k msgServer) refundLastBidder(ctx sdk.Context) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	lastBid := k.GetHighestBid(ctx)
	lastBidAmount := lastBid.Amount.Amount
	lastBidder, err := sdk.AccAddressFromBech32(lastBid.Bidder)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error(err.Error())
		return err
	}

	bidAmount := sdk.NewCoins(sdk.NewCoin(chaintypes.HeliosCoin, lastBidAmount))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, lastBidder, bidAmount); err != nil {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Error("Bidder refund failed", "lastBidderAddr", lastBidder.String(), "coin", bidAmount.String())
		return errors.Wrap(err, "deposit failed")
	}

	return nil
}
