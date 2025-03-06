package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyLimit                      = []byte("Limit")
	KeyMaxScheduledCallsPerWallet = []byte("MaxScheduledCallsPerWallet")
	KeyMinimumGasFeeMultiplier    = []byte("MinimumGasFeeMultiplier")

	DefaultLimit                      = uint64(5)
	DefaultMaxScheduledCallsPerWallet = uint64(10)
	DefaultMinimumGasFeeMultiplier    = "1.2"
)

// ParamKeyTable returns the param key table for the cron module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(limit uint64, maxScheduledCallsPerWallet uint64, minimumGasFeeMultiplier string) Params {
	return Params{
		Limit:                      limit,
		MaxScheduledCallsPerWallet: maxScheduledCallsPerWallet,
		MinimumGasFeeMultiplier:    minimumGasFeeMultiplier,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultLimit, DefaultMaxScheduledCallsPerWallet, DefaultMinimumGasFeeMultiplier)
}

// ParamSetPairs returns the param set pairs for the cron module
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyLimit, &p.Limit, validateLimit),
		paramtypes.NewParamSetPair(KeyMaxScheduledCallsPerWallet, &p.MaxScheduledCallsPerWallet, validateMaxScheduledCallsPerWallet),
		paramtypes.NewParamSetPair(KeyMinimumGasFeeMultiplier, &p.MinimumGasFeeMultiplier, validateMinimumGasFeeMultiplier),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateLimit(p.Limit); err != nil {
		return err
	}
	if err := validateMaxScheduledCallsPerWallet(p.MaxScheduledCallsPerWallet); err != nil {
		return err
	}
	if err := validateMinimumGasFeeMultiplier(p.MinimumGasFeeMultiplier); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func validateLimit(i interface{}) error {
	l, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if l == 0 {
		return fmt.Errorf("limit cannot be zero")
	}
	return nil
}

func validateMaxScheduledCallsPerWallet(i interface{}) error {
	l, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if l == 0 {
		return fmt.Errorf("max scheduled calls per wallet cannot be zero")
	}
	return nil
}

func validateMinimumGasFeeMultiplier(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == "" {
		return fmt.Errorf("minimum gas fee multiplier cannot be empty")
	}
	return nil
}
