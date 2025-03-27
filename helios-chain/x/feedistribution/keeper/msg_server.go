package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	// govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/feedistribution/types"

	"github.com/ethereum/go-ethereum/crypto"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the feedistribution module.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// UpdateParams implements the MsgServer.UpdateParams method. It allows the authority
// to update the module parameters.
func (ms msgServer) UpdateParams(goCtx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	// if ms.authority.String() != msg.Authority {
	// 	return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
	// }

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.Params.Validate(); err != nil {
		return nil, err
	}

	if err := ms.SetParams(ctx, msg.Params); err != nil {
		return nil, err
	}

	return &types.MsgUpdateParamsResponse{}, nil
}

// RegisterRevenue implements the MsgServer.RegisterRevenue method. It registers
// a contract for revenue distribution.
func (k Keeper) RegisterRevenue(
	goCtx context.Context,
	msg *types.MsgRegisterRevenue,
) (*types.MsgRegisterRevenueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	params := k.GetParams(ctx)
	if !params.EnableFeeDistribution {
		return nil, types.ErrFeeDistributionDisabled
	}

	contract := common.HexToAddress(msg.ContractAddress)

	//! Implement this
	// if k.IsRevenueRegistered(ctx, contract) {
	// 	return nil, errorsmod.Wrapf(
	// 		types.ErrRevenueAlreadyRegistered,
	// 		"contract is already registered %s", contract,
	// 	)
	// }

	deployer := sdk.MustAccAddressFromBech32(msg.DeployerAddress)
	deployerAccount := k.evmKeeper.GetAccountWithoutBalance(ctx, common.BytesToAddress(deployer))
	if deployerAccount == nil {
		return nil, errorsmod.Wrapf(
			errortypes.ErrNotFound,
			"deployer account not found %s", msg.DeployerAddress,
		)
	}

	if deployerAccount.IsContract() {
		return nil, errorsmod.Wrapf(
			types.ErrRevenueDeployerIsNotEOA,
			"deployer cannot be a contract %s", msg.DeployerAddress,
		)
	}

	// contract must already be deployed, to avoid spam registrations
	contractAccount := k.evmKeeper.GetAccountWithoutBalance(ctx, contract)

	if contractAccount == nil || !contractAccount.IsContract() {
		return nil, errorsmod.Wrapf(
			types.ErrRevenueNoContractDeployed,
			"no contract code found at address %s", msg.ContractAddress,
		)
	}

	var withdrawer sdk.AccAddress
	if msg.WithdrawerAddress != "" && msg.WithdrawerAddress != msg.DeployerAddress {
		withdrawer = sdk.MustAccAddressFromBech32(msg.WithdrawerAddress)
	}

	derivedContract := common.BytesToAddress(deployer)

	// the contract can be directly deployed by an EOA or created through one
	// or more factory contracts. If it was deployed by an EOA account, then
	// msg.Nonces contains the EOA nonce for the deployment transaction.
	// If it was deployed by one or more factories, msg.Nonces contains the EOA
	// nonce for the origin factory contract, then the nonce of the factory
	// for the creation of the next factory/contract.
	for _, nonce := range msg.Nonces {
		ctx.GasMeter().ConsumeGas(
			params.AddrDerivationCostCreate,
			"revenue registration: address derivation CREATE opcode",
		)

		derivedContract = crypto.CreateAddress(derivedContract, nonce)
	}

	if contract != derivedContract {
		return nil, errorsmod.Wrapf(
			errortypes.ErrorInvalidSigner,
			"not contract deployer or wrong nonce: expected %s instead of %s",
			derivedContract, msg.ContractAddress,
		)
	}

	// prevent storing the same address for deployer and withdrawer
	revenue := types.NewRevenue(contract, deployer, withdrawer)
	k.SetRevenue(ctx, revenue)
	k.SetDeployerMap(ctx, deployer, contract)

	// The effective withdrawer is the withdraw address that is stored after the
	// revenue registration is completed. It defaults to the deployer address if
	// the withdraw address in the msg is omitted. When omitted, the withdraw map
	// dosn't need to be set.
	effectiveWithdrawer := msg.DeployerAddress

	if len(withdrawer) != 0 {
		k.SetWithdrawerMap(ctx, withdrawer, contract)
		effectiveWithdrawer = msg.WithdrawerAddress
	}

	k.Logger(ctx).Debug(
		"registering contract for transaction fees",
		"contract", msg.ContractAddress, "deployer", msg.DeployerAddress,
		"withdraw", effectiveWithdrawer,
	)

	ctx.EventManager().EmitEvents(
		sdk.Events{
			sdk.NewEvent(
				types.EventTypeRegisterRevenue,
				sdk.NewAttribute(sdk.AttributeKeySender, msg.DeployerAddress),
				sdk.NewAttribute(types.AttributeKeyContract, msg.ContractAddress),
				sdk.NewAttribute(types.AttributeKeyWithdrawerAddress, effectiveWithdrawer),
			),
		},
	)

	return &types.MsgRegisterRevenueResponse{}, nil
}

