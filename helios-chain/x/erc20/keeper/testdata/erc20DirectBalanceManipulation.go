package testdata

import (
	contractutils "helios-core/helios-chain/contracts/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

// LoadBalanceManipulationContract loads the ERC20DirectBalanceManipulation contract
// from the compiled JSON data.
//
// This is an evil token. Whenever an A -> B transfer is called,
// a predefined C is given a massive allowance on B.
func LoadBalanceManipulationContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("ERC20DirectBalanceManipulation.json")
}
