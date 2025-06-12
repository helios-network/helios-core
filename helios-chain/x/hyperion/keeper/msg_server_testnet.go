package keeper

import (
	"context"
	"strconv"

	cmn "helios-core/helios-chain/precompiles/common"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Helios-Chain-Labs/metrics"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/hyperion/types"
)

func (k msgServer) ChangeInitializer(c context.Context, msg *types.MsgChangeInitializer) (*types.MsgChangeInitializerResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	if msg.NewInitializer == "" {
		return nil, errors.Wrap(types.ErrInvalid, "NewInitializer cannot be empty")
	}

	params := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(params.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}
	params.Initializer = msg.NewInitializer

	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, params)

	return &types.MsgChangeInitializerResponse{}, nil
}

func (k msgServer) UpdateChainSmartContract(c context.Context, msg *types.MsgUpdateChainSmartContract) (*types.MsgUpdateChainSmartContractResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.BridgeCounterpartyAddress = msg.BridgeContractAddress
	hyperionParams.BridgeContractStartHeight = msg.BridgeContractStartHeight
	hyperionParams.ContractSourceHash = msg.ContractSourceHash

	k.Keeper.setLastObservedEventNonce(ctx, hyperionParams.HyperionId, 0)
	k.Keeper.SetLastObservedEthereumBlockHeight(ctx, hyperionParams.HyperionId, msg.BridgeContractStartHeight-1, uint64(ctx.BlockHeight()))
	k.Keeper.SetID(ctx, types.GetLastOutgoingBatchIDKey(hyperionParams.HyperionId), 0)
	hyperionParams.OffsetValsetNonce = uint64(0)

	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgUpdateChainSmartContractResponse{}, nil
}

func (k msgServer) UpdateChainLogo(c context.Context, msg *types.MsgUpdateChainLogo) (*types.MsgUpdateChainLogoResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.BridgeChainLogo = msg.Logo
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgUpdateChainLogoResponse{}, nil
}

func (k msgServer) UpdateChainName(c context.Context, msg *types.MsgUpdateChainName) (*types.MsgUpdateChainNameResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.BridgeChainName = msg.Name
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgUpdateChainNameResponse{}, nil
}

func (k msgServer) DeleteChain(c context.Context, msg *types.MsgDeleteChain) (*types.MsgDeleteChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}
	params := k.Keeper.GetParams(ctx)

	// Clean all datas for the chain
	k.Keeper.CleanValsetConfirms(ctx, hyperionParams.HyperionId)
	k.Keeper.CleanValsets(ctx, hyperionParams.HyperionId)
	k.Keeper.DeleteBatchs(ctx, hyperionParams.HyperionId)
	k.Keeper.CleanBatchConfirms(ctx, hyperionParams.HyperionId)
	k.Keeper.CleanPoolTransactions(ctx, hyperionParams.HyperionId)
	k.Keeper.CleanAttestations(ctx, hyperionParams.HyperionId)

	// Set the last observed event nonce to 0
	k.Keeper.setLastObservedEventNonce(ctx, hyperionParams.HyperionId, 0)
	k.Keeper.SetLastObservedEthereumBlockHeight(ctx, hyperionParams.HyperionId, 0, 0)
	k.Keeper.SetLastOutgoingBatchID(ctx, hyperionParams.HyperionId, 0)
	k.Keeper.SetLastOutgoingPoolID(ctx, hyperionParams.HyperionId, 0)
	k.Keeper.SetLastObservedValset(ctx, hyperionParams.HyperionId, types.Valset{
		HyperionId: hyperionParams.HyperionId,
	})

	for i, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			params.CounterpartyChainParams = append(params.CounterpartyChainParams[:i], params.CounterpartyChainParams[i+1:]...)
			break
		}
	}

	k.Keeper.SetParams(ctx, params)

	return &types.MsgDeleteChainResponse{}, nil
}

