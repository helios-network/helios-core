package types

import (
	"fmt"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyEnableFeeDistribution = []byte("EnableFeeDistribution")
	KeyDeveloperShares       = []byte("DeveloperShares")
)

// DefaultDeveloperShares is the default percentage of fees that go to contract deployers (10%)
var DefaultDeveloperShares = math.LegacyNewDecWithPrec(10, 2) // 0.10 or 10%

// DefaultParams returns default parameters
func DefaultParams() Params {
	return Params{
		EnableFeeDistribution: true,
		DeveloperShares:       DefaultDeveloperShares,
	}
}

// Validate performs basic validation on feedistribution parameters.
func (p Params) Validate() error {
	if err := validateEnableFeeDistribution(p.EnableFeeDistribution); err != nil {
		return err
	}

	if err := validateDeveloperShares(p.DeveloperShares); err != nil {
		return err
	}

	return nil
}

func validateEnableFeeDistribution(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateDeveloperShares(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	shares, err := math.LegacyNewDecFromStr(v)
	if err != nil {
		return fmt.Errorf("invalid developer shares: %s", err)
	}

	if shares.IsNegative() {
		return fmt.Errorf("developer shares cannot be negative")
	}

	if shares.GT(math.LegacyOneDec()) {
		return fmt.Errorf("developer shares cannot be greater than 100%%")
	}

	return nil
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of feedistribution module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyEnableFeeDistribution, &p.EnableFeeDistribution, validateBool),
		paramtypes.NewParamSetPair(KeyDeveloperShares, &p.DeveloperShares, validateShares),
	}
}

// validateBool validates that the provided parameter is a bool
func validateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

// validateShares validates that the provided shares are between 0 and 1
func validateShares(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("shares cannot be negative")
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("shares cannot be greater than 1")
	}

	return nil
}
