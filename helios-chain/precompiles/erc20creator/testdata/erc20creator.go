package testdata

import (
	contractutils "helios-core/helios-chain/contracts/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

func LoadErc20CreatorCallerContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("Erc20CreatorCaller.json")
}
