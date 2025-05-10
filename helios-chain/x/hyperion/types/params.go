package types

import (
	"bytes"
	"fmt"
	"reflect"
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
		CounterpartyChainParams: []*CounterpartyChainParams{
			DefaultPolygonAmoyTestnet21ChainParams(),
			DefaultEthereumSepoliaTestnet22ChainParams(),
			// DefaultLocalPolygonAmoyTestnet21ChainParams(),
		},
		Admins: []string{"helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf"}, // for whitelisting and blacklisting
	}
}

func DefaultLocalPolygonAmoyTestnet21ChainParams() *CounterpartyChainParams {
	return &CounterpartyChainParams{
		HyperionId:                    21,
		BridgeCounterpartyAddress:     common.HexToAddress("0xB8ed88AcD8b7ac80d9f546F4D75F33DD19dD5746").Hex(),
		BridgeChainId:                 80002,
		BridgeChainName:               "Polygon Amoy Testnet",
		BridgeChainLogo:               "51ff5cb29b89cebe3bb8c9c3191fd5109122a5419c2c0bbebddd7a080b20a3b1",
		BridgeChainType:               "evm",
		SignedValsetsWindow:           25000,
		SignedBatchesWindow:           25000,
		SignedClaimsWindow:            25000,
		TargetBatchTimeout:            3600000, // 1 hour
		TargetOutgoingTxTimeout:       600000,  // 10 minutes
		AverageBlockTime:              4000,    // 4 seconds
		AverageCounterpartyBlockTime:  1000,    // chain blocktime 2seconds
		SlashFractionValset:           math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBatch:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionClaim:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionConflictingClaim: math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBadEthSignature:  math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		UnbondSlashingValsetsWindow:   25000,
		ClaimSlashingEnabled:          false,
		BridgeContractStartHeight:     20193284,
		ValsetReward:                  sdktypes.Coin{Denom: "ahelios", Amount: math.NewInt(0)},
		Initializer:                   "helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf",
		DefaultTokens: []*TokenAddressToDenomWithGenesisInfos{
			{
				TokenAddressToDenom: &TokenAddressToDenom{
					Denom:              "ahelios",
					TokenAddress:       "0x8916f85e0Da4A2Ff2c304e67105dd9d6B0a7F81c",
					IsCosmosOriginated: true,
					Symbol:             "HLS",
					Decimals:           18,
					IsConcensusToken:   true,
				},
				DefaultHolders: []*HolderWithAmount{
					{
						Address: common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
				},
			},
			{
				TokenAddressToDenom: &TokenAddressToDenom{
					Denom:              "upol",
					TokenAddress:       "0x12aa76b88c093c1fe950b609baaa4f46c72b3d6e",
					IsCosmosOriginated: false,
					Symbol:             "POL",
					Decimals:           18,
					IsConcensusToken:   true,
				},
				DefaultHolders: []*HolderWithAmount{
					{
						Address: common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0x17267eb1fec301848d4b5140eddcfc48945427ab").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0xd1cfbbb140972530a3798fe1ba11312eb8c99582").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0xc9728bfb36f8d2f9d39a5e7ce19aa11af27db440").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0x5eb1275822495570f48b8e573431d217d5708197").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
				},
			},
		},
		Rpcs: []*Rpc{
			{
				Url:            "https://rpc-amoy.polygon.technology",
				Reputation:     0,
				LastHeightUsed: 0,
			},
		},
	}
}

func DefaultPolygonAmoyTestnet21ChainParams() *CounterpartyChainParams {
	return &CounterpartyChainParams{
		HyperionId:                    21,
		BridgeCounterpartyAddress:     common.HexToAddress("0x87180495C8393C810fBD0882265B4C3b1EF2431e").Hex(),
		BridgeChainId:                 80002,
		BridgeChainName:               "Polygon Amoy Testnet",
		BridgeChainLogo:               "51ff5cb29b89cebe3bb8c9c3191fd5109122a5419c2c0bbebddd7a080b20a3b1",
		BridgeChainType:               "evm",
		SignedValsetsWindow:           25000,
		SignedBatchesWindow:           25000,
		SignedClaimsWindow:            25000,
		TargetBatchTimeout:            3600000, // 1 hour
		TargetOutgoingTxTimeout:       600000,  // 10 minutes
		AverageBlockTime:              4000,    // 4 seconds
		AverageCounterpartyBlockTime:  1000,    // chain blocktime 1seconds
		SlashFractionValset:           math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBatch:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionClaim:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionConflictingClaim: math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBadEthSignature:  math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		UnbondSlashingValsetsWindow:   25000,
		ClaimSlashingEnabled:          false,
		BridgeContractStartHeight:     20101217,
		ValsetReward:                  sdktypes.Coin{Denom: "ahelios", Amount: math.NewInt(0)},
		Initializer:                   "helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf",
		DefaultTokens: []*TokenAddressToDenomWithGenesisInfos{
			{
				TokenAddressToDenom: &TokenAddressToDenom{
					Denom:              "ahelios",
					TokenAddress:       "0x8916f85e0Da4A2Ff2c304e67105dd9d6B0a7F81c",
					IsCosmosOriginated: true,
					Symbol:             "HLS",
					Decimals:           18,
					IsConcensusToken:   true,
				},
				DefaultHolders: []*HolderWithAmount{
					{
						Address: common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
				},
			},
			{
				TokenAddressToDenom: &TokenAddressToDenom{
					Denom:              "upol",
					TokenAddress:       "0x12aa76b88c093c1fe950b609baaa4f46c72b3d6e",
					IsCosmosOriginated: false,
					Symbol:             "POL",
					Decimals:           18,
					IsConcensusToken:   true,
				},
				DefaultHolders: []*HolderWithAmount{
					{
						Address: common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0x17267eb1fec301848d4b5140eddcfc48945427ab").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0xd1cfbbb140972530a3798fe1ba11312eb8c99582").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0xc9728bfb36f8d2f9d39a5e7ce19aa11af27db440").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0x5eb1275822495570f48b8e573431d217d5708197").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
				},
			},
		},
		Rpcs: []*Rpc{
			{
				Url:            "https://rpc-amoy.polygon.technology",
				Reputation:     0,
				LastHeightUsed: 0,
			},
		},
	}
}

