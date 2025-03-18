package types

import (
	"fmt"
	"time"
)

type StreamResponseMap struct {
	tradeEventsCounter    uint64
	BlockHeight           uint64
	BlockTime             time.Time
	BankBalancesByAccount map[string][]*BankBalance
}

func NewStreamResponseMap() *StreamResponseMap {
	return &StreamResponseMap{
		BankBalancesByAccount: map[string][]*BankBalance{},
	}
}

func NewChainStreamResponse() *StreamResponse {
	return &StreamResponse{
		BankBalances: []*BankBalance{},
	}
}

func (m *StreamRequest) Validate() error {
	if m.BankBalancesFilter == nil {
		return fmt.Errorf("at least one filter must be set")
	}
	return nil
}

func (m *StreamResponseMap) NextTradeEventNumber() (tradeNumber uint64) {
	currentTradesNumber := m.tradeEventsCounter
	m.tradeEventsCounter++
	return currentTradesNumber
}
