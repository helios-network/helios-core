package keeper

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/log"
	"github.com/hashicorp/go-metrics"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	types "helios-core/helios-chain/x/chronos/types"
	erc20types "helios-core/helios-chain/x/erc20/types"
)

type Keeper struct {
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	memKey        storetypes.StoreKey
	accountKeeper types.AccountKeeper
	evmKeeper     erc20types.EVMKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	evmKeeper erc20types.EVMKeeper,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		accountKeeper: accountKeeper,
		evmKeeper:     evmKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) ExecuteReadySchedules(ctx sdk.Context, executionStage types.ExecutionStage) {
	telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), types.LabelExecuteReadySchedules)
	schedules := k.getSchedulesReadyForExecution(ctx, executionStage)

	for _, schedule := range schedules {
		err := k.executeSchedule(ctx, schedule)
		recordExecutedSchedule(err, schedule)
	}
}

func (k *Keeper) AddSchedule(ctx sdk.Context, schedule types.Schedule) error {
	if k.scheduleExists(ctx, schedule.Id) {
		return fmt.Errorf("schedule already exists with id=%d", schedule.Id)
	}

	k.StoreSchedule(ctx, schedule)
	k.changeTotalCount(ctx, 1)

	return nil
}

func (k *Keeper) RemoveSchedule(ctx sdk.Context, id uint64, owner sdk.AccAddress) error {
	schedule, found := k.GetSchedule(ctx, id)
	if !found {
		return fmt.Errorf("schedule not found")
	}
	if schedule.OwnerAddress != owner.String() {
		return fmt.Errorf("unauthorized removal")
	}

	k.removeSchedule(ctx, id)
	k.changeTotalCount(ctx, -1)

	return nil
}

func (k *Keeper) GetSchedule(ctx sdk.Context, id uint64) (types.Schedule, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	bz := store.Get(GetScheduleIDBytes(id))
	if bz == nil {
		return types.Schedule{}, false
	}

	var schedule types.Schedule
	k.cdc.MustUnmarshal(bz, &schedule)
	return schedule, true
}

func (k *Keeper) getSchedulesReadyForExecution(ctx sdk.Context, executionStage types.ExecutionStage) []types.Schedule {
	params := k.GetParams(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	var schedules []types.Schedule
	count := uint64(0)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	currentBlock := uint64(ctx.BlockHeight())
	for ; iterator.Valid(); iterator.Next() {
		var schedule types.Schedule
		k.cdc.MustUnmarshal(iterator.Value(), &schedule)

		if schedule.ExecutionStage == executionStage &&
			currentBlock >= schedule.NextExecutionBlock &&
			(schedule.ExpirationBlock == 0 || currentBlock <= schedule.ExpirationBlock) {
			schedules = append(schedules, schedule)
			count++
			if count >= params.Limit {
				k.Logger(ctx).Info("Reached execution limit for the block")
				break
			}
		}
	}

	return schedules
}

// ParseParams converts string parameters into proper types based on ABI definitions.
func ParseParams(contractABI abi.ABI, methodName string, params []string) ([]interface{}, error) {
	method, exist := contractABI.Methods[methodName]
	if !exist {
		return nil, fmt.Errorf("method %s not found in ABI", methodName)
	}

	if len(method.Inputs) != len(params) {
		return nil, fmt.Errorf("expected %d parameters, got %d", len(method.Inputs), len(params))
	}

	parsed := make([]interface{}, len(params))
	for i, input := range method.Inputs {
		arg, err := ParseABIParam(input.Type, params[i])
		if err != nil {
			return nil, fmt.Errorf("error parsing param %d: %w", i, err)
		}
		parsed[i] = arg
	}

	return parsed, nil
}

// ParseABIParam parses a single parameter from string to the specified ABI type.
func ParseABIParam(typ abi.Type, param string) (interface{}, error) {
	switch typ.T {
	case abi.StringTy:
		return param, nil
	case abi.AddressTy:
		return common.HexToAddress(param), nil
	case abi.UintTy, abi.IntTy:
		bigInt, ok := new(big.Int).SetString(param, 10)
		if !ok {
			return nil, fmt.Errorf("invalid integer: %s", param)
		}
		return bigInt, nil
	case abi.BoolTy:
		return strconv.ParseBool(param)
	default:
		return nil, fmt.Errorf("unsupported ABI type: %s", typ.String())
	}
}

func (k *Keeper) executeSchedule(ctx sdk.Context, schedule types.Schedule) error {
	ownerAddress := common.HexToAddress(schedule.OwnerAddress)
	contractAddress := common.HexToAddress(schedule.ContractAddress)

	// Load ABI
	contractABI, err := abi.JSON(strings.NewReader(schedule.AbiJson))
	if err != nil {
		k.Logger(ctx).Error("Invalid ABI JSON", "schedule_id", schedule.Id, "error", err)
		return err
	}

	// Parse parameters (you need to implement ParseParams to correctly parse strings to ABI arguments)
	parsedParams, err := ParseParams(contractABI, schedule.MethodName, schedule.Params)
	if err != nil {
		return fmt.Errorf("failed to parse params: %w", err)
	}

	// Pack the call data
	callData, err := contractABI.Pack(schedule.MethodName, parsedParams...)
	if err != nil {
		k.Logger(ctx).Error("ABI packing failed", "schedule_id", schedule.Id, "error", err)
		return err
	}

	// Execute the call using EVM Keeper
	_, err = k.evmKeeper.CallEVMWithData(ctx, ownerAddress, &contractAddress, callData, true)
	if err != nil {
		k.Logger(ctx).Error("EVM execution failed", "schedule_id", schedule.Id, "error", err)
		return err
	}

	// Update the next execution block after successful execution
	schedule.NextExecutionBlock = uint64(ctx.BlockHeight()) + schedule.Frequency
	k.StoreSchedule(ctx, schedule)

	return nil
}

func (k *Keeper) StoreSchedule(ctx sdk.Context, schedule types.Schedule) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	bz := k.cdc.MustMarshal(&schedule)
	store.Set(GetScheduleIDBytes(schedule.Id), bz)
}