func DefaultEthereumSepoliaTestnet22ChainParams() *CounterpartyChainParams {
	return &CounterpartyChainParams{
		HyperionId:                    22,
		BridgeCounterpartyAddress:     common.HexToAddress("0xA2512e1f33020d34915124218EdbEC20901755b2").Hex(),
		BridgeChainId:                 11155111,
		BridgeChainName:               "Sepolia Testnet",
		BridgeChainLogo:               "45fa0204dcbb461f9899168a8b56162ecc832919b0c8b81b85f7de2abba408aa",
		BridgeChainType:               "evm",
		SignedValsetsWindow:           25000,
		SignedBatchesWindow:           25000,
		SignedClaimsWindow:            25000,
		TargetBatchTimeout:            3600000, // 1 hour
		TargetOutgoingTxTimeout:       600000,  // 10 minutes
		AverageBlockTime:              4000,    // 4 seconds
		AverageCounterpartyBlockTime:  12000,   // chain blocktime 12seconds
		SlashFractionValset:           math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBatch:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionClaim:            math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionConflictingClaim: math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		SlashFractionBadEthSignature:  math.LegacyNewDec(1).Quo(math.LegacyNewDec(1000)),
		UnbondSlashingValsetsWindow:   25000, // 25000 blocks
		ClaimSlashingEnabled:          false,
		BridgeContractStartHeight:     8062855,
		ValsetReward:                  sdktypes.Coin{Denom: "ahelios", Amount: math.NewInt(0)},
		Initializer:                   "helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf",
		DefaultTokens: []*TokenAddressToDenomWithGenesisInfos{
			{
				TokenAddressToDenom: &TokenAddressToDenom{
					Denom:              "ahelios",
					TokenAddress:       common.HexToAddress("0x462D63407eb86531dce7f948F2145382bc269E7C").Hex(),
					IsCosmosOriginated: true,
					IsConcensusToken:   true,
					Symbol:             "HLS",
					Decimals:           18,
				},
				DefaultHolders: []*HolderWithAmount{
					{
						Address: common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
				},
			},
			{
				TokenAddressToDenom: &TokenAddressToDenom{
					Denom:              "ueth",
					TokenAddress:       "0xe0dc6cad46f380c179622f4e85e744fc55e1373b",
					IsCosmosOriginated: false,
					Symbol:             "ETH",
					Decimals:           18,
					IsConcensusToken:   true,
				},
				DefaultHolders: []*HolderWithAmount{
					{
						Address: common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0x17267eB1FEC301848d4B5140eDDCFC48945427Ab").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0xd1cfbbb140972530a3798fe1ba11312eb8c99582").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0xc9728bfb36f8d2f9d39a5e7ce19aa11af27db440").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
					{
						Address: common.HexToAddress("0x5eb1275822495570f48b8e573431d217d5708197").Hex(),
						Amount:  math.LegacyNewDec(100).Mul(math.LegacyNewDec(10).Power(18)).RoundInt(),
					},
				},
				Logo: "45fa0204dcbb461f9899168a8b56162ecc832919b0c8b81b85f7de2abba408aa",
			},
		},
		Rpcs: []*Rpc{
			{
				Url:            "https://0xrpc.io/sep",
				Reputation:     0,
				LastHeightUsed: 0,
			},
		},
	}
}

// ValidateBasic checks that the parameters have valid values.
func (p Params) ValidateBasic() error {
	for _, cp := range p.CounterpartyChainParams {
		if err := validateCounterpartyChainParams(*cp); err != nil {
			return err
		}
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

func (p CounterpartyChainParams) ValidateBasic() error {
	return validateCounterpartyChainParams(p)
}

func validateCounterpartyChainParams(i interface{}) error {
	fmt.Println(reflect.TypeOf(i))
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
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
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
