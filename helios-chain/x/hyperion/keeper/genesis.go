package keeper

import (
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/hyperion/types"
)

// NormalizeGenesis takes care of formatting in the internal structures, as they're used as values
// in the keeper eventually, while having raw strings in them.
func NormalizeGenesis(data *types.GenesisState) {
	for _, counterpartyParams := range data.Params.CounterpartyChainParams {
		counterpartyParams.BridgeCounterpartyAddress = common.HexToAddress(counterpartyParams.BridgeCounterpartyAddress).Hex()
	}

	for _, subState := range data.SubStates {

		for _, valset := range subState.Valsets {
			for _, member := range valset.Members {
				member.EthereumAddress = common.HexToAddress(member.EthereumAddress).Hex()
			}
		}

		for _, valsetConfirm := range subState.ValsetConfirms {
			valsetConfirm.EthAddress = common.HexToAddress(valsetConfirm.EthAddress).Hex()
		}

		for _, batch := range subState.Batches {
			batch.TokenContract = common.HexToAddress(batch.TokenContract).Hex()

			for _, outgoingTx := range batch.Transactions {
				outgoingTx.DestAddress = common.HexToAddress(outgoingTx.DestAddress).Hex()
				outgoingTx.Fee.Contract = common.HexToAddress(outgoingTx.Fee.Contract).Hex()
				outgoingTx.Token.Contract = common.HexToAddress(outgoingTx.Token.Contract).Hex()
			}
		}

		for _, batchConfirm := range subState.BatchConfirms {
			batchConfirm.EthSigner = common.HexToAddress(batchConfirm.EthSigner).Hex()
			batchConfirm.TokenContract = common.HexToAddress(batchConfirm.TokenContract).Hex()
		}

		for _, orchestrator := range subState.OrchestratorAddresses {
			orchestrator.EthAddress = common.HexToAddress(orchestrator.EthAddress).Hex()
		}
	}
}

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data *types.GenesisState) {
	k.CreateModuleAccount(ctx)

	NormalizeGenesis(data)

	k.SetParams(ctx, data.Params)

	for _, counterparty := range data.Params.CounterpartyChainParams {
		for _, token := range counterparty.DefaultTokens {
			k.CreateOrLinkTokenToChain(ctx, counterparty.BridgeChainId, counterparty.BridgeChainName, token)
		}
	}

	for _, blacklistAddress := range data.BlacklistAddresses {
		blacklistAddr := common.HexToAddress(blacklistAddress)
		k.SetBlacklistAddress(ctx, blacklistAddr)
	}

	for _, subState := range data.SubStates {

		for _, valset := range subState.Valsets {

			if valset.HyperionId == subState.HyperionId {
				k.StoreValsetUnsafe(ctx, valset)
			}
		}

		for _, valsetConfirm := range subState.ValsetConfirms {
			if valsetConfirm.HyperionId == subState.HyperionId {
				k.SetValsetConfirm(ctx, valsetConfirm)
			}
		}

		for _, batch := range subState.Batches {
			if batch.HyperionId == subState.HyperionId {
				k.StoreBatchUnsafe(ctx, batch)
			}
		}

		for _, batchConfirm := range subState.BatchConfirms {
			if batchConfirm.HyperionId == subState.HyperionId {
				k.SetBatchConfirm(ctx, batchConfirm)
			}
		}

		// reset pool transactions in state
		for _, tx := range subState.UnbatchedTransfers {
			if tx.HyperionId == subState.HyperionId {
				if err := k.setPoolEntry(ctx, tx); err != nil {
					panic(err)
				}
			}
		}

		// reset attestations in state
		for _, attestation := range subState.Attestations {
			claim, err := k.UnpackAttestationClaim(attestation)
			if err != nil {
				panic("couldn't UnpackAttestationClaim")
			}

			k.SetAttestation(ctx, claim.GetHyperionId(), claim.GetEventNonce(), claim.ClaimHash(), attestation)
		}
		k.setLastObservedEventNonce(ctx, subState.HyperionId, subState.LastObservedNonce)
		k.SetLastObservedEthereumBlockHeight(ctx, subState.HyperionId, subState.LastObservedEthereumHeight.EthereumBlockHeight, subState.LastObservedEthereumHeight.CosmosBlockHeight)
		k.SetLastOutgoingBatchID(ctx, subState.HyperionId, subState.LastOutgoingBatchId)
		k.SetLastOutgoingPoolID(ctx, subState.HyperionId, subState.LastOutgoingPoolId)
		k.SetLastObservedValset(ctx, subState.HyperionId, subState.LastObservedValset)

		for _, attestation := range subState.Attestations {
			claim, err := k.UnpackAttestationClaim(attestation)
			if err != nil {
				panic("couldn't UnpackAttestationClaim")
			}

			// reconstruct the latest event nonce for every validator
			// if somehow this genesis state is saved when all attestations
			// have been cleaned up GetLastEventNonceByValidator handles that case
			//
			// if we where to save and load the last event nonce for every validator
			// then we would need to carry that state forever across all chain restarts
			// but since we've already had to handle the edge case of new validators joining
			// while all attestations have already been cleaned up we can do this instead and
			// not carry around every validators event nonce counter forever.
			for _, vote := range attestation.Votes {
				val, err := sdk.ValAddressFromBech32(vote)
				if err != nil {
					k.Logger(ctx).Error("failed to get last event by validator and hyperion id", "error", err)
					continue
				}
				lastEvent := k.GetLastEventByValidatorAndHyperionId(ctx, attestation.HyperionId, val)
				if claim.GetEventNonce() > lastEvent.EthereumEventNonce {
					k.setLastEventByValidatorAndHyperionId(ctx, attestation.HyperionId, val, claim.GetEventNonce(), claim.GetBlockHeight())
				}
			}
		}

		// reset delegate keys in state
		for _, keys := range subState.OrchestratorAddresses {
			err := keys.ValidateBasic()
			if err != nil {
				panic("Invalid delegate key in Genesis!")
			}
			validatorAccountAddress, _ := sdk.AccAddressFromBech32(keys.Sender)
			valAddress := sdk.ValAddress(validatorAccountAddress.Bytes())
			orchestrator, _ := sdk.AccAddressFromBech32(keys.Orchestrator)

			// set the orchestrator Cosmos address
			k.SetOrchestratorValidator(ctx, keys.HyperionId, valAddress, orchestrator)

			// set the orchestrator Ethereum address
			k.SetEthAddressForValidator(ctx, keys.HyperionId, valAddress, common.HexToAddress(keys.EthAddress))
		}
	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	p := k.GetParams(ctx)

	blacklistAddresses := k.GetAllBlacklistAddresses(ctx)

	subStates := make([]*types.GenesisHyperionState, 0)

	for _, param := range p.CounterpartyChainParams {
		// param.HyperionId
		var (
			batches                         = k.GetOutgoingTxBatches(ctx, param.HyperionId)
			valsets                         = k.GetValsets(ctx, param.HyperionId)
			attmap                          = k.GetAttestationMapping(ctx, param.HyperionId)
			vsconfs                         = []*types.MsgValsetConfirm{}
			batchconfs                      = []*types.MsgConfirmBatch{}
			attestations                    = []*types.Attestation{}
			orchestratorAddresses           = k.GetOrchestratorAddresses(ctx, param.HyperionId)
			lastObservedEventNonce          = k.GetLastObservedEventNonce(ctx, param.HyperionId)
			lastObservedEthereumBlockHeight = k.GetLastObservedEthereumBlockHeight(ctx, param.HyperionId)
			unbatchedTransfers              = k.GetPoolTransactions(ctx, param.HyperionId)
		)

		// export valset confirmations from state
		for _, vs := range valsets {
			vsconfs = append(vsconfs, k.GetValsetConfirms(ctx, vs.HyperionId, vs.Nonce)...)
		}

		// export batch confirmations from state
		for _, batch := range batches {
			batchconfs = append(batchconfs, k.GetBatchConfirmByNonceAndTokenContract(ctx, param.HyperionId, batch.BatchNonce, common.HexToAddress(batch.TokenContract))...)
		}

		// sort attestation map keys since map iteration is non-deterministic
		attestationHeights := make([]uint64, 0, len(attmap))
		for k := range attmap {
			attestationHeights = append(attestationHeights, k)
		}
		sort.SliceStable(attestationHeights, func(i, j int) bool {
			return attestationHeights[i] < attestationHeights[j]
		})

		for _, height := range attestationHeights {
			attestations = append(attestations, attmap[height]...)
		}

		lastOutgoingBatchID := k.GetLastOutgoingBatchID(ctx, param.HyperionId)
		lastOutgoingPoolID := k.GetLastOutgoingPoolID(ctx, param.HyperionId)
		lastObservedValset := k.GetLastObservedValset(ctx, param.HyperionId)

		subStates = append(subStates, &types.GenesisHyperionState{
			HyperionId:                 param.HyperionId,
			LastObservedNonce:          lastObservedEventNonce,
			LastObservedEthereumHeight: &lastObservedEthereumBlockHeight,
			Valsets:                    valsets,
			ValsetConfirms:             vsconfs,
			Batches:                    batches,
			BatchConfirms:              batchconfs,
			Attestations:               attestations,
			OrchestratorAddresses:      orchestratorAddresses,
			UnbatchedTransfers:         unbatchedTransfers,
			LastOutgoingBatchId:        lastOutgoingBatchID,
			LastOutgoingPoolId:         lastOutgoingPoolID,
			LastObservedValset:         *lastObservedValset,
		})
	}

	return types.GenesisState{
		Params:             p,
		SubStates:          subStates,
		BlacklistAddresses: blacklistAddresses,
	}
}
