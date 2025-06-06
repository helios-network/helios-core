package types

import (
	"encoding/hex"
	"fmt"

	"cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto/tmhash"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

var (
	_ sdk.Msg = &MsgValsetConfirm{}
	_ sdk.Msg = &MsgSendToChain{}
	_ sdk.Msg = &MsgRequestBatch{}
	_ sdk.Msg = &MsgConfirmBatch{}
	_ sdk.Msg = &MsgSetOrchestratorAddresses{}
	_ sdk.Msg = &MsgERC20DeployedClaim{}
	_ sdk.Msg = &MsgDepositClaim{}
	_ sdk.Msg = &MsgWithdrawClaim{}
	_ sdk.Msg = &MsgExternalDataClaim{}
	_ sdk.Msg = &MsgCancelSendToChain{}
	_ sdk.Msg = &MsgValsetUpdatedClaim{}
	_ sdk.Msg = &MsgSubmitBadSignatureEvidence{}
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgBlacklistAddresses{}
	_ sdk.Msg = &MsgRevokeBlacklist{}
)

func (m *MsgUpdateParams) Route() string { return RouterKey }

func (m *MsgUpdateParams) Type() string { return "update_params" }

func (m *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Authority); err != nil {
		return errors.Wrap(err, "invalid authority address")
	}

	if err := m.Params.ValidateBasic(); err != nil {
		return errors.Wrap(err, "invalid params")
	}

	return nil
}

func (m *MsgUpdateParams) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(m))
}

func (m *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Authority)
	return []sdk.AccAddress{addr}
}

// NewMsgSetOrchestratorAddress returns a new MsgSetOrchestratorAddresses
func NewMsgSetOrchestratorAddress(sender, orchestrator sdk.AccAddress, ethAddr common.Address) *MsgSetOrchestratorAddresses {
	return &MsgSetOrchestratorAddresses{
		Sender:       sender.String(),
		Orchestrator: orchestrator.String(),
		EthAddress:   ethAddr.Hex(),
	}
}

// Route should return the name of the module
func (msg *MsgSetOrchestratorAddresses) Route() string { return RouterKey }

// Type should return the action
func (msg *MsgSetOrchestratorAddresses) Type() string { return "set_operator_address" }

// ValidateBasic performs stateless checks
func (msg *MsgSetOrchestratorAddresses) ValidateBasic() (err error) {
	if _, err = sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender)
	}
	if _, err = sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	if err := ValidateEthAddress(msg.EthAddress); err != nil {
		return errors.Wrap(err, "ethereum address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgSetOrchestratorAddresses) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg *MsgSetOrchestratorAddresses) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// NewMsgValsetConfirm returns a new msgValsetConfirm
func NewMsgValsetConfirm(nonce uint64, ethAddress string, validator sdk.AccAddress, signature string) *MsgValsetConfirm {
	return &MsgValsetConfirm{
		Nonce:        nonce,
		Orchestrator: validator.String(),
		EthAddress:   ethAddress,
		Signature:    signature,
	}
}

// Route should return the name of the module
func (msg *MsgValsetConfirm) Route() string { return RouterKey }

// Type should return the action
func (msg *MsgValsetConfirm) Type() string { return "valset_confirm" }

// ValidateBasic performs stateless checks
func (msg *MsgValsetConfirm) ValidateBasic() (err error) {
	if _, err = sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	if err := ValidateEthAddress(msg.EthAddress); err != nil {
		return errors.Wrap(err, "ethereum address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgValsetConfirm) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg *MsgValsetConfirm) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// NewMsgSendToChain returns a new msgSendToChain
func NewMsgSendToChain(sender sdk.AccAddress, destAddress string, send, bridgeFee sdk.Coin) *MsgSendToChain {
	return &MsgSendToChain{
		Sender:    sender.String(),
		Dest:      destAddress,
		Amount:    send,
		BridgeFee: bridgeFee,
	}
}

// Route should return the name of the module
func (msg MsgSendToChain) Route() string { return RouterKey }

// Type should return the action
func (msg MsgSendToChain) Type() string { return "send_to_chain" }

// ValidateBasic runs stateless checks on the message
// Checks if the Eth address is valid
func (msg MsgSendToChain) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender)
	}

	// fee and send must be of the same denom
	if msg.Amount.Denom != msg.BridgeFee.Denom {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, fmt.Sprintf("fee and amount must be the same type %s != %s", msg.Amount.Denom, msg.BridgeFee.Denom))
	}

	if !msg.Amount.IsValid() || msg.Amount.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "amount")
	}
	if !msg.BridgeFee.IsValid() || msg.BridgeFee.IsZero() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, "fee")
	}
	if err := ValidateEthAddress(msg.Dest); err != nil {
		return errors.Wrap(err, "ethereum address")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSendToChain) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSendToChain) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// NewMsgRequestBatch returns a new msgRequestBatch
