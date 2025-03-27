package keeper

import (
	"context"
	"encoding/hex"
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

type msgServer struct {
	Keeper Keeper

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

// [Used In Hyperion] SetOrchestratorAddresses handles the setting of orchestrator and Ethereum addresses for a validator.
// -------------
// MsgSetOrchestratorAddresses
// This function ensures that the validator exists and that the orchestrator and Ethereum addresses are not already set.
// If the validator is valid and bonded and the addresses are not already associated with another validator, it sets the orchestrator
// and Ethereum addresses for the given validator. It then emits an event with the new addresses to signal the successful
// update of the orchestrator and Ethereum addresses.
// -------------
func (k msgServer) SetOrchestratorAddresses(c context.Context, msg *types.MsgSetOrchestratorAddresses) (*types.MsgSetOrchestratorAddressesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	validatorAddr := sdk.ValAddress(validatorAccountAddr.Bytes())

	// get orchestrator address if available. otherwise default to validator address.
	var orchestratorAddr sdk.AccAddress
	if msg.Orchestrator != "" {
		orchestratorAddr, _ = sdk.AccAddressFromBech32(msg.Orchestrator)
	} else {
		orchestratorAddr = validatorAccountAddr
	}

	valAddr, foundExistingOrchestratorKey := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchestratorAddr)
	ethAddress, foundExistingEthAddress := k.Keeper.GetEthAddressByValidator(ctx, msg.HyperionId, validatorAddr)
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
	k.Keeper.SetOrchestratorValidator(ctx, msg.HyperionId, validatorAddr, orchestratorAddr)
	// set the ethereum address
	fmt.Println("msg.EthAddress: ", msg.EthAddress)
	ethAddr := common.HexToAddress(msg.EthAddress)
	fmt.Println("ethAddr: ", ethAddr)
	k.Keeper.SetEthAddressForValidator(ctx, msg.HyperionId, validatorAddr, ethAddr)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetOrchestratorAddresses{
		ValidatorAddress:    validatorAddr.String(),
		OrchestratorAddress: orchestratorAddr.String(),
		OperatorEthAddress:  msg.EthAddress,
		HyperionId:          msg.HyperionId,
	})
	fmt.Println("SetOrchestratorAddresses success")

	return &types.MsgSetOrchestratorAddressesResponse{}, nil

}

// [Not Used In Hyperion] AddCounterpartyChainParams msgServer allows adding connectivity for a new blockchain.
// -------------
// MsgAddCounterpartyChainParams
// For this, a new HyperionId and a new BridgeChainId are required.
// The function validates the counterparty chain parameters provided in the message,
// checks that the HyperionId and BridgeChainId are not already in use to avoid duplicates,
// and if the validations are successful, it adds the new counterparty chain parameters
// to the existing parameters and saves them.
// TODO: using it via proposal and check it well approved before adds.
// -------------
func (k msgServer) AddCounterpartyChainParams(c context.Context, msg *types.MsgAddCounterpartyChainParams) (*types.MsgAddCounterpartyChainParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	k.Keeper.Logger(ctx).Info("AddCounterpartyChainParams -1")
	// todo check msg.orchestrator funds and pay the cost of AddCounterpartyChain to the fundation

	if err := msg.CounterpartyChainParams.ValidateBasic(); err != nil {
		return nil, err
	}

	k.Keeper.Logger(ctx).Info("AddCounterpartyChainParams 0")
	params := k.Keeper.GetParams(ctx)

	k.Keeper.Logger(ctx).Info("AddCounterpartyChainParams 1")

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.HyperionId == msg.CounterpartyChainParams.HyperionId {
			return nil, errors.Wrap(types.ErrDuplicate, "HyperionId already exists")
		}
		if counterpartyChainParam.BridgeChainId == msg.CounterpartyChainParams.BridgeChainId {
			return nil, errors.Wrap(types.ErrDuplicate, "BridgeChainId already exists")
		}
	}

	k.Keeper.Logger(ctx).Info("AddCounterpartyChainParams 2")

	params.CounterpartyChainParams = append(params.CounterpartyChainParams, msg.CounterpartyChainParams)
	k.Keeper.SetParams(ctx, params)

	k.Keeper.Logger(ctx).Info("AddCounterpartyChainParams 3")

	return &types.MsgAddCounterpartyChainParamsResponse{}, nil
}

