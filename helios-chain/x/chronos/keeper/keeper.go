package keeper

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/log"
	"github.com/hashicorp/go-metrics"

	cmn "helios-core/helios-chain/precompiles/common"
	rpctypes "helios-core/helios-chain/rpc/types"
	"helios-core/helios-chain/testnet"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core"

	types "helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	errors "cosmossdk.io/errors"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/crypto"
)

type Keeper struct {
	cdc      codec.BinaryCodec
	storeKey storetypes.StoreKey
	memKey   storetypes.StoreKey

	accountKeeper types.AccountKeeper
	EvmKeeper     types.EVMKeeper
	bankKeeper    bankkeeper.Keeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	evmKeeper types.EVMKeeper,
	bankKeeper bankkeeper.Keeper,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		accountKeeper: accountKeeper,
		EvmKeeper:     evmKeeper,
		bankKeeper:    bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) ExecuteCrons(ctx sdk.Context, batchFees *types.BatchFeesWithIds) {
	telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), types.LabelExecuteReadyCrons)

	for _, cronId := range batchFees.Ids {
		cron, ok := k.GetCron(ctx, cronId)
		if !ok {
			continue
		}
		err := k.executeCron(ctx, cron)
		if err != nil {
			k.Logger(ctx).Info("Cron Executed With Error", "err", err)
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}
		recordExecutedCron(err, cron)
	}
}

func (k *Keeper) PushReadyCronsToQueue(ctx sdk.Context) uint64 {
	crons := k.getCronsReadyForExecutionWithFilter(ctx)
	count := 0
	for _, cron := range crons {
		if cron.CronType == types.CALLBACK_CONDITIONED_CRON { // they are added in queue by the keeper Hyperion or other
			continue
		}
		if k.ExistsInCronQueue(ctx, cron) {
			continue
		}
		// check if the cron has enough balance to pay the fees
		balance := k.CronBalance(ctx, cron)
		if balance.IsNegative() || balance.BigInt().Cmp(big.NewInt(0)) < 0 {
			k.Logger(ctx).Info("Cron has not enough balance to pay the fees", "cronId", cron.Id, "balance", balance.BigInt())
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}
		if balance.BigInt().Cmp(sdkmath.NewIntFromBigInt(cron.MaxGasPrice.BigInt()).Mul(sdkmath.NewInt(int64(cron.GasLimit))).BigInt()) <= 0 {
			k.Logger(ctx).Info("Cron has not enough balance to pay the fees", "cronId", cron.Id, "balance", balance.BigInt())
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}
		// check if cron has enough balance to pay the fees in current block
		k.AppendToCronQueue(ctx, cron)
		count++
	}
	return uint64(count)
}

func (k *Keeper) DeductFeesActivesCrons(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	decUtils, err := NewMonoDecoratorUtils(ctx, k.EvmKeeper)
	if err != nil {
		return err
	}
	baseDenom := evmtypes.GetEVMCoinDenom()
	crons := k.GetAllCrons(ctx)

	// cost of keep active one cron = 100 gas at 1Gwei (it's equals to 0.0000001 ahelios)
	gasLimit := params.CronActiveGasCostPerBlock // 100 Gas by Default
	gasPrice := decUtils.BaseFee                 // (example: big.NewInt(1000000000) == 1 Gwei

	// one active day for one cron at 1Gwei GasPrice = 0.001728 ahelios
	tx := ethtypes.NewTransaction(0, common.Address{}, big.NewInt(0), gasLimit, gasPrice, []byte{})
	fees := sdk.Coins{{Denom: baseDenom, Amount: sdkmath.NewIntFromBigInt(tx.Cost())}}

	k.SetTotalCronCount(ctx, uint64(len(crons)))

	for _, cron := range crons {
		balance := k.CronBalance(ctx, cron)
		cost := tx.Cost()

		if cron.ExpirationBlock != 0 && cron.ExpirationBlock <= uint64(ctx.BlockHeight()) {
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}

		if cron.CronType == types.CALLBACK_CONDITIONED_CRON { // it's free for callback conditioned crons
			continue
		}

		if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
			if cron.NextExecutionBlock < uint64(ctx.BlockHeight())-100 { // if the cron is not executed in the last 100 blocks, remove it
				k.emitCronCancelledEvent(ctx, cron)
				k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
				continue
			}
		}

		if balance.IsNegative() || balance.BigInt().Cmp(cost) < 0 {
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}
		err := k.ConsumeFeesAndEmitEvent(ctx, fees, sdk.MustAccAddressFromBech32(cron.Address), cron) // deduct 100 Gas mult BaseFee
		if err != nil {
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}

		// determine if the cron as enough balance to pay the max fees of execution in current block
		maxCost := cron.MaxGasPrice.Mul(sdkmath.NewInt(int64(cron.GasLimit)))
		if balance.BigInt().Cmp(maxCost.BigInt()) < 0 {
			k.emitCronCancelledEvent(ctx, cron)
			k.RemoveCron(ctx, cron.Id, sdk.MustAccAddressFromBech32(cron.OwnerAddress))
			continue
		}

		// update cron
		k.UpdateCronTotalFeesPaid(ctx, cron, fees[0].Amount)
	}

	return nil
}

func (k *Keeper) UpdateCronTotalFeesPaid(ctx sdk.Context, cron types.Cron, fees sdkmath.Int) {
	newTotalFeesPaid := cron.TotalFeesPaid.Add(fees)
	cron.TotalFeesPaid = &newTotalFeesPaid
	k.StoreSetCron(ctx, cron)
}

func (k *Keeper) CronInTransfer(ctx sdk.Context, cron types.Cron, amount *big.Int) error {
	account := k.EvmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(cron.OwnerAddress))
	balance := sdkmath.NewIntFromBigInt(account.Balance)

	if balance.IsNegative() && balance.BigInt().Cmp(amount) < 0 {
		return errors.Wrapf(errortypes.ErrInsufficientFunds, "cronTransfer")
	}

	if !balance.IsNegative() && balance.BigInt().Cmp(big.NewInt(0)) > 0 {
		err := k.bankKeeper.SendCoins(ctx,
			sdk.MustAccAddressFromBech32(cron.OwnerAddress),
			sdk.MustAccAddressFromBech32(cron.Address),
			sdk.NewCoins(sdk.NewCoin(evmtypes.GetEVMCoinDenom(), sdkmath.NewIntFromBigInt(amount))),
		)
		if err != nil {
			return errors.Wrapf(errortypes.ErrInsufficientFunds, err.Error())
		}
	}
	return nil
}