func NewMsgRequestBatch(orchestrator sdk.AccAddress) *MsgRequestBatch {
	return &MsgRequestBatch{
		Orchestrator: orchestrator.String(),
	}
}

// Route should return the name of the module
func (msg MsgRequestBatch) Route() string { return RouterKey }

// Type should return the action
func (msg MsgRequestBatch) Type() string { return "request_batch" }

// ValidateBasic performs stateless checks
func (msg MsgRequestBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgRequestBatch) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgRequestBatch) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// Route should return the name of the module
func (msg MsgConfirmBatch) Route() string { return RouterKey }

// Type should return the action
func (msg MsgConfirmBatch) Type() string { return "confirm_batch" }

// ValidateBasic performs stateless checks
func (msg MsgConfirmBatch) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	if err := ValidateEthAddress(msg.EthSigner); err != nil {
		return errors.Wrap(err, "eth signer")
	}
	if err := ValidateEthAddress(msg.TokenContract); err != nil {
		return errors.Wrap(err, "token contract")
	}
	_, err := hex.DecodeString(msg.Signature)
	if err != nil {
		return errors.Wrapf(sdkerrors.ErrUnknownRequest, "Could not decode hex string %s", msg.Signature)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgConfirmBatch) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgConfirmBatch) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// EthereumClaim represents a claim on ethereum state
type EthereumClaim interface {
	// The hyperion id of the counterparty chain that the claimed event occurred on.
	GetHyperionId() uint64
	// All Ethereum claims that we relay from the Hyperion contract and into the module
	// have a nonce that is monotonically increasing and unique, since this nonce is
	// issued by the Ethereum contract it is immutable and must be agreed on by all validators
	// any disagreement on what claim goes to what nonce means someone is lying.
	GetEventNonce() uint64
	// The block height that the claimed event occurred on. This EventNonce provides sufficient
	// ordering for the execution of all claims. The block height is used only for batchTimeouts
	// when we go to create a new batch we set the timeout some number of batches out from the last
	// known height plus projected block progress since then.
	GetBlockHeight() uint64
	// the delegate address of the claimer, for MsgDepositClaim and MsgWithdrawClaim
	// this is sent in as the sdk.AccAddress of the delegated key. it is up to the user
	// to disambiguate this into a sdk.ValAddress
	GetClaimer() sdk.AccAddress
	// Which type of claim this is
	GetType() ClaimType
	ValidateBasic() error
	// The claim hash of this claim. This is used to store these claims and also used to check if two different
	// validators claims agree. Therefore it's extremely important that this include all elements of the claim
	// with the exception of the orchestrator who sent it in, which will be used as a different part of the index
	ClaimHash() []byte

	GetRpcUsed() string
}

var (
	_ EthereumClaim = &MsgDepositClaim{}
	_ EthereumClaim = &MsgWithdrawClaim{}
	_ EthereumClaim = &MsgERC20DeployedClaim{}
)

// GetType returns the claim type
func (msg *MsgExternalDataClaim) GetType() ClaimType {
	return CLAIM_TYPE_WITHDRAW
}

