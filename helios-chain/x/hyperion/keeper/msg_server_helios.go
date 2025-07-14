package keeper

import (
	"context"
	"fmt"

	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/Helios-Chain-Labs/metrics"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/hyperion/types"
)

// [Not Used In Hyperion] SetOrchestratorAddresses handles the setting of orchestrator and Ethereum addresses for a validator.
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
	ethAddr := common.HexToAddress(msg.EthAddress)
	k.Keeper.SetEthAddressForValidator(ctx, msg.HyperionId, validatorAddr, ethAddr)

	if _, err := k.Keeper.GetOrchestratorHyperionData(ctx, orchestratorAddr, msg.HyperionId); err != nil {
		k.Keeper.SetOrchestratorHyperionData(ctx, orchestratorAddr, msg.HyperionId, types.OrchestratorHyperionData{
			HyperionId:                 msg.HyperionId,
			MinimumTxFee:               math.NewInt(0),
			MinimumBatchFee:            math.NewInt(0),
			TotalSlashCount:            0,
			TotalSlashAmount:           math.NewInt(0),
			SlashData:                  make([]*types.SlashData, 0),
			TxOutTransfered:            0,
			TxInTransfered:             0,
			BatchCreated:               0,
			BatchConfirmed:             0,
			FeeCollected:               math.NewInt(0),
			ExternalDataTxExecuted:     0,
			ExternalDataTxFeeCollected: math.NewInt(0),
		})
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetOrchestratorAddresses{
		ValidatorAddress:    validatorAddr.String(),
		OrchestratorAddress: orchestratorAddr.String(),
		OperatorEthAddress:  msg.EthAddress,
		HyperionId:          msg.HyperionId,
	})

	return &types.MsgSetOrchestratorAddressesResponse{}, nil
}

