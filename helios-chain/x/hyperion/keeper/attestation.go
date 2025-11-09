package keeper

import (
	"fmt"
	"strings"

	// "cosmossdk.io/errors"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/Helios-Chain-Labs/metrics"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"

	testnet "helios-core/helios-chain/testnet"
	"helios-core/helios-chain/x/hyperion/types"
)

func (k *Keeper) Attest(ctx sdk.Context, claim types.EthereumClaim, anyClaim *codectypes.Any) (*types.Attestation, error) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	// defer doneFn()

	valAddr, found := k.GetOrchestratorValidator(ctx, claim.GetHyperionId(), claim.GetClaimer())
	if !found {
		metrics.ReportFuncError(k.svcTags)
		panic("Could not find ValAddr for delegate key, should be checked by now")
	}

	// Check that the nonce of this event is exactly one higher than the last nonce stored by this validator.
	// We check the event nonce in processAttestation as well,
	// but checking it here gives individual eth signers a chance to retry,
	// and prevents validators from submitting two claims with the same nonce
	lastEvent := k.GetLastEventByValidatorAndHyperionId(ctx, claim.GetHyperionId(), valAddr)
	lastObservedBlockHeight := k.GetLastObservedEthereumBlockHeight(ctx, claim.GetHyperionId()).EthereumBlockHeight

	if lastEvent.EthereumEventNonce == 0 && lastEvent.EthereumEventHeight == 0 {
		// if hyperion happens to query too early without a bonded validator even existing setup the base event
		lowestObservedNonce := k.GetLastObservedEventNonce(ctx, claim.GetHyperionId())

		k.setLastEventByValidatorAndHyperionId(
			ctx,
			claim.GetHyperionId(),
			valAddr,
			lowestObservedNonce,
			lastObservedBlockHeight,
		)

		lastEvent = k.GetLastEventByValidatorAndHyperionId(ctx, claim.GetHyperionId(), valAddr)
	}

	// if claim.GetBlockHeight() < lastObservedBlockHeight {
	// 	// test
	// 	// 3 april 2025
	// 	// maybe exclude claim.GetBlockHeight() < GetLastObservedEthereumBlockHeight for increase security
	// 	// not sure if we have some number of tx detected can stuck some hyperions
	// 	metrics.ReportFuncError(k.svcTags)
	// 	k.Logger(ctx).Info(fmt.Sprintf("New Attest Nonce of Hyperion Orchestrator %s Nonce=%d , claim Attested Nonce=%d , ethHeight=%d", valAddr.String(), lastEvent.EthereumEventNonce, claim.GetEventNonce(), lastObservedBlockHeight))
	// 	return nil, errors.Wrap(types.ErrNonContiguousEthEventBlockHeight, fmt.Sprintf("ErrNonContiguousEthEventBlockHeight %d < %d for Validator=%s", claim.GetBlockHeight(), lastObservedBlockHeight, valAddr.String()))
	// }

	// if claim.GetEventNonce() < lastEvent.EthereumEventNonce+1 { // accept superior and same
	// 	metrics.ReportFuncError(k.svcTags)
	// 	k.Logger(ctx).Info(fmt.Sprintf("New Attest Nonce of Hyperion Orchestrator %s Nonce=%d , claim Attested Nonce=%d", valAddr.String(), lastEvent.EthereumEventNonce, claim.GetEventNonce()))
	// 	return nil, errors.Wrap(types.ErrNonContiguousEventNonce, fmt.Sprintf("ErrNonContiguousEventNonce %d != %d for Validator=%s", claim.GetEventNonce(), lastEvent.EthereumEventNonce+1, valAddr.String()))
	// }

	// Tries to get an attestation with the same eventNonce and claim as the claim that was submitted.
	att := k.GetAttestation(ctx, claim.GetHyperionId(), claim.GetEventNonce(), claim.ClaimHash())
	isNewAttestation := false

	// If it does not exist, create a new one.
	if att == nil {
		att = &types.Attestation{
			Observed:   false,
			Height:     uint64(ctx.BlockHeight()),
			Claim:      anyClaim,
			HyperionId: claim.GetHyperionId(),
		}
		isNewAttestation = true
	}

	if att.Observed {
		return nil, errors.Wrap(types.ErrAttestationAlreadyObserved, "Attestation already Observed")
	}

	if k.NonceAlreadyObserved(ctx, claim.GetHyperionId(), claim.GetEventNonce()) {
		return nil, errors.Wrap(types.ErrAttestationAlreadyObserved, "Attestation already Observed")
	}

	if ctx.BlockHeight() > testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 && att.ContainsVote(valAddr.String()) {
		return nil, errors.Wrap(types.ErrAttestationAlreadyVoted, "Attestation already voted")
	}

	// Add the validator's vote to this attestation
	att.Votes = append(att.Votes, valAddr.String()+":"+fmt.Sprintf("%X", tmhash.Sum(ctx.TxBytes())))
	// Add the rpc used to this attestation
	att.RpcsUsed = append(att.RpcsUsed, claim.GetRpcUsed())

	k.SetAttestation(ctx, claim.GetHyperionId(), claim.GetEventNonce(), claim.ClaimHash(), att)

	k.setLastEventByValidatorAndHyperionId(ctx, claim.GetHyperionId(), valAddr, claim.GetEventNonce(), claim.GetBlockHeight())

	lastEvent = k.GetLastEventByValidatorAndHyperionId(ctx, claim.GetHyperionId(), valAddr)
	k.Logger(ctx).Info(fmt.Sprintf("Attest Update Nonce of Hyperion Orchestrator %s newNonce=%d , claim Attested Nonce=%d", valAddr.String(), lastEvent.EthereumEventNonce, claim.GetEventNonce()))

	attestationId := types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash())

	if isNewAttestation {
		emitNewClaimEvent(ctx, claim, attestationId)
	}

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventAttestationVote{
		EventNonce:    claim.GetEventNonce(),
		AttestationId: attestationId,
		Voter:         valAddr.String(),
	})

	return att, nil
}

