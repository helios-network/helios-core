package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"

	authzcdc "github.com/cosmos/cosmos-sdk/x/authz/codec"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

// Legacy Amino Codec for JSON Serialization
var amino = codec.NewLegacyAmino()

// Protobuf Codec for Message Serialization
var ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

// Amino JSON codec for backwards compatibility
var AminoCdc = codec.NewAminoCodec(amino) //nolint:staticcheck

func init() {
	RegisterLegacyAminoCodec(amino)
	RegisterLegacyAminoCodec(authzcdc.Amino)
	amino.Seal()
}

// RegisterInterfaces registers the interfaces for the proto stuff
func RegisterInterfaces(registry types.InterfaceRegistry) {
	// Register Hyperion Messages
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgValsetConfirm{},
		&MsgSendToChain{},
		&MsgRequestBatch{},
		&MsgConfirmBatch{},
		&MsgDepositClaim{},
		&MsgWithdrawClaim{},
		&MsgERC20DeployedClaim{},
		&MsgSetOrchestratorAddresses{},
		&MsgValsetUpdatedClaim{},
		&MsgCancelSendToChain{},
		&MsgSubmitBadSignatureEvidence{},
		&MsgUpdateParams{},
		&MsgBlacklistAddresses{},
		&MsgRevokeBlacklist{},

		&MsgAddCounterpartyChainParams{},
		&MsgUpdateChainSmartContract{},
		&MsgUpdateChainLogo{},
		&MsgUpdateChainName{},
		&MsgDeleteChain{},
		&MsgChangeInitializer{},
		&MsgAddRpc{},
		&MsgRemoveRpc{},

		&MsgClearValset{},
		&MsgForceSetValsetAndLastObservedEventNonce{},
		&MsgSetTokenToChain{},
		&MsgMintToken{},
		&MsgBurnToken{},
		&MsgSetMinCallExternalDataGas{},
		&MsgSetValsetNonce{},
		&MsgSetValsetReward{},
		&MsgSetUnbondSlashingValsetsWindow{},
		&MsgUnpauseChain{},
		&MsgPauseChain{},
		&MsgUpdateDefaultToken{},
		&MsgUpdateOutTxTimeout{},
		&MsgCancelAllPendingOutgoingTxs{},
		&MsgRemoveTokenFromChain{},
		&MsgUpdateChainTokenLogo{},
		&MsgUpdateAverageBlockTime{},
	)

	registry.RegisterInterface(
		"hyperion.v1beta1.EthereumClaim",
		(*EthereumClaim)(nil),
		&MsgDepositClaim{},
		&MsgWithdrawClaim{},
		&MsgERC20DeployedClaim{},
		&MsgValsetUpdatedClaim{},
	)

	registry.RegisterImplementations(
		(*govv1beta1.Content)(nil),
		&HyperionProposal{},
	)

	// Register MsgService Descriptor
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

