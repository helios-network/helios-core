package hyperion

import (
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/Helios-Chain-Labs/metrics"

	"helios-core/helios-chain/x/hyperion/keeper"
	"helios-core/helios-chain/x/hyperion/types"
)

type BlockHandler struct {
	k keeper.Keeper

	svcTags metrics.Tags
}

func NewBlockHandler(k keeper.Keeper) *BlockHandler {
	return &BlockHandler{
		k: k,

		svcTags: metrics.Tags{
			"svc": "hyperion_b",
		},
	}
}

// EndBlocker is called at the end of every block
func (h *BlockHandler) EndBlocker(ctx sdk.Context) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	params := h.k.GetParams(ctx)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		h.slashing(ctx, counterpartyChainParams)
	}
	h.attestationTally(ctx)
	h.cleanupTimedOutBatches(ctx)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		h.createValsets(ctx, counterpartyChainParams)
		h.pruneValsets(ctx, counterpartyChainParams)
	}
	h.pruneAttestations(ctx)
}

func (h *BlockHandler) createValsets(ctx sdk.Context, params *types.CounterpartyChainParams) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	// Auto ValsetRequest Creation.
	// WARNING: do not use k.GetLastObservedValset in this function, it *will* result in losing control of the bridge
	// 1. If there are no valset requests, create a new one.
	// 2. If there is at least one validator who started unbonding in current block. (we persist last unbonded block height in hooks.go)
	//      This will make sure the unbonding validator has to provide an attestation to a new Valset
	//	    that excludes him before he completely Unbonds.  Otherwise he will be slashed
	// 3. If power change between validators of CurrentValset and latest valset request is > 5%

	// get the last valsets to compare against
	latestValset := h.k.GetLatestValset(ctx)
	lastUnbondingHeight := h.k.GetLastUnbondingBlockHeight(ctx)

	if (latestValset == nil) || (lastUnbondingHeight == uint64(ctx.BlockHeight())) ||
		(types.BridgeValidators(h.k.GetCurrentValset(ctx, params.HyperionId).Members).PowerDiff(latestValset.Members) > 0.05) {
		// if the conditions are true, put in a new validator set request to be signed and submitted to Ethereum
		h.k.SetValsetRequest(ctx, params.HyperionId)
	}
}

// Iterate over all attestations currently being voted on in order of nonce
// and prune those that are older than the current nonce and no longer have any
// use. This could be combined with create attestation and save some computation
// but (A) pruning keeps the iteration small in the first place and (B) there is
// already enough nuance in the other handler that it's best not to complicate it further
func (h *BlockHandler) pruneAttestations(ctx sdk.Context) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	for _, counterParty := range h.k.GetParams(ctx).CounterpartyChainParams {
		hyperionId := counterParty.HyperionId

		attmap := h.k.GetAttestationMapping(ctx, hyperionId)

		h.k.Logger(ctx).Info("pruneAttestations", "attmap", len(attmap))

		// We make a slice with all the event nonces that are in the attestation mapping
		keys := make([]uint64, 0, len(attmap))
		for k := range attmap {
			keys = append(keys, k)
		}
		// Then we sort it
		sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })

		// lastObservedEventNonce := h.k.GetLastObservedEventNonce(ctx)
		// This iterates over all keys (event nonces) in the attestation mapping. Each value contains
		// a slice with one or more attestations at that event nonce. There can be multiple attestations
		// at one event nonce when validators disagree about what event happened at that nonce.
		for _, nonce := range keys {
			// This iterates over all attestations at a particular event nonce.
			// They are ordered by when the first attestation at the event nonce was received.
			// This order is not important.
			for _, att := range attmap[nonce] {
				// we delete all attestations earlier than the current event nonce
				// if nonce < lastObservedEventNonce {
				if att.Observed {
					h.k.Logger(ctx).Info("pruneAttestations", "pruning", att.HyperionId)
					h.k.DeleteAttestation(ctx, att.HyperionId, att)
				}
				// }
			}
		}

	}
}

func (h *BlockHandler) slashing(ctx sdk.Context, params *types.CounterpartyChainParams) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	// Slash validator for not confirming valset requests, batch requests and not attesting claims rightfully
	h.valsetSlashing(ctx, params)
	h.batchSlashing(ctx, params)

	// See https://github.com/Gravity-Bridge/Gravity-Bridge/blob/main/spec/slashing-spec.md#gravslash-05-failure-to-submit-eth-oracle-claims---intentionally-not-implemented
	// if params.ClaimSlashingEnabled {
	//	h.claimsSlashing(ctx, params)
	//}
}

