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
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	currenttypes "helios-core/helios-chain/modules/evm/types"
)

// Parameter keys
var (
	ParamStoreKeyEVMDenom            = []byte("EVMDenom")
	ParamStoreKeyEnableCreate        = []byte("EnableCreate")
	ParamStoreKeyEnableCall          = []byte("EnableCall")
	ParamStoreKeyExtraEIPs           = []byte("EnableExtraEIPs")
	ParamStoreKeyChainConfig         = []byte("ChainConfig")
	ParamStoreKeyAllowUnprotectedTxs = []byte("AllowUnprotectedTxs")
)

// Deprecated: ParamKeyTable returns the parameter key table.
// Usage of x/params to manage parameters is deprecated in favor of x/gov
// controlled execution of MsgUpdateParams messages. These types remain solely
// for migration purposes and will be removed in a future release.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&V0Params{})
}

// Deprecated: ParamSetPairs returns the parameter set pairs.
// Usage of x/params to manage parameters is deprecated in favor of x/gov
// controlled execution of MsgUpdateParams messages. These types remain solely
// for migration purposes and will be removed in a future release.
func (p *V0Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyEVMDenom, &p.EvmDenom, currenttypes.ValidateEVMDenom),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableCreate, &p.EnableCreate, currenttypes.ValidateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyEnableCall, &p.EnableCall, currenttypes.ValidateBool),
		paramtypes.NewParamSetPair(ParamStoreKeyExtraEIPs, &p.ExtraEIPs, validateEIPs),
		paramtypes.NewParamSetPair(ParamStoreKeyChainConfig, &p.ChainConfig, ValidateChainConfig),
		paramtypes.NewParamSetPair(ParamStoreKeyAllowUnprotectedTxs, &p.AllowUnprotectedTxs, currenttypes.ValidateBool),
	}
}
