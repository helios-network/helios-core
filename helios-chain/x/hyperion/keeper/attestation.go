package keeper

import (
	"fmt"
	"strconv"

	// "cosmossdk.io/errors"
	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	"github.com/Helios-Chain-Labs/metrics"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"helios-core/helios-chain/x/hyperion/types"
)

func (k *Keeper) Attest(ctx sdk.Context, claim types.EthereumClaim, anyClaim *codectypes.Any) (*types.Attestation, error) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	// defer doneFn()

	valAddr, found := k.GetOrchestratorValidatorByHyperionID(ctx, claim.GetClaimer(), claim.GetHyperionId())
	if !found {
		metrics.ReportFuncError(k.svcTags)
		panic("Could not find ValAddr for delegate key, should be checked by now")
	}

	// Check that the nonce of this event is exactly one higher than the last nonce stored by this validator.
	// We check the event nonce in processAttestation as well,
	// but checking it here gives individual eth signers a chance to retry,
	// and prevents validators from submitting two claims with the same nonce
	// lastEvent := k.GetLastEventByValidator(ctx, valAddr)
	lastEvent := k.GetLastEventByValidatorByHyperionID(ctx, valAddr, claim.GetHyperionId())

	if lastEvent.EthereumEventNonce == 0 && lastEvent.EthereumEventHeight == 0 {
		// if hyperion happens to query too early without a bonded validator even existing setup the base event
		// lowestObservedNonce := k.GetLastObservedEventNonce(ctx)
		lowestObservedNonce := k.GetLastObservedEventNonceForHyperionID(ctx, claim.GetHyperionId())
		// blockHeight := k.GetLastObservedEthereumBlockHeight(ctx).EthereumBlockHeight
		blockHeight := k.GetLastObservedEthereumBlockHeightForHyperionID(ctx, claim.GetHyperionId()).EthereumBlockHeight

		// k.setLastEventByValidator(
		// 	ctx,
		// 	valAddr,
		// 	lowestObservedNonce,
		// 	blockHeight,
		// )
		k.setLastEventByValidatorByHyperionID(
			ctx,
			valAddr,
			lowestObservedNonce,
			blockHeight,
			claim.GetHyperionId(),
		)
		// lastEvent = k.GetLastEventByValidator(ctx, valAddr)
		lastEvent = k.GetLastEventByValidatorByHyperionID(ctx, valAddr, claim.GetHyperionId())
	}

	if claim.GetEventNonce() != lastEvent.EthereumEventNonce+1 {
		metrics.ReportFuncError(k.svcTags)
		k.Logger(ctx).Info(fmt.Sprintf("New Attest Nonce of Hyperion Orchestrator %s Nonce=%d , claim Attested Nonce=%d", valAddr.String(), lastEvent.EthereumEventNonce, claim.GetEventNonce()))
		return nil, errors.Wrap(types.ErrNonContiguousEventNonce, fmt.Sprintf("ErrNonContiguousEventNonce %d != %d for Validator=%s", claim.GetEventNonce(), lastEvent.EthereumEventNonce+1, valAddr.String()))
	}

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

	// Add the validator's vote to this attestation
	att.Votes = append(att.Votes, valAddr.String())

	k.SetAttestation(ctx, claim.GetHyperionId(), claim.GetEventNonce(), claim.ClaimHash(), att)
	k.setLastEventByValidator(ctx, valAddr, claim.GetEventNonce(), claim.GetBlockHeight(), claim.GetHyperionId())

	lastEvent = k.GetLastEventByValidator(ctx, valAddr, claim.GetHyperionId())
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

func getRequiredPower(totalPower math.Int) math.Int {
	return totalPower.Mul(math.NewInt(66)).Quo(math.NewInt(100))
}

