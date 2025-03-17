package testdata

import (
	contractutils "helios-core/helios-chain/contracts/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

func LoadStakingCallerTwoContract() (evmtypes.CompiledContract, error) {
	return contractutils.LoadContractFromJSONFile("StakingCallerTwo.json")
}
