package testdata

import (
	contractutils "helios-core/helios-chain/contracts/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

// LoadWEVMOS9Contract load the WEVMOS9 contract from the json representation of
// the Solidity contract.
func LoadWEVMOS9Contract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("WEVMOS9.json")
}
