package keeper

import (
	"context"
	"encoding/hex"
	"fmt"
	"slices"

	evmtypes "helios-core/helios-chain/x/evm/types"

	cmn "helios-core/helios-chain/precompiles/common"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto/tmhash"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/types"
)

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
			"signature verification failed expected sig by %s with hyperion-id %d with checkpoint %s found %s",
			ethAddress.String(), msg.HyperionId, checkpoint.Hex(), msg.Signature,
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

	orchestratorData, err := k.Keeper.GetOrchestratorHyperionData(ctx, orchaddr, msg.HyperionId)
	if err != nil {
		k.Keeper.Logger(ctx).Error("failed to get orchestrator data", "error", err, "hyperion_id", msg.HyperionId, "orchestrator", orchaddr)
		return nil, errors.Wrap(types.ErrInvalid, "failed to get orchestrator data")
	}
	orchestratorData.BatchConfirmed++
	k.Keeper.SetOrchestratorHyperionData(ctx, orchaddr, msg.HyperionId, *orchestratorData)

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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	txHash := ""
	decodedTx, err := k.Keeper.txDecoder(ctx.TxBytes())
	if err == nil {
		for _, msg := range decodedTx.GetMsgs() {
			ethMessage, ok := msg.(*evmtypes.MsgEthereumTx)
			if !ok {
				// Just considers Ethereum transactions
				continue
			}
			txHash = ethMessage.AsTransaction().Hash().Hex()
		}
	}
	if txHash == "" { // it's a pure cosmos tx
		txHash = fmt.Sprintf("%X", tmhash.Sum(ctx.TxBytes()))
	}

	// Fees validation (looks not working) Todo check Better @Jiji @JeremyGuyet
	///////////////////////////////
	lowestFeeValidator := k.Keeper.GetLowestFeeValidator(ctx, msg.DestChainId)
	if lowestFeeValidator == nil {
		return nil, errors.Wrap(types.ErrInvalid, "no lowest fee validator found")
	}
	lowestFee := k.Keeper.GetFeeByValidator(ctx, msg.DestChainId, *lowestFeeValidator)
	if lowestFee == nil {
		return nil, errors.Wrap(types.ErrInvalid, "no lowest fee found")
	}
	///////////////////////////////

	if msg.BridgeFee.Denom != "ahelios" {
		return nil, errors.Wrap(types.ErrInvalid, "fee denom must be ahelios")
	}
	if msg.BridgeFee.Amount.LT(lowestFee.Amount) {
		return nil, errors.Wrap(types.ErrInvalid, "fee is less than the lowest fee")
	}
	// fmt.Println("chainId :", msg.DestChainId)
	// fmt.Println("denom :", msg.Amount.Denom)
	// fmt.Println("lowestFee :", lowestFee)
	// fmt.Println("msg.BridgeFee :", msg.BridgeFee)
	//-------------------------------------

	dest := common.HexToAddress(msg.Dest)
	if k.Keeper.InvalidSendToChainAddress(ctx, dest) {
		return nil, errors.Wrap(types.ErrInvalidEthDestination, "destination address is invalid or blacklisted")
	}

	hyperionParams := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.DestChainId)

	if hyperionParams == nil {
		return nil, errors.Wrap(types.ErrInvalidEthDestination, "destination chainId doesn't exists")
	}
	if hyperionParams.Paused {
		return nil, errors.Wrap(types.ErrInvalidEthDestination, "destination chain is paused")
	}
	hyperionId := hyperionParams.HyperionId

	txID, err := k.Keeper.AddToOutgoingPool(ctx, sender, common.HexToAddress(msg.Dest), msg.Amount, msg.BridgeFee, hyperionId, txHash)
	if err != nil {
		return nil, err
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSendToChain{
		HyperionId:   hyperionId,
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
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	// Check if the denom is a hyperion coin, if not, check if there is a deployed ERC20 representing it.
	// If not, error out
	tokenAddressToDenom, exists := k.Keeper.GetTokenFromDenom(ctx, msg.HyperionId, msg.Denom)
	if !exists {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalid, "token not found")
	}
	tokenContract := common.HexToAddress(tokenAddressToDenom.TokenAddress)

	batch, err := k.Keeper.BuildOutgoingTXBatch(ctx, tokenContract, msg.HyperionId, OutgoingTxBatchSize, sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0)), sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(0)))
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