func (k msgServer) PauseChain(c context.Context, msg *types.MsgPauseChain) (*types.MsgPauseChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	params := k.Keeper.GetParams(ctx)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(counterpartyChainParam.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
				continue
			}
			counterpartyChainParam.Paused = true
			break
		}
	}
	k.Keeper.SetParams(ctx, params)
	return &types.MsgPauseChainResponse{}, nil
}

func (k msgServer) UnpauseChain(c context.Context, msg *types.MsgUnpauseChain) (*types.MsgUnpauseChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	params := k.Keeper.GetParams(ctx)
	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(counterpartyChainParam.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
				continue
			}
			counterpartyChainParam.Paused = false
			break
		}
	}
	k.Keeper.SetParams(ctx, params)
	return &types.MsgUnpauseChainResponse{}, nil
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
			if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(counterpartyChainParam.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
				continue
			}
			k.Keeper.setLastObservedEventNonce(ctx, counterpartyChainParam.HyperionId, 0)
			break
		}
	}

	return &types.MsgClearValsetResponse{}, nil
}

func (k msgServer) ForceSetValsetAndLastObservedEventNonce(c context.Context, msg *types.MsgForceSetValsetAndLastObservedEventNonce) (*types.MsgForceSetValsetAndLastObservedEventNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	hyperionParams := k.Keeper.GetCounterpartyChainParams(ctx)[msg.HyperionId]

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
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

	hyperionParams.OffsetValsetNonce = msg.Valset.Nonce
	k.Keeper.SetCounterpartyChainParams(ctx, msg.HyperionId, hyperionParams)

	return &types.MsgForceSetValsetAndLastObservedEventNonceResponse{}, nil
}

func (k msgServer) AddRpc(c context.Context, msg *types.MsgAddRpc) (*types.MsgAddRpcResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	lastObservedEthereumBlockHeight := k.Keeper.GetLastObservedEthereumBlockHeight(ctx, hyperionParams.HyperionId)

	if lastObservedEthereumBlockHeight.EthereumBlockHeight == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "LastObservedBlockHeight not found")
	}

	k.Keeper.UpdateRpcUsed(ctx, hyperionParams.HyperionId, msg.RpcUrl, lastObservedEthereumBlockHeight.EthereumBlockHeight)

	return &types.MsgAddRpcResponse{}, nil
}

func (k msgServer) RemoveRpc(c context.Context, msg *types.MsgRemoveRpc) (*types.MsgRemoveRpcResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	params := k.Keeper.GetParams(ctx)

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.ChainId {
			if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(counterpartyChainParam.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
				continue
			}
			counterpartyChainParam.Rpcs = types.RemoveRpcFromSlice(counterpartyChainParam.Rpcs, msg.RpcUrl)
			break
		}
	}

	k.Keeper.SetParams(ctx, params)

	return &types.MsgRemoveRpcResponse{}, nil
}

func (k msgServer) SetTokenToChain(c context.Context, msg *types.MsgSetTokenToChain) (*types.MsgSetTokenToChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	k.Keeper.CreateOrLinkTokenToChain(ctx, hyperionParams.BridgeChainId, hyperionParams.BridgeChainName, &types.TokenAddressToDenomWithGenesisInfos{
		TokenAddressToDenom: msg.Token,
		DefaultHolders:      make([]*types.HolderWithAmount, 0),
		Logo:                "",
	})

	return &types.MsgSetTokenToChainResponse{}, nil
}

func (k msgServer) RemoveTokenFromChain(c context.Context, msg *types.MsgRemoveTokenFromChain) (*types.MsgRemoveTokenFromChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	k.Keeper.RemoveTokenFromChainMetadata(ctx, hyperionParams.HyperionId, msg.Token)

	return &types.MsgRemoveTokenFromChainResponse{}, nil
}

func (k msgServer) MintToken(c context.Context, msg *types.MsgMintToken) (*types.MsgMintTokenResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	err := k.Keeper.MintToken(ctx, hyperionParams.HyperionId, common.HexToAddress(msg.TokenAddress), msg.Amount, common.HexToAddress(msg.ReceiverAddress))

	if err != nil {
		return nil, errors.Wrap(err, "MintToken failed")
	}

	return &types.MsgMintTokenResponse{}, nil
}

