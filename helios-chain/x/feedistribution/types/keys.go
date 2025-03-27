package types

import (
	"github.com/ethereum/go-ethereum/common"
)

const (
	prefixContract = iota + 1
	prefixBlockFees
	prefixParams
	prefixRevenue
	prefixCodeHash
	prefixDeployer
	prefixWithdrawer
)

const (
	// ModuleName defines the module name
	ModuleName = "feedistribution"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

// prefix bytes for the feedistribution persistent store
var (
	// KeyPrefixContract is the prefix to retrieve all ContractInfo
	KeyPrefixContract = []byte{prefixContract}

	// KeyPrefixBlockFees is the prefix to retrieve all BlockFees
	KeyPrefixBlockFees = []byte{prefixBlockFees}

	// KeyPrefixParams is the prefix to retrieve module parameters
	KeyPrefixParams = []byte{prefixParams}

	// KeyPrefixRevenue is the prefix to retrieve all Revenue objects
	KeyPrefixRevenue = []byte{prefixRevenue}

	// KeyPrefixCodeHash is the prefix to retrieve all contract code hashes
	KeyPrefixCodeHash = []byte{prefixCodeHash}

	// KeyPrefixDeployer is the prefix to retrieve all deployer addresses
	KeyPrefixDeployer = []byte{prefixDeployer}

	// KeyPrefixWithdrawer is the prefix to retrieve all withdrawer addresses
	KeyPrefixWithdrawer = []byte{prefixWithdrawer}
)

// GetBlockFeesKey returns the store key to retrieve block fees for a specific contract
func GetBlockFeesKey(contract common.Address) []byte {
	return append(KeyPrefixBlockFees, contract.Bytes()...)
}
