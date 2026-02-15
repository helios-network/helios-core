package hyperion

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/hyperion/keeper"
	"helios-core/helios-chain/x/hyperion/types"
)

// NewHyperionProposalHandler creates a governance handler to manage all hyperion proposal types.
func NewHyperionProposalHandler(k keeper.Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.HyperionProposal:
			return HandleHyperionProposal(ctx, k, c)
		default:
			return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized hyperion proposal content type: %T", c)
		}
	}
}

func HandleHyperionProposal(ctx sdk.Context, k keeper.Keeper, proposal *types.HyperionProposal) error {
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

	var msg sdk.Msg

	if err := k.Cdc().UnmarshalInterfaceJSON([]byte(proposal.Msg), &msg); err != nil {
		return err
	}

	switch msg := msg.(type) {
	case *types.MsgUpdateParams:
		msg.Authority = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateParams(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgAddCounterpartyChainParams:
		msg.Authority = cmn.AnyToHexAddress(k.GetAuthority()).Hex()
		_, err := keeper.NewMsgServerImpl(k).AddCounterpartyChainParams(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateChainSmartContract:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateChainSmartContract(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateChainLogo:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateChainLogo(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateChainName:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateChainName(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgDeleteChain:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).DeleteChain(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgChangeInitializer:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).ChangeInitializer(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgBlacklistAddresses:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).BlacklistAddresses(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgRevokeBlacklist:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).RevokeBlacklist(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgAddRpc:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).AddRpc(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgRemoveRpc:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).RemoveRpc(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgPauseChain:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).PauseChain(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUnpauseChain:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UnpauseChain(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetUnbondSlashingValsetsWindow:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetUnbondSlashingValsetsWindow(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetValsetReward:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetValsetReward(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetMinCallExternalDataGas:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetMinCallExternalDataGas(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetValsetNonce:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetValsetNonce(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgBurnToken:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).BurnToken(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgMintToken:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).MintToken(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetTokenToChain:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetTokenToChain(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgRemoveTokenFromChain:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).RemoveTokenFromChain(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgForceSetValsetAndLastObservedEventNonce:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).ForceSetValsetAndLastObservedEventNonce(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgClearValset:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).ClearValset(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateDefaultToken:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateDefaultToken(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateOutTxTimeout:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateOutTxTimeout(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgCancelAllPendingOutgoingTxs:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).CancelAllPendingOutgoingTxs(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateChainTokenLogo:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateChainTokenLogo(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateAverageBlockTime:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateAverageBlockTime(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgUpdateAverageCounterpartyBlockTime:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateAverageCounterpartyBlockTime(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetLastBatchNonce:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetLastBatchNonce(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgSetWhitelistedAddresses:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).SetWhitelistedAddresses(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgAddOneWhitelistedAddress:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).AddOneWhitelistedAddress(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgRemoveOneWhitelistedAddress:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).RemoveOneWhitelistedAddress(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgCleanSkippedTxs:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).CleanSkippedTxs(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgCleanAllSkippedTxs:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).CleanAllSkippedTxs(ctx, msg)
		if err != nil {
			return err
		}
	default:
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized hyperion proposal message type: %T", msg)
	}
	return nil
}
