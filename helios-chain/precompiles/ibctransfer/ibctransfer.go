package ibctransfer

import (
	"embed"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/evm/core/vm"
	ibckeeper "helios-core/helios-chain/x/ibc/transfer/keeper"
)

// Embed abi json file to the executable binary. Needed when importing as dependency.
//
//go:embed abi.json
var f embed.FS

var _ vm.PrecompiledContract = &Precompile{}

type Precompile struct {
	cmn.Precompile
	ibcKeeper ibckeeper.Keeper
}

func NewPrecompile() *Precompile {
	return &Precompile{}
}

func (p *Precompile) RequiredGas(input []byte) uint64 {
	return 0
}

func (p *Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	return nil, nil
}
