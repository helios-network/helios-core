package types

import sdk "github.com/cosmos/cosmos-sdk/types"

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
