package keeper

import (
	"helios-core/helios-chain/x/hyperion/types"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k *Keeper) GetOrchestratorData(ctx sdk.Context, orchestrator sdk.AccAddress) (*types.OrchestratorData, error) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetOrchestratorDataKey(orchestrator)
	bz := store.Get(key)
	if len(bz) == 0 {
		return &types.OrchestratorData{
			Orchestrator:             orchestrator.String(),
			OrchestratorHyperionData: make([]*types.OrchestratorHyperionData, 0),
		}, nil
	}

	var orchestratorData types.OrchestratorData
	k.cdc.MustUnmarshal(bz, &orchestratorData)
	return &orchestratorData, nil
}

func (k *Keeper) SetOrchestratorData(ctx sdk.Context, orchestrator sdk.AccAddress, orchestratorData types.OrchestratorData) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetOrchestratorDataKey(orchestrator)
	store.Set(key, k.cdc.MustMarshal(&orchestratorData))
}

func (k *Keeper) DeleteOrchestratorData(ctx sdk.Context, orchestrator sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetOrchestratorDataKey(orchestrator)
	store.Delete(key)
}

func (k *Keeper) GetOrchestratorHyperionData(ctx sdk.Context, orchestrator sdk.AccAddress, hyperionId uint64) (*types.OrchestratorHyperionData, error) {
	orchestratorData, err := k.GetOrchestratorData(ctx, orchestrator)
	if err != nil {
		return nil, err
	}

	for _, orchestratorHyperionData := range orchestratorData.OrchestratorHyperionData {
		if orchestratorHyperionData.HyperionId == hyperionId {
			return orchestratorHyperionData, nil
		}
	}

	return nil, errors.Wrap(types.ErrInvalid, "orchestrator hyperion data not found")
}

func (k *Keeper) SetOrchestratorHyperionData(ctx sdk.Context, orchestrator sdk.AccAddress, hyperionId uint64, orchestratorHyperionData types.OrchestratorHyperionData) error {
	orchestratorData, err := k.GetOrchestratorData(ctx, orchestrator)
	if err != nil {
		orchestratorData = &types.OrchestratorData{
			Orchestrator:             orchestrator.String(),
			OrchestratorHyperionData: make([]*types.OrchestratorHyperionData, 0),
		}
	}

	// replace if already exists
	replaced := false
	for i, data := range orchestratorData.OrchestratorHyperionData {
		if data.HyperionId == hyperionId {
			orchestratorData.OrchestratorHyperionData[i] = &orchestratorHyperionData
			replaced = true
			break
		}
	}

	if !replaced {
		orchestratorData.OrchestratorHyperionData = append(orchestratorData.OrchestratorHyperionData, &orchestratorHyperionData)
	}

	k.SetOrchestratorData(ctx, orchestrator, *orchestratorData)
	return nil
}

func (k *Keeper) AddSlashData(ctx sdk.Context, orchestrator sdk.AccAddress, hyperionId uint64, slashData types.SlashData) error {
	orchestratorData, err := k.GetOrchestratorHyperionData(ctx, orchestrator, hyperionId)
	if err != nil {
		return err
	}

	orchestratorData.TotalSlashCount++
	orchestratorData.TotalSlashAmount = orchestratorData.TotalSlashAmount.Add(slashData.SlashAmount)
	orchestratorData.SlashData = append(orchestratorData.SlashData, &slashData)
	allSlashData := make([]*types.SlashData, 0)
	for _, slash := range orchestratorData.SlashData {
		if slash.SlashTimestamp >= uint64(ctx.BlockTime().Unix())-uint64(86400) {
			allSlashData = append(allSlashData, slash)
		}
	}
	orchestratorData.SlashData = allSlashData
	k.SetOrchestratorHyperionData(ctx, orchestrator, hyperionId, *orchestratorData)
	return nil
}