func (k *Keeper) CronOutTransfer(ctx sdk.Context, cron types.Cron, amount *big.Int) error {
	balance := k.CronBalance(ctx, cron)

	if balance.IsNegative() && balance.BigInt().Cmp(amount) < 0 {
		return errors.Wrapf(errortypes.ErrInsufficientFunds, "cronTransfer")
	}

	if !balance.IsNegative() && balance.BigInt().Cmp(big.NewInt(0)) > 0 {
		err := k.bankKeeper.SendCoins(ctx,
			sdk.MustAccAddressFromBech32(cron.Address),
			sdk.MustAccAddressFromBech32(cron.OwnerAddress),
			sdk.NewCoins(sdk.NewCoin(evmtypes.GetEVMCoinDenom(), sdkmath.NewIntFromBigInt(amount))),
		)
		if err != nil {
			return errors.Wrapf(errortypes.ErrInsufficientFunds, err.Error())
		}
	}
	return nil
}

func (k *Keeper) CronBalance(ctx sdk.Context, cron types.Cron) sdkmath.Int {
	account := k.EvmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(cron.Address))
	if account == nil {
		return sdkmath.ZeroInt()
	}
	balance := sdkmath.NewIntFromBigInt(account.Balance)

	return balance
}

func (k *Keeper) AddCron(ctx sdk.Context, cron types.Cron) {
	k.StoreSetCron(ctx, cron)
	k.StoreSetCronAddress(ctx, cron)
	k.StoreChangeTotalCount(ctx, 1)
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_1 < int64(ctx.BlockHeight()) {
		k.StoreCronIndexByOwnerAddress(ctx, cron)
	}
}

func (k *Keeper) RemoveCron(ctx sdk.Context, id uint64, owner sdk.AccAddress) error {
	cron, found := k.GetCron(ctx, id)
	if !found {
		return fmt.Errorf("cron not found")
	}
	if cron.OwnerAddress != owner.String() {
		return fmt.Errorf("unauthorized removal")
	}

	// send back cron wallet funds
	balance := k.CronBalance(ctx, cron)

	if !balance.IsNegative() && balance.BigInt().Cmp(big.NewInt(0)) > 0 {
		err := k.CronOutTransfer(ctx, cron, balance.BigInt())
		if err != nil {
			return err
		}
		refundedCount := k.GetCronRefundedLastBlockCount(ctx)
		k.StoreChangeCronRefundedLastBlockTotalCount(ctx, refundedCount+1)
	}

	if k.ExistsInCronQueue(ctx, cron) {
		k.RemoveFromCronQueue(ctx, cron)
	}

	k.StoreRemoveCron(ctx, cron.Id)
	k.StoreArchiveCron(ctx, cron)
	k.StoreChangeTotalCount(ctx, -1)
	k.StoreChangeArchivedTotalCount(ctx, 1)

	return nil
}

func (k *Keeper) GetCronOrArchivedCron(ctx sdk.Context, id uint64) (types.Cron, bool) {
	cron, ok := k.GetCron(ctx, id)
	if ok {
		return cron, true
	}
	return k.GetArchivedCron(ctx, id)
}

func (k *Keeper) GetCron(ctx sdk.Context, id uint64) (types.Cron, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)
	bz := store.Get(GetCronIDBytes(id))
	if bz == nil {
		return types.Cron{}, false
	}

	var cron types.Cron
	k.cdc.MustUnmarshal(bz, &cron)
	return cron, true
}

func (k *Keeper) getCronsReadyForExecution(ctx sdk.Context) []types.Cron {
	params := k.GetParams(ctx)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)
	var crons []types.Cron
	count := uint64(0)

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	currentBlock := uint64(ctx.BlockHeight())
	for ; iterator.Valid(); iterator.Next() {
		var cron types.Cron
		k.cdc.MustUnmarshal(iterator.Value(), &cron)

		if currentBlock >= cron.NextExecutionBlock &&
			(cron.ExpirationBlock == 0 || currentBlock <= cron.ExpirationBlock) {
			crons = append(crons, cron)
			count++
			if count >= params.ExecutionsLimitPerBlock {
				k.Logger(ctx).Info("Reached execution limit for the block")
				break
			}
		}
	}

	sort.Slice(crons, func(i, j int) bool {
		return crons[i].Id < crons[j].Id
	})

	return crons
}

func (k *Keeper) getCronsReadyForExecutionWithFilter(ctx sdk.Context) []types.Cron {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)
	var crons []types.Cron

	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	currentBlock := uint64(ctx.BlockHeight())
	for ; iterator.Valid(); iterator.Next() {
		var cron types.Cron
		k.cdc.MustUnmarshal(iterator.Value(), &cron)

		if testnet.TESTNET_BLOCK_NUMBER_UPDATE_2 < int64(ctx.BlockHeight()) {
			if currentBlock == cron.NextExecutionBlock &&
				(cron.ExpirationBlock == 0 || currentBlock <= cron.ExpirationBlock) {
				crons = append(crons, cron)
			}
		} else {
			if currentBlock >= cron.NextExecutionBlock &&
				(cron.ExpirationBlock == 0 || currentBlock <= cron.ExpirationBlock) {
				crons = append(crons, cron)
			}
		}
	}

	return crons
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
	case abi.BytesTy:
		return hexutil.Decode(param)
	default:
		return nil, fmt.Errorf("unsupported ABI type: %s", typ.String())
	}
}

