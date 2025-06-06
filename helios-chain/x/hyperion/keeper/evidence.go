package keeper

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/hyperion/types"

	"github.com/Helios-Chain-Labs/metrics"
)

func (k *Keeper) CheckBadSignatureEvidence(
	ctx sdk.Context,
	msg *types.MsgSubmitBadSignatureEvidence,
) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	var subject types.EthereumSigned

	err := k.cdc.UnpackAny(msg.Subject, &subject)
	if err != nil {
		return err
	}

	switch subject := subject.(type) {
	case *types.OutgoingTxBatch:
		return k.checkBadSignatureEvidenceInternal(ctx, subject, msg.Signature, subject.HyperionId)
	case *types.Valset:
		return k.checkBadSignatureEvidenceInternal(ctx, subject, msg.Signature, subject.HyperionId)

	default:
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrInvalid, "Bad signature must be over a batch, valset, or logic call")
	}
}

func (k *Keeper) checkBadSignatureEvidenceInternal(ctx sdk.Context, subject types.EthereumSigned, signature string, hyperionId uint64) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// Get checkpoint of the supposed bad signature (fake valset, batch, or logic call submitted to eth)
	checkpoint := subject.GetCheckpoint(hyperionId)

	// Try to find the checkpoint in the archives. If it exists, we don't slash because
	// this is not a bad signature
	if k.GetPastEthSignatureCheckpoint(ctx, hyperionId, checkpoint) {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrInvalid, "Checkpoint exists, cannot slash")
	}

	// Decode Eth signature to bytes
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrInvalid, "signature decoding")
	}

	// Get eth address of the offending validator using the checkpoint and the signature
	ethAddress, err := types.EthAddressFromSignature(checkpoint, sigBytes)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrInvalid, fmt.Sprintf("signature to eth address failed with checkpoint %s and signature %s", checkpoint.Hex(), signature))
	}

	// Find the offending validator by eth address
	val, found := k.GetValidatorByEthAddress(ctx, hyperionId, ethAddress)
	if !found {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(types.ErrInvalid, fmt.Sprintf("Did not find validator for eth address %s", ethAddress))
	}

	// Slash the offending validator
	cons, err := val.GetConsAddr()
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(err, "Could not get consensus key address for validator")
	}

	counterpartyChainParams := k.GetCounterpartyChainParams(ctx)[hyperionId]
	_, err = k.StakingKeeper.Slash(ctx, cons, ctx.BlockHeight(), val.ConsensusPower(k.StakingKeeper.PowerReduction(ctx)), counterpartyChainParams.SlashFractionBadEthSignature)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		return errors.Wrap(err, "Could not slash validator")
	}

	return nil
}

// SetPastEthSignatureCheckpoint puts the checkpoint of a valset, batch, or logic call into a set
// in order to prove later that it existed at one point.
func (k *Keeper) SetPastEthSignatureCheckpoint(ctx sdk.Context, hyperionId uint64, checkpoint common.Hash) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetPastEthSignatureCheckpointKey(hyperionId, checkpoint), []byte{0x1})
}

// GetPastEthSignatureCheckpoint tells you whether a given checkpoint has ever existed
func (k *Keeper) GetPastEthSignatureCheckpoint(ctx sdk.Context, hyperionId uint64, checkpoint common.Hash) (found bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	if bytes.Equal(store.Get(types.GetPastEthSignatureCheckpointKey(hyperionId, checkpoint)), []byte{0x1}) {
		return true
	} else {
		return false
	}
}
