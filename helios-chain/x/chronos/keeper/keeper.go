package keeper

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/log"
	"github.com/hashicorp/go-metrics"

	cmn "helios-core/helios-chain/precompiles/common"
	rpctypes "helios-core/helios-chain/rpc/types"

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
	cdc           codec.BinaryCodec
	storeKey      storetypes.StoreKey
	memKey        storetypes.StoreKey
	accountKeeper types.AccountKeeper
	evmKeeper     types.EVMKeeper
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
		evmKeeper:     evmKeeper,
		bankKeeper:    bankKeeper,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func (k *Keeper) ExecuteAllReadyCrons(ctx sdk.Context) {
	telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), types.LabelExecuteReadyCrons)
	crons := k.getCronsReadyForExecution(ctx)

	for _, cron := range crons {
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

func (k *Keeper) ExecuteReadyCrons(ctx sdk.Context, executionStage types.ExecutionStage) {
	telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), types.LabelExecuteReadyCrons)
	crons := k.getCronsReadyForExecutionWithFilter(ctx, executionStage)

	for _, cron := range crons {
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

func (k *Keeper) DeductFeesActivesCrons(ctx sdk.Context) error {
	params := k.GetParams(ctx)
	decUtils, err := NewMonoDecoratorUtils(ctx, k.evmKeeper)
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

	for _, cron := range crons {
		balance := k.CronBalance(ctx, cron)
		cost := tx.Cost()

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
	account := k.evmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(cron.OwnerAddress))
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
	account := k.evmKeeper.GetAccount(ctx, cmn.AnyToHexAddress(cron.Address))
	balance := sdkmath.NewIntFromBigInt(account.Balance)

	return balance
}

func (k *Keeper) AddCron(ctx sdk.Context, cron types.Cron) {
	k.StoreSetCron(ctx, cron)
	k.StoreSetCronAddress(ctx, cron)
	k.StoreChangeTotalCount(ctx, 1)
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
	}

	k.StoreRemoveCron(ctx, id)
	k.StoreChangeTotalCount(ctx, -1)

	return nil
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

func (k *Keeper) GetCronTransactionResultByNonce(ctx sdk.Context, nonce uint64) (types.CronTransactionResult, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionResultKey)
	bz := store.Get(GetTxIDBytes(nonce))
	if bz == nil {
		return types.CronTransactionResult{}, false
	}

	var txResult types.CronTransactionResult
	k.cdc.MustUnmarshal(bz, &txResult)
	return txResult, true
}

func (k *Keeper) GetCronTransactionResultByHash(ctx sdk.Context, hash string) (types.CronTransactionResult, bool) {
	nonce, ok := k.GetTxNonceByHash(ctx, hash)
	if !ok {
		return types.CronTransactionResult{}, false
	}
	return k.GetCronTransactionResultByNonce(ctx, nonce)
}

func (k *Keeper) GetCronTransactionResultsByBlockNumber(ctx sdk.Context, blockNumber uint64) ([]types.CronTransactionResult, bool) {
	txHashs, ok := k.GetBlockTxHashs(ctx, blockNumber)
	if !ok {
		return []types.CronTransactionResult{}, false
	}
	txs := make([]types.CronTransactionResult, 0)
	for _, txHash := range txHashs {
		tx, ok := k.GetCronTransactionResultByHash(ctx, txHash)
		if !ok {
			continue
		}
		txs = append(txs, tx)
	}
	return txs, true
}

func (k *Keeper) GetCronTransactionReceiptByHash(ctx sdk.Context, hash string) (*types.CronTransactionReceiptRPC, bool) {
	tx, ok := k.GetCronTransactionResultByHash(ctx, hash)
	if !ok {
		return nil, false
	}
	receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
	return receiptTx, true
}

