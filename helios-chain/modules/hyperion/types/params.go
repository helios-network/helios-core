package types

import (
	"bytes"
	"fmt"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
)

// DefaultParamspace defines the default hyperion module parameter subspace
const (
	DefaultParamspace = ModuleName
)

// DefaultParams returns a copy of the default params
func DefaultParams() *Params {
	return &Params{
		CounterpartyChainParams: []*CounterpartyChainParams{DefaultCounterpartyChainParams()},
		Admins:                  []string{"helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf"}, // for whitelisting and blacklisting
	}
}

// DefaultCounterpartyChainParams returns a copy of the default counterparty chain params
func DefaultCounterpartyChainParams() *CounterpartyChainParams {
	return &CounterpartyChainParams{
		HyperionId:                    "hyperion-ethereum-mainnet",
		BridgeCounterpartyAddress:     common.HexToAddress("0x648d15cba34705B0e863502d23B31416Aed2Dc22").Hex(),
		BridgeChainId:                 1,
		SignedValsetsWindow:           25000,
		SignedBatchesWindow:           25000,
		SignedClaimsWindow:            25000,
		TargetBatchTimeout:            43200000,
		AverageBlockTime:              2000,
		AverageCounterpartyBlockTime:  15000,
		SlashFractionValset:           math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBatch:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionClaim:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionConflictingClaim: math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBadEthSignature:  math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		CosmosCoinDenom:               "helios",
		CosmosCoinErc20Contract:       "",
		UnbondSlashingValsetsWindow:   25000,
		ClaimSlashingEnabled:          false,
		ValsetReward:                  sdktypes.Coin{Denom: "helios", Amount: math.NewInt(0)},
	}
}

// ValidateBasic checks that the parameters have valid values.
func (p Params) ValidateBasic() error {
	for _, cp := range p.CounterpartyChainParams {
		validateCounterpartyChainParams(cp)
	}
	if err := validateAdmins(p.Admins); err != nil {
		return errors.Wrap(err, "admins")
	}
	return nil
}

// Equal returns a boolean determining if two Params types are identical.
func (p Params) Equal(p2 Params) bool {
	bz1 := ModuleCdc.MustMarshalLengthPrefixed(&p)
	bz2 := ModuleCdc.MustMarshalLengthPrefixed(&p2)
	return bytes.Equal(bz1, bz2)
}

func validateCounterpartyChainParams(i interface{}) error {
	v, ok := i.(CounterpartyChainParams)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if err := validateHyperionID(v.HyperionId); err != nil {
		return errors.Wrap(err, "hyperion id")
	}
	if err := validateContractHash(v.ContractSourceHash); err != nil {
		return errors.Wrap(err, "contract hash")
	}
	if err := validateBridgeContractAddress(v.BridgeCounterpartyAddress); err != nil {
		return errors.Wrap(err, "bridge contract address")
	}
	if err := validateBridgeContractStartHeight(v.BridgeContractStartHeight); err != nil {
		return errors.Wrap(err, "bridge contract start height")
	}
	if err := validateBridgeChainID(v.BridgeChainId); err != nil {
		return errors.Wrap(err, "bridge chain id")
	}
	if err := validateCosmosCoinDenom(v.CosmosCoinDenom); err != nil {
		return errors.Wrap(err, "cosmos coin denom")
	}
	if err := validateCosmosCoinErc20Contract(v.CosmosCoinErc20Contract); err != nil {
		return errors.Wrap(err, "cosmos coin erc20 contract address")
	}
	if err := validateTargetBatchTimeout(v.TargetBatchTimeout); err != nil {
		return errors.Wrap(err, "Batch timeout")
	}
	if err := validateAverageBlockTime(v.AverageBlockTime); err != nil {
		return errors.Wrap(err, "Block time")
	}
	if err := validateAverageEthereumBlockTime(v.AverageCounterpartyBlockTime); err != nil {
		return errors.Wrap(err, "Ethereum block time")
	}
	if err := validateSignedValsetsWindow(v.SignedValsetsWindow); err != nil {
		return errors.Wrap(err, "signed blocks window")
	}
	if err := validateSignedBatchesWindow(v.SignedBatchesWindow); err != nil {
		return errors.Wrap(err, "signed blocks window")
	}
	if err := validateSignedClaimsWindow(v.SignedClaimsWindow); err != nil {
		return errors.Wrap(err, "signed blocks window")
	}
	if err := validateSlashFractionValset(v.SlashFractionValset); err != nil {
		return errors.Wrap(err, "slash fraction valset")
	}
	if err := validateSlashFractionBatch(v.SlashFractionBatch); err != nil {
		return errors.Wrap(err, "slash fraction valset")
	}
	if err := validateSlashFractionClaim(v.SlashFractionClaim); err != nil {
		return errors.Wrap(err, "slash fraction valset")
	}
	if err := validateSlashFractionConflictingClaim(v.SlashFractionConflictingClaim); err != nil {
		return errors.Wrap(err, "slash fraction valset")
	}
	if err := validateSlashFractionBadEthSignature(v.SlashFractionBadEthSignature); err != nil {
		return errors.Wrap(err, "slash fraction BadEthSignature")
	}
	if err := validateUnbondSlashingValsetsWindow(v.UnbondSlashingValsetsWindow); err != nil {
		return errors.Wrap(err, "unbond Slashing valset window")
	}
	if err := validateClaimSlashingEnabled(v.ClaimSlashingEnabled); err != nil {
		return errors.Wrap(err, "claim slashing enabled")
	}

	return nil
}

