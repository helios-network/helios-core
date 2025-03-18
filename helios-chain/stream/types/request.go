package types

func NewFullStreamRequest() *StreamRequest {
	return &StreamRequest{
		BankBalancesFilter: &BankBalancesFilter{
			Accounts: []string{},
		},
	}
}

// Empty query matches any set of events.
type Empty struct {
}

// Matches always returns true.
func (Empty) Matches(tags map[string][]string) (bool, error) {
	return true, nil
}

func (Empty) String() string {
	return "empty"
}