// NewMonoDecoratorUtils returns a new DecoratorUtils instance.
//
// These utilities are extracted once at the beginning of the ante handle process,
// and are used throughout the entire decorator chain.
// This avoids redundant calls to the keeper and thus improves speed of transaction processing.
// All prices, fees and balances are converted into 18 decimals here to be
// correctly used in the EVM.
func NewMonoDecoratorUtils(
	ctx sdk.Context,
	ek types.EVMKeeper,
) (*types.DecoratorUtils, error) {
	evmParams := ek.GetParams(ctx)
	ethCfg := evmtypes.GetEthChainConfig()
	blockHeight := big.NewInt(ctx.BlockHeight())

	rules := ethCfg.Rules(blockHeight, true)
	baseFee := ek.GetBaseFee(ctx)
	baseDenom := evmtypes.GetEVMCoinDenom()

	if rules.IsLondon && baseFee == nil {
		return nil, errors.Wrap(
			evmtypes.ErrInvalidBaseFee,
			"base fee is supported but evm block context value is nil",
		)
	}

	// get the gas prices adapted accordingly
	// to the evm denom decimals
	globalMinGasPrice := ek.GetMinGasPrice(ctx)

	// Mempool gas price should be scaled to the 18 decimals representation. If
	// it is already a 18 decimal token, this is a no-op.
	mempoolMinGasPrice := evmtypes.ConvertAmountTo18DecimalsLegacy(ctx.MinGasPrices().AmountOf(baseDenom))
	return &types.DecoratorUtils{
		EvmParams:          evmParams,
		Rules:              rules,
		Signer:             ethtypes.MakeSigner(ethCfg, blockHeight),
		BaseFee:            baseFee,
		MempoolMinGasPrice: mempoolMinGasPrice,
		GlobalMinGasPrice:  globalMinGasPrice,
		BlockTxIndex:       ek.GetTxIndexTransient(ctx),
		GasWanted:          0,
		MinPriority:        int64(math.MaxInt64),
		// TxGasLimit and TxFee are set to zero because they are updated
		// summing up the values of all messages contained in a tx.
		TxGasLimit: 0,
		TxFee:      new(big.Int),
	}, nil
}

func (k *Keeper) GetCronTransaction(ctx sdk.Context, cron types.Cron, nonce uint64) (*ethtypes.Transaction, error) {
	contractAddress := cmn.AnyToHexAddress(cron.ContractAddress)

	// Load ABI
	contractABI, err := abi.JSON(strings.NewReader(cron.AbiJson))
	if err != nil {
		k.Logger(ctx).Error("Invalid ABI JSON", "cron_id", cron.Id, "error", err)
		return nil, err
	}

	// Parse parameters (you need to implement ParseParams to correctly parse strings to ABI arguments)
	parsedParams, err := ParseParams(contractABI, cron.MethodName, cron.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to parse params: %w", err)
	}

	k.Logger(ctx).Info("ABI packing", "cron_id", cron.Id, "cron.MethodName", cron.MethodName, "parsedParams", parsedParams)
	// Pack the call data
	callData, err := contractABI.Pack(cron.MethodName, parsedParams...)
	if err != nil {
		k.Logger(ctx).Error("ABI packing failed", "cron_id", cron.Id, "error", err)
		return nil, err
	}

	// get BaseFee
	decUtils, err := NewMonoDecoratorUtils(ctx, k.EvmKeeper)
	if err != nil {
		return nil, err
	}

	// check acceptence in terms of fees
	if cron.MaxGasPrice.BigInt().Cmp(decUtils.BaseFee) < 0 {
		return nil, fmt.Errorf("max gas price too low: %d (max gas price) < %d (base fee)", cron.MaxGasPrice, decUtils.BaseFee)
	}

	gasLimit := cron.GasLimit    // Limite de gaz (si gas inferieur au montant consommé potentiel "intrinsic gas too low" ErrIntrinsicGas)
	gasPrice := decUtils.BaseFee // Prix du gaz (20 Gwei)
	toAddress := contractAddress // Adresse du destinataire
	value := big.NewInt(0)       // Montant à envoyer (0 ahelios)
	data := callData             // Données de la transaction (vide pour une simple transaction)

	// Créer la transaction
	tx := ethtypes.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	return tx, nil
}

func VerifyFee(
	txData evmtypes.TxData,
	denom string,
	baseFee *big.Int,
	homestead, istanbul, isCheckTx bool,
) (sdk.Coins, error) {
	isContractCreation := txData.GetTo() == nil

	gasLimit := txData.GetGas()

	var accessList ethtypes.AccessList
	if txData.GetAccessList() != nil {
		accessList = txData.GetAccessList()
	}

	intrinsicGas, err := core.IntrinsicGas(txData.GetData(), accessList, isContractCreation, homestead, istanbul)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to retrieve intrinsic gas, contract creation = %t; homestead = %t, istanbul = %t",
			isContractCreation, homestead, istanbul,
		)
	}

	// intrinsic gas verification during CheckTx
	if isCheckTx && gasLimit < intrinsicGas {
		return nil, errors.Wrapf(
			errortypes.ErrOutOfGas,
			"gas limit too low: %d (gas limit) < %d (intrinsic gas)", gasLimit, intrinsicGas,
		)
	}

	if baseFee != nil && txData.GetGasFeeCap().Cmp(baseFee) < 0 {
		return nil, errors.Wrapf(errortypes.ErrInsufficientFee,
			"the tx gasfeecap is lower than the tx baseFee: %s (gasfeecap), %s (basefee) ",
			txData.GetGasFeeCap(),
			baseFee)
	}

	feeAmt := txData.EffectiveFee(baseFee)
	if feeAmt.Sign() == 0 {
		// zero fee, no need to deduct
		return sdk.Coins{}, nil
	}

	return sdk.Coins{{Denom: denom, Amount: sdkmath.NewIntFromBigInt(feeAmt)}}, nil
}

func (k *Keeper) ConsumeFeesAndEmitEvent(
	ctx sdk.Context,
	fees sdk.Coins,
	from sdk.AccAddress,
	cron types.Cron,
) error {
	if err := k.deductFees(
		ctx,
		fees,
		from,
		cron,
	); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeTx,
			sdk.NewAttribute(sdk.AttributeKeyFee, fees.String()),
		),
	)
	return nil
}

func (k *Keeper) deductFees(
	ctx sdk.Context,
	fees sdk.Coins,
	feePayer sdk.AccAddress,
	cron types.Cron,
) error {
	if fees.IsZero() {
		return nil
	}

	k.Logger(ctx).Debug("DeductFees", "fees", fees, "feePayer", cmn.AnyToHexAddress(feePayer.String()), "cronId", cron.Id)

	if err := k.EvmKeeper.DeductTxCostsFromUserBalance(
		ctx,
		fees,
		common.BytesToAddress(feePayer),
	); err != nil {
		return errors.Wrapf(err, "failed to deduct transaction costs from user balance")
	}

	return nil
}

