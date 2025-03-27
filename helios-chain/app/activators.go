package app

import (
	"helios-core/helios-chain/app/eips"
	"helios-core/helios-chain/x/evm/core/vm"
)

// evmosActivators defines a map of opcode modifiers associated
// with a key defining the corresponding EIP.
var evmosActivators = map[string]func(*vm.JumpTable){
	"evmos_0": eips.Enable0000,
	"evmos_1": eips.Enable0001,
	"evmos_2": eips.Enable0002,
}
