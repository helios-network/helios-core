package keeper

import (
	"context"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

func (k msgServer) UpdateChainSmartContract(c context.Context, msg *types.MsgUpdateChainSmartContract) (*types.MsgUpdateChainSmartContractResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	// validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	// todo check msg.Sender is testnet admin

	params := k.Keeper.GetParams(ctx)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			counterpartyChainParam.BridgeCounterpartyAddress = msg.BridgeContractAddress
			counterpartyChainParam.BridgeContractStartHeight = msg.BridgeContractStartHeight
			k.Keeper.SetCounterpartyChainParams(ctx, counterpartyChainParam.HyperionId, counterpartyChainParam)

			k.Keeper.setLastObservedEventNonce(ctx, counterpartyChainParam.HyperionId, 0)
			k.Keeper.SetLastObservedEthereumBlockHeight(ctx, counterpartyChainParam.HyperionId, msg.BridgeContractStartHeight-1, uint64(ctx.BlockHeight()))
			k.Keeper.SetID(ctx, types.GetLastOutgoingBatchIDKey(counterpartyChainParam.HyperionId), 0)
			break
		}
	}

	return &types.MsgUpdateChainSmartContractResponse{}, nil
}

func (k msgServer) UpdateChainLogo(c context.Context, msg *types.MsgUpdateChainLogo) (*types.MsgUpdateChainLogoResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	params := k.Keeper.GetParams(ctx)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			counterpartyChainParam.BridgeChainLogo = msg.Logo
			k.Keeper.SetCounterpartyChainParams(ctx, counterpartyChainParam.HyperionId, counterpartyChainParam)
			break
		}
	}

	return &types.MsgUpdateChainLogoResponse{}, nil
}

func (k msgServer) UpdateChainName(c context.Context, msg *types.MsgUpdateChainName) (*types.MsgUpdateChainNameResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	params := k.Keeper.GetParams(ctx)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			counterpartyChainParam.BridgeChainName = msg.Name
			k.Keeper.SetCounterpartyChainParams(ctx, counterpartyChainParam.HyperionId, counterpartyChainParam)
			break
		}
	}

	return &types.MsgUpdateChainNameResponse{}, nil
}

func (k msgServer) DeleteChain(c context.Context, msg *types.MsgDeleteChain) (*types.MsgDeleteChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	ctx := sdk.UnwrapSDKContext(c)
	params := k.Keeper.GetParams(ctx)

	for i, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			params.CounterpartyChainParams = append(params.CounterpartyChainParams[:i], params.CounterpartyChainParams[i+1:]...)
			break
		}
	}

	k.Keeper.SetParams(ctx, params)

	return &types.MsgDeleteChainResponse{}, nil
}

func (k msgServer) ClearValset(c context.Context, msg *types.MsgClearValset) (*types.MsgClearValsetResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	ctx := sdk.UnwrapSDKContext(c)
	params := k.Keeper.GetParams(ctx)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			k.Keeper.setLastObservedEventNonce(ctx, counterpartyChainParam.HyperionId, 0)
		}
	}

	return &types.MsgClearValsetResponse{}, nil
}

func (k msgServer) ForceSetValsetAndLastObservedEventNonce(c context.Context, msg *types.MsgForceSetValsetAndLastObservedEventNonce) (*types.MsgForceSetValsetAndLastObservedEventNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	hyperionParams := k.Keeper.GetCounterpartyChainParams(ctx)[msg.HyperionId]

	if hyperionParams.Initializer != msg.Signer {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	valset := k.Keeper.GetValset(ctx, msg.HyperionId, msg.Valset.Nonce)
	if valset == nil {
		k.Keeper.SetLastObservedValset(ctx, msg.HyperionId, *msg.Valset)
	}

	k.Keeper.setLastObservedEventNonce(ctx, msg.HyperionId, msg.LastObservedEventNonce)
	k.Keeper.SetLastObservedEthereumBlockHeight(ctx, msg.HyperionId, msg.LastObservedEthereumBlockHeight, uint64(ctx.BlockHeight()))
	k.Keeper.SetID(ctx, types.GetLastOutgoingBatchIDKey(msg.HyperionId), msg.LastObservedEventNonce)
	k.Keeper.SetLastUnbondingBlockHeight(ctx, uint64(ctx.BlockHeight()))

	return &types.MsgForceSetValsetAndLastObservedEventNonceResponse{}, nil
}