// Iterate over all attestations currently being voted on in order of nonce and
// "Observe" those who have passed the threshold. Break the loop once we see
// an attestation that has not passed the threshold
func (h *BlockHandler) attestationTally(ctx sdk.Context) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	for _, counterParty := range h.k.GetParams(ctx).CounterpartyChainParams {
		hyperionId := counterParty.HyperionId
		attmap := h.k.GetAttestationMapping(ctx, hyperionId)
		// We make a slice with all the event nonces that are in the attestation mapping
		keys := make([]uint64, 0, len(attmap))
		fmt.Println("attmap", len(attmap))
		for k := range attmap {
			keys = append(keys, k)
		}
		// Then we sort it
		sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })

		// This iterates over all keys (event nonces) in the attestation mapping. Each value contains
		// a slice with one or more attestations at that event nonce. There can be multiple attestations
		// at one event nonce when validators disagree about what event happened at that nonce.
		for _, nonce := range keys {
			fmt.Println("nonce", nonce)
			// This iterates over all attestations at a particular event nonce.
			// They are ordered by when the first attestation at the event nonce was received.
			// This order is not important.
			for _, attestation := range attmap[nonce] {
				// We check if the event nonce is exactly 1 higher than the last attestation that was
				// observed. If it is not, we just move on to the next nonce. This will skip over all
				// attestations that have already been observed.
				//
				// Once we hit an event nonce that is one higher than the last observed event, we stop
				// skipping over this conditional and start calling tryAttestation (counting votes)
				// Once an attestation at a given event nonce has enough votes and becomes observed,
				// every other attestation at that nonce will be skipped, since the lastObservedEventNonce
				// will be incremented.
				//
				// Then we go to the next event nonce in the attestation mapping, if there is one. This
				// nonce will once again be one higher than the lastObservedEventNonce.
				// If there is an attestation at this event nonce which has enough votes to be observed,
				// we skip the other attestations and move on to the next nonce again.
				// If no attestation becomes observed, when we get to the next nonce, every attestation in
				// it will be skipped. The same will happen for every nonce after that.
				fmt.Println("h.k.GetLastObservedEventNonce(ctx)", h.k.GetLastObservedEventNonce(ctx, attestation.HyperionId))
				// if nonce == h.k.GetLastObservedEventNonce(ctx)+1 {
				h.k.TryAttestation(ctx, attestation)
				// }
			}
		}
	}
}

// cleanupTimedOutBatches deletes batches that have passed their expiration on Ethereum
// keep in mind several things when modifying this function
// A) unlike nonces timeouts are not monotonically increasing, meaning batch 5 can have a later timeout than batch 6
//
//	this means that we MUST only cleanup a single batch at a time
//
// B) it is possible for ethereumHeight to be zero if no events have ever occurred, make sure your code accounts for this
// C) When we compute the timeout we do our best to estimate the Ethereum block height at that very second. But what we work with
//
//	here is the Ethereum block height at the time of the last Deposit or Withdraw to be observed. It's very important we do not
//	project, if we do a slowdown on ethereum could cause a double spend. Instead timeouts will *only* occur after the timeout period
//	AND any deposit or withdraw has occurred to update the Ethereum block height.
func (h *BlockHandler) cleanupTimedOutBatches(ctx sdk.Context) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	for _, counterParty := range h.k.GetParams(ctx).CounterpartyChainParams {
		hyperionId := counterParty.HyperionId

		ethereumHeight := h.k.GetLastObservedEthereumBlockHeight(ctx, hyperionId).EthereumBlockHeight
		batches := h.k.GetOutgoingTxBatches(ctx)

		for _, batch := range batches {
			if batch.BatchTimeout < ethereumHeight {
				err := h.k.CancelOutgoingTXBatch(ctx, common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId)
				if err != nil {
					ctx.Logger().Error("failed to cancel outgoing tx batch", "error", err, "block", batch.Block, "batch_nonce", batch.BatchNonce)
				}
			}
		}
	}
}

