package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// NewRevenue returns an instance of Revenue. If the provided withdrawer
// address is empty, it sets the value to an empty string.
func NewRevenue(contract common.Address, deployer, withdrawer sdk.AccAddress) Revenue {
	withdrawerAddr := ""
	if len(withdrawer) > 0 {
		withdrawerAddr = withdrawer.String()
	}

	return Revenue{
		ContractAddress:   contract.String(),
		DeployerAddress:   deployer.String(),
		WithdrawerAddress: withdrawerAddr,
	}
}

// GetWithdrawerAddr returns the withdrawer address as sdk.AccAddress
func (r Revenue) GetWithdrawerAddr() sdk.AccAddress {
	if r.WithdrawerAddress == "" {
		return nil
	}
	addr, err := sdk.AccAddressFromBech32(r.WithdrawerAddress)
	if err != nil {
		return nil
	}
	return addr
}

// GetDeployerAddr returns the deployer address as sdk.AccAddress
func (r Revenue) GetDeployerAddr() sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(r.DeployerAddress)
	if err != nil {
		return nil
	}
	return addr
}
