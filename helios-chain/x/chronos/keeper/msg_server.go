package keeper

import (
	"context"
	"fmt"

	cmn "helios-core/helios-chain/precompiles/common"

	"github.com/cometbft/cometbft/crypto"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"helios-core/helios-chain/x/chronos/types"

	errors "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

type msgServer struct {
	keeper Keeper
}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// CreateCron create a new cron
func (k msgServer) CreateCron(goCtx context.Context, req *types.MsgCreateCron) (*types.MsgCreateCronResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, fmt.Sprintf("only the owner can create a cron %s != %s", req.OwnerAddress, req.Sender))
	}

	newID := k.keeper.StoreGetNextCronID(ctx)

	if k.keeper.StoreCronExists(ctx, newID) { // impossible
		return nil, fmt.Errorf("cron already exists with id=%d", newID)
	}

	amount := req.AmountToDeposit.BigInt() // ahelios

	// check Balance of OwnerAddress
	account := k.keeper.EvmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(req.OwnerAddress))

	contractAccount := k.keeper.EvmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(req.ContractAddress))
	contractCode := k.keeper.EvmKeeper.GetCode(ctx, common.BytesToHash(contractAccount.CodeHash))
	if len(contractCode) == 0 {
		return nil, errors.Wrap(errortypes.ErrInvalidRequest, "The specified address is not a smart contract")
	}

	balance := sdkmath.NewIntFromBigInt(account.Balance)

	if balance.IsNegative() || balance.BigInt().Cmp(amount) < 0 {
		return nil, errors.Wrapf(errortypes.ErrInsufficientFunds, fmt.Sprintf("Balance too low: %d (balance) < %d (amountToDeposit)", balance.BigInt(), amount))
	}

	cronAddress := sdk.AccAddress(crypto.AddressHash([]byte(fmt.Sprintf("cron_%d", newID)))) // Générer une adresse unique basée sur cronId
	acc := k.keeper.accountKeeper.NewAccountWithAddress(ctx, cronAddress)
	k.keeper.accountKeeper.SetAccount(ctx, acc)

	// initiate value
	totalFeesPaid := sdkmath.NewInt(0)

	newCron := types.Cron{
		Id:                        newID,
		Address:                   cronAddress.String(),
		OwnerAddress:              req.OwnerAddress,
		ContractAddress:           req.ContractAddress,
		AbiJson:                   req.AbiJson,
		MethodName:                req.MethodName,
		Params:                    req.Params,
		Frequency:                 req.Frequency,
		NextExecutionBlock:        uint64(ctx.BlockHeight()) + req.Frequency,
		ExpirationBlock:           req.ExpirationBlock,
		GasLimit:                  req.GasLimit,
		MaxGasPrice:               req.MaxGasPrice,
		TotalExecutedTransactions: 0,
		TotalFeesPaid:             &totalFeesPaid,
		CronType:                  types.LEGACY_CRON,
	}

	if err := k.keeper.CronInTransfer(ctx, newCron, amount); err != nil {
		return nil, fmt.Errorf("initial transfer failed amount=%s", hexutil.EncodeBig(amount))
	}

	k.keeper.AddCron(ctx, newCron)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"CreateCron",
			sdk.NewAttribute("cron_id", fmt.Sprintf("%d", newCron.Id)),
			sdk.NewAttribute("owner_address", req.OwnerAddress),
			sdk.NewAttribute("contract_address", req.ContractAddress),
			sdk.NewAttribute("method_name", req.MethodName),
		),
	)

	return &types.MsgCreateCronResponse{
		CronId:      newCron.Id,
		CronAddress: cmn.AnyToHexAddress(newCron.Address).String(),
	}, nil
}

// UpdateCron modifies an existing cron
func (k msgServer) UpdateCron(goCtx context.Context, req *types.MsgUpdateCron) (*types.MsgUpdateCronResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	cron, found := k.keeper.GetCron(ctx, req.CronId)
	if !found {
		return nil, errors.Wrapf(errortypes.ErrNotFound, "cron %d not found", req.CronId)
	}

	if cron.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, "only the owner can edit")
	}
	cron.Frequency = req.NewFrequency
	cron.Params = req.NewParams
	cron.ExpirationBlock = req.NewExpirationBlock
	cron.GasLimit = req.NewGasLimit
	cron.MaxGasPrice = req.NewMaxGasPrice

	k.keeper.StoreSetCron(ctx, cron)

	return &types.MsgUpdateCronResponse{Success: true}, nil
}

