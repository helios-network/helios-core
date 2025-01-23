package stream

import (
	"fmt"
	"helios-core/helios-chain/stream/types"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func ABCIToBankBalances(ev abci.Event) (messages []*types.BankBalance, err error) {
	if Topic(ev.Type) != BankBalances {
		return nil, fmt.Errorf("unexpected topic: %s", ev.Type)
	}

	balanceUpdates := []banktypes.BalanceUpdate{}

	for _, attr := range ev.Attributes {
		switch attr.Key {
		case "balance_updates":
			err = json.Unmarshal([]byte(attr.Value), &balanceUpdates)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal ABCI event to BankBalance: %w", err)
			}
		}
	}

	for idx := range balanceUpdates {
		address := sdk.AccAddress(balanceUpdates[idx].Addr).String()
		denom := string(balanceUpdates[idx].Denom)
		amount := balanceUpdates[idx].Amt
		messages = append(messages, &types.BankBalance{
			Account: address,
			Balances: sdk.Coins{
				sdk.NewCoin(denom, amount),
			},
		})
	}

	return messages, nil
}