func emitNewClaimEvent(ctx sdk.Context, claim types.EthereumClaim, attestationId []byte) {
	switch claim := claim.(type) {
	case *types.MsgDepositClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventDepositClaim{
			HyperionId:          claim.HyperionId,
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			EthereumSender:      claim.GetEthereumSender(),
			CosmosReceiver:      claim.GetCosmosReceiver(),
			TokenContract:       claim.GetTokenContract(),
			Amount:              claim.Amount,
			AttestationId:       attestationId,
			OrchestratorAddress: claim.GetOrchestrator(),
			Data:                claim.Data,
		})
	case *types.MsgWithdrawClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventWithdrawClaim{
			HyperionId:          claim.HyperionId,
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			BatchNonce:          claim.GetBatchNonce(),
			TokenContract:       claim.GetTokenContract(),
			OrchestratorAddress: claim.GetOrchestrator(),
			AttestationId:       attestationId,
		})
	case *types.MsgERC20DeployedClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventERC20DeployedClaim{
			HyperionId:          claim.HyperionId,
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			CosmosDenom:         claim.GetCosmosDenom(),
			TokenContract:       claim.GetTokenContract(),
			Name:                claim.GetName(),
			Symbol:              claim.GetSymbol(),
			Decimals:            claim.GetDecimals(),
			OrchestratorAddress: claim.GetOrchestrator(),
			AttestationId:       attestationId,
		})
	case *types.MsgValsetUpdatedClaim:
		// nolint:errcheck //ignored on purpose
		ctx.EventManager().EmitTypedEvent(&types.EventValsetUpdateClaim{
			HyperionId:          claim.HyperionId,
			EventNonce:          claim.GetEventNonce(),
			EventHeight:         claim.GetBlockHeight(),
			ValsetNonce:         claim.GetValsetNonce(),
			ValsetMembers:       claim.GetMembers(),
			RewardAmount:        claim.RewardAmount,
			RewardToken:         claim.GetRewardToken(),
			OrchestratorAddress: claim.GetOrchestrator(),
			AttestationId:       attestationId,
		})
	}
}