// [Used In Hyperion] ValsetConfirm handles MsgValsetConfirm
// -------------
// MsgValsetConfirm
// this is the message sent by the validators when they wish to submit their
// signatures over the validator set at a given block height. A validator must
// first call MsgSetEthAddress to set their Ethereum address to be used for
// signing. Then someone (anyone) must make a ValsetRequest the request is
// essentially a messaging mechanism to determine which block all validators
// should submit signatures over. Finally validators sign the validator set,
// powers, and Ethereum addresses of the entire validator set at the height of a
// ValsetRequest and submit that signature with this message.
//
// If a sufficient number of validators (66% of voting power) (A) have set
// Ethereum addresses and (B) submit ValsetConfirm messages with their
// signatures it is then possible for anyone to view these signatures in the
// chain store and submit them to Ethereum to update the validator set
// -------------
func (k msgServer) ValsetConfirm(c context.Context, msg *types.MsgValsetConfirm) (*types.MsgValsetConfirmResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	valset := k.Keeper.GetValset(ctx, msg.HyperionId, msg.Nonce)
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
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress, found := k.Keeper.GetEthAddressByValidator(ctx, msg.HyperionId, validator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrEmpty, "no eth address found")
	}

	if err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress); err != nil {
		description := fmt.Sprintf(
			"signature verification failed expected sig by %s with hyperion-id %s with checkpoint %s found %s",
			ethAddress, msg.HyperionId, checkpoint.Hex(), msg.Signature,
		)

		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, description)
	}

	// persist signature
	if k.Keeper.GetValsetConfirm(ctx, msg.HyperionId, msg.Nonce, orchaddr) != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "signature duplicate")
	}
	k.Keeper.SetValsetConfirm(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventValsetConfirm{
		HyperionId:          msg.HyperionId,
		ValsetNonce:         msg.Nonce,
		OrchestratorAddress: orchaddr.String(),
	})

	return &types.MsgValsetConfirmResponse{}, nil
}

// [Used In Hyperion (present in the client but not used)] SendToChain handles MsgSendToChain
// -------------
// MsgSendToChain
// This is the message that a user calls when they want to bridge an asset
// it will later be removed when it is included in a batch and successfully
// submitted tokens are removed from the users balance immediately
// -------
// AMOUNT:
// the coin to send across the bridge, note the restriction that this is a
// single coin not a set of coins that is normal in other Cosmos messages
// FEE:
// the fee paid for the bridge, distinct from the fee paid to the chain to
// actually send this message in the first place. So a successful send has
// two layers of fees for the user
// -------------
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
	if k.Keeper.InvalidSendToChainAddress(ctx, dest) {
		return nil, errors.Wrap(types.ErrInvalidEthDestination, "destination address is invalid or blacklisted")
	}

	txID, err := k.Keeper.AddToOutgoingPool(ctx, sender, common.HexToAddress(msg.Dest), msg.Amount, msg.BridgeFee, msg.DestHyperionId)
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

// [Used In Hyperion]  RequestBatch handles MsgRequestBatch
// -------------
// MsgRequestBatch
// this is a message anyone can send that requests a batch of transactions to
// send across the bridge be created for whatever block height this message is
// included in. This acts as a coordination point, the handler for this message
// looks at the AddToOutgoingPool tx's in the store and generates a batch, also
// available in the store tied to this message. The validators then grab this
// batch, sign it, submit the signatures with a MsgConfirmBatch before a relayer
// can finally submit the batch
// -------------
func (k msgServer) RequestBatch(c context.Context, msg *types.MsgRequestBatch) (*types.MsgRequestBatchResponse, error) {
	fmt.Println("RequestBatch, got msg request batch from hyperion - msg: ", msg)
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	// Check if the denom is a hyperion coin, if not, check if there is a deployed ERC20 representing it.
	// If not, error out
	_, tokenContract, err := k.Keeper.DenomToERC20Lookup(ctx, msg.Denom, msg.HyperionId)
	if err != nil {
		fmt.Println("RequestBatch - err: ", err)
		return nil, err
	}

	batch, err := k.Keeper.BuildOutgoingTXBatch(ctx, tokenContract, msg.HyperionId, OutgoingTxBatchSize)
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

// [Used In Hyperion] ConfirmBatch handles MsgConfirmBatch
// -------------
// MsgConfirmBatch
// When validators observe a MsgRequestBatch they form a batch by ordering
// transactions currently in the txqueue in order of highest to lowest fee,
// cutting off when the batch either reaches a hardcoded maximum size (to be
// decided, probably around 100) or when transactions stop being profitable
// (TODO determine this without nondeterminism) This message includes the batch
// as well as an Ethereum signature over this batch by the validator
// -------------
func (k msgServer) ConfirmBatch(c context.Context, msg *types.MsgConfirmBatch) (*types.MsgConfirmBatchResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	tokenContract := common.HexToAddress(msg.TokenContract)

	// fetch the outgoing batch given the nonce
	batch := k.Keeper.GetOutgoingTXBatch(ctx, tokenContract, msg.Nonce, msg.HyperionId)
	if batch == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "couldn't find batch")
	}

	checkpoint := batch.GetCheckpoint(msg.HyperionId)

	sigBytes, err := hex.DecodeString(msg.Signature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "signature decoding")
	}

	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	ethAddress, found := k.Keeper.GetEthAddressByValidator(ctx, msg.HyperionId, validator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrEmpty, "eth address not found")
	}

	err = types.ValidateEthereumSignature(checkpoint, sigBytes, ethAddress)
	if err != nil {
		description := fmt.Sprintf(
			"signature verification failed expected sig by %s with hyperion-id %s with checkpoint %s found %s",
			ethAddress, msg.HyperionId, checkpoint.Hex(), msg.Signature,
		)

		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, description)
	}

	// check if we already have this confirm
	if k.Keeper.GetBatchConfirm(ctx, msg.HyperionId, msg.Nonce, tokenContract, orchaddr) != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "duplicate signature")
	}
	k.Keeper.SetBatchConfirm(ctx, msg)

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventConfirmBatch{
		HyperionId:          msg.HyperionId,
		BatchNonce:          msg.Nonce,
		OrchestratorAddress: orchaddr.String(),
	})

	return nil, nil
}

