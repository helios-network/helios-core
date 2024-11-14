package types

import (
	fmt "fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	v0types "helios-core/helios-chain/modules/evm/migrations/v0/types"
	currenttypes "helios-core/helios-chain/modules/evm/types"
)

func (p V4Params) ToParams() currenttypes.Params {
	chainConfig := currenttypes.ChainConfig{
		HomesteadBlock:      p.ChainConfig.HomesteadBlock,
		DAOForkBlock:        p.ChainConfig.DAOForkBlock,
		DAOForkSupport:      p.ChainConfig.DAOForkSupport,
		EIP150Block:         p.ChainConfig.EIP150Block,
		EIP150Hash:          p.ChainConfig.EIP150Hash,
		EIP155Block:         p.ChainConfig.EIP155Block,
		EIP158Block:         p.ChainConfig.EIP158Block,
		ByzantiumBlock:      p.ChainConfig.ByzantiumBlock,
		ConstantinopleBlock: p.ChainConfig.ConstantinopleBlock,
		PetersburgBlock:     p.ChainConfig.PetersburgBlock,
		IstanbulBlock:       p.ChainConfig.IstanbulBlock,
		MuirGlacierBlock:    p.ChainConfig.MuirGlacierBlock,
		BerlinBlock:         p.ChainConfig.BerlinBlock,
		LondonBlock:         p.ChainConfig.LondonBlock,
		ArrowGlacierBlock:   p.ChainConfig.ArrowGlacierBlock,
		GrayGlacierBlock:    p.ChainConfig.GrayGlacierBlock,
		MergeNetsplitBlock:  p.ChainConfig.MergeNetsplitBlock,
	}
	return currenttypes.Params{
		EvmDenom:            p.EvmDenom,
		EnableCreate:        p.EnableCreate,
		EnableCall:          p.EnableCall,
		ExtraEIPs:           p.ExtraEIPs.EIPs,
		AllowUnprotectedTxs: p.AllowUnprotectedTxs,
		ChainConfig:         chainConfig,
	}
}

// Validate performs basic validation on evm parameters.
func (p V4Params) Validate() error {
	if err := currenttypes.ValidateEVMDenom(p.EvmDenom); err != nil {
		return err
	}

	if err := validateEIPs(p.ExtraEIPs); err != nil {
		return err
	}

	if err := currenttypes.ValidateBool(p.EnableCall); err != nil {
		return err
	}

	if err := currenttypes.ValidateBool(p.EnableCreate); err != nil {
		return err
	}

	if err := currenttypes.ValidateBool(p.AllowUnprotectedTxs); err != nil {
		return err
	}

	return v0types.ValidateChainConfig(p.ChainConfig)
}

func validateEIPs(i interface{}) error {
	eips, ok := i.(ExtraEIPs)
	if !ok {
		return fmt.Errorf("invalid EIP slice type: %T", i)
	}

	for _, eip := range eips.EIPs {
		if !vm.ValidEip(int(eip)) {
			return fmt.Errorf("EIP %d is not activateable, valid EIPS are: %s", eip, vm.ActivateableEips())
		}
	}

	return nil
}