func (k *Keeper) TxAsMessage(tx *ethtypes.Transaction, baseFee *big.Int, from common.Address) ethtypes.Message {

	gasFeeCap := new(big.Int).Set(tx.GasFeeCap())
	gasTipCap := new(big.Int).Set(tx.GasTipCap())
	gasPrice := new(big.Int).Set(tx.GasPrice())
	if baseFee != nil {
		gasPrice = ethmath.BigMin(gasPrice.Add(gasTipCap, baseFee), gasFeeCap)
	}

	msg := ethtypes.NewMessage(
		from,
		tx.To(),
		tx.Nonce(),
		tx.Value(),
		tx.Gas(),
		gasPrice,
		gasFeeCap,
		gasTipCap,
		tx.Data(),
		tx.AccessList(),
		false,
	)
	return msg
}

func (k *Keeper) executeCronEvm(ctx sdk.Context, cron types.Cron, tx *ethtypes.Transaction) (*evmtypes.MsgEthereumTxResponse, error) {
	ownerAddress := cmn.AnyToHexAddress(cron.OwnerAddress)
	baseDenom := evmtypes.GetEVMCoinDenom()

	account := k.EvmKeeper.GetAccount(ctx, ownerAddress)
	balance := sdkmath.NewIntFromBigInt(account.Balance)
	cost := tx.Cost()

	if balance.IsNegative() || balance.BigInt().Cmp(cost) < 0 {
		return nil, errors.Wrapf(
			errortypes.ErrInsufficientFunds,
			"sender balance < tx cost (%s < %s)", balance, cost,
		)
	}

	// 1. get BaseFee
	decUtils, err := NewMonoDecoratorUtils(ctx, k.EvmKeeper)
	if err != nil {
		return nil, err
	}
	// 2. format DataTx for calculating gasConsumption
	txData, err := evmtypes.NewTxDataFromTx(tx)
	if err != nil {
		return nil, err
	}
	// 3. calculating gasConsumption
	msgFees, err := VerifyFee(
		txData,
		baseDenom,
		decUtils.BaseFee,
		decUtils.Rules.IsHomestead,
		decUtils.Rules.IsIstanbul,
		true,
	)
	if err != nil {
		return nil, err
	}
	// 5. Consume Fees on cron Wallet
	err = k.ConsumeFeesAndEmitEvent(
		ctx,
		msgFees,
		sdk.MustAccAddressFromBech32(cron.Address),
		cron,
	)
	if err != nil {
		return nil, err
	}
	// update cron fees paid
	k.UpdateCronTotalFeesPaid(ctx, cron, msgFees[0].Amount)
	// 5. prepare tx for evm
	msg := k.TxAsMessage(tx, decUtils.BaseFee, ownerAddress)
	// 6. execute tx without commit
	_, err = k.EvmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), false)
	if err != nil {
		k.Logger(ctx).Error("EVM execution estimate failed", "cron_id", cron.Id, "error", err)
		return nil, err
	}
	// 7. execute tx
	res, err := k.EvmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), true)
	if err != nil {
		k.Logger(ctx).Error("EVM execution failed", "cron_id", cron.Id, "error", err)
		return nil, err
	}
	// 7. refund gas in order to match the Ethereum gas consumption
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(msg.Gas()-res.GasUsed), msg.GasPrice())
	refundCoins := sdk.Coins{sdk.NewCoin(baseDenom, sdkmath.NewIntFromBigInt(remaining))}
	k.Logger(ctx).Info("RefundFees", "fees", refundCoins, "receiver", msg.From().String())

	msgForRefund := k.TxAsMessage(tx, nil, cmn.AnyToHexAddress(cron.Address))

	remainingR := new(big.Int).Mul(new(big.Int).SetUint64(msgForRefund.Gas()-res.GasUsed), msgForRefund.GasPrice())
	refundCoinsR := sdk.Coins{sdk.NewCoin(baseDenom, sdkmath.NewIntFromBigInt(remainingR))}
	k.Logger(ctx).Info("RefundFees", "fees", refundCoinsR, "receiver", msgForRefund.From().String())

	if err = k.EvmKeeper.RefundGas(ctx, msgForRefund, msgForRefund.Gas()-res.GasUsed, baseDenom); err != nil {
		return nil, errors.Wrapf(err, "failed to refund gas leftover gas to sender %s", msg.From())
	}
	if refundCoinsR[0].Amount.GT(sdkmath.NewInt(0)) { // update cron fees paid
		k.UpdateCronTotalFeesPaid(ctx, cron, refundCoinsR[0].Amount)
	}
	return res, nil
}

func (k *Keeper) executeCron(ctx sdk.Context, cron types.Cron) error {
	nonce := k.StoreGetNonce(ctx)

	if cron.CronType == types.CALLBACK_CONDITIONED_CRON {
		callBackData, ok := k.GetCronCallBackData(ctx, cron.Id)

		if !ok { // all ok
			return nil
		}
		// setup params and go to execution
		cron.Params = []string{
			hexutil.Encode(callBackData.Data),
			hexutil.Encode(callBackData.Error),
		}
		// set new expiration
		cron.ExpirationBlock = uint64(ctx.BlockHeight())
		k.StoreSetCron(ctx, cron)
	}

	tx, err := k.GetCronTransaction(ctx, cron, nonce)
	if err != nil {
		return err
	}

	bytesTx, err := tx.MarshalBinary()
	if err != nil {
		return err
	}

	ethereumTxCasted := &evmtypes.MsgEthereumTx{}
	if err := ethereumTxCasted.FromEthereumTx(tx); err != nil {
		k.Logger(ctx).Error("transaction converting failed", "error", err.Error())
		return err
	}

	// default res for accepted errors
	res := &evmtypes.MsgEthereumTxResponse{
		Hash:    tx.Hash().Hex(),
		Logs:    []*evmtypes.Log{},
		Ret:     []byte{},
		VmError: "",
		GasUsed: 0,
	}

	///////////////////////////////////////////////////
	// Execution can be processed
	///////////////////////////////////////////////////
	resFromExecution, err := k.executeCronEvm(ctx, cron, tx)
	if err != nil {
		res.VmError = err.Error()
	} else {
		res = resFromExecution
	}

	// add txHash on logs
	for _, log := range res.Logs {
		log.TxHash = tx.Hash().Hex()
	}
	bytesRes, _ := res.Marshal()
	cronTxResult := types.CronTransactionResult{
		BlockHash:   hexutil.Encode(ctx.HeaderHash()),
		BlockNumber: uint64(ctx.BlockHeight()),
		TxHash:      tx.Hash().Hex(),
		Tx:          bytesTx,
		Result:      bytesRes,
		Nonce:       nonce,
		From:        cmn.AnyToHexAddress(cron.OwnerAddress).String(),
		CronId:      cron.Id,
		CronAddress: cmn.AnyToHexAddress(cron.Address).String(),
	}

	// Update the next execution block after successful execution
	cron.NextExecutionBlock = uint64(ctx.BlockHeight()) + cron.Frequency
	cron.TotalExecutedTransactions += 1

	k.StoreSetCron(ctx, cron)
	k.StoreCronTransactionResult(ctx, cron, cronTxResult)
	k.StoreSetTransactionNonceByHash(ctx, tx.Hash().Hex(), nonce)
	k.StoreSetTransactionHashInBlock(ctx, cronTxResult.BlockNumber, tx.Hash().Hex())
	k.StoreSetNonce(ctx, nonce+1)
	k.StoreChangeCronExecutedLastBlockTotalCount(ctx, k.GetCronExecutedLastBlockCount(ctx)+1)
	return nil
}