// [Used In Hyperion] DepositClaim handles MsgDepositClaim
// -------------
// MsgDepositClaim
// When more than 66% of the active validator set has
// claimed to have seen the deposit enter the source blockchain coins are
// issued to the Cosmos address in question
// -------------
func (k msgServer) DepositClaim(c context.Context, msg *types.MsgDepositClaim) (*types.MsgDepositClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.Keeper.StakingKeeper.Validator(ctx, validator)
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
		metadata, msg, err := k.Keeper.parseClaimData(ctx, msg.Data)
		if err != nil {
			k.Keeper.Logger(ctx).Info("claim data is not valid", "err", err)
			return nil, err
		}
		if metadata != nil {
			if _, err := k.Keeper.ValidateTokenMetaData(ctx, metadata); err != nil {
				k.Keeper.Logger(ctx).Info("claim data is not valid - TokenMetaData is not valid", "err", err)
				return nil, err
			}
		}
		if msg != nil {
			if _, err := k.Keeper.handleValidateMsg(ctx, msg); err != nil {
				k.Keeper.Logger(ctx).Info("claim data is not valid - sdk.Msg is not valid", "err", err)
				return nil, err
			}
		}
	}

	a, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Keeper.Attest(ctx, msg, a)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	k.Keeper.Logger(ctx).Info("DepositClaim Received with success", "msg", msg)

	return &types.MsgDepositClaimResponse{}, nil
}

// [Used In Hyperion] WithdrawClaim handles MsgWithdrawClaim
// -------------
// WithdrawClaim claims that a batch of withdrawal
// operations on the bridge contract was executed.
// -------------
func (k msgServer) WithdrawClaim(c context.Context, msg *types.MsgWithdrawClaim) (*types.MsgWithdrawClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.Keeper.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	a, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Keeper.Attest(ctx, msg, a)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	k.Keeper.Logger(ctx).Info("WithdrawClaim Received with success", "msg", msg)

	return &types.MsgWithdrawClaimResponse{}, nil
}

// [Used In Hyperion] ERC20DeployedClaim handles MsgERC20Deployed
// -------------
// MsgERC20DeployedClaim claims that new erc20 token
// was deployed on the source blockchain and will be linked
// as ERC20 to cosmosDenom in hyperion Module on HyperionId
// -------------
func (k msgServer) ERC20DeployedClaim(c context.Context, msg *types.MsgERC20DeployedClaim) (*types.MsgERC20DeployedClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orch, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orch)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.Keeper.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	a, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Keeper.Attest(ctx, msg, a)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	k.Keeper.Logger(ctx).Info("ERC20DeployedClaim Received with success", "msg", msg)

	return &types.MsgERC20DeployedClaimResponse{}, nil
}