// CancelCron cancels a cron
func (k msgServer) CancelCron(goCtx context.Context, req *types.MsgCancelCron) (*types.MsgCancelCronResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	schedule, found := k.keeper.GetCron(ctx, req.CronId)
	if !found {
		return nil, errors.Wrapf(errortypes.ErrNotFound, "cron %d not found", req.CronId)
	}

	if schedule.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, "only owner can cancel the schedule")
	}

	if err := k.keeper.RemoveCron(ctx, req.CronId, sdk.MustAccAddressFromBech32(req.OwnerAddress)); err != nil {
		return nil, errors.Wrap(err, "failed to remove schedule")
	}

	return &types.MsgCancelCronResponse{Success: true}, nil
}

func (k msgServer) CreateCallBackConditionedCron(goCtx context.Context, req *types.MsgCreateCallBackConditionedCron) (*types.MsgCreateCallBackConditionedCronResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.OwnerAddress != req.Sender {
		return nil, errors.Wrap(errortypes.ErrUnauthorized, fmt.Sprintf("only the owner can create a cron %s != %s", req.OwnerAddress, req.Sender))
	}

	newID := k.keeper.StoreGetNextCronID(ctx)

	if k.keeper.StoreCronExists(ctx, newID) { // impossible
		return nil, fmt.Errorf("cron already exists with id=%d", newID)
	}

	amount := req.AmountToDeposit.BigInt() // ahelios

	// check Balance of OwnerAddress
	account := k.keeper.EvmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(req.OwnerAddress))
	balance := sdkmath.NewIntFromBigInt(account.Balance)

	if balance.IsNegative() || balance.BigInt().Cmp(amount) < 0 {
		return nil, errors.Wrapf(errortypes.ErrInsufficientFunds, fmt.Sprintf("Balance too low: %d (balance) < %d (amountToDeposit)", balance.BigInt(), amount))
	}

	cronAddress := sdk.AccAddress(crypto.AddressHash([]byte(fmt.Sprintf("cron_%d", newID)))) // Générer une adresse unique basée sur cronId
	acc := k.keeper.accountKeeper.NewAccountWithAddress(ctx, cronAddress)
	k.keeper.accountKeeper.SetAccount(ctx, acc)

	// initiate value
	totalFeesPaid := sdkmath.NewInt(0)

	newCron := types.Cron{
		Id:                        newID,
		Address:                   cronAddress.String(),
		OwnerAddress:              req.OwnerAddress,
		ContractAddress:           req.ContractAddress,
		AbiJson:                   `[ { "inputs": [ { "internalType": "bytes", "name": "data", "type": "bytes" }, { "internalType": "bytes", "name": "error", "type": "bytes" } ], "name": "` + req.MethodName + `", "outputs": [], "payable": false, "stateMutability": "nonpayable", "type": "function" } ]`,
		MethodName:                req.MethodName,
		Params:                    []string{},
		Frequency:                 0,
		NextExecutionBlock:        uint64(ctx.BlockHeight()),
		ExpirationBlock:           req.ExpirationBlock,
		GasLimit:                  req.GasLimit,
		MaxGasPrice:               req.MaxGasPrice,
		TotalExecutedTransactions: 0,
		TotalFeesPaid:             &totalFeesPaid,
		CronType:                  types.CALLBACK_CONDITIONED_CRON,
	}

	if err := k.keeper.CronInTransfer(ctx, newCron, amount); err != nil {
		return nil, fmt.Errorf("initial transfer failed amount=%s", hexutil.EncodeBig(amount))
	}

	k.keeper.AddCron(ctx, newCron)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"CreateCron",
			sdk.NewAttribute("cron_id", fmt.Sprintf("%d", newCron.Id)),
			sdk.NewAttribute("owner_address", req.OwnerAddress),
			sdk.NewAttribute("contract_address", req.ContractAddress),
			sdk.NewAttribute("method_name", req.MethodName),
		),
	)

	return &types.MsgCreateCallBackConditionedCronResponse{
		CronId:      newCron.Id,
		CronAddress: cmn.AnyToHexAddress(newCron.Address).String(),
	}, nil
}