// ValidateBasic performs stateless checks
func (msg *MsgExternalDataClaim) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	if err := ValidateEthAddress(msg.ExternalContractAddress); err != nil {
		return errors.Wrap(err, "external contract address")
	}
	if msg.TxNonce == 0 {
		return fmt.Errorf("nonce == 0")
	}
	return nil
}

// Hash implements WithdrawBatch.Hash
func (msg *MsgExternalDataClaim) ClaimHash() []byte {
	path := fmt.Sprintf("%s/%d/%d/%d/%s/%s/%s", msg.ExternalContractAddress, msg.BlockHeight, msg.TxNonce, msg.HyperionId, msg.Orchestrator, msg.CallDataResult, msg.CallDataResultError)
	return tmhash.Sum([]byte(path))
}

// GetSignBytes encodes the message for signing
func (msg MsgExternalDataClaim) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

func (msg MsgExternalDataClaim) GetClaimer() sdk.AccAddress {
	err := msg.ValidateBasic()
	if err != nil {
		panic("MsgExternalDataClaim failed ValidateBasic! Should have been handled earlier")
	}
	val, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	return val
}

// GetSigners defines whose signature is required
func (msg MsgExternalDataClaim) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// Route should return the name of the module
func (msg MsgExternalDataClaim) Route() string { return RouterKey }

// Type should return the action
func (msg MsgExternalDataClaim) Type() string { return "external_data_claim" }

// GetType returns the type of the claim
func (msg *MsgDepositClaim) GetType() ClaimType {
	return CLAIM_TYPE_DEPOSIT
}

