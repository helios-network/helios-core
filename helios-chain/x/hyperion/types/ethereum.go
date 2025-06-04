package types

import (
	"bytes"
	"fmt"
	"strconv"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	gethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	// HyperionDenomPrefix indicates the prefix for all assests minted by this module
	HyperionDenomPrefix = ModuleName

	// HyperionDenomSeparator is the separator for hyperion denoms
	HyperionDenomSeparator = "-"

	// ETHContractAddressLen is the length of contract address bytes
	ETHContractAddressLen = 20

	// ZeroAddress is an EthAddress containing the zero ethereum address
	ZeroAddressString = "0x0000000000000000000000000000000000000000"
)

// EthAddrLessThan migrates the Ethereum address less than function
func EthAddrLessThan(e, o string) bool {
	return bytes.Compare([]byte(e), []byte(o)) == -1
}

// Returns a new EthAddress with 0x0000000000000000000000000000000000000000 as the wrapped address
func ZeroAddress() gethcommon.Address {
	return gethcommon.HexToAddress(ZeroAddressString)
}

// Creates a new EthAddress from a string, performing validation and returning any validation errors
func NewEthAddress(address string) (*gethcommon.Address, error) {
	if err := ValidateEthAddress(address); err != nil {
		return nil, errors.Wrap(err, "invalid input address")
	}

	addr := gethcommon.HexToAddress(address)
	return &addr, nil
}

// ValidateEthAddress validates the ethereum address strings
func ValidateEthAddress(address string) error {
	if address == "" {
		return fmt.Errorf("empty")
	}
	if !gethcommon.IsHexAddress(address) {
		return fmt.Errorf("%s is not a valid ETH address", address)
	}
	return nil
}

func NewSDKIntERC20Token(amount math.Int, contract gethcommon.Address) *Token {
	return &Token{Amount: amount, Contract: contract.Hex()}
}

func NewHyperionDenom(hyperionId uint64, tokenContract gethcommon.Address) string {
	return fmt.Sprintf("%s%s%s%s%s", HyperionDenomPrefix, HyperionDenomSeparator, strconv.FormatUint(hyperionId, 10), HyperionDenomSeparator, tokenContract.Hex())
}
