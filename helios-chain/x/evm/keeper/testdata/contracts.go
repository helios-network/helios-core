// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)

package testdata

import (
	contractutils "helios-core/helios-chain/contracts/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

func LoadERC20Contract() (evmtypes.CompiledContract, error) {
	return contractutils.LegacyLoadContractFromJSONFile("ERC20Contract.json")
}

func LoadMessageCallContract() (evmtypes.CompiledContract, error) {
	return contractutils.LegacyLoadContractFromJSONFile("MessageCallContract.json")
}
