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
	prefixScheduleKey       = iota + 1 // 1
	prefixScheduleCountKey             // 2
	prefixParamsKey                    // 3
	prefixNextScheduleIDKey            // 4
)

var (
	// ScheduleKey is the prefix for storing individual schedules
	ScheduleKey = []byte{prefixScheduleKey}

	// ScheduleCountKey is the key for storing the total count of schedules
	ScheduleCountKey = []byte{prefixScheduleCountKey}

	// ParamsKey is the key for storing module parameters
	ParamsKey = []byte{prefixParamsKey}

	// NextScheduleIDKey is the key for storing the next schedule ID counter
	NextScheduleIDKey = []byte{prefixNextScheduleIDKey}
)

// GetScheduleKey returns the key for a specific schedule by name
// Note: This function is deprecated. Use schedule IDs with GetScheduleIDBytes
// in the keeper package instead.
func GetScheduleKey(name string) []byte {
	return []byte(name)
}