// RegisterLegacyAminoCodec registers the necessary Hyperion messages for JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSetOrchestratorAddresses{}, "hyperion/MsgSetOrchestratorAddresses", nil)
	cdc.RegisterConcrete(&MsgValsetConfirm{}, "hyperion/MsgValsetConfirm", nil)
	cdc.RegisterConcrete(&MsgSendToChain{}, "hyperion/MsgSendToChain", nil)
	cdc.RegisterConcrete(&MsgCancelSendToChain{}, "hyperion/MsgCancelSendToChain", nil)
	cdc.RegisterConcrete(&MsgRequestBatch{}, "hyperion/MsgRequestBatch", nil)
	cdc.RegisterConcrete(&MsgConfirmBatch{}, "hyperion/MsgConfirmBatch", nil)
	cdc.RegisterConcrete(&Valset{}, "hyperion/Valset", nil)
	cdc.RegisterConcrete(&MsgDepositClaim{}, "hyperion/MsgDepositClaim", nil)
	cdc.RegisterConcrete(&MsgWithdrawClaim{}, "hyperion/MsgWithdrawClaim", nil)
	cdc.RegisterConcrete(&MsgERC20DeployedClaim{}, "hyperion/MsgERC20DeployedClaim", nil)
	cdc.RegisterConcrete(&MsgValsetUpdatedClaim{}, "hyperion/MsgValsetUpdatedClaim", nil)
	cdc.RegisterConcrete(&OutgoingTxBatch{}, "hyperion/OutgoingTxBatch", nil)
	cdc.RegisterConcrete(&OutgoingTransferTx{}, "hyperion/OutgoingTransferTx", nil)
	cdc.RegisterConcrete(&Token{}, "hyperion/ERC20Token", nil)
	cdc.RegisterConcrete(&IDSet{}, "hyperion/IDSet", nil)
	cdc.RegisterConcrete(&Attestation{}, "hyperion/Attestation", nil)
	cdc.RegisterConcrete(&MsgSubmitBadSignatureEvidence{}, "hyperion/MsgSubmitBadSignatureEvidence", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "hyperion/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgBlacklistAddresses{}, "hyperion/MsgBlacklistAddresses", nil)
	cdc.RegisterConcrete(&MsgRevokeBlacklist{}, "hyperion/MsgRevokeBlacklist", nil)

	cdc.RegisterConcrete(&MsgAddCounterpartyChainParams{}, "hyperion/MsgAddCounterpartyChainParams", nil)
	cdc.RegisterConcrete(&MsgUpdateChainSmartContract{}, "hyperion/MsgUpdateChainSmartContract", nil)
	cdc.RegisterConcrete(&MsgUpdateChainLogo{}, "hyperion/MsgUpdateChainLogo", nil)
	cdc.RegisterConcrete(&MsgUpdateChainName{}, "hyperion/MsgUpdateChainName", nil)
	cdc.RegisterConcrete(&MsgDeleteChain{}, "hyperion/MsgDeleteChain", nil)
	cdc.RegisterConcrete(&MsgChangeInitializer{}, "hyperion/MsgChangeInitializer", nil)
	cdc.RegisterConcrete(&MsgAddRpc{}, "hyperion/MsgAddRpc", nil)
	cdc.RegisterConcrete(&MsgRemoveRpc{}, "hyperion/MsgRemoveRpc", nil)

	cdc.RegisterConcrete(&MsgClearValset{}, "hyperion/MsgClearValset", nil)
	cdc.RegisterConcrete(&MsgForceSetValsetAndLastObservedEventNonce{}, "hyperion/MsgForceSetValsetAndLastObservedEventNonce", nil)
	cdc.RegisterConcrete(&MsgSetTokenToChain{}, "hyperion/MsgSetTokenToChain", nil)
	cdc.RegisterConcrete(&MsgMintToken{}, "hyperion/MsgMintToken", nil)
	cdc.RegisterConcrete(&MsgBurnToken{}, "hyperion/MsgBurnToken", nil)
	cdc.RegisterConcrete(&MsgSetMinCallExternalDataGas{}, "hyperion/MsgSetMinCallExternalDataGas", nil)
	cdc.RegisterConcrete(&MsgSetValsetReward{}, "hyperion/MsgSetValsetReward", nil)
	cdc.RegisterConcrete(&MsgSetUnbondSlashingValsetsWindow{}, "hyperion/MsgSetUnbondSlashingValsetsWindow", nil)
	cdc.RegisterConcrete(&MsgUnpauseChain{}, "hyperion/MsgUnpauseChain", nil)
	cdc.RegisterConcrete(&MsgPauseChain{}, "hyperion/MsgPauseChain", nil)
	cdc.RegisterConcrete(&MsgUpdateDefaultToken{}, "hyperion/MsgUpdateDefaultToken", nil)
	cdc.RegisterConcrete(&MsgUpdateOutTxTimeout{}, "hyperion/MsgUpdateOutTxTimeout", nil)
	cdc.RegisterConcrete(&MsgCancelAllPendingOutgoingTxs{}, "hyperion/MsgCancelAllPendingOutgoingTxs", nil)
	cdc.RegisterConcrete(&MsgRemoveTokenFromChain{}, "hyperion/MsgRemoveTokenFromChain", nil)
	cdc.RegisterConcrete(&MsgUpdateChainTokenLogo{}, "hyperion/MsgUpdateChainTokenLogo", nil)
	cdc.RegisterConcrete(&MsgUpdateAverageBlockTime{}, "hyperion/MsgUpdateAverageBlockTime", nil)

	cdc.RegisterConcrete(&Params{}, "hyperion/Params", nil)
}