func (k msgServer) BurnToken(c context.Context, msg *types.MsgBurnToken) (*types.MsgBurnTokenResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	err := k.Keeper.BurnToken(ctx, hyperionParams.HyperionId, common.HexToAddress(msg.TokenAddress), msg.Amount, cmn.AnyToHexAddress(msg.Signer))

	if err != nil {
		return nil, errors.Wrap(err, "BurnToken failed")
	}

	return &types.MsgBurnTokenResponse{}, nil
}

func (k msgServer) SetValsetNonce(c context.Context, msg *types.MsgSetValsetNonce) (*types.MsgSetValsetNonceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.OffsetValsetNonce = msg.ValsetNonce
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgSetValsetNonceResponse{}, nil
}

func (k msgServer) SetMinCallExternalDataGas(c context.Context, msg *types.MsgSetMinCallExternalDataGas) (*types.MsgSetMinCallExternalDataGasResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.MinCallExternalDataGas = msg.MinCallExternalDataGas
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgSetMinCallExternalDataGasResponse{}, nil
}

func (k msgServer) SetValsetReward(c context.Context, msg *types.MsgSetValsetReward) (*types.MsgSetValsetRewardResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	tokenAddress := common.HexToAddress(msg.TokenAddress)

	tokenDenom, err := k.Keeper.erc20Keeper.GetTokenDenom(ctx, tokenAddress)
	if err != nil {
		return nil, errors.Wrap(err, "Token not found in the ERC20Keeper")
	}

	_, exists := k.Keeper.GetTokenFromDenom(ctx, hyperionParams.HyperionId, tokenDenom)
	if !exists { // force the reward to be zero
		return nil, errors.Wrap(types.ErrInvalid, "Token Denom not found in the Hyperion associated tokens list")
	}

	hyperionParams.ValsetReward = sdk.Coin{Denom: tokenDenom, Amount: msg.Amount}
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgSetValsetRewardResponse{}, nil
}

func (k msgServer) SetUnbondSlashingValsetsWindow(c context.Context, msg *types.MsgSetUnbondSlashingValsetsWindow) (*types.MsgSetUnbondSlashingValsetsWindowResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.UnbondSlashingValsetsWindow = msg.UnbondSlashingValsetsWindow
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgSetUnbondSlashingValsetsWindowResponse{}, nil
}

func (k msgServer) UpdateDefaultToken(c context.Context, msg *types.MsgUpdateDefaultToken) (*types.MsgUpdateDefaultTokenResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	for _, token := range hyperionParams.DefaultTokens {
		if token.TokenAddressToDenom.TokenAddress == msg.TokenAddress {
			token.TokenAddressToDenom.IsConcensusToken = msg.IsConcensusToken
			token.TokenAddressToDenom.Decimals = msg.Decimals
			token.TokenAddressToDenom.Symbol = msg.Symbol
			token.TokenAddressToDenom.ChainId = strconv.FormatUint(msg.ChainId, 10)
			token.TokenAddressToDenom.IsCosmosOriginated = msg.IsCosmosOriginated
			token.TokenAddressToDenom.Denom = msg.Denom
			break
		}
	}
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgUpdateDefaultTokenResponse{}, nil
}

func (k msgServer) UpdateOutTxTimeout(c context.Context, msg *types.MsgUpdateOutTxTimeout) (*types.MsgUpdateOutTxTimeoutResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.TargetBatchTimeout = msg.TargetBatchTimeout
	hyperionParams.TargetOutgoingTxTimeout = msg.TargetOutgoingTxTimeout
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	return &types.MsgUpdateOutTxTimeoutResponse{}, nil
}