// UpdateRevenue implements the MsgServer.UpdateRevenue method. It updates
// the withdrawer address for a registered contract.
func (ms msgServer) UpdateRevenue(goCtx context.Context, msg *types.MsgUpdateRevenue) (*types.MsgUpdateRevenueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Basic validation
	deployer, err := sdk.AccAddressFromBech32(msg.DeployerAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid deployer address: %s", err.Error())
	}

	withdrawer, err := sdk.AccAddressFromBech32(msg.WithdrawerAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid withdrawer address: %s", err.Error())
	}

	if err := types.ValidateAddress(msg.ContractAddress); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid contract address: %s", err.Error())
	}

	// Get contract info
	contractInfo, found := ms.GetContractInfo(ctx, msg.ContractAddress)
	if !found {
		return nil, types.ErrContractNotRegistered.Wrapf("contract %s", msg.ContractAddress)
	}

	// Verify deployer
	if contractInfo.DeployerAddress != msg.DeployerAddress {
		return nil, types.ErrUnauthorized.Wrapf("deployer %s", msg.DeployerAddress)
	}

	// Update withdrawer
	contractInfo.WithdrawerAddress = withdrawer.String()
	ms.SetContractInfo(ctx, contractInfo)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeUpdateWithdrawer,
			sdk.NewAttribute(types.AttributeKeyContract, msg.ContractAddress),
			sdk.NewAttribute(types.AttributeKeyDeployer, deployer.String()),
			sdk.NewAttribute(types.AttributeKeyWithdrawer, withdrawer.String()),
		),
	})

	return &types.MsgUpdateRevenueResponse{}, nil
}

// CancelRevenue implements the MsgServer.CancelRevenue method. It cancels
// the fee distribution for a registered contract.
func (ms msgServer) CancelRevenue(goCtx context.Context, msg *types.MsgCancelRevenue) (*types.MsgCancelRevenueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Basic validation
	deployer, err := sdk.AccAddressFromBech32(msg.DeployerAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid deployer address: %s", err.Error())
	}

	if err := types.ValidateAddress(msg.ContractAddress); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid contract address: %s", err.Error())
	}

	// Get contract info
	contractInfo, found := ms.GetContractInfo(ctx, msg.ContractAddress)
	if !found {
		return nil, types.ErrContractNotRegistered.Wrapf("contract %s", msg.ContractAddress)
	}

	// Verify deployer
	if contractInfo.DeployerAddress != msg.DeployerAddress {
		return nil, types.ErrUnauthorized.Wrapf("deployer %s", msg.DeployerAddress)
	}

	// Delete contract info
	ms.DeleteContractInfo(ctx, msg.ContractAddress)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeDeleteRevenue,
			sdk.NewAttribute(types.AttributeKeyContract, msg.ContractAddress),
			sdk.NewAttribute(types.AttributeKeyDeployer, deployer.String()),
		),
	})

	return &types.MsgCancelRevenueResponse{}, nil
}
