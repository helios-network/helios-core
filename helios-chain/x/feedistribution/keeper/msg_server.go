package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/ethereum/go-ethereum/common"

	"helios-core/helios-chain/x/feedistribution/types"
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
	if ms.authority.String() != msg.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", ms.authority, msg.Authority)
	}

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
func (ms msgServer) RegisterRevenue(goCtx context.Context, msg *types.MsgRegisterRevenue) (*types.MsgRegisterRevenueResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Basic validation
	deployer, err := sdk.AccAddressFromBech32(msg.DeployerAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid deployer address: %s", err.Error())
	}

	var withdrawer sdk.AccAddress
	if msg.WithdrawerAddress != "" {
		withdrawer, err = sdk.AccAddressFromBech32(msg.WithdrawerAddress)
		if err != nil {
			return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid withdrawer address: %s", err.Error())
		}
	} else {
		withdrawer = deployer
	}

	if err := types.ValidateAddress(msg.ContractAddress); err != nil {
		return nil, errorsmod.Wrapf(types.ErrInvalidAddress, "invalid contract address: %s", err.Error())
	}

	// Convert hex address to Ethereum common.Address
	contractAddress := common.HexToAddress(msg.ContractAddress)

	// Check if contract exists and get deployer
	if _, found := ms.evmKeeper.GetContractDeployerAddress(ctx, contractAddress); !found {
		return nil, types.ErrContractDeployerNotFound.Wrapf("contract %s", msg.ContractAddress)
	}

	// Check if contract is already registered
	if _, found := ms.GetContractInfo(ctx, msg.ContractAddress); found {
		return nil, types.ErrContractAlreadyRegistered.Wrapf("contract %s", msg.ContractAddress)
	}

	// Create and store contract info
	contractInfo := types.ContractInfo{
		ContractAddress:   msg.ContractAddress,
		DeployerAddress:   msg.DeployerAddress,
		WithdrawerAddress: withdrawer.String(),
		DeploymentHeight:  ctx.BlockHeight(),
	}
	ms.SetContractInfo(ctx, contractInfo)

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.EventTypeRegisterRevenue,
			sdk.NewAttribute(types.AttributeKeyContract, contractAddress.String()),
			sdk.NewAttribute(types.AttributeKeyDeployer, deployer.String()),
			sdk.NewAttribute(types.AttributeKeyWithdrawer, withdrawer.String()),
		),
	})

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