func (k *Keeper) BuildCronCanceledEvent(ctx sdk.Context, a abi.ABI, tx *ethtypes.Transaction, from, to common.Address, cronId uint64, success bool) (*evmtypes.Log, error) {
	// Prepare the event topics
	event := a.Events["CronCancelled"]
	topics := make([]string, 3)

	// The first topic is always the signature of the event.
	topics[0] = event.ID.String()

	var err error
	tp1, err := cmn.MakeTopic(from) // index 1
	if err != nil {
		return nil, err
	}
	topics[1] = tp1.String()

	tp2, err := cmn.MakeTopic(to) // index 2
	if err != nil {
		return nil, err
	}
	topics[2] = tp2.String()

	arguments := abi.Arguments{event.Inputs[2], event.Inputs[3]} // cronId, success
	packed, err := arguments.Pack(cronId, success)
	if err != nil {
		return nil, err
	}

	return &evmtypes.Log{
		Address:     to.String(),
		Topics:      topics,
		Data:        packed,
		BlockNumber: uint64(ctx.BlockHeight()),
		TxHash:      tx.Hash().String(),
		TxIndex:     tx.Nonce(),
		Index:       tx.Nonce(),
	}, nil
}

func (k *Keeper) emitCronCancelledEvent(ctx sdk.Context, cron types.Cron) error {
	nonce := k.StoreGetNonce(ctx)

	abiStr := `[{ "anonymous": false, "inputs": [ { "indexed": true, "internalType": "address", "name": "fromAddress", "type": "address" }, { "indexed": true, "internalType": "address", "name": "toAddress", "type": "address" }, { "indexed": false, "internalType": "uint64", "name": "cronId", "type": "uint64" }, { "indexed": false, "internalType": "bool", "name": "success", "type": "bool" } ], "name": "CronCancelled", "type": "event" },{ "inputs": [ { "internalType": "uint64", "name": "cronId", "type": "uint64" } ], "name": "cancelCron", "outputs": [ { "internalType": "bool", "name": "success", "type": "bool" } ], "stateMutability": "nonpayable", "type": "function" }]`
	// Load ABI
	contractABI, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		k.Logger(ctx).Error("Invalid ABI JSON", "cron_id", cron.Id, "error", err)
		return err
	}

	// Pack the call data
	callData, err := contractABI.Pack("cancelCron", cron.Id)
	if err != nil {
		k.Logger(ctx).Error("ABI packing failed", "cron_id", cron.Id, "error", err)
		return err
	}

	tx := ethtypes.NewTransaction(
		nonce,
		common.HexToAddress("0x0000000000000000000000000000000000000830"),
		big.NewInt(0),
		0,
		&big.Int{},
		callData,
	)

	bytesTx, err := tx.MarshalBinary()
	if err != nil {
		return err
	}

	log, err := k.BuildCronCanceledEvent(ctx, contractABI, tx, cmn.AnyToHexAddress(cron.OwnerAddress), common.HexToAddress("0x0000000000000000000000000000000000000830"), cron.Id, true)
	if err != nil {
		return err
	}

	log.BlockNumber = uint64(ctx.BlockHeight())
	log.Removed = false
	log.BlockHash = hexutil.Encode(ctx.HeaderHash())
	res := &evmtypes.MsgEthereumTxResponse{
		Hash:    tx.Hash().Hex(),
		Logs:    []*evmtypes.Log{log},
		Ret:     []byte{},
		VmError: "",
		GasUsed: 0,
	}
	for _, log := range res.Logs {
		log.TxHash = tx.Hash().Hex()
	}
	bytesRes, _ := res.Marshal()

	cronTxResult := types.CronTransactionResult{
		BlockHash:   hexutil.Encode(ctx.HeaderHash()),
		BlockNumber: uint64(ctx.BlockHeight()),
		TxHash:      tx.Hash().Hex(),
		Tx:          bytesTx,
		Result:      bytesRes,
		Nonce:       nonce,
		From:        cmn.AnyToHexAddress(cron.OwnerAddress).String(),
		CronId:      cron.Id,
		CronAddress: cmn.AnyToHexAddress(cron.Address).String(),
	}

	k.StoreCronTransactionResult(ctx, cron, cronTxResult)
	k.StoreSetTransactionNonceByHash(ctx, tx.Hash().Hex(), nonce)
	k.StoreSetTransactionHashInBlock(ctx, cronTxResult.BlockNumber, tx.Hash().Hex())
	k.StoreSetNonce(ctx, nonce+1)

	return nil
}