func (k msgServer) SetOrchestratorAddressesWithFee(c context.Context, msg *types.MsgSetOrchestratorAddressesWithFee) (*types.MsgSetOrchestratorAddressesWithFeeResponse, error) {
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

	if msg.MinimumTxFee.Denom != sdk.DefaultBondDenom || msg.MinimumBatchFee.Denom != sdk.DefaultBondDenom {
		return nil, errors.Wrap(types.ErrInvalid, "fee denom must be "+sdk.DefaultBondDenom)
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
	ethAddr := cmn.AnyToHexAddress(validatorAddr.String())

	k.Keeper.SetEthAddressForValidator(ctx, msg.HyperionId, validatorAddr, ethAddr)
	// set the fee
	k.Keeper.SetFeeForValidator(ctx, msg.HyperionId, validatorAddr, msg.MinimumTxFee)

	if _, err := k.Keeper.GetOrchestratorHyperionData(ctx, orchestratorAddr, msg.HyperionId); err != nil {
		k.Keeper.SetOrchestratorHyperionData(ctx, orchestratorAddr, msg.HyperionId, types.OrchestratorHyperionData{
			HyperionId:                 msg.HyperionId,
			MinimumTxFee:               msg.MinimumTxFee.Amount,
			MinimumBatchFee:            msg.MinimumBatchFee.Amount,
			TotalSlashCount:            0,
			TotalSlashAmount:           math.NewInt(0),
			SlashData:                  make([]*types.SlashData, 0),
			TxOutTransfered:            0,
			TxInTransfered:             0,
			BatchCreated:               0,
			BatchConfirmed:             0,
			FeeCollected:               math.NewInt(0),
			ExternalDataTxExecuted:     0,
			ExternalDataTxFeeCollected: math.NewInt(0),
		})
	} else {
		orchestratorData, err := k.Keeper.GetOrchestratorHyperionData(ctx, orchestratorAddr, msg.HyperionId)
		if err != nil {
			return nil, err
		}
		orchestratorData.MinimumTxFee = msg.MinimumTxFee.Amount
		orchestratorData.MinimumBatchFee = msg.MinimumBatchFee.Amount
		k.Keeper.SetOrchestratorHyperionData(ctx, orchestratorAddr, msg.HyperionId, *orchestratorData)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventSetOrchestratorAddresses{
		ValidatorAddress:    validatorAddr.String(),
		OrchestratorAddress: orchestratorAddr.String(),
		OperatorEthAddress:  msg.EthAddress,
		HyperionId:          msg.HyperionId,
	})

	return &types.MsgSetOrchestratorAddressesWithFeeResponse{}, nil
}

func (k msgServer) UpdateOrchestratorAddressesFee(c context.Context, msg *types.MsgUpdateOrchestratorAddressesFee) (*types.MsgUpdateOrchestratorAddressesFeeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	validatorAddr := sdk.ValAddress(validatorAccountAddr.Bytes())

	// get the fee
	fee, found := k.Keeper.GetFeeByValidator(ctx, msg.HyperionId, validatorAddr)
	if !found {
		return nil, errors.Wrap(types.ErrInvalid, "no fee found")
	}

	if msg.MinimumTxFee.Denom != sdk.DefaultBondDenom || msg.MinimumBatchFee.Denom != sdk.DefaultBondDenom {
		return nil, errors.Wrap(types.ErrInvalid, "fee denom must be "+sdk.DefaultBondDenom)
	}

	// update the fee
	fee.Amount = msg.MinimumTxFee.Amount
	k.Keeper.SetFeeForValidator(ctx, msg.HyperionId, validatorAddr, fee)

	orchestratorData, err := k.Keeper.GetOrchestratorHyperionData(ctx, validatorAccountAddr, msg.HyperionId)
	if err != nil {
		return nil, err
	}
	orchestratorData.MinimumTxFee = msg.MinimumTxFee.Amount
	orchestratorData.MinimumBatchFee = msg.MinimumBatchFee.Amount
	k.Keeper.SetOrchestratorHyperionData(ctx, validatorAccountAddr, msg.HyperionId, *orchestratorData)

	return &types.MsgUpdateOrchestratorAddressesFeeResponse{}, nil
}

func (k msgServer) DeleteOrchestratorAddressesFee(c context.Context, msg *types.MsgDeleteOrchestratorAddressesFee) (*types.MsgDeleteOrchestratorAddressesFeeResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	validatorAddr := sdk.ValAddress(validatorAccountAddr.Bytes())

	// get the fee
	_, found := k.Keeper.GetFeeByValidator(ctx, msg.HyperionId, validatorAddr)
	if !found {
		return nil, errors.Wrap(types.ErrInvalid, "no fee found")
	}

	k.Keeper.DeleteFeeForValidator(ctx, msg.HyperionId, validatorAddr)

	return &types.MsgDeleteOrchestratorAddressesFeeResponse{}, nil
}

func (k msgServer) UnSetOrchestratorAddresses(c context.Context, msg *types.MsgUnSetOrchestratorAddresses) (*types.MsgUnSetOrchestratorAddressesResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	validatorAccountAddr, _ := sdk.AccAddressFromBech32(msg.Sender)
	validatorAddr := sdk.ValAddress(validatorAccountAddr.Bytes())

	fmt.Println("msg.EthAddress: ", msg.EthAddress)
	ethAddr := common.HexToAddress(msg.EthAddress)
	fmt.Println("ethAddr: ", ethAddr)

	k.Keeper.DeleteOrchestratorValidator(ctx, msg.HyperionId, validatorAccountAddr)
	k.Keeper.DeleteEthAddressForValidator(ctx, msg.HyperionId, validatorAddr, ethAddr)
	k.Keeper.DeleteFeeForValidator(ctx, msg.HyperionId, validatorAddr)

	return &types.MsgUnSetOrchestratorAddressesResponse{}, nil

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

	// todo check msg.orchestrator funds and pay the cost of AddCounterpartyChain to the fundation

	fmt.Println("msg.Orchestrator: ", msg.Authority)
	fmt.Println("k.Keeper.GetAuthority(): ", k.Keeper.GetAuthority())

	if cmn.AnyToHexAddress(msg.Authority).Hex() != cmn.AnyToHexAddress(k.Keeper.GetAuthority()).Hex() {
		return nil, errors.Wrap(types.ErrInvalidSigner, "signer is not the authority")
	}

	vp, err := k.Keeper.GetLastValidatorPower(ctx, cmn.AnyToHexAddress(msg.CounterpartyChainParams.Initializer))
	if err != nil {
		return nil, err
	}
	if vp == 0 {
		return nil, errors.Wrap(types.ErrInvalidSigner, "initializer is not a validator")
	}

	if err := msg.CounterpartyChainParams.ValidateBasic(); err != nil {
		return nil, err
	}

	params := k.Keeper.GetParams(ctx)

	if msg.CounterpartyChainParams.HyperionId == 0 {
		return nil, errors.Wrap(types.ErrInvalidHyperionId, "HyperionId cannot be 0")
	}

	if msg.CounterpartyChainParams.HyperionId != msg.CounterpartyChainParams.BridgeChainId {
		return nil, errors.Wrap(types.ErrInvalidHyperionId, "HyperionId and BridgeChainId must be the same")
	}

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.HyperionId == msg.CounterpartyChainParams.HyperionId {
			return nil, errors.Wrap(types.ErrDuplicate, "HyperionId already exists")
		}
		if counterpartyChainParam.BridgeChainId == msg.CounterpartyChainParams.BridgeChainId {
			return nil, errors.Wrap(types.ErrDuplicate, "BridgeChainId already exists")
		}
	}

	params.CounterpartyChainParams = append(params.CounterpartyChainParams, msg.CounterpartyChainParams)
	k.Keeper.SetParams(ctx, params)

	for _, token := range msg.CounterpartyChainParams.DefaultTokens {
		k.Keeper.CreateOrLinkTokenToChain(ctx, msg.CounterpartyChainParams.BridgeChainId, msg.CounterpartyChainParams.BridgeChainName, token)
	}
	// setup a default value LastObservedEthereumBlockHeight
	k.Keeper.SetNewLastObservedEthereumBlockHeight(ctx, msg.CounterpartyChainParams.HyperionId, msg.CounterpartyChainParams.BridgeContractStartHeight)

	// set proposer as first validator
	k.Keeper.SetOrchestratorValidator(ctx, msg.CounterpartyChainParams.HyperionId, cmn.ValAddressFromHexAddress(cmn.AnyToHexAddress(msg.CounterpartyChainParams.Initializer)), cmn.AccAddressFromHexAddress(cmn.AnyToHexAddress(msg.CounterpartyChainParams.Initializer)))
	k.Keeper.SetEthAddressForValidator(ctx, msg.CounterpartyChainParams.HyperionId, cmn.ValAddressFromHexAddress(cmn.AnyToHexAddress(msg.CounterpartyChainParams.Initializer)), cmn.AnyToHexAddress(msg.CounterpartyChainParams.Initializer))

	// set first valset
	k.Keeper.SetLastObservedValset(ctx, msg.CounterpartyChainParams.HyperionId, types.Valset{
		HyperionId: msg.CounterpartyChainParams.HyperionId,
		Nonce:      1,
		Members: []*types.BridgeValidator{
			{
				Power:           1431655765,
				EthereumAddress: cmn.AnyToHexAddress(msg.CounterpartyChainParams.Initializer).Hex(),
			},
		},
		Height:       msg.CounterpartyChainParams.BridgeContractStartHeight - 1,
		RewardAmount: math.NewIntFromUint64(0),
		RewardToken:  common.Address{0x0000000000000000000000000000000000000000}.Hex(),
	})
	k.Keeper.setLastObservedEventNonce(ctx, msg.CounterpartyChainParams.HyperionId, 1)

	return &types.MsgAddCounterpartyChainParamsResponse{}, nil
}