func (h *BlockHandler) valsetSlashing(ctx sdk.Context, params *types.CounterpartyChainParams) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	maxHeight := uint64(0)

	// don't slash in the beginning before there aren't even SignedValsetsWindow blocks yet
	if uint64(ctx.BlockHeight()) > params.SignedValsetsWindow {
		maxHeight = uint64(ctx.BlockHeight()) - params.SignedValsetsWindow
	} else {
		// we can't slash anyone if SignedValsetWindow blocks have not passed
		return
	}

	unslashedValsets := h.k.GetUnslashedValsets(ctx, maxHeight)

	// unslashedValsets are sorted by nonce in ASC order
	for _, vs := range unslashedValsets {
		confirms := h.k.GetValsetConfirms(ctx, vs.Nonce)

		// SLASH BONDED VALIDATORS who didn't attest valset request
		currentBondedSet, _ := h.k.StakingKeeper.GetBondedValidatorsByPower(ctx)

		for i := range currentBondedSet {
			consAddr, _ := currentBondedSet[i].GetConsAddr()
			valSigningInfo, err := h.k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)

			exist := err == nil
			//  Slash validator ONLY if he joined after valset is created
			if exist && valSigningInfo.StartHeight < int64(vs.Height) {
				// Check if validator has confirmed valset or not
				found := false
				for _, conf := range confirms {
					valAddr, _ := sdk.ValAddressFromBech32(currentBondedSet[i].GetOperator())
					ethAddress, exists := h.k.GetEthAddressByValidator(ctx, valAddr)
					// This may have an issue if the validator changes their eth address
					// TODO this presents problems for delegate key rotation see issue #344
					if exists && common.HexToAddress(conf.EthAddress) == ethAddress {
						found = true
						break
					}
				}
				// slash validators for not confirming valsets
				if !found {
					cons, _ := currentBondedSet[i].GetConsAddr()
					consPower := currentBondedSet[i].ConsensusPower(h.k.StakingKeeper.PowerReduction(ctx))

					_, _ = h.k.StakingKeeper.Slash(
						ctx,
						cons,
						ctx.BlockHeight(),
						consPower,
						params.SlashFractionValset,
					)

					if !currentBondedSet[i].IsJailed() {
						_ = h.k.StakingKeeper.Jail(ctx, cons)
					}

					// nolint:errcheck //ignored on purpose
					ctx.EventManager().EmitTypedEvent(&types.EventValidatorSlash{
						HyperionId:       params.HyperionId,
						Power:            consPower,
						Reason:           "missing_valset_confirm",
						ConsensusAddress: sdk.ConsAddress(consAddr).String(),
						OperatorAddress:  currentBondedSet[i].OperatorAddress,
						Moniker:          currentBondedSet[i].GetMoniker(),
					})
				}
			}
		}

		// SLASH UNBONDING VALIDATORS who didn't attest valset request
		stakingParams, _ := h.k.StakingKeeper.GetParams(ctx)
		blockTime := ctx.BlockTime().Add(stakingParams.UnbondingTime)
		blockHeight := ctx.BlockHeight()
		unbondingValIterator, _ := h.k.StakingKeeper.ValidatorQueueIterator(ctx, blockTime, blockHeight)
		defer unbondingValIterator.Close()

		// All unbonding validators
		for ; unbondingValIterator.Valid(); unbondingValIterator.Next() {
			unbondingValidators := h.k.DeserializeValidatorIterator(unbondingValIterator.Value())
			for _, valAddr := range unbondingValidators.Addresses {
				addr, err := sdk.ValAddressFromBech32(valAddr)
				if err != nil {
					metrics.ReportFuncError(h.svcTags)
					panic(err)
				}

				validator, _ := h.k.StakingKeeper.GetValidator(ctx, addr)
				valConsAddr, _ := validator.GetConsAddr()
				valSigningInfo, err := h.k.SlashingKeeper.GetValidatorSigningInfo(ctx, valConsAddr)

				exist := err == nil
				// Only slash validators who joined after valset is created and they are unbonding and UNBOND_SLASHING_WINDOW didn't passed
				if exist && valSigningInfo.StartHeight < int64(vs.Height) && validator.IsUnbonding() && vs.Height < uint64(validator.UnbondingHeight)+params.UnbondSlashingValsetsWindow {
					// Check if validator has confirmed valset or not
					found := false
					for _, conf := range confirms {
						ethAddress, exists := h.k.GetEthAddressByValidator(ctx, addr)
						if exists && common.HexToAddress(conf.EthAddress) == ethAddress {
							found = true
							break
						}
					}

					// slash validators for not confirming valsets
					if !found {
						consPower := validator.ConsensusPower(h.k.StakingKeeper.PowerReduction(ctx))

						_, _ = h.k.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), consPower, params.SlashFractionValset)

						if !validator.IsJailed() {
							_ = h.k.StakingKeeper.Jail(ctx, valConsAddr)
						}

						// nolint:errcheck //ignored on purpose
						ctx.EventManager().EmitTypedEvent(&types.EventValidatorSlash{
							HyperionId:       params.HyperionId,
							Power:            consPower,
							Reason:           "missing_valset_confirm",
							ConsensusAddress: validator.String(),
							OperatorAddress:  validator.OperatorAddress,
							Moniker:          validator.GetMoniker(),
						})
					}
				}
			}
		}

		// then we set the latest slashed valset  nonce
		h.k.SetLastSlashedValsetNonce(ctx, vs.Nonce)
	}
}