func (k *Keeper) removeSchedule(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	store.Delete(GetScheduleIDBytes(id))
}

func (k *Keeper) scheduleExists(ctx sdk.Context, id uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)
	return store.Has(GetScheduleIDBytes(id))
}

func (k *Keeper) changeTotalCount(ctx sdk.Context, increment int32) {
	store := ctx.KVStore(k.storeKey)
	count := k.getScheduleCount(ctx) + increment
	store.Set(types.ScheduleCountKey, sdk.Uint64ToBigEndian(uint64(count)))
}

func (k *Keeper) getScheduleCount(ctx sdk.Context) int32 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ScheduleCountKey)
	if bz == nil {
		return 0
	}
	return int32(sdk.BigEndianToUint64(bz))
}

func GetScheduleIDBytes(id uint64) []byte {
	return sdk.Uint64ToBigEndian(id)
}

func recordExecutedSchedule(err error, schedule types.Schedule) {
	telemetry.IncrCounterWithLabels([]string{types.LabelScheduleExecutionsCount}, 1, []metrics.Label{
		telemetry.NewLabel(telemetry.MetricLabelNameModule, types.ModuleName),
		telemetry.NewLabel(types.MetricLabelSuccess, strconv.FormatBool(err == nil)),
		telemetry.NewLabel(types.MetricLabelScheduleName, strconv.FormatUint(schedule.Id, 10)),
	})
}

// GetNextScheduleID returns a new unique schedule ID
func (k *Keeper) GetNextScheduleID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	// Get the current next ID
	bz := store.Get(types.NextScheduleIDKey)

	// If no ID exists yet, start from 1
	var id uint64 = 1
	if bz != nil {
		id = sdk.BigEndianToUint64(bz)
		// Increment for the next call
		store.Set(types.NextScheduleIDKey, sdk.Uint64ToBigEndian(id+1))
	} else {
		// First time, store 2 as the next ID
		store.Set(types.NextScheduleIDKey, sdk.Uint64ToBigEndian(2))
	}

	return id
}

// Add this to your keeper.go file

// GetAllSchedules returns all schedules
func (k Keeper) GetAllSchedules(ctx sdk.Context) []types.Schedule {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ScheduleKey)

	var schedules []types.Schedule
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var schedule types.Schedule
		k.cdc.MustUnmarshal(iterator.Value(), &schedule)
		schedules = append(schedules, schedule)
	}

	return schedules
}