func (k msgServer) UpdateCounterpartyChainInfosParams(c context.Context, msg *types.MsgUpdateCounterpartyChainInfosParams) (*types.MsgUpdateCounterpartyChainInfosParamsResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)

	params := k.Keeper.GetParams(ctx)

	if params == nil {
		return nil, errors.Wrap(types.ErrEmpty, "BridgeChainId not found")
	}

	updated := false

	for _, counterpartyChainParam := range params.CounterpartyChainParams {
		if counterpartyChainParam.BridgeChainId == msg.BridgeChainId {

			if cmn.AnyToHexAddress(counterpartyChainParam.Initializer).Hex() != cmn.AnyToHexAddress(msg.Signer).Hex() {
				return nil, errors.Wrap(types.ErrInvalidSigner, "signer is not the initializer")
			}

			counterpartyChainParam.BridgeChainLogo = msg.BridgeChainLogo
			counterpartyChainParam.BridgeChainName = msg.BridgeChainName

			// check if the counterparty chain param is valid
			if err := counterpartyChainParam.ValidateBasic(); err != nil {
				return nil, err
			}
			updated = true
			break
		}
	}

	if !updated {
		return nil, errors.Wrap(types.ErrEmpty, "BridgeChainId not found")
	}

	k.Keeper.SetParams(ctx, params)

	return &types.MsgUpdateCounterpartyChainInfosParamsResponse{}, nil
}

