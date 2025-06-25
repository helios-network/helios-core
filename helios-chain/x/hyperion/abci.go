package hyperion

import (
	"fmt"
	"sort"
	"strings"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"

	cmn "helios-core/helios-chain/precompiles/common"

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
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	params := h.k.GetParams(ctx)
	for _, counterpartyChainParams := range params.CounterpartyChainParams {
		if counterpartyChainParams.Paused {
			continue
		}
		h.slashing(ctx, counterpartyChainParams)
		h.attestationTally(ctx, counterpartyChainParams)
		h.cleanupTimedOutBatches(ctx, counterpartyChainParams)
		h.cleanupTimedOutOutgoingTx(ctx, counterpartyChainParams)
		h.createValsets(ctx, counterpartyChainParams)
		h.pruneValsets(ctx, counterpartyChainParams)
		h.pruneAttestations(ctx, counterpartyChainParams)
		h.executeExternalDataTxs(ctx, counterpartyChainParams)
	}
}

func (h *BlockHandler) createValsets(ctx sdk.Context, params *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	// Auto ValsetRequest Creation.
	// WARNING: do not use k.GetLastObservedValset in this function, it *will* result in losing control of the bridge
	// 1. If there are no valset requests, create a new one.
	// 2. If there is at least one validator who started unbonding in current block. (we persist last unbonded block height in hooks.go)
	//      This will make sure the unbonding validator has to provide an attestation to a new Valset
	//	    that excludes him before he completely Unbonds.  Otherwise he will be slashed
	// 3. If power change between validators of CurrentValset and latest valset request is > 5%

	// get the last valsets to compare against
	latestValset := h.k.GetLatestValset(ctx, params.HyperionId)
	lastUnbondingHeight := h.k.GetLastUnbondingBlockHeight(ctx)

	if (latestValset == nil) || (lastUnbondingHeight == uint64(ctx.BlockHeight())) ||
		(types.BridgeValidators(h.k.GetCurrentValset(ctx, params.HyperionId).Members).PowerDiff(latestValset.Members) > 0.05) {
		// if the conditions are true, put in a new validator set request to be signed and submitted to Ethereum
		h.k.Logger(ctx).Info("HYPERION - ABCI.go - createValsets -> SetValsetRequest", "hyperionId", params.HyperionId)
		h.k.SetValsetRequest(ctx, params.HyperionId, params.OffsetValsetNonce)
	}
}

// Iterate over all attestations currently being voted on in order of nonce
// and prune those that are older than the current nonce and no longer have any
// use. This could be combined with create attestation and save some computation
// but (A) pruning keeps the iteration small in the first place and (B) there is
// already enough nuance in the other handler that it's best not to complicate it further
func (h *BlockHandler) pruneAttestations(ctx sdk.Context, counterParty *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	hyperionId := counterParty.HyperionId
	attmap := h.k.GetAttestationMapping(ctx, hyperionId)

	// We make a slice with all the event nonces that are in the attestation mapping
	keys := make([]uint64, 0, len(attmap))
	for k := range attmap {
		keys = append(keys, k)
	}
	// Then we sort it
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })

	lastObservedEventNonce := h.k.GetLastObservedEventNonce(ctx, hyperionId)
	// This iterates over all keys (event nonces) in the attestation mapping. Each value contains
	// a slice with one or more attestations at that event nonce. There can be multiple attestations
	// at one event nonce when validators disagree about what event happened at that nonce.
	for _, nonce := range keys {
		// This iterates over all attestations at a particular event nonce.
		// They are ordered by when the first attestation at the event nonce was received.
		// This order is not important.
		for _, att := range attmap[nonce] {
			// we delete all attestations earlier than the current event nonce
			if nonce < lastObservedEventNonce {
				if att.Observed {
					h.k.DeleteAttestation(ctx, att.HyperionId, att)
					h.k.StoreNonceObserved(ctx, att.HyperionId, nonce)

					claim, err := h.k.UnpackAttestationClaim(att)
					if err != nil {
						h.k.Logger(ctx).Error("HYPERION - ABCI.go - pruneAttestations -> ", "error", err)
						continue
					}
					// store finalized attestation if it's MsgDepositClaim
					if claim, ok := claim.(*types.MsgDepositClaim); ok {

						validators := []string{}
						proofs := []string{}
						for _, validator := range att.Votes {
							validatorSplitted := strings.Split(validator, ":")
							validators = append(validators, cmn.AnyToHexAddress(validatorSplitted[0]).String())
							proofs = append(proofs, validatorSplitted[1])
						}

						tokenToDenom, _ := h.k.GetTokenFromAddress(ctx, claim.HyperionId, common.HexToAddress(claim.TokenContract))

						tokenAddress := ""
						if tokenToDenom != nil {
							tokenAddress = tokenToDenom.Denom
						}

						h.k.StoreFinalizedTx(ctx, &types.TransferTx{
							HyperionId:  claim.HyperionId,
							Id:          claim.EventNonce,
							Height:      claim.BlockHeight,
							Sender:      cmn.AnyToHexAddress(claim.EthereumSender).String(),
							DestAddress: cmn.AnyToHexAddress(claim.CosmosReceiver).String(),
							SentToken: &types.Token{
								Amount:   claim.Amount,
								Contract: claim.TokenContract,
							},
							SentFee: &types.Token{
								Amount:   math.NewInt(0),
								Contract: "",
							},
							ReceivedToken: &types.Token{
								Amount:   claim.Amount,
								Contract: tokenAddress,
							},
							ReceivedFee: &types.Token{
								Amount:   math.NewInt(0),
								Contract: "",
							},
							Status:    "BRIDGED",
							Direction: "IN",
							ChainId:   counterParty.BridgeChainId,
							TxHash:    claim.TxHash,
							Proof: &types.Proof{
								Orchestrators: strings.Join(validators, ","),
								Hashs:         strings.Join(proofs, ","),
							},
						})
					}
				}
			}
		}
	}
}