func (k *Keeper) GetCronTransactionReceiptByNonce(ctx sdk.Context, nonce uint64) (*types.CronTransactionReceiptRPC, bool) {
	tx, ok := k.GetCronTransactionResultByNonce(ctx, nonce)
	if !ok {
		return nil, false
	}
	receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
	return receiptTx, true
}

func (k *Keeper) GetCronTransactionReceiptsByBlockNumber(ctx sdk.Context, blockNumber uint64) ([]*types.CronTransactionReceiptRPC, bool) {
	txHashs, ok := k.GetBlockTxHashs(ctx, blockNumber)
	if !ok {
		return []*types.CronTransactionReceiptRPC{}, false
	}
	txs := make([]*types.CronTransactionReceiptRPC, 0)
	for _, txHash := range txHashs {
		tx, ok := k.GetCronTransactionResultByHash(ctx, txHash)
		if !ok {
			continue
		}
		receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
		txs = append(txs, receiptTx)
	}
	return txs, true
}

func (k *Keeper) GetCronTransactionLogsByBlockNumber(ctx sdk.Context, blockNumber uint64) ([]*evmtypes.Log, bool) {
	txHashs, ok := k.GetBlockTxHashs(ctx, blockNumber)
	if !ok {
		return []*evmtypes.Log{}, false
	}
	txs := make([]*evmtypes.Log, 0)
	for _, txHash := range txHashs {
		tx, ok := k.GetCronTransactionResultByHash(ctx, txHash)
		if !ok {
			continue
		}
		receiptTx, _ := k.FormatCronTransactionResultToCronTransactionReceiptRPC(ctx, tx)
		txs = append(txs, receiptTx.Logs...)
	}
	return txs, true
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

	return crons
}

func (k *Keeper) getCronsReadyForExecutionWithFilter(ctx sdk.Context, executionStage types.ExecutionStage) []types.Cron {
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

		if cron.ExecutionStage == executionStage &&
			currentBlock >= cron.NextExecutionBlock &&
			(cron.ExpirationBlock == 0 || currentBlock <= cron.ExpirationBlock) {
			crons = append(crons, cron)
			count++
			if count >= params.ExecutionsLimitPerBlock {
				k.Logger(ctx).Info("Reached execution limit for the block")
				break
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

	// Pack the call data
	callData, err := contractABI.Pack(cron.MethodName, parsedParams...)
	if err != nil {
		k.Logger(ctx).Error("ABI packing failed", "cron_id", cron.Id, "error", err)
		return nil, err
	}

	// get BaseFee
	decUtils, err := NewMonoDecoratorUtils(ctx, k.evmKeeper)
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

	if err := k.evmKeeper.DeductTxCostsFromUserBalance(
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

	account := k.evmKeeper.GetAccount(ctx, ownerAddress)
	balance := sdkmath.NewIntFromBigInt(account.Balance)
	cost := tx.Cost()

	if balance.IsNegative() || balance.BigInt().Cmp(cost) < 0 {
		return nil, errors.Wrapf(
			errortypes.ErrInsufficientFunds,
			"sender balance < tx cost (%s < %s)", balance, cost,
		)
	}

	// 1. get BaseFee
	decUtils, err := NewMonoDecoratorUtils(ctx, k.evmKeeper)
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
	// 4. Consume Fees on cron Wallet
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
	// 6. execute tx
	res, err := k.evmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), true)
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

	if err = k.evmKeeper.RefundGas(ctx, msgForRefund, msgForRefund.Gas()-res.GasUsed, baseDenom); err != nil {
		return nil, errors.Wrapf(err, "failed to refund gas leftover gas to sender %s", msg.From())
	}
	if refundCoinsR[0].Amount.GT(sdkmath.NewInt(0)) { // update cron fees paid
		k.UpdateCronTotalFeesPaid(ctx, cron, refundCoinsR[0].Amount)
	}
	return res, nil
}

func (k *Keeper) executeCron(ctx sdk.Context, cron types.Cron) error {
	nonce := k.StoreGetNonce(ctx)

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
	}

	// Update the next execution block after successful execution
	cron.NextExecutionBlock = uint64(ctx.BlockHeight()) + cron.Frequency
	cron.TotalExecutedTransactions += 1

	k.StoreSetCron(ctx, cron)
	k.StoreCronTransactionResult(ctx, cron, cronTxResult)
	k.StoreSetTransactionNonceByHash(ctx, tx.Hash().Hex(), nonce)
	k.StoreSetTransactionHashInBlock(ctx, cronTxResult.BlockNumber, tx.Hash().Hex())
	k.StoreSetNonce(ctx, nonce+1)
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
		Ret:     hexutil.Encode(castedResponse.Ret), // Ret is the bytes of call return
		VmError: castedResponse.VmError,
		CronId:  txResult.CronId,
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

func (k *Keeper) StoreSetCronAddress(ctx sdk.Context, cron types.Cron) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronAddressKey)
	store.Set([]byte(cron.Address), sdk.Uint64ToBigEndian(cron.Id))
}