func (k *Keeper) GetRequiredPower(totalPower math.Int, powerPercentage uint64) math.Int {
	return totalPower.Mul(math.NewInt(int64(powerPercentage))).Quo(math.NewInt(100))
}

// TryAttestation checks if an attestation has enough votes to be applied to the consensus state
// and has not already been marked Observed, then calls processAttestation to actually apply it to the state,
// and then marks it Observed and emits an event.
func (k *Keeper) TryAttestation(ctx sdk.Context, att *types.Attestation, force bool) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	claim, err := k.UnpackAttestationClaim(att)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic("could not cast to claim")
	}
	// If the attestation has not yet been Observed, sum up the votes and see if it is ready to apply to the state.
	// This conditional stops the attestation from accidentally being applied twice.
	if !att.Observed {
		// Sum the current powers of all validators who have voted and see if it passes the current threshold
		totalPower := k.GetCurrentValsetTotalPower(ctx, claim.GetHyperionId())
		requiredPower := k.GetRequiredPower(totalPower, 66)
		attestationPower := math.ZeroInt()
		for _, validatorAndTxProof := range att.Votes {
			validatorAndTxProofSplitted := strings.Split(validatorAndTxProof, ":")
			validator := validatorAndTxProofSplitted[0]

			val, err := sdk.ValAddressFromBech32(validator)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				panic(err)
			}
			validatorPower, err := k.StakingKeeper.GetLastValidatorPower(ctx, val)
			if err != nil {
				metrics.ReportFuncError(k.svcTags)
				panic("can't get total power: " + err.Error())
			}
			// Add it to the attestation power's sum
			attestationPower = attestationPower.Add(math.NewInt(validatorPower))
			// If the power of all the validators that have voted on the attestation is higher or equal to the threshold,
			// process the attestation, set Observed to true, and break
			if attestationPower.GTE(requiredPower) {
				lastEventNonce := k.GetLastObservedEventNonce(ctx, claim.GetHyperionId())
				// // this check is performed at the next level up so this should never panic
				// // outside of programmer error.
				// if claim.GetEventNonce() != lastEventNonce+1 {
				// 	metrics.ReportFuncError(k.svcTags)
				// 	panic("attempting to apply events to state out of order")
				// }
				if lastEventNonce < claim.GetEventNonce() {
					k.setLastObservedEventNonce(ctx, claim.GetHyperionId(), claim.GetEventNonce())
					k.SetNewLastObservedEthereumBlockHeight(ctx, claim.GetHyperionId(), claim.GetBlockHeight())
				}
				k.StoreNonceObserved(ctx, claim.GetHyperionId(), claim.GetEventNonce())
				att.Observed = true
				k.SetAttestation(ctx, claim.GetHyperionId(), claim.GetEventNonce(), claim.ClaimHash(), att)

				k.processAttestation(ctx, claim, att)
				k.emitObservedEvent(ctx, att, claim)

				// update the rpc used
				if strings.Contains(claim.GetRpcUsed(), "https://") {
					k.UpdateRpcUsed(ctx, claim.GetHyperionId(), claim.GetRpcUsed(), claim.GetBlockHeight())
				}

				// handle the case where user sends arbitrary data in the MsgDepositClaim
				k.ProcessClaimData(ctx, claim)
				break
			}
		}
	} else {
		// We panic here because this should never happen
		metrics.ReportFuncError(k.svcTags)
		// panic("attempting to process observed attestation")
	}
}

