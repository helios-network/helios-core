// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:LGPL-3.0-only

package types

import (
	"fmt"

	"cosmossdk.io/math"
)

// Parameter store key
var (
	DefaultEnableRevenue   = true
	DefaultDeveloperShares = math.LegacyNewDecWithPrec(10, 2) // 10%
	// DefaultAddrDerivationCostCreate Cost for executing `crypto.CreateAddress` must be at least 36 gas for the
	// contained keccak256(word) operation
	DefaultAddrDerivationCostCreate = uint64(50)
)

var (
	ParamsKey                             = []byte("Params")
	ParamStoreKeyEnableRevenue            = []byte("EnableRevenue")
	ParamStoreKeyDeveloperShares          = []byte("DeveloperShares")
	ParamStoreKeyAddrDerivationCostCreate = []byte("AddrDerivationCostCreate")
)

// NewParams creates a new Params object
func NewParams(
	enableRevenue bool,
	developerShares math.LegacyDec,
	addrDerivationCostCreate uint64,
) Params {
	return Params{
		EnableRevenue:            enableRevenue,
		DeveloperShares:          developerShares,
		AddrDerivationCostCreate: addrDerivationCostCreate,
	}
}

func DefaultParams() Params {
	return Params{
		EnableRevenue:            DefaultEnableRevenue,
		DeveloperShares:          DefaultDeveloperShares,
		AddrDerivationCostCreate: DefaultAddrDerivationCostCreate,
	}
}

func validateUint64(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateShares(i interface{}) error {
	v, ok := i.(math.LegacyDec)

	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("invalid parameter: nil")
	}

	if v.IsNegative() {
		return fmt.Errorf("value cannot be negative: %T", i)
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("value cannot be greater than 1: %T", i)
	}

	return nil
}

func (p Params) Validate() error {
	if err := validateBool(p.EnableRevenue); err != nil {
		return err
	}
	if err := validateShares(p.DeveloperShares); err != nil {
		return err
	}
	return validateUint64(p.AddrDerivationCostCreate)
}
