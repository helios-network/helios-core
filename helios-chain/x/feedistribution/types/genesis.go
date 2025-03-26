package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// Set default parameters
		Params: DefaultParams(),
		// Initialize empty slices
		BlockFees: []BlockFees{},
		Contracts: []ContractInfo{},
	}
}

// Validate performs basic genesis state validation
func (gs GenesisState) Validate() error {
	// Validate parameters
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Track contract addresses to check for duplicates
	contractAddrs := make(map[string]bool)

	// Validate contracts
	for _, contract := range gs.Contracts {
		// Check for duplicate contracts
		if contractAddrs[contract.ContractAddress] {
			return fmt.Errorf("duplicate contract address: %s", contract.ContractAddress)
		}
		contractAddrs[contract.ContractAddress] = true

		// Validate contract address
		if err := ValidateAddress(contract.ContractAddress); err != nil {
			return fmt.Errorf("invalid contract address: %w", err)
		}

		// Validate deployer address
		if _, err := sdk.AccAddressFromBech32(contract.DeployerAddress); err != nil {
			return fmt.Errorf("invalid deployer address: %w", err)
		}

		// Validate withdrawer address
		if _, err := sdk.AccAddressFromBech32(contract.WithdrawerAddress); err != nil {
			return fmt.Errorf("invalid withdrawer address: %w", err)
		}

		// Validate deployment height
		if contract.DeploymentHeight < 0 {
			return fmt.Errorf("invalid deployment height: %d", contract.DeploymentHeight)
		}
	}

	// Validate block fees
	for _, blockFee := range gs.BlockFees {
		// Validate contract address
		if err := ValidateAddress(blockFee.ContractAddress); err != nil {
			return fmt.Errorf("invalid contract address in block fees: %w", err)
		}

		// Validate accumulated fees
		if blockFee.AccumulatedFees.IsNegative() {
			return fmt.Errorf("negative accumulated fees for contract %s", blockFee.ContractAddress)
		}
	}

	return nil
}