// ValidateBasic performs stateless checks
func (msg *MsgDepositClaim) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.CosmosReceiver); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.CosmosReceiver)
	}
	if err := ValidateEthAddress(msg.EthereumSender); err != nil {
		return errors.Wrap(err, "eth sender")
	}
	if err := ValidateEthAddress(msg.TokenContract); err != nil {
		return errors.Wrap(err, "erc20 token")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	if msg.EventNonce == 0 {
		return fmt.Errorf("nonce == 0")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgDepositClaim) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

func (msg MsgDepositClaim) GetClaimer() sdk.AccAddress {
	err := msg.ValidateBasic()
	if err != nil {
		panic("MsgDepositClaim failed ValidateBasic! Should have been handled earlier")
	}

	val, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	return val
}

// GetSigners defines whose signature is required
func (msg MsgDepositClaim) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// Type should return the action
func (msg MsgDepositClaim) Type() string { return "deposit_claim" }

// Route should return the name of the module
func (msg MsgDepositClaim) Route() string { return RouterKey }

const (
	TypeMsgWithdrawClaim = "withdraw_claim"
)

// Hash implements BridgeDeposit.Hash
// modify this with care as it is security sensitive. If an element of the claim is not in this hash a single hostile validator
// could engineer a hash collision and execute a version of the claim with any unhashed data changed to benefit them.
// note that the Orchestrator is the only field excluded from this hash, this is because that value is used higher up in the store
// structure for who has made what claim and is verified by the msg ante-handler for signatures
func (msg *MsgDepositClaim) ClaimHash() []byte {
	path := fmt.Sprintf("%d/%d/%s/%s/%s/%s/%s", msg.EventNonce, msg.BlockHeight, msg.TokenContract, msg.Amount.String(), msg.EthereumSender, msg.CosmosReceiver, msg.Data)
	return tmhash.Sum([]byte(path))
}

// GetType returns the claim type
func (msg *MsgWithdrawClaim) GetType() ClaimType {
	return CLAIM_TYPE_WITHDRAW
}

// ValidateBasic performs stateless checks
func (msg *MsgWithdrawClaim) ValidateBasic() error {
	if msg.EventNonce == 0 {
		return fmt.Errorf("event_nonce == 0")
	}
	if msg.BatchNonce == 0 {
		return fmt.Errorf("batch_nonce == 0")
	}
	if err := ValidateEthAddress(msg.TokenContract); err != nil {
		return errors.Wrap(err, "erc20 token")
	}
	if _, err := sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	return nil
}

// Hash implements WithdrawBatch.Hash
func (msg *MsgWithdrawClaim) ClaimHash() []byte {
	path := fmt.Sprintf("%s/%d/%d/%s", msg.TokenContract, msg.BatchNonce, msg.EventNonce, msg.TokenContract)
	return tmhash.Sum([]byte(path))
}

// GetSignBytes encodes the message for signing
func (msg MsgWithdrawClaim) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

func (msg MsgWithdrawClaim) GetClaimer() sdk.AccAddress {
	err := msg.ValidateBasic()
	if err != nil {
		panic("MsgWithdrawClaim failed ValidateBasic! Should have been handled earlier")
	}
	val, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	return val
}

// GetSigners defines whose signature is required
func (msg MsgWithdrawClaim) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// Route should return the name of the module
func (msg MsgWithdrawClaim) Route() string { return RouterKey }

// Type should return the action
func (msg MsgWithdrawClaim) Type() string { return "withdraw_claim" }

const (
	TypeMsgDepositClaim = "deposit_claim"
)

// EthereumClaim implementation for MsgERC20DeployedClaim
// ======================================================

// GetType returns the type of the claim
func (e *MsgERC20DeployedClaim) GetType() ClaimType {
	return CLAIM_TYPE_ERC20_DEPLOYED
}

// ValidateBasic performs stateless checks
func (e *MsgERC20DeployedClaim) ValidateBasic() error {
	if err := ValidateEthAddress(e.TokenContract); err != nil {
		return errors.Wrap(err, "erc20 token")
	}
	if _, err := sdk.AccAddressFromBech32(e.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, e.Orchestrator)
	}
	if e.EventNonce == 0 {
		return fmt.Errorf("nonce == 0")
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgERC20DeployedClaim) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

func (msg MsgERC20DeployedClaim) GetClaimer() sdk.AccAddress {
	err := msg.ValidateBasic()
	if err != nil {
		panic("MsgERC20DeployedClaim failed ValidateBasic! Should have been handled earlier")
	}

	val, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	return val
}

// GetSigners defines whose signature is required
func (msg MsgERC20DeployedClaim) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// Type should return the action
func (msg MsgERC20DeployedClaim) Type() string { return "ERC20_deployed_claim" }

// Route should return the name of the module
func (msg MsgERC20DeployedClaim) Route() string { return RouterKey }

// Hash implements BridgeDeposit.Hash
// modify this with care as it is security sensitive. If an element of the claim is not in this hash a single hostile validator
// could engineer a hash collision and execute a version of the claim with any unhashed data changed to benefit them.
// note that the Orchestrator is the only field excluded from this hash, this is because that value is used higher up in the store
// structure for who has made what claim and is verified by the msg ante-handler for signatures
func (b *MsgERC20DeployedClaim) ClaimHash() []byte {
	path := fmt.Sprintf("%d/%d/%s/%s/%s/%s/%d", b.EventNonce, b.BlockHeight, b.CosmosDenom, b.TokenContract, b.Name, b.Symbol, b.Decimals)
	return tmhash.Sum([]byte(path))
}

// GetType returns the type of the claim
func (e *MsgValsetUpdatedClaim) GetType() ClaimType {
	return CLAIM_TYPE_VALSET_UPDATED
}

// ValidateBasic performs stateless checks
func (e *MsgValsetUpdatedClaim) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(e.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, e.Orchestrator)
	}
	if e.EventNonce == 0 {
		return fmt.Errorf("nonce == 0")
	}
	err := ValidateEthAddress(e.RewardToken)
	if err != nil {
		return err
	}

	for _, member := range e.Members {
		err := ValidateEthAddress(member.EthereumAddress)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgValsetUpdatedClaim) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

func (msg MsgValsetUpdatedClaim) GetClaimer() sdk.AccAddress {
	err := msg.ValidateBasic()
	if err != nil {
		panic("MsgERC20DeployedClaim failed ValidateBasic! Should have been handled earlier")
	}

	val, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	return val
}