func (k msgServer) CancelAllPendingOutgoingTxs(c context.Context, msg *types.MsgCancelAllPendingOutgoingTxs) (*types.MsgCancelAllPendingOutgoingTxsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	batches := k.Keeper.GetOutgoingTxBatches(ctx, hyperionParams.HyperionId)

	for _, batch := range batches {
		err := k.Keeper.CancelOutgoingTXBatch(ctx, common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId)
		if err != nil {
			ctx.Logger().Error("failed to cancel outgoing tx batch", "error", err, "block", batch.Block, "batch_nonce", batch.BatchNonce)
		}
	}

	txs := k.Keeper.GetPoolTransactions(ctx, hyperionParams.HyperionId)

	for _, tx := range txs {
		alreadyInBatch := false
		batches := k.Keeper.GetOutgoingTxBatches(ctx, hyperionParams.HyperionId)
		for _, batch := range batches {
			for _, batchTx := range batch.Transactions {
				if batchTx.Id == tx.Id {
					alreadyInBatch = true
					break
				}
			}
		}

		if !alreadyInBatch { // we can process cancel
			sender, _ := sdk.AccAddressFromBech32(tx.Sender)
			err := k.Keeper.RemoveFromOutgoingPoolAndRefund(ctx, hyperionParams.HyperionId, tx.Id, sender)
			if err != nil {
				ctx.Logger().Error("failed to cancel outgoing tx", "error", err, "txId", tx.Id, "sender", tx.Sender)
			}
		}
	}
	return &types.MsgCancelAllPendingOutgoingTxsResponse{}, nil
}

func (k msgServer) CancelPendingOutgoingTxs(c context.Context, msg *types.MsgCancelPendingOutgoingTxs) (*types.MsgCancelPendingOutgoingTxsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	txs := k.Keeper.GetPoolTransactions(ctx, hyperionParams.HyperionId)

	count := 0
	for _, tx := range txs {
		if count >= int(msg.Count) {
			break
		}
		sender, _ := sdk.AccAddressFromBech32(tx.Sender)
		err := k.Keeper.RemoveFromOutgoingPoolAndRefund(ctx, hyperionParams.HyperionId, tx.Id, sender)
		if err != nil {
			ctx.Logger().Error("failed to cancel outgoing tx", "error", err, "txId", tx.Id, "sender", tx.Sender)
		}
		count++
	}
	return &types.MsgCancelPendingOutgoingTxsResponse{}, nil
}

func (k msgServer) UpdateChainTokenLogo(c context.Context, msg *types.MsgUpdateChainTokenLogo) (*types.MsgUpdateChainTokenLogoResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	err := k.Keeper.UpdateChainTokenLogo(ctx, msg.ChainId, common.HexToAddress(msg.TokenAddress), msg.Logo)
	if err != nil {
		return nil, errors.Wrap(err, "UpdateChainTokenLogo failed")
	}

	return &types.MsgUpdateChainTokenLogoResponse{}, nil
}

func (k msgServer) UpdateAverageBlockTime(c context.Context, msg *types.MsgUpdateAverageBlockTime) (*types.MsgUpdateAverageBlockTimeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	if msg.ChainId == 0 {
		return nil, errors.Wrap(types.ErrInvalid, "ChainId cannot be 0")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalid, "HyperionParams not found")
	}

	if k.Keeper.authority != msg.Signer && cmn.AnyToHexAddress(hyperionParams.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
		return nil, errors.Wrap(types.ErrInvalid, "not the initializer")
	}

	hyperionParams.AverageBlockTime = msg.AverageBlockTime
	k.Keeper.SetCounterpartyChainParams(ctx, msg.ChainId, hyperionParams)

	_, err := k.CancelAllPendingOutgoingTxs(ctx, &types.MsgCancelAllPendingOutgoingTxs{
		ChainId: hyperionParams.BridgeChainId,
		Signer:  msg.Signer,
	})
	if err != nil {
		return nil, errors.Wrap(err, "CancelAllPendingOutgoingTxs failed")
	}

	return &types.MsgUpdateAverageBlockTimeResponse{}, nil
}
