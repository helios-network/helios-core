package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/modules/hyperion/types"
)

type msgServer struct {
	Keeper

	svcTags metrics.Tags
}

// NewMsgServerImpl returns an implementation of the gov MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper,

		svcTags: metrics.Tags{
			"svc": "hyperion_h",
		},
	}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) SetOrchestratorAddresses(c context.Context, msg *types.MsgSetOrchestratorAddresses) (*types.MsgSetOrchestratorAddressesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	log.Println("validatorAccountAddr: ", validatorAccountAddr)
	validatorAddr := sdk.ValAddress(validatorAccountAddr.Bytes())
	log.Println("validatorAddr: ", validatorAddr)
	// get orchestrator address if available. otherwise default to validator address.
	var orchestratorAddr sdk.AccAddress
	if msg.Orchestrator != "" {
		orchestratorAddr, _ = sdk.AccAddressFromBech32(msg.Orchestrator)
	} else {
		orchestratorAddr = validatorAccountAddr
	}

	valAddr, foundExistingOrchestratorKey := k.GetOrchestratorValidatorByHyperionID(ctx, orchestratorAddr, msg.HyperionId)
	ethAddress, foundExistingEthAddress := k.GetEthAddressByValidatorByHyperionID(ctx, validatorAddr, msg.HyperionId)
	fmt.Println("valAddr: ", valAddr)
	fmt.Println("orchestratorAddr: ", orchestratorAddr)
	fmt.Println("ethAddress: ", ethAddress)

	// ensure that the validator exists
	if val, err := k.Keeper.StakingKeeper.Validator(ctx, validatorAddr); err != nil || val == nil {
		if err == nil {
			err = stakingtypes.ErrNoValidatorFound
		}
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, validatorAddr.String())
	} else if foundExistingOrchestratorKey || foundExistingEthAddress {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrResetDelegateKeys, validatorAddr.String())
	}

	// set the orchestrator address
	k.SetOrchestratorValidatorByHyperionID(ctx, validatorAddr, orchestratorAddr, msg.HyperionId)
	// set the ethereum address
	fmt.Println("msg.EthAddress: ", msg.EthAddress)
	ethAddr := common.HexToAddress(msg.EthAddress)
	fmt.Println("ethAddr: ", ethAddr)
	k.SetEthAddressForValidator(ctx, validatorAddr, ethAddr, msg.HyperionId)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetOrchestratorAddresses{
		ValidatorAddress:    validatorAddr.String(),
		OrchestratorAddress: orchestratorAddr.String(),
		OperatorEthAddress:  msg.EthAddress,
	})
	fmt.Println("SetOrchestratorAddresses success")

	return &types.MsgSetOrchestratorAddressesResponse{}, nil

}

// ValsetConfirm handles MsgValsetConfirm
func (k msgServer) ValsetConfirm(c context.Context, msg *types.MsgValsetConfirm) (*types.MsgValsetConfirmResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	valset := k.GetValset(ctx, msg.Nonce, msg.HyperionId)
	if valset == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "couldn't find valset")
	}

	checkpoint := valset.GetCheckpoint(msg.HyperionId)

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "signature decoding")
	}
	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidatorByHyperionID(ctx, orchaddr, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress, found := k.GetEthAddressByValidatorByHyperionID(ctx, validator, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrEmpty, "no eth address found")
	}

	if err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress); err != nil {
		description := fmt.Sprintf(
			"signature verification failed expected sig by %s with hyperion-id %d with checkpoint %s found %s",
			ethAddress, msg.HyperionId, checkpoint.Hex(), msg.Signature,
		)

		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, description)
	}

	// persist signature
	if k.GetValsetConfirm(ctx, msg.Nonce, orchaddr, msg.HyperionId) != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "signature duplicate")
	}
	k.SetValsetConfirm(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventValsetConfirm{
		HyperionId:          msg.HyperionId,
		ValsetNonce:         msg.Nonce,
		OrchestratorAddress: orchaddr.String(),
	})

	return &types.MsgValsetConfirmResponse{}, nil
}

