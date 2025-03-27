package types

import (
	errorsmod "cosmossdk.io/errors"
)

// Revenue module sentinel errors
var (
	ErrInvalidAddress            = errorsmod.Register(ModuleName, 1, "invalid address")
	ErrUnauthorized              = errorsmod.Register(ModuleName, 2, "unauthorized")
	ErrContractNotRegistered     = errorsmod.Register(ModuleName, 3, "contract not registered")
	ErrRevenueAlreadyRegistered  = errorsmod.Register(ModuleName, 4, "contract already registered")
	ErrRevenueDeployerIsNotEOA   = errorsmod.Register(ModuleName, 5, "deployer is not an EOA")
	ErrRevenueNoContractDeployed = errorsmod.Register(ModuleName, 6, "no contract deployed at address")
	// ErrInvalidFee                = errorsmod.Register(ModuleName, 5, "invalid fee")
	// ErrInvalidWithdrawer         = errorsmod.Register(ModuleName, 6, "invalid withdrawer address")
	ErrFeeDistributionDisabled = errorsmod.Register(ModuleName, 7, "fee distribution is disabled")
	// ErrInvalidParam              = errorsmod.Register(ModuleName, 8, "invalid parameter")
	ErrContractAlreadyRegistered = errorsmod.Register(ModuleName, 9, "contract already registered")
	ErrContractDeployerNotFound  = errorsmod.Register(ModuleName, 10, "contract deployer not found")
	// ErrInvalidSigner             = errorsmod.Register(ModuleName, 11, "invalid signer")
)

// ValidateAddress validates an Ethereum address
func ValidateAddress(address string) error {
	if !IsHexAddress(address) {
		return ErrInvalidAddress
	}
	return nil
}

// IsHexAddress verifies whether a string can represent a valid hex-encoded
// Ethereum address or not.
func IsHexAddress(address string) bool {
	if len(address) == 0 {
		return false
	}

	// Check if address has hex prefix
	if address[0:2] != "0x" {
		return false
	}

	// Check address is 20 bytes long (40 hex characters + 0x)
	if len(address) != 42 {
		return false
	}

	// Check if address contains only hexadecimal characters
	for _, c := range address[2:] {
		if !isHexCharacter(c) {
			return false
		}
	}

	return true
}

// isHexCharacter returns bool of c being a valid hexadecimal character.
func isHexCharacter(c rune) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}
