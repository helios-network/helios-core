package grpc

import (
	"context"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// GetBalance returns the balance for the given address and denom.
func (gqh *IntegrationHandler) GetBalance(address sdktypes.AccAddress, denom string) (*banktypes.QueryBalanceResponse, error) {
	bankClient := gqh.network.GetBankClient()
	return bankClient.Balance(context.Background(), &banktypes.QueryBalanceRequest{
		Address: address.String(),
		Denom:   denom,
	})
}

// GetAllBalances returns all the balances for the given address.
func (gqh *IntegrationHandler) GetAllBalances(address sdktypes.AccAddress) (*banktypes.QueryAllBalancesResponse, error) {
	bankClient := gqh.network.GetBankClient()
	return bankClient.AllBalances(context.Background(), &banktypes.QueryAllBalancesRequest{
		Address: address.String(),
	})
}

// GetTotalSupply returns all the balances for the given address.
func (gqh *IntegrationHandler) GetTotalSupply() (*banktypes.QueryTotalSupplyResponse, error) {
	bankClient := gqh.network.GetBankClient()
	return bankClient.TotalSupply(context.Background(), &banktypes.QueryTotalSupplyRequest{})
}

// GetSpendableBalance returns the spendable balance for the given denomination.
func (gqh *IntegrationHandler) GetSpendableBalance(address sdktypes.AccAddress, denom string) (*banktypes.QuerySpendableBalanceByDenomResponse, error) {
	bankClient := gqh.network.GetBankClient()
	return bankClient.SpendableBalanceByDenom(context.Background(), &banktypes.QuerySpendableBalanceByDenomRequest{Address: address.String(), Denom: denom})
}