// SendToChain handles MsgSendToChain
func (k msgServer) SendToChain(c context.Context, msg *types.MsgSendToChain) (*types.MsgSendToChainResponse, error) {
	fmt.Println("SendToChain for hyperionId: ", msg.DestHyperionId)
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	dest := common.HexToAddress(msg.Dest)
	if k.InvalidSendToChainAddress(ctx, dest) {
		return nil, errors.Wrap(types.ErrInvalidEthDestination, "destination address is invalid or blacklisted")
	}

	txID, err := k.AddToOutgoingPool(ctx, sender, common.HexToAddress(msg.Dest), msg.Amount, msg.BridgeFee, msg.DestHyperionId)
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSendToChain{
		HyperionId:   msg.DestHyperionId,
		OutgoingTxId: txID,
		Sender:       sender.String(),
		Receiver:     msg.Dest,
		Amount:       msg.Amount,
		BridgeFee:    msg.BridgeFee,
	})

	return &types.MsgSendToChainResponse{}, nil
}

// RequestBatch handles MsgRequestBatch
func (k msgServer) RequestBatch(c context.Context, msg *types.MsgRequestBatch) (*types.MsgRequestBatchResponse, error) {
	fmt.Println("RequestBatch, got msg request batch from hyperion - msg: ", msg)
	log.Println("RequestBatch, got msg request batch from hyperion - msg: ")
	log.Println(msg)
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	// Check if the denom is a hyperion coin, if not, check if there is a deployed ERC20 representing it.
	// If not, error out
	_, tokenContract, err := k.DenomToERC20Lookup(ctx, msg.Denom, msg.HyperionId)
	if err != nil {
		fmt.Println("RequestBatch - err: ", err)
		return nil, err
	}

	batch, err := k.BuildOutgoingTXBatch(ctx, tokenContract, msg.HyperionId, OutgoingTxBatchSize)
	if err != nil {
		return nil, err
	}

	batchTxIDs := make([]uint64, 0, len(batch.Transactions))

	for _, outgoingTransferTx := range batch.Transactions {
		batchTxIDs = append(batchTxIDs, outgoingTransferTx.Id)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventOutgoingBatch{
		HyperionId:          msg.HyperionId,
		Denom:               msg.Denom,
		OrchestratorAddress: msg.Orchestrator,
		BatchNonce:          batch.BatchNonce,
		BatchTimeout:        batch.BatchTimeout,
		BatchTxIds:          batchTxIDs,
	})

	return &types.MsgRequestBatchResponse{}, nil
}

// ConfirmBatch handles MsgConfirmBatch
func (k msgServer) ConfirmBatch(c context.Context, msg *types.MsgConfirmBatch) (*types.MsgConfirmBatchResponse, error) {
	fmt.Println("ConfirmBatch - msg: ", msg)
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	tokenContract := common.HexToAddress(msg.TokenContract)

	// fetch the outgoing batch given the nonce
	batch := k.GetOutgoingTXBatch(ctx, tokenContract, msg.Nonce, msg.HyperionId)
	if batch == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "couldn't find batch")
	}
	fmt.Println("ConfirmBatch - batch: ", batch)

	checkpoint := batch.GetCheckpoint(msg.HyperionId)
	fmt.Println("ConfirmBatch - checkpoint: ", checkpoint)

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "signature decoding")
	}
	fmt.Println("ConfirmBatch - sigBytes: ", sigBytes)
	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidatorByHyperionID(ctx, orchaddr, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}
	fmt.Println("ConfirmBatch - validator: ", validator)
	ethAddress, found := k.GetEthAddressByValidatorByHyperionID(ctx, validator, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrEmpty, "eth address not found")
	}
	fmt.Println("ConfirmBatch - ethAddress: ", ethAddress)
	err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress)
	if err != nil {
		description := fmt.Sprintf(
			"signature verification failed expected sig by %s with hyperion-id %s with checkpoint %s found %s",
			ethAddress, msg.HyperionId, checkpoint.Hex(), msg.Signature,
		)

		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, description)
	}
	fmt.Println("ConfirmBatch - err: ", err)
	// check if we already have this confirm
	if k.GetBatchConfirm(ctx, msg.Nonce, tokenContract, orchaddr, msg.HyperionId) != nil {
		fmt.Println("ConfirmBatch - duplicate signature")
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "duplicate signature")
	}
	k.SetBatchConfirm(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventConfirmBatch{
		HyperionId:          msg.HyperionId,
		BatchNonce:          msg.Nonce,
		OrchestratorAddress: orchaddr.String(),
	})

	return nil, nil
}