// GetSigners defines whose signature is required
func (msg MsgValsetUpdatedClaim) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}

// Type should return the action
func (msg MsgValsetUpdatedClaim) Type() string { return "Valset_Updated_Claim" }

// Route should return the name of the module
func (msg MsgValsetUpdatedClaim) Route() string { return RouterKey }

// Hash implements BridgeDeposit.Hash
func (b *MsgValsetUpdatedClaim) ClaimHash() []byte {
	path := fmt.Sprintf("%d/%d/%d/%s/", b.ValsetNonce, b.EventNonce, b.BlockHeight, b.Members)
	return tmhash.Sum([]byte(path))
}

// NewMsgCancelSendToChain returns a new msgMsgCancelSendToChain
func NewMsgCancelSendToChain(sender sdk.AccAddress, id uint64) *MsgCancelSendToChain {
	return &MsgCancelSendToChain{
		TransactionId: id,
		Sender:        sender.String(),
	}
}

// Route should return the name of the module
func (msg *MsgCancelSendToChain) Route() string { return RouterKey }

// Type should return the action
func (msg *MsgCancelSendToChain) Type() string { return "cancel_send_to_chain" }

// ValidateBasic performs stateless checks
func (msg *MsgCancelSendToChain) ValidateBasic() (err error) {
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender)
	}
	return nil
}

// GetSignBytes encodes the message for signing
func (msg *MsgCancelSendToChain) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg *MsgCancelSendToChain) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// MsgSubmitBadSignatureEvidence
// ======================================================

// ValidateBasic performs stateless checks
func (e *MsgSubmitBadSignatureEvidence) ValidateBasic() error {
	return nil
}

// GetSignBytes encodes the message for signing
func (msg MsgSubmitBadSignatureEvidence) GetSignBytes() []byte {
	return sdk.MustSortJSON(amino.MustMarshalJSON(msg))
}

// GetSigners defines whose signature is required
func (msg MsgSubmitBadSignatureEvidence) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// Type should return the action
func (msg MsgSubmitBadSignatureEvidence) Type() string { return "Submit_Bad_Signature_Evidence" }

// Route should return the name of the module
func (msg MsgSubmitBadSignatureEvidence) Route() string { return RouterKey }

func (m *MsgBlacklistAddresses) Route() string { return RouterKey }

func (m *MsgBlacklistAddresses) Type() string { return "blacklist_addresses" }

func (m *MsgBlacklistAddresses) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errors.Wrap(err, "invalid signer address")
	}

	for _, blacklistAddress := range m.BlacklistAddresses {

		_, err := NewEthAddress(blacklistAddress)
		if err != nil {
			return errors.Wrapf(err, "invalid blacklist address %s", blacklistAddress)
		}
	}

	return nil
}

func (m *MsgBlacklistAddresses) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(m))
}

func (m *MsgBlacklistAddresses) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Signer)
	return []sdk.AccAddress{addr}
}

func (m *MsgRevokeBlacklist) Route() string { return RouterKey }

func (m *MsgRevokeBlacklist) Type() string { return "revoke_blacklist" }

func (m *MsgRevokeBlacklist) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Signer); err != nil {
		return errors.Wrap(err, "invalid signer address")
	}

	for _, blacklistAddress := range m.BlacklistAddresses {

		_, err := NewEthAddress(blacklistAddress)
		if err != nil {
			return errors.Wrapf(err, "invalid blacklist address %s", blacklistAddress)
		}
	}

	return nil
}

func (m *MsgRevokeBlacklist) GetSignBytes() []byte {
	return sdk.MustSortJSON(ModuleCdc.MustMarshal(m))
}

func (m *MsgRevokeBlacklist) GetSigners() []sdk.AccAddress {
	addr, _ := sdk.AccAddressFromBech32(m.Signer)
	return []sdk.AccAddress{addr}
}
