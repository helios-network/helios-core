package hyperion

import (
	"fmt"
	"math/big"

	cmn "helios-core/helios-chain/precompiles/common"

	"helios-core/helios-chain/x/evm/core/vm"
	hyperionkeeper "helios-core/helios-chain/x/hyperion/keeper"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	cosmosmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	AddCounterpartyChainParamsMethod = "addCounterpartyChainParams"
	SetOrchestratorAddressesMethod   = "setOrchestratorAddresses"
	SendToChainMethod                = "sendToChain"
	RequestDataHyperion              = "requestData"
)

func (p Precompile) AddCounterpartyChainParams(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	ctx.Logger().Info("AddCounterpartyChainParams -10")

	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	ctx.Logger().Info("AddCounterpartyChainParams -9")

	hyperionId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 hyperionId")
	}

	contractSourceHash, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string contractSourceHash")
	}

	bridgeCounterpartyAddress, ok := args[2].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string bridgeCounterpartyAddress")
	}

	bridgeChainId, ok := args[3].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 bridgeChainId")
	}

	bridgeContractStartHeight, ok := args[4].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 bridgeContractStartHeight")
	}

	ctx.Logger().Info("AddCounterpartyChainParams -8")

	msg := &hyperiontypes.MsgAddCounterpartyChainParams{
		Orchestrator: cmn.AccAddressFromHexAddress(origin).String(),
		CounterpartyChainParams: &hyperiontypes.CounterpartyChainParams{
			HyperionId:                    hyperionId,
			ContractSourceHash:            contractSourceHash, // hash of the BridgeCounterparty Smart Contract
			BridgeCounterpartyAddress:     bridgeCounterpartyAddress,
			BridgeChainId:                 bridgeChainId,
			SignedValsetsWindow:           25000,
			SignedBatchesWindow:           25000,
			SignedClaimsWindow:            25000,
			TargetBatchTimeout:            43200000,
			AverageBlockTime:              2000,
			AverageCounterpartyBlockTime:  15000,
			SlashFractionValset:           cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionBatch:            cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionClaim:            cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			SlashFractionConflictingClaim: cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			UnbondSlashingValsetsWindow:   25000,
			SlashFractionBadEthSignature:  cosmosmath.LegacyNewDecFromIntWithPrec(cosmosmath.NewInt(1), 3), // 0.001
			CosmosCoinDenom:               "ahelios",
			CosmosCoinErc20Contract:       "",
			ClaimSlashingEnabled:          false,
			BridgeContractStartHeight:     bridgeContractStartHeight,
			ValsetReward:                  sdk.Coin{Denom: "ahelios", Amount: cosmosmath.NewInt(0)},
		},
	}

	ctx.Logger().Info("AddCounterpartyChainParams -7", "msg", msg)

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.AddCounterpartyChainParams(ctx, msg)
	if err != nil {
		return nil, err
	}

	ctx.Logger().Info("AddCounterpartyChainParams -6")

	return method.Outputs.Pack(true)
}

func (p Precompile) SetOrchestratorAddresses(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	orchestratorAddress, ok := args[0].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid hex address")
	}

	msg := &hyperiontypes.MsgSetOrchestratorAddresses{
		Sender:       cmn.AccAddressFromHexAddress(origin).String(),
		Orchestrator: cmn.AccAddressFromHexAddress(orchestratorAddress).String(),
		EthAddress:   orchestratorAddress.String(),
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.SetOrchestratorAddresses(ctx, msg)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p Precompile) SendToChain(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 3 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 3, len(args))
	}

	dest, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string address")
	}

	amount, ok := args[1].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for AmountToDeposit")
	}
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	bridgeFee, ok := args[2].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 for AmountToDeposit")
	}
	if bridgeFee.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	amountV := cosmosmath.NewIntFromBigInt(amount)
	bridgeFeeV := cosmosmath.NewIntFromBigInt(bridgeFee)

	msg := &hyperiontypes.MsgSendToChain{
		Sender:         cmn.AccAddressFromHexAddress(origin).String(),
		DestHyperionId: 1,
		Dest:           dest,
		Amount:         sdk.NewCoin("aaa", amountV),
		BridgeFee:      sdk.NewCoin("aaa", bridgeFeeV),
	}

	msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	_, err := msgSrv.SendToChain(ctx, msg)
	if err != nil {
		return nil, err
	}

	// if err := p.EmitCronCreatedEvent(ctx, stateDB, origin, p.Address(), resp.CronId); err != nil {
	// 	return nil, err
	// }

	return method.Outputs.Pack(true)
}

func (p Precompile) RequestData(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	// taskId := big.NewInt(123)
	// return method.Outputs.Pack(taskId)
	// Pack the taskId as a bytes32 output

	// Vérification du nombre d'arguments
	if len(args) != 6 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 6, len(args))
	}

	// Extraction des paramètres
	// chainId, ok := args[0].(*big.Int)
	// if !ok {
	// 	return nil, fmt.Errorf("invalid chainId type")
	// }

	// sourceContract, ok := args[1].(common.Address)
	// if !ok {
	// 	return nil, fmt.Errorf("invalid sourceContract type")
	// }

	// abiCallData, ok := args[2].([]byte)
	// if !ok {
	// 	return nil, fmt.Errorf("invalid abiCallData type")
	// }

	// callbackSelector, ok := args[3].([4]byte)
	// if !ok {
	// 	return nil, fmt.Errorf("invalid callbackSelector type")
	// }

	maxCallbackGas, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid maxCallbackGas type")
	}

	bridgeFee, ok := args[5].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid bridgeFee type")
	}

	// Vérification des fonds
	totalFee := new(big.Int).Add(maxCallbackGas, bridgeFee)
	if contract.Value().Cmp(totalFee) < 0 {
		return nil, fmt.Errorf("insufficient funds for bridge operation")
	}

	// Création du message pour le module Hyperion
	//TODO: send msg to hyperion for processing cron request data request callback
	// msg := &hyperiontypes.MsgRequestExternalData{
	// 	Requester:         cmn.AccAddressFromHexAddress(origin).String(),
	// 	ChainId:           chainId.Uint64(),
	// 	SourceContract:    sourceContract.Hex(),
	// 	AbiCallData:       abiCallData,
	// 	CallbackSelector:  callbackSelector[:],
	// 	MaxCallbackGas:    maxCallbackGas.Uint64(),
	// 	BridgeFee:         cosmosmath.NewIntFromBigInt(bridgeFee),
	// 	InitiatorContract: cmn.AccAddressFromHexAddress(origin).String(),
	// }

	// // Envoi du message via le serveur de messages
	// msgSrv := hyperionkeeper.NewMsgServerImpl(p.hyperionKeeper)
	// resp, err := msgSrv.RequestExternalData(ctx, msg)
	// if err != nil {
	// 	return nil, err
	// }

	// Transfert des fonds au module Hyperion
	feeCoins := sdk.NewCoins(sdk.NewCoin("ahelios", cosmosmath.NewIntFromBigInt(totalFee)))
	if err := p.bankKeeper.SendCoinsFromAccountToModule(
		ctx,
		cmn.AccAddressFromHexAddress(origin),
		hyperiontypes.ModuleName,
		feeCoins,
	); err != nil {
		return nil, err
	}

	// Retourne l'ID de la tâche créée
	// return method.Outputs.Pack(resp.TaskId)
	return method.Outputs.Pack(0)
}