func (k msgServer) RequestBatchWithMinimumFee(c context.Context, msg *types.MsgRequestBatchWithMinimumFee) (*types.MsgRequestBatchWithMinimumFeeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	// Check if the denom is a hyperion coin, if not, check if there is a deployed ERC20 representing it.
	// If not, error out
	tokenAddressToDenom, exists := k.Keeper.GetTokenFromDenom(ctx, msg.HyperionId, msg.Denom)
	if !exists {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrapf(types.ErrInvalid, "token not found")
	}
	tokenContract := common.HexToAddress(tokenAddressToDenom.TokenAddress)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}

	orchestratorFee := k.Keeper.GetFeeByValidator(ctx, msg.HyperionId, validator)
	if orchestratorFee == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "orchestrator fee not found - please set a fee for your orchestrator")
	}

	if msg.MinimumBatchFee.Amount.LT(orchestratorFee.Amount) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "minimum batch fee is less than the orchestrator fee")
	}

	batch, err := k.Keeper.BuildOutgoingTXBatchWithIds(ctx, tokenContract, msg.HyperionId, OutgoingTxBatchSize, msg.MinimumBatchFee, msg.MinimumTxFee, msg.TxIds)
	if err != nil {
		return nil, err
	}

	// Update orchestrator data
	orchestratorData, err := k.Keeper.GetOrchestratorHyperionData(ctx, orchestrator, msg.HyperionId)
	if err != nil {
		k.Keeper.Logger(ctx).Error("failed to get orchestrator data", "error", err, "hyperion_id", msg.HyperionId, "orchestrator", orchestrator)
		return nil, err
	}
	orchestratorData.BatchCreated++
	k.Keeper.SetOrchestratorHyperionData(ctx, orchestrator, msg.HyperionId, *orchestratorData)

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

	return &types.MsgRequestBatchWithMinimumFeeResponse{}, nil
}

func (k msgServer) ConfirmMultipleBatches(c context.Context, msg *types.MsgConfirmMultipleBatches) (*types.MsgConfirmMultipleBatchesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	for _, batch := range msg.Batches {
		_, err := k.ConfirmBatch(ctx, &types.MsgConfirmBatch{
			HyperionId:    msg.HyperionId,
			Nonce:         batch.Nonce,
			TokenContract: batch.TokenContract,
			Signature:     batch.Signature,
			Orchestrator:  msg.Orchestrator,
		})
		if err != nil && !errors.IsOf(err, types.ErrDuplicate) { // ignore duplicate errors
			return nil, err
		}
	}

	return &types.MsgConfirmMultipleBatchesResponse{}, nil
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
			"signature verification failed expected sig by %s with hyperion-id %d with checkpoint %s found %s",
			ethAddress.String(), msg.HyperionId, checkpoint.Hex(), msg.Signature,
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
		metadata, _, err := k.Keeper.parseClaimData(ctx, msg.Data)
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
		// if msg != nil {
		// 	if _, err := k.Keeper.handleValidateMsg(ctx, msg); err != nil {
		// 		k.Keeper.Logger(ctx).Info("claim data is not valid - sdk.Msg is not valid", "err", err)
		// 		return nil, err
		// 	}
		// }
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

	return &types.MsgWithdrawClaimResponse{}, nil
}

func (k msgServer) ExternalDataClaim(c context.Context, msg *types.MsgExternalDataClaim) (*types.MsgExternalDataClaimResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	orchestrator, _ := sdk.AccAddressFromBech32(msg.Orchestrator)
	validator, found := k.Keeper.GetOrchestratorValidator(ctx, msg.HyperionId, orchestrator)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrUnknown, "validator")
	}
	orchestratorHexAddress := cmn.AnyToHexAddress(msg.Orchestrator)

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
	externalContractAddress := common.HexToAddress(msg.ExternalContractAddress)
	tx := k.Keeper.GetOutgoingExternalDataTX(ctx, externalContractAddress, msg.TxNonce, msg.HyperionId)

	if tx == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "tx not found or already executed")
	}

	if slices.Contains(tx.Votes, orchestratorHexAddress.Hex()) {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrInvalid, "already voted")
	}

	tx.Votes = append(tx.Votes, orchestratorHexAddress.Hex())
	tx.Claims = append(tx.Claims, msg)

	k.Keeper.StoreExternalData(ctx, tx)

	return &types.MsgExternalDataClaimResponse{}, nil
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

	return &types.MsgValsetUpdatedClaimResponse{}, nil
}
