package types

import (
	"cosmossdk.io/errors"
)

var (
	ErrInternal                         = errors.Register(ModuleName, 1, "internal")
	ErrDuplicate                        = errors.Register(ModuleName, 2, "duplicate")
	ErrInvalid                          = errors.Register(ModuleName, 3, "invalid")
	ErrTimeout                          = errors.Register(ModuleName, 4, "timeout")
	ErrUnknown                          = errors.Register(ModuleName, 5, "unknown")
	ErrEmpty                            = errors.Register(ModuleName, 6, "empty")
	ErrOutdated                         = errors.Register(ModuleName, 7, "outdated")
	ErrUnsupported                      = errors.Register(ModuleName, 8, "unsupported")
	ErrNonContiguousEventNonce          = errors.Register(ModuleName, 9, "non contiguous event nonce")
	ErrNoUnbatchedTxsFound              = errors.Register(ModuleName, 10, "no unbatched txs found")
	ErrResetDelegateKeys                = errors.Register(ModuleName, 11, "can not set orchestrator addresses more than once")
	ErrSupplyOverflow                   = errors.Register(ModuleName, 12, "supply cannot exceed max ERC20 value")
	ErrInvalidEthSender                 = errors.Register(ModuleName, 13, "invalid ethereum sender on claim")
	ErrInvalidEthDestination            = errors.Register(ModuleName, 14, "invalid ethereum destination")
	ErrNoLastClaimForValidator          = errors.Register(ModuleName, 15, "missing previous claim for validator")
	ErrNonContiguousEthEventBlockHeight = errors.Register(ModuleName, 16, "non contiguous eth event nonce height")
	ErrInvalidHyperionId                = errors.Register(ModuleName, 17, "invalid hyperion id")
	ErrInvalidSigner                    = errors.Register(ModuleName, 18, "invalid signer")
	ErrAttestationAlreadyVoted          = errors.Register(ModuleName, 19, "attestation already voted")
	ErrAttestationAlreadyObserved       = errors.Register(ModuleName, 20, "attestation already observed")
)