func (k *Keeper) FormatCronTransactionResultToCronTransactionRPC(ctx sdk.Context, txResult types.CronTransactionResult) (*types.CronTransactionRPC, error) {
	var castedRes ethtypes.Transaction
	err := castedRes.UnmarshalBinary(txResult.Tx)
	if err != nil {
		k.Logger(ctx).Info("failed to unmarshal result", "err", err)
		return nil, fmt.Errorf("failed to unmarshal result")
	}
	ethereumTxCasted := &evmtypes.MsgEthereumTx{}
	if err := ethereumTxCasted.FromEthereumTx(&castedRes); err != nil {
		k.Logger(ctx).Error("transaction converting failed", "error", err.Error())
		return nil, fmt.Errorf("transaction converting failed")
	}
	ethCfg := evmtypes.GetEthChainConfig()

	height := uint64(txResult.BlockNumber)
	index := uint64(0)        // ou récupérer l'index de la transaction si disponible
	baseFee := big.NewInt(0)  // ou récupérer le baseFee du bloc si disponible
	chainID := ethCfg.ChainID // remplacer par votre chainID
	from := common.HexToAddress(txResult.From)

	// blockHash := ctx.HeaderHash()

	txToRPC, _ := rpctypes.NewUnsignedTransactionFromMsg(
		ethereumTxCasted,
		common.BytesToHash([]byte{}),
		height,
		index,
		baseFee,
		chainID,
		from,
	)

	rpcTxMap := &types.CronTransactionRPC{
		BlockHash:        txResult.BlockHash,
		BlockNumber:      hexutil.EncodeUint64(txResult.BlockNumber),
		ChainId:          hexutil.EncodeBig(chainID),
		From:             from.String(),
		Gas:              hexutil.EncodeUint64(uint64(txToRPC.Gas)),
		GasPrice:         hexutil.EncodeBig(txToRPC.GasPrice.ToInt()),
		Hash:             txToRPC.Hash.Hex(),
		Input:            hexutil.Encode(txToRPC.Input),
		Nonce:            hexutil.EncodeUint64(uint64(txToRPC.Nonce)),
		R:                "0x0",
		S:                "0x0",
		To:               txToRPC.To.String(),
		TransactionIndex: hexutil.Uint64(txResult.Nonce).String(),
		Type:             hexutil.EncodeUint64(uint64(2)),
		V:                "0x1",
		Value:            hexutil.EncodeBig(txToRPC.Value.ToInt()),
		CronId:           txResult.CronId,
		CronAddress:      txResult.CronAddress,
	}
	return rpcTxMap, nil
}

func (k *Keeper) GetCronTransactionByNonce(ctx sdk.Context, nonce uint64) (*types.CronTransactionRPC, error) {
	res, ok := k.GetCronTransactionResultByNonce(ctx, nonce)
	if !ok {
		k.Logger(ctx).Info("failed to load GetCronTransactionByNonce", "nonce", nonce)
		return nil, fmt.Errorf("nonce %d not found", nonce)
	}

	return k.FormatCronTransactionResultToCronTransactionRPC(ctx, res)
}

func (k *Keeper) GetCronTransactionByHash(ctx sdk.Context, hash string) (*types.CronTransactionRPC, error) {
	res, ok := k.GetCronTransactionResultByHash(ctx, hash)
	if !ok {
		k.Logger(ctx).Info("failed to load GetCronTransactionByHash", "hash", hash)
		return nil, fmt.Errorf("hash %s not found", hash)
	}

	return k.FormatCronTransactionResultToCronTransactionRPC(ctx, res)
}

func (k *Keeper) FormatCronTransactionResultToCronTransactionReceiptRPC(ctx sdk.Context, txResult types.CronTransactionResult) (*types.CronTransactionReceiptRPC, error) {
	var castedRes ethtypes.Transaction
	err := castedRes.UnmarshalBinary(txResult.Tx)
	if err != nil {
		k.Logger(ctx).Info("failed to unmarshal result", "err", err)
		return nil, fmt.Errorf("failed to unmarshal result")
	}
	ethereumTxCasted := &evmtypes.MsgEthereumTx{}
	if err := ethereumTxCasted.FromEthereumTx(&castedRes); err != nil {
		k.Logger(ctx).Error("transaction converting failed", "error", err.Error())
		return nil, fmt.Errorf("transaction converting failed")
	}

	var castedResponse evmtypes.MsgEthereumTxResponse
	err = castedResponse.Unmarshal(txResult.Result)
	if err != nil {
		k.Logger(ctx).Info("failed to unmarshal result", "err", err)
		return nil, fmt.Errorf("failed to unmarshal result")
	}

	var status hexutil.Uint
	if castedResponse.Failed() {
		status = hexutil.Uint(ethtypes.ReceiptStatusFailed)
	} else {
		status = hexutil.Uint(ethtypes.ReceiptStatusSuccessful)
	}

	from := common.HexToAddress(txResult.From)

	var ethLogs []*ethtypes.Log
	for _, log := range castedResponse.Logs {
		ethLogs = append(ethLogs, log.ToEthereum())
	}

	txData, err := evmtypes.UnpackTxData(ethereumTxCasted.Data)
	if err != nil {
		k.Logger(ctx).Info("failed to unpack tx data", "error", err.Error())
		return nil, fmt.Errorf("failed to unpack tx data")
	}

	receipt := types.CronTransactionReceiptRPC{
		// Consensus fields
		Status:            hexutil.Uint64(status).String(),
		CumulativeGasUsed: hexutil.Uint64(castedResponse.GasUsed).String(),
		LogsBloom:         hexutil.Encode(ethtypes.BytesToBloom(ethtypes.LogsBloom(ethLogs)).Bytes()),
		Logs:              castedResponse.Logs,

		// Implementation fields
		TransactionHash: ethereumTxCasted.Hash,
		GasUsed:         hexutil.Uint64(castedResponse.GasUsed).String(),

		// Inclusion information
		BlockHash:        txResult.BlockHash,
		BlockNumber:      hexutil.Uint64(txResult.BlockNumber).String(),
		TransactionIndex: hexutil.Uint64(txResult.Nonce).String(),
		Result:           hexutil.Encode(txResult.Result),

		// Addresses
		From: from.String(),
		To:   txData.GetTo().String(),
		Type: hexutil.Uint(uint64(2)).String(),

		// returns data
		Ret:         hexutil.Encode(castedResponse.Ret), // Ret is the bytes of call return
		VmError:     castedResponse.VmError,
		CronId:      txResult.CronId,
		CronAddress: txResult.CronAddress,
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if txData.GetTo() == nil {
		receipt.ContractAddress = crypto.CreateAddress(from, txData.GetNonce()).String()
	}

	return &receipt, nil
}

func (k *Keeper) GetTransactionReceipt(ctx sdk.Context, cron types.Cron, ethMsg *evmtypes.MsgEthereumTx, res *evmtypes.MsgEthereumTxResponse) (map[string]interface{}, error) {
	txData, err := evmtypes.UnpackTxData(ethMsg.Data)
	if err != nil {
		k.Logger(ctx).Info("failed to unpack tx data", "error", err.Error())
		return nil, err
	}

	cumulativeGasUsed := uint64(0)

	var status hexutil.Uint
	if res.Failed() {
		status = hexutil.Uint(ethtypes.ReceiptStatusFailed)
	} else {
		status = hexutil.Uint(ethtypes.ReceiptStatusSuccessful)
	}

	from := cmn.AnyToHexAddress(cron.OwnerAddress)

	var ethLogs []*ethtypes.Log
	for _, log := range res.Logs {
		ethLogs = append(ethLogs, log.ToEthereum())
	}

	logs := ethLogs

	receipt := map[string]interface{}{
		// Consensus fields: These fields are defined by the Yellow Paper
		"status":            status,
		"cumulativeGasUsed": hexutil.Uint64(cumulativeGasUsed),
		"logsBloom":         ethtypes.BytesToBloom(ethtypes.LogsBloom(logs)),
		"logs":              logs,

		// Implementation fields: These fields are added by geth when processing a transaction.
		// They are stored in the chain database.
		"transactionHash": ethMsg.Hash,
		"contractAddress": nil,
		"gasUsed":         res.GasUsed,

		// Inclusion information: These fields provide information about the inclusion of the
		// transaction corresponding to this receipt.
		"blockHash":        common.BytesToHash([]byte{}).Hex(),
		"blockNumber":      hexutil.Uint64(ctx.BlockHeight()), //nolint:gosec // G115
		"transactionIndex": hexutil.Uint64(0),                 //nolint:gosec // G115

		// sender and receiver (contract or EOA) addreses
		"from": from,
		"to":   txData.GetTo(),
		"type": hexutil.Uint(ethMsg.AsTransaction().Type()),
	}

	if logs == nil {
		receipt["logs"] = [][]*ethtypes.Log{}
	}

	// If the ContractAddress is 20 0x0 bytes, assume it is not a contract creation
	if txData.GetTo() == nil {
		receipt["contractAddress"] = crypto.CreateAddress(from, txData.GetNonce())
	}

	return receipt, nil
}

func (k *Keeper) StoreSetCron(ctx sdk.Context, cron types.Cron) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)
	bz := k.cdc.MustMarshal(&cron)
	store.Set(GetCronIDBytes(cron.Id), bz)
}