// TryAttestation checks if an attestation has enough votes to be applied to the consensus state
// and has not already been marked Observed, then calls processAttestation to actually apply it to the state,
// and then marks it Observed and emits an event.
func (k *Keeper) TryAttestation(ctx sdk.Context, att *types.Attestation) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	fmt.Println("TryAttestation=======================")

	claim, err := k.UnpackAttestationClaim(att)
	if err != nil {
		metrics.ReportFuncError(k.svcTags)
		panic("could not cast to claim")
	}
	// If the attestation has not yet been Observed, sum up the votes and see if it is ready to apply to the state.
	// This conditional stops the attestation from accidentally being applied twice.
	if !att.Observed {
		fmt.Println("TryAttestation=======================")
		// Sum the current powers of all validators who have voted and see if it passes the current threshold
		totalPower, err := k.StakingKeeper.GetLastTotalPower(ctx)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			panic("can't get total power: " + err.Error())
		}
		requiredPower := getRequiredPower(totalPower)
		attestationPower := math.ZeroInt()
		for _, validator := range att.Votes {
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
				// lastEventNonce := k.GetLastObservedEventNonce(ctx)
				lastEventNonce := k.GetLastObservedEventNonceForHyperionID(ctx, claim.GetHyperionId())
				// this check is performed at the next level up so this should never panic
				// outside of programmer error.
				if claim.GetEventNonce() != lastEventNonce+1 {
					metrics.ReportFuncError(k.svcTags)
					panic("attempting to apply events to state out of order")
				}
				// k.setLastObservedEventNonce(ctx, claim.GetEventNonce())
				// k.SetLastObservedEthereumBlockHeight(ctx, claim.GetBlockHeight())
				k.setLastObservedEventNonceForHyperionID(ctx, claim.GetEventNonce(), claim.GetHyperionId())
				k.SetLastObservedEthereumBlockHeightForHyperionID(ctx, claim.GetBlockHeight(), claim.GetHyperionId())

				att.Observed = true
				k.SetAttestation(ctx, claim.GetHyperionId(), claim.GetEventNonce(), claim.ClaimHash(), att)

				k.processAttestation(ctx, claim)
				k.emitObservedEvent(ctx, att, claim)

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
func (k *Keeper) processAttestation(ctx sdk.Context, claim types.EthereumClaim) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()
	fmt.Println("processAttestation=======================")

	// then execute in a new Tx so that we can store state on failure
	xCtx, commit := ctx.CacheContext()
	if err := k.AttestationHandler.Handle(xCtx, claim); err != nil { // execute with a transient storage
		// If the attestation fails, something has gone wrong and we can't recover it. Log and move on
		// The attestation will still be marked "Observed", and validators can still be slashed for not
		// having voted for it.
		k.Logger(ctx).Error("attestation failed",
			"claim_type", claim.GetType().String(),
			"id", hexutil.Encode(types.GetAttestationKeyWithHash(claim.GetEventNonce(), claim.ClaimHash())),
			"nonce", claim.GetEventNonce(),
			"err", err.Error(),
		)
		fmt.Println("processAttestation - err: ", err)
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
				k.Logger(ctx).Info("no claim data sdk.Msg to handle")
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
	// Tenter de caster le message en *MsgSendToChain
	if msgCasted, ok := msg.(*types.MsgSendToChain); ok {
		// Le casting a réussi, vous pouvez maintenant utiliser msgCasted
		// Traitez le message ici
		k.Logger(ctx).Info("Received MsgSendToChain", "data", msgCasted)

		msgSrv := NewMsgServerImpl(*k)
		_, err := msgSrv.SendToChain(ctx, msgCasted)
		if err != nil {
			return err
		}
		return nil
	}

	// Si le casting échoue, vous pouvez gérer l'erreur
	k.Logger(ctx).Error("Failed to cast msg to MsgSendToChain")
	return types.ErrUnknown // Ou une autre erreur appropriée
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

func (k Keeper) GetAttestationMapping(ctx sdk.Context, hyperionId uint64) map[uint64][]*types.Attestation {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.OracleAttestationKey, sdk.Uint64ToBigEndian(hyperionId)...))

	var crons []*types.Attestation
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var cron types.Attestation
		k.cdc.MustUnmarshal(iterator.Value(), &cron)
		crons = append(crons, &cron)
	}

	k.Logger(ctx).Info("Attestations list", "size", len(crons))

	out := make(map[uint64][]*types.Attestation)

	for _, att := range crons {
		claim, err := k.UnpackAttestationClaim(att)
		if err != nil {
			metrics.ReportFuncError(k.svcTags)
			panic("couldn't UnpackAttestationClaim")
		}

		eventNonce := claim.GetEventNonce()
		out[eventNonce] = append(out[eventNonce], att)

		k.Logger(ctx).Info("Adding attestation to map", "eventNonce", eventNonce, "currentSize", len(out[eventNonce]))
	}

	k.Logger(ctx).Info("Final attestation mapping size", "size", len(out))

	return out
}

// GetAttestationMapping returns a mapping of eventnonce -> attestations at that nonce
// func (k *Keeper) GetAttestationMapping(ctx sdk.Context, hyperionId uint64) (out map[uint64][]*types.Attestation) {
// 	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
// 	// defer doneFn()

// 	out = make(map[uint64][]*types.Attestation)
// 	k.IterateAttestations(ctx, hyperionId, func(_ []byte, attestation *types.Attestation) (stop bool) {
// 		claim, err := k.UnpackAttestationClaim(attestation)
// 		if err != nil {
// 			metrics.ReportFuncError(k.svcTags)
// 			panic("couldn't UnpackAttestationClaim")
// 		}

// 		eventNonce := claim.GetEventNonce()
// 		out[eventNonce] = append(out[eventNonce], attestation)

// 		k.Logger(ctx).Info("Adding attestation to map", "eventNonce", eventNonce, "currentSize", len(out[eventNonce]))

// 		return false
// 	})

// 	k.Logger(ctx).Info("Final attestation mapping size", "size", len(out))

// 	return out
// }

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

		k.Logger(ctx).Info("Iterate Attestation", "att", attestation.HyperionId)

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
func (k *Keeper) GetLastObservedValset(ctx sdk.Context, hyperionID uint64) *types.Valset {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLastObservedValsetKey(hyperionID))

	if len(bytes) == 0 {
		return nil
	}

	valset := types.Valset{}
	k.cdc.MustUnmarshal(bytes, &valset)

	return &valset
}