// DepositClaim handles MsgDepositClaim
// TODO it is possible to submit an old msgDepositClaim (old defined as covering an event nonce that has already been
// executed aka 'observed' and had it's slashing window expire) that will never be cleaned up in the endblocker. This
// should not be a security risk as 'old' events can never execute but it does store spam in the chain.
func (k msgServer) DepositClaim(c context.Context, msg *types.MsgDepositClaim) (*types.MsgDepositClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidatorByHyperionID(ctx, orchestrator, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	// Check if the claim data is a valid sdk.Msg. If not, ignore the data
	if msg.Data != "" {
		ethereumSenderHeliosAccAddr := sdk.AccAddress(common.FromHex(msg.EthereumSender))
		if _, err := k.ValidateClaimData(ctx, msg.Data, ethereumSenderHeliosAccAddr); err != nil {
			k.Logger(ctx).Info("claim data is not a valid sdk.Msg", err)
			msg.Data = ""
		}
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgDepositClaimResponse{}, nil
}

// WithdrawClaim handles MsgWithdrawClaim
// TODO it is possible to submit an old msgWithdrawClaim (old defined as covering an event nonce that has already been
// executed aka 'observed' and had it's slashing window expire) that will never be cleaned up in the endblocker. This
// should not be a security risk as 'old' events can never execute but it does store spam in the chain.
func (k msgServer) WithdrawClaim(c context.Context, msg *types.MsgWithdrawClaim) (*types.MsgWithdrawClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidatorByHyperionID(ctx, orchestrator, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgWithdrawClaimResponse{}, nil
}

// ERC20DeployedClaim handles MsgERC20Deployed
func (k msgServer) ERC20DeployedClaim(c context.Context, msg *types.MsgERC20DeployedClaim) (*types.MsgERC20DeployedClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orch, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidatorByHyperionID(ctx, orch, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgERC20DeployedClaimResponse{}, nil
}

// ValsetUpdateClaim handles claims for executing a validator set update on Ethereum
func (k msgServer) ValsetUpdateClaim(c context.Context, msg *types.MsgValsetUpdatedClaim) (*types.MsgValsetUpdatedClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.GetOrchestratorValidatorByHyperionID(ctx, orchaddr, msg.HyperionId)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	any, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Attest(ctx, msg, any)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	return &types.MsgValsetUpdatedClaimResponse{}, nil
}

func (k msgServer) CancelSendToChain(c context.Context, msg *types.MsgCancelSendToChain) (*types.MsgCancelSendToChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	err = k.RemoveFromOutgoingPoolAndRefund(ctx, msg.TransactionId, sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventCancelSendToChain{
		OutgoingTxId: msg.TransactionId,
	})

	return &types.MsgCancelSendToChainResponse{}, nil
}

func (k msgServer) SubmitBadSignatureEvidence(c context.Context, msg *types.MsgSubmitBadSignatureEvidence) (*types.MsgSubmitBadSignatureEvidenceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	err := k.CheckBadSignatureEvidence(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSubmitBadSignatureEvidence{
		BadEthSignature:        msg.Signature,
		BadEthSignatureSubject: msg.Subject.String(),
	})

	if err != nil {
		metrics.ReportFuncError(k.svcTags)
	}

	return &types.MsgSubmitBadSignatureEvidenceResponse{}, err
}

func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.Authority != k.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)
	k.SetParams(ctx, &msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

func (k msgServer) BlacklistEthereumAddresses(ctx context.Context, msg *types.MsgBlacklistEthereumAddresses) (*types.MsgBlacklistEthereumAddressesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.authority == msg.Signer || k.isAdmin(sdkContext, msg.Signer)
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Hyperion module admins", msg.Signer)
	}

	for _, address := range msg.BlacklistAddresses {
		blacklistAddr, err := types.NewEthAddress(address)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", address)
		}
		k.SetEthereumBlacklistAddress(sdkContext, *blacklistAddr)
	}

	return &types.MsgBlacklistEthereumAddressesResponse{}, nil
}

func (k msgServer) RevokeEthereumBlacklist(ctx context.Context, msg *types.MsgRevokeEthereumBlacklist) (*types.MsgRevokeEthereumBlacklistResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.authority == msg.Signer || k.isAdmin(sdkContext, msg.Signer)
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Hyperion module admins", msg.Signer)
	}

	for _, blacklistAddress := range msg.BlacklistAddresses {

		blacklistAddr, err := types.NewEthAddress(blacklistAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", blacklistAddress)
		}

		if !k.IsOnBlacklist(sdkContext, *blacklistAddr) {
			return nil, fmt.Errorf("invalid blacklist address")
		} else {
			k.DeleteEthereumBlacklistAddress(sdkContext, *blacklistAddr)
		}
	}

	return &types.MsgRevokeEthereumBlacklistResponse{}, nil
}
