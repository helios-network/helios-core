package types

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
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

// NewERC20Token returns a new instance of an ERC20
func NewERC20Token(amount uint64, contract gethcommon.Address) *Token {
	return &Token{Amount: math.NewIntFromUint64(amount), Contract: contract.Hex()}
}

func NewSDKIntERC20Token(amount math.Int, contract gethcommon.Address) *Token {
	return &Token{Amount: amount, Contract: contract.Hex()}
}

// HyperionCoin returns the hyperion representation of an ERC20 token
func (e *Token) HyperionCoin(hyperionId uint64) sdk.Coin {
	return sdk.NewCoin(HyperionDenomString(hyperionId, gethcommon.HexToAddress(e.Contract)), e.Amount)
}

type HyperionDenom []byte

func (p HyperionDenom) String() string {
	contractAddress, err := p.TokenContract()
	if err != nil {
		// the case of unparseable hyperion denom
		return fmt.Sprintf("%x(error: %s)", []byte(p), err.Error())
	}

	hyperionId, err := p.HyperionId()
	if err != nil {
		// the case of unparseable hyperion denom
		return fmt.Sprintf("%x(error: %s)", []byte(p), err.Error())
	}

	return HyperionDenomString(hyperionId, contractAddress)
}

func (p HyperionDenom) HyperionId() (uint64, error) {
	fullPrefix := []byte(HyperionDenomPrefix + HyperionDenomSeparator)
	if !bytes.HasPrefix(p, fullPrefix) {
		err := fmt.Errorf("denom '%x' byte prefix not equal to expected '%x'", []byte(p), fullPrefix)
		return 0, err
	}

	addressWithHyperionIdBytes := bytes.TrimPrefix(p, fullPrefix)
	if len(addressWithHyperionIdBytes) <= ETHContractAddressLen+len(HyperionDenomSeparator) {
		err := fmt.Errorf("failed to validate Ethereum address bytes: %x", addressWithHyperionIdBytes)
		return 0, err
	}

	// Extract hyperionId from the bytes
	hyperionIdBytes := addressWithHyperionIdBytes[:len(addressWithHyperionIdBytes)-(ETHContractAddressLen+len(HyperionDenomSeparator))]
	hyperionId, err := strconv.ParseUint(string(hyperionIdBytes), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse hyperionId: %v", err)
	}

	return hyperionId, nil
}

func (p HyperionDenom) TokenContract() (gethcommon.Address, error) {
	fullPrefix := []byte(HyperionDenomPrefix + HyperionDenomSeparator)
	if !bytes.HasPrefix(p, fullPrefix) {
		err := fmt.Errorf("denom '%x' byte prefix not equal to expected '%x'", []byte(p), fullPrefix)
		return gethcommon.Address{}, err
	}

	addressWithHyperionIdBytes := bytes.TrimPrefix(p, fullPrefix)
	if len(addressWithHyperionIdBytes) <= ETHContractAddressLen {
		err := fmt.Errorf("failed to validate Ethereum address bytes: %x", addressWithHyperionIdBytes)
		return gethcommon.Address{}, err
	}

	// Dépasser le hyperionId et HyperionDenomSeparator pour obtenir uniquement l'adresse du contrat
	addressBytes := addressWithHyperionIdBytes[len(addressWithHyperionIdBytes)-ETHContractAddressLen:]

	return gethcommon.BytesToAddress(addressBytes), nil
}

func NewHyperionDenom(hyperionId uint64, tokenContract gethcommon.Address) HyperionDenom {
	buf := append([]byte(HyperionDenomPrefix), []byte(HyperionDenomSeparator)...)
	buf = append(buf, []byte(strconv.FormatUint(hyperionId, 10))...)
	buf = append(buf, []byte(HyperionDenomSeparator)...)
	buf = append(buf, tokenContract.Bytes()...)

	return HyperionDenom(buf)
}

func NewHyperionDenomFromString(hyperionId uint64, denom string) (HyperionDenom, error) {
	fullPrefix := HyperionDenomPrefix + HyperionDenomSeparator + strconv.FormatUint(hyperionId, 10) + HyperionDenomSeparator
	if !strings.HasPrefix(denom, fullPrefix) {
		err := fmt.Errorf("denom '%s' string prefix not equal to expected '%s'", denom, fullPrefix)
		return nil, err
	}

	addressHex := strings.TrimPrefix(denom, fullPrefix)
	if err := ValidateEthAddress(addressHex); err != nil {
		return nil, err
	}

	hyperionDenom := NewHyperionDenom(hyperionId, gethcommon.HexToAddress(addressHex))
	return hyperionDenom, nil
}

func HyperionDenomString(hyperionId uint64, tokenContract gethcommon.Address) string {
	return fmt.Sprintf("%s%s%s%s%s", HyperionDenomPrefix, HyperionDenomSeparator, strconv.FormatUint(hyperionId, 10), HyperionDenomSeparator, tokenContract.Hex())
}

// ValidateBasic permforms stateless validation
func (e *Token) ValidateBasic(hyperionId uint64) error {
	if err := ValidateEthAddress(e.Contract); err != nil {
		return errors.Wrap(err, "ethereum address")
	}

	if !e.HyperionCoin(hyperionId).IsValid() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, e.HyperionCoin(hyperionId).String())
	}

	if !e.HyperionCoin(hyperionId).IsPositive() {
		return errors.Wrap(sdkerrors.ErrInvalidCoins, e.HyperionCoin(hyperionId).String())
	}

	return nil
}

// Add adds one ERC20 to another
func (e *Token) Add(o *Token) (*Token, error) {
	if e.Contract != o.Contract {
		return nil, fmt.Errorf("invalid contract address")
	}

	sum := e.Amount.Add(o.Amount)
	if !sum.IsUint64() {
		return nil, fmt.Errorf("invalid amount")
	}

	return NewERC20Token(sum.Uint64(), gethcommon.HexToAddress(e.Contract)), nil
}