// processAttestation actually applies the attestation to the consensus state
func (k *Keeper) processAttestation(ctx sdk.Context, claim types.EthereumClaim, att *types.Attestation) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// then execute in a new Tx so that we can store state on failure
	xCtx, commit := ctx.CacheContext()
	if err := k.AttestationHandler.Handle(xCtx, claim, att); err != nil { // execute with a transient storage
		// If the attestation fails, something has gone wrong and we can't recover it. Log and move on
		// The attestation will still be marked "Observed", and validators can still be slashed for not
		// having voted for it.
		k.Logger(ctx).Error("attestation failed",
			"claim_type", claim.GetType().String(),
			"id", hexutil.Encode(types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash())),
			"nonce", claim.GetEventNonce(),
			"err", err.Error(),
		)
	} else {
		commit() // persist transient storage
	}
}

// emitObservedEvent emits an event with information about an attestation that has been applied to
// consensus state.
func (k *Keeper) emitObservedEvent(ctx sdk.Context, _ *types.Attestation, claim types.EthereumClaim) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// nolint:errcheck //ignored on purpose
	ctx.EventManager().EmitTypedEvent(&types.EventAttestationObserved{
		AttestationType: claim.GetType(),
		BridgeContract:  k.GetBridgeContractAddress(ctx)[claim.GetHyperionId()].Hex(),
		BridgeChainId:   k.GetBridgeChainID(ctx)[claim.GetHyperionId()],
		AttestationId:   types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash()),
		Nonce:           claim.GetEventNonce(),
	})
}

func (k *Keeper) ProcessClaimData(ctx sdk.Context, claim types.EthereumClaim) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	defer func() {
		if r := recover(); r != nil {
			k.Logger(ctx).Error("Panic recovered inside ProcessClaimData", "panic", r)
			return
		}
	}()

	switch claim := claim.(type) {
	case *types.MsgDepositClaim:
		// Handle arbitrary data in deposit claim
		if claim.Data != "" {

			_, msg, _ := k.parseClaimData(ctx, claim.Data)

			// Check if the claim data is a valid sdk.Msg. If not, ignore the data
			if msg == nil {
				k.Logger(ctx).Debug("no claim data sdk.Msg to handle")
				return
			}

			// then execute sdk.msg in a new cache ctx so that we can avoid state changes on failure
			xCtx, commit := ctx.CacheContext()
			xCtx = xCtx.WithValue(baseapp.DoNotFailFastSendContextKey, nil) // enable fail fast during msg execution

			// Process the claim data msg
			if err := k.HandleMsg(xCtx, *msg); err != nil {
				k.Logger(ctx).Error("attestation HandleMsg Failed",
					"claim_type", claim.GetType().String(),
					"id", hexutil.Encode(types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash())),
					"nonce", claim.GetEventNonce(),
					"data", claim.Data,
					"msg", msg,
					"err", err.Error(),
				)
			} else {
				commit()
			}
		}
	}
}

func (k *Keeper) HandleMsg(ctx sdk.Context, msg sdk.Msg) error {
	// TODO: creer un message qui emglobe tous les messages potentiels avec r,v,s et content pour verifier la signature.
	// Tenter de caster le message en *MsgSendToChain TODO in the future
	// if msgCasted, ok := msg.(*types.MsgSendToChain); ok {
	// 	// Le casting a réussi, vous pouvez maintenant utiliser msgCasted
	// 	// Traitez le message ici
	// 	k.Logger(ctx).Info("Received MsgSendToChain", "data", msgCasted)

	// 	msgSrv := NewMsgServerImpl(*k)
	// 	_, err := msgSrv.SendToChain(ctx, msgCasted)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return nil
	// }

	// Si le casting échoue, vous pouvez gérer l'erreur
	// k.Logger(ctx).Error("Failed to cast msg to MsgSendToChain")
	return nil // Ou une autre erreur appropriée
}

// SetAttestation sets the attestation in the store
func (k *Keeper) SetAttestation(ctx sdk.Context, hyperionId uint64, eventNonce uint64, claimHash []byte, att *types.Attestation) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	// defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))
	aKey := types.GetAttestationKeyWithHash(eventNonce, claimHash)
	store.Set(aKey, k.cdc.MustMarshal(att))
}

