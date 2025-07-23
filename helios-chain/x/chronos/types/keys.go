package types

import "math/big"

const (
	// ModuleName defines the module name
	ModuleName = "chronos"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey defines the module's message routing key
	RouterKey = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_cron"
)

// Key prefixes for different types of data
const (
	prefixCronKey                          = iota + 1 // 1
	prefixCronAddressKey                              // 2
	prefixCronCountKey                                // 3
	prefixParamsKey                                   // 4
	prefixNextCronIDKey                               // 5
	prefixCronNonceKey                                // 6
	prefixCronTransactionResultKey                    // 7
	prefixCronBlockTransactionHashsKey                // 8
	prefixCronTransactionHashToNonceKey               // 9
	prefixCronTransactionResultByCronIdKey            // 10
	prefixCronCallBackDataKey                         // 11
	prefixCronArchivedKey                             // 12
	prefixCronArchivedCountKey                        // 13
	prefixCronRefundedLastBlockCountKey               // 14
	prefixCronExecutedLastBlockCountKey               // 15
	prefixCronQueueKey                                // 16
	prefixSecondIndexOutgoingTXFeeKey                 // 17
	prefixCronQueueCountKey                           // 18
	prefixCronIndexByOwnerAddressKey                  // 19
)

var (
	// CronKey is the prefix for storing individual crons
	CronKey = []byte{prefixCronKey}

	// CronAddressKey is the prefix for storing individual crons Address Key
	CronAddressKey = []byte{prefixCronAddressKey}

	// CronCountKey is the key for storing the total count of crons
	CronCountKey = []byte{prefixCronCountKey}

	// ParamsKey is the key for storing module parameters
	ParamsKey = []byte{prefixParamsKey}

	// NextCronIDKey is the key for storing the next cron ID counter
	NextCronIDKey = []byte{prefixNextCronIDKey}

	// CronNonceKey is the key for storing the transactions nonce of all crons
	CronNonceKey = []byte{prefixCronNonceKey}

	// CronTransactionResultKey it the key for storing the individual crons tx results
	CronTransactionResultKey = []byte{prefixCronTransactionResultKey}

	// CronBlockTransactionHashsKey it the key for storing the individual crons tx results Hash Key
	CronBlockTransactionHashsKey = []byte{prefixCronBlockTransactionHashsKey}

	// CronTransactionHashToNonceKey it the key for storing the individual crons tx results Nonce Key
	CronTransactionHashToNonceKey = []byte{prefixCronTransactionHashToNonceKey}

	// CronTransactionResultByCronIdKey it the key for storing the multiple cron tx results on CronIdKey
	CronTransactionResultByCronIdKey = []byte{prefixCronTransactionResultByCronIdKey}

	CronCallBackDataKey = []byte{prefixCronCallBackDataKey}

	// todo testnet remove this keys
	CronArchivedKey               = []byte{prefixCronArchivedKey}
	CronArchivedCountKey          = []byte{prefixCronArchivedCountKey}
	CronRefundedLastBlockCountKey = []byte{prefixCronRefundedLastBlockCountKey}
	CronExecutedLastBlockCountKey = []byte{prefixCronExecutedLastBlockCountKey}

	CronQueueKey = []byte{prefixCronQueueKey}

	SecondIndexOutgoingTXFeeKey = []byte{prefixSecondIndexOutgoingTXFeeKey}

	CronQueueCountKey = []byte{prefixCronQueueCountKey}

	CronIndexByOwnerAddressKey = []byte{prefixCronIndexByOwnerAddressKey}
)

var (
	ArchiveStoreCronKey                          = []byte("cron:")
	ArchiveStoreTxKey                            = []byte("tx:")
	ArchiveStoreCronTxNonceKey                   = []byte("nid:")
	ArchiveStoreCronTransactionResultByCronIdKey = []byte("r:")
	ArchiveStoreBlockTransactionHashsKey         = []byte("b:")
	ArchiveStoreExecutedLastBlockCountKey        = []byte("executed_last_block_count")
	ArchiveStoreRefundedLastBlockCountKey        = []byte("refunded_last_block_count")
	ArchiveStoreArchivedCronCountKey             = []byte("archived_cron_count")
	ArchiveStoreCronCountKey                     = []byte("cron_count")
)

// GetCronKey returns the key for a specific cron by name
// Note: This function is deprecated. Use cron IDs with GetCronIDBytes
// in the keeper package instead.
func GetCronKey(name string) []byte {
	return []byte(name)
}

// GetFeeSecondIndexKey returns the following key format
// prefix      					fee_amount
// [0x9][0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
func GetFeeSecondIndexKey(feesPriority *big.Int) []byte {
	buf := make([]byte, 0, len(SecondIndexOutgoingTXFeeKey)+32)
	buf = append(buf, SecondIndexOutgoingTXFeeKey...)

	// sdk.BigInt represented as a zero-extended big-endian byte slice (32 bytes)
	amount := make([]byte, 32)
	amount = feesPriority.FillBytes(amount)
	buf = append(buf, amount...)

	return buf
}