func (k *Keeper) StoreCronIndexByOwnerAddress(ctx sdk.Context, cron types.Cron) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronIndexByOwnerAddressKey)
	store.Set(append([]byte(cmn.AnyToHexAddress(cron.OwnerAddress).Hex()), GetCronIDBytes(cron.Id)...), []byte{})
}

func (k *Keeper) StoreSetCronAddress(ctx sdk.Context, cron types.Cron) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronAddressKey)
	store.Set([]byte(cron.Address), sdk.Uint64ToBigEndian(cron.Id))
}

// todo Store this in the cron attributes
func (k *Keeper) StoreCronCallBackData(ctx sdk.Context, cronId uint64, callbackData *types.CronCallBackData) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronCallBackDataKey)
	bz := k.cdc.MustMarshal(callbackData)
	store.Set(sdk.Uint64ToBigEndian(cronId), bz)

	// set the cron in the queue
	cron, ok := k.GetCron(ctx, cronId)
	if !ok {
		k.Logger(ctx).Info("Cron not found", "cronId", cronId)
		return
	}
	if k.ExistsInCronQueue(ctx, cron) {
		return
	}
	k.AppendToCronQueue(ctx, cron)
}

func (k *Keeper) GetCronCallBackData(ctx sdk.Context, cronId uint64) (*types.CronCallBackData, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronCallBackDataKey)
	bz := store.Get(sdk.Uint64ToBigEndian(cronId))
	if bz == nil {
		k.Logger(ctx).Info("GetCronCallBackData", "bz", bz)
		return nil, false
	}

	var cronCallBackData types.CronCallBackData
	k.cdc.MustUnmarshal(bz, &cronCallBackData)
	return &cronCallBackData, true
}

func (k *Keeper) GetCronIdByAddress(ctx sdk.Context, address string) (uint64, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronAddressKey)
	bz := store.Get([]byte(address))
	if bz == nil {
		return 0, false
	}

	id := sdk.BigEndianToUint64(bz)
	return id, true
}

func (k *Keeper) StoreRemoveCron(ctx sdk.Context, id uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)
	store.Delete(GetCronIDBytes(id))
}

func (k *Keeper) StoreCronExists(ctx sdk.Context, id uint64) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)
	return store.Has(GetCronIDBytes(id))
}

func (k *Keeper) StoreCronExistsByAddress(ctx sdk.Context, address string) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronAddressKey)
	return store.Has([]byte(address))
}

func (k *Keeper) ExistsInCronQueue(ctx sdk.Context, cron types.Cron) bool {
	if testnet.TESTNET_BLOCK_NUMBER_UPDATE_0 < int64(ctx.BlockHeight()) {
		return cron.QueueTimestamp != -1 && cron.QueueTimestamp != 0
	}
	return cron.QueueTimestamp != -1
}

func (k *Keeper) AppendToCronQueue(ctx sdk.Context, cron types.Cron) {
	store := ctx.KVStore(k.storeKey)
	idxKey := types.GetFeeSecondIndexKey(cron.MaxGasPrice.BigInt())
	var idSet types.IDSet
	if store.Has(idxKey) {
		bz := store.Get(idxKey)
		k.cdc.MustUnmarshal(bz, &idSet)
	}
	idSet.Ids = append(idSet.Ids, &types.IdAndTimestamp{Id: cron.Id, Timestamp: uint64(ctx.BlockTime().Unix())})
	store.Set(idxKey, k.cdc.MustMarshal(&idSet))

	cron.QueueTimestamp = ctx.BlockTime().Unix()
	k.StoreSetCron(ctx, cron)
}

func (k *Keeper) RemoveFromCronQueue(ctx sdk.Context, cron types.Cron) error {
	store := ctx.KVStore(k.storeKey)
	idxKey := types.GetFeeSecondIndexKey(cron.MaxGasPrice.BigInt())

	var idSet types.IDSet
	bz := store.Get(idxKey)
	if bz == nil {
		return errors.Wrap(errortypes.ErrLogic, "fee")
	}

	k.cdc.MustUnmarshal(bz, &idSet)
	for i := range idSet.Ids {
		if idSet.Ids[i].Id == cron.Id {
			idSet.Ids = append(idSet.Ids[0:i], idSet.Ids[i+1:]...)
			if len(idSet.Ids) != 0 {
				store.Set(idxKey, k.cdc.MustMarshal(&idSet))
				cron.QueueTimestamp = -1
				k.StoreSetCron(ctx, cron)
			} else {
				store.Delete(idxKey)
				cron.QueueTimestamp = -1
				k.StoreSetCron(ctx, cron)
			}
			return nil
		}
	}
	return errors.Wrap(errortypes.ErrNotFound, "tx id")
}

