// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. 
package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/core/vm"
	currenttypes "helios-core/helios-chain/modules/evm/types"
)

func (p V0Params) ToParams() currenttypes.Params {
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
		ExtraEIPs:           p.ExtraEIPs,
		AllowUnprotectedTxs: p.AllowUnprotectedTxs,
		ChainConfig:         chainConfig,
	}
}

// Validate performs basic validation on evm parameters.
func (p V0Params) Validate() error {
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

	return ValidateChainConfig(p.ChainConfig)
}

func validateEIPs(i interface{}) error {
	eips, ok := i.([]int64)
	if !ok {
		return fmt.Errorf("invalid EIP slice type: %T", i)
	}

	for _, eip := range eips {
		if !vm.ValidEip(int(eip)) {
			return fmt.Errorf("EIP %d is not activateable, valid EIPS are: %s", eip, vm.ActivateableEips())
		}
	}

	return nil
}

func ValidateChainConfig(i interface{}) error {
	cfg, ok := i.(V0ChainConfig)
	if !ok {
		return fmt.Errorf("invalid chain config type: %T", i)
	}
	return cfg.Validate()
}
