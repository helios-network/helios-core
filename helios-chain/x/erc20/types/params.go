package types

import (
	"fmt"
	"slices"
	"strings"

	"helios-core/helios-chain/types"
	"helios-core/helios-chain/utils"

	"github.com/ethereum/go-ethereum/common"
)

const (
	// WEVMOSContractMainnet is the WEVMOS contract address for mainnet
	WEVMOSContractMainnet = "0xD4949664cD82660AaE99bEdc034a0deA8A0bd517"
	// WEVMOSContractTestnet is the WEVMOS contract address for testnet
	WEVMOSContractTestnet = "0xcc491f589b45d4a3c679016195b3fb87d7848210"
)

// chainsWEVMOSHex is an utility map used to retrieve the WEVMOS contract
// address in hex format from the chain ID.
var chainsWEVMOSHex = map[string]string{
	utils.MainnetChainID: WEVMOSContractMainnet,
	utils.TestnetChainID: WEVMOSContractTestnet,
}

// GetWEVMOSContractHex returns the hex format of address for the WEVMOS contract given the
// chainID. If the chainID is not found, it defaults to the mainnet address.
func GetWEVMOSContractHex(chainID string) string {
	id := strings.Split(chainID, "-")[0]
	address, found := chainsWEVMOSHex[id]
	// default to mainnet address
	if !found {
		address = chainsWEVMOSHex[utils.MainnetChainID]
	}
	return address
}

// Parameter store key
var (
	ParamStoreKeyEnableErc20        = []byte("EnableErc20")
	ParamStoreKeyDynamicPrecompiles = []byte("DynamicPrecompiles")
	ParamStoreKeyNativePrecompiles  = []byte("NativePrecompiles")

	ParamStoreKeyDynamicPrecompilePrefix = []byte("DP")
	ParamStoreKeyNativePrecompilePrefix  = []byte("NP")
	// DefaultNativePrecompiles defines the default precompiles for the wrapped native coin
	// NOTE: If you modify this, make sure you modify it on the local_node genesis script as well
	DefaultNativePrecompiles = []string{WEVMOSContractMainnet}
	// DefaultDynamicPrecompiles defines the default active dynamic precompiles
	DefaultDynamicPrecompiles []string
)

// NewParams creates a new Params object
func NewParams(
	enableErc20 bool,
	nativePrecompiles []string,
	dynamicPrecompiles []string,
) Params {
	slices.Sort(nativePrecompiles)
	slices.Sort(dynamicPrecompiles)
	return Params{
		EnableErc20:        enableErc20,
		NativePrecompiles:  nativePrecompiles,
		DynamicPrecompiles: dynamicPrecompiles,
	}
}

func DefaultParams() Params {
	return Params{
		EnableErc20:        true,
		NativePrecompiles:  DefaultNativePrecompiles,
		DynamicPrecompiles: DefaultDynamicPrecompiles,
	}
}

func ValidateBool(i interface{}) error {
	_, ok := i.(bool)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func (p Params) Validate() error {
	if err := ValidateBool(p.EnableErc20); err != nil {
		return err
	}

	return nil
}

// ValidatePrecompiles checks if the precompile addresses are valid and unique.
func ValidatePrecompiles(i interface{}) ([]common.Address, error) {
	precompiles, ok := i.([]string)
	if !ok {
		return nil, fmt.Errorf("invalid precompile slice type: %T", i)
	}

	precAddrs := make([]common.Address, 0, len(precompiles))
	for _, precompile := range precompiles {
		err := types.ValidateAddress(precompile)
		if err != nil {
			return nil, fmt.Errorf("invalid precompile %s", precompile)
		}
		precAddrs = append(precAddrs, common.HexToAddress(precompile))
	}

	// NOTE: Check that the precompiles are sorted. This is required
	// to ensure determinism
	if !slices.IsSorted(precompiles) {
		return nil, fmt.Errorf("precompiles need to be sorted: %s", precompiles)
	}
	return precAddrs, nil
}
