package keeper

import (
	"fmt"
	"sort"
	"strconv"

	cmn "helios-core/helios-chain/precompiles/common"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/hyperion/types"

	erc20types "helios-core/helios-chain/x/erc20/types"
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

		for _, token := range subState.TokenAddressToDenoms {
			token.TokenAddress = common.HexToAddress(token.TokenAddress).Hex()
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

			tokenPair, ok := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, token.TokenAddressToDenom.Denom))

			if !ok {
				coinMetadata := banktypes.Metadata{
					Description: fmt.Sprintf("Token %s created with Hyperion", token.TokenAddressToDenom.Denom),
					Base:        token.TokenAddressToDenom.Denom,
					Name:        token.TokenAddressToDenom.Symbol,
					Symbol:      token.TokenAddressToDenom.Symbol,
					Decimals:    uint32(token.TokenAddressToDenom.Decimals),
					Display:     token.TokenAddressToDenom.Symbol,
					DenomUnits: []*banktypes.DenomUnit{
						{
							Denom:    token.TokenAddressToDenom.Denom,
							Exponent: 0,
						},
						{
							Denom:    token.TokenAddressToDenom.Symbol,
							Exponent: uint32(token.TokenAddressToDenom.Decimals),
						},
					},
					Logo: token.Logo,
				}

				contractAddr, err := k.erc20Keeper.DeployERC20Contract(ctx, coinMetadata)
				if err != nil {
					panic(fmt.Errorf("failed to deploy ERC20 contract: %w", err))
				}
				tokenPair = erc20types.NewTokenPair(contractAddr, token.TokenAddressToDenom.Denom, erc20types.OWNER_MODULE)
				k.erc20Keeper.SetToken(ctx, tokenPair)
				k.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
			}

			if token.TokenAddressToDenom.IsConcensusToken && !k.erc20Keeper.IsAssetWhitelisted(ctx, token.TokenAddressToDenom.Denom) {
				asset := erc20types.Asset{
					Denom:           token.TokenAddressToDenom.Denom,
					ContractAddress: tokenPair.Erc20Address,
					ChainId:         strconv.FormatUint(counterparty.BridgeChainId, 10), // Exemple de chainId, à ajuster si nécessaire
					ChainName:       counterparty.BridgeChainName,
					Decimals:        uint64(token.TokenAddressToDenom.Decimals),
					BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
					Symbol:          token.TokenAddressToDenom.Symbol,
				}
				k.erc20Keeper.AddAssetToConsensusWhitelist(ctx, asset)
			}

			for _, holder := range token.DefaultHolders {
				holder.Address = common.HexToAddress(holder.Address).Hex()

				k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.Coins{sdk.NewCoin(token.TokenAddressToDenom.Denom, holder.Amount)})
				k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, cmn.AccAddressFromHexAddressString(holder.Address), sdk.Coins{sdk.NewCoin(token.TokenAddressToDenom.Denom, holder.Amount)})
			}
		}
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
					panic(err)
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

		// populate state with cosmos originated denom-erc20 mapping
		chainId := k.GetChainIdFromHyperionId(ctx, subState.HyperionId)

		for _, item := range subState.TokenAddressToDenoms {
			k.SetToken(ctx, subState.HyperionId, item)
			metadata, found := k.bankKeeper.GetDenomMetaData(ctx, item.Denom)

			chainMetadata := &banktypes.ChainMetadata{
				ChainId:         chainId,
				ContractAddress: common.HexToAddress(item.TokenAddress).String(),
				Symbol:          metadata.Symbol,
				Decimals:        uint32(metadata.Decimals),
				IsOriginated:    !item.IsCosmosOriginated,
			}

			if found {
				metadata.ChainsMetadatas = append(metadata.ChainsMetadatas, chainMetadata)
				k.bankKeeper.SetDenomMetaData(ctx, metadata)
			} else {
				k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
					Base: item.Denom,
					ChainsMetadatas: []*banktypes.ChainMetadata{
						chainMetadata,
					},
				})
			}
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
			orchestratorAddresses           = k.GetOrchestratorAddresses(ctx, param.HyperionId)
			lastObservedEventNonce          = k.GetLastObservedEventNonce(ctx, param.HyperionId)
			lastObservedEthereumBlockHeight = k.GetLastObservedEthereumBlockHeight(ctx, param.HyperionId)
			tokens                          = k.GetAllTokens(ctx, param.HyperionId)
			unbatchedTransfers              = k.GetPoolTransactions(ctx, param.HyperionId)
			ethereumBlacklistAddresses      = k.GetAllEthereumBlacklistAddresses(ctx) // same for all
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
			TokenAddressToDenoms:       tokens,
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
