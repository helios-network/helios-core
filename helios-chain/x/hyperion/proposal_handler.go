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
	// Validate the proposal
	if err := proposal.ValidateBasic(); err != nil {
		return err
	}

	var msg sdk.Msg

	if err := k.Cdc().UnmarshalInterfaceJSON([]byte(proposal.Msg), &msg); err != nil {
		return err
	}

	switch msg := msg.(type) {
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
	case *types.MsgUpdateParams:
		// msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).UpdateParams(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgBlacklistEthereumAddresses:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).BlacklistEthereumAddresses(ctx, msg)
		if err != nil {
			return err
		}
	case *types.MsgRevokeEthereumBlacklist:
		msg.Signer = k.GetAuthority()
		_, err := keeper.NewMsgServerImpl(k).RevokeEthereumBlacklist(ctx, msg)
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

	default:
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized hyperion proposal message type: %T", msg)
	}
	return nil
}