// GetAttestation return an attestation given a nonce
func (k *Keeper) GetAttestation(ctx sdk.Context, hyperionId uint64, eventNonce uint64, claimHash []byte) *types.Attestation {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	// defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))
	aKey := types.GetAttestationKeyWithHash(eventNonce, claimHash)
	bz := store.Get(aKey)
	if len(bz) == 0 {
		return nil
	}

	var att types.Attestation
	k.cdc.MustUnmarshal(bz, &att)

	return &att
}

// DeleteAttestation deletes an attestation given an event nonce and claim
func (k *Keeper) DeleteAttestation(ctx sdk.Context, hyperionId uint64, att *types.Attestation) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	// defer doneFn()

	claim, err := k.UnpackAttestationClaim(att)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic("Bad Attestation in DeleteAttestation")
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))
	store.Delete(types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash()))
}

func (k *Keeper) CleanAttestations(ctx sdk.Context, hyperionId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}

func (k Keeper) GetAttestationMapping(ctx sdk.Context, hyperionId uint64) map[uint64][]*types.Attestation {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))

	var atts []*types.Attestation
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var cron types.Attestation
		k.cdc.MustUnmarshal(iterator.Value(), &cron)
		atts = append(atts, &cron)
	}

	out := make(map[uint64][]*types.Attestation)

	for _, att := range atts {
		claim, err := k.UnpackAttestationClaim(att)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			panic("couldn't UnpackAttestationClaim")
		}

		eventNonce := claim.GetEventNonce()
		out[eventNonce] = append(out[eventNonce], att)
	}

	return out
}

// IterateAttestations iterates through all attestations
func (k *Keeper) IterateAttestations(ctx sdk.Context, hyperionId uint64, cb func(k []byte, v *types.Attestation) (stop bool)) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	// defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		attestation := types.Attestation{}

		k.cdc.MustUnmarshal(iter.Value(), &attestation)

		// cb returns true to stop early
		if cb(iter.Key(), &attestation) {
			return
		}
	}
}

// GetLastObservedValset retrieves the last observed validator set from the store
// WARNING: This value is not an up to date validator set on Ethereum, it is a validator set
// that AT ONE POINT was the one in the Gravity bridge on Ethereum. If you assume that it's up
// to date you may break the bridge
func (k *Keeper) GetLastObservedValset(ctx sdk.Context, hyperionId uint64) *types.Valset {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedValsetKey, sdk.Uint64ToBigEndian(hyperionId)...))
	bytes := store.Get(sdk.Uint64ToBigEndian(0))

	if len(bytes) == 0 {
		return nil
	}

	valset := types.Valset{}
	k.cdc.MustUnmarshal(bytes, &valset)

	return &valset
}

// SetLastObservedValset updates the last observed validator set in the store
func (k *Keeper) SetLastObservedValset(ctx sdk.Context, hyperionId uint64, valset types.Valset) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedValsetKey, sdk.Uint64ToBigEndian(hyperionId)...))
	store.Set(sdk.Uint64ToBigEndian(0), k.cdc.MustMarshal(&valset))
}

// setLastObservedEventNonce sets the latest observed event nonce
func (k *Keeper) setLastObservedEventNonce(ctx sdk.Context, hyperionId uint64, nonce uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedEventNonceKey, sdk.Uint64ToBigEndian(hyperionId)...))
	store.Set(sdk.Uint64ToBigEndian(0), types.UInt64Bytes(nonce))
}

// GetLastObservedEventNonce returns the latest observed event nonce
func (k *Keeper) GetLastObservedEventNonce(ctx sdk.Context, hyperionId uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedEventNonceKey, sdk.Uint64ToBigEndian(hyperionId)...))
	bytes := store.Get(sdk.Uint64ToBigEndian(0))

	if len(bytes) == 0 {
		return 0
	}
	return types.UInt64FromBytes(bytes)
}

