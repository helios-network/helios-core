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
		counterpartyParams.CosmosCoinErc20Contract = common.HexToAddress(counterpartyParams.CosmosCoinErc20Contract).Hex()
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
				outgoingTx.Erc20Fee.Contract = common.HexToAddress(outgoingTx.Erc20Fee.Contract).Hex()
				outgoingTx.Erc20Token.Contract = common.HexToAddress(outgoingTx.Erc20Token.Contract).Hex()
			}
		}

		for _, batchConfirm := range subState.BatchConfirms {
			batchConfirm.EthSigner = common.HexToAddress(batchConfirm.EthSigner).Hex()
			batchConfirm.TokenContract = common.HexToAddress(batchConfirm.TokenContract).Hex()
		}

		for _, orchestrator := range subState.OrchestratorAddresses {
			orchestrator.EthAddress = common.HexToAddress(orchestrator.EthAddress).Hex()
		}

		for _, token := range subState.Erc20ToDenoms {
			token.Erc20 = common.HexToAddress(token.Erc20).Hex()
		}
	}
}

// InitGenesis starts a chain from a genesis state
func InitGenesis(ctx sdk.Context, k Keeper, data *types.GenesisState) {
	k.CreateModuleAccount(ctx)

	NormalizeGenesis(data)

	k.SetParams(ctx, data.Params)

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
		k.setLastObservedEventNonceForHyperionID(ctx, subState.HyperionId, subState.LastObservedNonce)
		k.SetLastObservedEthereumBlockHeight(ctx, subState.HyperionId, subState.LastObservedEthereumHeight)
		k.SetLastOutgoingBatchID(ctx, subState.HyperionId, subState.LastOutgoingBatchId)
		k.SetLastOutgoingPoolID(ctx, subState.HyperionId, subState.LastOutgoingPoolId)
		k.SetLastObservedValset(ctx, subState.LastObservedValset, subState.HyperionId)

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
					panic(err)
				}
				lastEvent := k.GetLastEventByValidatorByHyperionID(ctx, val, attestation.HyperionId)
				if claim.GetEventNonce() > lastEvent.EthereumEventNonce {
					k.setLastEventByValidatorByHyperionID(ctx, val, claim.GetEventNonce(), claim.GetBlockHeight(), attestation.HyperionId)
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
			k.SetEthAddressForValidator(ctx, valAddress, common.HexToAddress(keys.EthAddress), keys.HyperionId)
		}

		// populate state with cosmos originated denom-erc20 mapping
		for _, item := range subState.Erc20ToDenoms {
			k.SetCosmosOriginatedDenomToERC20ByHyperionID(ctx, item.Denom, common.HexToAddress(item.Erc20), subState.HyperionId)
		}

		for _, blacklistAddress := range subState.EthereumBlacklist {
			blacklistAddr := common.HexToAddress(blacklistAddress)
			k.SetEthereumBlacklistAddress(ctx, blacklistAddr)
		}

	}
}

// ExportGenesis exports all the state needed to restart the chain
// from the current state of the chain
func ExportGenesis(ctx sdk.Context, k Keeper) types.GenesisState {
	p := k.GetParams(ctx)

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
			orchestratorAddresses           = k.GetOrchestratorAddressesByHyperionID(ctx, param.HyperionId)
			lastObservedEventNonce          = k.GetLastObservedEventNonceForHyperionID(ctx, param.HyperionId)
			lastObservedEthereumBlockHeight = k.GetLastObservedEthereumBlockHeightForHyperionID(ctx, param.HyperionId)
			erc20ToDenoms                   = k.GetAllERC20ToDenom(ctx, param.HyperionId)
			unbatchedTransfers              = k.GetPoolTransactions(ctx, param.HyperionId)
			ethereumBlacklistAddresses      = k.GetAllEthereumBlacklistAddresses(ctx) // same for all
		)

		// export valset confirmations from state
		for _, vs := range valsets {
			vsconfs = append(vsconfs, k.GetValsetConfirms(ctx, vs.HyperionId, vs.Nonce)...)
		}

		// export batch confirmations from state
		for _, batch := range batches {
			batchconfs = append(batchconfs, k.GetBatchConfirmByNonceAndTokenContract(ctx, batch.BatchNonce, common.HexToAddress(batch.TokenContract), param.HyperionId)...)
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
			LastObservedEthereumHeight: lastObservedEthereumBlockHeight.EthereumBlockHeight,
			Valsets:                    valsets,
			ValsetConfirms:             vsconfs,
			Batches:                    batches,
			BatchConfirms:              batchconfs,
			Attestations:               attestations,
			OrchestratorAddresses:      orchestratorAddresses,
			Erc20ToDenoms:              erc20ToDenoms,
			UnbatchedTransfers:         unbatchedTransfers,
			LastOutgoingBatchId:        lastOutgoingBatchID,
			LastOutgoingPoolId:         lastOutgoingPoolID,
			LastObservedValset:         *lastObservedValset,
			EthereumBlacklist:          ethereumBlacklistAddresses,
		})
	}

	return types.GenesisState{
		Params:    p,
		SubStates: subStates,
	}
}