func validateHyperionID(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if _, err := strToFixByteArray(v); err != nil {
		return err
	}
	return nil
}

func validateContractHash(i interface{}) error {
	if _, ok := i.(string); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateBridgeChainID(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateBridgeContractStartHeight(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateTargetBatchTimeout(i interface{}) error {
	val, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	} else if val < 60000 {
		return fmt.Errorf("invalid target batch timeout, less than 60 seconds is too short")
	}
	return nil
}

func validateAverageBlockTime(i interface{}) error {
	val, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	} else if val < 100 {
		return fmt.Errorf("invalid average Cosmos block time, too short for latency limitations")
	}
	return nil
}

func validateAverageEthereumBlockTime(i interface{}) error {
	val, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	} else if val < 100 {
		return fmt.Errorf("invalid average Ethereum block time, too short for latency limitations")
	}
	return nil
}

func validateBridgeContractAddress(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if err := ValidateEthAddress(v); err != nil {
		if !strings.Contains(err.Error(), "empty") {
			return err
		}
	}
	return nil
}

func validateSignedValsetsWindow(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateUnbondSlashingValsetsWindow(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSlashFractionValset(i interface{}) error {
	if _, ok := i.(math.LegacyDec); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSignedBatchesWindow(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSignedClaimsWindow(i interface{}) error {
	if _, ok := i.(uint64); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSlashFractionBatch(i interface{}) error {
	if _, ok := i.(math.LegacyDec); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSlashFractionClaim(i interface{}) error {
	if _, ok := i.(math.LegacyDec); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSlashFractionConflictingClaim(i interface{}) error {
	if _, ok := i.(math.LegacyDec); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func strToFixByteArray(s string) ([32]byte, error) {
	var out [32]byte
	if len([]byte(s)) > 32 {
		return out, fmt.Errorf("string too long")
	}
	copy(out[:], s)
	return out, nil
}

func validateCosmosCoinDenom(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if _, err := strToFixByteArray(v); err != nil {
		return err
	}
	return nil
}

func validateCosmosCoinErc20Contract(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	// empty address is valid
	if v == "" {
		return nil
	}

	return ValidateEthAddress(v)
}

func validateClaimSlashingEnabled(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateSlashFractionBadEthSignature(i interface{}) error {
	if _, ok := i.(math.LegacyDec); !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateValsetReward(i interface{}) error {
	return nil
}

func validateAdmins(i interface{}) error {
	v, ok := i.([]string)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	admins := make(map[string]struct{})

	for _, admin := range v {
		adminAddr, err := sdk.AccAddressFromBech32(admin)
		if err != nil {
			return fmt.Errorf("invalid admin address: %s", admin)
		}

		if _, found := admins[adminAddr.String()]; found {
			return fmt.Errorf("duplicate admin: %s", admin)
		}
		admins[adminAddr.String()] = struct{}{}
	}

	return nil
}