// [Used In Hyperion] ValsetUpdateClaim handles claims for executing a validator set update on Ethereum
// -------------
// MsgValsetUpdatedClaim this message permit to share to
// hyperion module the valset was updated on source blockchain
// this permit to insure the power is well share on both side.
// -------------
func (k msgServer) ValsetUpdateClaim(c context.Context, msg *types.MsgValsetUpdatedClaim) (*types.MsgValsetUpdatedClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchaddr, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchaddr)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	// return an error if the validator isn't in the active set
	val, err := k.Keeper.StakingKeeper.Validator(ctx, validator)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "validator can't be retrieved")
	}
	if val == nil || !val.IsBonded() {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(sdkerrors.ErrorInvalidSigner, "validator not in active set")
	}

	a, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	// Add the claim to the store
	_, err = k.Keeper.Attest(ctx, msg, a)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(err, "create attestation")
	}

	k.Keeper.Logger(ctx).Info("ValsetUpdateClaim Received with success", "msg", msg)

	return &types.MsgValsetUpdatedClaimResponse{}, nil
}

// [Not Used In Hyperion] CancelSendToChain
// -------------
// MsgCancelSendToChain permit to cancel send
// to chain if the sendtochain is always in the tx pool.
// -------------
func (k msgServer) CancelSendToChain(c context.Context, msg *types.MsgCancelSendToChain) (*types.MsgCancelSendToChainResponse, error) {
	// c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	// defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	err = k.Keeper.RemoveFromOutgoingPoolAndRefund(ctx, msg.TransactionId, sender)
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

// [Not Used In Hyperion] SubmitBadSignatureEvidence
// -------------
// MsgSubmitBadSignatureEvidence
// This call allows anyone to submit evidence
// that a validator has signed a valset, batch,
// or logic call that never existed. Subject
// contains the batch, valset, or logic call.
// -------------
func (k msgServer) SubmitBadSignatureEvidence(c context.Context, msg *types.MsgSubmitBadSignatureEvidence) (*types.MsgSubmitBadSignatureEvidenceResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	err := k.Keeper.CheckBadSignatureEvidence(ctx, msg)

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

// [Not Used In Hyperion] UpdateParams
// -------------
// MsgUpdateParams
// This call permit to change in one time all the params.
// TODO: remove this call not good to have an authority who can touch alone the params.
// -------------
func (k msgServer) UpdateParams(c context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	if msg.Authority != k.Keeper.authority {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "invalid authority: expected %s, got %s", k.Keeper.authority, msg.Authority)
	}

	if err := msg.Params.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)
	k.Keeper.SetParams(ctx, &msg.Params)

	return &types.MsgUpdateParamsResponse{}, nil
}

// [Not Used In Hyperion] BlacklistEthereumAddresses
// -------------
// MsgBlacklistEthereumAddresses
// Defines the message used to add Ethereum addresses to all hyperion blacklists.
// TODO: adding this call on proposals and remove authority
// -------------
func (k msgServer) BlacklistEthereumAddresses(ctx context.Context, msg *types.MsgBlacklistEthereumAddresses) (*types.MsgBlacklistEthereumAddressesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.Keeper.authority == msg.Signer || k.Keeper.isAdmin(sdkContext, msg.Signer)
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Hyperion module admins", msg.Signer)
	}

	for _, address := range msg.BlacklistAddresses {
		blacklistAddr, err := types.NewEthAddress(address)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", address)
		}
		k.Keeper.SetEthereumBlacklistAddress(sdkContext, *blacklistAddr)
	}

	return &types.MsgBlacklistEthereumAddressesResponse{}, nil
}

// [Not Used In Hyperion] RevokeEthereumBlacklist
// -------------
// MsgRevokeEthereumBlacklist
// Defines the message used to remove Ethereum addresses from hyperion blacklist.
// TODO: adding this call on proposals and remove authority
// -------------
func (k msgServer) RevokeEthereumBlacklist(ctx context.Context, msg *types.MsgRevokeEthereumBlacklist) (*types.MsgRevokeEthereumBlacklistResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.Keeper.authority == msg.Signer || k.Keeper.isAdmin(sdkContext, msg.Signer)
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Hyperion module admins", msg.Signer)
	}

	for _, blacklistAddress := range msg.BlacklistAddresses {

		blacklistAddr, err := types.NewEthAddress(blacklistAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", blacklistAddress)
		}

		if !k.Keeper.IsOnBlacklist(sdkContext, *blacklistAddr) {
			return nil, fmt.Errorf("invalid blacklist address")
		} else {
			k.Keeper.DeleteEthereumBlacklistAddress(sdkContext, *blacklistAddr)
		}
	}

	return &types.MsgRevokeEthereumBlacklistResponse{}, nil
}