// [Not Used In Hyperion] CancelSendToChain
// -------------
// MsgCancelSendToChain permit to cancel send
// to chain if the sendtochain is always in the tx pool.
// -------------
func (k msgServer) CancelSendToChain(c context.Context, msg *types.MsgCancelSendToChain) (*types.MsgCancelSendToChainResponse, error) {
	c, doneFn := metrics.ReportFuncCallAndTimingCtx(c, k.svcTags)
	defer doneFn()

	ctx := sdk.UnwrapSDKContext(c)
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, err
	}

	params := k.Keeper.GetHyperionParamsFromChainId(ctx, msg.ChainId)
	if params == nil {
		metrics.ReportFuncError(k.svcTags)
		return nil, errors.Wrap(types.ErrDuplicate, "BridgeChainId not found")
	}

	err = k.Keeper.RemoveFromOutgoingPoolAndRefund(ctx, params.HyperionId, msg.TransactionId, sender)
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

// [Not Used In Hyperion] BlacklistAddresses
// -------------
// MsgBlacklistAddresses
// Defines the message used to add addresses to all hyperion blacklists.
// TODO: adding this call on proposals and remove authority
// -------------
func (k msgServer) BlacklistAddresses(ctx context.Context, msg *types.MsgBlacklistAddresses) (*types.MsgBlacklistAddressesResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.Keeper.authority == msg.Signer
	if !isValidSigner {
		return nil, errors.Wrapf(govtypes.ErrInvalidSigner, "the signer %s is not the valid authority or one of the Hyperion module admins", msg.Signer)
	}

	for _, address := range msg.BlacklistAddresses {
		blacklistAddr, err := types.NewEthAddress(address)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid blacklist address %s", address)
		}
		k.Keeper.SetBlacklistAddress(sdkContext, *blacklistAddr)
	}

	return &types.MsgBlacklistAddressesResponse{}, nil
}

// [Not Used In Hyperion] RevokeBlacklist
// -------------
// MsgRevokeBlacklist
// Defines the message used to remove addresses from hyperion blacklist.
// TODO: adding this call on proposals and remove authority
// -------------
func (k msgServer) RevokeBlacklist(ctx context.Context, msg *types.MsgRevokeBlacklist) (*types.MsgRevokeBlacklistResponse, error) {
	defer metrics.ReportFuncCallAndTiming(k.svcTags)()

	sdkContext := sdk.UnwrapSDKContext(ctx)

	isValidSigner := k.Keeper.authority == msg.Signer
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
			k.Keeper.DeleteBlacklistAddress(sdkContext, *blacklistAddr)
		}
	}

	return &types.MsgRevokeBlacklistResponse{}, nil
}