func (k *Keeper) StoreCronTransactionResult(ctx sdk.Context, cron types.Cron, tx types.CronTransactionResult) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionResultKey)
	bz := k.cdc.MustMarshal(&tx)
	store.Set(GetTxIDBytes(tx.Nonce), bz)

	// Stockage uniquement du nonce dans l'index secondaire pour éviter les doublons
	storeByCronId := prefix.NewStore(ctx.KVStore(k.storeKey), append(types.CronTransactionResultByCronIdKey, sdk.Uint64ToBigEndian(cron.Id)...))

	// ici on ne stocke que le nonce (très léger) comme référence
	storeByCronId.Set(GetTxIDBytes(tx.Nonce), []byte{}) // pas besoin de valeur car on récupère la donnée via le nonce dans le store principal
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

func (k *Keeper) GetBlockTxHashs(ctx sdk.Context, blockNumber uint64) ([]string, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronBlockTransactionHashsKey)
	bz := store.Get(GetBlockIDBytes(blockNumber))
	if bz == nil {
		return []string{}, false
	}

	var txHashes []string
	err := json.Unmarshal(bz, &txHashes)
	if err != nil {
		return []string{}, false
	}
	return txHashes, true
}

func (k *Keeper) StoreSetTransactionHashInBlock(ctx sdk.Context, blockNumber uint64, txHash string) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronBlockTransactionHashsKey)

	txHashes, _ := k.GetBlockTxHashs(ctx, blockNumber)
	txHashes = append(txHashes, txHash)

	bz, _ := json.Marshal(&txHashes)
	store.Set(GetBlockIDBytes(blockNumber), bz)
}

func (k *Keeper) GetTxNonceByHash(ctx sdk.Context, txHash string) (uint64, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionHashToNonceKey)
	bz := store.Get([]byte(txHash))
	if bz == nil {
		return 0, false
	}

	nonce := sdk.BigEndianToUint64(bz)
	return nonce, true
}

func (k *Keeper) StoreSetTransactionNonceByHash(ctx sdk.Context, txHash string, nonce uint64) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.CronTransactionHashToNonceKey)
	store.Set([]byte(txHash), sdk.Uint64ToBigEndian(nonce))
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

func (k *Keeper) StoreChangeTotalCount(ctx sdk.Context, increment int32) {
	store := ctx.KVStore(k.storeKey)
	count := k.getCronCount(ctx) + increment
	store.Set(types.CronCountKey, sdk.Uint64ToBigEndian(uint64(count)))
}

func (k *Keeper) getCronCount(ctx sdk.Context) int32 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CronCountKey)
	if bz == nil {
		return 0
	}
	return int32(sdk.BigEndianToUint64(bz))
}

func GetCronIDBytes(id uint64) []byte {
	return sdk.Uint64ToBigEndian(id)
}

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