func (h *BlockHandler) batchSlashing(ctx sdk.Context, params *types.CounterpartyChainParams) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	// #2 condition
	// We look through the full bonded set (not just the active set, include unbonding validators)
	// and we slash users who haven't signed a batch confirmation that is >15hrs in blocks old
	maxHeight := uint64(0)

	// don't slash in the beginning before there aren't even SignedBatchesWindow blocks yet
	if uint64(ctx.BlockHeight()) > params.SignedBatchesWindow {
		maxHeight = uint64(ctx.BlockHeight()) - params.SignedBatchesWindow
	} else {
		// we can't slash anyone if this window has not yet passed
		return
	}

	unslashedBatches := h.k.GetUnslashedBatches(ctx, maxHeight)

	for _, batch := range unslashedBatches {
		// SLASH BONDED VALIDTORS who didn't attest batch requests
		currentBondedSet, _ := h.k.StakingKeeper.GetBondedValidatorsByPower(ctx)
		confirms := h.k.GetBatchConfirmByNonceAndTokenContract(ctx, batch.BatchNonce, common.HexToAddress(batch.TokenContract))
		for i := range currentBondedSet {
			// Don't slash validators who joined after batch is created
			consAddr, _ := currentBondedSet[i].GetConsAddr()

			valSigningInfo, err := h.k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
			if exist := err == nil; exist && valSigningInfo.StartHeight > int64(batch.Block) {
				continue
			}

			found := false
			for _, batchConfirmation := range confirms {
				// TODO this presents problems for delegate key rotation see issue #344
				orchestratorAcc, _ := sdk.AccAddressFromBech32(batchConfirmation.Orchestrator)
				delegatedOperator, delegatedFound := h.k.GetOrchestratorValidator(ctx, orchestratorAcc)
				operatorAddr, _ := sdk.ValAddressFromBech32(currentBondedSet[i].GetOperator())
				if delegatedFound && delegatedOperator.Equals(operatorAddr) {
					found = true
					break
				}
			}

			if !found {
				cons, _ := currentBondedSet[i].GetConsAddr()
				consPower := currentBondedSet[i].ConsensusPower(h.k.StakingKeeper.PowerReduction(ctx))

				_, _ = h.k.StakingKeeper.Slash(ctx, cons, ctx.BlockHeight(), consPower, params.SlashFractionBatch)

				if !currentBondedSet[i].IsJailed() {
					_ = h.k.StakingKeeper.Jail(ctx, cons)
				}

				// nolint:errcheck //ignored on purpose
				ctx.EventManager().EmitTypedEvent(&types.EventValidatorSlash{
					HyperionId:       params.HyperionId,
					Power:            consPower,
					Reason:           "missing_batch_confirm",
					ConsensusAddress: currentBondedSet[i].String(),
					OperatorAddress:  currentBondedSet[i].OperatorAddress,
					Moniker:          currentBondedSet[i].GetMoniker(),
				})
			}
		}

		// then we set the latest slashed batch block
		h.k.SetLastSlashedBatchBlock(ctx, batch.Block)
	}
}

func (h *BlockHandler) pruneValsets(ctx sdk.Context, params *types.CounterpartyChainParams) {
	// ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	// defer doneFn()

	for _, counterParty := range h.k.GetParams(ctx).CounterpartyChainParams {
		hyperionId := counterParty.HyperionId
		// Validator set pruning
		// prune all validator sets with a nonce less than the
		// last observed nonce, they can't be submitted any longer
		//
		// Only prune valsets after the signed valsets window has passed
		// so that slashing can occur the block before we remove them
		lastObserved := h.k.GetLastObservedValset(ctx, hyperionId)
		currentBlock := uint64(ctx.BlockHeight())
		tooEarly := currentBlock < params.SignedValsetsWindow
		if lastObserved != nil && !tooEarly {
			earliestToPrune := currentBlock - params.SignedValsetsWindow
			sets := h.k.GetValsets(ctx)

			for _, set := range sets {
				if set.Nonce < lastObserved.Nonce && set.Height < earliestToPrune {
					h.k.DeleteValset(ctx, set.Nonce)
				}
			}
		}
	}
}
