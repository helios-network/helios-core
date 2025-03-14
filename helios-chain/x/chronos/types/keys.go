package types

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
)

// GetCronKey returns the key for a specific cron by name
// Note: This function is deprecated. Use cron IDs with GetCronIDBytes
// in the keeper package instead.
func GetCronKey(name string) []byte {
	return []byte(name)
}
