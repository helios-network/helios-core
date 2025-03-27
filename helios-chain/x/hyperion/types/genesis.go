package types

import "cosmossdk.io/errors"

// ValidateBasic validates genesis state by looping through the params and
// calling their validation functions
func (s GenesisState) ValidateBasic() error {
	if err := s.Params.ValidateBasic(); err != nil {
		return errors.Wrap(err, "params")
	}
	return nil
}

// DefaultGenesisState returns empty genesis state
func DefaultGenesisState() *GenesisState {
	params := DefaultParams()

	return &GenesisState{
		Params:    params,
		SubStates: DefaultSubStates(params),
		// Erc20ToDenoms: MockErc20ToDenoms(),
	}
}