// SetLastObservedValset updates the last observed validator set in the store
func (k *Keeper) SetLastObservedValset(ctx sdk.Context, valset types.Valset, hyperionID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetLastObservedValsetKey(hyperionID)
	store.Set(key, k.cdc.MustMarshal(&valset))
}

// GetLastObservedEventNonceForHyperionID returns the latest observed event nonce
func (k *Keeper) GetLastObservedEventNonceForHyperionID(ctx sdk.Context, hyperionID uint64) uint64 {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLastObservedEventNonceForHyperionIDKey(hyperionID))

	if len(bytes) == 0 {
		return 0
	}
	return types.UInt64FromBytes(bytes)
}

// SetLastObservedEthereumBlockHeight sets the block height in the store.
func (k *Keeper) SetLastObservedEthereumBlockHeight(ctx sdk.Context, hyperionId uint64, ethereumHeight uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.LastObservedEthereumBlockHeightKey, sdk.Uint64ToBigEndian(hyperionId)...))
	height := types.LastObservedEthereumBlockHeight{
		EthereumBlockHeight: ethereumHeight,
		CosmosBlockHeight:   uint64(ctx.BlockHeight()),
	}

	store.Set(sdk.Uint64ToBigEndian(0), k.cdc.MustMarshal(&height))
}

// SetLastObservedEthereumBlockHeightForHyperionID sets the block height in the store.
func (k *Keeper) SetLastObservedEthereumBlockHeightForHyperionID(ctx sdk.Context, ethereumHeight uint64, hyperionID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	height := types.LastObservedEthereumBlockHeight{
		ChainId:             strconv.FormatUint(hyperionID, 10),
		EthereumBlockHeight: ethereumHeight,
		CosmosBlockHeight:   uint64(ctx.BlockHeight()),
	}

	store.Set(types.GetLastObservedEthereumBlockHeightForHyperionIDKey(hyperionID), k.cdc.MustMarshal(&height))
}

// GetLastObservedEthereumBlockHeightForHyperionID height gets the block height to of the last observed attestation from
// the store
func (k *Keeper) GetLastObservedEthereumBlockHeightForHyperionID(ctx sdk.Context, hyperionID uint64) types.LastObservedEthereumBlockHeight {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	bytes := store.Get(types.GetLastObservedEthereumBlockHeightForHyperionIDKey(hyperionID))

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


// setLastObservedEventNonce sets the latest observed event nonce
func (k *Keeper) setLastObservedEventNonce(ctx sdk.Context, nonce uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	store.Set(types.LastObservedEventNonceKey, types.UInt64Bytes(nonce))
}

func (k *Keeper) setLastObservedEventNonceForHyperionID(ctx sdk.Context, nonce uint64, hyperionID uint64) {
	fmt.Println("setLastObservedEventNonceForHyperionID - nonce: ", nonce, "hyperionID: ", hyperionID)

	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	key := types.GetLastObservedEventNonceForHyperionIDKey(hyperionID)
	store.Set(key, types.UInt64Bytes(nonce))
}

func (k *Keeper) setLastEventByValidator(ctx sdk.Context, validator sdk.ValAddress, nonce, blockHeight uint64, hyperionID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	lastClaimEvent := types.LastClaimEvent{
		EthereumEventNonce:  nonce,
		EthereumEventHeight: blockHeight,
	}

	store.Set(types.GetLastEventByValidatorKey(validator, hyperionID), k.cdc.MustMarshal(&lastClaimEvent))

}

func (k *Keeper) setLastEventByValidatorByHyperionID(ctx sdk.Context, validator sdk.ValAddress, nonce, blockHeight uint64, hyperionID uint64) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	store := ctx.KVStore(k.storeKey)
	lastClaimEvent := types.LastClaimEvent{
		EthereumEventNonce:  nonce,
		EthereumEventHeight: blockHeight,
	}

	store.Set(types.GetLastEventByValidatorKeyByHyperionID(validator, hyperionID), k.cdc.MustMarshal(&lastClaimEvent))

}

// GetLastEventByValidator returns the latest event for a given validator
func (k *Keeper) GetLastEventByValidator(ctx sdk.Context, validator sdk.ValAddress, hyperionID uint64) types.LastClaimEvent {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	rawEvent := ctx.KVStore(k.storeKey).Get(types.GetLastEventByValidatorKey(validator, hyperionID))
	if len(rawEvent) == 0 {
		return types.LastClaimEvent{}
	}

	// Unmarshall last observed event by validator
	var lastEvent types.LastClaimEvent
	k.cdc.MustUnmarshal(rawEvent, &lastEvent)

	return lastEvent
}

// GetLastEventByValidator returns the latest event for a given validator
func (k *Keeper) GetLastEventByValidatorByHyperionID(ctx sdk.Context, validator sdk.ValAddress, hyperionID uint64) types.LastClaimEvent {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	// rawEvent := ctx.KVStore(k.storeKey).Get(types.GetLastEventByValidatorKey(validator))
	rawEvent := ctx.KVStore(k.storeKey).Get(types.GetLastEventByValidatorKeyByHyperionID(validator, hyperionID))
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
