package types

func DefaultSubStates(params *Params) []*GenesisHyperionState {
	substate := make([]*GenesisHyperionState, 0)

	for _, counterParty := range params.CounterpartyChainParams {
		substate = append(substate, &GenesisHyperionState{
			HyperionId:            counterParty.HyperionId,
			LastObservedNonce:     0,
			Valsets:               make([]*Valset, 0),
			ValsetConfirms:        make([]*MsgValsetConfirm, 0),
			Batches:               make([]*OutgoingTxBatch, 0),
			BatchConfirms:         make([]*MsgConfirmBatch, 0),
			Attestations:          make([]*Attestation, 0),
			OrchestratorAddresses: make([]*MsgSetOrchestratorAddresses, 0),
			Erc20ToDenoms:         MockErc20ToDenoms(), //make([]*ERC20ToDenom, 0),
			UnbatchedTransfers:    make([]*OutgoingTransferTx, 0),
			LastObservedEthereumHeight: &LastObservedEthereumBlockHeight{
				CosmosBlockHeight:   1, // set to 1 by default
				EthereumBlockHeight: counterParty.BridgeContractStartHeight,
			},
			LastOutgoingBatchId: 0,
			LastOutgoingPoolId:  0,
			LastObservedValset: Valset{
				HyperionId: counterParty.HyperionId,
			},
			EthereumBlacklist: []string{},
		})
	}

	return substate
}