// GetLastObservedEthereumBlockHeight height gets the block height to of the last observed attestation from
// the store
func (k *Keeper) GetLastObservedEthereumBlockHeight(ctx sdk.Context, hyperionId uint64) types.LastObservedEthereumBlockHeight {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedEthereumBlockHeightKey, sdk.Uint64ToBigEndian(hyperionId)...))
	bytes := store.Get(sdk.Uint64ToBigEndian(0))

	if len(bytes) == 0 {
		return types.LastObservedEthereumBlockHeight{
			CosmosBlockHeight:   0,
			EthereumBlockHeight: 0,
		}
	}

	height := types.LastObservedEthereumBlockHeight{}
	k.cdc.MustUnmarshal(bytes, &height)

	return height
}

// SetLastObservedEthereumBlockHeight sets the block height in the store.
func (k *Keeper) SetNewLastObservedEthereumBlockHeight(ctx sdk.Context, hyperionId uint64, ethereumHeight uint64) {
	k.SetLastObservedEthereumBlockHeight(ctx, hyperionId, ethereumHeight, uint64(ctx.BlockHeight()))
}

func (k *Keeper) SetLastObservedEthereumBlockHeight(ctx sdk.Context, hyperionId uint64, ethereumHeight uint64, heliosBlockHeight uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedEthereumBlockHeightKey, sdk.Uint64ToBigEndian(hyperionId)...))
	height := types.LastObservedEthereumBlockHeight{
		EthereumBlockHeight: ethereumHeight,
		CosmosBlockHeight:   heliosBlockHeight,
	}

	store.Set(sdk.Uint64ToBigEndian(0), k.cdc.MustMarshal(&height))
}

func (k *Keeper) setLastEventByValidatorAndHyperionId(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress, nonce, blockHeight uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastEventByValidatorKey, sdk.Uint64ToBigEndian(hyperionId)...))
	lastClaimEvent := types.LastClaimEvent{
		EthereumEventNonce:  nonce,
		EthereumEventHeight: blockHeight,
	}

	store.Set(types.GetLastEventByValidatorKey(hyperionId, validator), k.cdc.MustMarshal(&lastClaimEvent))
}

func (k *Keeper) CleanLastEventByValidator(ctx sdk.Context, hyperionId uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastEventByValidatorKey, sdk.Uint64ToBigEndian(hyperionId)...))
	iter := store.Iterator(nil, nil)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		store.Delete(iter.Key())
	}
}

// GetLastEventByValidator returns the latest event for a given validator
func (k *Keeper) GetLastEventByValidatorAndHyperionId(ctx sdk.Context, hyperionId uint64, validator sdk.ValAddress) types.LastClaimEvent {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastEventByValidatorKey, sdk.Uint64ToBigEndian(hyperionId)...))

	rawEvent := store.Get(types.GetLastEventByValidatorKey(hyperionId, validator))
	if len(rawEvent) == 0 {
		return types.LastClaimEvent{}
	}

	// Unmarshall last observed event by validator
	var lastEvent types.LastClaimEvent
	k.cdc.MustUnmarshal(rawEvent, &lastEvent)

	return lastEvent
}

// func (k *Keeper) PruneAttestation7005(ctx sdk.Context, hyperionId uint64) {
// 	//	fetch the old key used to set attestation 7005
// 	var key7005 []byte
// 	k.IterateAttestations(ctx, hyperionId, func(key []byte, att *types.Attestation) (stop bool) {
// 		claim, err := k.UnpackAttestationClaim(att)
// 		if err != nil {
// 			return false
// 		}

// 		if claim.GetEventNonce() != 7005 {
// 			return false
// 		}

// 		key7005 = key

// 		return true
// 	})

// 	if key7005 == nil {
// 		return
// 	}

// 	// prune the store (DeleteAttestation won't work)
// 	ctx.KVStore(k.storeKey).Delete(key7005)
// }