func (h *BlockHandler) slashing(ctx sdk.Context, params *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	// Slash validator for not confirming valset requests, batch requests and not attesting claims rightfully
	h.valsetSlashing(ctx, params)
	h.batchSlashing(ctx, params)
}

// Iterate over all attestations currently being voted on in order of nonce and
// "Observe" those who have passed the threshold. Break the loop once we see
// an attestation that has not passed the threshold
func (h *BlockHandler) attestationTally(ctx sdk.Context, counterParty *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	hyperionId := counterParty.HyperionId
	attmap := h.k.GetAttestationMapping(ctx, hyperionId)
	// We make a slice with all the event nonces that are in the attestation mapping
	keys := make([]uint64, 0, len(attmap))
	h.k.Logger(ctx).Info("HYPERION - ABCI.go - attestationTally ->", "attmap", len(attmap))
	for k := range attmap {
		keys = append(keys, k)
	}
	// Then we sort it
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })

	// This iterates over all keys (event nonces) in the attestation mapping. Each value contains
	// a slice with one or more attestations at that event nonce. There can be multiple attestations
	// at one event nonce when validators disagree about what event happened at that nonce.
	for _, nonce := range keys {
		// h.k.Logger(ctx).Info("HYPERION - ABCI.go - attestationTally ->", "nonce", nonce)
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
			// h.k.Logger(ctx).Info("HYPERION - ABCI.go - attestationTally ->", "h.k.GetLastObservedEventNonce(ctx)", h.k.GetLastObservedEventNonce(ctx, attestation.HyperionId))
			// if nonce == h.k.GetLastObservedEventNonce(ctx, attestation.HyperionId)+1 {
			// 	h.k.TryAttestation(ctx, attestation, false)
			// }
			h.k.TryAttestation(ctx, attestation, false)
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
func (h *BlockHandler) cleanupTimedOutBatches(ctx sdk.Context, counterParty *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	hyperionId := counterParty.HyperionId

	ethereumHeight := h.k.GetLastObservedEthereumBlockHeight(ctx, hyperionId).EthereumBlockHeight
	batches := h.k.GetOutgoingTxBatches(ctx, hyperionId)

	for _, batch := range batches {
		if batch.BatchTimeout < ethereumHeight {
			err := h.k.CancelOutgoingTXBatch(ctx, common.HexToAddress(batch.TokenContract), batch.BatchNonce, batch.HyperionId)
			if err != nil {
				ctx.Logger().Error("failed to cancel outgoing tx batch", "error", err, "block", batch.Block, "batch_nonce", batch.BatchNonce)
			}
		}
	}
}

func (h *BlockHandler) cleanupTimedOutOutgoingTx(ctx sdk.Context, counterParty *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	hyperionId := counterParty.HyperionId
	projectedEthereumHeight := h.k.GetProjectedCurrentEthereumHeight(ctx, hyperionId)
	txs := h.k.GetPoolTransactions(ctx, hyperionId)

	for _, tx := range txs {
		if tx.TxTimeout < projectedEthereumHeight {
			alreadyInBatch := false

			batches := h.k.GetOutgoingTxBatches(ctx, hyperionId)
			for _, batch := range batches {
				for _, batchTx := range batch.Transactions {
					if batchTx.Id == tx.Id {
						alreadyInBatch = true
						break
					}
				}
			}

			if !alreadyInBatch { // we can process cancel
				sender, _ := sdk.AccAddressFromBech32(tx.Sender)
				err := h.k.RemoveFromOutgoingPoolAndRefund(ctx, tx.HyperionId, tx.Id, sender)
				if err != nil {
					ctx.Logger().Error("failed to cancel outgoing tx", "error", err, "txId", tx.Id, "sender", tx.Sender)
				}
			}
		}
	}
}

func (h *BlockHandler) valsetSlashing(ctx sdk.Context, params *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	maxHeight := uint64(0)

	// don't slash in the beginning before there aren't even SignedValsetsWindow blocks yet
	if uint64(ctx.BlockHeight()) > params.SignedValsetsWindow {
		maxHeight = uint64(ctx.BlockHeight()) - params.SignedValsetsWindow
	} else {
		// we can't slash anyone if SignedValsetWindow blocks have not passed
		return
	}

	unslashedValsets := h.k.GetUnslashedValsets(ctx, params.HyperionId, maxHeight)

	// unslashedValsets are sorted by nonce in ASC order
	for _, vs := range unslashedValsets {
		confirms := h.k.GetValsetConfirms(ctx, vs.HyperionId, vs.Nonce)

		// SLASH BONDED VALIDATORS who didn't attest valset request
		currentBondedSet, _ := h.k.StakingKeeper.GetBondedValidatorsByPower(ctx)

		for i := range currentBondedSet {
			if currentBondedSet[i].IsJailed() {
				// if the validator is jailed, we can skip the validator
				continue
			}

			consAddr, _ := currentBondedSet[i].GetConsAddr()
			valSigningInfo, err := h.k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)

			exist := err == nil
			//  Slash validator ONLY if he joined after valset is created
			if exist && valSigningInfo.StartHeight < int64(vs.Height) {
				// Check if validator has confirmed valset or not
				found := false
				accAddr, _ := sdk.AccAddressFromBech32(currentBondedSet[i].GetOperator())

				valAddr, exists := h.k.GetOrchestratorValidator(ctx, vs.HyperionId, accAddr)

				if !exists {
					// if the validator is not found, it means that the validator is not an orchestrator
					// so we can skip the validator
					continue
				}

				for _, conf := range confirms {
					ethAddress, exists := h.k.GetEthAddressByValidator(ctx, vs.HyperionId, valAddr)
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
				accAddr, _ := sdk.AccAddressFromBech32(valAddr)
				_, exists := h.k.GetOrchestratorValidator(ctx, vs.HyperionId, accAddr)
				if !exists {
					// if the validator is not found, it means that the validator is not an orchestrator
					// so we can skip the validator
					continue
				}
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
						ethAddress, exists := h.k.GetEthAddressByValidator(ctx, vs.HyperionId, addr)
						if exists && common.HexToAddress(conf.EthAddress) == ethAddress {
							found = true
							break
						}
					}

					// slash validators for not confirming valsets
					if !found {
						consPower := validator.ConsensusPower(h.k.StakingKeeper.PowerReduction(ctx))

						_, _ = h.k.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), consPower, params.SlashFractionValset)
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
		h.k.SetLastSlashedValsetNonce(ctx, vs.HyperionId, vs.Nonce)
	}
}

func (h *BlockHandler) batchSlashing(ctx sdk.Context, params *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

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

	unslashedBatches := h.k.GetUnslashedBatches(ctx, params.HyperionId, maxHeight)

	for _, batch := range unslashedBatches {
		// SLASH BONDED VALIDTORS who didn't attest batch requests
		currentBondedSet, _ := h.k.StakingKeeper.GetBondedValidatorsByPower(ctx)
		confirms := h.k.GetBatchConfirmByNonceAndTokenContract(ctx, batch.HyperionId, batch.BatchNonce, common.HexToAddress(batch.TokenContract))
		for i := range currentBondedSet {
			// Don't slash validators who joined after batch is created
			consAddr, _ := currentBondedSet[i].GetConsAddr()
			accAddr, _ := sdk.AccAddressFromBech32(currentBondedSet[i].GetOperator())
			_, exists := h.k.GetOrchestratorValidator(ctx, batch.HyperionId, accAddr)
			if !exists {
				// if the validator is not found, it means that the validator is not an orchestrator
				// so we can skip the validator
				continue
			}

			valSigningInfo, err := h.k.SlashingKeeper.GetValidatorSigningInfo(ctx, consAddr)
			if exist := err == nil; exist && valSigningInfo.StartHeight > int64(batch.Block) {
				continue
			}

			found := false
			for _, batchConfirmation := range confirms {
				// TODO this presents problems for delegate key rotation see issue #344
				orchestratorAcc, _ := sdk.AccAddressFromBech32(batchConfirmation.Orchestrator)
				delegatedOperator, delegatedFound := h.k.GetOrchestratorValidator(ctx, batchConfirmation.HyperionId, orchestratorAcc)
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
		h.k.SetLastSlashedBatchBlock(ctx, params.HyperionId, batch.Block)
	}
}

func (h *BlockHandler) pruneValsets(ctx sdk.Context, params *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

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
			sets := h.k.GetValsets(ctx, hyperionId)

			for _, set := range sets {
				if set.Nonce < lastObserved.Nonce && set.Height < earliestToPrune {
					h.k.DeleteValset(ctx, set.HyperionId, set.Nonce)
				}
			}
		}
	}
}

func (h *BlockHandler) selectBestClaimFromListOfClaims(claims []*types.MsgExternalDataClaim) *types.MsgExternalDataClaim {
	// If no claims, return nil
	if len(claims) == 0 {
		return nil
	}

	// If only one claim, return it
	if len(claims) == 1 {
		return claims[0]
	}

	// Map to store frequency of each combination
	frequencies := make(map[string]int)
	claimsByKey := make(map[string]*types.MsgExternalDataClaim)

	// Count frequencies of each unique combination
	for _, claim := range claims {
		// Create a unique key combining the relevant fields
		key := fmt.Sprintf("%d|%s|%s",
			claim.TxNonce,
			claim.CallDataResult,
			claim.CallDataResultError,
		)

		frequencies[key]++
		claimsByKey[key] = claim
	}

	// Find the key with highest frequency
	var maxFreq int
	var bestKey string
	for key, freq := range frequencies {
		if freq > maxFreq {
			maxFreq = freq
			bestKey = key
		}
	}

	// Return the claim corresponding to the most frequent combination
	return claimsByKey[bestKey]
}

func (h *BlockHandler) executeExternalDataTxs(ctx sdk.Context, counterParty *types.CounterpartyChainParams) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, h.svcTags)
	defer doneFn()

	if counterParty.BridgeChainType != "evm" {
		return
	}

	txs := h.k.GetOutgoingExternalDataTXs(ctx, counterParty.HyperionId)

	totalPower := h.k.GetCurrentValsetTotalPower(ctx, counterParty.HyperionId)
	requiredPower := h.k.GetRequiredPower(totalPower, 33)
	attestationPower := math.ZeroInt()

	for _, tx := range txs {

		if tx.Timeout < uint64(ctx.BlockHeight()) {
			h.k.RefundExternalData(ctx, *tx)
			h.k.DeleteExternalData(ctx, *tx)
			h.k.Logger(ctx).Debug("HYPERION - ABCI.go - executeAllExternalDataTxs -> deleted tx", "tx", tx)
			continue
		}

		if tx.Timeout-50 > uint64(ctx.BlockHeight()) {
			h.k.Logger(ctx).Debug("HYPERION - ABCI.go - executeAllExternalDataTxs -> waiting reasonable time for the tx to be executed", "tx", tx)
			continue
		}

		// check all claims and range the similar results in on map with the result as key and the count as value
		bestClaim := h.selectBestClaimFromListOfClaims(tx.Claims)

		if bestClaim == nil {
			h.k.Logger(ctx).Error("HYPERION - ABCI.go - executeAllExternalDataTxs -> no best claim found yet", "tx", tx)
			continue
		}

		for _, vote := range tx.Votes {
			val := cmn.ValAddressFromHexAddressString(vote)
			validatorPower, err := h.k.StakingKeeper.GetLastValidatorPower(ctx, val)
			if err != nil {
				metrics.ReportFuncError(h.svcTags)
				h.k.Logger(ctx).Error("HYPERION - ABCI.go - executeAllExternalDataTxs -> GetLastValidatorPower", "error", err)
				break
			}
			// Add it to the attestation power's sum
			attestationPower = attestationPower.Add(math.NewInt(validatorPower))

			h.k.Logger(ctx).Debug("HYPERION - ABCI.go - executeAllExternalDataTxs -> attestationPower", "attestationPower", attestationPower, "requiredPower", requiredPower)

			if attestationPower.GTE(requiredPower) {

				//todo check claims Results
				h.k.Logger(ctx).Debug("HYPERION - ABCI.go - executeAllExternalDataTxs -> attestationPower", "attestationPower", attestationPower)

				h.k.OutgoingExternalDataTxExecuted(ctx, tx, bestClaim, &types.Attestation{
					Observed: true,
					Votes:    tx.Votes, // maybe exclude the false claims from attestation to do not reward them
				})

				// update the rpc used
				if strings.Contains(bestClaim.RpcUsed, "https://") {
					h.k.UpdateRpcUsed(ctx, tx.HyperionId, bestClaim.RpcUsed, bestClaim.BlockHeight)
				}
				break
			}
		}
	}
}
