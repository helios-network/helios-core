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

	if len(args) != 5 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 5, len(args))
	}

	hyperionId, ok := args[0].(uint64)
	if !ok {
		return nil, fmt.Errorf("invalid uint64 hyperionId")
	}

	dest, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string dest address")
	}

	contractAddress, ok := args[2].(common.Address)
	if !ok {
		return nil, fmt.Errorf("invalid contract address")
	}

	amount, ok := args[3].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint256 for AmountToDeposit")
	}
	if amount.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	bridgeFee, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid uint256 for AmountToDeposit")
	}
	if bridgeFee.Cmp(big.NewInt(0)) <= 0 {
		return nil, fmt.Errorf("invalid zero AmountToDeposit")
	}

	amountV := cosmosmath.NewIntFromBigInt(amount)
	bridgeFeeV := cosmosmath.NewIntFromBigInt(bridgeFee)

	tokenPair, ok := p.erc20Keeper.GetTokenPair(ctx, p.erc20Keeper.GetTokenPairID(ctx, contractAddress.String()))

	if !ok {
		// no erc20 exists for this token
		return nil, fmt.Errorf("invalid token %s not registered as tokenPair", contractAddress.String())
	}

	msg := &hyperiontypes.MsgSendToChain{
		Sender:         cmn.AccAddressFromHexAddress(origin).String(),
		DestHyperionId: hyperionId,
		Dest:           dest,
		Amount:         sdk.NewCoin(tokenPair.Denom, amountV),
		BridgeFee:      sdk.NewCoin(tokenPair.Denom, bridgeFeeV),
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
