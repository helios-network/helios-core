package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

var _ paramtypes.ParamSet = (*Params)(nil)

var (
	KeyCronActiveGasCostPerBlock = []byte("CronActiveGasCostPerBlock")
	KeyExecutionsLimitPerBlock   = []byte("ExecutionsLimitPerBlock")

	DefaultCronActiveGasCostPerBlock = uint64(10)  // 10 Gas
	DefaultExecutionsLimitPerBlock   = uint64(100) // 100
)

// ParamKeyTable returns the param key table for the cron module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(cronActiveGasCostPerBlock uint64, executionsLimitPerBlock uint64) Params {
	return Params{
		CronActiveGasCostPerBlock: cronActiveGasCostPerBlock,
		ExecutionsLimitPerBlock:   executionsLimitPerBlock,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultCronActiveGasCostPerBlock, DefaultExecutionsLimitPerBlock)
}

// ParamSetPairs returns the param set pairs for the cron module
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyCronActiveGasCostPerBlock, &p.CronActiveGasCostPerBlock, validateCronActiveGasCostPerBlock),
		paramtypes.NewParamSetPair(KeyExecutionsLimitPerBlock, &p.ExecutionsLimitPerBlock, validateExecutionsLimitPerBlock),
	}
}

// Validate validates the set of params
func (p Params) Validate() error {
	if err := validateCronActiveGasCostPerBlock(p.CronActiveGasCostPerBlock); err != nil {
		return err
	}
	if err := validateExecutionsLimitPerBlock(p.ExecutionsLimitPerBlock); err != nil {
		return err
	}
	return nil
}

// String implements the Stringer interface
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func validateCronActiveGasCostPerBlock(i interface{}) error {
	l, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if l == 0 {
		return fmt.Errorf("limit cannot be zero")
	}
	return nil
}

func validateExecutionsLimitPerBlock(i interface{}) error {
	l, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if l == 0 {
		return fmt.Errorf("limit cannot be zero")
	}
	return nil
}
