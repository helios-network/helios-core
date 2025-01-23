package stream

import (
	"fmt"
	"helios-core/helios-chain/stream/types"

	abci "github.com/cometbft/cometbft/abci/types"
)

func handleBankBalanceEvent(inBuffer *types.StreamResponseMap, ev abci.Event) error {
	msgs, err := ABCIToBankBalances(ev)
	if err != nil {
		return fmt.Errorf("error converting ABCI event to BankBalance: %w", err)
	}
	for _, msg := range msgs {
		if _, ok := inBuffer.BankBalancesByAccount[msg.Account]; !ok {
			inBuffer.BankBalancesByAccount[msg.Account] = make([]*types.BankBalance, 0)
		}
		inBuffer.BankBalancesByAccount[msg.Account] = append(inBuffer.BankBalancesByAccount[msg.Account], msg)
	}
	return nil
}
