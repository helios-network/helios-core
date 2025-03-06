package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state for the cron module.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		ScheduleList: []Schedule{},
		Params:       DefaultParams(),
	}
}

// Validate performs basic genesis state validation.
func (gs GenesisState) Validate() error {
	scheduleIDMap := make(map[uint64]struct{})

	for _, elem := range gs.ScheduleList {
		if _, exists := scheduleIDMap[elem.Id]; exists {
			return fmt.Errorf("duplicate schedule ID found: %d", elem.Id)
		}
		scheduleIDMap[elem.Id] = struct{}{}

		if _, err := sdk.AccAddressFromBech32(elem.OwnerAddress); err != nil {
			return fmt.Errorf("invalid owner address (%s): %w", elem.OwnerAddress, err)
		}

		if elem.ContractAddress == "" || elem.MethodName == "" || elem.Frequency == 0 {
			return fmt.Errorf("schedule fields cannot be empty or zero")
		}
	}

	return gs.Params.Validate()
}