func (k *Keeper) GetBatchFees(ctx sdk.Context) *types.BatchFeesWithIds {
	params := k.GetParams(ctx)
	prefixStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.SecondIndexOutgoingTXFeeKey)
	iter := prefixStore.Iterator(nil, nil)
	// iterate over [SecondIndexOutgoingTXFeeKey][hyperionId] prefixes
	defer iter.Close()

	batchFees := &types.BatchFeesWithIds{
		TotalFees:       sdkmath.NewInt(0),
		Ids:             make([]uint64, 0),
		Fees:            make([]sdkmath.Int, 0),
		ExpiredIds:      make([]uint64, 0),
		TotalQueueCount: 0,
	}

	for ; iter.Valid(); iter.Next() {
		var ids types.IDSet
		k.cdc.MustUnmarshal(iter.Value(), &ids)

		key := iter.Key()

		feeAmountBytes := key[32:]
		feeAmount := big.NewInt(0).SetBytes(feeAmountBytes)
		sdkFeeAmount := sdkmath.NewIntFromBigInt(feeAmount)

		for _, idAndTimestamp := range ids.Ids {
			if idAndTimestamp.Timestamp < uint64(ctx.BlockTime().Unix())-types.DefaultCronQueueTimeout {
				batchFees.ExpiredIds = append(batchFees.ExpiredIds, idAndTimestamp.Id)
				batchFees.TotalQueueCount++
				continue
			}
			if batchFees.TotalQueueCount >= params.ExecutionsLimitPerBlock {
				// check if the tx as greater fee than others txs in the batch
				sort.SliceStable(batchFees.Fees, func(i, j int) bool {
					return batchFees.Fees[i].GT(batchFees.Fees[j])
				})
				// if the tx has greater fee than the first tx in the batch, replace the first tx with the new tx
				if batchFees.Fees[0].LT(sdkFeeAmount) {
					batchFees.Fees = append(batchFees.Fees, sdkFeeAmount)
					batchFees.Ids = append(batchFees.Ids, idAndTimestamp.Id)
				}
			} else {
				// add fee amount
				totalFees := batchFees.TotalFees
				totalFees = totalFees.Add(sdkFeeAmount)
				batchFees.TotalFees = totalFees
				batchFees.Fees = append(batchFees.Fees, sdkFeeAmount)
				batchFees.Ids = append(batchFees.Ids, idAndTimestamp.Id)
			}
			batchFees.TotalQueueCount++
		}
	}

	return batchFees
}

func (k *Keeper) GetCronQueueCount(ctx sdk.Context) int32 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CronQueueCountKey)
	if bz == nil {
		return 0
	}
	return int32(sdk.BigEndianToUint64(bz))
}

func (k *Keeper) SetCronQueueCount(ctx sdk.Context, count int32) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CronQueueCountKey, sdk.Uint64ToBigEndian(uint64(count)))
}

func GetCronIDBytes(id uint64) []byte {
	return sdk.Uint64ToBigEndian(id)
}

// func GetCronIdAndFeesPriorityIDBytes(id uint64, feesPriority *big.Int) []byte {
// 	// sdk.BigInt represented as a zero-extended big-endian byte slice (32 bytes)
// 	amount := make([]byte, 32)
// 	amount = feesPriority.FillBytes(amount)

// 	return append(amount, )
// }

func GetTxIDBytes(id uint64) []byte {
	return sdk.Uint64ToBigEndian(id)
}

func GetBlockIDBytes(id uint64) []byte {
	return sdk.Uint64ToBigEndian(id)
}

func recordExecutedCron(err error, cron types.Cron) {
	telemetry.IncrCounterWithLabels([]string{types.LabelCronExecutionsCount}, 1, []metrics.Label{
		telemetry.NewLabel(telemetry.MetricLabelNameModule, types.ModuleName),
		telemetry.NewLabel(types.MetricLabelSuccess, strconv.FormatBool(err == nil)),
		telemetry.NewLabel(types.MetricLabelCronName, strconv.FormatUint(cron.Id, 10)),
	})
}

// GetNextScheduleID returns a new unique schedule ID
func (k *Keeper) StoreGetNextCronID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	// Get the current next ID
	bz := store.Get(types.NextCronIDKey)

	// If no ID exists yet, start from 1
	var id uint64 = 1
	if bz != nil {
		id = sdk.BigEndianToUint64(bz)
		// Increment for the next call
		store.Set(types.NextCronIDKey, sdk.Uint64ToBigEndian(id+1))
	} else {
		// First time, store 2 as the next ID
		store.Set(types.NextCronIDKey, sdk.Uint64ToBigEndian(2))
	}

	return id
}

// GetNextScheduleID returns a new unique schedule ID
func (k *Keeper) StoreGetNonce(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	// Get the current next ID
	bz := store.Get(types.CronNonceKey)

	// If no ID exists yet, start from 1
	var id uint64 = 1
	if bz != nil {
		id = sdk.BigEndianToUint64(bz)
	}
	return id
}

func (k *Keeper) StoreSetNonce(ctx sdk.Context, nonce uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.CronNonceKey, sdk.Uint64ToBigEndian(nonce))
}

// GetAllCrons returns all crons
func (k Keeper) GetAllCrons(ctx sdk.Context) []types.Cron {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronKey)

	var crons []types.Cron
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var cron types.Cron
		k.cdc.MustUnmarshal(iterator.Value(), &cron)
		crons = append(crons, cron)
	}

	return crons
}

// todo remove this function
func (k *Keeper) StoreChangeTotalCount(ctx sdk.Context, increment int32) {
	store := ctx.KVStore(k.storeKey)
	count := k.GetCronCount(ctx) + increment
	store.Set(types.CronCountKey, sdk.Uint64ToBigEndian(uint64(count)))
}

// todo remove this function
func (k *Keeper) GetCronCount(ctx sdk.Context) int32 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CronCountKey)
	if bz == nil {
		return 0
	}
	return int32(sdk.BigEndianToUint64(bz))
}
